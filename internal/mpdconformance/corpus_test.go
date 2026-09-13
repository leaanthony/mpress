package mpdconformance

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/components"
	"github.com/leaanthony/mpress/internal/content"
)

func TestEmbeddedCorpusIsValid(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := corpus.CountLevel(LevelTop), 18; got != want {
		t.Fatalf("top-level fixture count = %d, want %d", got, want)
	}
	if got, want := corpus.CountLevel(LevelNested), 9; got != want {
		t.Fatalf("nested fixture count = %d, want %d", got, want)
	}
	wantCategories := []string{"blocks", "components", "document", "footnotes", "imports", "links", "lists", "nesting", "tables", "text"}
	if got := corpus.Categories(); !reflect.DeepEqual(got, wantCategories) {
		t.Fatalf("categories = %q, want %q", got, wantCategories)
	}
}

func TestMinimalDocumentHasNoPreamble(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		if fixture.ID == "document/minimal" {
			if got, want := fixture.MPD, "Hello world.\n"; got != want {
				t.Fatalf("minimal MPD = %q, want %q", got, want)
			}
			return
		}
	}
	t.Fatal("minimal document fixture is missing")
}

func TestCodeFenceKeepsDirectivesLiteral(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		if fixture.ID != "blocks/all" {
			continue
		}
		for _, required := range []string{"````go", "\n@end\n", "\n```\n", "\n````\n"} {
			if !strings.Contains(fixture.MPD, required) {
				t.Errorf("code fixture does not contain %q", required)
			}
		}
		if strings.Contains(fixture.MPD, `\@`) {
			t.Fatal("code fixture escapes @ even though fenced code is opaque")
		}
		if strings.Contains(fixture.MPD, "@code{") {
			t.Fatal("code fixture uses the removed @code block")
		}
		return
	}
	t.Fatal("code block fixture is missing")
}

func TestMPDUsesRegistryClassifiedLeavesWithoutMarkers(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		fenceWidth := 0
		for lineNumber, line := range strings.Split(fixture.MPD, "\n") {
			trimmed := strings.TrimSpace(line)
			if run := leadingBackticks(trimmed); run >= 3 {
				if fenceWidth == 0 {
					fenceWidth = run
				} else if run >= fenceWidth && strings.TrimSpace(trimmed[run:]) == "" {
					fenceWidth = 0
				}
				continue
			}
			if fenceWidth == 0 && strings.HasPrefix(trimmed, "@") && strings.HasSuffix(trimmed, " /") {
				t.Errorf("fixture %q line %d uses a removed leaf marker: %s", fixture.ID, lineNumber+1, trimmed)
			}
		}
	}
}

func TestLineBreakAndContinuationContract(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		if fixture.ID != "text/blocks" {
			continue
		}
		for _, source := range []string{"The first line ends here.\nThis line starts after a hard break.", "This sentence is \\\ncontinued without a rendered line break.", "@hr\n"} {
			if !strings.Contains(fixture.MPD, source) {
				t.Errorf("MPD line fixture does not contain %q", source)
			}
		}
		for _, output := range []string{"The first line ends here.<br>\nThis line starts after a hard break.", "This sentence is continued without a rendered line break.", "<hr>"} {
			if !strings.Contains(fixture.HTML, output) {
				t.Errorf("rendered line fixture does not contain %q", output)
			}
		}
		return
	}
	t.Fatal("text block fixture is missing")
}

func TestMetadataReferenceGrammarAndNesting(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	wantReferences := []string{
		"@metadata[title]",
		"@metadata[schema]",
		"@metadata[order]",
		"@metadata[draft]",
		"@metadata[tags]",
		"@metadata[wails.redirect]",
		"@metadata[wails.owner]",
	}
	foundTop := false
	foundNested := false
	for _, fixture := range corpus.Fixtures {
		switch fixture.ID {
		case "document/metadata-reference":
			foundTop = true
			for _, reference := range wantReferences {
				if !strings.Contains(fixture.MPD, reference) {
					t.Errorf("metadata fixture does not contain %q", reference)
				}
			}
			if strings.Contains(fixture.MPD, "@metadata{") {
				t.Error("metadata fixture uses component attribute syntax for an inline reference")
			}
		case "nesting/note-details":
			foundNested = strings.Contains(fixture.MPD, "@metadata[title]")
		}
	}
	if !foundTop {
		t.Error("top-level metadata reference fixture is missing")
	}
	if !foundNested {
		t.Error("nested component metadata reference fixture is missing")
	}
}

