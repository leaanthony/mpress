package content

import (
	"fmt"
	stdhtml "html"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var referenceCodeLanguageRE = regexp.MustCompile(`[^A-Za-z0-9_+.-]`)

func TestCodeFenceFastPathsMatchPreviousOutput(t *testing.T) {
	rng := rand.New(rand.NewSource(90210))
	alphabet := []byte("abcXYZ019_+.- /:|\t\r\n<>&'\"")
	for trial := 0; trial < 2000; trial++ {
		value := make([]byte, rng.Intn(80))
		for i := range value {
			value[i] = alphabet[rng.Intn(len(alphabet))]
		}
		input := string(value)
		if got, want := sanitizeCodeLanguage(input), referenceCodeLanguageRE.ReplaceAllString(input, ""); got != want {
			t.Fatalf("language sanitizer mismatch for %q: got %q, want %q", input, got, want)
		}
		lines := strings.Split(input, "|")
		raw := strings.Join(lines, "\n")
		want := stdhtml.EscapeString(raw)
		want = strings.ReplaceAll(want, "\r\n", "&#10;")
		want = strings.ReplaceAll(want, "\n", "&#10;")
		want = strings.ReplaceAll(want, "\r", "&#10;")
		if got := escapeCodeAttribute(lines); got != want {
			t.Fatalf("code attribute escape mismatch for %#v: got %q, want %q", lines, got, want)
		}
	}
}

// knownDivergent lists the fence shapes where the AST renderer deliberately
// differs from the superseded string pre-pass, with the reason. Every other
// shape must render byte for byte identically, so an unintended change in
// generated markup fails this test.
var knownDivergent = map[string]string{
	// The pre-pass was line based and could not see block context, so it lifted
	// a fence out of its container entirely. goldmark nests it correctly; see
	// TestFenceInListStaysInsideTheList.
	"indented-in-list": "pre-pass ejected the fence from the list item",
	"in-blockquote":    "pre-pass ejected the fence from the blockquote",
	// An unterminated fence made the pre-pass emit a trailing blank code line
	// that goldmark does not report.
	"unclosed": "pre-pass appended a phantom trailing line",
}

func fenceCases() map[string]string {
	return map[string]string{
		"plain":            "```go\nfunc main() {}\n```\n",
		"no-language":      "```\nplain text\n```\n",
		"tilde":            "~~~python\nprint(1)\n~~~\n",
		"title":            "```go title=\"main.go\"\nfunc main() {}\n```\n",
		"marked-lines":     "```go {1,3}\na\nb\nc\n```\n",
		"ins-del":          "```go ins={1} del={2}\nadded\nremoved\n```\n",
		"terminal-bash":    "```bash\necho hello\n```\n",
		"terminal-console": "```console\n$ ls -la\n```\n",
		"terminal-ps":      "```powershell\nGet-ChildItem\n```\n",
		"empty-body":       "```go\n```\n",
		"blank-lines":      "```go\na\n\nb\n```\n",
		"html-in-code":     "```html\n<div class=\"x\">&amp;</div>\n```\n",
		"backticks-inside": "~~~md\n```go\nnested\n```\n~~~\n",
		"surrounded":       "# Title\n\nBefore.\n\n```go\nx := 1\n```\n\nAfter.\n",
		"two-fences":       "```go\na\n```\n\n```js\nb\n```\n",
		"indented-in-list": "- item\n\n  ```go\n  x := 1\n  ```\n\n- next\n",
		"in-blockquote":    "> quoted\n>\n> ```go\n> x := 1\n> ```\n",
		"unclosed":         "```go\nnever closed\n",
		"crlf":             "```go\r\nx := 1\r\n```\r\n",
		"long-fence":       "````go\n```\ninner\n```\n````\n",
	}
}

// TestFenceRenderingMatchesLegacyPrePass pins the AST renderer against the
// superseded pre-pass across a corpus of fence shapes.
func TestFenceRenderingMatchesLegacyPrePass(t *testing.T) {
	legacy, current := newLegacyFencePrePassRenderer(), NewRenderer()
	for name, src := range fenceCases() {
		doc := "---\ntitle: T\n---\n\n" + src
		want, _, err := legacy.Parse("t.md", "en", doc)
		if err != nil {
			t.Fatalf("%s: legacy: %v", name, err)
		}
		got, _, err := current.Parse("t.md", "en", doc)
		if err != nil {
			t.Fatalf("%s: current: %v", name, err)
		}
		reason, expectedDifferent := knownDivergent[name]
		switch {
		case want.HTML == got.HTML && expectedDifferent:
			t.Errorf("%s: expected divergence (%s) but output now matches; update knownDivergent", name, reason)
		case want.HTML != got.HTML && !expectedDifferent:
			t.Errorf("%s: output diverged\nlegacy:  %s\ncurrent: %s", name, want.HTML, got.HTML)
		}
	}
}

// TestFenceRenderingUnchangedOnRealDocs asserts both renderers agree on the
// corpus this change actually has to preserve: the repository's own docs.
func TestFenceRenderingUnchangedOnRealDocs(t *testing.T) {
	root := "../../docs"
	if _, err := os.Stat(root); err != nil {
		t.Skip("docs corpus unavailable")
	}
	legacy, current := newLegacyFencePrePassRenderer(), NewRenderer()
	checked := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		// D2 now intentionally renders as a diagram; its contract is covered
		// by d2_test.go. Preserve this comparison for ordinary code fences.
		source = []byte(strings.ReplaceAll(string(source), "\n```d2\n", "\n```text\n"))
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return fmt.Errorf("resolve %s relative to %s: %w", path, root, relErr)
		}
		want, _, errA := legacy.ParseBytes(rel, "en", source)
		got, _, errB := current.ParseBytes(rel, "en", source)
		if errA != nil {
			return fmt.Errorf("parse %s with legacy renderer: %w", rel, errA)
		}
		if errB != nil {
			return fmt.Errorf("parse %s with AST renderer: %w", rel, errB)
		}
		if want == nil || got == nil {
			return fmt.Errorf("parse %s returned a nil page", rel)
		}
		checked++
		if want.HTML != got.HTML {
			t.Errorf("%s: AST code fences changed the generated page", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no documentation pages were compared")
	}
	t.Logf("compared %d documentation pages", checked)
}

// TestNestedFenceContentIsNotProcessedAsComponents covers component examples
// inside the same list and blockquote contexts that motivated the AST renderer.
// Component expansion runs before Goldmark, so fence protection must preserve
// both the container and the exact source that the author intended to show.
func TestNestedFenceContentIsNotProcessedAsComponents(t *testing.T) {
	tests := []struct {
		name       string
		markdown   string
		dataCode   string
		container  string
		containers int
	}{
		{
			name: "native component example in a list",
			markdown: "1. Add a button:\n\n" +
				"   ```md\n" +
				"   @button[Get started](/)\n" +
				"   ```\n\n" +
				"2. Continue.\n",
			dataCode:   `data-code="@button[Get started](/)"`,
			container:  "<ol",
			containers: 1,
		},
		{
			name: "MDX component example in a blockquote",
			markdown: "> An imported example:\n>\n" +
				"> ```html\n" +
				"> <Aside>\n" +
				"> Keep this literal.\n" +
				"> </Aside>\n" +
				"> ```\n",
			dataCode:   `data-code="&lt;Aside&gt;&#10;Keep this literal.&#10;&lt;/Aside&gt;"`,
			container:  "<blockquote>",
			containers: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := "---\ntitle: Nested fence\n---\n\n" + test.markdown
			page, diagnostics, err := NewRenderer().Parse("nested.md", "en", doc)
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %+v", diagnostics)
			}
			if !strings.Contains(page.HTML, test.dataCode) {
				t.Errorf("nested fence content was transformed before rendering\nwant: %s\nHTML: %s", test.dataCode, page.HTML)
			}
			if count := strings.Count(page.HTML, test.container); count != test.containers {
				t.Errorf("nested fence broke its container: found %d %s elements, want %d\nHTML: %s", count, test.container, test.containers, page.HTML)
			}
		})
	}
}

