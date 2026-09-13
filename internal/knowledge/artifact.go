// Package knowledge builds and serves the read-only knowledge representation
// of an M-Press site. The artifact is deterministic and contains no credentials
// or authoring controls.
package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	Schema        = 2
	Directory     = "knowledge"
	ManifestFile  = "manifest.json"
	PagesFile     = "pages.json"
	ChunksFile    = "chunks.json"
	IndexFile     = "index.json"
	maxChunkWords = 500
	chunkOverlap  = 40
)

type Manifest struct {
	Schema          int           `json:"schema"`
	Title           string        `json:"title"`
	Description     string        `json:"description,omitempty"`
	BaseURL         string        `json:"baseUrl,omitempty"`
	DefaultLanguage string        `json:"defaultLanguage"`
	Languages       []string      `json:"languages"`
	CurrentVersion  string        `json:"currentVersion"`
	Digest          string        `json:"digest"`
	Pages           []PageSummary `json:"pages"`
	Artifacts       Artifacts     `json:"artifacts"`
}

type Artifacts struct {
	Pages  string `json:"pages"`
	Chunks string `json:"chunks"`
	Index  string `json:"index"`
}

type PageSummary struct {
	ID          string   `json:"id"`
	ResourceURI string   `json:"resourceUri"`
	Language    string   `json:"language"`
	Version     string   `json:"version"`
	Route       string   `json:"route"`
	URL         string   `json:"url"`
	Source      string   `json:"source"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ChunkCount  int      `json:"chunkCount"`
}

type Page struct {
	PageSummary
	Text string `json:"text"`
}

type Chunk struct {
	ID          string   `json:"id"`
	PageID      string   `json:"pageId"`
	ResourceURI string   `json:"resourceUri"`
	Language    string   `json:"language"`
	Version     string   `json:"version"`
	Route       string   `json:"route"`
	URL         string   `json:"url"`
	Source      string   `json:"source"`
	PageTitle   string   `json:"pageTitle"`
	Title       string   `json:"title"`
	HeadingID   string   `json:"headingId,omitempty"`
	HeadingPath []string `json:"headingPath,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Part        int      `json:"part,omitempty"`
	Text        string   `json:"text"`
}

type Index struct {
	Schema int         `json:"schema"`
	Terms  []IndexTerm `json:"terms"`
}

type IndexTerm struct {
	Term     string    `json:"term"`
	Postings []Posting `json:"postings"`
}

type Posting struct {
	ChunkID string `json:"chunkId"`
	Text    int    `json:"text,omitempty"`
	Page    int    `json:"page,omitempty"`
	Title   int    `json:"title,omitempty"`
	Route   int    `json:"route,omitempty"`
	Tags    int    `json:"tags,omitempty"`
}

// Generate writes the complete portable knowledge bundle beneath output.
func Generate(output string, cfg config.Config, pagesByLanguage map[string][]*content.Page) error {
	return generate(output, cfg, pagesByLanguage, 20<<20)
}

