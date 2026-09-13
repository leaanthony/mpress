package mpd

import (
	"strings"
	"testing"
)

func TestParseMetadataAndReferences(t *testing.T) {
	source := []byte("---\r\nschema = 1\r\ntitle = \"Fast docs\"\r\ncustom.owner = {\"name\":\"Lea\"}\r\n---\r\n# @metadata[title]\r\n\r\n@metadata[missing]\r\n")
	doc := Parse("metadata.mpd", source)
	if got, want := len(doc.Attributes), 3; got != want {
		t.Fatalf("attributes = %d, want %d", got, want)
	}
	if got := countKind(doc, KindMetadataReference); got != 2 {
		t.Fatalf("metadata references = %d, want 2", got)
	}
	if !hasDiagnostic(doc, "mpd-metadata-reference") {
		t.Fatal("missing undefined metadata diagnostic")
	}
	if got := doc.Text(doc.Attributes[2].Name); string(got) != "custom.owner" {
		t.Fatalf("custom name = %q", got)
	}
}

func TestParseCodeFenceIsOpaque(t *testing.T) {
	source := []byte("````go {title=\"Build\"}\n@note\n```\n@end\n````\n\nAfter.\n")
	doc := Parse("code.mpd", source)
	if got := countKind(doc, KindCode); got != 1 {
		t.Fatalf("code nodes = %d, want 1", got)
	}
	if got := countKind(doc, KindComponent); got != 0 {
		t.Fatalf("components inside opaque code = %d", got)
	}
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
}

func TestLiteralComponentsKeepTheirBodiesOpaque(t *testing.T) {
	source := []byte("@terminal prompt=\"$\"\n# A shell comment\n$ mpress build\n- literal output\n@end\n")
	doc := Parse("terminal.mpd", source)
	if got := countKind(doc, KindHeading); got != 0 {
		t.Fatalf("headings inside terminal = %d, want 0", got)
	}
	if got := countKind(doc, KindList); got != 0 {
		t.Fatalf("lists inside terminal = %d, want 0", got)
	}
	markdown, err := Markdown(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# A shell comment\n$ mpress build\n- literal output", "@terminal", "@end"} {
		if !strings.Contains(string(markdown), want) {
			t.Fatalf("rendered Markdown missing %q:\n%s", want, markdown)
		}
	}
}

func TestParagraphStopsBeforeAdjacentCodeFence(t *testing.T) {
	source := []byte("Generated binding:\n```typescript\nexport function GetTime(): Promise<Date>\n```\n")
	doc := Parse("adjacent-code.mpd", source)
	if got := countKind(doc, KindCode); got != 1 {
		t.Fatalf("code nodes = %d, want 1", got)
	}
	if got := countKind(doc, KindParagraph); got != 1 {
		t.Fatalf("paragraph nodes = %d, want 1", got)
	}
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
}

func TestParseCodeFenceInsideList(t *testing.T) {
	source := []byte("1. Build the app\n  ```go\n  func main() {}\n  ```\n2. Run it\n")
	doc := Parse("list-code.mpd", source)
	if got := countKind(doc, KindCode); got != 1 {
		t.Fatalf("code nodes = %d, want 1", got)
	}
	if got := countKind(doc, KindItem); got != 2 {
		t.Fatalf("items = %d, want 2", got)
	}
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
}

func TestParseNestedListContinuation(t *testing.T) {
	source := []byte("3. Add:\n  - Unit tests\n  - A regression test if you touched\n    the bindings generator\n4. Run checks locally\n  before pushing.\n")
	doc := Parse("nested-continuation.mpd", source)
	if got := countKind(doc, KindList); got != 2 {
		t.Fatalf("lists = %d, want 2", got)
	}
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
}

func TestParseNestedComponentsAndTable(t *testing.T) {
	source := []byte("@tabs\n@tab label=\"Go\"\n@note type=\"tip\"\nUse *Go*.\n@end\n@end\n@end\n\n@table header search\n| Name | Time |\n| Parse | `2 ms` |\n@end\n")
	doc := Parse("nested.mpd", source)
	for kind, want := range map[Kind]int{KindComponent: 3, KindTable: 1, KindTableRow: 2, KindTableCell: 4, KindStrong: 1, KindCodeSpan: 1} {
		if got := countKind(doc, kind); got != want {
			t.Errorf("%s nodes = %d, want %d", kind, got, want)
		}
	}
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
}

