package content

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseBytesMatchesParse(t *testing.T) {
	source := []byte("---\ntitle: Bytes\n---\n\n# Bytes\n\nA paragraph with **Markdown**.\n")
	renderer := NewRenderer()
	fromString, stringDiagnostics, err := renderer.Parse("bytes.md", "en", string(source))
	if err != nil {
		t.Fatal(err)
	}
	fromBytes, byteDiagnostics, err := renderer.ParseBytes("bytes.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromString, fromBytes) || !reflect.DeepEqual(stringDiagnostics, byteDiagnostics) {
		t.Fatalf("ParseBytes differs from Parse: string=%#v bytes=%#v", fromString, fromBytes)
	}
}

func TestLeadingH1BecomesPageTitleWithoutRenderingTwice(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("index.md", "en", "# Hello world\n\nThis site was built from Markdown.\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if page.Title != "Hello world" {
		t.Fatalf("title = %q, want Hello world", page.Title)
	}
	if strings.Contains(page.HTML, "<h1") {
		t.Fatalf("inferred title was also rendered in the body: %s", page.HTML)
	}
	if strings.Count(page.HTML, "Hello world") != 0 {
		t.Fatalf("inferred title leaked into body HTML: %s", page.HTML)
	}
	if len(page.Headings) != 0 {
		t.Fatalf("page title should not appear in the table of contents: %#v", page.Headings)
	}
	if !strings.Contains(page.HTML, "This site was built from Markdown.") {
		t.Fatalf("body content was removed with the title: %s", page.HTML)
	}
}

func TestExplicitTitlePreservesAuthoredH1(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("index.md", "en", "---\ntitle: Site title\n---\n\n# Authored heading\n")
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	if !strings.Contains(page.HTML, "<h1") || !strings.Contains(page.HTML, "Authored heading") {
		t.Fatalf("explicitly authored H1 was removed: %s", page.HTML)
	}
}

