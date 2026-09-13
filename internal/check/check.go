package check

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

type BrokenLink struct{ File, Href, Reason string }

// Run validates all local references in the generated site. The output is
// indexed once before references are checked so repeated navigation links do
// not cause repeated stat, read, and fragment scans.
func Run(output string) ([]BrokenLink, error) {
	collector := NewCollector()
	if err := collector.IndexOutput(output); err != nil {
		return nil, err
	}
	return collector.Finalize(), nil
}

// Collector indexes generated output while it is produced. It avoids reading
// rendered HTML back from disk during a build check. Call Finalize only after
// every generated page and static target has been registered.
type Collector struct{ index siteIndex }

// HTMLIndex is the reusable result of scanning one generated HTML page. It is
// intentionally free of collector state so render workers can build it in
// parallel and a caller can commit the finished indexes deterministically.
type HTMLIndex struct {
	Refs []string
	IDs  map[string]struct{}
}

func NewCollector() *Collector { return &Collector{} }

// Reset starts a fresh output index. The output directory must be the same
// directory that AddHTML and IndexRemaining describe.
func (collector *Collector) Reset(output string) {
	collector.index = newSiteIndex(output)
}

// AddFile records a generated or copied file relative to the output root.
func (collector *Collector) AddFile(relative string) {
	collector.index.addFile(cleanKey(relative))
}

// AddHTML records one generated HTML page directly from its rendered bytes.
// It is safe to replace a previously indexed static HTML file with the same
// relative path.
func (collector *Collector) AddHTML(relative string, data []byte) {
	collector.AddParsedHTML(relative, ParseHTML(data))
}

// AddParsedHTML records HTML that has already been tokenized by a render
// worker. Collector mutation remains serial, while HTML parsing no longer
// occupies the build's critical path after rendering completes.
func (collector *Collector) AddParsedHTML(relative string, parsed HTMLIndex) {
	key := cleanKey(relative)
	collector.index.addFile(key)
	collector.index.ids[key] = parsed.IDs
	page := indexedPage{key: key, rel: filepath.ToSlash(relative), refs: parsed.Refs}
	if position, exists := collector.index.pagePositions[key]; exists {
		collector.index.pages[position] = page
		return
	}
	collector.index.pagePositions[key] = len(collector.index.pages)
	collector.index.pages = append(collector.index.pages, page)
}

// IndexOutput builds a complete index of an existing output directory. It is
// used by the standalone `mpress check` command.
func (collector *Collector) IndexOutput(output string) error {
	collector.Reset(output)
	return collector.IndexRemaining()
}

// IndexRemaining inventories files that were not already supplied with
// AddHTML. It is used by the standalone checker to scan an existing site.
func (collector *Collector) IndexRemaining() error {
	return collector.indexPath(collector.index.output, false)
}

// IndexDirectory indexes an existing directory relative to the output root.
// It is useful for mounted version snapshots, where M-Press did not render
// the individual files in the current build.
func (collector *Collector) IndexDirectory(relative string) error {
	path := filepath.Join(collector.index.output, filepath.FromSlash(cleanKey(relative)))
	return collector.indexPath(path, true)
}

// FileCount returns the number of output files registered with the collector.
func (collector *Collector) FileCount() int { return len(collector.index.files) }

func (collector *Collector) indexPath(root string, allowMissing bool) error {
	index := &collector.index
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(index.output, path)
		if err != nil {
			return err
		}
		key := cleanKey(rel)
		if entry.IsDir() {
			index.addDir(key)
			return nil
		}
		index.addFile(key)
		if filepath.Ext(path) != ".html" {
			return nil
		}
		if _, indexed := index.ids[key]; indexed {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		collector.AddHTML(filepath.ToSlash(rel), data)
		return nil
	})
	if allowMissing && os.IsNotExist(err) {
		return nil
	}
	return err
}

// Finalize resolves collected references after all targets are known. Pages
// are sorted by generated path so failures remain deterministic regardless of
// the order in which a builder supplied them.
func (collector *Collector) Finalize() []BrokenLink {
	index := &collector.index
	pages := append([]indexedPage(nil), index.pages...)
	sort.SliceStable(pages, func(left, right int) bool {
		return pages[left].rel < pages[right].rel
	})
	var broken []BrokenLink
	for _, page := range pages {
		for _, href := range page.refs {
			if skip(href) {
				continue
			}
			target, fragment := resolveTarget(page.key, href)
			target = index.normaliseTarget(target)
			if !index.hasTarget(target) {
				broken = append(broken, BrokenLink{page.rel, href, "target not found"})
				continue
			}
			if fragment != "" && !index.hasFragment(target, fragment) {
				broken = append(broken, BrokenLink{page.rel, href, "fragment not found"})
			}
		}
	}
	return broken
}

type indexedPage struct {
	key  string
	rel  string
	refs []string
}

