package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kungfusheep/glint"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/mpd"
	"github.com/leaanthony/mpress/internal/quickedit"
)

// Bump this when the content parser or its output contract changes. The
// source hash prevents stale pages after edits; this version handles parser
// changes that produce different output for identical Markdown.
// Increment this whenever parser or component rendering changes can alter the
// cached page HTML without changing the source document.
// v20 preserves native MPD emphasis adjacent to translated words.
const parseCacheVersion = "mpress-parse-v20"

const maxParseWorkers = 8

type parseJob struct {
	index      int
	lang       string
	rel        string
	sourcePath string
	override   bool
	source     string
}

type parseResult struct {
	job       parseJob
	page      *content.Page
	diags     []content.Diagnostic
	quickEdit *quickedit.Document
	err       error
}

type parseCacheEntry struct {
	Page        *content.Page        `json:"page" glint:"page"`
	Diagnostics []content.Diagnostic `json:"diagnostics,omitempty" glint:"diagnostics"`
	QuickEdit   *quickedit.Document  `json:"quickEdit,omitempty" glint:"quickEdit"`
}

type parseCache struct {
	dir       string
	hasLegacy bool
	decoders  sync.Pool
}

var parseCacheEncoder = glint.NewEncoder[parseCacheEntry]()

func newParseCache(projectDir string) *parseCache {
	dir := filepath.Join(projectDir, ".mpress", "cache", "parse")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil
	}
	cache := &parseCache{dir: dir, hasLegacy: dirHasSuffix(dir, ".json")}
	cache.decoders.New = func() any { return glint.NewDecoder[parseCacheEntry]() }
	return cache
}

// dirHasSuffix reports whether any entry in dir ends with suffix. It reads the
// directory once so the load path can skip the per-page legacy-cache probe when
// no legacy entries exist, which is every cache written since the Glint format.
func dirHasSuffix(dir, suffix string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), suffix) {
			return true
		}
	}
	return false
}

func (c *parseCache) key(job parseJob, source []byte) string {
	h := sha256.New()
	h.Write([]byte(parseCacheVersion))
	h.Write([]byte{0})
	h.Write([]byte(job.lang))
	h.Write([]byte{0})
	h.Write([]byte(job.rel))
	h.Write([]byte{0})
	h.Write(source)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *parseCache) loadEntry(key string) (parseCacheEntry, bool) {
	if c == nil {
		return parseCacheEntry{}, false
	}
	if data, err := os.ReadFile(filepath.Join(c.dir, key+".glint")); err == nil {
		entry, err := c.decode(data)
		if err == nil && entry.Page != nil {
			return entry, true
		}
	}
	// Skip the legacy JSON probe unless the cache directory actually contains
	// legacy entries. On a cache-cold build this removes one guaranteed-failing
	// open per page.
	if !c.hasLegacy {
		return parseCacheEntry{}, false
	}
	data, err := os.ReadFile(filepath.Join(c.dir, key+".json"))
	if err != nil {
		return parseCacheEntry{}, false
	}
	var entry parseCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil || entry.Page == nil {
		return parseCacheEntry{}, false
	}
	// Convert old cache entries on their first successful read. A cache miss is
	// always safe, but this makes upgrades benefit from Glint immediately.
	c.saveWithQuickEdit(key, entry.Page, entry.Diagnostics, entry.QuickEdit)
	return entry, true
}

func (c *parseCache) load(key string) (*content.Page, []content.Diagnostic, bool) {
	entry, ok := c.loadEntry(key)
	return entry.Page, entry.Diagnostics, ok
}

func (c *parseCache) save(key string, page *content.Page, diags []content.Diagnostic) {
	c.saveWithQuickEdit(key, page, diags, nil)
}