// TestFenceInListStaysInsideTheList covers the bug this renderer fixes. The
// superseded string pre-pass rewrote fences line by line with no block context,
// so a fenced block inside a list item was lifted out of the list: one ordered
// list became two and the code was stranded between them. The AST renderer
// keeps the block inside its list item. Both halves are asserted so a
// regression to the old behaviour fails loudly.
func TestFenceInListStaysInsideTheList(t *testing.T) {
	src := "---\ntitle: Install\n---\n\n" +
		"1. Install the binary:\n\n   ```sh\n   go install ./cmd/mpress\n   ```\n\n" +
		"2. Run it:\n\n   ```sh\n   mpress build\n   ```\n"

	legacy, _, err := newLegacyFencePrePassRenderer().Parse("i.md", "en", src)
	if err != nil {
		t.Fatal(err)
	}
	if lists := strings.Count(legacy.HTML, "<ol"); lists != 2 {
		t.Errorf("expected the superseded pre-pass to split the list into 2 <ol> elements, got %d; "+
			"if it now reports 1 the pre-pass and this comparison can be deleted", lists)
	}

	current, _, err := NewRenderer().Parse("i.md", "en", src)
	if err != nil {
		t.Fatal(err)
	}
	if lists := strings.Count(current.HTML, "<ol"); lists != 1 {
		t.Errorf("code fences must stay inside their list item, got %d <ol> elements: %s", lists, current.HTML)
	}
	if !strings.Contains(current.HTML, "</li>") || strings.Contains(current.HTML, "<ol start=") {
		t.Errorf("ordered list was not kept intact: %s", current.HTML)
	}
}

func benchmarkDocument() string {
	var b strings.Builder
	b.WriteString("---\ntitle: Bench\n---\n\n# Bench\n\n")
	for i := 0; i < 24; i++ {
		fmt.Fprintf(&b, "## Section %d\n\nParagraph with **bold**, `code`, and a [link](/page-0001/).\n\n"+
			"```go\nfunc sample%d() {}\n```\n\n- one\n- two\n- three\n\n", i, i)
	}
	return b.String()
}

func BenchmarkParseLegacyFencePrePass(b *testing.B) {
	renderer, source := newLegacyFencePrePassRenderer(), benchmarkDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := renderer.Parse("b.md", "en", source); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseASTFences(b *testing.B) {
	renderer, source := NewRenderer(), benchmarkDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := renderer.Parse("b.md", "en", source); err != nil {
			b.Fatal(err)
		}
	}
}
