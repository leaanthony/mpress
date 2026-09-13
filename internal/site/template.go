package site

import (
	"bytes"
	"encoding/json"
	stdhtml "html"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/icons"
	"github.com/leaanthony/mpress/internal/navigation"
	"github.com/leaanthony/mpress/internal/quickedit"
)

// templateData is deliberately a typed rendering model. The default renderer
// does not interpret a template at build time: it writes one of the fixed page
// layouts directly. This keeps the default theme fast on constrained CI and
// Kubernetes workers while leaving Markdown content entirely data driven.
type templateData struct {
	Config config.Config
	Page   *content.Page
	Nav    []navigation.Item
	// CompiledNav is the per-language precompiled sidebar. When set, renderDocs
	// replays it instead of re-deriving the nav HTML for every page.
	CompiledNav   *compiledNav
	Root          string
	LangLinks     []languageLink
	Prev          *navigation.Item
	Next          *navigation.Item
	VersionLinks  []versionLink
	SearchURL     string
	CSSVersion    string
	JSVersion     string
	CanonicalURL  string
	Alternates    []alternateLink
	CurrentRoutes map[string]*content.Page
	Blog          *blogArchiveData
	BlogArticle   *blogArticleData
	QuickEdit     *quickedit.Document
	NotFound      bool
}

type languageLink struct {
	Code, Label, URL, FallbackLabel string
	Available                       bool
}

type alternateLink struct{ Language, URL string }

type versionLink struct {
	Label, URL string
	Current    bool
}

func renderPage(data templateData) ([]byte, error) {
	var b bytes.Buffer
	if err := renderPageInto(data, &b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// renderPageInto renders into a caller-supplied buffer so hot paths can reuse
// one buffer per worker instead of allocating a page-sized array per page.
func renderPageInto(data templateData, b *bytes.Buffer) error {
	// The page body is normally the largest dynamic fragment. Reserve enough
	// room for the shell too, which avoids the repeated buffer growth that was
	// otherwise obscured by html/template's own allocations.
	b.Grow(len(data.Page.HTML) + 12*1024)
	r := pageRenderer{data: &data, out: b}
	r.render()
	return nil
}

type pageRenderer struct {
	data *templateData
	out  *bytes.Buffer
}

func (r *pageRenderer) raw(value string) { r.out.WriteString(value) }

func (r *pageRenderer) text(value string) { r.raw(stdhtml.EscapeString(value)) }

func (r *pageRenderer) attr(value string) { r.text(value) }

func (r *pageRenderer) url(value string) { r.attr(safeURL(value)) }

func (r *pageRenderer) icon(name string, size int) {
	r.raw(icons.Lucide(name, size))
}

func (r *pageRenderer) starlightIcon(name string, size int) {
	r.raw(icons.Starlight(name, size))
}

// safeURL preserves the relative URLs M-Press generates while rejecting the
// executable schemes that html/template rejects in URL attributes.
func safeURL(value string) string {
	compact := strings.Map(func(character rune) rune {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return -1
		}
		return character
	}, value)
	lower := strings.ToLower(compact)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "vbscript:") {
		return "#ZgotmplZ"
	}
	return value
}

func (r *pageRenderer) render() {
	r.renderHead()
	r.renderHeader()
	r.renderAccessibilityPanel()
	r.renderContributionDialog()
	r.renderImageLightbox()
	r.renderQuickEditBar()
	switch {
	case r.data.NotFound:
		r.renderNotFound()
	case r.data.Blog != nil:
		r.renderBlogIndex()
	case r.data.BlogArticle != nil:
		r.renderBlogArticle()
	case r.data.Page.Layout == "landing":
		r.renderLanding()
	default:
		r.renderDocs()
	}
	r.raw("<footer><a class=\"mpress-footer-credit\" href=\"https://m-press.me\">")
	r.renderFooterCredit()
	r.raw("</a><span>")
	r.uiText("Markdown in. Beautiful docs out.")
	r.raw("</span></footer><script src=\"")
	r.url(r.data.Root + "assets/mpress.js")
	if r.data.JSVersion != "" {
		r.raw(`?v=`)
		r.attr(r.data.JSVersion)
	}
	r.raw(`" defer></script></body></html>`)
}

func (r *pageRenderer) renderHead() {
	d := r.data
	pageTitle := d.Page.Title + " · " + d.Config.Site.Title
	description := d.Page.Description
	if description == "" {
		description = d.Config.Site.Description
	}
	r.raw(`<!doctype html><html lang="`)
	r.attr(d.Page.Language)
	r.raw(`"`)
	if isRTLLanguage(d.Page.Language) {
		r.raw(` dir="rtl"`)
	}
	r.raw(` data-theme="`)
	r.attr(d.Config.Theme.ColorScheme)
	r.raw(`"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>`)
	r.text(pageTitle)
	r.raw(`</title><meta name="description" content="`)
	r.attr(description)
	r.raw(`"><meta property="og:title" content="`)
	r.attr(pageTitle)
	r.raw(`"><meta property="og:description" content="`)
	r.attr(description)
	r.raw(`"><meta property="og:site_name" content="`)
	r.attr(d.Config.Site.Title)
	r.raw(`"><meta property="og:type" content="`)
	if d.BlogArticle != nil {
		r.raw("article")
	} else {
		r.raw(`website`)
	}
	r.raw(`"><meta name="twitter:card" content="`)
	if d.Config.Site.SocialImage != "" && d.Config.Site.BaseURL != "" {
		r.raw(`summary_large_image`)
	} else {
		r.raw(`summary`)
	}
	r.raw(`"><meta name="twitter:title" content="`)
	r.attr(pageTitle)
	r.raw(`"><meta name="twitter:description" content="`)
	r.attr(description)
	r.raw(`">`)
	if d.NotFound {
		r.raw(`<meta name="robots" content="noindex,follow">`)
	}
	if imageURL := r.socialImageURL(); imageURL != "" {
		r.raw(`<meta property="og:image" content="`)
		r.url(imageURL)
		r.raw(`"><meta name="twitter:image" content="`)
		r.url(imageURL)
		r.raw(`">`)
	}
	if d.Config.Contribution.Enabled {
		r.raw(`<meta name="mpress:repository" content="`)
		r.attr(d.Config.Contribution.Repository)
		r.raw(`"><meta name="mpress:branch" content="`)
		r.attr(d.Config.Contribution.Branch)
		r.raw(`"><meta name="mpress:source" content="`)
		r.attr(r.contributionSourcePath())
		r.raw(`"><meta name="mpress:route" content="`)
		r.attr(r.contributionRoute())
		r.raw(`">`)
		if d.Config.Contribution.Guide != "" {
			r.raw(`<meta name="mpress:guide" content="`)
			r.attr(d.Config.Contribution.Guide)
			r.raw(`">`)
		}
	}
	if d.CanonicalURL != "" {
		r.raw(`<meta property="og:url" content="`)
		r.url(d.CanonicalURL)
		r.raw(`">`)
		r.raw(`<link rel="canonical" href="`)
		r.url(d.CanonicalURL)
		r.raw(`">`)
		for _, alternate := range d.Alternates {
			r.raw(`<link rel="alternate" hreflang="`)
			r.attr(alternate.Language)
			r.raw(`" href="`)
			r.url(alternate.URL)
			r.raw(`">`)
		}
	}
	r.renderUIMessages()
	r.raw(`<script>document.documentElement.classList.add('js');`)
	r.raw(themeBootstrapJS)
	if d.Config.Accessibility.Enabled {
		r.raw(accessibilityBootstrapJS)
	}
	r.raw(`</script><link rel="stylesheet" href="`)
	r.url(d.Root + "assets/mpress.css")
	if d.CSSVersion != "" {
		r.raw(`?v=`)
		r.attr(d.CSSVersion)
	}
	r.raw(`">`)
	if d.Config.Site.Favicon != "" {
		r.raw(`<link rel="icon" href="`)
		r.url(r.resourceURL(d.Config.Site.Favicon))
		r.raw(`">`)
	}
	r.raw(`</head><body class="`)
	switch {
	case d.NotFound:
		r.raw("not-found-page")
	case d.Blog != nil:
		r.raw("blog-index-page")
	case d.BlogArticle != nil:
		r.raw("blog-article-page")
	case d.Page.Layout == "landing":
		r.raw("landing-page")
	default:
		r.raw("docs-page")
		if d.Page.Layout == "wide" {
			r.raw(" docs-page-wide")
		}
	}
	r.raw(`" data-search="`)
	r.url(d.SearchURL)
	r.raw(`" data-search-placeholder="`)
	r.uiText(d.Config.Search.Placeholder)
	r.raw(`" data-search-max-results="`)
	r.attr(strconv.Itoa(d.Config.Search.MaxResults))
	r.raw(`" data-search-recent="`)
	r.attr(strconv.FormatBool(d.Config.Search.RememberRecent))
	r.raw(`" data-shortcut-search="`)
	r.attr(d.Config.Search.Shortcut)
	r.raw(`" data-shortcut-accessibility="`)
	r.attr(d.Config.Accessibility.Shortcut)
	r.raw("\"><a class=\"skip\" href=\"#content\">")
	r.uiText("Skip to content")
	r.raw("</a>")
}

