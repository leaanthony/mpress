package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/knowledge"
	"github.com/leaanthony/mpress/internal/quickedit"
	docversion "github.com/leaanthony/mpress/internal/version"
)

func TestBuildWritesSearchableKnowledgeArtifacts(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Knowledge integration
  baseURL: https://docs.example.test
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Install Acme\ntags: [setup]\n---\n\n# Install Acme\n\nConfigure a secure token.\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	store, err := knowledge.Load(filepath.Join(root, "site"))
	if err != nil {
		t.Fatal(err)
	}
	results := store.Search(knowledge.SearchOptions{Query: "secure token"})
	if len(results) != 1 || results[0].URL != "https://docs.example.test/" {
		t.Fatalf("unexpected knowledge search results: %#v", results)
	}
}

func TestBuildCanDisableKnowledgeArtifacts(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: No knowledge artifacts
build:
  contentDir: content
  staticDir: static
  outputDir: site
knowledge:
  enabled: false
`)
	writeFixture(t, root, "content/index.md", "# Home\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "site", knowledge.Directory)); !os.IsNotExist(err) {
		t.Fatalf("disabled knowledge directory exists: %v", err)
	}
}

func TestBuildParseCacheInvalidatesChangedSource(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Cache test
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: First\n---\n\n# First\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	entries, err := filepath.Glob(filepath.Join(root, ".mpress", "cache", "parse", "*.glint"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("parse cache was not populated: %v", err)
	}

	writeFixture(t, root, "content/index.md", "---\ntitle: Second\n---\n\n# Second\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "Second") || strings.Contains(string(page), "First") {
		t.Fatalf("changed source reused stale parse cache: %s", page)
	}
}

func TestBuildWritesStaticCacheHeaders(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Cache headers
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	writeFixture(t, root, "static/_headers", "/custom/*\n  X-Test: kept\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	headers, err := os.ReadFile(filepath.Join(root, "site", "_headers"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(headers)
	for _, want := range []string{"/assets/*", "/versions/*", "max-age=31536000", "/*.html"} {
		if !strings.Contains(text, want) {
			t.Fatalf("cache headers missing %q: %s", want, text)
		}
	}
	if !strings.Contains(text, "X-Test: kept") {
		t.Fatalf("custom cache headers were discarded: %s", text)
	}
}

func TestProductionBuildOmitsScheduledBlogPosts(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Scheduled posts
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	writeFixture(t, root, "content/blog/index.md", "---\ntitle: Blog\n---\n\n# Blog\n")
	writeFixture(t, root, "content/blog/future.md", "---\ntitle: Future post\ndate: 2999-01-01\n---\n\nNot published yet.\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{OutputDir: "production"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "production", "blog", "future", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("scheduled article was included in production: %v", err)
	}
	if _, err := Build(root, BuildOptions{OutputDir: "preview", IncludeDrafts: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "preview", "blog", "future", "index.html")); err != nil {
		t.Fatalf("scheduled article is missing from development preview: %v", err)
	}
}

func TestParseCacheReadsGlintAndLegacyJSON(t *testing.T) {
	root := t.TempDir()
	cache := newParseCache(root)
	if cache == nil {
		t.Fatal("expected parse cache")
	}
	page := &content.Page{SourcePath: "guide.md", Language: "en", URLPath: "guide", OutputPath: "guide/index.html", Title: "Guide", HTML: "<h1>Guide</h1>", PlainText: "Guide", Headings: []content.Heading{{ID: "guide", Text: "Guide", Level: 1}}}
	document := &quickedit.Document{Version: quickedit.Version, Revision: "revision", Segments: []quickedit.Segment{{ID: "body-001", Kind: "paragraph", Original: "Guide", Start: 0, End: 5, SourceHash: "source"}}}
	cache.saveWithQuickEdit("glint", page, []content.Diagnostic{{Severity: "warning", Code: "example", Message: "Example"}}, document)
	loaded, diagnostics, ok := cache.load("glint")
	if !ok || loaded == nil || loaded.Title != page.Title || loaded.HTML != page.HTML || len(loaded.Headings) != 1 || len(diagnostics) != 1 {
		t.Fatalf("Glint cache did not round-trip: page=%#v diagnostics=%#v ok=%t", loaded, diagnostics, ok)
	}
	entry, ok := cache.loadEntry("glint")
	if !ok || entry.QuickEdit == nil || entry.QuickEdit.Revision != document.Revision || len(entry.QuickEdit.Segments) != 1 {
		t.Fatalf("quick-edit metadata did not round-trip: %#v", entry.QuickEdit)
	}

	legacy := parseCacheEntry{Page: page, Diagnostics: []content.Diagnostic{{Severity: "warning", Code: "legacy", Message: "Legacy"}}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache.dir, "legacy.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// A real build starts with any legacy JSON entries already on disk, so a
	// freshly constructed cache detects them and enables the migration path.
	cache = newParseCache(root)
	loaded, diagnostics, ok = cache.load("legacy")
	if !ok || loaded == nil || loaded.Title != page.Title || len(diagnostics) != 1 || diagnostics[0].Code != "legacy" {
		t.Fatalf("legacy JSON cache did not load: page=%#v diagnostics=%#v ok=%t", loaded, diagnostics, ok)
	}
}

func TestBuildMultilingualSiteWithoutCopyingMissingPages(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Test Docs
  description: A test site
  baseURL: https://docs.example.test
  socialImage: social-card.png
  defaultLanguage: en
  languages: [en, fr]
  languageLabels: {en: English, fr: Français}
  defaultLanguageAtRoot: true
  missingTranslation: link-to-default
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
search:
  enabled: true
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Welcome\n\n[Guide](/guide/)\n")
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n")
	writeFixture(t, root, "content/fr/index.md", "---\ntitle: Accueil\n---\n\n# Bonjour\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "static/social-card.png", "image fixture")

	result, err := Build(root, BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("build: %v; diagnostics: %#v", err, result.Diagnostics)
	}
	if result.Pages != 3 {
		t.Fatalf("expected 3 pages, got %d", result.Pages)
	}
	wantTimingNames := []string{"discover", "parse", "prepare", "render", "finalize"}
	if len(result.Timings) != len(wantTimingNames) {
		t.Fatalf("expected %d build timings, got %#v", len(wantTimingNames), result.Timings)
	}
	for index, name := range wantTimingNames {
		if result.Timings[index].Name != name || result.Timings[index].Status != "passed" || result.Timings[index].DurationMS < 0 {
			t.Fatalf("unexpected build timing %d: %#v", index, result.Timings[index])
		}
	}
	assertExists(t, filepath.Join(root, "site", "index.html"))
	assertExists(t, filepath.Join(root, "site", "guide", "index.html"))
	assertExists(t, filepath.Join(root, "site", "fr", "index.html"))
	if _, err := os.Stat(filepath.Join(root, "site", "fr", "guide", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("missing French page was copied instead of omitted")
	}
	if broken, err := check.Run(filepath.Join(root, "site")); err != nil || len(broken) != 0 {
		t.Fatalf("link check: %v, %#v", err, broken)
	}
	home, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`rel="canonical" href="https://docs.example.test/"`,
		`property="og:title" content="Home · Test Docs"`,
		`property="og:description" content="A test site"`,
		`property="og:type" content="website"`,
		`property="og:url" content="https://docs.example.test/"`,
		`property="og:image" content="https://docs.example.test/social-card.png"`,
		`name="twitter:card" content="summary_large_image"`,
		`name="twitter:image" content="https://docs.example.test/social-card.png"`,
		`hreflang="fr" href="https://docs.example.test/fr/"`,
		`hreflang="x-default" href="https://docs.example.test/"`,
		`id="search" type="button"`,
		`role="combobox" aria-label="Search" aria-autocomplete="list"`,
		`aria-current="page"`,
	} {
		if !strings.Contains(string(home), want) {
			t.Errorf("generated translation metadata missing %q: %s", want, home)
		}
	}
	bootstrapAt := strings.Index(string(home), "localStorage.getItem('mpress-theme')")
	stylesheetAt := strings.Index(string(home), `<link rel="stylesheet"`)
	if bootstrapAt < 0 || stylesheetAt < 0 || bootstrapAt > stylesheetAt {
		t.Fatalf("theme preference bootstrap must run before the stylesheet: bootstrap=%d stylesheet=%d", bootstrapAt, stylesheetAt)
	}
	guide, err := os.ReadFile(filepath.Join(root, "site", "guide", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), `Français`) || !strings.Contains(string(guide), `English fallback`) {
		t.Fatalf("missing-translation label did not name the configured default language: %s", guide)
	}
	theme, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(theme), "--accent: #5375f6") || !strings.Contains(string(theme), "--hover-light-seed: #7593ff") || !strings.Contains(string(theme), "--hover-dark-seed: #7593ff") || !strings.Contains(string(theme), "--hover-light: color-mix(in srgb, var(--hover-light-seed) 45%, #000)") || !strings.Contains(string(theme), "--hover-dark: color-mix(in srgb, var(--hover-dark-seed) 30%, #fff)") || !strings.Contains(string(theme), "--hover: var(--hover-light)") || !strings.Contains(string(theme), "--hover: var(--hover-dark)") {
		t.Fatalf("generated theme does not use the default interaction colours")
	}
	for _, want := range []string{"html { max-width: 100%; overflow-x: clip;", "body {\n  max-width: 100%;", ".sidebar .nav-label", ".sidebar nav > a, .sidebar nav > .nav-label { color: var(--text); font-size: 16px; font-weight: 600; }", ".sidebar details > div", "padding-left: .65rem", ".docs-stage > .toc { grid-column: 4; }", ".sidebar, .toc { position: sticky; top: 58px; align-self: start;", ".toc a.active", "font-size: 13px", "font-size: 38px", "font-size: 15px", "font-weight: 400", ".primary-links a:hover { background: var(--surface); color: var(--hover);", ".mpress-tabs [role=\"tablist\"]", "gap: 1.25rem", "border-bottom-color: var(--accent)", ".mpress-cards-1 { grid-template-columns: minmax(0, 1fr); }", ".docs-page > footer { position: absolute;", "bottom: var(--mpress-devbar-height, 0px)", ".docs-page .sidebar nav::after, .docs-page .toc::after", ".mpress-a11y-options { position: absolute;", "transform: translate(16px, -50%)", "--warning:", ".utility-menu-panel[popover]", "backdrop-filter: blur(16px)", "a[aria-current=\"true\"]", "--search-highlight: color-mix(in srgb, var(--text) 17%, var(--surface-solid))", ".mpress-search-item.active { background: var(--search-selected)", ".mpress-search-item mark, .mpress-search-preview mark", "background: var(--search-highlight); color: var(--text)", ".mpress-search-preview-headings a:hover { background: var(--search-hover); color: var(--text);", ".mpress-search-clear:hover { background: var(--search-hover); color: var(--text);"} {
		if !strings.Contains(string(theme), want) {
			t.Errorf("generated theme lost navigation hierarchy rule %q", want)
		}
	}
	if strings.Contains(string(theme), ".mpress-admonition::before") {
		t.Fatal("default admonitions must not render a decorative left-side accent")
	}
	script, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.js"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"const tocItems", "requestAnimationFrame(updateTOC)", "aria-current', 'location'", "navigator.userAgentData?.platform", "window.mpressKeyboard", "shortcutMatches", "dataset.shortcutSearch", "dataset.shortcutAccessibility", "mpress-search-overlay", "query.addEventListener('pointerdown'", "scoreItem", "fuzzyMatch", "shared >= Math.max(6, Math.min(8, shorter - 1))", "Math.max(wordScore, 80)", "mpress-recent-searches", "renderPreview", "syncThemeButton", "const themeModes = ['system', 'dark', 'light']", "const utilityMenus", "positionUtilityMenu", "menu.showPopover()", "setAccessibilitySelectOpen", "syncAccessibilitySelect", "const tabControllers", `:scope > [role="tablist"] > [role="tab"]`, `:scope > [role="tabpanel"]`, "peer.syncKey !== controller.syncKey", "peer.select(peerIndex, false)"} {
		if !strings.Contains(string(script), want) {
			t.Errorf("generated theme lost table-of-contents scrollspy %q", want)
		}
	}
	if strings.Contains(string(script), "current.link.scrollIntoView") {
		t.Error("table-of-contents scrollspy must not move the document while the user scrolls")
	}
	sitemap, _ := os.ReadFile(filepath.Join(root, "site", "sitemap.xml"))
	if !strings.Contains(string(sitemap), "https://docs.example.test/guide/") {
		t.Fatalf("sitemap lost nested route: %s", sitemap)
	}
	robots, _ := os.ReadFile(filepath.Join(root, "site", "robots.txt"))
	if string(robots) != "User-agent: *\nAllow: /\nSitemap: https://docs.example.test/sitemap.xml\n" {
		t.Fatalf("unexpected robots.txt: %q", robots)
	}
	llms, _ := os.ReadFile(filepath.Join(root, "site", "llms.txt"))
	for _, want := range []string{"# Test Docs", "> A test site", "[Home](https://docs.example.test/)", "[Guide](https://docs.example.test/guide/)"} {
		if !strings.Contains(string(llms), want) {
			t.Errorf("generated llms.txt missing %q: %s", want, llms)
		}
	}
	if strings.Contains(string(llms), "Accueil") {
		t.Fatalf("llms.txt should describe the default-language corpus once: %s", llms)
	}
	notFound, _ := os.ReadFile(filepath.Join(root, "site", "404.html"))
	for _, want := range []string{`class="not-found-page"`, `class="mpress-not-found"`, `Page not found`, `name="robots" content="noindex,follow"`, `href="./"`} {
		if !strings.Contains(string(notFound), want) {
			t.Errorf("generated 404 page missing %q: %s", want, notFound)
		}
	}
	index, _ := os.ReadFile(filepath.Join(root, "site", "fr", "search-index.json"))
	var items []map[string]any
	if err := json.Unmarshal(index, &items); err != nil || len(items) != 1 {
		t.Fatalf("French search index: %v, %s", err, index)
	}
	if _, ok := items[0]["headings"]; !ok {
		t.Fatalf("French search index does not include headings for rich result previews: %s", index)
	}
}

func TestSearchIndexKeepsPageTitleWhenMatchingBodyHeadingIsNormalised(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	page := &content.Page{
		Title:       "Window Options",
		Description: "Configure application windows.",
		URLPath:     "window-options",
		PlainText:   "Set the width, height, and appearance of a window.",
		Headings: []content.Heading{
			{ID: "webviewwindowoptions-structure", Text: "WebviewWindowOptions Structure", Level: 2},
		},
	}
	if err := writeSearchIndex(root, cfg.Site.DefaultLanguage, cfg, []*content.Page{page}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var items []searchItem
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Headings) != 2 {
		t.Fatalf("search index lost the page title heading: %s", data)
	}
	if got := items[0].Headings[0]; got.ID != "content" || got.Text != page.Title || got.Level != 1 {
		t.Fatalf("unexpected title heading: %#v", got)
	}

	page.Headings = append([]content.Heading{
		{ID: "window-options", Text: page.Title, Level: 1},
	}, page.Headings...)
	if err := writeSearchIndex(root, cfg.Site.DefaultLanguage, cfg, []*content.Page{page}); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, "search-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Headings) != 2 || items[0].Headings[0].ID != "window-options" {
		t.Fatalf("search index duplicated an existing page title heading: %s", data)
	}
}

func TestBuildPublishesContributionHandoff(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Contributor docs
  defaultLanguage: en
  languages: [en, fr]
  defaultLanguageAtRoot: true
build:
  contentDir: content
  staticDir: static
  outputDir: site
contribution:
  enabled: true
  repository: https://github.com/example/docs.git
  branch: next
  quickEdit: true
  guide: CONTRIBUTING.md
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	writeFixture(t, root, "content/fr/index.md", "---\ntitle: Accueil\n---\n\n# Accueil\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "fr", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(page)
	for _, want := range []string{
		`name="mpress:repository" content="https://github.com/example/docs.git"`,
		`name="mpress:branch" content="next"`,
		`name="mpress:source" content="content/fr/index.md"`,
		`name="mpress:route" content="/fr/"`,
		`name="mpress:guide" content="CONTRIBUTING.md"`,
		`data-mpress-contribute`,
		`aria-labelledby="mpress-contribute-title"`,
		`mpress-contribute-choices-public`,
		`Translate documentation`,
		`data-contribute-translate`,
		`mpress-contribute.sh`,
		`mpress-contribute.ps1`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("contribution page missing %q", want)
		}
	}
	for _, hiddenFromProduction := range []string{`id="mpress-quick-edit-data"`, `data-quick-edit-bar`, `data-contribute-quick-edit`, `Fix this page`, `Continue on computer`} {
		if strings.Contains(markup, hiddenFromProduction) {
			t.Errorf("production contribution page exposes local quick editing %q", hiddenFromProduction)
		}
	}
	if _, err := Build(root, BuildOptions{Strict: true, Development: true, OutputDir: "dev-site"}); err != nil {
		t.Fatal(err)
	}
	devPage, err := os.ReadFile(filepath.Join(root, "dev-site", "fr", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	devMarkup := string(devPage)
	for _, localOnly := range []string{`id="mpress-quick-edit-data"`, `data-quick-edit-bar`, `data-contribute-quick-edit`, `Fix this page`} {
		if !strings.Contains(devMarkup, localOnly) {
			t.Errorf("development contribution page is missing local quick editing %q", localOnly)
		}
	}
	for _, name := range []string{"mpress-contribute.sh", "mpress-contribute.ps1"} {
		contents, readErr := os.ReadFile(filepath.Join(root, "site", name))
		if readErr != nil || !strings.Contains(string(contents), "mpress") || !strings.Contains(string(contents), "https://github.com/example/docs.git") || !strings.Contains(string(contents), "next") {
			t.Errorf("installer %s was not generated: %v", name, readErr)
		}
	}
	script, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.js"))
	runtime := string(script)
	if err != nil || !strings.Contains(runtime, "data-contribute-command") || !strings.Contains(runtime, "navigator.userAgentData?.platform") || !strings.Contains(runtime, "| sh -s --") || !strings.Contains(runtime, "mpress-quick-edit:") || !strings.Contains(runtime, "Command copied") || !strings.Contains(runtime, "new Blob") || !strings.Contains(runtime, ".mpress-draft") || !strings.Contains(runtime, "downloadQuickEditDraft") {
		t.Fatalf("contribution command runtime was not generated: %v", err)
	}
	if strings.Contains(runtime, "encodeQuickEditDraft") || strings.Contains(runtime, "btoa(") {
		t.Fatal("browser draft contents must not be embedded in the contribution command")
	}
	for _, want := range []string{"markdownForQuickEditNode", "safeQuickEditLink", "restoreQuickEditHTML", "placeQuickEditFormat", "restoreQuickEditSelection", "collectTextNodes", "node.textContent", "replaceChild", "event.clipboardData?.getData('text/plain')", "{b: 'bold', i: 'italic', k: 'link'}"} {
		if !strings.Contains(runtime, want) {
			t.Errorf("quick-edit formatting runtime is missing %q", want)
		}
	}
	styles, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.css"))
	hiddenRule := regexp.MustCompile(`\.mpress-quick-edit-bar\[hidden\]\s*\{\s*display\s*:\s*none\s*;?\s*\}`)
	if err != nil || !hiddenRule.Match(styles) {
		t.Fatalf("hidden quick-edit controls must stay hidden before activation: %v", err)
	}
	if strings.Contains(markup, `data-contribute-platform="`) || !strings.Contains(markup, "data-contribute-platform-label") || strings.Count(markup, "data-contribute-copy") != 1 {
		t.Fatalf("contribution dialog should show one detected platform and one copy action")
	}
	for _, want := range []string{"Edit this page locally", "What this command does", "View the macOS and Linux script", "View the Windows script", "Already installed?", "data-contribute-manual"} {
		if !strings.Contains(markup, want) {
			t.Errorf("contribution handoff is missing %q", want)
		}
	}
}