func generate(output string, cfg config.Config, pagesByLanguage map[string][]*content.Page, compressionThreshold int) error {
	version := strings.TrimSpace(cfg.Version.Current)
	if version == "" {
		version = "current"
	}
	var pages []Page
	var chunks []Chunk
	for _, language := range cfg.Site.Languages {
		languagePages := append([]*content.Page(nil), pagesByLanguage[language]...)
		sort.Slice(languagePages, func(i, j int) bool {
			if languagePages[i].URLPath != languagePages[j].URLPath {
				return languagePages[i].URLPath < languagePages[j].URLPath
			}
			return languagePages[i].SourcePath < languagePages[j].SourcePath
		})
		for _, sourcePage := range languagePages {
			page := makePage(cfg, language, version, sourcePage)
			pageChunks, err := makeChunks(page, sourcePage.HTML)
			if err != nil {
				return fmt.Errorf("build knowledge chunks for %s: %w", sourcePage.SourcePath, err)
			}
			page.ChunkCount = len(pageChunks)
			pages = append(pages, page)
			chunks = append(chunks, pageChunks...)
		}
	}
	index := makeIndex(chunks)
	pagesData, err := marshalArtifact(pages)
	if err != nil {
		return err
	}
	chunksData, err := marshalArtifact(chunks)
	if err != nil {
		return err
	}
	// The inverted index is substantially larger than the page and chunk
	// metadata. Keep it compact so large documentation sets remain below static
	// hosting per-file limits without changing the portable JSON representation.
	indexData, err := marshalCompactArtifact(index)
	if err != nil {
		return err
	}
	digestHash := sha256.New()
	_, _ = digestHash.Write(pagesData)
	_, _ = digestHash.Write(chunksData)
	_, _ = digestHash.Write(indexData)
	summaries := make([]PageSummary, len(pages))
	for i := range pages {
		summaries[i] = pages[i].PageSummary
	}
	files, artifacts, schema, err := encodeArtifacts(pagesData, chunksData, indexData, compressionThreshold)
	if err != nil {
		return err
	}
	manifest := Manifest{
		Schema: schema, Title: cfg.Site.Title, Description: cfg.Site.Description,
		BaseURL: strings.TrimRight(cfg.Site.BaseURL, "/"), DefaultLanguage: cfg.Site.DefaultLanguage,
		Languages: append([]string(nil), cfg.Site.Languages...), CurrentVersion: version,
		Digest: hex.EncodeToString(digestHash.Sum(nil)), Pages: summaries,
		Artifacts: artifacts,
	}
	manifestData, err := marshalArtifact(manifest)
	if err != nil {
		return err
	}
	directory := filepath.Join(output, Directory)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	files[ManifestFile] = manifestData
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(directory, name), data, 0o644); err != nil {
			return err
		}
	}
	return removeStaleArtifacts(directory, files)
}

