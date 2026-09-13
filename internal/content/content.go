package content

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	legacycomponents "github.com/leaanthony/mpress/internal/components"
	"github.com/leaanthony/mpress/internal/highlight"
	"github.com/leaanthony/mpress/internal/icons"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

type Diagnostic struct {
	Severity   string `json:"severity" glint:"severity"`
	Code       string `json:"code" glint:"code"`
	File       string `json:"file,omitempty" glint:"file"`
	Line       int    `json:"line,omitempty" glint:"line"`
	Column     int    `json:"column,omitempty" glint:"column"`
	Message    string `json:"message" glint:"message"`
	Suggestion string `json:"suggestion,omitempty" glint:"suggestion"`
}

type Frontmatter struct {
	Title           string   `yaml:"title" glint:"title"`
	TranslationKey  string   `yaml:"translationKey" glint:"translationKey"`
	Description     string   `yaml:"description" glint:"description"`
	Draft           bool     `yaml:"draft" glint:"draft"`
	Layout          string   `yaml:"layout" glint:"layout"`
	Slug            string   `yaml:"slug" glint:"slug"`
	Order           int      `yaml:"order" glint:"order"`
	Date            string   `yaml:"date" glint:"date"`
	Template        string   `yaml:"template" glint:"template"`
	Banner          Banner   `yaml:"banner" glint:"banner"`
	Hero            Hero     `yaml:"hero" glint:"hero"`
	Tags            []string `yaml:"tags" glint:"tags"`
	Author          string   `yaml:"author" glint:"author"`
	Authors         []string `yaml:"authors" glint:"authors"`
	Image           string   `yaml:"image" glint:"image"`
	ImageMode       string   `yaml:"imageMode" glint:"imageMode"`
	ImageFit        string   `yaml:"imageFit" glint:"imageFit"`
	ImageBackground string   `yaml:"imageBackground" glint:"imageBackground"`
	ImageWidth      int      `yaml:"imageWidth" glint:"imageWidth"`
	ShowTags        *bool    `yaml:"showTags" glint:"showTags"`
	HeadingSize     string   `yaml:"headingSize" glint:"headingSize"`
	// SourcePath preserves the original path relative to an imported content
	// root. It is used for edit links; rendering and routing continue to use
	// Page.SourcePath.
	SourcePath string `yaml:"sourcePath" glint:"sourcePath"`
	Generated  bool   `yaml:"generated" glint:"generated"`
}

type Banner struct {
	Content string `yaml:"content" glint:"content"`
}

type Hero struct {
	Tagline string       `yaml:"tagline" glint:"tagline"`
	Image   HeroImage    `yaml:"image" glint:"image"`
	Actions []HeroAction `yaml:"actions" glint:"actions"`
}

type HeroImage struct {
	Dark  string `yaml:"dark" glint:"dark"`
	Light string `yaml:"light" glint:"light"`
	Alt   string `yaml:"alt" glint:"alt"`
}

type HeroAction struct {
	Text    string `yaml:"text" glint:"text"`
	Link    string `yaml:"link" glint:"link"`
	Icon    string `yaml:"icon" glint:"icon"`
	Variant string `yaml:"variant" glint:"variant"`
}

type Heading struct {
	ID    string `glint:"id"`
	Text  string `glint:"text"`
	Level int    `glint:"level"`
}

type Page struct {
	SourcePath   string      `glint:"sourcePath"`
	Language     string      `glint:"language"`
	URLPath      string      `glint:"urlPath"`
	OutputPath   string      `glint:"outputPath"`
	Title        string      `glint:"title"`
	Description  string      `glint:"description"`
	HTML         string      `glint:"html"`
	PlainText    string      `glint:"plainText"`
	Draft        bool        `glint:"draft"`
	Layout       string      `glint:"layout"`
	Order        int         `glint:"order"`
	Headings     []Heading   `glint:"headings"`
	Meta         Frontmatter `glint:"meta"`
	LastModified time.Time   `glint:"lastModified"`
}

type Renderer struct {
	md goldmark.Markdown
	// legacyFencePrePass restores the superseded string pre-pass. It exists so
	// the migration test can compare both renderers; production always uses the
	// AST renderer in codefence_renderer.go.
	legacyFencePrePass bool
}