func (c *parseCache) saveWithQuickEdit(key string, page *content.Page, diags []content.Diagnostic, document *quickedit.Document) {
	if c == nil || page == nil {
		return
	}
	buffer := glint.NewBufferFromPoolWithCap(len(page.HTML) + len(page.PlainText) + 1024)
	defer buffer.ReturnToPool()
	parseCacheEncoder.Marshal(&parseCacheEntry{Page: page, Diagnostics: diags, QuickEdit: document}, buffer)
	tmp, err := os.CreateTemp(c.dir, ".parse-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err = tmp.Write(buffer.Bytes); err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err != nil {
		os.Remove(tmpName)
		return
	}
	// Only remove the temp file if the atomic rename fails. Removing it
	// unconditionally fired a failing unlink on every successful save, since the
	// temp no longer exists once it has been renamed into place.
	if err = os.Rename(tmpName, filepath.Join(c.dir, key+".glint")); err != nil {
		os.Remove(tmpName)
	}
}

func (c *parseCache) decode(data []byte) (entry parseCacheEntry, err error) {
	decoder := c.decoders.Get().(*glint.Decoder[parseCacheEntry])
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("invalid Glint parse cache: %v", recovered)
		}
		if err == nil {
			c.decoders.Put(decoder)
		}
	}()
	err = decoder.Unmarshal(data, &entry)
	return entry, err
}

func parseDiscoveredPages(projectDir, contentDir string, cfg config.Config, discovered map[string][]string, opts BuildOptions) []parseResult {
	var jobs []parseJob
	for _, lang := range cfg.Site.Languages {
		for _, rel := range discovered[lang] {
			overrideKey := filepath.ToSlash(rel)
			if lang != cfg.Site.DefaultLanguage {
				overrideKey = filepath.ToSlash(filepath.Join(lang, rel))
			}
			source, overridden := opts.SourceOverrides[overrideKey]
			jobs = append(jobs, parseJob{
				index:      len(jobs),
				lang:       lang,
				rel:        rel,
				sourcePath: content.SourcePath(contentDir, lang, cfg.Site.DefaultLanguage, rel),
				override:   overridden,
				source:     source,
			})
		}
	}
	cache := newParseCache(projectDir)
	results := make([]parseResult, len(jobs))
	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > maxParseWorkers {
		workerCount = maxParseWorkers
	}
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}
	if workerCount == 0 {
		return results
	}
	// Writing a parse-cache entry is disk I/O that would otherwise stall a parse
	// worker between two CPU-bound parses. Draining the saves through a separate
	// pool lets the write syscalls overlap with continued parsing, which is the
	// largest single cost of a cache-cold build. The saves are still awaited
	// before this function returns so the cache is complete and no temp files
	// leak.
	var (
		saveFn    func(key string, page *content.Page, diags []content.Diagnostic, document *quickedit.Document)
		saveQueue chan saveTask
		savers    sync.WaitGroup
	)
	if cache != nil {
		saveQueue = make(chan saveTask, len(jobs))
		saverCount := workerCount
		savers.Add(saverCount)
		for i := 0; i < saverCount; i++ {
			go func() {
				defer savers.Done()
				for task := range saveQueue {
					cache.saveWithQuickEdit(task.key, task.page, task.diags, task.document)
				}
			}()
		}
		saveFn = func(key string, page *content.Page, diags []content.Diagnostic, document *quickedit.Document) {
			saveQueue <- saveTask{key: key, page: page, diags: diags, document: document}
		}
	}

	queue := make(chan parseJob)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer workers.Done()
			renderer := content.NewRenderer()
			for job := range queue {
				results[job.index] = parseOne(renderer, cache, saveFn, job, cfg.Contribution.Enabled && opts.Development)
			}
		}()
	}
	go func() {
		for _, job := range jobs {
			queue <- job
		}
		close(queue)
	}()
	workers.Wait()
	if saveQueue != nil {
		close(saveQueue)
		savers.Wait()
	}
	return results
}

type saveTask struct {
	key      string
	page     *content.Page
	diags    []content.Diagnostic
	document *quickedit.Document
}

