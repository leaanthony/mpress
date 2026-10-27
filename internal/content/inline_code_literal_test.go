package content

import (
	"strings"
	"testing"
)

func TestInlineCodeKeepsComponentSyntaxInert(t *testing.T) {
	source := "# Guide\n\n" +
		"Intro `@button[Inline](#){primary}` here.\n\n" +
		"- List item `@image{src=literal.png}`\n" +
		"  - Nested `` `@video{src=literal.mp4}` ``\n\n" +
		"> Quote `{badge.primary:Literal}` here.\n" +
		"> > Nested @button[Quote](/quote/){secondary}\n\n" +
		"```md\n@button[Fenced](#){primary}\n```\n\n" +
		"Then @button[Live](/live/){primary} and @image{src=live.png alt=Live image}.\n"
	page, diagnostics, err := NewRenderer().Parse("guide.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			t.Fatalf("unexpected diagnostic: %#v", diagnostic)
		}
	}
	for _, literal := range []string{
		"<code>@button[Inline](#){primary}</code>",
		"<code>@image{src=literal.png}</code>",
		"<code>`@video{src=literal.mp4}`</code>",
		"<code>{badge.primary:Literal}</code>",
		"@button[Fenced](#){primary}",
	} {
		if !strings.Contains(page.HTML, literal) {
			t.Errorf("literal example %q was rewritten:\n%s", literal, page.HTML)
		}
	}
	if !strings.Contains(page.HTML, `<a class="mpress-button mpress-button-primary" href="/live/">Live</a>`) {
		t.Errorf("adjacent live button was lost:\n%s", page.HTML)
	}
	if !strings.Contains(page.HTML, `<a class="mpress-button mpress-button-secondary" href="/quote/">Quote</a>`) {
		t.Errorf("nested live button was lost:\n%s", page.HTML)
	}
	if !strings.Contains(page.HTML, `src="live.png"`) {
		t.Errorf("adjacent live image was lost:\n%s", page.HTML)
	}
	if strings.Contains(page.HTML, "mpress-badge") {
		t.Errorf("literal badge directive was rendered:\n%s", page.HTML)
	}
}

func TestInlineCodeDoesNotPairAcrossBlocks(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "separate paragraphs",
			source: "# Guide\n\nAn unmatched ` opener.\n\nUse @button[Live](/guide/){primary} here.\n\nA separate ` paragraph.\n",
		},
		{
			name:   "escaped opener",
			source: "# Guide\n\nEscaped \\` then @button[Live](/guide/){primary} and unmatched `.\n",
		},
		{
			name:   "heading and list transitions",
			source: "# Guide ` heading\n\n- Item ` one\n\n1. Item ` two\n\n> Quote ` three\n\nUse @button[Live](/guide/){primary} here.\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, diagnostics, err := NewRenderer().Parse("guide.md", "en", test.source)
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %#v", diagnostics)
			}
			if !strings.Contains(page.HTML, `<a class="mpress-button mpress-button-primary" href="/guide/">Live</a>`) {
				t.Fatalf("unmatched punctuation hid live syntax:\n%s", page.HTML)
			}
		})
	}
}

func TestInlineCodeContinuesInsideContainer(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "blockquote continuation",
			source: "# Guide\n\n> Start `example\n> @button[Literal](#){primary}` ends.\n",
		},
		{
			name:   "list continuation",
			source: "# Guide\n\n- Start `example\n  @button[Literal](#){primary}` ends.\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, diagnostics, err := NewRenderer().Parse("guide.md", "en", test.source)
			if err != nil {
				t.Fatal(err)
			}
			if len(diagnostics) != 0 {
				t.Fatalf("unexpected diagnostics: %#v", diagnostics)
			}
			if !strings.Contains(page.HTML, "<code>example @button[Literal](#){primary}</code>") {
				t.Fatalf("container code span lost its literal directive:\n%s", page.HTML)
			}
			if strings.Contains(page.HTML, `class="mpress-button`) {
				t.Fatalf("container code span was expanded live:\n%s", page.HTML)
			}
		})
	}
}