// NewRenderer builds the documentation Markdown renderer. Code fences are
// parsed by goldmark and rendered from the AST, so a fence keeps whatever block
// context it was written in.
func NewRenderer() *Renderer {
	return &Renderer{md: goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer, codeFenceExtension{}),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
	)}
}

// newLegacyFencePrePassRenderer rebuilds the superseded renderer that expanded
// code fences with the renderCodeFences string pass before parsing. It is only
// used by TestFenceRenderingMatchesLegacyPrePass, which pins the migration; it
// and renderCodeFences can be deleted once this change has shipped.
func newLegacyFencePrePassRenderer() *Renderer {
	return &Renderer{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer),
			goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
			goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
		),
		legacyFencePrePass: true,
	}
}

func (r *Renderer) ParseFile(path, rel, lang string) (*Page, []Diagnostic, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	page, diagnostics, err := r.ParseBytes(rel, lang, data)
	if err == nil {
		if info, statErr := os.Stat(path); statErr == nil {
			page.LastModified = info.ModTime().UTC()
		}
	}
	return page, diagnostics, err
}

// ParseBytes parses immutable Markdown bytes without copying them into an
// intermediate string. The parser only reads the source during this call.
func (r *Renderer) ParseBytes(rel, lang string, source []byte) (*Page, []Diagnostic, error) {
	return r.Parse(rel, lang, bytesToString(source))
}

func (r *Renderer) Parse(rel, lang, source string) (*Page, []Diagnostic, error) {
	meta, body, err := splitFrontmatter(source)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", rel, err)
	}
	diagnostics := validateSourceText(rel, body)
	processed, componentDiagnostics := r.processComponents(rel, lang, body)
	diagnostics = append(diagnostics, componentDiagnostics...)
	if r.legacyFencePrePass {
		processed = renderCodeFences(processed)
	}
	var out bytes.Buffer
	if err := r.md.Convert(stringToBytes(processed), &out); err != nil {
		return nil, diagnostics, err
	}
	htmlBody := out.String()
	pageHeadings := headings(htmlBody)
	title := strings.TrimSpace(meta.Title)
	if title == "" && len(pageHeadings) > 0 {
		title = pageHeadings[0].Text
		// A leading level-one heading is the conventional page title when no
		// explicit title is provided. The page template renders that title in its
		// header, so retaining the same H1 in the body would display it twice.
		if pageHeadings[0].Level == 1 {
			htmlBody = removeFirstH1(htmlBody)
			pageHeadings = pageHeadings[1:]
		}
	}
	if title == "" {
		title = titleFromFilename(rel)
		diagnostics = append(diagnostics, Diagnostic{Severity: "warning", Code: "title-fallback", File: rel, Message: "page has no title; using filename"})
	}
	layout := meta.Layout
	if layout == "" && meta.Template == "splash" {
		layout = "landing"
	}
	if layout == "" && filepath.ToSlash(rel) == "blog/index.md" {
		layout = "blog-index"
	}
	if layout != "landing" && layout != "blog-index" && (strings.TrimSpace(meta.Title) == "" || firstH1MatchesTitle(htmlBody, title)) {
		htmlBody = normaliseDocumentHeadings(htmlBody, title)
	}
	htmlBody = uniqueHeadingIDs(htmlBody)
	htmlBody = labelTaskCheckboxes(htmlBody)
	htmlBody = makeScrollableRegionsFocusable(htmlBody)
	pageHeadings = headings(htmlBody)
	diagnostics = append(diagnostics, headingDiagnostics(rel, pageHeadings)...)
	url := routeFor(rel, meta.Slug)
	page := &Page{SourcePath: filepath.ToSlash(rel), Language: lang, URLPath: url, OutputPath: outputFor(url), Title: title,
		Description: meta.Description, HTML: htmlBody, PlainText: stripHTML(htmlBody), Draft: meta.Draft, Layout: layout, Order: meta.Order, Meta: meta}
	page.Headings = pageHeadings
	for i := range diagnostics {
		if diagnostics[i].File == "" {
			diagnostics[i].File = rel
		}
	}
	return page, diagnostics, nil
}