func TestVideoArrayUsesCompactMarkdownInterchange(t *testing.T) {
	document := Parse("video.mpd", []byte(`@video base="/media" src=["setup.webm","setup.mp4"] captions=["captions.en.vtt","captions.fr.vtt"]`))
	if len(document.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %#v", document.Diagnostics)
	}
	markdown, err := Markdown(document)
	if err != nil {
		t.Fatal(err)
	}
	got := string(markdown)
	for _, want := range []string{
		`@video{base="/media"`,
		`src="setup.webm,setup.mp4"`,
		`captions="captions.en.vtt,captions.fr.vtt"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Markdown interchange missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@end") {
		t.Fatalf("leaf video emitted an end marker:\n%s", got)
	}
}

func TestLinkCardIsALeafComponent(t *testing.T) {
	document := Parse("leaf.mpd", []byte("@container\n@linkcard title=\"Guide\" href=\"/guide/\"\n@end\n"))
	if len(document.Diagnostics) != 0 {
		t.Fatalf("leaf linkcard broke its containing component: %#v", document.Diagnostics)
	}
}

func TestParseLineBreaksAndContinuation(t *testing.T) {
	doc := Parse("lines.mpd", []byte("first\nsecond\n\njoined \\\n+line\n"))
	if got := countKind(doc, KindHardBreak); got != 1 {
		t.Fatalf("hard breaks = %d, want 1", got)
	}
	if got := countKind(doc, KindSoftBreak); got != 1 {
		t.Fatalf("continuations = %d, want 1", got)
	}
}

func TestParseListsQuotesAndTasks(t *testing.T) {
	source := []byte("- [ ] one\n- [x] two\n  - nested\n\n> quote\n>\n> > nested\n")
	doc := Parse("blocks.mpd", source)
	if got := countKind(doc, KindList); got != 2 {
		t.Fatalf("lists = %d, want 2", got)
	}
	if got := countKind(doc, KindItem); got != 3 {
		t.Fatalf("items = %d, want 3", got)
	}
	if got := countKind(doc, KindQuote); got != 2 {
		t.Fatalf("quotes = %d, want 2", got)
	}
}

func TestParseDiagnosticsAndRecovery(t *testing.T) {
	tests := []struct{ name, source, code string }{
		{"metadata", "---\nschema = 1\ntitle = nope\n", "mpd-metadata"},
		{"schema", "---\nschema = 9\n---\n", "mpd-version"},
		{"end", "@end\n", "mpd-end"},
		{"component", "@note type=\"tip\"\nunclosed\n", "mpd-unclosed"},
		{"directive", "@made-up value=1\n", "mpd-directive"},
		{"attribute", "@image src=bare\n", "mpd-attribute"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := Parse(test.name+".mpd", []byte(test.source))
			if !hasDiagnostic(doc, test.code) {
				t.Fatalf("diagnostics %#v do not contain %q", doc.Diagnostics, test.code)
			}
		})
	}
}

func TestReferenceValidation(t *testing.T) {
	source := []byte("[Known][known] [Missing][missing].[^note] [^lost]\n\n@link id=\"known\" destination=\"/known\"\n\n@footnote id=\"note\"\nDefined.\n@end\n\n@link id=\"known\" destination=\"/duplicate\"\n")
	doc := Parse("references.mpd", source)
	count := 0
	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.Code == "mpd-reference" {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("reference diagnostics = %d, want 3: %#v", count, doc.Diagnostics)
	}
}

func TestReferenceValidationOverflowAndJSONEscapes(t *testing.T) {
	source := []byte("[A][a] [B][b] [C][c] [D][d] [E][escaped].\n\n@link id=\"a\" destination=\"/a\"\n@link id=\"b\" destination=\"/b\"\n@link id=\"c\" destination=\"/c\"\n@link id=\"d\" destination=\"/d\"\n@link id=\"esc\\u0061ped\" destination=\"/e\"\n")
	doc := Parse("references-overflow.mpd", source)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
}

func TestCodeFenceAttributesAndOrderedNumbers(t *testing.T) {
	source := []byte("7. first\n8. second\n\n```go {title=\"Build\" lineNumbers}\nrun()\n```\n")
	doc := Parse("attributes.mpd", source)
	if len(doc.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v", doc.Diagnostics)
	}
	var numbers []string
	var code *Node
	for index := range doc.Nodes {
		node := &doc.Nodes[index]
		if node.Kind == KindItem {
			numbers = append(numbers, string(doc.Text(node.Name)))
		}
		if node.Kind == KindCode {
			code = node
		}
	}
	if got := strings.Join(numbers, ","); got != "7,8" {
		t.Fatalf("ordered numbers = %q, want 7,8", got)
	}
	if code == nil || string(doc.Text(code.Name)) != "go" || code.AttrCount != 2 {
		t.Fatalf("code node = %#v", code)
	}
	if first := doc.Attributes[code.FirstAttr]; string(doc.Text(first.Name)) != "title" || string(doc.Text(first.Value)) != `"Build"` {
		t.Fatalf("first code attribute = %#v", first)
	}
}

func TestTableWidthAndUnicodePositions(t *testing.T) {
	source := []byte("@table\n| α | beta |\n| only-one |\n@end\n")
	doc := Parse("table.mpd", source)
	if !hasDiagnostic(doc, "mpd-table") {
		t.Fatalf("missing table-width diagnostic: %#v", doc.Diagnostics)
	}
	var cells []Node
	for _, node := range doc.Nodes {
		if node.Kind == KindTableCell {
			cells = append(cells, node)
		}
	}
	if len(cells) != 3 || cells[1].Position.Column != 7 {
		t.Fatalf("cell positions = %#v", cells)
	}
}

func TestUnmatchedDelimiterKeepsConnectedChildren(t *testing.T) {
	doc := Parse("inline.mpd", []byte("_outer *strong*\n"))
	if got := countKind(doc, KindStrong); got != 1 {
		t.Fatalf("strong nodes = %d, want 1", got)
	}
	assertTreeLinks(t, doc)
}

func TestParseDoesNotCopySource(t *testing.T) {
	source := []byte("Hello world.\n")
	doc := Parse("minimal.mpd", source)
	if len(doc.Source) == 0 || &doc.Source[0] != &source[0] {
		t.Fatal("parser copied its immutable input")
	}
}

func TestMetadataLimitReportsAndRecovers(t *testing.T) {
	source := []byte("---\nschema = 1\ntitle = \"Limited\"\ndescription = \"Still scanned\"\n---\n\nBody.\n")
	doc := ParseWithOptions("limit.mpd", source, Options{MaxAttributes: 1})
	if !hasDiagnostic(doc, "mpd-limit") {
		t.Fatalf("missing limit diagnostic: %#v", doc.Diagnostics)
	}
	if got := countKind(doc, KindParagraph); got != 1 {
		t.Fatalf("paragraphs after limited metadata = %d, want 1", got)
	}
	if got := len(doc.Attributes); got != 1 {
		t.Fatalf("retained attributes = %d, want 1", got)
	}
}

func TestEncodingLineEndingsAndDefensiveLimits(t *testing.T) {
	for name, ending := range map[string]string{"lf": "\n", "crlf": "\r\n", "cr": "\r"} {
		t.Run(name, func(t *testing.T) {
			source := []byte("# Heading" + ending + ending + "first" + ending + "second" + ending)
			doc := Parse(name+".mpd", source)
			if len(doc.Diagnostics) != 0 || countKind(doc, KindHardBreak) != 1 {
				t.Fatalf("diagnostics=%#v hard-breaks=%d", doc.Diagnostics, countKind(doc, KindHardBreak))
			}
		})
	}
	invalid := Parse("utf8.mpd", []byte{'o', 'k', '\r', '\n', 0xff, '\n'})
	if !hasDiagnostic(invalid, "mpd-utf8") {
		t.Fatalf("missing UTF-8 diagnostic: %#v", invalid.Diagnostics)
	}
	if got := invalid.Diagnostics[0].Position; got != (Position{Line: 2, Column: 1}) {
		t.Fatalf("invalid UTF-8 position = %#v, want line 2 column 1", got)
	}
	tooLarge := ParseWithOptions("large.mpd", []byte("\xef\xbb\xbf12345"), Options{MaxFileSize: 4})
	if !hasDiagnostic(tooLarge, "mpd-limit") {
		t.Fatalf("missing file-size diagnostic: %#v", tooLarge.Diagnostics)
	}
	deep := ParseWithOptions("deep.mpd", []byte("@note\n@note\ntext\n@end\n@end\n"), Options{MaxNesting: 1})
	if !hasDiagnostic(deep, "mpd-limit") {
		t.Fatalf("missing nesting diagnostic: %#v", deep.Diagnostics)
	}
}

func TestStructuralIndentAndRawHTMLAttributes(t *testing.T) {
	for name, source := range map[string]string{
		"odd-list":  " - item\n",
		"tab-list":  "\t- item\n",
		"deep-list": "- parent\n    - too deep\n",
	} {
		t.Run(name, func(t *testing.T) {
			doc := Parse(name+".mpd", []byte(source))
			if !hasDiagnostic(doc, "mpd-indent") {
				t.Fatalf("missing indent diagnostic: %#v", doc.Diagnostics)
			}
		})
	}
	raw := Parse("raw.mpd", []byte("@rawHTML mode=\"unsafe\"\n<p>text</p>\n@end\n"))
	if !hasDiagnostic(raw, "mpd-attribute") {
		t.Fatalf("missing raw HTML attribute diagnostic: %#v", raw.Diagnostics)
	}
}

func TestNestingLimitsApplyToAllRecursiveSyntax(t *testing.T) {
	tests := map[string]string{
		"quote":  "> > > deeply quoted\n",
		"list":   "- one\n  - two\n    - three\n      - four\n",
		"inline": "_*_*_*nested\n",
		"role":   "@mark[@mark[@mark[nested]]]\n",
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			doc := ParseWithOptions(name+".mpd", []byte(source), Options{MaxNesting: 2})
			if !hasDiagnostic(doc, "mpd-limit") {
				t.Fatalf("missing nesting limit diagnostic: %#v", doc.Diagnostics)
			}
			if doc.Nodes[doc.Root].Source.End != uint32(len(source)) {
				t.Fatal("parser did not retain the complete source")
			}
			assertTreeLinks(t, doc)
		})
	}
}

