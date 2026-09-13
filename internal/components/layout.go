package components

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

type markdownLayout struct {
	Meta    map[string]string
	Content string
	Kind    string
}

type DocsPreview struct {
	Meta    map[string]string
	Content string
}

type Headline struct {
	Meta    map[string]string
	Content string
}

type PreviewTabs struct {
	Meta    map[string]string
	Content string
}

// FileTabs presents related source files and their rendered output in one
// compact, keyboard-accessible group. It shares the canonical tab behaviour
// but has a distinct visual treatment for filenames and previews.
type FileTabs struct {
	Meta    map[string]string
	Content string
}

type CompactCallout struct {
	Meta    map[string]string
	Content string
}

type Timeline struct {
	Meta    map[string]string
	Content string
}

type Capabilities struct {
	Meta    map[string]string
	Content string
}

type Resources struct {
	Meta    map[string]string
	Content string
}

func init() {
	Registry["section"] = func(meta map[string]string) Component { return &markdownLayout{Meta: meta, Kind: "section"} }
	Registry["columns"] = func(meta map[string]string) Component { return &markdownLayout{Meta: meta, Kind: "columns"} }
	Registry["column"] = func(meta map[string]string) Component { return &markdownLayout{Meta: meta, Kind: "column"} }
	Registry["actions"] = func(meta map[string]string) Component { return &markdownLayout{Meta: meta, Kind: "actions"} }
	Registry["headline"] = func(meta map[string]string) Component { return &Headline{Meta: meta} }
	Registry["docs-preview"] = func(meta map[string]string) Component { return &DocsPreview{Meta: meta} }
	Registry["preview-tabs"] = func(meta map[string]string) Component { return &PreviewTabs{Meta: meta} }
	Registry["file-tabs"] = func(meta map[string]string) Component { return &FileTabs{Meta: meta} }
	Registry["callout"] = func(meta map[string]string) Component { return &CompactCallout{Meta: meta} }
	Registry["timeline"] = func(meta map[string]string) Component { return &Timeline{Meta: meta} }
	Registry["capabilities"] = func(meta map[string]string) Component { return &Capabilities{Meta: meta} }
	Registry["resources"] = func(meta map[string]string) Component { return &Resources{Meta: meta} }
}

func (h *Headline) Parse(content string) error {
	h.Content = content
	return nil
}

func (h *Headline) Render() (string, error) {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(h.Content), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, html.EscapeString(line))
		}
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("headline requires at least one line")
	}
	return `<h1 class="mpress-headline">` + strings.Join(lines, "<br>") + `</h1>`, nil
}

func renderMarkdownFragment(source string) string {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
	)
	var out bytes.Buffer
	if err := md.Convert([]byte(strings.TrimSpace(source)), &out); err != nil {
		return html.EscapeString(source)
	}
	return out.String()
}

var fileTabCodeBlockRE = regexp.MustCompile(`(?s)<pre><code[^>]*>.*?</code></pre>`)

func renderFileTabFragment(source string) string {
	rendered := renderMarkdownFragment(source)
	// The complete file-tabs component is inserted into a larger Markdown
	// document as HTML. Keep code block payloads on one source line so a second
	// rendering pass cannot interpret headings, lists, or paragraphs inside the
	// literal file. Character references still become real newlines in the DOM.
	return fileTabCodeBlockRE.ReplaceAllStringFunc(rendered, func(block string) string {
		block = strings.ReplaceAll(block, "\r\n", "&#10;")
		return strings.ReplaceAll(block, "\n", "&#10;")
	})
}

func (l *markdownLayout) Parse(content string) error {
	l.Content = content
	return nil
}