func removeFirstH1(rendered string) string {
	start := strings.Index(rendered, "<h1")
	if start < 0 {
		return rendered
	}
	openEnd := strings.IndexByte(rendered[start:], '>')
	if openEnd < 0 {
		return rendered
	}
	contentStart := start + openEnd + 1
	closeStart := strings.Index(rendered[contentStart:], "</h1>")
	if closeStart < 0 {
		return rendered
	}
	end := contentStart + closeStart + len("</h1>")
	if end < len(rendered) && rendered[end] == '\n' {
		end++
	}
	return rendered[:start] + rendered[end:]
}

var (
	emptyLinkLabelRE   = regexp.MustCompile(`(?m)\[\s*\](?:\(|\[)`)
	unnamedImageLinkRE = regexp.MustCompile(`(?m)\[\s*!\[\s*\]\([^)]*\)\s*\]\(`)
)

func validateSourceText(rel, source string) []Diagnostic {
	var diagnostics []Diagnostic
	for offset, value := range source {
		if value >= 0x20 || value == '\n' || value == '\r' || value == '\t' {
			continue
		}
		line, column := sourcePosition(source, offset)
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "invalid-control-character", File: rel, Line: line, Column: column, Message: fmt.Sprintf("control character U+%04X is not permitted in documentation text", value), Suggestion: "Remove the control character or replace it with visible text."})
	}
	prose := maskCodeForValidation(source)
	for _, match := range emptyLinkLabelRE.FindAllStringIndex(prose, -1) {
		if match[0] > 0 && source[match[0]-1] == '!' {
			continue
		}
		if insideInlineCode(source, match[0]) {
			continue
		}
		line, column := sourcePosition(source, match[0])
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "empty-link-label", File: rel, Line: line, Column: column, Message: "link labels must contain accessible text", Suggestion: "Add concise link text between the square brackets."})
	}
	for _, match := range unnamedImageLinkRE.FindAllStringIndex(prose, -1) {
		line, column := sourcePosition(source, match[0])
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "unnamed-image-link", File: rel, Line: line, Column: column, Message: "image-only links must have an accessible name", Suggestion: "Add useful alternative text to the image or visible text to the link."})
	}
	return diagnostics
}

func maskCodeForValidation(source string) string {
	masked := []byte(source)
	inFence := false
	fence := ""
	offset := 0
	for _, line := range strings.SplitAfter(source, "\n") {
		trimmed := strings.TrimSpace(line)
		marker := ""
		if strings.HasPrefix(trimmed, "```") {
			marker = "```"
		} else if strings.HasPrefix(trimmed, "~~~") {
			marker = "~~~"
		}
		maskLine := inFence || marker != ""
		if marker != "" {
			if !inFence {
				inFence, fence = true, marker
			} else if marker == fence {
				inFence, fence = false, ""
			}
		}
		if maskLine {
			for index := offset; index < offset+len(line); index++ {
				if masked[index] != '\n' && masked[index] != '\r' {
					masked[index] = ' '
				}
			}
		}
		offset += len(line)
	}
	return string(masked)
}

func insideInlineCode(source string, offset int) bool {
	offset = min(offset, len(source))
	lineStart := strings.LastIndex(source[:offset], "\n") + 1
	return strings.Count(source[lineStart:offset], "`")%2 == 1
}

func sourcePosition(source string, offset int) (line, column int) {
	offset = min(offset, len(source))
	line = 1 + strings.Count(source[:offset], "\n")
	lineStart := strings.LastIndex(source[:offset], "\n") + 1
	column = 1 + len([]rune(source[lineStart:offset]))
	return line, column
}

func normaliseDocumentHeadings(body, title string) string {
	firstH1 := regexp.MustCompile(`(?s)<h1 id="[^"]+">(.*?)</h1>`)
	if match := firstH1.FindStringSubmatchIndex(body); match != nil {
		text := strings.TrimSpace(stdhtml.UnescapeString(stripTags(body[match[2]:match[3]], "")))
		if strings.EqualFold(text, strings.TrimSpace(title)) {
			body = body[:match[0]] + body[match[1]:]
		}
	}
	body = strings.ReplaceAll(body, `<h1 id="`, `<h2 id="`)
	body = strings.ReplaceAll(body, `</h1>`, `</h2>`)
	return uniqueHeadingIDs(body)
}