func TestTruncatedTaskMarkerDoesNotPanic(t *testing.T) {
	doc := Parse("task.mpd", []byte("- [ ]\n"))
	if doc == nil || len(doc.Nodes) == 0 {
		t.Fatal("parser did not return a document")
	}
	assertTreeLinks(t, doc)
}

func TestIndentedQuoteAlwaysAdvances(t *testing.T) {
	doc := Parse("quote.mpd", []byte("     > quoted\n"))
	if got := countKind(doc, KindQuote); got != 1 {
		t.Fatalf("quotes = %d, want 1", got)
	}
	if got := doc.Nodes[doc.Root].Source.End; got != uint32(len(doc.Source)) {
		t.Fatalf("root source end = %d, want %d", got, len(doc.Source))
	}
}

func TestIndentedListAlwaysAdvances(t *testing.T) {
	doc := Parse("list.mpd", []byte("      - item\n"))
	for _, node := range doc.Nodes {
		if node.Kind == KindList {
			if node.Level != 0 || node.Position.Column != 7 {
				t.Fatalf("indented list level=%d position=%#v", node.Level, node.Position)
			}
			return
		}
	}
	t.Fatal("parser did not produce a list")
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{"", "Hello", "- [ ]\n", "@note\n@end\n", "```\n@end\n```", "---\nschema = 1\n---\n", strings.Repeat("@", 64)} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, source []byte) {
		doc := Parse("fuzz.mpd", source)
		if doc == nil || len(doc.Nodes) == 0 || doc.Root != 0 {
			t.Fatal("parser did not return a root document")
		}
		if doc.Nodes[0].Source.End != uint32(len(source)) {
			t.Fatal("root does not cover source")
		}
	})
}