func TestBuildOmitsContributionHandoffWhenDisabled(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  title: Private docs\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n")
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "data-mpress-contribute") {
		t.Fatal("disabled contribution action was rendered")
	}
	if _, err := os.Stat(filepath.Join(root, "site", "mpress-contribute.sh")); !os.IsNotExist(err) {
		t.Fatalf("disabled contribution installer exists: %v", err)
	}
}

func TestContributionBuildSupportsNativeMPDWithoutQuickEditMetadata(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Native contributor docs
build:
  contentDir: content
  staticDir: static
  outputDir: site
contribution:
  enabled: true
  repository: https://github.com/example/docs.git
`)
	writeFixture(t, root, "content/index.mpd", "---\nschema = 1\ntitle = \"Home\"\n---\n\n# Home\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("native MPD contribution build failed: %v", err)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(page)
	if !strings.Contains(markup, "data-mpress-contribute") {
		t.Fatal("native MPD page lost the contribution handoff")
	}
	if strings.Contains(markup, "data-quick-edit-bar") {
		t.Fatal("native MPD page exposed quick edit without safe source ranges")
	}
}

func TestBuildOmitsAccessibilityAssetsWhenDisabled(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Minimal docs
accessibility:
  enabled: false
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true, MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"index.html", filepath.Join("assets", "mpress.css"), filepath.Join("assets", "mpress.js")} {
		data, err := os.ReadFile(filepath.Join(root, "site", path))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "mpress-accessibility") || strings.Contains(string(data), "data-a11y-") {
			t.Fatalf("disabled accessibility feature remains in %s: %s", path, data)
		}
	}
}

func TestBuildCollectorRegistersRenderedAndStaticFilesWithoutOutputRescan(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Indexed site
build:
  contentDir: content
  staticDir: static
  outputDir: site
search:
  enabled: true
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n\n[Static page](/static.html#ready)\n\n![Asset](/assets/app.js)\n")
	writeFixture(t, root, "static/static.html", `<h1 id="ready">Ready</h1>`)
	writeFixture(t, root, "static/assets/app.js", `console.log("ready")`)

	collector := check.NewCollector()
	result, err := Build(root, BuildOptions{Strict: true, LinkCollector: collector})
	if err != nil {
		t.Fatal(err)
	}
	if result.Files != countFiles(filepath.Join(root, "site")) {
		t.Fatalf("registered file count %d does not match output", result.Files)
	}
	if got := collector.Finalize(); len(got) != 0 {
		t.Fatalf("incremental collector found broken links: %#v", got)
	}
	if got, err := check.Run(filepath.Join(root, "site")); err != nil || len(got) != 0 {
		t.Fatalf("standalone check disagreed: %v, %#v", err, got)
	}
}

func TestBuildOpensCurrentNavAncestorsAndRendersNestedTOC(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Test Docs
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nWelcome.\n")
	writeFixture(t, root, "content/guides/cli/install.md", "---\ntitle: Install\n---\n\n## Configure\n\n### Choose a target\n")
	writeFixture(t, root, "content/_nav.yaml", `- label: Guides
  collapsed: true
  items:
    - label: CLI
      collapsed: true
      items:
        - label: Install
          link: /guides/cli/install/
`)
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "guides", "cli", "install", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(page)
	if strings.Count(markup, "<details open>") < 2 {
		t.Fatalf("current navigation ancestors were not expanded: %s", markup)
	}
	for _, want := range []string{`aria-current="page"`, `class="toc-level-2" href="#configure"`, `class="toc-level-3" href="#choose-a-target"`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("generated page missing %q: %s", want, markup)
		}
	}
}

func TestBuildRefusesOutputOutsideProject(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  title: Safe\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: "+outside+"\n")
	writeFixture(t, root, "content/index.md", "# Safe\n")
	writeFixture(t, outside, "sentinel.txt", "keep")
	if _, err := Build(root, BuildOptions{}); err == nil || !strings.Contains(err.Error(), "unsafe output") {
		t.Fatalf("expected unsafe-output error, got %v", err)
	}
	assertExists(t, filepath.Join(outside, "sentinel.txt"))
}

func TestLandingLayoutKeepsHeaderAndReplacesDocumentationChrome(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Product Docs
  logoLight: logo-light.svg
  logoDark: logo-dark.svg
  headerLinks:
    - label: Components
      url: /components/
    - label: Support
      url: /support/
      type: button
      variant: outline
social:
  github: https://github.com/example/docs
  discord: https://discord.example/docs
  x: https://x.example/docs
contribution:
  enabled: true
  repository: https://github.com/example/docs
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
versioning:
  enabled: true
  current: Next
`)
	writeFixture(t, root, "content/index.md", `---
title: Product home
layout: landing
---
@section{variant=hero|class=product-hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Build clearly.
Ship confidently.
@end
@actions
@button[Read the guide](/guide/){primary}
@end
@end
@column{variant=story-visual|class="product-shot product-shot-reversed"}
@image{light="/logo-light.svg" dark="/logo-dark.svg" alt="Product documentation"}
@end
@end
@end
`)
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\nGuide content.\n")
	writeFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n- label: Guide\n  link: /guide/\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	home, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(home)
	markup = strings.Replace(markup, `href="./support/"`, `href="/support/"`, 1)
	for _, want := range []string{`class="landing-page"`, `data-shortcut-search="Mod+K"`, `data-shortcut-accessibility="Mod+A"`, `<header>`, `class="landing-main"`, `product-hero`, `mpress-section-hero`, `mpress-columns-hero`, `mpress-headline`, `Build clearly.<br>Ship confidently.`, `mpress-button-primary`, `class="theme-logo-light"`, `class="theme-logo-dark"`, `class="primary-links"`, `>Components</a>`, `class="header-link-button header-link-button-outline" href="/support/">Support</a>`, `aria-label="GitHub"`, `aria-label="Discord"`, `aria-label="X"`, `aria-label="Select language: en"`, `lucide-languages`, `id="mpress-language-menu"`, `class="header-utility-cluster"`, `header-contribute`, `aria-label="Accessibility settings"`, `id="mpress-language-menu"`, `aria-current="true"`, `class="search-shortcut" aria-hidden="true">Ctrl K</kbd>`, `id="theme" class="header-group theme-toggle"`, `aria-label="Theme: System. Switch to Dark"`, `lucide-monitor`, `lucide-moon`, `lucide-sun`} {
		if !strings.Contains(markup, want) {
			t.Errorf("landing page missing %q: %s", want, markup)
		}
	}
	if strings.Contains(markup, "@section") || strings.Contains(markup, "@end") {
		t.Fatalf("native landing directives leaked into generated HTML: %s", markup)
	}
	if strings.Contains(markup, `<select id="theme"`) || strings.Contains(markup, `>Auto</option>`) {
		t.Error("landing page still renders the old theme selector")
	}
	socialIndex := strings.Index(markup, `class="header-group social-links"`)
	languageIndex := strings.Index(markup, `class="header-group utility-select utility-menu language-select"`)
	themeIndex := strings.Index(markup, `id="theme" class="header-group theme-toggle"`)
	if socialIndex < 0 || languageIndex < 0 || themeIndex < 0 || socialIndex > languageIndex || languageIndex > themeIndex {
		t.Errorf("language and theme controls are not in the expected order: %s", markup)
	}
	clusterStart := strings.Index(markup, `class="header-utility-cluster"`)
	contributionIndex := strings.Index(markup, `class="header-group contribution-header"`)
	accessibilityIndex := strings.Index(markup, `class="header-group utility-select accessibility-select"`)
	clusterEndOffset := -1
	if clusterStart >= 0 {
		clusterEndOffset = strings.Index(markup[clusterStart:], `</button></span></nav>`)
	}
	clusterEnd := clusterStart + clusterEndOffset
	if clusterStart < 0 || contributionIndex < clusterStart || accessibilityIndex < contributionIndex || languageIndex < accessibilityIndex || themeIndex < languageIndex || clusterEndOffset < 0 || themeIndex > clusterEnd {
		t.Errorf("header preferences are not grouped in the expected order: %s", markup)
	}
	for _, want := range []string{`starlight-icon-github`, `starlight-icon-discord`, `starlight-icon-x-com`, `fill="currentColor"`} {
		if !strings.Contains(markup, want) {
			t.Errorf("landing page missing Starlight-compatible social icon %q: %s", want, markup)
		}
	}
	for _, unwanted := range []string{`class="sidebar"`, `class="toc"`, `<article><h1>Product home</h1>`, `class="pager"`, `aria-label="Select version"`, `id="mpress-version-menu"`} {
		if strings.Contains(markup, unwanted) {
			t.Errorf("landing page contains documentation chrome %q", unwanted)
		}
	}
	guide, err := os.ReadFile(filepath.Join(root, "site", "guide", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(guide), `class="sidebar"`) || !strings.Contains(string(guide), `<article><header class="docs-page-header"><h1>Guide</h1></header>`) {
		t.Fatalf("standard page lost documentation chrome: %s", guide)
	}
}