func (l *markdownLayout) Render() (string, error) {
	variant := safeToken(l.Meta["variant"])
	tag := "div"
	classes := []string{"mpress-" + l.Kind}
	attrs := ""
	if variant != "" {
		classes = append(classes, "mpress-"+l.Kind+"-"+variant)
	}
	for _, class := range strings.Fields(l.Meta["class"]) {
		if token := safeToken(class); token != "" {
			classes = append(classes, token)
		}
	}
	if id := strings.TrimSpace(l.Meta["id"]); id != "" {
		attrs += ` id="` + html.EscapeString(id) + `"`
	}
	if label := strings.TrimSpace(l.Meta["aria-label"]); label != "" {
		attrs += ` aria-label="` + html.EscapeString(label) + `"`
	}

	switch l.Kind {
	case "section":
		tag = "section"
		classes = append(classes, sectionThemeClasses(variant)...)
	case "columns":
		classes = append(classes, columnsThemeClasses(variant)...)
	case "column":
		classes = append(classes, columnThemeClasses(variant)...)
		if l.Meta["as"] == "header" || variant == "heading" {
			tag = "header"
		}
	case "actions":
		classes = append(classes, "mp-home-actions")
	}

	body := renderMarkdownFragment(l.Content)
	if l.Kind == "actions" {
		body = renderActionsFragment(l.Content)
	}
	if l.Kind == "section" {
		return fmt.Sprintf(`<%s class="%s"%s><div class="mpress-section-inner mp-home-shell">%s</div></%s>`, tag, strings.Join(classes, " "), attrs, body, tag), nil
	}
	return fmt.Sprintf(`<%s class="%s"%s>%s</%s>`, tag, strings.Join(classes, " "), attrs, body, tag), nil
}

func renderActionsFragment(source string) string {
	body := strings.TrimSpace(renderMarkdownFragment(source))
	// Goldmark wraps adjacent block-rendered buttons in paragraphs. Actions are
	// a control group, so those paragraph wrappers are invalid layout children.
	body = strings.ReplaceAll(body, "<p>", "")
	body = strings.ReplaceAll(body, "</p>", "")
	return strings.TrimSpace(body)
}

func sectionThemeClasses(variant string) []string {
	switch variant {
	case "hero":
		return []string{"mp-home-hero"}
	case "showcase":
		return []string{"mp-home-section", "mp-home-showcase-section"}
	case "workflow":
		return []string{"mp-home-section", "mp-home-workflow-section"}
	case "migration":
		return []string{"mp-home-section", "mp-home-migration-section"}
	case "resources":
		return []string{"mp-home-section", "mp-home-resources-section"}
	case "story":
		return []string{"mp-home-section", "mp-home-story-section"}
	case "final":
		return []string{"mp-home-final"}
	default:
		return []string{"mp-home-section"}
	}
}

func columnsThemeClasses(variant string) []string {
	switch variant {
	case "hero":
		return []string{"mp-home-hero-grid"}
	case "heading":
		return []string{"mp-home-section-heading"}
	case "workflow":
		return []string{"mp-home-two-column"}
	case "migration":
		return []string{"mp-home-migration"}
	case "story":
		return []string{"mp-home-story"}
	default:
		return nil
	}
}

func columnThemeClasses(variant string) []string {
	switch variant {
	case "hero-copy":
		return []string{"mp-home-hero-copy"}
	case "heading":
		return []string{"mp-home-column-heading"}
	case "site-screenshot":
		return []string{"mp-home-site-screenshot"}
	case "timeline":
		return []string{"mp-home-timeline-column"}
	case "capabilities":
		return []string{"mp-home-capabilities-column"}
	case "story-copy":
		return []string{"mp-home-story-copy"}
	case "story-visual":
		return []string{"mp-home-story-visual"}
	default:
		return nil
	}
}

func (d *DocsPreview) Parse(content string) error {
	d.Content = content
	return nil
}