func TestHeadingScannerPreservesRenderedHTMLSemantics(t *testing.T) {
	body := `<h2 id="install">Install <em>M-Press</em> &amp; tools</h2>` +
		`<h3 id="mismatch">Accepted by the old expression</h4>` +
		`<h4 id="">Ignored empty ID</h4>` +
		`<h5 class="extra" id="ignored">Ignored extra attribute</h5>`
	got := headings(body)
	want := []Heading{
		{Level: 2, ID: "install", Text: "Install M-Press & tools"},
		{Level: 3, ID: "mismatch", Text: "Accepted by the old expression"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("headings() = %#v, want %#v", got, want)
	}
}

func BenchmarkRendererParseInputConversion(b *testing.B) {
	source := []byte("# Benchmark\n\n" + strings.Repeat("A paragraph with **Markdown** and a [link](/guide/).\n\n", 80))
	b.Run("string", func(b *testing.B) {
		renderer := NewRenderer()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, _, err := renderer.Parse("benchmark.md", "en", string(source)); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("bytes", func(b *testing.B) {
		renderer := NewRenderer()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, _, err := renderer.ParseBytes("benchmark.md", "en", source); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestComponentsAndCodeProtection(t *testing.T) {
	source := "---\ntitle: Components\n---\n\n" +
		"Use `Array<T>` and:\n\n" +
		"```ts\nconst item: Result<T> = value\n```\n\n" +
		"<Aside type=\"tip\" title=\"Small is good\">Markdown **works**.</Aside>\n\n" +
		"<LinkCard title=\"Install\" href=\"/install/\" description=\"Start here\" />\n"
	page, diagnostics, err := NewRenderer().Parse("index.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	for _, want := range []string{"Array&lt;T&gt;", "mpress-admonition tip", "<strong>works</strong>", "mpress-link-card"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestTextFirstMarkdownForms(t *testing.T) {
	source := `---
title: Settings
---

@form[config.save]

# Edit this site

Change the settings below.

## Identity

Site title*: [text](site.title)

Description: [textarea](site.description){rows=4 placeholder="A short description"}

## Appearance

Colour scheme: [select](theme.mode)
- Use the device setting = system
- Light
- Dark

[ ] Enable accessibility controls (accessibility.enabled)

Theme: [radio](theme.mode)
- (x) System = system
- ( ) Light = light

~~~text
Site title*: [text](site.title)
~~~

[Save and rebuild](submit)

@end

[This stays a link](/outside/)
`
	page, diagnostics, err := NewRenderer().Parse("settings.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	for _, want := range []string{
		`<form class="mpress-form" data-mpress-action="config.save" method="post">`,
		`<label for="site-title"><span>Site title <sup class="mpress-form-required" aria-hidden="true">*</sup></span><input type="text" id="site-title" name="site.title" required`,
		`<textarea id="site-description" name="site.description" placeholder="A short description" rows="4">`,
		`<select id="theme-mode" name="theme.mode">`,
		`<option value="system">Use the device setting</option>`,
		`<input type="checkbox" id="accessibility-enabled" name="accessibility.enabled">`,
		`<input type="radio" id="theme-mode-2-1" name="theme.mode" value="system" checked>`,
		`<button class="mpress-form-button mpress-form-button-primary" type="submit">Save and rebuild</button>`,
		// Fences inside a form body are rendered through the same code frame as
		// every other fence. They previously bypassed it, because the form body is
		// converted by a nested goldmark pass that the old string pre-pass never saw.
		`<code class="language-text"><span class="mpress-code-line">Site title*: [text](site.title)</span>`,
		`<a href="/outside/">This stays a link</a>`,
	} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered form missing %q:\n%s", want, page.HTML)
		}
	}
	if strings.Contains(page.HTML, `@form`) || strings.Contains(page.HTML, `@end`) {
		t.Fatalf("form directives leaked into output: %s", page.HTML)
	}
}

func TestInvalidFormIsDiagnosable(t *testing.T) {
	_, diagnostics, err := NewRenderer().Parse("settings.md", "en", "@form[config.save]\n\nSite title: [text](site.title)\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) == 0 || diagnostics[0].Code != "invalid-form" {
		t.Fatalf("expected invalid-form diagnostic, got %#v", diagnostics)
	}
}

func TestDocumentHeadingsAreNormalisedForThePageShell(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("guide.md", "en", "---\ntitle: Guide\n---\n\n# Guide\n\n# Another section\n\n## Detail\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if strings.Contains(page.HTML, "<h1") || !strings.Contains(page.HTML, `<h2 id="another-section">Another section</h2>`) {
		t.Fatalf("document headings were not normalised: %s", page.HTML)
	}
}

func TestInvalidControlCharactersAndEmptyLinksAreErrors(t *testing.T) {
	_, diagnostics, err := NewRenderer().Parse("broken.md", "en", "# Broken\n\nA\x01B and [](https://example.com).\n\n[![](image.png)](/details/)\n")
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, diagnostic := range diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes["invalid-control-character"] || !codes["empty-link-label"] || !codes["unnamed-image-link"] {
		t.Fatalf("missing source diagnostics: %#v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Line == 0 || diagnostic.Column == 0 || diagnostic.Suggestion == "" {
			t.Fatalf("diagnostic is not actionable: %#v", diagnostic)
		}
	}
}

func TestLinkValidationIgnoresCode(t *testing.T) {
	_, diagnostics, err := NewRenderer().Parse("code.md", "en", "# Code\n\n```go\njson.MarshalIndent(data, \"\", \"  \")\nvar literal = `[](/not-a-link)`\n```\n\nInline `[](/also-not-a-link)`.\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "empty-link-label" {
			t.Fatalf("code was treated as prose: %#v", diagnostics)
		}
	}
}

func TestDuplicateHeadingIDsAreMadeUnique(t *testing.T) {
	page, _, err := NewRenderer().Parse("guide.md", "en", "# First\n\n## Content\n\n## Repeat\n\n## Repeat\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.HTML, `id="repeat"`) || !strings.Contains(page.HTML, `id="repeat-1"`) {
		t.Fatalf("duplicate heading IDs remain: %s", page.HTML)
	}
	if !strings.Contains(page.HTML, `id="content-2"`) || strings.Contains(page.HTML, `id="content"`) {
		t.Fatalf("structural content ID was not reserved: %s", page.HTML)
	}
}

func TestExplicitTitleReservesStructuralIDsAndFocusesCode(t *testing.T) {
	page, _, err := NewRenderer().Parse("guide.md", "en", "---\ntitle: Guide\n---\n\n# Introduction\n\n## Content\n\n```sh\nprintf hello\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<h1 id="introduction">`, `id="content-2"`, `<pre tabindex="0"`} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("rendered page is missing %q: %s", want, page.HTML)
		}
	}
	if strings.Contains(page.HTML, `id="content"`) {
		t.Fatalf("rendered body conflicts with the page shell: %s", page.HTML)
	}
}

func TestFormSyntaxInsideFenceStaysLiteral(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("example.md", "en", "# Example\n\n```md\n@form[config.save]\n\nSite title: [text](site.title)\n\n@end\n```\n")
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	if strings.Contains(page.HTML, `<form class="mpress-form"`) {
		t.Fatalf("form syntax inside a code fence was rendered: %s", page.HTML)
	}
	if !strings.Contains(page.HTML, `@form[config.save]`) || !strings.Contains(page.HTML, `@end`) {
		t.Fatalf("form example was not preserved: %s", page.HTML)
	}
}

func TestShortcutFormFieldRendersCaptureMetadata(t *testing.T) {
	source := "# Shortcut settings\n\n@form[config.site]\n\nSearch shortcut: [shortcut](search.shortcut){placeholder=\"Mod+K\"}\n\n@end\n"
	page, diagnostics, err := NewRenderer().Parse("shortcut.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`type="text"`, `data-mpress-control="shortcut"`, `inputmode="none"`, `autocomplete="off"`, `aria-describedby="search-shortcut-status"`, `data-shortcut-status`} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("shortcut field missing %q: %s", want, page.HTML)
		}
	}
}

func TestUnknownMDXIsVisibleAndStrictlyDiagnosable(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("index.md", "en", "# Home\n\n<MorphText />\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "unsupported-component" {
		t.Fatalf("expected unsupported-component diagnostic, got %#v", diagnostics)
	}
	if !strings.Contains(page.HTML, "mpress-unsupported") || !strings.Contains(page.HTML, "MorphText") {
		t.Fatalf("unsupported component disappeared from output: %s", page.HTML)
	}
}

func TestNativeTabs(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("tabs.md", "en", "# Tabs\n\n:::tabs\n[Go]\nUse `Go`.\n[Python]\nUse Python.\n:::endtabs\n")
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	if strings.Count(page.HTML, `role="tab"`) != 2 || strings.Count(page.HTML, `role="tabpanel"`) != 2 {
		t.Fatalf("tabs not rendered: %s", page.HTML)
	}
	if !strings.Contains(page.HTML, "<code>Go</code>") {
		t.Fatalf("inline code inside tab was not rendered: %s", page.HTML)
	}
}

func TestImportOnlyPageSkipsComponentPassesWithoutChangingOutput(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("import.md", "en", "import { Aside } from '@astrojs/starlight/components';\n\n# Heading\n\nOrdinary **Markdown**.\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if strings.Contains(page.HTML, "import { Aside }") || !strings.Contains(page.HTML, "<strong>Markdown</strong>") {
		t.Fatalf("import-only page rendered incorrectly: %s", page.HTML)
	}
}

func TestNativeBlocksDoNotConsumeFollowingMarkdown(t *testing.T) {
	source := "# Page\n\n:::cards\nFirst card\n---\nSecond card\n:::\n\n## After cards\n\n:::note{type=\"tip\" title=\"Notice\"}\nBody survives.\n:::\n\n## After note\n"
	page, diagnostics, err := NewRenderer().Parse("blocks.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`id="after-cards"`, `id="after-note"`, "Body survives.", "mpress-card-grid"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestCompatibilityBlocksPreserveMarkdownBoundaries(t *testing.T) {
	source := `# Page

:::audience{role="writer"}
Use **Markdown** here.
:::

## After audience

:::tutorial{title="Try it"}
### Build
Run ` + "`mpress build`" + `.
:::

## After tutorial
`
	page, diagnostics, err := NewRenderer().Parse("compatibility.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`id="after-audience"`, `id="after-tutorial"`, `<strong>Markdown</strong>`, `<code>mpress build</code>`} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, page.HTML)
		}
	}
	if strings.Contains(page.HTML, `&lt;/div&gt;`) {
		t.Fatalf("component closing tags became code: %s", page.HTML)
	}
}

func TestNativeSteps(t *testing.T) {
	source := "# Steps\n\n:::steps\n1. Install `mpress`.\n2. Build the site.\n:::\n"
	page, diagnostics, err := NewRenderer().Parse("steps.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`class="mpress-timeline mpress-steps"`, "<ol>", "<code>mpress</code>"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered steps missing %q: %s", want, page.HTML)
		}
	}
}

