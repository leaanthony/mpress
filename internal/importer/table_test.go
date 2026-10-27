package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/mpd"
)

// renderImportedTable converts Markdown into native MPD, renders it through the
// production content pipeline, and returns both interchange stages.
func renderImportedTable(t *testing.T, source string) (native, markdown, html string) {
	t.Helper()
	converted, err := MarkdownToMPD(source)
	if err != nil {
		t.Fatalf("MarkdownToMPD failed: %v", err)
	}
	document := mpd.Parse("imported.mpd", []byte(converted))
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("imported MPD is invalid: %s at line %d: %s\n%s", diagnostic.Code, diagnostic.Position.Line, diagnostic.Message, converted)
		}
	}
	rendered, err := mpd.Markdown(document)
	if err != nil {
		t.Fatalf("mpd.Markdown failed: %v\n%s", err, converted)
	}
	page, diagnostics, err := content.NewRenderer().ParseBytes("imported.md", "en", rendered)
	if err != nil {
		t.Fatalf("render failed: %v\n%s", err, rendered)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			t.Fatalf("render diagnostic %s: %s\n%s", diagnostic.Code, diagnostic.Message, rendered)
		}
	}
	return converted, string(rendered), page.HTML
}

func TestMarkdownToMPDRichTableKeepsFlagsAndRowBoundaries(t *testing.T) {
	source := `# Table

@table{header search filter sort paginate column-separators page-size=1}
| Name | State |
| First | Published |
| Second | In review |
@end
`
	native, markdown, html := renderImportedTable(t, source)

	for _, want := range []string{
		`@table header search filter sort paginate column-separators page-size="1"`,
		"| Name | State |",
		"| First | Published |",
		"| Second | In review |",
	} {
		if !strings.Contains(native, want) {
			t.Errorf("converted MPD missing %q:\n%s", want, native)
		}
	}
	for _, line := range strings.Split(native, "\n") {
		if strings.Contains(line, "|") && strings.HasSuffix(strings.TrimRight(line, " \t"), `\`) {
			t.Errorf("table row kept a soft-wrap backslash: %q\n%s", line, native)
		}
	}
	if strings.Contains(markdown, "| --- |") {
		t.Errorf("rich table gained a Markdown separator row:\n%s", markdown)
	}
	if got := strings.Count(html, "data-table-row"); got != 2 {
		t.Errorf("rendered body rows = %d, want 2 rows (header excluded)\n%s", got, html)
	}
	if got := strings.Count(html, "<tr"); got != 3 {
		t.Errorf("rendered rows = %d, want 3 (one header, two data rows)\n%s", got, html)
	}
	for _, want := range []string{
		`data-table-search="true"`,
		`data-table-filter="true"`,
		`data-table-sort="true"`,
		`data-table-paginate="true"`,
		`data-table-column-separators="true"`,
		`data-table-page-size="1"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered table missing %q\n%s", want, html)
		}
	}
}