func firstH1MatchesTitle(body, title string) bool {
	firstH1 := regexp.MustCompile(`(?s)<h1 id="[^"]+">(.*?)</h1>`)
	match := firstH1.FindStringSubmatchIndex(body)
	if match == nil {
		return false
	}
	text := strings.TrimSpace(stdhtml.UnescapeString(stripTags(body[match[2]:match[3]], "")))
	return strings.EqualFold(text, strings.TrimSpace(title))
}

func uniqueHeadingIDs(body string) string {
	headingOpen := regexp.MustCompile(`<h[1-6] id="([^"]+)"`)
	seen := map[string]int{"content": 1, "mpress-sidebar": 1, "mpress-search-dialog": 1, "mpress-search-listbox": 1}
	return headingOpen.ReplaceAllStringFunc(body, func(tag string) string {
		match := headingOpen.FindStringSubmatch(tag)
		id := match[1]
		seen[id]++
		if seen[id] == 1 {
			return tag
		}
		return strings.Replace(tag, `id="`+id+`"`, fmt.Sprintf(`id="%s-%d"`, id, seen[id]), 1)
	})
}

func labelTaskCheckboxes(body string) string {
	body = strings.ReplaceAll(body, `<input checked="" disabled="" type="checkbox">`, `<input checked="" disabled="" type="checkbox" aria-label="Completed task">`)
	body = strings.ReplaceAll(body, `<input disabled="" type="checkbox">`, `<input disabled="" type="checkbox" aria-label="Incomplete task">`)
	return body
}

var preOpenTag = regexp.MustCompile(`<pre(?: [^>]*)?>`)

func makeScrollableRegionsFocusable(body string) string {
	return preOpenTag.ReplaceAllStringFunc(body, func(tag string) string {
		if strings.Contains(tag, ` tabindex=`) {
			return tag
		}
		return strings.Replace(tag, "<pre", `<pre tabindex="0"`, 1)
	})
}

func headingDiagnostics(rel string, values []Heading) []Diagnostic {
	var diagnostics []Diagnostic
	previous := 0
	for _, heading := range values {
		if previous > 0 && heading.Level > previous+1 {
			diagnostics = append(diagnostics, Diagnostic{Severity: "warning", Code: "heading-level-jump", File: rel, Message: fmt.Sprintf("heading %q jumps from level %d to level %d", heading.Text, previous, heading.Level)})
		}
		previous = heading.Level
	}
	return diagnostics
}

var (
	codeFenceTitleRE       = regexp.MustCompile(`(?:^|[\s{])title=(?:"([^"]*)"|'([^']*)')`)
	codeFenceLineNumbersRE = regexp.MustCompile(`(?:^|[\s{])lineNumbers=(?:true|"true"|'true')(?:$|[\s}])`)
	codeFenceMarkRE        = regexp.MustCompile(`(?:^|\s)\{([0-9,\-\s]+)\}`)
	codeFenceInsRE         = regexp.MustCompile(`(?:^|\s)ins=\{([0-9,\-\s]+)\}`)
	codeFenceDelRE         = regexp.MustCompile(`(?:^|\s)del=\{([0-9,\-\s]+)\}`)
)

// renderCodeFences gives every fenced block the same production code chrome.
// It also preserves Expressive Code metadata used by Starlight imports.
func renderCodeFences(source string) string {
	if !strings.Contains(source, "```") && !strings.Contains(source, "~~~") {
		return source
	}
	lines := strings.Split(source, "\n")
	var out strings.Builder
	for i := 0; i < len(lines); {
		trimmed := strings.TrimLeft(lines[i], " \t")
		marker, info, ok := annotatedFenceOpening(trimmed)
		if !ok {
			out.WriteString(lines[i])
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
			i++
			continue
		}

		end := i + 1
		for end < len(lines) && !annotatedFenceClosing(lines[end], marker) {
			end++
		}
		codeLines := lines[i+1 : min(end, len(lines))]
		openingIndent := lines[i][:len(lines[i])-len(trimmed)]
		if openingIndent != "" {
			codeLines = removeFenceIndent(codeLines, openingIndent)
		}
		if isTerminalFence(info) {
			out.WriteString(renderTerminalFence(info, codeLines))
		} else {
			out.WriteString(renderAnnotatedCodeFrame(info, codeLines))
		}
		if end >= len(lines) {
			i = len(lines)
			continue
		}
		if end < len(lines)-1 {
			out.WriteByte('\n')
		}
		i = end + 1
	}
	return out.String()
}