func TestNativeStepsMayContainNotesTabsAndFencedCode(t *testing.T) {
	source := `# Tutorial

:::steps

1. ## Start the app

   Run the first command:

   ~~~bash
   wails3 task dev
   ~~~

   :::note{type="note" title="Keep the app open"}

       This note belongs to the first step.
__INDENTED_BLANK__
:::

2. ## Build a release

   Choose a platform:

:::tabs
[macOS]

~~~bash
wails3 task build:darwin
~~~

[Linux]

~~~bash
wails3 task build:linux
~~~
:::endtabs

3. ## Finish

   The final step must remain inside the component.

:::
`
	source = strings.Replace(source, "__INDENTED_BLANK__", "   ", 1)
	page, diagnostics, err := NewRenderer().Parse("nested-steps.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{
		`class="mpress-timeline mpress-steps"`,
		`class="mpress-admonition mpress-admonition-note"`,
		`class="mpress-tabs"`,
		`class="mpress-terminal mpress-terminal-macos"`,
		`data-commands="wails3 task dev"`,
		`Start the app`,
		`Build a release`,
		`Finish`,
	} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("nested steps missing %q:\n%s", want, page.HTML)
		}
	}
	if got := strings.Count(page.HTML, `class="mpress-timeline-entry mpress-step"`); got != 3 {
		t.Fatalf("expected 3 rendered steps, got %d:\n%s", got, page.HTML)
	}
	for _, raw := range []string{":::steps", ":::note", ":::tabs", ":::endtabs"} {
		if strings.Contains(page.HTML, raw) {
			t.Fatalf("raw directive %q leaked into rendered HTML:\n%s", raw, page.HTML)
		}
	}
}