func isRTLLanguage(language string) bool {
	base := strings.ToLower(strings.TrimSpace(language))
	if index := strings.IndexAny(base, "-_"); index >= 0 {
		base = base[:index]
	}
	switch base {
	case "ar", "dv", "fa", "he", "ku", "ps", "sd", "ug", "ur", "yi":
		return true
	default:
		return false
	}
}

func (r *pageRenderer) socialImageURL() string {
	value := strings.TrimSpace(r.data.Config.Site.SocialImage)
	if value == "" || strings.TrimSpace(r.data.Config.Site.BaseURL) == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return strings.TrimRight(r.data.Config.Site.BaseURL, "/") + "/" + strings.TrimLeft(value, "/")
}

func (r *pageRenderer) renderHeader() {
	d := r.data
	r.raw("<header><button id=\"menu\" type=\"button\" aria-label=\"")
	r.uiText("Toggle navigation")
	r.raw("\" aria-expanded=\"false\" aria-controls=\"mpress-sidebar\">")
	r.icon("menu", 20)
	r.raw(`</button><a class="brand" href="`)
	r.url(r.pageURL("/"))
	r.raw(`">`)
	site := d.Config.Site
	switch {
	case site.LogoLight != "" && site.LogoDark != "":
		r.raw(`<span class="theme-logo" style="--mpress-logo-width:`)
		r.attr(site.LogoWidth)
		r.raw(`"><img class="theme-logo-light" src="`)
		r.url(r.resourceURL(site.LogoLight))
		r.raw(`" alt="`)
		r.attr(site.Title)
		r.raw(`"><img class="theme-logo-dark" src="`)
		r.url(r.resourceURL(site.LogoDark))
		r.raw(`" alt="`)
		r.attr(site.Title)
		r.raw(`"></span>`)
	case site.LogoLight != "":
		r.raw(`<img class="brand-logo" style="--mpress-logo-width:`)
		r.attr(site.LogoWidth)
		r.raw(`" src="`)
		r.url(r.resourceURL(site.LogoLight))
		r.raw(`" alt="`)
		r.attr(site.Title)
		r.raw(`">`)
	case site.LogoDark != "":
		r.raw(`<img class="brand-logo" style="--mpress-logo-width:`)
		r.attr(site.LogoWidth)
		r.raw(`" src="`)
		r.url(r.resourceURL(site.LogoDark))
		r.raw(`" alt="`)
		r.attr(site.Title)
		r.raw(`">`)
	default:
		r.raw(`<span class="brand-mark" aria-hidden="true">M</span><span>`)
		r.text(site.Title)
		r.raw(`</span>`)
	}
	r.raw(`</a>`)
	if len(site.HeaderLinks) > 0 {
		r.raw("<nav class=\"primary-links\" aria-label=\"")
		r.uiText("Primary navigation")
		r.raw("\">")
		for _, link := range site.HeaderLinks {
			r.raw(`<a`)
			if link.Type == "button" {
				variant := strings.TrimSpace(link.Variant)
				if variant == "" {
					variant = "primary"
				}
				r.raw(` class="header-link-button header-link-button-`)
				r.attr(variant)
				r.raw(`"`)
				if variant == "custom" {
					r.raw(` style="--header-button-color:`)
					r.attr(link.Color)
					r.raw(`"`)
				}
			}
			r.raw(` href="`)
			r.url(r.pageURL(link.URL))
			r.raw(`">`)
			r.uiText(link.Label)
			r.raw(`</a>`)
		}
		r.raw(`</nav>`)
	}
	if d.Config.Search.Enabled {
		r.raw("<div class=\"search\"><span>")
		r.uiText("Search")
		r.raw("</span>")
		r.icon("search", 18)
		r.raw("<button id=\"search\" type=\"button\" role=\"combobox\" aria-label=\"")
		r.uiText("Search")
		r.raw("\" aria-autocomplete=\"list\" aria-haspopup=\"dialog\" aria-expanded=\"false\" aria-controls=\"mpress-search-dialog\">")
		r.uiText(d.Config.Search.Placeholder)
		r.raw(`</button><kbd class="search-shortcut" aria-hidden="true">Ctrl K</kbd></div>`)
	} else {
		r.raw(`<span class="header-spacer" aria-hidden="true"></span>`)
	}
	r.raw("<nav class=\"header-links\" aria-label=\"")
	r.uiText("Documentation controls")
	r.raw("\">")
	r.renderSocialLinks()
	r.raw("<span class=\"header-utility-cluster\" role=\"group\" aria-label=\"")
	r.uiText("Site preferences")
	r.raw("\">")
	if d.Config.Contribution.Enabled {
		r.raw(`<div class="header-group contribution-header">`)
		r.renderContributionTrigger("Edit this documentation", "header-contribute")
		r.raw(`</div>`)
	}
	r.renderAccessibilityButton()
	r.renderLanguageMenu()
	r.renderVersionMenu()
	r.raw("<button id=\"theme\" class=\"header-group theme-toggle\" type=\"button\" data-theme-mode=\"system\" aria-label=\"")
	r.uiText("Theme: System. Switch to Dark")
	r.raw("\" title=\"")
	r.uiText("Theme: System")
	r.raw("\">")
	r.icon("monitor", 17)
	r.icon("moon", 17)
	r.icon("sun", 17)
	r.raw(`</button></span></nav></header>`)
}

func (r *pageRenderer) renderContributionTrigger(label, className string) {
	r.raw(`<button class="mpress-contribute-trigger `)
	r.attr(className)
	r.raw("\" type=\"button\" data-mpress-contribute aria-haspopup=\"dialog\" aria-label=\"")
	r.uiText("Edit this documentation")
	r.raw("\" title=\"")
	r.uiText("Edit this documentation")
	r.raw("\">")
	r.icon("pencil", 19)
	r.raw(`<span>`)
	r.uiText(label)
	r.raw(`</span></button>`)
}