func TestNavbarUsesFullHeightGroupSeparators(t *testing.T) {
	for _, want := range []string{
		`.header-group + .header-group::before { content: ""; position: absolute; left: 0; top: 50%; width: 1px; height: 32px;`,
		`.header-utility-cluster::before { content: ""; position: absolute; left: 0; top: 50%; width: 1px; height: 32px;`,
		`.header-utility-cluster > .header-group::before { display: none; }`,
		`.header-contribute { width: 34px; min-width: 34px; height: 34px;`,
		`.header-links .header-utility-cluster > :is(.accessibility-select, .language-select) .utility-menu-trigger { width: 34px; min-width: 34px; height: 34px;`,
		`.header-group.theme-toggle { width: 34px; min-width: 34px; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Fatalf("default navbar separator rule is missing %q", want)
		}
	}
}

func TestDefaultThemeKeepsTheProjectEditorNavigationCompact(t *testing.T) {
	for _, want := range []string{
		`body.mpress-workspace-document .layout { grid-template-columns: minmax(14.5rem, 16rem) minmax(0, 1fr); }`,
		`body.mpress-workspace-document .mpress-workspace-group-toggle { appearance: none; display: flex; width: 100%;`,
		`body.mpress-workspace-document .mpress-workspace-group-toggle[aria-expanded="false"] .lucide { transform: rotate(-90deg); }`,
		`body.mpress-workspace-document .mpress-workspace-group-links { display: grid;`,
		`body.mpress-workspace-document .mpress-workspace-sidebar .mpress-workspace-link.active { background: var(--accent); color: #fff;`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Fatalf("compact project editor navigation rule is missing %q", want)
		}
	}
}

func TestDefaultThemeKeepsLanguageSelectionAvailableOnMobile(t *testing.T) {
	if !strings.Contains(defaultThemeCSS, `.utility-select:not(.accessibility-select):not(.language-select), .social-links { display: none; }`) {
		t.Fatal("mobile theme hides the language selector")
	}
}

func TestBuildGeneratesBlogArchiveWithNewestPostAsHero(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Product Journal
  description: Project news
social:
  rss: /blog/rss.xml
blog:
  landingStyle: featured
  tagline: Product updates from the Wails team.
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nHome.\n")
	writeFixture(t, root, "content/blog/index.md", "---\ntitle: Blog\ndescription: Releases, stories, and engineering notes.\n---\n\nThis source body is replaced by the generated archive.\n")
	writeFixture(t, root, "content/blog/older.md", `---
title: The first release
description: The project takes its first public step.
date: 2025-01-10
author: Ada
tags: [release]
image: /images/older.png
---

The older article body.
`)
	writeFixture(t, root, "content/blog/latest.md", `---
title: A new foundation
description: The next generation is ready to try.
date: 2026-08-02
authors: [Lea Anthony, The Wails Team]
tags: [release, engineering]
image: /images/latest.png
imageFit: contain
imageBackground: "#123456"
imageWidth: 82
showTags: false
headingSize: large
---

The latest article body contains enough words to calculate its reading time.
`)
	writeFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n- label: Blog\n  link: /blog/\n")
	writeFixture(t, root, "static/images/latest.png", "latest")
	writeFixture(t, root, "static/images/older.png", "older")

	result, err := Build(root, BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("build blog archive: %v; diagnostics: %#v", err, result.Diagnostics)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "blog", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(page)
	for _, want := range []string{
		`class="blog-index-page"`, `class="blog-layout" data-blog-landing-style="featured"`, `Product updates from the Wails team.`, `class="blog-hero"`, `A new foundation`,
		`src="../images/latest.png"`, `2 August 2026`, `Lea Anthony, The Wails Team`,
		`class="blog-grid"`, `The first release`, `data-blog-filter="engineering"`,
		`href="../blog/rss.xml"`, `class="blog-hero-label-row"`, `data-blog-style-editor`, `data-blog-tags-visible="false"`, `data-blog-heading-size="large"`, `data-blog-image-editor data-blog-image-mode="panel" data-blog-image-fit="contain" data-blog-image-width="82" data-blog-image-background="#123456" style="--blog-image-background:#123456;--blog-floating-image-width:82%"`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("generated blog archive missing %q: %s", want, markup)
		}
	}
	if strings.Index(markup, "A new foundation") > strings.Index(markup, "The first release") {
		t.Fatalf("newest post was not rendered before the older post: %s", markup)
	}
	if strings.Contains(markup, "This source body is replaced") {
		t.Fatalf("blog index rendered its placeholder Markdown instead of the generated archive: %s", markup)
	}
	if strings.Contains(markup, "mpress-dev-image-edit") {
		t.Fatal("development image controls leaked into the static build")
	}
	articlePage, err := os.ReadFile(filepath.Join(root, "site", "blog", "latest", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	articleMarkup := string(articlePage)
	for _, want := range []string{
		`class="blog-article-page"`, `class="blog-layout blog-article-layout"`,
		`property="og:type" content="article"`,
		`class="blog-sidebar" aria-label="Blog archive"`, `class="blog-archive-home"`,
		`<span class="blog-sidebar-label">2026</span>`, `<span class="blog-sidebar-label">2025</span>`,
		`class="blog-article-link active" aria-current="page"`, `A new foundation`, `The first release`,
		`class="blog-article-main"`, `class="blog-article-body"`, `The latest article body`,
	} {
		if !strings.Contains(articleMarkup, want) {
			t.Errorf("generated blog article missing %q: %s", want, articleMarkup)
		}
	}
	if strings.Contains(articleMarkup, `id="mpress-sidebar"`) || strings.Contains(articleMarkup, `class="toc"`) || strings.Contains(articleMarkup, `class="layout"`) {
		t.Fatalf("blog article rendered documentation navigation or table of contents: %s", articleMarkup)
	}
	if strings.Index(articleMarkup, `<span class="blog-sidebar-label">2026</span>`) > strings.Index(articleMarkup, `<span class="blog-sidebar-label">2025</span>`) {
		t.Fatalf("blog article archive is not newest first: %s", articleMarkup)
	}
	gridConfig := strings.ReplaceAll(`site:
  title: Product Journal
blog:
  landingStyle: grid
  tagline: Every article has equal weight.
build:
  contentDir: content
  staticDir: static
  outputDir: site-grid
  navFile: _nav.yaml
`, "\r\n", "\n")
	writeFixture(t, root, "mpress.yaml", gridConfig)
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("build grid blog archive: %v", err)
	}
	gridPage, err := os.ReadFile(filepath.Join(root, "site-grid", "blog", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	gridMarkup := string(gridPage)
	if !strings.Contains(gridMarkup, `data-blog-landing-style="grid"`) || !strings.Contains(gridMarkup, "Every article has equal weight.") || strings.Contains(gridMarkup, `class="blog-hero"`) || strings.Count(gridMarkup, `class="blog-card"`) != 2 {
		t.Fatalf("grid blog style did not render every post as a card: %s", gridMarkup)
	}
	theme, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{".blog-hero", ".blog-grid", `.blog-layout[data-blog-landing-style="list"] .blog-grid`, ".blog-filter.active { background: var(--accent-fill); color: var(--accent-fill-text);", ".blog-article-main", ".blog-article-link.active { background: var(--accent-fill); color: var(--accent-fill-text);", ":is(.blog-index-page, .blog-article-page) #menu", `.blog-hero[data-blog-image-fit="contain"]`, `.blog-hero[data-blog-image-mode="floating"]`, `--blog-image-default-background: #fff`, `--blog-image-default-background: #000`, `[data-blog-tags-visible="false"]`, `[data-blog-heading-size="large"]`, `var(--blog-image-background`, `var(--blog-floating-image-width, 100%)`} {
		if !strings.Contains(string(theme), want) {
			t.Errorf("generated theme missing blog rule %q", want)
		}
	}
	script, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "const blogFilters") || !strings.Contains(string(script), "post.hidden") {
		t.Fatal("generated theme is missing blog topic filtering")
	}
}

func TestDefaultThemeExplainedComponentUsesAccessibleFloatingAnnotations(t *testing.T) {
	for _, want := range []string{
		`.mpress-explained-line[data-ref]::after`,
		`.mpress-explained-tooltip`,
		`.mpress-explained-enhanced .mpress-explained-prose`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default theme CSS is missing explained annotation rule %q", want)
		}
	}
	for _, want := range []string{
		`tooltip.role = 'tooltip'`,
		`line.setAttribute('aria-describedby', descriptionID)`,
		`line.dataset.explanation?.trim()`,
		`fallback.querySelector('.mpress-explained-ref')?.remove()`,
		`line.addEventListener('mousemove'`,
		`line.addEventListener('focus'`,
		`line.addEventListener('click'`,
		`event.key === 'Escape'`,
	} {
		if !strings.Contains(defaultThemeJS, want) {
			t.Errorf("default theme JavaScript is missing explained annotation behaviour %q", want)
		}
	}
}

func TestDefaultThemeSupportsInteractiveTables(t *testing.T) {
	for _, want := range []string{
		`.mpress-table-toolbar`,
		`.js .mpress-table-toolbar`,
		`.mpress-table-search input:focus`,
		`[data-table-sort-column]`,
		`th[aria-sort="ascending"]`,
		`.mpress-table-filter-trigger.utility-menu-trigger`,
		`.mpress-data-table th:not(:last-child) .mpress-table-filter-trigger`,
		`.mpress-table-column-separators th:not(:last-child), .mpress-table-column-separators td:not(:last-child)`,
		`.mpress-table-filter-menu.utility-menu-panel button[aria-current="true"]`,
		`.mpress-table-status`,
		`.mpress-table-pagination`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default theme CSS is missing table rule %q", want)
		}
	}
	for _, want := range []string{
		`new Intl.Collator`,
		`document.querySelectorAll('.mpress-data-table')`,
		`query?.addEventListener('input', () =>`,
		`filter.querySelectorAll('[data-table-filter-value]')`,
		`heading?.setAttribute('aria-sort', sortDirection)`,
		`rows.forEach(row => row.hidden = !visible.has(row))`,
		`root.dataset.tablePaginate === 'true'`,
		`previousPage?.addEventListener('click'`,
	} {
		if !strings.Contains(defaultThemeJS, want) {
			t.Errorf("default theme JavaScript is missing table behaviour %q", want)
		}
	}
}

func TestDefaultThemeSupportsResponsiveNativeVideo(t *testing.T) {
	for _, want := range []string{
		`.mpress-video`,
		`.mpress-video video`,
		`width: 100%`,
		`height: auto`,
		`.mpress-video figcaption`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default theme CSS is missing video rule %q", want)
		}
	}
}

func TestDefaultThemeStylesChangelogAsReleaseFeed(t *testing.T) {
	for _, want := range []string{
		`.mpress-timeline-entry::before`,
		`.mpress-timeline-entry:last-child::before`,
		`.mpress-timeline-marker`,
		`.mpress-changelog`,
		`.mpress-changelog-entry:last-child`,
		`.mpress-changelog-body { margin-left: 2.9rem; }`,
		`.mp-home-timeline`,
		`.mpress-changelog-header`,
		`.mpress-changelog-section`,
		`.mpress-changelog-items li::before`,
		`.mpress-changelog-category-visually-hidden`,
		`.mpress-badge-feature`,
		`.mpress-badge-breaking`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default theme CSS is missing changelog rule %q", want)
		}
	}
}