func removeFenceIndent(lines []string, indent string) []string {
	result := make([]string, len(lines))
	for i, line := range lines {
		if strings.HasPrefix(line, indent) {
			result[i] = strings.TrimPrefix(line, indent)
			continue
		}
		result[i] = line
	}
	return result
}

func isTerminalFence(info string) bool {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return false
	}
	switch strings.ToLower(fields[0]) {
	case "bash", "sh", "shell", "zsh", "fish", "powershell", "pwsh", "console", "terminal":
		return true
	default:
		return false
	}
}

func renderTerminalFence(info string, lines []string) string {
	meta := map[string]string{"frame": "macos", "prompt": "none"}
	fields := strings.Fields(info)
	if len(fields) > 0 {
		meta["language"] = strings.ToLower(fields[0])
	}
	if title := fenceMetadataValue(codeFenceTitleRE, info); title != "" {
		meta["title"] = title
	} else {
		if len(fields) > 0 {
			switch strings.ToLower(fields[0]) {
			case "powershell", "pwsh":
				meta["title"] = "PowerShell"
			case "console", "terminal":
				meta["title"] = "Terminal"
			default:
				meta["title"] = strings.ToLower(fields[0])
			}
		}
	}
	terminal := legacycomponents.Terminal{Meta: meta}
	_ = terminal.Parse(strings.Join(lines, "\n"))
	markup, _ := terminal.Render()
	return markup
}

func annotatedFenceOpening(line string) (marker, info string, ok bool) {
	if len(line) < 3 || line[0] != '`' && line[0] != '~' {
		return "", "", false
	}
	length := 0
	for length < len(line) && line[length] == line[0] {
		length++
	}
	if length < 3 {
		return "", "", false
	}
	return line[:length], strings.TrimSpace(line[length:]), true
}

func annotatedFenceClosing(line, marker string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < len(marker) || trimmed[0] != marker[0] {
		return false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == marker[0] {
		length++
	}
	return length >= len(marker) && strings.TrimSpace(trimmed[length:]) == ""
}

func hasAnnotatedFenceMetadata(info string) bool {
	return codeFenceTitleRE.MatchString(info) || codeFenceLineNumbersRE.MatchString(info) || codeFenceMarkRE.MatchString(info) || codeFenceInsRE.MatchString(info) || codeFenceDelRE.MatchString(info)
}

func renderAnnotatedCodeFrame(info string, lines []string) string {
	language := strings.Fields(info)
	lang := "text"
	if len(language) > 0 {
		lang = sanitizeCodeLanguage(language[0])
		if lang == "" {
			lang = "text"
		}
	}
	title := fenceMetadataValue(codeFenceTitleRE, info)
	lineNumbers := codeFenceLineNumbersRE.MatchString(info)
	marked := parseCodeLineRanges(fenceMetadataValue(codeFenceMarkRE, info))
	inserted := parseCodeLineRanges(fenceMetadataValue(codeFenceInsRE, info))
	deleted := parseCodeLineRanges(fenceMetadataValue(codeFenceDelRE, info))

	var b strings.Builder
	rawCodeAttribute := escapeCodeAttribute(lines)
	b.Grow(len(rawCodeAttribute)*2 + len(lines)*64 + 256)
	b.WriteString(`<div class="mpress-codeframe`)
	if lineNumbers {
		b.WriteString(` mpress-codeframe-line-numbers`)
	}
	b.WriteString(`"><div class="mpress-codeframe-header">`)
	if title != "" {
		b.WriteString(`<div class="mpress-codeframe-title">` + stdhtml.EscapeString(title) + `</div>`)
	}
	b.WriteString(`<span class="mpress-codeframe-language">` + stdhtml.EscapeString(lang) + `</span>`)
	b.WriteString(icons.CopyButton("mpress-copy mpress-code-copy", "Copy code"))
	b.WriteString(`</div>`)
	b.WriteString(`<pre data-code="` + rawCodeAttribute + `"><code class="language-` + stdhtml.EscapeString(lang) + `">`)
	for i, line := range lines {
		lineNumber := i + 1
		b.WriteString(`<span class="mpress-code-line`)
		if marked[lineNumber] {
			b.WriteString(` mpress-code-line-marked`)
		}
		if inserted[lineNumber] {
			b.WriteString(` mpress-code-line-inserted`)
		}
		if deleted[lineNumber] {
			b.WriteString(` mpress-code-line-deleted`)
		}
		b.WriteString(`">`)
		highlight.WriteLine(&b, lang, line)
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
		b.WriteString(`</span>`)
	}
	b.WriteString(`</code></pre></div>`)
	return b.String()
}

func sanitizeCodeLanguage(value string) string {
	firstInvalid := -1
	for i := 0; i < len(value); i++ {
		if !codeLanguageByte(value[i]) {
			firstInvalid = i
			break
		}
	}
	if firstInvalid < 0 {
		return value
	}
	var result strings.Builder
	result.Grow(len(value))
	result.WriteString(value[:firstInvalid])
	for i := firstInvalid + 1; i < len(value); i++ {
		if codeLanguageByte(value[i]) {
			result.WriteByte(value[i])
		}
	}
	return result.String()
}

func codeLanguageByte(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' ||
		value >= '0' && value <= '9' || value == '_' || value == '+' ||
		value == '.' || value == '-'
}

func escapeCodeAttribute(lines []string) string {
	raw := strings.Join(lines, "\n")
	var result strings.Builder
	result.Grow(len(raw))
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '&':
			result.WriteString("&amp;")
		case '\'':
			result.WriteString("&#39;")
		case '<':
			result.WriteString("&lt;")
		case '>':
			result.WriteString("&gt;")
		case '"':
			result.WriteString("&#34;")
		case '\r':
			result.WriteString("&#10;")
			if i+1 < len(raw) && raw[i+1] == '\n' {
				i++
			}
		case '\n':
			result.WriteString("&#10;")
		default:
			result.WriteByte(raw[i])
		}
	}
	return result.String()
}