func (r *pageRenderer) contributionSourcePath() string {
	parts := []string{r.data.Config.Build.ContentDir}
	if r.data.Page.Language != r.data.Config.Site.DefaultLanguage {
		parts = append(parts, r.data.Page.Language)
	}
	parts = append(parts, r.data.Page.SourcePath)
	return filepath.ToSlash(filepath.Join(parts...))
}

func (r *pageRenderer) contributionRoute() string {
	prefix := ""
	if r.data.Page.Language != r.data.Config.Site.DefaultLanguage || !r.data.Config.Site.DefaultAtRoot {
		prefix = "/" + r.data.Page.Language
	}
	route := prefix + "/" + strings.Trim(r.data.Page.URLPath, "/")
	if r.data.Page.URLPath != "" {
		route += "/"
	}
	return route
}

func (r *pageRenderer) renderContributionDialog() {
	if !r.data.Config.Contribution.Enabled {
		return
	}
	r.raw(`<dialog id="mpress-contribute-dialog" class="mpress-contribute-dialog" data-installer-shell="`)
	r.url(r.data.Root + "mpress-contribute.sh")
	r.raw(`" data-installer-powershell="`)
	r.url(r.data.Root + "mpress-contribute.ps1")
	r.raw(`" aria-labelledby="mpress-contribute-title">`)
	if r.data.Config.Contribution.QuickEdit {
		r.renderQuickEditData()
	}
	r.raw(`<header><div class="mpress-contribute-heading"><span class="mpress-contribute-mark" aria-hidden="true">`)
	r.icon("pencil", 23)
	r.raw("</span><div><span class=\"mpress-contribute-eyebrow\">")
	r.uiText("Contribute")
	r.raw("</span><h2 id=\"mpress-contribute-title\">")
	r.uiText("Edit this documentation")
	r.raw("</h2></div></div><button type=\"button\" class=\"mpress-contribute-close\" data-contribute-close aria-label=\"")
	r.uiText("Close contribution instructions")
	r.raw("\">")
	r.icon("x", 18)
	if r.data.QuickEdit != nil {
		r.raw("</button></header><p class=\"mpress-contribute-intro\" data-contribute-intro>")
		r.uiText("Improve this page, translate the documentation, or open the complete project on your computer.")
		r.raw("</p><div class=\"mpress-contribute-choices\" data-contribute-choices><button type=\"button\" data-contribute-quick-edit>")
		r.icon("pencil", 18)
		r.raw("<span><strong>")
		r.uiText("Fix this page")
		r.raw("</strong><small>")
		r.uiText("Change text here while the development server is running.")
		r.raw("</small></span></button>")
	} else {
		r.raw("</button></header><p class=\"mpress-contribute-intro\" data-contribute-intro>")
		r.uiText("Translate the documentation or open the complete project on your computer.")
		r.raw("</p><div class=\"mpress-contribute-choices mpress-contribute-choices-public\" data-contribute-choices>")
	}
	r.raw(`<button type="button" data-contribute-translate>`)
	r.icon("languages", 18)
	r.raw("<span><strong>")
	r.uiText("Translate documentation")
	r.raw("</strong><small>")
	r.uiText("Choose a language and let M-Press prepare the best available workflow.")
	r.raw("</small></span></button><button type=\"button\" data-contribute-computer>")
	r.icon("terminal", 18)
	r.raw("<span><strong>")
	r.uiText("Edit this page locally")
	r.raw("</strong><small>")
	r.uiText("Open this exact page in a safe local contribution checkout.")
	r.raw("</small></span></button></div><div data-contribute-setup hidden><div class=\"mpress-contribute-platform\"><strong data-contribute-platform-label>")
	r.uiText("Command for macOS and Linux")
	r.raw("</strong><span>")
	r.uiText("Detected automatically")
	r.raw("</span></div><div class=\"mpress-contribute-command\" tabindex=\"0\" aria-label=\"")
	r.uiText("Contribution command")
	r.raw("\"><code data-contribute-command></code></div><button type=\"button\" class=\"mpress-contribute-copy\" data-contribute-copy>")
	r.icon("copy", 16)
	r.raw("<span>")
	r.uiText("Copy command")
	r.raw("</span></button><details class=\"mpress-contribute-explainer\"><summary>")
	r.uiText("What this command does")
	r.raw("</summary><ol><li>")
	r.uiText("Uses M-Press if it is already installed.")
	r.raw("</li><li>")
	r.uiText("Otherwise downloads the official M-Press release for this computer.")
	r.raw("</li><li>")
	r.uiText("Checks out the configured documentation repository on a private local branch.")
	r.raw("</li><li>")
	r.uiText("Opens this exact page in the development server.")
	r.raw("</li></ol><p><a href=\"")
	r.url(r.data.Root + "mpress-contribute.sh")
	r.raw("\" target=\"_blank\" rel=\"noopener\">")
	r.uiText("View the macOS and Linux script")
	r.raw("</a> · <a href=\"")
	r.url(r.data.Root + "mpress-contribute.ps1")
	r.raw("\" target=\"_blank\" rel=\"noopener\">")
	r.uiText("View the Windows script")
	r.raw("</a></p><div class=\"mpress-contribute-manual\"><strong>")
	r.uiText("Already installed?")
	r.raw("</strong><code data-contribute-manual></code></div></details></div><p class=\"mpress-contribute-status\" data-contribute-status aria-live=\"polite\">")
	r.uiText("Nothing is published until you choose to submit your work.")
	r.raw("</p></dialog>")
}

func (r *pageRenderer) renderImageLightbox() {
	if !strings.Contains(r.data.Page.HTML, "mpress-image-expand") {
		return
	}
	r.raw("<dialog id=\"mpress-image-lightbox\" class=\"mpress-image-lightbox\" aria-label=\"")
	r.uiText("Expanded image")
	r.raw("\"><button type=\"button\" class=\"mpress-image-lightbox-close\" data-image-lightbox-close aria-label=\"")
	r.uiText("Close expanded image")
	r.raw("\">")
	r.icon("x", 20)
	r.raw(`</button><div class="mpress-image-lightbox-content" data-image-lightbox-content></div></dialog>`)
}

