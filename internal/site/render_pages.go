package site

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/leaanthony/mpress/internal/check"
)

// bytesToString creates an immutable string view over data for read-only use.
// The CSS token scanner only reads the bytes synchronously and never retains the
// string, so this avoids copying every rendered page just to scan it.
func bytesToString(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(data), len(data))
}

// maxRenderWorkers bounds the render fan-out. Page rendering is CPU bound with
// a short per-page write, so the pool tracks the available cores like the parse
// stage rather than oversubscribing them.
const maxRenderWorkers = 8

type renderedPageResult struct {
	pageOut   string
	bytes     []byte
	linkIndex check.HTMLIndex
}

// renderPages renders and writes every page for one language concurrently. Each
// worker builds its own templateData through prepare, renders it, and writes the
// file, so the CPU-heavy renderPage call and its output write leave the serial
// build path. Results are returned in input order so downstream indexing stays
// deterministic.
//
// When collectTokens is set, each worker scans its rendered pages for CSS class
// and id tokens into a worker-local set; the sets are merged and returned. This
// folds the HTML scan the CSS purge needs into the parallel render instead of
// re-reading every page from a serial buffer afterwards. The returned token map
// is nil when collectTokens is false.
//
// collectLinks tokenizes HTML inside the render worker. The finished index is
// returned for serial collector mutation, which preserves deterministic health
// output without retaining every page's rendered bytes.
//
// prepare must be safe to call from multiple goroutines. It only reads the
// shared navigation, route, and blog maps, which are fixed before rendering
// begins, so no synchronisation is required inside it. Preparation errors stop
// the worker pool and are returned to the caller.
func renderPages(outputDir string, count int, collectTokens, collectLinks bool, prepare func(i int) (string, templateData, error)) ([]renderedPageResult, map[string]struct{}, error) {
	results := make([]renderedPageResult, count)
	if count == 0 {
		if collectTokens {
			return results, map[string]struct{}{}, nil
		}
		return results, nil, nil
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > maxRenderWorkers {
		workerCount = maxRenderWorkers
	}
	if workerCount > count {
		workerCount = count
	}

	// createdDirs caches output directories that already exist so repeated
	// pages in one section cost a map hit instead of a MkdirAll that re-stats
	// the whole parent chain.
	var createdDirs sync.Map

	// A single page is not worth a goroutine or the channel handshake.
	if workerCount == 1 {
		var tokens map[string]struct{}
		if collectTokens {
			tokens = make(map[string]struct{})
		}
		reuse := newRenderBuffer()
		for i := 0; i < count; i++ {
			res, err := renderAndWrite(outputDir, i, prepare, reuse, &createdDirs, collectLinks)
			if err != nil {
				return results, tokens, err
			}
			if collectTokens {
				addHTMLTokens(tokens, bytesToString(res.bytes))
			}
			res.bytes = nil
			results[i] = res
		}
		return results, tokens, nil
	}

	var (
		next     int64 = -1
		firstMu  sync.Mutex
		firstEr  error
		workers  sync.WaitGroup
		localTok = make([]map[string]struct{}, workerCount)
	)
	fail := func(err error) {
		firstMu.Lock()
		if firstEr == nil {
			firstEr = err
		}
		firstMu.Unlock()
	}
	workers.Add(workerCount)
	for w := 0; w < workerCount; w++ {
		go func(worker int) {
			defer workers.Done()
			var tokens map[string]struct{}
			if collectTokens {
				tokens = make(map[string]struct{})
				localTok[worker] = tokens
			}
			reuse := newRenderBuffer()
			for {
				i := int(atomic.AddInt64(&next, 1))
				if i >= count {
					return
				}
				firstMu.Lock()
				stop := firstEr != nil
				firstMu.Unlock()
				if stop {
					return
				}
				res, err := renderAndWrite(outputDir, i, prepare, reuse, &createdDirs, collectLinks)
				if err != nil {
					fail(err)
					return
				}
				if collectTokens {
					addHTMLTokens(tokens, bytesToString(res.bytes))
				}
				res.bytes = nil
				results[i] = res
			}
		}(w)
	}
	workers.Wait()
	if firstEr != nil {
		return results, nil, firstEr
	}
	var tokens map[string]struct{}
	if collectTokens {
		tokens = make(map[string]struct{})
		for _, local := range localTok {
			for token := range local {
				tokens[token] = struct{}{}
			}
		}
	}
	return results, tokens, nil
}

// newRenderBuffer returns one reusable worker-local render buffer. Link
// indexing consumes the page before this buffer is reused, so validation never
// needs to force a fresh allocation per generated page.
func newRenderBuffer() *bytes.Buffer {
	return &bytes.Buffer{}
}

func renderAndWrite(outputDir string, i int, prepare func(i int) (string, templateData, error), reuse *bytes.Buffer, createdDirs *sync.Map, collectLinks bool) (renderedPageResult, error) {
	pageOut, data, err := prepare(i)
	if err != nil {
		return renderedPageResult{}, err
	}
	// The caller does not retain page bytes, so render into the worker's buffer.
	// Reuse avoids allocating and zeroing a fresh page-sized backing array for
	// every page, including builds that collect link health data.
	reuse.Reset()
	if err := renderPageInto(data, reuse); err != nil {
		return renderedPageResult{}, err
	}
	rendered := reuse.Bytes()
	var linkIndex check.HTMLIndex
	if collectLinks {
		linkIndex = check.ParseHTML(rendered)
	}
	dest := filepath.Join(outputDir, filepath.FromSlash(pageOut))
	dir := filepath.Dir(dest)
	if _, ok := createdDirs.Load(dir); !ok {
		// Most page directories sit directly under an existing parent, so try
		// the single Mkdir syscall first; MkdirAll stats the whole parent chain
		// before creating anything. Fall back for nested paths and treat an
		// already-existing directory as success, matching MkdirAll.
		if err := os.Mkdir(dir, 0o755); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				err = os.MkdirAll(dir, 0o755)
			} else if errors.Is(err, fs.ErrExist) {
				err = nil
			}
			if err != nil {
				return renderedPageResult{}, err
			}
		}
		createdDirs.Store(dir, struct{}{})
	}
	if err := os.WriteFile(dest, rendered, 0o644); err != nil {
		return renderedPageResult{}, err
	}
	return renderedPageResult{pageOut: pageOut, bytes: rendered, linkIndex: linkIndex}, nil
}