func countKind(doc *Document, kind Kind) int {
	count := 0
	for _, node := range doc.Nodes {
		if node.Kind == kind {
			count++
		}
	}
	return count
}

func hasDiagnostic(doc *Document, code string) bool {
	for _, diagnostic := range doc.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func assertTreeLinks(t *testing.T, doc *Document) {
	t.Helper()
	seen := make([]bool, len(doc.Nodes))
	var walk func(uint32)
	walk = func(index uint32) {
		if int(index) >= len(doc.Nodes) {
			t.Fatalf("node index %d is out of range", index)
		}
		if seen[index] {
			t.Fatalf("node %d is linked more than once", index)
		}
		seen[index] = true
		node := &doc.Nodes[index]
		if node.Source.End < node.Source.Start || int(node.Source.End) > len(doc.Source) {
			t.Fatalf("node %d (%s) has invalid source range %#v", index, node.Kind, node.Source)
		}
		for label, sourceRange := range map[string]Range{"content": node.Content, "name": node.Name} {
			if sourceRange.End < sourceRange.Start || int(sourceRange.End) > len(doc.Source) {
				t.Fatalf("node %d (%s) has invalid %s range %#v", index, node.Kind, label, sourceRange)
			}
		}
		if node.Position.Line == 0 || node.Position.Column == 0 {
			t.Fatalf("node %d (%s) has invalid position %#v", index, node.Kind, node.Position)
		}
		for child := node.FirstChild; child != noIndex; child = doc.Nodes[child].NextSibling {
			childRange := doc.Nodes[child].Source
			if childRange.Start < node.Source.Start || childRange.End > node.Source.End {
				t.Fatalf("child %d range %#v is outside parent %d range %#v", child, childRange, index, node.Source)
			}
			walk(child)
		}
	}
	walk(doc.Root)
	for index, linked := range seen {
		if !linked {
			t.Fatalf("node %d (%s) is unreachable from the root", index, doc.Nodes[index].Kind)
		}
	}
}