func (d *DocsPreview) Render() (string, error) {
	variant := safeToken(d.Meta["variant"])
	brand := valueOr(d.Meta["brand"], "Product docs")
	search := valueOr(d.Meta["search"], "Search documentation")
	nav := renderPreviewNav(d.Meta["nav"])
	article := renderMarkdownFragment(d.Content)
	if breadcrumb := strings.TrimSpace(d.Meta["breadcrumb"]); breadcrumb != "" {
		article = `<div class="mp-home-breadcrumb">` + html.EscapeString(breadcrumb) + `</div>` + article
	}
	if variant == "showcase" {
		version := valueOr(d.Meta["version"], "v1.0")
		language := valueOr(d.Meta["language"], "EN")
		toc := renderPreviewTOC(d.Meta["toc"])
		return fmt.Sprintf(`<div class="mpress-doc-preview mpress-doc-preview-showcase mp-home-showcase"><div class="mp-home-showcase-bar"><span class="mp-home-docmark">M</span><strong>%s</strong><label>%s <kbd>⌘ K</kbd></label><button type="button">%s</button><button type="button">%s</button></div><div class="mp-home-showcase-body"><nav>%s</nav><article>%s</article><aside class="mp-home-toc"><strong>On this page</strong>%s</aside></div></div>`, html.EscapeString(brand), html.EscapeString(search), html.EscapeString(version), html.EscapeString(language), nav, article, toc), nil
	}
	mode := valueOr(d.Meta["mode"], "System")
	command := valueOr(d.Meta["command"], "mpress build --strict")
	status := valueOr(d.Meta["status"], "Built 12 pages · 0 errors")
	output := valueOr(d.Meta["output"], "site/")
	return fmt.Sprintf(`<div class="mpress-doc-preview mpress-doc-preview-compact mp-home-product"><div class="mp-home-docbar"><span class="mp-home-docmark">M</span><strong>%s</strong><span class="mp-home-docsearch">%s</span><span class="mp-home-docmode">%s</span></div><div class="mp-home-docbody"><nav>%s</nav><article>%s</article></div><div class="mp-home-terminal"><span class="prompt">$</span> %s <span class="result">%s</span><strong>%s</strong></div></div>`, html.EscapeString(brand), html.EscapeString(search), html.EscapeString(mode), nav, article, html.EscapeString(command), html.EscapeString(status), html.EscapeString(output)), nil
}