type siteIndex struct {
	output string
	files  map[string]struct{}
	dirs   map[string]struct{}
	ids    map[string]map[string]struct{}
	pages  []indexedPage

	// Non-HTML fragments are unusual, but the previous validator inspected
	// their bytes. Cache those reads lazily to preserve that behaviour without
	// reading every binary asset during indexing.
	nonHTMLBodies map[string][]byte
	pagePositions map[string]int
}

func newSiteIndex(output string) siteIndex {
	return siteIndex{
		output:        output,
		files:         make(map[string]struct{}),
		dirs:          make(map[string]struct{}),
		ids:           make(map[string]map[string]struct{}),
		nonHTMLBodies: make(map[string][]byte),
		pagePositions: make(map[string]int),
	}
}

func (index *siteIndex) addFile(key string) {
	index.files[key] = struct{}{}
	index.addParents(key)
}

func (index *siteIndex) addDir(key string) {
	index.dirs[key] = struct{}{}
	index.addParents(key)
}

func (index *siteIndex) addParents(key string) {
	directory := filepath.Dir(filepath.FromSlash(key))
	for {
		clean := cleanKey(directory)
		index.dirs[clean] = struct{}{}
		if clean == "" {
			return
		}
		directory = filepath.Dir(directory)
	}
}

func (index siteIndex) hasTarget(key string) bool {
	if _, ok := index.files[key]; ok {
		return true
	}
	return false
}

func (index siteIndex) normaliseTarget(key string) string {
	if _, ok := index.dirs[key]; ok {
		return cleanKey(filepath.Join(filepath.FromSlash(key), "index.html"))
	}
	if filepath.Ext(filepath.FromSlash(key)) == "" {
		return cleanKey(filepath.Join(filepath.FromSlash(key), "index.html"))
	}
	return key
}

func (index *siteIndex) hasFragment(key, fragment string) bool {
	if ids, ok := index.ids[key]; ok {
		_, found := ids[fragment]
		return found
	}
	if body, ok := index.nonHTMLBodies[key]; ok {
		return bytesContainID(body, fragment)
	}

	body, err := os.ReadFile(filepath.Join(index.output, filepath.FromSlash(key)))
	if err != nil {
		index.nonHTMLBodies[key] = []byte{}
		return false
	}
	index.nonHTMLBodies[key] = body
	return bytesContainID(body, fragment)
}

func ParseHTML(data []byte) HTMLIndex {
	index := HTMLIndex{IDs: make(map[string]struct{})}
	tokenizer := html.NewTokenizer(bytes.NewReader(data))
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			return index
		}
		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}
		_, moreAttrs := tokenizer.TagName()
		for moreAttrs {
			key, value, more := tokenizer.TagAttr()
			switch {
			case bytes.Equal(key, []byte("href")), bytes.Equal(key, []byte("src")):
				index.Refs = append(index.Refs, string(value))
			case bytes.Equal(key, []byte("id")):
				index.IDs[string(value)] = struct{}{}
			}
			moreAttrs = more
		}
	}
}

func resolveTarget(sourceKey, href string) (target, fragment string) {
	path := href
	if index := strings.IndexByte(path, '#'); index >= 0 {
		fragment = path[index+1:]
		path = path[:index]
	}
	if index := strings.IndexByte(path, '?'); index >= 0 {
		path = path[:index]
	}
	if strings.Contains(path, "%") {
		decoded, err := url.PathUnescape(path)
		if err != nil {
			return "", ""
		}
		path = decoded
	}
	if strings.Contains(fragment, "%") {
		decoded, err := url.PathUnescape(fragment)
		if err != nil {
			return "", ""
		}
		fragment = decoded
	}
	if path == "" {
		target = sourceKey
	} else if strings.HasPrefix(path, "/") {
		target = cleanKey(strings.TrimPrefix(path, "/"))
	} else {
		sourceDir := filepath.Dir(filepath.FromSlash(sourceKey))
		target = cleanKey(filepath.Join(sourceDir, filepath.FromSlash(path)))
	}
	return target, fragment
}

func cleanKey(path string) string {
	if path == "." || path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func bytesContainID(data []byte, id string) bool {
	s := string(data)
	return strings.Contains(s, `id="`+id+`"`) || strings.Contains(s, `id='`+id+`'`)
}

func skip(h string) bool {
	if strings.HasPrefix(h, "#") {
		return false
	}
	if strings.HasPrefix(h, "//") || hasURIScheme(h) {
		return true
	}
	if !strings.Contains(h, "%") {
		return false
	}
	_, err := url.PathUnescape(h)
	return err != nil
}

func hasURIScheme(value string) bool {
	for index, character := range value {
		switch {
		case character == ':':
			return index > 0
		case index == 0 && ((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')):
			continue
		case index > 0 && ((character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '+' || character == '-' || character == '.'):
			continue
		default:
			return false
		}
	}
	return false
}

func Format(items []BrokenLink) string {
	var b strings.Builder
	for _, x := range items {
		fmt.Fprintf(&b, "%s: %s (%s)\n", x.File, x.Href, x.Reason)
	}
	return b.String()
}