func TestBuildPreservesProjectRobotsAndNotFoundPages(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Custom system pages
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nHome.\n")
	writeFixture(t, root, "static/robots.txt", "User-agent: ExampleBot\nDisallow: /\n")
	writeFixture(t, root, "static/404.html", "<!doctype html><title>Custom missing page</title>")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	robots, err := os.ReadFile(filepath.Join(root, "site", "robots.txt"))
	if err != nil || string(robots) != "User-agent: ExampleBot\nDisallow: /\n" {
		t.Fatalf("custom robots.txt was replaced: %v, %q", err, robots)
	}
	notFound, err := os.ReadFile(filepath.Join(root, "site", "404.html"))
	if err != nil || !strings.Contains(string(notFound), "Custom missing page") {
		t.Fatalf("custom 404 page was replaced: %v, %q", err, notFound)
	}
}

func TestBuildUsesAnInMemorySourceOverrideInAnIsolatedOutput(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  title: Preview Docs\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n")
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nOriginal saved content.\n")
	writeFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	previewDir := filepath.Join(root, ".mpress", "live-preview", "site")
	result, err := Build(root, BuildOptions{
		Strict: true, IncludeDrafts: true, OutputDir: previewDir,
		SourceOverrides: map[string]string{"index.md": "---\ntitle: Home\n---\n\nUnsaved live preview content.\n"},
	})
	if err != nil {
		t.Fatalf("preview build: %v; diagnostics: %#v", err, result.Diagnostics)
	}
	preview, err := os.ReadFile(filepath.Join(previewDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(preview), "Unsaved live preview content") || strings.Contains(string(preview), "Original saved content") {
		t.Fatalf("preview output did not use the source override: %s", preview)
	}
	source, err := os.ReadFile(filepath.Join(root, "content", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "Original saved content") || strings.Contains(string(source), "Unsaved live preview content") {
		t.Fatalf("preview build changed the saved source: %s", source)
	}
	if _, err := os.Stat(filepath.Join(root, "site", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("preview build wrote to the production output: %v", err)
	}
}

func TestCapturedVersionVerifiesAndMountsAsAWorkingSubsite(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Versioned Docs
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
versioning:
  enabled: true
  current: Next
  artifactsDir: .mpress/versions
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n[Guide](/guide/)\n")
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\nVersion one. [Home](/)\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(current), `aria-label="Select version"`) {
		t.Fatal("version selector rendered before any historical version was captured")
	}
	if err := docversion.Capture(root, "1.0", false); err != nil {
		t.Fatal(err)
	}
	if err := docversion.Verify(root, "1.0"); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\nNext version.\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	current, err = os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`aria-label="Select version"`, `id="mpress-version-menu"`, `>Next</span>`, `>1.0</span>`} {
		if !strings.Contains(string(current), want) {
			t.Fatalf("version selector missing %q after capturing a historical version", want)
		}
	}
	markup := string(current)
	clusterStart := strings.Index(markup, `class="header-utility-cluster"`)
	versionIndex := strings.Index(markup, `class="header-group utility-select utility-menu version-select"`)
	clusterEndOffset := -1
	if clusterStart >= 0 {
		clusterEndOffset = strings.Index(markup[clusterStart:], `</button></span></nav>`)
	}
	if clusterStart < 0 || versionIndex < clusterStart || clusterEndOffset < 0 || versionIndex > clusterStart+clusterEndOffset {
		t.Fatalf("version selector is not part of the site preferences group: %s", markup)
	}
	assertExists(t, filepath.Join(root, "site", "versions", "1.0", "assets", "mpress.css"))
	historical, err := os.ReadFile(filepath.Join(root, "site", "versions", "1.0", "guide", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	historicalMarkup := string(historical)
	for _, want := range []string{
		`class="utility-menu-current">1.0</span>`,
		`href="/versions/1.0/guide/" role="menuitem" aria-current="true"`,
		`href="/guide/" role="menuitem"`,
		`href="/versions/1.0/">Home</a>`,
	} {
		if !strings.Contains(historicalMarkup, want) {
			t.Fatalf("historical version is missing %q: %s", want, historicalMarkup)
		}
	}
	if broken, err := check.Run(filepath.Join(root, "site")); err != nil || len(broken) != 0 {
		t.Fatalf("mounted version link check: %v, %#v", err, broken)
	}
}

func writeFixture(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}