func TestEmojiShortcodeContract(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		if fixture.ID != "text/inline" {
			continue
		}
		for _, source := range []string{`:rocket:`, `:sparkles:`, `:not_an_emoji:`, `\:rocket:`, "`:rocket:`"} {
			if !strings.Contains(fixture.MPD, source) {
				t.Errorf("MPD emoji fixture does not contain %q", source)
			}
		}
		for _, output := range []string{"🚀", "✨", `:not_an_emoji:`, `escaped :rocket:`, `<code>:rocket:</code>`} {
			if !strings.Contains(fixture.HTML, output) {
				t.Errorf("rendered emoji fixture does not contain %q", output)
			}
		}
		return
	}
	t.Fatal("inline text fixture is missing")
}

func TestTopLevelCorpusCoversEveryRegisteredComponent(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	covered := make(map[string]bool)
	for _, fixture := range corpus.Fixtures {
		if fixture.Level != LevelTop {
			continue
		}
		for _, name := range fixture.Components {
			if covered[name] {
				t.Fatalf("component %q is claimed by more than one top-level fixture", name)
			}
			covered[name] = true
		}
	}
	var got, want []string
	for name := range covered {
		got = append(got, name)
	}
	for name := range components.Registry {
		want = append(want, name)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("covered components do not match registry\ncovered: %v\nregistry: %v", got, want)
	}
}

func TestNestedCorpusCoversContainerFamilies(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	covered := make(map[string]bool)
	for _, fixture := range corpus.Fixtures {
		if fixture.Level != LevelNested {
			continue
		}
		covered[fixture.ParentComponent] = true
		if len(fixture.NestedComponents) == 0 {
			t.Errorf("nested fixture %q does not name its child components", fixture.ID)
		}
	}
	want := []string{"audience", "container", "details", "if", "note", "section", "steps", "tabs", "tutorial"}
	for _, name := range want {
		if !covered[name] {
			t.Errorf("nestable component family %q has no second-level fixture", name)
		}
	}
}

func TestNestedTabsFixtureUsesDefaultTerminalChrome(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		if fixture.ID != "nesting/tabs-rich" {
			continue
		}
		if strings.Contains(fixture.MPD, `frame="plain"`) || strings.Contains(fixture.Markdown, `frame="plain"`) {
			t.Fatalf("nested tabs fixture suppresses the default terminal chrome")
		}
		if !strings.Contains(fixture.HTML, `mpress-terminal-macos`) {
			t.Fatalf("nested tabs fixture does not render the default terminal frame: %s", fixture.HTML)
		}
		return
	}
	t.Fatal("nested tabs fixture is missing")
}

func TestCorpusTerminalsDeclarePromptAndCommentCharacters(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		for sourceName, source := range map[string]string{"MPD": fixture.MPD, "Markdown": fixture.Markdown} {
			for _, line := range strings.Split(source, "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, "@terminal{") && !strings.HasPrefix(line, "@terminal ") {
					continue
				}
				if !strings.Contains(line, `prompt=`) || !strings.Contains(line, `comment=`) {
					t.Errorf("%s fixture %q terminal must declare prompt and comment metadata: %s", sourceName, fixture.ID, line)
				}
			}
		}
	}
}

func TestHTMLGoldensMatchProductionRenderer(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	renderer := content.NewRenderer()
	for _, fixture := range corpus.Fixtures {
		fixture := fixture
		t.Run(strings.ReplaceAll(fixture.ID, "/", "_"), func(t *testing.T) {
			page, diagnostics, err := renderer.Parse(fixture.ID+".md", "en", fixture.Markdown)
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Severity == "error" {
					t.Fatalf("render diagnostic %s: %s", diagnostic.Code, diagnostic.Message)
				}
			}
			if page.HTML != fixture.HTML {
				t.Fatalf("HTML golden is stale; run `go run ./cmd/mpd-fixtures -root .`")
			}
		})
	}
}