func renderPreviewNav(source string) string {
	var out strings.Builder
	for _, group := range strings.Split(source, ";") {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		parts := strings.SplitN(group, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fmt.Fprintf(&out, "<small>%s</small>", html.EscapeString(strings.TrimSpace(parts[0])))
		for _, item := range strings.Split(parts[1], ",") {
			item = strings.TrimSpace(item)
			active := strings.HasSuffix(item, "*")
			item = strings.TrimSuffix(item, "*")
			class := ""
			if active {
				class = ` class="active"`
			}
			fmt.Fprintf(&out, "<a%s>%s</a>", class, html.EscapeString(item))
		}
	}
	return out.String()
}

func renderPreviewTOC(source string) string {
	var out strings.Builder
	for _, item := range strings.Split(source, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		active := strings.HasSuffix(item, "*")
		item = strings.TrimSuffix(item, "*")
		class := ""
		if active {
			class = ` class="active"`
		}
		fmt.Fprintf(&out, "<a%s>%s</a>", class, html.EscapeString(item))
	}
	return out.String()
}

func (p *PreviewTabs) Parse(content string) error {
	p.Content = content
	return nil
}

func (p *PreviewTabs) Render() (string, error) {
	tabs := &Tabs{Meta: p.Meta}
	if err := tabs.Parse(p.Content); err != nil {
		return "", fmt.Errorf("preview tabs: %w", err)
	}
	rendered, err := tabs.Render()
	if err != nil {
		return "", err
	}
	rendered = strings.Replace(rendered, `class="mpress-tabs"`, `class="mpress-preview-tabs"`, 1)
	rendered = strings.Replace(rendered, `<div role="tablist">`, `<div class="mp-home-demo-tabs" role="tablist">`, 1)
	return rendered, nil
}

func (f *FileTabs) Parse(content string) error {
	f.Content = content
	return nil
}

func (f *FileTabs) Render() (string, error) {
	tabs := &Tabs{Meta: f.Meta}
	if err := tabs.Parse(f.Content); err != nil {
		return "", fmt.Errorf("file tabs: %w", err)
	}
	// File panels are complete fragments. Rendering them here prevents source
	// such as headings or frontmatter from becoming live page content while the
	// surrounding custom element is processed.
	rendered := tabs.renderWithPanelContent(renderFileTabFragment)
	rendered = strings.Replace(rendered, `class="mpress-tabs"`, `class="mpress-tabs mpress-file-tabs"`, 1)
	return rendered, nil
}

func (c *CompactCallout) Parse(content string) error {
	c.Content = content
	return nil
}

func (c *CompactCallout) Render() (string, error) {
	title := valueOr(c.Meta["title"], "Note")
	return fmt.Sprintf(`<aside class="mpress-compact-callout"><strong>%s</strong><div>%s</div></aside>`, html.EscapeString(title), renderMarkdownFragment(c.Content)), nil
}

var timelineItemRE = regexp.MustCompile(`(?m)^\s*(\d+)\.\s+\*\*([^*]+)\*\*\s+(.+?)\s*$`)

func (t *Timeline) Parse(content string) error {
	t.Content = content
	return nil
}

func (t *Timeline) Render() (string, error) {
	matches := timelineItemRE.FindAllStringSubmatch(t.Content, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("timeline requires numbered items with bold titles")
	}
	var out strings.Builder
	out.WriteString(`<ol class="mpress-timeline mp-home-timeline">`)
	for _, match := range matches {
		fmt.Fprintf(&out, `<li class="mpress-timeline-entry mpress-step"><span class="mpress-timeline-marker mpress-step-number">%s</span><div class="mpress-timeline-body mpress-step-body"><h3 class="mpress-step-title">%s</h3><div class="mpress-step-content"><p>%s</p></div></div></li>`, html.EscapeString(match[1]), html.EscapeString(strings.TrimSpace(match[2])), renderInlineMarkdown(match[3]))
	}
	out.WriteString(`</ol>`)
	return out.String(), nil
}

var capabilityItemRE = regexp.MustCompile(`(?m)^\s*-\s+\*\*([^*]+)\*\*\s+(.+?)\s*$`)

func (c *Capabilities) Parse(content string) error {
	c.Content = content
	return nil
}

func (c *Capabilities) Render() (string, error) {
	matches := capabilityItemRE.FindAllStringSubmatch(c.Content, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("capabilities require bullet items with bold names")
	}
	var out strings.Builder
	out.WriteString(`<dl class="mp-home-capabilities">`)
	for _, match := range matches {
		fmt.Fprintf(&out, `<div><dt>%s</dt><dd>%s</dd></div>`, html.EscapeString(strings.TrimSpace(match[1])), renderInlineMarkdown(match[2]))
	}
	out.WriteString(`</dl>`)
	return out.String(), nil
}

func (r *Resources) Parse(content string) error {
	r.Content = content
	return nil
}

func (r *Resources) Render() (string, error) {
	var out strings.Builder
	out.WriteString(`<div class="mp-home-resources">`)
	count := 0
	for _, block := range cardSeparatorRegex.Split(strings.TrimSpace(r.Content), -1) {
		lines := nonEmptyLines(block)
		if len(lines) < 3 {
			continue
		}
		link := cardLinkRegex.FindStringSubmatch(lines[0])
		if link == nil {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(lines[1], "# "))
		description := strings.TrimSpace(strings.Join(lines[2:], " "))
		fmt.Fprintf(&out, `<a href="%s"><span>%s</span><strong>%s</strong><em>%s</em></a>`, html.EscapeString(link[2]), html.EscapeString(link[1]), html.EscapeString(title), html.EscapeString(description))
		count++
	}
	out.WriteString(`</div>`)
	if count == 0 {
		return "", fmt.Errorf("resources require blocks with a link, heading, and description")
	}
	return out.String(), nil
}

func nonEmptyLines(source string) []string {
	var lines []string
	for _, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}

func renderInlineMarkdown(source string) string {
	rendered := strings.TrimSpace(renderMarkdownFragment(source))
	rendered = strings.TrimPrefix(rendered, "<p>")
	rendered = strings.TrimSuffix(rendered, "</p>")
	return rendered
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