func TestTabbedCodeFencesDedentAndPreserveBlankLines(t *testing.T) {
	source := `# Tabbed code

:::steps
1. ## Configure the app

:::tabs
[Go]

            ~~~go
            package main

            func main() {}
            ~~~

            :::note{type="note" title="Inside the tab"}

                The nested note stays out of the code frame.

:::

[Shell]

            ~~~bash
            wails3 task build
            ~~~
:::endtabs

:::
`
	page, diagnostics, err := NewRenderer().Parse("tabbed-code.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{
		`data-code="package main&#10;&#10;func main() {}"`,
		`class="mpress-token-keyword">package`,
		`data-commands="wails3 task build"`,
		`class="mpress-admonition mpress-admonition-note"`,
		`The nested note stays out of the code frame.`,
	} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("tabbed code missing %q:\n%s", want, page.HTML)
		}
	}
	if strings.Contains(page.HTML, `data-code="   `) || strings.Contains(page.HTML, `data-commands="   `) || strings.Contains(page.HTML, `class="mpress-code-line">   `) {
		t.Fatalf("tabbed code retained source indentation:\n%s", page.HTML)
	}
	if got := strings.Count(page.HTML, `class="mpress-codeframe"`); got != 1 {
		t.Fatalf("expected one code frame, got %d:\n%s", got, page.HTML)
	}
	if got := strings.Count(page.HTML, `class="mpress-terminal mpress-terminal-macos"`); got != 1 {
		t.Fatalf("expected one terminal, got %d:\n%s", got, page.HTML)
	}
}