func TestNativeTableRowsKeepInlineCodeScopesSeparate(t *testing.T) {
	source := "# Guide\n\n@table{header}\n| Name | Action |\n| First ` | @button[Live](/guide/){primary} |\n| Last ` | Text |\n@end\n"
	page, diagnostics, err := NewRenderer().Parse("guide.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if !strings.Contains(page.HTML, `<a class="mpress-button mpress-button-primary" href="/guide/">Live</a>`) {
		t.Fatalf("live table action was hidden:\n%s", page.HTML)
	}
	if count := strings.Count(page.HTML, "data-table-row"); count != 2 {
		t.Fatalf("expected two data rows, got %d:\n%s", count, page.HTML)
	}
	for _, want := range []string{"First", "Last", "Text"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("table output missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestNativeTableCellKeepsLiteralComponentSyntax(t *testing.T) {
	source := "# Guide\n\n@table{header}\n| Name | Example |\n| First | `@button[Literal](#){primary}` |\n@end\n"
	page, diagnostics, err := NewRenderer().Parse("guide.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if !strings.Contains(page.HTML, "<code>@button[Literal](#){primary}</code>") {
		t.Fatalf("table cell example was not literal:\n%s", page.HTML)
	}
	if strings.Contains(page.HTML, `class="mpress-button`) {
		t.Fatalf("table cell example was rendered live:\n%s", page.HTML)
	}
	if count := strings.Count(page.HTML, "data-table-row"); count != 1 {
		t.Fatalf("expected one data row, got %d:\n%s", count, page.HTML)
	}
}

func TestAuthoredInlineCodePlaceholderIsPreserved(t *testing.T) {
	source := "# Guide\n\nAuthored @@MPRESS_INLINECODE_000000@@ remains; code `@button[Literal](#){primary}` remains.\n"
	page, diagnostics, err := NewRenderer().Parse("guide.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if !strings.Contains(page.HTML, "@@MPRESS_INLINECODE_000000@@") {
		t.Fatalf("authored placeholder text was overwritten:\n%s", page.HTML)
	}
	if !strings.Contains(page.HTML, "<code>@button[Literal](#){primary}</code>") {
		t.Fatalf("inline component example was not literal:\n%s", page.HTML)
	}
	if strings.Contains(page.HTML, `class="mpress-button`) {
		t.Fatalf("inline component example was rendered live:\n%s", page.HTML)
	}
}

func TestInlineCodeDelimiterRunsAndEscapes(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "double backticks with internal single",
			source: "# Guide\n\nUse ``a ` @button[Literal](#){primary} b`` here.\n",
			want:   "<code>a ` @button[Literal](#){primary} b</code>",
		},
		{
			name:   "single backticks with internal double",
			source: "# Guide\n\nUse `a `` @button[Literal](#){primary} `` b` here.\n",
			want:   "<code>a `` @button[Literal](#){primary} `` b</code>",
		},
		{
			name:   "crlf source",
			source: "# Guide\r\n\r\nUse `@button[Läses](#){primary}` på svenska.\r\n",
			want:   "<code>@button[Läses](#){primary}</code>",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, _, err := NewRenderer().Parse("guide.md", "en", test.source)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(page.HTML, test.want) {
				t.Fatalf("missing %q:\n%s", test.want, page.HTML)
			}
			if strings.Contains(page.HTML, `class="mpress-button`) {
				t.Fatalf("inline component example was rendered live:\n%s", page.HTML)
			}
		})
	}
}

func TestLegacyAndMarkdownTableRowsKeepLiveActions(t *testing.T) {
	rows := "| Name | Action |\n| First ` | @button[Live](/guide/){primary} |\n| Last ` | Text |\n"
	for _, source := range []string{
		"# Guide\n\n:::table{header}\n" + rows + ":::\n",
		"# Guide\n\n" + strings.Replace(rows, "| Name | Action |\n", "| Name | Action |\n| --- | --- |\n", 1),
	} {
		page, diagnostics, err := NewRenderer().Parse("guide.md", "en", source)
		if err != nil || len(diagnostics) != 0 {
			t.Fatalf("parse: %v %#v", err, diagnostics)
		}
		if !strings.Contains(page.HTML, `class="mpress-button mpress-button-primary"`) {
			t.Fatalf("live table action was hidden: %s", page.HTML)
		}
		if strings.Count(page.HTML, "<tr") != 3 {
			t.Fatalf("table rows changed: %s", page.HTML)
		}
	}
}

func TestFencedTableExamplesKeepLiteralCellCode(t *testing.T) {
	source := "# Guide\n\n```md\n@table{header}\n| Name | Example |\n| First | `@button[Literal](#){primary}` |\n@end\n```\n"
	for _, variant := range []string{source, strings.ReplaceAll(strings.ReplaceAll(source, "@table{header}", ":::table{header}"), "@end", ":::")} {
		page, diagnostics, err := NewRenderer().Parse("guide.md", "en", variant)
		if err != nil || len(diagnostics) != 0 {
			t.Fatalf("parse: %v %#v", err, diagnostics)
		}
		if !strings.Contains(page.HTML, "@button[Literal](#){primary}") || strings.Contains(page.HTML, "MPRESS_INLINE_") || strings.ContainsRune(page.HTML, 0) {
			t.Fatalf("literal table cell changed: %s", page.HTML)
		}
	}
}