func TestViewerIsReproducible(t *testing.T) {
	corpus, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(t.TempDir(), "viewer")
	if err := GenerateViewer(corpus, generated); err != nil {
		t.Fatal(err)
	}
	wantRoot := filepath.Join(packageRoot(t), "testdata", "viewer")
	wantFiles, err := FileSet(wantRoot)
	if err != nil {
		t.Fatal(err)
	}
	gotFiles, err := FileSet(generated)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("viewer file set differs\ngot: %v\nwant: %v", gotFiles, wantFiles)
	}
	for _, name := range wantFiles {
		got, err := os.ReadFile(filepath.Join(generated, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(wantRoot, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("generated viewer file %q is stale", name)
		}
	}
}

func TestViewerSourcePaneFillsDesktopFixtureHeight(t *testing.T) {
	for _, want := range []string{
		`.mpd-source { display: grid; grid-template-rows: 42px minmax(0, 1fr);`,
		`height: 100%; align-self: stretch;`,
		`.mpd-source pre { min-height: 0; height: 100%; max-height: none;`,
		`.mpd-source { height: auto; border-right: 0;`,
		`.mpd-source pre { height: auto; max-height: 560px; }`,
	} {
		if !strings.Contains(viewerCSS, want) {
			t.Errorf("fixture viewer source pane CSS is missing %q", want)
		}
	}
}

func TestMPDDisplayFormatterIndentsComponentsAndProtectsCode(t *testing.T) {
	input := "@tabs\n@tab label=\"Command\"\n@terminal prompt=\"$\"\n$ build\n@end\n@end\n@tab label=\"Source\"\n```go\n@end\n```\n@end\n@end\n"
	want := "@tabs\n  @tab label=\"Command\"\n    @terminal prompt=\"$\"\n      $ build\n    @end\n  @end\n  @tab label=\"Source\"\n    ```go\n    @end\n    ```\n  @end\n@end"
	if got := formatMPDForDisplay(input); got != want {
		t.Fatalf("formatted MPD differs\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestMPDDisplayFormatterKeepsInlineAndLeafDirectivesAtCurrentDepth(t *testing.T) {
	input := "@section\n@metadata[title]\n@image src=\"hero.png\"\n@end\n"
	want := "@section\n  @metadata[title]\n  @image src=\"hero.png\"\n@end"
	if got := formatMPDForDisplay(input); got != want {
		t.Fatalf("formatted MPD differs\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestMPDDisplayFormatterKeepsRawHTMLLiteral(t *testing.T) {
	input := "@rawHTML\n<section>@note{type=\"tip\"}</section>\n@end\n"
	want := "@rawHTML\n  <section>@note{type=\"tip\"}</section>\n@end"
	if got := formatMPDForDisplay(input); got != want {
		t.Fatalf("formatted raw HTML differs\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestCanonicalDirectiveSyntaxRejectsBracesAndLeafMarkers(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		line   int
	}{
		{name: "braced attributes", source: "@note{type=\"tip\"}\n@end\n", line: 1},
		{name: "compact leaf", source: "@hr/\n", line: 1},
		{name: "spaced leaf", source: "@hr /\n", line: 1},
		{name: "braces in code", source: "```mpd\n@note{type=\"tip\"}\n```\n", line: 0},
		{name: "braces in raw HTML", source: "@rawHTML\n@note{type=\"tip\"}\n@end\n", line: 0},
		{name: "removed generic raw", source: "@raw format=\"html\"\n<p>HTML</p>\n@end\n", line: 1},
		{name: "attributes on raw HTML", source: "@rawHTML format=\"html\"\n<p>HTML</p>\n@end\n", line: 1},
		{name: "canonical", source: "@note type=\"tip\"\n@end\n@hr\n@image src=\"/hero.png\"\n", line: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			line, _ := invalidCanonicalDirective(test.source)
			if line != test.line {
				t.Fatalf("invalid line = %d, want %d", line, test.line)
			}
		})
	}
}

func TestViewerIncludesThemeAndCorpusControls(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(packageRoot(t), "testdata", "viewer", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	for _, required := range []string{
		`data-preview-theme="system"`,
		`data-preview-theme="light"`,
		`data-preview-theme="dark"`,
		`data-fixture-search`,
		`assets/mpress.css`,
		`assets/mpress.js`,
	} {
		if !strings.Contains(html, required) {
			t.Errorf("viewer does not contain %q", required)
		}
	}
	if got, want := strings.Count(html, `class="mpress-theme-image"`), 1; got != want {
		t.Errorf("viewer contains %d rendered theme-image components, want %d; source examples may have been executed", got, want)
	}
}

func packageRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate package directory")
	}
	return filepath.Dir(file)
}