func TestStepCodeFencePreservesNestedIndentation(t *testing.T) {
	source := `# Go code

:::steps
1. ## Prepare the project

   Keep this first step so later headings cannot contribute line breaks to indentation.

2. ## Sign the release

   ~~~go title="cmd/sign/main.go"
   package main

   import (
       "fmt"
       "os"
   )

   func main() {
       fmt.Println(os.Args[0])
   }
   ~~~
:::
`
	page, diagnostics, err := NewRenderer().Parse("step-code.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{
		`data-code="package main&#10;&#10;import (&#10;    &#34;fmt&#34;&#10;    &#34;os&#34;&#10;)`,
		`func main() {&#10;    fmt.Println(os.Args[0])&#10;}`,
	} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("step code lost nested indentation; missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestMoreThanTenProtectedCodeBlocksRestoreExactly(t *testing.T) {
	var source strings.Builder
	source.WriteString("# Many examples\n\n")
	for i := 0; i < 14; i++ {
		source.WriteString("`Array<T>`\n\n```md\n![example](/missing.png)\n```\n\n")
	}
	page, diagnostics, err := NewRenderer().Parse("many.md", "en", source.String())
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	if got := strings.Count(page.HTML, `language-md`); got != 14 {
		t.Fatalf("expected 14 fenced blocks, got %d:\n%s", got, page.HTML)
	}
	if got := strings.Count(page.HTML, `<img`); got != 0 {
		t.Fatalf("code examples became live images: %d\n%s", got, page.HTML)
	}
}

func TestComponentDetectorIgnoresOrdinaryAtMentions(t *testing.T) {
	// Changelogs contain thousands of contributor handles. An @mention is prose,
	// not an M-Press component, and must not make the complete component and form
	// pipeline run for an otherwise ordinary Markdown document.
	source := "# Release notes\n\nThanks to @leaanthony and @taliesin-ai for this release.\n\nUse `@variant{name=\"plain\"}` in an example.\n"
	if hasComponentSyntax(source) {
		t.Fatalf("ordinary @mentions were detected as components: %q", source)
	}
	page, diagnostics, err := NewRenderer().Parse("changelog.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{"@leaanthony", "@taliesin-ai", `@variant{name=&quot;plain&quot;}`} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("ordinary Markdown lost %q: %s", want, page.HTML)
		}
	}
}

func TestComponentDetectorStillRecognisesLiveFormsAndComponents(t *testing.T) {
	for _, source := range []string{
		"@form[save]\n\nTitle*: [text](site.title)\n\n@end\n",
		":::note\nUseful information.\n:::\n",
		"<Aside type=\"tip\">Useful information.</Aside>",
		"Choose an option: @button[Start](/start/){primary}",
		"Diagram: @image{light=\"/diagram.svg\" alt=\"Diagram\"}",
	} {
		if !hasComponentSyntax(source) {
			t.Fatalf("live component syntax was not detected: %q", source)
		}
	}
}

func TestInlineNativeComponentsRenderMidParagraph(t *testing.T) {
	source := "# Start\n\nChoose an option: @button[Start](/start/){primary}. Diagram: @image{light=\"/diagram.svg\" alt=\"Diagram\"}\n"
	page, diagnostics, err := NewRenderer().Parse("inline.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`class="mpress-button mpress-button-primary"`, `href="/start/"`, `src="/diagram.svg"`, `alt="Diagram"`} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("inline component missing %q: %s", want, page.HTML)
		}
	}
}

