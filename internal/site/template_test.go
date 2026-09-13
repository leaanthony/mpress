package site

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/navigation"
)

func TestTypedRendererEscapesDataAndPreservesRenderedMarkdown(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Title = `Docs <unsafe>`
	cfg.Social.GitHub = "javascript:alert(1)"
	page := &content.Page{
		Language:    `en" onclick="alert(1)`,
		URLPath:     "guide",
		Title:       `<script>alert(1)</script>`,
		Description: `" onfocus="alert(1)`,
		HTML:        `<p data-rendered="markdown">Safe rendered body</p>`,
	}
	markup, err := renderPage(templateData{
		Config:        cfg,
		Page:          page,
		Nav:           []navigation.Item{{Label: `<Guide>`, Link: "/guide/"}},
		Root:          "../",
		SearchURL:     "../search-index.json",
		CSSVersion:    "css123",
		JSVersion:     "js456",
		CurrentRoutes: map[string]*content.Page{"guide": page},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	for _, want := range []string{
		`lang="en&#34; onclick=&#34;alert(1)"`,
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`content="&#34; onfocus=&#34;alert(1)"`,
		`href="#ZgotmplZ"`,
		`&lt;Guide&gt;`,
		`<p data-rendered="markdown">Safe rendered body</p>`,
		`<a class="mpress-footer-credit" href="https://m-press.me">Built with <strong>M-Press</strong></a>`,
		`href="../assets/mpress.css?v=css123"`,
		`src="../assets/mpress.js?v=js456"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered page missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, `<script>alert(1)</script>`) {
		t.Fatalf("untrusted title was emitted as HTML: %s", got)
	}
}

func TestRTLLanguagesSetDocumentDirection(t *testing.T) {
	for _, language := range []string{"ar", "fa-IR", "he", "ur_PK"} {
		if !isRTLLanguage(language) {
			t.Errorf("%q should use right-to-left direction", language)
		}
	}
	for _, language := range []string{"en", "de", "fr", "ja", "zh-CN"} {
		if isRTLLanguage(language) {
			t.Errorf("%q should not use right-to-left direction", language)
		}
	}

	cfg := config.Default()
	page := &content.Page{Language: "ar", Title: "دليل", HTML: `<p>مرحبا</p>`}
	markup, err := renderPage(templateData{Config: cfg, Page: page, SearchURL: "search-index.json"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markup), `<html lang="ar" dir="rtl"`) {
		t.Fatalf("Arabic page does not declare its direction: %s", markup)
	}
}

func TestPageEditSourcePrefersOnlySafeImportedPaths(t *testing.T) {
	page := &content.Page{SourcePath: "guide.mpd", Meta: content.Frontmatter{SourcePath: "guide.mdx"}}
	if got, want := pageEditSource(page), "guide.mdx"; got != want {
		t.Fatalf("pageEditSource() = %q, want %q", got, want)
	}
	for _, unsafe := range []string{"../outside.md", "/absolute.md", ".."} {
		page.Meta.SourcePath = unsafe
		if got, want := pageEditSource(page), "guide.mpd"; got != want {
			t.Errorf("pageEditSource() with %q = %q, want fallback %q", unsafe, got, want)
		}
	}
}

func TestDefaultTOCNeverScrollsHorizontally(t *testing.T) {
	for _, want := range []string{
		`.toc { display: flex; min-width: 0;`,
		`overflow-x: hidden; border-left: 1px solid var(--border);`,
		`.toc a { max-width: calc(100% + 1.23rem);`,
		`overflow-wrap: anywhere;`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default TOC overflow protection is missing %q", want)
		}
	}
}

func TestAccessibilityPanelUsesTallerResponsiveBody(t *testing.T) {
	for _, want := range []string{
		`display: grid;`,
		`max-height: min(820px, calc(100dvh - 1rem));`,
		`grid-template-rows: auto auto minmax(0, 1fr) auto;`,
		`.mpress-accessibility-body { min-height: 400px; overflow-y: auto; overscroll-behavior: contain; }`,
	} {
		if !strings.Contains(accessibilityCSS, want) {
			t.Errorf("accessibility panel CSS is missing %q", want)
		}
	}
}

func TestStepTitlesAlignWithNumberMarkers(t *testing.T) {
	want := `.mpress-step-title { display: flex; min-height: 2.1rem; align-items: center; margin: 0 0 .25rem;`
	if !strings.Contains(defaultThemeCSS, want) {
		t.Errorf("default step title alignment is missing %q", want)
	}
}

func TestFileTreeUsesBranchConnectorsInsteadOfBullets(t *testing.T) {
	for _, want := range []string{
		`.mpress-filetree ul { margin: 0; padding: 0; list-style: none; }`,
		`.mpress-filetree > ul { padding-left: .9rem; }`,
		`.mpress-filetree ul ul { padding-left: 1.6rem; }`,
		`.mpress-filetree li > ul::before { content: ""; position: absolute; top: -.8rem; left: .9rem; height: .8rem;`,
		`.mpress-filetree ul ul > li::before { content: "";`,
		`border-left: 1px solid var(--border);`,
		`.mpress-filetree ul ul > li::after { content: ""; position: absolute; top: .8rem; left: -.7rem; width: 1.15rem;`,
		`border-top: 1px solid var(--border);`,
		`.mpress-filetree ul ul > li:last-child::before { bottom: auto; height: .8rem; }`,
		`.mpress-filetree-row { position: relative; z-index: 1; display: flex; align-items: center;`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default file tree connector CSS is missing %q", want)
		}
	}
}

func TestPricingCardsCenterContentAndUseFullWidthActions(t *testing.T) {
	for _, want := range []string{
		`.mpress-pricing-card { position: relative; padding: 1.2rem; border: 1px solid var(--border); border-radius: 9px; background: var(--surface); text-align: center; }`,
		`.mpress-pricing-badge { position: absolute; top: -.65rem; left: 50%;`,
		`transform: translateX(-50%);`,
		`.mpress-pricing-cta { display: flex; width: 100%;`,
		`justify-content: center;`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default pricing alignment CSS is missing %q", want)
		}
	}
}

func TestTutorialCheckboxUsesCheckmarkInsteadOfSolidFill(t *testing.T) {
	for _, want := range []string{
		`.mpress-tutorial-check input:checked + .mpress-tutorial-checkmark { border-color: var(--accent-2); background: transparent; }`,
		`color: var(--accent-2); stroke-width: 3;`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default tutorial checkbox CSS is missing %q", want)
		}
	}
}

func TestTutorialLessonNumberMatchesTheTitleScale(t *testing.T) {
	want := `.mpress-tutorial-step-num { min-width: 1.15rem; color: var(--muted); font-size: .875rem; font-variant-numeric: tabular-nums;`
	if !strings.Contains(defaultThemeCSS, want) {
		t.Errorf("default tutorial lesson number CSS is missing %q", want)
	}
}

func TestTerminalHasLightAndDarkThemeTokens(t *testing.T) {
	for _, want := range []string{
		`--terminal-bg: #fbfcfe;`,
		`--terminal-title-bg: #eef1f5;`,
		`--terminal-bg: #1f2026;`,
		`.mpress-terminal { margin: 1rem 0; overflow: hidden; border: 1px solid var(--terminal-border);`,
		`background: var(--terminal-bg); color: var(--terminal-text);`,
		`.mpress-terminal-title { position: relative;`,
		`background: var(--terminal-title-bg); color: var(--terminal-output);`,
		`.mpress-prompt { color: var(--terminal-prompt); user-select: none; }`,
		`.mpress-cmd { color: var(--terminal-command); }`,
		`.mpress-comment { color: var(--terminal-comment); font-style: italic; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default terminal theme CSS is missing %q", want)
		}
	}
}

func TestCodeFramesCanDisplayLineNumbers(t *testing.T) {
	for _, want := range []string{
		`.mpress-codeframe-line-numbers code { counter-reset: mpress-code-line; }`,
		`.mpress-codeframe-line-numbers .mpress-code-line { position: relative; padding-left: 4rem; }`,
		`content: counter(mpress-code-line); counter-increment: mpress-code-line;`,
		`font-variant-numeric: tabular-nums;`,
		`.mpress-codeframe-line-numbers .mpress-code-line-marked::before`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default code line number CSS is missing %q", want)
		}
	}
}

func TestGeneratedSelectsUseCustomDropdownListboxes(t *testing.T) {
	for _, want := range []string{
		`.mpress-ddlb-native { position: absolute !important;`,
		`.mpress-ddlb-trigger { position: relative; display: flex; width: 100%;`,
		`.mpress-ddlb-menu[popover] { position: fixed;`,
		`.mpress-ddlb-option[aria-selected="true"]`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default custom DDLB CSS is missing %q", want)
		}
	}
	for _, want := range []string{
		`select:not([multiple]):not([data-mpress-ddlb-ready])`,
		`trigger.setAttribute('role', 'combobox')`,
		`menu.setAttribute('role', 'listbox')`,
		`item.setAttribute('role', 'option')`,
		`select.dispatchEvent(new Event('change', {bubbles: true}))`,
		`const ddlbObserver = new MutationObserver`,
	} {
		if !strings.Contains(defaultThemeJS, want) {
			t.Errorf("default custom DDLB runtime is missing %q", want)
		}
	}
}

func TestAPIPlaygroundDescriptionHasBalancedVerticalSpacing(t *testing.T) {
	for _, want := range []string{
		`.mpress-api-description { padding: 1rem; }`,
		`.mpress-api-description > :first-child { margin-top: 0; }`,
		`.mpress-api-description > :last-child { margin-bottom: 0; }`,
		`.mpress-api-playground .mpress-api-description { padding-bottom: 0; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default API description spacing CSS is missing %q", want)
		}
	}
}

func TestDetailsContentAndNestedNotesHaveBalancedSpacing(t *testing.T) {
	for _, want := range []string{
		`.mpress-details-content { padding: 1rem; }`,
		`.mpress-details-content > :first-child { margin-top: 0; }`,
		`.mpress-details-content > :last-child { margin-bottom: 0; }`,
		`.mpress-details-content > .mpress-admonition { margin: 1rem 0 0; }`,
		`.mpress-details-content > .mpress-admonition:first-child { margin-top: 0; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default Details spacing CSS is missing %q", want)
		}
	}
}

func TestStepComponentsAndAudienceNotesHaveClearTopSpacing(t *testing.T) {
	for _, want := range []string{
		`.mpress-step-content { padding-top: .75rem; }`,
		`.mpress-step-content > :first-child { margin-top: 0; }`,
		`.mpress-audience > .mpress-admonition { margin: 2rem 0; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default nested component spacing CSS is missing %q", want)
		}
	}
}

func TestDefaultThemeDefinesCompleteHeadingScale(t *testing.T) {
	for _, want := range []string{
		`article h1 { max-width: 28ch; margin: 0 0 .7rem; font-size: 38px;`,
		`article h2 { margin: 3rem 0 .85rem; scroll-margin-top: 88px; font-size: 26px;`,
		`article h3 { margin: 2.25rem 0 .7rem; scroll-margin-top: 88px; font-size: 21px;`,
		`article h4 { margin: 2rem 0 .6rem; scroll-margin-top: 88px; font-size: 18px;`,
		`article h5 { margin: 1.75rem 0 .55rem; scroll-margin-top: 88px; font-size: 16px;`,
		`article h6 { margin: 1.5rem 0 .5rem; scroll-margin-top: 88px; font-size: 14px;`,
		`article h1 + h2, article h2 + h3, article h3 + h4, article h4 + h5, article h5 + h6 { margin-top: 1.5rem; }`,
		`.landing-main > h1, .landing-main > h2, .landing-main > h3, .landing-main > p { margin-top: 0; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("default heading scale CSS is missing %q", want)
		}
	}
}

func TestTerminalCommentsHaveAVisibleNonCommandStyle(t *testing.T) {
	want := `.mpress-comment { color: var(--terminal-comment); font-style: italic; }`
	if !strings.Contains(defaultThemeCSS, want) {
		t.Errorf("default terminal comment CSS is missing %q", want)
	}
}

func TestGeneratedCodeBlocksHaveLightAndDarkThemeTokens(t *testing.T) {
	for _, want := range []string{
		`--code: #f4f6fa;`,
		`--code-text: #252d3a;`,
		`--code-comment: #627082;`,
		`--code: #060a12;`,
		`--code-text: #eef4ff;`,
		`.mpress-token-comment { color: var(--code-comment); font-style: italic; }`,
		`.mpress-token-function { color: var(--code-function); }`,
		`border: 1px solid var(--code-copy-border);`,
		`.mpress-codeframe-header { display: flex; min-height: 40px;`,
		`.mpress-codeframe pre { margin: 0; padding-block: .7rem .9rem; border: 0; border-radius: 0; }`,
		`.mpress-codeframe-language { flex: none; padding: 0 0 0 .7rem; border-left: 1px solid var(--border);`,
		`.mpress-codeframe-header > .mpress-copy { position: static; height: 28px; width: 28px; min-height: 28px; margin: 0 .4rem 0 auto; background: var(--surface); line-height: 1; transform: translateY(-1px); }`,
		`.mpress-copy.is-copied .lucide-check { display: block; }`,
		`.mpress-diff-title { padding: .5rem .75rem; border-bottom: 1px solid var(--border); background: var(--panel); color: var(--muted);`,
		`.mpress-diff-added { background: var(--code-added-bg); color: var(--code-added-text); }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("generated code theme CSS is missing %q", want)
		}
	}
}

func TestRichTablesKeepCellSpacingAndRowRulesOutsideArticles(t *testing.T) {
	for _, want := range []string{
		`.mpress-data-table th, .mpress-data-table td { padding: .7rem .85rem; border-bottom: 1px solid var(--border); text-align: left; }`,
		`.mpress-data-table th { background: var(--panel); font-size: .8rem; letter-spacing: .04em; text-transform: uppercase; }`,
		`.mpress-data-table tbody tr:last-child td { border-bottom: 0; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("rich table CSS is missing %q", want)
		}
	}
}

func TestLandingFileTabsKeepAStableViewport(t *testing.T) {
	for _, want := range []string{
		`.mp-home-stack-visual .mpress-file-tabs { display: grid; width: 100%; height: 440px; grid-template-rows: auto minmax(0, 1fr); }`,
		`.mp-home-stack-visual .mpress-file-tabs > [role="tabpanel"] { min-height: 0; height: 100%; overflow: auto; background: var(--code); }`,
		`.mp-home-stack-visual .mpress-file-tabs > [role="tabpanel"] > pre { box-sizing: border-box; min-height: 100%; }`,
		`.mp-home-stack-visual .mpress-file-tabs .mpress-image-expand img { width: 100%; height: 100%; object-fit: contain; object-position: center; }`,
		`.mp-home-stack-visual .mpress-file-tabs { height: 360px; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("landing file tabs CSS is missing %q", want)
		}
	}
}

func TestLandingComponentPreviewKeepsAStableViewport(t *testing.T) {
	for _, want := range []string{
		`.mp-home-story-components .mp-home-story { grid-template-columns: 1fr;`,
		`.mp-home-story-components .mp-home-story-copy { display: grid; max-width: none; grid-template-columns: minmax(280px, .82fr) minmax(0, 1.18fr);`,
		`.mp-home-components-visual .mpress-preview-tabs { display: grid; height: 540px; margin: 0; grid-template-rows: auto minmax(0, 1fr); }`,
		`.mp-home-components-visual .mp-home-demo-tabs { gap: .15rem 1rem; margin-top: 0; flex-wrap: wrap; }`,
		`.mp-home-components-visual .mp-home-demo-tabs [role="tab"] { flex: 0 0 auto; }`,
		`.mp-home-components-visual .mpress-preview-tabs > [role="tabpanel"] { min-height: 0; padding-top: 1rem; overflow: auto; }`,
		`.mp-home-components-visual .mpress-codeframe pre { max-height: 220px; overflow: auto; font-size: .72rem; line-height: 1.55; }`,
		`.mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1.25rem; align-content: start; align-items: start; }`,
		`.mp-home-story-components .mpress-preview-tabs > [role="tabpanel"]:not([hidden]) > .mpress-qr { margin-inline: auto; justify-self: center; }`,
		`.mp-home-story-components .mpress-preview-source pre code { white-space: pre-wrap; overflow-wrap: anywhere; }`,
		`.mp-home-story-components .mp-home-story-copy { display: contents; }`,
		`.mpress-preview-source-label { display: flex; height: 34px; align-items: center;`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("landing component preview CSS is missing %q", want)
		}
	}
}

func TestQRCodeUsesLightAndDarkThemeTokens(t *testing.T) {
	for _, want := range []string{
		`--qr-bg: #fff;`,
		`--qr-foreground: #171a21;`,
		`--qr-bg: #060a12;`,
		`--qr-foreground: #f0f2f5;`,
		`background: var(--qr-bg); text-align: center;`,
		`.mpress-qr svg > rect:first-child { fill: var(--qr-bg); }`,
		`.mpress-qr svg > rect:not(:first-child) { fill: var(--qr-foreground); }`,
		`.mpress-qr-label { color: var(--qr-foreground);`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("QR theme CSS is missing %q", want)
		}
	}
}

func TestDocumentationLayoutCSSSupportsFixedPercentageAndHiddenTOC(t *testing.T) {
	fixed := documentationLayoutCSS(config.Default().Theme.Layout)
	for _, want := range []string{
		`--layout-content-width: clamp(30rem, 50rem, 100rem)`,
		`--layout-stage-fill: max(clamp(.75rem, 2.5rem, 6rem), calc((100% - clamp(30rem, 50rem, 100rem) - clamp(12rem, 16rem, 25rem)) / 2))`,
		`grid-template-columns: var(--layout-stage-fill) minmax(0, clamp(30rem, 50rem, 100rem)) var(--layout-stage-fill) minmax(0, clamp(12rem, 16rem, 25rem))`,
		`width: 100%`,
		`@media (max-width: 1180px)`,
	} {
		if !strings.Contains(fixed, want) {
			t.Errorf("fixed layout CSS is missing %q", want)
		}
	}

	percentage := documentationLayoutCSS(config.LayoutConfig{
		ContentWidth: "75%", WideContentWidth: "90%", SidebarWidth: "300px",
		TOCWidth: "256px", ContentTOCGap: "40px", Alignment: "left", TOC: "right",
	})
	if !strings.Contains(percentage, `grid-template-columns: var(--layout-stage-fill) minmax(0, clamp(30rem, 75%, 100rem)) var(--layout-stage-fill) minmax(0, clamp(12rem, 256px, 25rem))`) ||
		!strings.Contains(percentage, `width: 100%`) {
		t.Fatalf("percentage layout CSS is incorrect: %s", percentage)
	}

	hidden := documentationLayoutCSS(config.LayoutConfig{
		ContentWidth: "720px", WideContentWidth: "1024px", SidebarWidth: "300px",
		TOCWidth: "256px", ContentTOCGap: "40px", Alignment: "cluster", TOC: "hidden",
	})
	if !strings.Contains(hidden, `.docs-stage .toc { display: none; }`) || !strings.Contains(hidden, `grid-template-columns: var(--layout-stage-fill) minmax(0, var(--layout-content-width)) minmax(var(--layout-stage-fill), 1fr)`) {
		t.Fatalf("hidden TOC layout CSS is incorrect: %s", hidden)
	}
}

func TestWideDocumentationPageUsesWideShell(t *testing.T) {
	cfg := config.Default()
	page := &content.Page{Language: "en", URLPath: "reference", Title: "Reference", Layout: "wide"}
	markup, err := renderPage(templateData{Config: cfg, Page: page, CurrentRoutes: map[string]*content.Page{"reference": page}})
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	for _, want := range []string{`class="docs-page docs-page-wide"`, `class="docs-stage"`, `<main id="content">`, `<aside class="toc">`} {
		if !strings.Contains(got, want) {
			t.Errorf("wide documentation page is missing %q", want)
		}
	}
}

func TestAccessibilityMenuIsEnabledByDefaultAndCanBeRemoved(t *testing.T) {
	cfg := config.Default()
	page := &content.Page{Language: "en", URLPath: "guide", Title: "Guide", HTML: `<p>Read this guide.</p>`}
	data := templateData{Config: cfg, Page: page, CurrentRoutes: map[string]*content.Page{"guide": page}}

	markup, err := renderPage(data)
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	for _, want := range []string{
		`id="accessibility"`, `mpress-accessibility-icon`, `viewBox="0 0 512 512"`, `aria-haspopup="dialog"`,
		`id="mpress-accessibility-panel"`, `data-utility-panel`, `Bionic reading`,
		`Focus mode`, `Reading guide`, `Reduce motion`, `High contrast`,
		`Red and green distinction`, `Saved only in this browser.`,
		`role="tablist"`, `data-a11y-tab="reading"`, `data-a11y-tab="focus"`,
		`data-a11y-tab="vision"`, `role="tabpanel"`, `role="listbox"`,
		`data-a11y-width`, `data-a11y-width-output`, `data-a11y-width-reset`, `data-a11y-width-mode="percent"`, `data-a11y-width-mode="fixed"`, `Reading column`, `Site layout`,
		`class="mpress-a11y-select-trigger"`, `aria-haspopup="listbox"`,
		`data-shortcut-search="Mod+K"`, `data-shortcut-accessibility="Mod+A"`,
		`localStorage.getItem('mpress-accessibility')`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("default accessibility UI is missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "fonts.googleapis.com") {
		t.Fatal("accessibility UI loads a remote font")
	}
	if strings.Contains(got, `<select data-a11y-select="colour">`) {
		t.Fatal("accessibility colour profile still uses a native select menu")
	}
	if strings.Contains(got, `lucide-wheelchair`) {
		t.Fatal("accessibility control still uses the wheelchair symbol")
	}
	if !strings.Contains(accessibilityJS, `button, [role="button"], a.mpress-button`) {
		t.Fatal("bionic reading must not rewrite interactive button labels")
	}
	if !strings.Contains(accessibilityCSS, `.mpress-accessibility-section h3 { margin: 0 0 .2rem; color: var(--text); font-size: 13px; font-weight: 700; letter-spacing: normal; text-transform: none; }`) {
		t.Fatal("accessibility section headings must use consistent title casing")
	}

	cfg.Accessibility.Enabled = false
	data.Config = cfg
	markup, err = renderPage(data)
	if err != nil {
		t.Fatal(err)
	}
	got = string(markup)
	for _, unwanted := range []string{`id="accessibility"`, `id="mpress-accessibility-panel"`, `mpress-accessibility`} {
		if strings.Contains(got, unwanted) {
			t.Errorf("disabled accessibility UI still contains %q: %s", unwanted, got)
		}
	}
}

func TestSearchPresentationUsesConfigurationAndCanBeRemoved(t *testing.T) {
	cfg := config.Default()
	cfg.Search.Placeholder = "Find Wails documentation"
	cfg.Search.MaxResults = 18
	cfg.Search.RememberRecent = false
	page := &content.Page{Language: "en", URLPath: "guide", Title: "Guide", HTML: `<p>Read this guide.</p>`}
	data := templateData{Config: cfg, Page: page, SearchURL: "search-index.json", CurrentRoutes: map[string]*content.Page{"guide": page}}

	markup, err := renderPage(data)
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	for _, want := range []string{
		`data-search-placeholder="Find Wails documentation"`, `data-search-max-results="18"`, `data-search-recent="false"`,
		`aria-label="Search"`, `>Find Wails documentation</button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("configured search UI is missing %q: %s", want, got)
		}
	}
	for _, want := range []string{"dataset.searchPlaceholder", "dataset.searchMaxResults", "dataset.searchRecent", "slice(0, searchMaxResults)"} {
		if !strings.Contains(defaultThemeJS, want) {
			t.Errorf("search runtime is missing %q", want)
		}
	}

	cfg.Search.Enabled = false
	data.Config = cfg
	markup, err = renderPage(data)
	if err != nil {
		t.Fatal(err)
	}
	got = string(markup)
	if strings.Contains(got, `id="search"`) || !strings.Contains(got, `class="header-spacer"`) {
		t.Fatalf("disabled search still renders an interactive control: %s", got)
	}
}

func TestDocumentationPageRendersFrontmatterDescriptionAsLead(t *testing.T) {
	cfg := config.Default()
	page := &content.Page{
		Language: "en", URLPath: "guide/intro", Title: "Introduction",
		Description: "Start here to build your first site.", HTML: `<p>Generated Markdown body.</p>`,
	}
	markup, err := renderPage(templateData{Config: cfg, Page: page, Root: "../", SearchURL: "../search-index.json"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	if !strings.Contains(got, `<header class="docs-page-header"><h1>Introduction</h1>`) || !strings.Contains(got, `</p></header><p>Generated Markdown body.</p>`) {
		t.Fatalf("documentation title and description are not grouped in the page header: %s", got)
	}
	if !strings.Contains(got, `<p class="page-lead">Start here to build your first site.</p>`) {
		t.Fatalf("frontmatter description was not rendered as the page lead: %s", got)
	}
	for _, want := range []string{
		`.docs-page-header { position: relative; margin-bottom: 2rem; padding-bottom: 1.8rem; }`,
		`.docs-page-header::after { position: absolute; right: calc(max(var(--layout-content-toc-gap), calc((100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)) * -1);`,
		`.docs-page-header::after { right: 0; left: 0; }`,
	} {
		if !strings.Contains(defaultThemeCSS, want) {
			t.Errorf("page header separator CSS is missing %q", want)
		}
	}
}

func TestExpandableImageAddsOneAccessibleLightbox(t *testing.T) {
	cfg := config.Default()
	page := &content.Page{
		Language: "en", URLPath: "guide/images", Title: "Images",
		HTML: `<button class="mpress-image-expand" type="button"><span class="mpress-image-expand-content"><img src="/diagram.png" alt="Diagram"></span></button>`,
	}
	markup, err := renderPage(templateData{Config: cfg, Page: page, Root: "../", SearchURL: "../search-index.json"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	for _, want := range []string{
		`id="mpress-image-lightbox"`,
		`aria-label="Expanded image"`,
		`data-image-lightbox-close`,
		`class="lucide lucide-x"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expandable image page is missing %q", want)
		}
	}
	if count := strings.Count(got, `id="mpress-image-lightbox"`); count != 1 {
		t.Fatalf("rendered %d image lightboxes, want one", count)
	}
	for _, want := range []string{
		`const imageLightbox = document.querySelector('#mpress-image-lightbox')`,
		`imageLightbox.showModal()`,
		`if (event.target === imageLightbox) imageLightbox.close()`,
		`if (event.key === 'Escape' && imageLightbox?.open) imageLightbox.close()`,
		`.mpress-image-lightbox::backdrop`,
	} {
		source := defaultThemeJS
		if strings.HasPrefix(want, ".") {
			source = defaultThemeCSS
		}
		if !strings.Contains(source, want) {
			t.Errorf("image lightbox assets are missing %q", want)
		}
	}
}

func BenchmarkTypedRendererDocumentation(b *testing.B) {
	cfg := config.Default()
	page := &content.Page{
		Language: "en", URLPath: "guide/intro", Title: "Introduction", Description: "A documentation page",
		HTML:     `<p>Generated Markdown body.</p>`,
		Headings: []content.Heading{{ID: "install", Text: "Install", Level: 2}, {ID: "next", Text: "Next", Level: 3}},
	}
	items := make([]navigation.Item, 0, 80)
	for index := 0; index < 80; index++ {
		items = append(items, navigation.Item{Label: "Page", Link: "/guide/page"})
	}
	items[40] = navigation.Item{Label: "Introduction", Link: "/guide/intro"}
	data := templateData{
		Config: cfg, Page: page, Root: "../", SearchURL: "../search-index.json",
		Nav:           []navigation.Item{{Label: "Guide", Collapsed: true, Items: items}},
		CurrentRoutes: map[string]*content.Page{"guide/intro": page},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := renderPage(data); err != nil {
			b.Fatal(err)
		}
	}
}