func TestMarkdownToMPDRichTableKeepsInlineFormattingAndCodePipes(t *testing.T) {
	source := `@table{header sort}
| Capability | Command |
| Plain | *Complete* |
| Code pipe | ` + "`a|b`" + ` |
| Link | [Docs](/guide) |
@end
`
	native, _, html := renderImportedTable(t, source)
	if !strings.Contains(native, "| Code pipe | `a\\|b` |") {
		t.Errorf("code pipe was not escaped for the table grammar:\n%s", native)
	}
	for _, want := range []string{
		"<em>Complete</em>",
		"<code>a|b</code>",
		`href="/guide"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered table missing %q\n%s", want, html)
		}
	}
	if got := strings.Count(html, "data-table-row"); got != 3 {
		t.Errorf("rendered body rows = %d, want 3\n%s", got, html)
	}
}

func TestMarkdownToMPDTableBooleanMetadataIsExplicit(t *testing.T) {
	native, _, html := renderImportedTable(t, "@table{header=true search=false filter=false sort=false paginate=false}\n| Name | State |\n| First | Published |\n@end\n")
	for _, want := range []string{`search="false"`, `filter="false"`, `sort="false"`, `paginate="false"`} {
		if !strings.Contains(native, want) {
			t.Errorf("converted MPD missing %q:\n%s", want, native)
		}
	}
	if strings.Contains(html, "data-table-query") {
		t.Errorf("search=false rendered a search control:\n%s", html)
	}
	if strings.Contains(html, "data-table-filter-column") {
		t.Errorf("filter=false rendered filter controls:\n%s", html)
	}
	if got := strings.Count(html, "<tr"); got != 2 {
		t.Errorf("rendered rows = %d, want 2 (one header, one data row)\n%s", got, html)
	}
}

func TestMarkdownToMPDNestedRichTableInContainer(t *testing.T) {
	source := `@details{title="Data"}
@table{header search}
| Name | State |
| First | Published |
@end
@end
`
	native, _, html := renderImportedTable(t, source)
	if !strings.Contains(native, "@details") || !strings.Contains(native, "@table header search") {
		t.Errorf("converted MPD lost the nested table:\n%s", native)
	}
	if got := strings.Count(html, "data-table-row"); got != 1 {
		t.Errorf("nested table body rows = %d, want 1\n%s", got, html)
	}
	if !strings.Contains(html, `data-table-search="true"`) {
		t.Errorf("nested table lost search metadata:\n%s", html)
	}
}

func TestMarkdownToMPDKeepsLiteralTableExamples(t *testing.T) {
	source := "```markdown\n@table{header search}\n| Name | State |\n@end\n```\n\nUse `@table{header}` in prose.\n"
	native, _, _ := renderImportedTable(t, source)
	if !strings.Contains(native, "@table{header search}\n| Name | State |\n@end") {
		t.Errorf("fenced table example was rewritten:\n%s", native)
	}
	if !strings.Contains(native, "`@table{header}`") {
		t.Errorf("inline table example was rewritten:\n%s", native)
	}
}

func TestMarkdownToMPDStaticGFMTableIsUnchanged(t *testing.T) {
	native, markdown, html := renderImportedTable(t, "| Left | Centre | Right |\n| :--- | :----: | ----: |\n| one | two | three |\n")
	if !strings.Contains(native, `@table header=true align=["left","center","right"]`) {
		t.Errorf("static table metadata changed:\n%s", native)
	}
	if !strings.Contains(markdown, "| :--- | :---: | ---: |") {
		t.Errorf("static table lost its GFM alignment separator:\n%s", markdown)
	}
	if got := strings.Count(html, "<tr"); got != 2 {
		t.Errorf("rendered static table rows = %d, want 2\n%s", got, html)
	}
}

func TestMarkdownToMPDRichTableRoundTripIsStable(t *testing.T) {
	source := `@table header search filter sort paginate column-separators page-size="1"
| Name | State |
| First | Published |
| Second | In review |
@end
`
	first := mpd.Parse("roundtrip.mpd", []byte(source))
	for _, diagnostic := range first.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("source MPD is invalid: %s: %s", diagnostic.Code, diagnostic.Message)
		}
	}
	portable, err := mpd.Markdown(first)
	if err != nil {
		t.Fatal(err)
	}
	reimported, err := MarkdownToMPD(string(portable))
	if err != nil {
		t.Fatal(err)
	}
	second := mpd.Parse("roundtrip.mpd", []byte(reimported))
	for _, diagnostic := range second.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("re-imported MPD is invalid: %s: %s\n%s", diagnostic.Code, diagnostic.Message, reimported)
		}
	}
	rendered, err := mpd.Markdown(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(rendered) != string(portable) {
		t.Errorf("round trip changed the interchange form:\nfirst:\n%s\nsecond:\n%s", portable, rendered)
	}
	if got := strings.Count(string(portable), "data-table"); got != 0 {
		t.Errorf("portable Markdown unexpectedly contains rendered table markup")
	}
	if !strings.Contains(reimported, "@table header search filter sort paginate column-separators page-size=\"1\"") {
		t.Errorf("round trip lost table metadata:\n%s", reimported)
	}
}

func TestMarkdownToMPDQuotedTableAttributeValues(t *testing.T) {
	native, _, _ := renderImportedTable(t, `@table{header title="A | B" align=["left","right"]}
| Name | State |
| First | Published |
@end
`)
	for _, want := range []string{`title="A | B"`, `align="[\"left\",\"right\"]"`} {
		if !strings.Contains(native, want) {
			t.Errorf("converted MPD missing quoted value %q:\n%s", want, native)
		}
	}
}