func TestAnnotatedCodeFencePreservesTitleAndLineRanges(t *testing.T) {
	source := "# Code\n\n```go title=\"main.go\" {2-3} ins={3} del={4}\nfunc main() {\nline two\nline three\nline four\n```\n"
	page, diagnostics, err := NewRenderer().Parse("code.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`class="mpress-codeframe"`, `class="mpress-codeframe-header"`, `class="mpress-codeframe-title">main.go`, `class="mpress-codeframe-language">go`, `mpress-code-line-marked`, `mpress-code-line-inserted`, `mpress-code-line-deleted`, `class="language-go"`, `mpress-token-keyword`, `mpress-token-function`} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("annotated fence missing %q:\n%s", want, page.HTML)
		}
	}
	if titleAt, languageAt := strings.Index(page.HTML, `class="mpress-codeframe-title"`), strings.Index(page.HTML, `class="mpress-codeframe-language"`); titleAt < 0 || languageAt < 0 || titleAt > languageAt {
		t.Fatalf("annotated fence should place its language after its title:\n%s", page.HTML)
	}
}

func TestBracedCodeFenceAttributesRenderTitleAndLineNumbers(t *testing.T) {
	source := "# Code\n\n````go {title=\"Build the site\" lineNumbers=true}\nsite, err := mpress.Build(config)\n@end\n```\n````\n"
	page, diagnostics, err := NewRenderer().Parse("code.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{
		`class="mpress-codeframe mpress-codeframe-line-numbers"`,
		`class="mpress-codeframe-title">Build the site`,
		`site, err`,
		`@end`,
		`class="mpress-token-string"`,
		"@end&#10;```",
	} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("braced annotated fence missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestExplainedComponentKeepsFloatingAnnotationHooks(t *testing.T) {
	source := "# Languages\n\n@explained\n~~~yaml\nsite: # (1)\n~~~\n\n(1) Site settings.\n@end\n"
	renderer := NewRenderer()
	processed, diagnostics := renderer.processComponents("languages.md", "en", source)
	if len(diagnostics) != 0 {
		t.Fatalf("component diagnostics: %#v", diagnostics)
	}
	for _, want := range []string{`data-ref="1" data-explanation="Site settings." tabindex="0"`, `class="mpress-explained-copy"`} {
		if !strings.Contains(processed, want) {
			t.Fatalf("processed explained component missing %q:\n%s", want, processed)
		}
	}
	page, diagnostics, err := renderer.Parse("languages.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`data-ref="1" data-explanation="Site settings." tabindex="0"`, `class="mpress-explained-copy"`} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered explained component missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestOrdinaryCodeFenceGetsHighlightedFrameAndCopySource(t *testing.T) {
	source := "# Code\n\n```go\npackage main\n\nfunc main() {}\n```\n"
	page, diagnostics, err := NewRenderer().Parse("code.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`class="mpress-codeframe"`, `class="mpress-codeframe-header"`, `class="mpress-codeframe-language">go`, `class="mpress-copy mpress-code-copy"`, `lucide-copy`, `lucide-check`, `data-copy-status aria-live="polite"`, `data-code="package main`, `class="language-go"`, `mpress-token-keyword`} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("ordinary fence missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestShellFenceGetsTerminalChrome(t *testing.T) {
	source := "# Install\n\n```bash\ngo install example.test/tool@latest\n```\n"
	page, diagnostics, err := NewRenderer().Parse("install.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	for _, want := range []string{`class="mpress-terminal mpress-terminal-macos"`, `class="mpress-terminal-title"><span>bash</span>`, `class="mpress-copy"`, `lucide-copy`, `lucide-check`, `data-commands="go install example.test/tool@latest"`, `class="mpress-token-function">go</span>`} {
		if !strings.Contains(page.HTML, want) {
			t.Fatalf("shell fence missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestUnterminatedShellFenceAtEndOfFileStillGetsTerminalChrome(t *testing.T) {
	source := "# Install\n\n```bash\ngo install example.test/tool@latest"
	page, diagnostics, err := NewRenderer().Parse("install.md", "en", source)
	if err != nil || len(diagnostics) != 0 {
		t.Fatalf("parse: %v, diagnostics: %#v", err, diagnostics)
	}
	if !strings.Contains(page.HTML, `class="mpress-terminal mpress-terminal-macos"`) || !strings.Contains(page.HTML, `data-commands="go install example.test/tool@latest"`) {
		t.Fatalf("unterminated shell fence did not render as a terminal:\n%s", page.HTML)
	}
}