func marshalArtifact(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func marshalCompactArtifact(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func makePage(cfg config.Config, language, version string, source *content.Page) Page {
	route := "/" + strings.Trim(source.URLPath, "/")
	if route != "/" {
		route += "/"
	}
	if language != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
		route = "/" + language + route
	}
	url := route
	if cfg.Site.BaseURL != "" {
		url = strings.TrimRight(cfg.Site.BaseURL, "/") + route
	}
	id := stableID("page", language, version, source.URLPath)
	return Page{PageSummary: PageSummary{
		ID: id, ResourceURI: "mpress://knowledge/page/" + id, Language: language, Version: version,
		Route: route, URL: url, Source: source.SourcePath, Title: source.Title,
		Description: source.Description, Tags: cleanTags(source.Meta.Tags),
	}, Text: normaliseText(source.PlainText)}
}

type section struct {
	level int
	id    string
	title string
	path  []string
	text  string
}

func makeChunks(page Page, renderedHTML string) ([]Chunk, error) {
	sections, err := sectionsFromHTML(renderedHTML, page.Title)
	if err != nil {
		return nil, err
	}
	if len(sections) == 0 && page.Text != "" {
		sections = []section{{title: page.Title, text: page.Text}}
	}
	var chunks []Chunk
	for _, section := range sections {
		parts := splitWords(section.text, maxChunkWords, chunkOverlap)
		for partIndex, text := range parts {
			part := 0
			if len(parts) > 1 {
				part = partIndex + 1
			}
			id := stableID("chunk", page.ID, section.id, fmt.Sprint(part))
			url := page.URL
			if section.id != "" {
				url += "#" + section.id
			}
			chunks = append(chunks, Chunk{
				ID: id, PageID: page.ID, ResourceURI: "mpress://knowledge/section/" + id,
				Language: page.Language, Version: page.Version, Route: page.Route, URL: url,
				Source: page.Source, PageTitle: page.Title, Title: section.title,
				HeadingID: section.id, HeadingPath: append([]string(nil), section.path...),
				Tags: append([]string(nil), page.Tags...), Part: part, Text: text,
			})
		}
	}
	return chunks, nil
}

func sectionsFromHTML(renderedHTML, pageTitle string) ([]section, error) {
	contextNode := &nethtml.Node{Type: nethtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := nethtml.ParseFragment(strings.NewReader(renderedHTML), contextNode)
	if err != nil {
		return nil, err
	}
	var sections []section
	current := section{title: pageTitle}
	var text strings.Builder
	var headings [7]string
	flush := func() {
		current.text = normaliseText(text.String())
		if current.text != "" {
			sections = append(sections, current)
		}
		text.Reset()
	}
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode {
			name := strings.ToLower(node.Data)
			if name == "script" || name == "style" || name == "svg" {
				return
			}
			if level := headingLevel(name); level > 0 {
				flush()
				title := normaliseText(nodeText(node))
				id := attribute(node, "id")
				headings[level] = title
				for index := level + 1; index < len(headings); index++ {
					headings[index] = ""
				}
				path := make([]string, 0, level)
				for index := 1; index <= level; index++ {
					if headings[index] != "" {
						path = append(path, headings[index])
					}
				}
				current = section{level: level, id: id, title: title, path: path}
				return
			}
		}
		if node.Type == nethtml.TextNode {
			text.WriteString(node.Data)
			text.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if node.Type == nethtml.ElementNode && isBlock(node.Data) {
			text.WriteByte('\n')
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	flush()
	return sections, nil
}

func headingLevel(name string) int {
	if len(name) == 2 && name[0] == 'h' && name[1] >= '1' && name[1] <= '6' {
		return int(name[1] - '0')
	}
	return 0
}

func nodeText(node *nethtml.Node) string {
	var value strings.Builder
	var walk func(*nethtml.Node)
	walk = func(current *nethtml.Node) {
		if current.Type == nethtml.TextNode {
			value.WriteString(current.Data)
			value.WriteByte(' ')
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return value.String()
}

func attribute(node *nethtml.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

func isBlock(name string) bool {
	switch strings.ToLower(name) {
	case "p", "div", "section", "article", "aside", "li", "pre", "blockquote", "table", "tr", "details":
		return true
	default:
		return false
	}
}

func splitWords(text string, maximum, overlap int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	if len(words) <= maximum {
		return []string{strings.Join(words, " ")}
	}
	var result []string
	for start := 0; start < len(words); {
		end := start + maximum
		if end > len(words) {
			end = len(words)
		}
		result = append(result, strings.Join(words[start:end], " "))
		if end == len(words) {
			break
		}
		start = end - overlap
	}
	return result
}

func makeIndex(chunks []Chunk) Index {
	terms := make(map[string][]Posting)
	for _, chunk := range chunks {
		counts := make(map[string]*Posting)
		add := func(value string, field func(*Posting)) {
			for _, token := range tokens(value) {
				posting := counts[token]
				if posting == nil {
					posting = &Posting{ChunkID: chunk.ID}
					counts[token] = posting
				}
				field(posting)
			}
		}
		add(chunk.Text, func(posting *Posting) { posting.Text++ })
		add(chunk.PageTitle, func(posting *Posting) { posting.Page++ })
		add(chunk.Title+" "+strings.Join(chunk.HeadingPath, " "), func(posting *Posting) { posting.Title++ })
		add(strings.NewReplacer("-", " ", "/", " ").Replace(chunk.Route), func(posting *Posting) { posting.Route++ })
		add(strings.Join(chunk.Tags, " "), func(posting *Posting) { posting.Tags++ })
		keys := make([]string, 0, len(counts))
		for term := range counts {
			keys = append(keys, term)
		}
		sort.Strings(keys)
		for _, term := range keys {
			terms[term] = append(terms[term], *counts[term])
		}
	}
	keys := make([]string, 0, len(terms))
	for term := range terms {
		keys = append(keys, term)
	}
	sort.Strings(keys)
	index := Index{Schema: 1, Terms: make([]IndexTerm, 0, len(keys))}
	for _, term := range keys {
		index.Terms = append(index.Terms, IndexTerm{Term: term, Postings: terms[term]})
	}
	return index
}

func tokens(value string) []string {
	raw := strings.FieldsFunc(strings.ToLower(value), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
	result := make([]string, 0, len(raw))
	for _, token := range raw {
		if searchStopWord(token) {
			continue
		}
		runes := []rune(token)
		if containsCJK(runes) {
			if len(runes) == 1 {
				result = append(result, token)
				continue
			}
			for index := 0; index < len(runes)-1; index++ {
				result = append(result, string(runes[index:index+2]))
			}
			continue
		}
		normalised := stemEnglish(token)
		result = append(result, normalised)
		switch normalised {
		case "autostart":
			result = append(result, "automatic", "launch", "login")
		case "automatic":
			result = append(result, "autostart")
		}
	}
	return result
}

func searchStopWord(value string) bool {
	switch value {
	case "a", "an", "and", "are", "as", "at", "be", "by", "for", "from", "in", "into", "is", "it", "of", "on", "or", "that", "the", "this", "to", "with", "your":
		return true
	default:
		return false
	}
}

func containsCJK(value []rune) bool {
	for _, character := range value {
		switch {
		case character >= 0x3400 && character <= 0x4DBF,
			character >= 0x4E00 && character <= 0x9FFF,
			character >= 0xF900 && character <= 0xFAFF:
			return true
		}
	}
	return false
}

func stemEnglish(value string) string {
	if len(value) <= 3 {
		return value
	}
	switch {
	case strings.HasSuffix(value, "ies") && len(value) > 4:
		return strings.TrimSuffix(value, "ies") + "y"
	case strings.HasSuffix(value, "ally") && len(value) > 6:
		return strings.TrimSuffix(value, "ally") + "al"
	case strings.HasSuffix(value, "sses"):
		return strings.TrimSuffix(value, "es")
	case strings.HasSuffix(value, "ing") && len(value) > 5:
		return trimDoubledSuffix(strings.TrimSuffix(value, "ing"))
	case strings.HasSuffix(value, "ed") && len(value) > 4:
		return trimDoubledSuffix(strings.TrimSuffix(value, "ed"))
	case strings.HasSuffix(value, "es") && len(value) > 4:
		return strings.TrimSuffix(value, "es")
	case strings.HasSuffix(value, "s") && !strings.HasSuffix(value, "ss"):
		return strings.TrimSuffix(value, "s")
	case strings.HasSuffix(value, "e") && len(value) > 4:
		return strings.TrimSuffix(value, "e")
	default:
		return value
	}
}

func trimDoubledSuffix(value string) string {
	if len(value) > 2 && value[len(value)-1] == value[len(value)-2] {
		return value[:len(value)-1]
	}
	return value
}

func stableID(parts ...string) string {
	hash := sha256.New()
	for index, part := range parts {
		if index > 0 {
			_, _ = hash.Write([]byte{0})
		}
		_, _ = hash.Write([]byte(part))
	}
	return hex.EncodeToString(hash.Sum(nil))[:24]
}

func normaliseText(value string) string { return strings.Join(strings.Fields(value), " ") }

func cleanTags(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

// Load reads and verifies a generated knowledge bundle.
func Load(output string) (*Store, error) {
	directory := filepath.Join(output, Directory)
	var manifest Manifest
	if err := readJSON(filepath.Join(directory, ManifestFile), &manifest); err != nil {
		return nil, err
	}
	if manifest.Schema != 1 && manifest.Schema != Schema {
		return nil, fmt.Errorf("knowledge schema %d is not supported", manifest.Schema)
	}
	var pages []Page
	var chunks []Chunk
	var index Index
	pageData, err := readArtifact(filepath.Join(directory, manifest.Artifacts.Pages))
	if err != nil {
		return nil, err
	}
	chunkData, err := readArtifact(filepath.Join(directory, manifest.Artifacts.Chunks))
	if err != nil {
		return nil, err
	}
	indexData, err := readArtifact(filepath.Join(directory, manifest.Artifacts.Index))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(pageData, &pages); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(chunkData, &chunks); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, err
	}
	digestHash := sha256.New()
	_, _ = digestHash.Write(pageData)
	_, _ = digestHash.Write(chunkData)
	_, _ = digestHash.Write(indexData)
	if digest := hex.EncodeToString(digestHash.Sum(nil)); digest != manifest.Digest {
		return nil, errors.New("knowledge artifact digest does not match manifest")
	}
	return newStore(manifest, pages, chunks, index), nil
}

// LoadAll loads the current artifact and any captured version artifacts mounted
// beneath the generated site. Mounted snapshots are relabelled in memory so
// version filters and resource identifiers cannot collide with the current
// corpus or with each other.
func LoadAll(output string) (*Store, error) {
	current, err := Load(output)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(output, "versions"))
	if os.IsNotExist(err) {
		return current, nil
	}
	if err != nil {
		return nil, err
	}
	pages := append([]Page(nil), current.Pages...)
	chunks := append([]Chunk(nil), current.Chunks...)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		root := filepath.Join(output, "versions", entry.Name())
		if _, statErr := os.Stat(filepath.Join(root, Directory, ManifestFile)); os.IsNotExist(statErr) {
			continue
		} else if statErr != nil {
			return nil, statErr
		}
		snapshot, loadErr := Load(root)
		if loadErr != nil {
			return nil, fmt.Errorf("load knowledge version %s: %w", entry.Name(), loadErr)
		}
		versionPages, versionChunks := relabelVersion(snapshot.Pages, snapshot.Chunks, entry.Name(), current.Manifest.BaseURL)
		pages = append(pages, versionPages...)
		chunks = append(chunks, versionChunks...)
	}
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Version != pages[j].Version {
			return pages[i].Version < pages[j].Version
		}
		if pages[i].Language != pages[j].Language {
			return pages[i].Language < pages[j].Language
		}
		return pages[i].Route < pages[j].Route
	})
	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].Version != chunks[j].Version {
			return chunks[i].Version < chunks[j].Version
		}
		if chunks[i].Language != chunks[j].Language {
			return chunks[i].Language < chunks[j].Language
		}
		if chunks[i].Route != chunks[j].Route {
			return chunks[i].Route < chunks[j].Route
		}
		return chunks[i].ID < chunks[j].ID
	})
	return newStore(current.Manifest, pages, chunks, makeIndex(chunks)), nil
}

func relabelVersion(pages []Page, chunks []Chunk, version, baseURL string) ([]Page, []Chunk) {
	type identity struct{ id, route, url string }
	pageIDs := make(map[string]identity, len(pages))
	relabelledPages := make([]Page, len(pages))
	for index, page := range pages {
		oldID := page.ID
		originalRoute := page.Route
		page.Version = version
		page.ID = stableID("page", page.Language, version, strings.Trim(originalRoute, "/"))
		page.ResourceURI = "mpress://knowledge/page/" + page.ID
		page.Route = "/versions/" + version + originalRoute
		page.URL = page.Route
		if baseURL != "" {
			page.URL = strings.TrimRight(baseURL, "/") + page.Route
		}
		pageIDs[oldID] = identity{id: page.ID, route: page.Route, url: page.URL}
		relabelledPages[index] = page
	}
	relabelledChunks := make([]Chunk, len(chunks))
	for index, chunk := range chunks {
		page := pageIDs[chunk.PageID]
		chunk.Version = version
		chunk.PageID = page.id
		chunk.ID = stableID("chunk", chunk.PageID, chunk.HeadingID, fmt.Sprint(chunk.Part))
		chunk.ResourceURI = "mpress://knowledge/section/" + chunk.ID
		chunk.Route = page.route
		chunk.URL = page.url
		if chunk.HeadingID != "" {
			chunk.URL += "#" + chunk.HeadingID
		}
		relabelledChunks[index] = chunk
	}
	return relabelledPages, relabelledChunks
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