func parseOne(renderer *content.Renderer, cache *parseCache, save func(string, *content.Page, []content.Diagnostic, *quickedit.Document), job parseJob, wantQuickEdit bool) parseResult {
	result := parseResult{job: job}
	if job.override {
		result.page, result.diags, result.err = parseContentSource(renderer, job.rel, job.lang, []byte(job.source))
		if result.err == nil && wantQuickEdit && !strings.EqualFold(filepath.Ext(job.rel), ".mpd") {
			document, extractErr := quickedit.Extract([]byte(job.source))
			if extractErr != nil {
				result.err = fmt.Errorf("prepare quick editing for %s: %w", job.rel, extractErr)
				return result
			}
			result.quickEdit = &document
		}
		return result
	}

	source, modified, err := readSourceWithModTime(job.sourcePath)
	if err != nil {
		result.err = err
		return result
	}
	key := ""
	if cache != nil {
		key = cache.key(job, source)
		if entry, ok := cache.loadEntry(key); ok {
			entry.Page.LastModified = modified
			result.page, result.diags, result.quickEdit = entry.Page, entry.Diagnostics, entry.QuickEdit
			if !wantQuickEdit {
				result.quickEdit = nil
			}
			if wantQuickEdit && result.quickEdit == nil && !strings.EqualFold(filepath.Ext(job.rel), ".mpd") {
				document, extractErr := quickedit.Extract(source)
				if extractErr != nil {
					result.err = fmt.Errorf("prepare quick editing for %s: %w", job.rel, extractErr)
					return result
				}
				result.quickEdit = &document
				if save != nil {
					save(key, result.page, result.diags, result.quickEdit)
				}
			}
			return result
		}
	}
	result.page, result.diags, result.err = parseContentSource(renderer, job.rel, job.lang, source)
	if result.err == nil {
		result.page.LastModified = modified
		if wantQuickEdit && !strings.EqualFold(filepath.Ext(job.rel), ".mpd") {
			document, extractErr := quickedit.Extract(source)
			if extractErr != nil {
				result.err = fmt.Errorf("prepare quick editing for %s: %w", job.rel, extractErr)
				return result
			}
			result.quickEdit = &document
		}
		if save != nil {
			save(key, result.page, result.diags, result.quickEdit)
		}
	}
	return result
}

func parseContentSource(renderer *content.Renderer, rel, lang string, source []byte) (*content.Page, []content.Diagnostic, error) {
	if !strings.EqualFold(filepath.Ext(rel), ".mpd") {
		return renderer.ParseBytes(rel, lang, source)
	}
	document := mpd.Parse(rel, source)
	markdown, err := mpd.Markdown(document)
	diagnostics := make([]content.Diagnostic, 0, len(document.Diagnostics))
	for _, diagnostic := range document.Diagnostics {
		severity := "warning"
		if diagnostic.Severity == mpd.SeverityError {
			severity = "error"
		}
		diagnostics = append(diagnostics, content.Diagnostic{
			Severity: severity,
			Code:     diagnostic.Code,
			File:     rel,
			Line:     int(diagnostic.Position.Line),
			Message:  diagnostic.Message,
		})
	}
	if err != nil {
		return nil, diagnostics, err
	}
	page, markdownDiagnostics, err := renderer.ParseBytes(rel, lang, markdown)
	return page, append(diagnostics, markdownDiagnostics...), err
}

// readSourceWithModTime reads a source file and its modification time from a
// single open file handle, avoiding a separate stat syscall per page and sizing
// the read buffer from the file length so the content is read in one call.
func readSourceWithModTime(path string) ([]byte, time.Time, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer file.Close()
	var (
		modified time.Time
		size     int
	)
	if info, statErr := file.Stat(); statErr == nil {
		modified = info.ModTime().UTC()
		if s := info.Size(); s > 0 {
			size = int(s)
		}
	}
	buf := bytes.NewBuffer(make([]byte, 0, size+bytes.MinRead))
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, time.Time{}, err
	}
	return buf.Bytes(), modified, nil
}
