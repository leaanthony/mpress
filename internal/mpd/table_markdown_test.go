package mpd_test

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/mpd"
)

func renderNativeMPD(t *testing.T, source string) (markdown, html string) {
	t.Helper()
	document := mpd.Parse("table.mpd", []byte(source))
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("native MPD is invalid: %s at line %d: %s\n%s", diagnostic.Code, diagnostic.Position.Line, diagnostic.Message, source)
		}
	}
	rendered, err := mpd.Markdown(document)
	if err != nil {
		t.Fatal(err)
	}
	page, diagnostics, err := content.NewRenderer().ParseBytes("table.md", "en", rendered)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			t.Fatalf("render diagnostic %s: %s\n%s", diagnostic.Code, diagnostic.Message, rendered)
		}
	}
	return string(rendered), page.HTML
}

func countMarkdownTableRows(markdown string) int {
	count := 0
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			count++
		}
	}
	return count
}

func TestMarkdownRichTableKeepsExactRowsAndControls(t *testing.T) {
	source := `@table header=true search=true filter=true sort=true paginate=true column-separators=true page-size=2
| Package | Platform | Downloads |
| Core | Linux | 1840 |
| Studio | macOS | 920 |
| Core | Windows | 1260 |
@end
`
	markdown, html := renderNativeMPD(t, source)
	if strings.Contains(markdown, "| --- |") {
		t.Errorf("rich table rendered a Markdown separator row:\n%s", markdown)
	}
	if got := countMarkdownTableRows(markdown); got != 4 {
		t.Errorf("Markdown table rows = %d, want 4 (three authored data rows and one header)\n%s", got, markdown)
	}
	if got := strings.Count(html, "data-table-row"); got != 3 {
		t.Errorf("rendered body rows = %d, want 3\n%s", got, html)
	}
	if got := strings.Count(html, "<tr"); got != 4 {
		t.Errorf("rendered rows = %d, want 4 (one header, three data rows)\n%s", got, html)
	}
	if got := strings.Count(html, "<td"); got != 9 {
		t.Errorf("rendered cells = %d, want 9\n%s", got, html)
	}
	for _, want := range []string{
		`data-table-search="true"`,
		`data-table-filter="true"`,
		`data-table-sort="true"`,
		`data-table-paginate="true"`,
		`data-table-column-separators="true"`,
		`data-table-page-size="2"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered table missing %q\n%s", want, html)
		}
	}
}

func TestMarkdownTableBooleanFalseDoesNotEnableControls(t *testing.T) {
	markdown, html := renderNativeMPD(t, `@table header=true search=false filter=false sort=false paginate=false column-separators=false
| Name | State |
| First | Published |
@end
`)
	if strings.Contains(markdown, "@table") {
		t.Errorf("all-disabled controls still produced a rich table:\n%s", markdown)
	}
	if strings.Contains(html, "data-table-query") {
		t.Errorf("search=false rendered a search control:\n%s", html)
	}
	if strings.Contains(html, "data-table-filter-column") {
		t.Errorf("filter=false rendered filter controls:\n%s", html)
	}
	if got := strings.Count(html, "<tr"); got != 2 {
		t.Errorf("rendered rows = %d, want 2\n%s", got, html)
	}
}

func TestMarkdownTableHeaderFalseStaysRichWithoutSeparator(t *testing.T) {
	markdown, html := renderNativeMPD(t, `@table header=false
| Name | State |
| First | Published |
| Second | In review |
@end
`)
	if !strings.Contains(markdown, "@table") {
		t.Errorf("header=false lost the explicit component form:\n%s", markdown)
	}
	if strings.Contains(markdown, "| --- |") {
		t.Errorf("header=false table gained a Markdown separator row:\n%s", markdown)
	}
	if strings.Contains(html, "<thead") {
		t.Errorf("header=false rendered a header row:\n%s", html)
	}
	if got := strings.Count(html, "data-table-row"); got != 3 {
		t.Errorf("rendered body rows = %d, want 3\n%s", got, html)
	}
}

func TestMarkdownStaticTableKeepsGFMSeparator(t *testing.T) {
	markdown, html := renderNativeMPD(t, `@table header=true align=["left","center","right"]
| Capability | Support | Time |
| Plain text | *Complete* | `+"`4 ms`"+` |
@end
`)
	if !strings.Contains(markdown, "| :--- | :---: | ---: |") {
		t.Errorf("static table lost its GFM alignment separator:\n%s", markdown)
	}
	if strings.Contains(markdown, "@table") {
		t.Errorf("static table unexpectedly used the rich component form:\n%s", markdown)
	}
	if got := strings.Count(html, "<tr"); got != 2 {
		t.Errorf("rendered rows = %d, want 2\n%s", got, html)
	}
}