func (r *pageRenderer) renderQuickEditData() {
	if r.data.QuickEdit == nil {
		return
	}
	payload := struct {
		Version  int                 `json:"version"`
		Source   string              `json:"source"`
		Route    string              `json:"route"`
		Revision string              `json:"revision"`
		Segments []quickedit.Segment `json:"segments"`
	}{
		Version: r.data.QuickEdit.Version, Source: r.contributionSourcePath(), Route: r.contributionRoute(),
		Revision: r.data.QuickEdit.Revision, Segments: r.data.QuickEdit.Segments,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	r.raw(`<script type="application/json" id="mpress-quick-edit-data">`)
	r.raw(string(data))
	r.raw(`</script>`)
}

func (r *pageRenderer) renderQuickEditBar() {
	if !r.data.Config.Contribution.Enabled || !r.data.Config.Contribution.QuickEdit || r.data.QuickEdit == nil || len(r.data.QuickEdit.Segments) == 0 {
		return
	}
	r.raw("<section class=\"mpress-quick-edit-bar\" data-quick-edit-bar hidden aria-label=\"")
	r.uiText("Quick editing controls")
	r.raw("\"><div class=\"mpress-quick-edit-state\"><span class=\"mpress-quick-edit-mark\" aria-hidden=\"true\">M</span><span><strong>")
	r.uiText("Quick editing")
	r.raw("</strong><small data-quick-edit-status>")
	r.uiText("Draft saved in this browser")
	r.raw("</small></span></div><div class=\"mpress-quick-edit-actions\"><button type=\"button\" data-quick-edit-add>")
	r.icon("plus", 15)
	r.raw("<span>")
	r.uiText("Add paragraph")
	r.raw("</span></button><button type=\"button\" data-quick-edit-discard>")
	r.uiText("Discard")
	r.raw("</button><button type=\"button\" class=\"primary\" data-quick-edit-contribute>")
	r.uiText("Continue on computer")
	r.raw("</button></div></section><div class=\"mpress-quick-edit-format\" data-quick-edit-format role=\"toolbar\" aria-label=\"")
	r.uiText("Text formatting")
	r.raw("\" hidden><button type=\"button\" data-quick-edit-format-command=\"bold\" aria-label=\"")
	r.uiText("Bold")
	r.raw("\" title=\"")
	r.uiText("Bold (Ctrl or Command B)")
	r.raw("\">")
	r.icon("bold", 16)
	r.raw("</button><button type=\"button\" data-quick-edit-format-command=\"italic\" aria-label=\"")
	r.uiText("Italic")
	r.raw("\" title=\"")
	r.uiText("Italic (Ctrl or Command I)")
	r.raw("\">")
	r.icon("italic", 16)
	r.raw("</button><button type=\"button\" data-quick-edit-format-command=\"code\" aria-label=\"")
	r.uiText("Inline code")
	r.raw("\" title=\"")
	r.uiText("Inline code")
	r.raw("\">")
	r.icon("code", 16)
	r.raw("</button><button type=\"button\" data-quick-edit-format-command=\"link\" aria-label=\"")
	r.uiText("Add link")
	r.raw("\" title=\"")
	r.uiText("Add link (Ctrl or Command K)")
	r.raw("\">")
	r.icon("link", 16)
	r.raw("</button><form class=\"mpress-quick-edit-link-form\" data-quick-edit-link-form hidden><label for=\"mpress-quick-edit-link\">")
	r.uiText("Link address")
	r.raw("</label><input id=\"mpress-quick-edit-link\" data-quick-edit-link-input type=\"text\" inputmode=\"url\" autocomplete=\"url\" placeholder=\"https://example.com\"><button type=\"submit\">")
	r.uiText("Apply")
	r.raw("</button><button type=\"button\" data-quick-edit-link-cancel aria-label=\"")
	r.uiText("Cancel link")
	r.raw("\">")
	r.icon("x", 15)
	r.raw(`</button></form></div>`)
}

func (r *pageRenderer) renderAccessibilityButton() {
	if !r.data.Config.Accessibility.Enabled {
		return
	}
	r.raw("<div class=\"header-group utility-select accessibility-select\"><button id=\"accessibility\" class=\"utility-menu-trigger\" type=\"button\" popovertarget=\"mpress-accessibility-panel\" aria-label=\"")
	r.uiText("Accessibility settings")
	r.raw("\" aria-haspopup=\"dialog\" aria-expanded=\"false\" title=\"")
	r.uiText("Accessibility settings")
	r.raw("\">")
	r.raw(icons.Accessibility(19))
	r.raw(`</button></div>`)
}

func (r *pageRenderer) renderAccessibilityPanel() {
	if !r.data.Config.Accessibility.Enabled {
		return
	}
	r.raw("<section id=\"mpress-accessibility-panel\" class=\"utility-menu-panel mpress-accessibility-panel\" popover role=\"dialog\" aria-labelledby=\"mpress-accessibility-title\" data-utility-panel><header class=\"mpress-accessibility-header\"><div><h2 id=\"mpress-accessibility-title\">")
	r.uiText("Accessibility")
	r.raw("</h2><p>")
	r.uiText("Adjust the site to make it easier to read and navigate.")
	r.raw("</p></div><button class=\"mpress-accessibility-close\" type=\"button\" data-a11y-close aria-label=\"")
	r.uiText("Close accessibility settings")
	r.raw("\">")
	r.icon("x", 18)
	r.raw("</button></header><div class=\"mpress-accessibility-tabs\" role=\"tablist\" aria-label=\"")
	r.uiText("Accessibility categories")
	r.raw("\"><button id=\"mpress-a11y-tab-reading\" type=\"button\" role=\"tab\" aria-selected=\"true\" aria-controls=\"mpress-a11y-panel-reading\" data-a11y-tab=\"reading\">")
	r.uiText("Reading")
	r.raw("</button><button id=\"mpress-a11y-tab-focus\" type=\"button\" role=\"tab\" aria-selected=\"false\" aria-controls=\"mpress-a11y-panel-focus\" data-a11y-tab=\"focus\" tabindex=\"-1\">")
	r.uiText("Focus")
	r.raw("</button><button id=\"mpress-a11y-tab-vision\" type=\"button\" role=\"tab\" aria-selected=\"false\" aria-controls=\"mpress-a11y-panel-vision\" data-a11y-tab=\"vision\" tabindex=\"-1\">")
	r.uiText("Vision")
	r.raw("</button></div><div class=\"mpress-accessibility-body\"><section id=\"mpress-a11y-panel-reading\" class=\"mpress-accessibility-section\" role=\"tabpanel\" aria-labelledby=\"mpress-a11y-tab-reading\" data-a11y-tabpanel=\"reading\"><h3>")
	r.uiText("Reading")
	r.raw("</h3><p>")
	r.uiText("Change how text is shown in the main content.")
	r.raw("</p><div class=\"mpress-a11y-choice\" role=\"radiogroup\" aria-label=\"")
	r.uiText("Text size")
	r.raw("\"><button type=\"button\" role=\"radio\" data-a11y-choice=\"text\" data-value=\"default\">")
	r.uiText("Default")
	r.raw("</button><button type=\"button\" role=\"radio\" data-a11y-choice=\"text\" data-value=\"large\">")
	r.uiText("Large")
	r.raw("</button><button type=\"button\" role=\"radio\" data-a11y-choice=\"text\" data-value=\"larger\">")
	r.uiText("Larger")
	r.raw("</button></div><div class=\"mpress-a11y-width\"><div class=\"mpress-a11y-width-heading\"><span>")
	r.uiText("Site layout")
	r.raw("</span></div><div class=\"mpress-a11y-width-mode\" role=\"radiogroup\" aria-label=\"")
	r.uiText("Site layout")
	r.raw("\"><button type=\"button\" role=\"radio\" data-a11y-choice=\"siteWidth\" data-value=\"full\">")
	r.uiText("Full width")
	r.raw("</button><button type=\"button\" role=\"radio\" data-a11y-choice=\"siteWidth\" data-value=\"fixed\">")
	r.uiText("Fixed width")
	r.raw("</button></div><small>")
	r.uiText("Choose whether navigation uses the full browser width.")
	r.raw("</small></div><div class=\"mpress-a11y-width\"><div class=\"mpress-a11y-width-heading\"><label for=\"mpress-a11y-width\">")
	r.uiText("Reading column")
	r.raw("</label><output for=\"mpress-a11y-width\" data-a11y-width-output>")
	r.uiText("Default")
	r.raw("</output></div><div class=\"mpress-a11y-width-mode\" role=\"radiogroup\" aria-label=\"")
	r.uiText("Reading column unit")
	r.raw("\"><button type=\"button\" role=\"radio\" aria-checked=\"true\" data-a11y-width-mode=\"percent\">")
	r.uiText("Relative")
	r.raw("</button><button type=\"button\" role=\"radio\" aria-checked=\"false\" data-a11y-width-mode=\"fixed\">")
	r.uiText("Fixed")
	r.raw("</button></div><input id=\"mpress-a11y-width\" type=\"range\" min=\"40\" max=\"100\" step=\"1\" value=\"70\" data-a11y-width aria-describedby=\"mpress-a11y-width-help\"><div class=\"mpress-a11y-width-footer\"><small id=\"mpress-a11y-width-help\">")
	r.uiText("Change the width of the article, not the site shell.")
	r.raw("</small><button type=\"button\" data-a11y-width-reset disabled>")
	r.uiText("Reset")
	r.raw("</button></div></div>")
	r.renderAccessibilityToggle("readable", "Readable font", "Use a simple, widely spaced font stack.")
	r.renderAccessibilityToggle("spacing", "Relaxed spacing", "Add more line, word, and letter space.")
	r.renderAccessibilityToggle("bionic", "Bionic reading", "Emphasise the start of longer words.")
	r.raw("</section><section id=\"mpress-a11y-panel-focus\" class=\"mpress-accessibility-section\" role=\"tabpanel\" aria-labelledby=\"mpress-a11y-tab-focus\" data-a11y-tabpanel=\"focus\" hidden><h3>")
	r.uiText("Focus and attention")
	r.raw("</h3><p>")
	r.uiText("Reduce distraction and keep your place while reading.")
	r.raw("</p>")
	r.renderAccessibilityToggle("focus", "Focus mode", "Dim navigation until you point to it.")
	r.renderAccessibilityToggle("guide", "Reading guide", "Follow the pointer with a horizontal guide.")
	r.renderAccessibilityToggle("motion", "Reduce motion", "Stop non-essential animation and smooth scrolling.")
	r.raw("</section><section id=\"mpress-a11y-panel-vision\" class=\"mpress-accessibility-section\" role=\"tabpanel\" aria-labelledby=\"mpress-a11y-tab-vision\" data-a11y-tabpanel=\"vision\" hidden><h3>")
	r.uiText("Vision and colour")
	r.raw("</h3><p>")
	r.uiText("Increase contrast or use a more distinct colour palette.")
	r.raw("</p><div class=\"mpress-a11y-field\"><span id=\"mpress-a11y-colour-label\">")
	r.uiText("Colour profile")
	r.raw("</span><div class=\"mpress-a11y-select\" data-a11y-select=\"colour\"><button class=\"mpress-a11y-select-trigger\" type=\"button\" aria-haspopup=\"listbox\" aria-expanded=\"false\" aria-labelledby=\"mpress-a11y-colour-label mpress-a11y-colour-value\"><span id=\"mpress-a11y-colour-value\" data-a11y-select-value>")
	r.uiText("Site colours")
	r.raw("</span>")
	r.icon("chevron-down", 15)
	r.raw("</button><div class=\"mpress-a11y-options\" role=\"listbox\" aria-labelledby=\"mpress-a11y-colour-label\" hidden><button class=\"mpress-a11y-option\" type=\"button\" role=\"option\" aria-selected=\"true\" data-value=\"default\">")
	r.uiText("Site colours")
	r.raw("</button><button class=\"mpress-a11y-option\" type=\"button\" role=\"option\" aria-selected=\"false\" data-value=\"red-green\" tabindex=\"-1\">")
	r.uiText("Red and green distinction")
	r.raw("</button><button class=\"mpress-a11y-option\" type=\"button\" role=\"option\" aria-selected=\"false\" data-value=\"blue-yellow\" tabindex=\"-1\">")
	r.uiText("Blue and yellow distinction")
	r.raw("</button><button class=\"mpress-a11y-option\" type=\"button\" role=\"option\" aria-selected=\"false\" data-value=\"low\" tabindex=\"-1\">")
	r.uiText("Low saturation")
	r.raw("</button></div></div></div>")
	r.renderAccessibilityToggle("contrast", "High contrast", "Increase text and border contrast.")
	r.renderAccessibilityToggle("links", "Underline links", "Show links with more than colour.")
	r.raw("</section></div><footer class=\"mpress-accessibility-footer\"><small>")
	r.uiText("Saved only in this browser.")
	r.raw("</small><button class=\"mpress-accessibility-reset\" type=\"button\" data-a11y-reset>")
	r.uiText("Reset settings")
	r.raw("</button></footer></section>")
}

func (r *pageRenderer) renderAccessibilityToggle(name, label, description string) {
	r.raw(`<label class="mpress-a11y-toggle"><span><strong>`)
	r.uiText(label)
	r.raw(`</strong><small>`)
	r.uiText(description)
	r.raw(`</small></span><input type="checkbox" data-a11y-toggle="`)
	r.attr(name)
	r.raw(`"></label>`)
}

func (r *pageRenderer) renderVersionMenu() {
	r.raw(`<!--mpress-version-menu:start-->`)
	defer r.raw(`<!--mpress-version-menu:end-->`)
	if len(r.data.VersionLinks) == 0 {
		return
	}
	r.raw("<div class=\"header-group utility-select utility-menu version-select\"><button class=\"utility-menu-trigger\" type=\"button\" popovertarget=\"mpress-version-menu\" aria-label=\"")
	r.uiText("Select version")
	r.raw("\" aria-haspopup=\"menu\" aria-expanded=\"false\">")
	r.icon("git-branch", 19)
	r.raw(`<span class="utility-menu-current">`)
	for _, version := range r.data.VersionLinks {
		if version.Current {
			r.renderVersionLabel(version.Label)
		}
	}
	r.raw(`</span>`)
	r.icon("chevron-down", 13)
	r.raw(`</button><menu id="mpress-version-menu" class="utility-menu-panel utility-version-menu" popover data-utility-menu>`)
	for _, version := range r.data.VersionLinks {
		r.raw(`<li><a href="`)
		r.url(version.URL)
		r.raw(`" role="menuitem"`)
		if version.Current {
			r.raw(` aria-current="true"`)
		}
		r.raw(`><span class="utility-version-label">`)
		r.renderVersionLabel(version.Label)
		r.raw(`</span>`)
		if version.Current {
			r.raw("<small>")
			r.uiText("Current documentation")
			r.raw("</small>")
		}
		r.raw(`</a></li>`)
	}
	r.raw(`</menu></div>`)
}

func (r *pageRenderer) renderSocialLinks() {
	social := r.data.Config.Social
	if social.GitHub == "" && social.Discord == "" && social.Reddit == "" && social.X == "" && social.RSS == "" && social.Sponsor == "" {
		return
	}
	r.raw(`<span class="header-group social-links">`)
	r.socialLink(social.GitHub, "GitHub", "github", true)
	r.socialLink(social.Discord, "Discord", "discord", true)
	r.socialLink(social.Reddit, "Reddit", "reddit", true)
	r.socialLink(social.X, "X", "x", true)
	r.socialLink(social.RSS, "RSS", "rss", true)
	r.socialLink(social.Sponsor, "Sponsor", "heart", false)
	r.raw(`</span>`)
}

func (r *pageRenderer) socialLink(href, label, iconName string, starlight bool) {
	if href == "" {
		return
	}
	r.raw(`<a class="social-link" href="`)
	r.url(href)
	r.raw(`" aria-label="`)
	r.uiText(label)
	r.raw(`" title="`)
	r.uiText(label)
	r.raw(`">`)
	if starlight {
		r.starlightIcon(iconName, 16)
	} else {
		r.icon(iconName, 16)
	}
	r.raw(`</a>`)
}

func (r *pageRenderer) renderLanguageMenu() {
	if len(r.data.LangLinks) == 0 {
		return
	}
	r.raw(`<div class="header-group utility-select utility-menu language-select"><button class="utility-menu-trigger" type="button" popovertarget="mpress-language-menu" aria-label="Select language: `)
	for _, language := range r.data.LangLinks {
		if language.Code == r.data.Page.Language {
			r.text(language.Label)
		}
	}
	r.raw("\" aria-haspopup=\"menu\" aria-expanded=\"false\" title=\"")
	r.uiText("Change language")
	r.raw("\">")
	r.icon("languages", 19)
	r.icon("chevron-down", 12)
	r.raw(`</button><menu id="mpress-language-menu" class="utility-menu-panel utility-language-menu" popover data-utility-menu>`)
	for _, language := range r.data.LangLinks {
		r.raw(`<li><a href="`)
		r.url(language.URL)
		r.raw(`" role="menuitem"`)
		if language.Code == r.data.Page.Language {
			r.raw(` aria-current="true"`)
		}
		r.raw(`><span>`)
		r.text(language.Label)
		r.raw(`</span>`)
		if !language.Available {
			r.raw(`<small>`)
			r.text(language.FallbackLabel)
			r.raw(` fallback</small>`)
		}
		r.raw(`</a></li>`)
	}
	r.raw(`</menu></div>`)
}

func (r *pageRenderer) renderBlogIndex() {
	d := r.data
	blog := d.Blog
	landingStyle := d.Config.Blog.LandingStyle
	if landingStyle != "grid" && landingStyle != "list" {
		landingStyle = "featured"
	}
	r.raw(`<div class="blog-layout" data-blog-landing-style="`)
	r.attr(landingStyle)
	r.raw("\"><aside class=\"blog-sidebar\" aria-label=\"")
	r.uiText("Blog archive")
	r.raw("\"><div class=\"blog-sidebar-inner\"><span class=\"blog-sidebar-label\">")
	r.uiText("Browse")
	r.raw("</span><button class=\"blog-filter active\" type=\"button\" data-blog-filter=\"\" aria-pressed=\"true\">")
	r.uiText("All articles")
	r.raw("</button>")
	if len(blog.Tags) > 0 {
		r.raw("<span class=\"blog-sidebar-label\">")
		r.uiText("Topics")
		r.raw("</span>")
		for _, tag := range blog.Tags {
			r.raw(`<button class="blog-filter" type="button" data-blog-filter="`)
			r.attr(tag)
			r.raw(`" aria-pressed="false">`)
			r.text(tag)
			r.raw(`</button>`)
		}
	}
	if d.Config.Social.RSS != "" {
		r.raw(`<a class="blog-rss" href="`)
		r.url(r.resourceURL(d.Config.Social.RSS))
		r.raw(`">`)
		r.icon("rss", 15)
		r.raw("<span>")
		r.uiText("RSS feed")
		r.raw("</span></a>")
	}
	r.raw("</div></aside><main class=\"blog-main\" id=\"content\"><header class=\"blog-intro\"><span class=\"blog-kicker\">")
	r.uiText("News and releases")
	r.raw("</span><h1>")
	r.text(d.Page.Title)
	r.raw(`</h1>`)
	tagline := strings.TrimSpace(d.Config.Blog.Tagline)
	if tagline == "" {
		tagline = d.Page.Description
	}
	if tagline != "" {
		r.raw(`<p>`)
		r.text(tagline)
		r.raw(`</p>`)
	}
	r.raw(`</header>`)
	if landingStyle == "featured" && len(blog.Posts) > 0 {
		r.renderBlogHero(blog.Posts[0])
	}
	archivePosts := blog.Posts
	archiveHeading := "Articles"
	if landingStyle == "featured" && len(archivePosts) > 0 {
		archivePosts = archivePosts[1:]
		archiveHeading = "More articles"
	}
	if len(archivePosts) > 0 {
		r.raw(`<section class="blog-archive" aria-labelledby="blog-archive-heading"><div class="blog-archive-heading"><h2 id="blog-archive-heading">`)
		r.uiText(archiveHeading)
		r.raw(`</h2><span>`)
		if len(archivePosts) == 1 {
			r.uiTextf("{0} article", len(archivePosts))
		} else {
			r.uiTextf("{0} articles", len(archivePosts))
		}
		r.raw(`</span></div><div class="blog-grid">`)
		for _, post := range archivePosts {
			r.renderBlogCard(post)
		}
		r.raw(`</div></section>`)
	}
	r.raw(`</main></div>`)
}

func (r *pageRenderer) renderBlogStyleAttrs(post blogPostData, hero bool) {
	r.raw(` data-blog-style-editor data-blog-tags="`)
	r.attr(strings.Join(post.Tags, "|"))
	r.raw(`" data-blog-tags-visible="`)
	r.raw(strconv.FormatBool(post.ShowTags))
	r.raw(`" data-blog-heading-size="`)
	r.attr(post.HeadingSize)
	r.raw(`"`)
	if post.Image == "" {
		return
	}
	r.raw(` data-blog-image-editor`)
	if hero {
		r.raw(` data-blog-image-mode="`)
		r.attr(post.ImageMode)
		r.raw(`" data-blog-image-fit="`)
		r.attr(post.ImageFit)
		r.raw(`" data-blog-image-width="`)
		r.raw(strconv.Itoa(post.ImageWidth))
		r.raw(`"`)
	} else {
		r.raw(` data-blog-image-fit="`)
		r.attr(post.ImageFit)
		r.raw(`"`)
	}
	if post.ImageBackground != "" {
		r.raw(` data-blog-image-background="`)
		r.attr(post.ImageBackground)
		r.raw(`"`)
	}
	if hero {
		r.raw(` style="`)
		if post.ImageBackground != "" {
			r.raw(`--blog-image-background:`)
			r.attr(post.ImageBackground)
			r.raw(`;`)
		}
		r.raw(`--blog-floating-image-width:`)
		r.raw(strconv.Itoa(post.ImageWidth))
		r.raw(`%"`)
	} else if post.ImageBackground != "" {
		r.raw(` style="--blog-image-background:`)
		r.attr(post.ImageBackground)
		r.raw(`"`)
	}
}

func (r *pageRenderer) renderBlogHero(post blogPostData) {
	r.raw(`<article class="blog-hero"`)
	r.renderBlogStyleAttrs(post, true)
	r.raw(`><a class="blog-hero-image`)
	if post.Image == "" {
		r.raw(` no-image`)
	}
	r.raw(`" href="`)
	r.url(r.pageURL(post.Route))
	r.raw(`" aria-label="Read `)
	r.attr(post.Title)
	r.raw(`">`)
	if post.Image != "" {
		r.raw(`<img src="`)
		r.url(r.resourceURL(post.Image))
		r.raw(`" alt="">`)
	} else {
		r.raw(`<span>`)
		r.icon("newspaper", 42)
		r.raw(`</span>`)
	}
	r.raw("</a><div class=\"blog-hero-copy\"><div class=\"blog-hero-label-row\"><span class=\"blog-featured-label\">")
	r.uiText("Latest article")
	r.raw("</span>")
	r.renderBlogTags(post.Tags)
	r.raw(`</div><h2><a href="`)
	r.url(r.pageURL(post.Route))
	r.raw(`">`)
	r.text(post.Title)
	r.raw(`</a></h2>`)
	if post.Excerpt != "" {
		r.raw(`<p>`)
		r.text(post.Excerpt)
		r.raw(`</p>`)
	}
	r.renderBlogMeta(post, true)
	r.raw(`<a class="blog-read" href="`)
	r.url(r.pageURL(post.Route))
	r.raw("\"><span>")
	r.uiText("Read article")
	r.raw("</span>")
	r.icon("arrow-right", 16)
	r.raw(`</a></div></article>`)
}

func (r *pageRenderer) renderBlogCard(post blogPostData) {
	r.raw(`<article class="blog-card"`)
	r.renderBlogStyleAttrs(post, false)
	r.raw(`>`)
	if post.Image != "" {
		r.raw(`<a class="blog-card-image" href="`)
		r.url(r.pageURL(post.Route))
		r.raw(`" aria-label="Read `)
		r.attr(post.Title)
		r.raw(`"><img src="`)
		r.url(r.resourceURL(post.Image))
		r.raw(`" alt=""></a>`)
	}
	r.raw(`<div class="blog-card-copy">`)
	r.renderBlogTags(post.Tags)
	r.raw(`<h3><a href="`)
	r.url(r.pageURL(post.Route))
	r.raw(`">`)
	r.text(post.Title)
	r.raw(`</a></h3>`)
	if post.Excerpt != "" {
		r.raw(`<p>`)
		r.text(post.Excerpt)
		r.raw(`</p>`)
	}
	r.raw(`<div class="blog-card-footer">`)
	r.renderBlogMeta(post, false)
	r.raw(`<a href="`)
	r.url(r.pageURL(post.Route))
	r.raw(`" aria-label="Read `)
	r.attr(post.Title)
	r.raw(`">`)
	r.icon("arrow-right", 16)
	r.raw(`</a></div></div></article>`)
}

func (r *pageRenderer) renderBlogTags(tags []string) {
	if len(tags) == 0 {
		return
	}
	r.raw(`<div class="blog-tags">`)
	for _, tag := range tags {
		r.raw(`<span>`)
		r.text(tag)
		r.raw(`</span>`)
	}
	r.raw(`</div>`)
}

func (r *pageRenderer) renderBlogMeta(post blogPostData, expanded bool) {
	r.raw(`<div class="blog-meta">`)
	if post.Date != "" {
		r.raw(`<time datetime="`)
		r.attr(post.Date)
		r.raw(`">`)
		r.renderBlogDate(post)
		r.raw(`</time>`)
	}
	if expanded && post.Author != "" {
		r.raw(`<span>`)
		r.text(post.Author)
		r.raw(`</span>`)
	}
	r.raw(`<span>`)
	if expanded {
		r.uiTextf("{0} min read", post.ReadingTime)
	} else {
		r.uiTextf("{0} min", post.ReadingTime)
	}
	r.raw(`</span></div>`)
}

func (r *pageRenderer) renderBlogArticle() {
	d := r.data
	article := d.BlogArticle
	r.raw("<div class=\"blog-layout blog-article-layout\"><aside class=\"blog-sidebar\" aria-label=\"")
	r.uiText("Blog archive")
	r.raw("\"><div class=\"blog-sidebar-inner\"><a class=\"blog-archive-home\" href=\"")
	r.url(r.pageURL(article.IndexRoute))
	r.raw(`">`)
	r.icon("arrow-left", 14)
	r.raw("<span>")
	r.uiText("All articles")
	r.raw("</span></a>")
	for _, period := range article.Periods {
		r.raw(`<section class="blog-period"><span class="blog-sidebar-label">`)
		r.text(period.Label)
		r.raw(`</span><div class="blog-period-posts">`)
		for _, post := range period.Posts {
			active := post.Route == d.Page.URLPath
			r.raw(`<a class="blog-article-link`)
			if active {
				r.raw(" ")
				r.raw("active")
			}
			r.raw(`"`)
			if active {
				r.raw(` aria-current="page"`)
			}
			r.raw(` href="`)
			r.url(r.pageURL(post.Route))
			r.raw(`"><span>`)
			r.text(post.Title)
			r.raw(`</span>`)
			if post.Date != "" {
				r.raw(`<time datetime="`)
				r.attr(post.Date)
				r.raw(`">`)
				r.renderBlogDate(post)
				r.raw(`</time>`)
			}
			r.raw(`</a>`)
		}
		r.raw(`</div></section>`)
	}
	r.raw(`</div></aside><main class="blog-article-main" id="content"><article class="blog-article"><header class="blog-article-header"><a class="blog-article-kicker" href="`)
	r.url(r.pageURL(article.IndexRoute))
	r.raw("\">")
	r.uiText("Blog")
	r.raw("</a><h1>")
	r.text(d.Page.Title)
	r.raw(`</h1>`)
	if d.Page.Description != "" {
		r.raw(`<p>`)
		r.text(d.Page.Description)
		r.raw(`</p>`)
	}
	r.renderBlogMeta(article.Post, true)
	r.raw(`</header><div class="blog-article-body">`)
	r.raw(d.Page.HTML)
	r.raw(`</div></article></main></div>`)
}

func (r *pageRenderer) renderLanding() {
	d := r.data
	if d.Page.Meta.Banner.Content != "" {
		r.raw(`<aside class="mpress-site-banner">`)
		r.raw(d.Page.Meta.Banner.Content)
		r.raw(`</aside>`)
	}
	r.raw(`<main class="landing-main" id="content">`)
	hero := d.Page.Meta.Hero
	if hero.Tagline != "" {
		r.raw(`<section class="mpress-frontmatter-hero"><div class="mpress-frontmatter-hero-inner">`)
		if hero.Image.Light != "" || hero.Image.Dark != "" {
			r.raw(`<span class="mpress-frontmatter-hero-image">`)
			if hero.Image.Light != "" {
				r.raw(`<img class="hero-logo-light" src="`)
				r.url(r.data.Root + strings.Trim(hero.Image.Light, "/"))
				r.raw(`" alt="`)
				r.attr(hero.Image.Alt)
				r.raw(`">`)
			}
			if hero.Image.Dark != "" {
				r.raw(`<img class="hero-logo-dark" src="`)
				r.url(r.data.Root + strings.Trim(hero.Image.Dark, "/"))
				r.raw(`" alt="`)
				r.attr(hero.Image.Alt)
				r.raw(`">`)
			}
			r.raw(`</span>`)
		}
		r.raw(`<h1 class="mpress-frontmatter-hero-title">`)
		r.text(d.Page.Title)
		r.raw(`</h1><p class="mpress-frontmatter-hero-tagline">`)
		r.text(hero.Tagline)
		r.raw(`</p>`)
		if len(hero.Actions) > 0 {
			r.raw(`<div class="mpress-frontmatter-hero-actions">`)
			for _, action := range hero.Actions {
				r.raw(`<a class="mpress-frontmatter-hero-action`)
				if action.Variant == "primary" {
					r.raw(` mpress-frontmatter-hero-action-primary`)
				}
				r.raw(`" href="`)
				r.url(r.pageURL(action.Link))
				r.raw(`"><span>`)
				r.text(action.Text)
				r.raw(`</span>`)
				if action.Icon != "" {
					r.icon(action.Icon, 16)
				}
				r.raw(`</a>`)
			}
			r.raw(`</div>`)
		}
		r.raw(`</div></section>`)
	}
	r.raw(d.Page.HTML)
	r.raw(`</main>`)
}

func (r *pageRenderer) renderNotFound() {
	r.raw("<main class=\"mpress-not-found\" id=\"content\"><div><span class=\"mpress-not-found-code\">404</span><h1>")
	r.uiText("Page not found")
	r.raw("</h1><p>")
	r.uiText("The page you requested does not exist or may have moved.")
	r.raw("</p><a class=\"mpress-button mpress-button-primary\" href=\"")
	r.url(r.pageURL("/"))
	r.raw("\">")
	r.uiText("Go to documentation home")
	r.raw("</a></div></main>")
}

func (r *pageRenderer) renderDocs() {
	d := r.data
	r.raw("<div class=\"layout\"><aside class=\"sidebar\" id=\"mpress-sidebar\" tabindex=\"-1\"><nav aria-label=\"")
	r.uiText("Documentation sidebar")
	r.raw("\">")
	if d.CompiledNav != nil {
		d.CompiledNav.render(r.out, d.Root, d.Page.URLPath)
	} else {
		r.renderNav(d.Nav, d.Page.URLPath)
	}
	r.raw(`</nav></aside><div class="docs-stage"><main id="content"><article><header class="docs-page-header"><h1>`)
	r.text(d.Page.Title)
	r.raw(`</h1>`)
	if d.Page.Description != "" {
		r.raw(`<p class="page-lead">`)
		r.text(d.Page.Description)
		r.raw(`</p>`)
	}
	r.raw(`</header>`)
	r.raw(d.Page.HTML)
	r.raw(`</article>`)
	if (d.Config.Social.EditURL != "" && !d.Page.Meta.Generated) || !d.Page.LastModified.IsZero() {
		r.raw(`<div class="page-meta">`)
		if d.Config.Social.EditURL != "" && !d.Page.Meta.Generated {
			r.raw(`<a href="`)
			r.url(pageEditURL(d.Config.Social.EditURL, pageEditSource(d.Page, d.Config.Site.DefaultLanguage)))
			r.raw(`">`)
			r.icon("pencil", 15)
			r.raw("<span>")
			r.uiText("Edit page")
			r.raw("</span></a>")
		}
		if !d.Page.LastModified.IsZero() {
			r.raw("<p>")
			r.uiText("Last updated:")
			r.raw(" <time datetime=\"")
			r.attr(isoTime(d.Page.LastModified))
			r.raw(`">`)
			r.text(r.readerDate(d.Page.LastModified))
			r.raw(`</time></p>`)
		}
		r.raw(`</div>`)
	}
	r.raw(`<nav class="pager">`)
	if d.Prev != nil {
		r.raw(`<a rel="prev" href="`)
		r.url(r.pageURL(d.Prev.Link))
		r.raw(`"><span class="pager-direction">`)
		r.icon("arrow-left", 16)
		r.raw("<span>")
		r.uiText("Previous")
		r.raw("</span></span><strong>")
		r.text(d.Prev.Label)
		r.raw(`</strong></a>`)
	} else {
		r.raw(`<span></span>`)
	}
	if d.Next != nil {
		r.raw(`<a rel="next" href="`)
		r.url(r.pageURL(d.Next.Link))
		r.raw("\"><span class=\"pager-direction\"><span>")
		r.uiText("Next")
		r.raw("</span>")
		r.icon("arrow-right", 16)
		r.raw(`</span><strong>`)
		r.text(d.Next.Label)
		r.raw(`</strong></a>`)
	}
	r.raw("</nav></main><aside class=\"toc\"><strong><span>")
	r.uiText("On this page")
	r.raw("</span><span class=\"toc-mobile-icon\">")
	r.icon("chevron-right", 14)
	r.raw("</span></strong><a class=\"toc-level-1\" href=\"#content\">")
	r.uiText("Overview")
	r.raw("</a>")
	for _, heading := range d.Page.Headings {
		if heading.Level != 2 && heading.Level != 3 {
			continue
		}
		r.raw(`<a class="toc-level-`)
		r.raw(strconv.Itoa(heading.Level))
		r.raw(`" href="#`)
		r.attr(heading.ID)
		r.raw(`">`)
		r.text(heading.Text)
		r.raw(`</a>`)
	}
	r.raw(`</aside></div></div>`)
}

func (r *pageRenderer) renderNav(items []navigation.Item, current string) {
	for _, item := range items {
		label, translated := item.LabelFor(r.data.Page.Language)
		if !translated && configuredLanguage(r.data.Config.Site.Languages, r.data.Page.Language) && r.data.Page.Language != r.data.Config.Site.DefaultLanguage && item.Link != "" {
			if page := r.data.CurrentRoutes[strings.Trim(item.Link, "/")]; page != nil && strings.TrimSpace(page.Title) != "" {
				label, translated = page.Title, true
			}
		}
		if len(item.Items) > 0 {
			r.raw(`<details`)
			if !item.Collapsed || navContains(item.Items, current) {
				r.raw(` open`)
			}
			r.raw(`><summary><span`)
			r.renderNavFallbackLabelAttributes(label, translated)
			r.raw(`>`)
			r.text(label)
			r.raw(`</span>`)
			r.icon("chevron-right", 14)
			r.raw(`</summary><div>`)
			r.renderNav(item.Items, current)
			r.raw(`</div></details>`)
			continue
		}
		if item.Link != "" {
			active := strings.Trim(item.Link, "/") == current
			r.raw(`<a class="`)
			if active {
				r.raw("active")
			}
			r.raw(`"`)
			if active {
				r.raw(` aria-current="page"`)
			}
			r.raw(` href="`)
			r.url(r.pageURL(item.Link))
			r.raw(`">`)
			r.text(label)
			r.raw(`</a>`)
			continue
		}
		r.raw(`<span class="nav-label"`)
		r.renderNavFallbackLabelAttributes(label, translated)
		r.raw(`>`)
		r.text(label)
		r.raw(`</span>`)
	}
}

func configuredLanguage(languages []string, language string) bool {
	for _, candidate := range languages {
		if candidate == language {
			return true
		}
	}
	return false
}

func (r *pageRenderer) renderNavFallbackLabelAttributes(label string, translated bool) {
	language := r.data.Page.Language
	defaultLanguage := r.data.Config.Site.DefaultLanguage
	if translated || language == "" || language == defaultLanguage || !configuredLanguage(r.data.Config.Site.Languages, language) {
		return
	}
	r.raw(` lang="`)
	r.attr(defaultLanguage)
	r.raw(`" title="`)
	r.attr(label + " is shown in " + strings.ToUpper(defaultLanguage) + " because a translated navigation label is not configured")
	r.raw(`"`)
}

func (r *pageRenderer) pageURL(link string) string {
	return pageNavURL(r.data.Root, link, r.data.Page.Language, r.data.Config.Site.DefaultLanguage, r.data.Config.Site.DefaultAtRoot, r.data.CurrentRoutes)
}

func (r *pageRenderer) resourceURL(value string) string {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "data:") {
		return value
	}
	return r.data.Root + strings.TrimLeft(value, "/")
}