func fenceMetadataValue(re *regexp.Regexp, info string) string {
	match := re.FindStringSubmatch(info)
	if len(match) == 0 {
		return ""
	}
	for _, value := range match[1:] {
		if value != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseCodeLineRanges(spec string) map[int]bool {
	lines := map[int]bool{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		bounds := strings.SplitN(part, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
		if err != nil || start < 1 {
			continue
		}
		end := start
		if len(bounds) == 2 {
			if parsed, err := strconv.Atoi(strings.TrimSpace(bounds[1])); err == nil && parsed >= start {
				end = parsed
			}
		}
		for line := start; line <= end; line++ {
			lines[line] = true
		}
	}
	return lines
}

func splitFrontmatter(source string) (Frontmatter, string, error) {
	var meta Frontmatter
	source = strings.ReplaceAll(source, "\r\n", "\n")
	if !strings.HasPrefix(source, "---\n") {
		return meta, source, nil
	}
	end := strings.Index(source[4:], "\n---\n")
	if end < 0 {
		return meta, source, fmt.Errorf("unterminated YAML frontmatter")
	}
	if err := yaml.Unmarshal(stringToBytes(source[4:4+end]), &meta); err != nil {
		return meta, source, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	return meta, source[4+end+5:], nil
}

func Discover(contentDir string, languages []string, defaultLanguage string) (map[string][]string, error) {
	result := make(map[string][]string, len(languages))
	langSet := map[string]bool{}
	for _, l := range languages {
		langSet[l] = true
	}
	err := filepath.WalkDir(contentDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" && ext != ".mpd" {
			return nil
		}
		rel, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if filepath.Base(rel) == "_nav.yaml" {
			return nil
		}
		parts := strings.Split(rel, "/")
		lang := defaultLanguage
		logical := rel
		if len(parts) > 1 && langSet[parts[0]] && parts[0] != defaultLanguage {
			lang = parts[0]
			logical = strings.Join(parts[1:], "/")
		}
		result[lang] = append(result[lang], logical)
		return nil
	})
	for lang := range result {
		sort.Strings(result[lang])
	}
	return result, err
}

func SourcePath(contentDir, lang, defaultLang, rel string) string {
	if lang == defaultLang {
		return filepath.Join(contentDir, filepath.FromSlash(rel))
	}
	return filepath.Join(contentDir, lang, filepath.FromSlash(rel))
}

func routeFor(rel, slug string) string {
	if strings.TrimSpace(slug) != "" {
		return strings.Trim(strings.TrimSpace(slug), "/")
	}
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, filepath.Ext(rel))
	parts := strings.Split(rel, "/")
	for i := range parts {
		parts[i] = strings.TrimLeft(parts[i], "0123456789_- ")
	}
	if parts[len(parts)-1] == "index" {
		parts = parts[:len(parts)-1]
	}
	return strings.Trim(strings.Join(parts, "/"), "/")
}
func outputFor(route string) string {
	if route == "" {
		return "index.html"
	}
	return filepath.ToSlash(filepath.Join(route, "index.html"))
}
func titleFromFilename(rel string) string {
	s := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	s = strings.TrimLeft(s, "0123456789_- ")
	s = strings.ReplaceAll(s, "-", " ")
	return strings.Title(s)
}

func headings(body string) []Heading {
	var result []Heading
	for offset := 0; offset < len(body); {
		relative := strings.Index(body[offset:], "<h")
		if relative < 0 {
			break
		}
		start := offset + relative
		if start+8 > len(body) || body[start+2] < '1' || body[start+2] > '6' || body[start+3:start+8] != ` id="` {
			offset = start + 2
			continue
		}
		idStart := start + 8
		idEnd := strings.IndexByte(body[idStart:], '"')
		if idEnd <= 0 {
			offset = start + 2
			continue
		}
		idEnd += idStart
		if idEnd+1 >= len(body) || body[idEnd+1] != '>' {
			offset = start + 2
			continue
		}
		contentStart := idEnd + 2
		contentEnd, closeEnd := headingClose(body, contentStart)
		if contentEnd < 0 {
			break
		}
		result = append(result, Heading{
			Level: int(body[start+2] - '0'),
			ID:    body[idStart:idEnd],
			Text:  strings.TrimSpace(stdhtml.UnescapeString(stripTags(body[contentStart:contentEnd], ""))),
		})
		offset = closeEnd
	}
	return result
}

func headingClose(body string, offset int) (int, int) {
	for offset < len(body) {
		relative := strings.Index(body[offset:], "</h")
		if relative < 0 {
			return -1, -1
		}
		start := offset + relative
		if start+5 <= len(body) && body[start+3] >= '1' && body[start+3] <= '6' && body[start+4] == '>' {
			return start, start + 5
		}
		offset = start + 3
	}
	return -1, -1
}
func firstHeading(body string) string {
	hs := headings(body)
	if len(hs) > 0 {
		return hs[0].Text
	}
	return ""
}

// stripTags replaces every <...> tag with sep. It matches the former
// `<[^>]+>` regexp exactly - a "<" with at least one following non-">"
// character up to the next ">" is a tag, anything else is literal text - while
// scanning the body once instead of running the regexp engine over it.
func stripTags(s, sep string) string {
	next := strings.IndexByte(s, '<')
	if next < 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for {
		b.WriteString(s[:next])
		end := strings.IndexByte(s[next+1:], '>')
		if end < 0 {
			b.WriteString(s[next:])
			break
		}
		if end == 0 {
			// "<>" is not a tag; emit the "<" and rescan from the ">".
			b.WriteByte('<')
			s = s[next+1:]
		} else {
			b.WriteString(sep)
			s = s[next+1+end+1:]
		}
		next = strings.IndexByte(s, '<')
		if next < 0 {
			b.WriteString(s)
			break
		}
	}
	return b.String()
}

// stripHTML converts rendered page HTML into the plain text used by search
// indexing and blog excerpts. It was previously a regexp replace plus
// strings.Fields/Join, which together dominated cache-cold parsing; the
// hand-rolled passes produce identical output.
func stripHTML(s string) string {
	s = stdhtml.UnescapeString(stripTags(s, " "))
	// Collapse runs of Unicode whitespace to single spaces, exactly like
	// strings.Join(strings.Fields(s), " ") but without building the
	// intermediate field slice.
	var b strings.Builder
	b.Grow(len(s))
	fieldStart := -1
	for i, r := range s {
		if unicode.IsSpace(r) {
			if fieldStart >= 0 {
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(s[fieldStart:i])
				fieldStart = -1
			}
			continue
		}
		if fieldStart < 0 {
			fieldStart = i
		}
	}
	if fieldStart >= 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(s[fieldStart:])
	}
	return b.String()
}

func ParseDate(value string) time.Time { t, _ := time.Parse("2006-01-02", value); return t }