func pageEditURL(base, source string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(source, "/")
}

func pageEditSource(page *content.Page, defaultLanguage string) string {
	if page == nil {
		return ""
	}
	path := filepath.ToSlash(filepath.Clean(strings.TrimSpace(page.Meta.SourcePath)))
	if path == "." || path == ".." || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") {
		path = page.SourcePath
	}
	if page.Language != "" && page.Language != defaultLanguage && !strings.HasPrefix(path, page.Language+"/") {
		path = page.Language + "/" + path
	}
	return path
}

func isoTime(value time.Time) string { return value.UTC().Format(time.RFC3339) }

func displayDate(value time.Time) string { return value.UTC().Format("Jan 2, 2006") }

func pageNavURL(root, link, language, defaultLanguage string, defaultAtRoot bool, routes map[string]*content.Page) string {
	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		return link
	}
	clean := strings.Trim(link, "/")
	if language != "" && (clean == language || strings.HasPrefix(clean, language+"/")) {
		return root + clean + "/"
	}
	if language != "" && (language != defaultLanguage || !defaultAtRoot) && routes[clean] != nil {
		if clean == "" {
			clean = language
		} else {
			clean = language + "/" + clean
		}
	}
	if clean == "" {
		return root
	}
	return root + clean + "/"
}

func navContains(items []navigation.Item, current string) bool {
	current = strings.Trim(current, "/")
	for _, item := range items {
		if strings.Trim(item.Link, "/") == current && current != "" {
			return true
		}
		if navContains(item.Items, current) {
			return true
		}
	}
	return false
}
