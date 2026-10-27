package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

// countMPDComponents counts named component nodes in a parsed document.
func countMPDComponents(document *mpd.Document) map[string]int {
	counts := make(map[string]int)
	var walk func(uint32)
	walk = func(index uint32) {
		node := document.Nodes[index]
		if node.Kind == mpd.KindComponent {
			counts[string(document.Text(node.Name))]++
		}
		children := document.Children(index)
		for {
			child, _, ok := children.Next()
			if !ok {
				return
			}
			walk(child)
		}
	}
	walk(document.Root)
	return counts
}

// mpdCodeText joins the content of every code node in a parsed document.
func mpdCodeText(document *mpd.Document) string {
	var texts []string
	for index := range document.Nodes {
		node := &document.Nodes[index]
		if node.Kind == mpd.KindCode {
			texts = append(texts, string(document.Text(node.Content)))
		}
	}
	return strings.Join(texts, "\n")
}

// Fenced examples are opaque: directive-looking lines inside them are code,
// not live structure. Both the @component and portable ::: forms must convert.
func TestMarkdownToMPDKeepsFencedDirectiveExamplesLiteral(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "at-component with backtick fence",
			source: "@note{type=tip}\n\n```text\n:::note\nliteral opening directive\n:::\n```\n\n@end\n",
		},
		{
			name:   "portable directive with backtick fence",
			source: ":::note{type=tip}\n\n```text\n:::note\nliteral opening directive\n:::\n```\n\n:::\n",
		},
		{
			name:   "portable directive with tilde fence",
			source: ":::note{type=tip}\n\n~~~text\n:::note\nliteral closing directive\n:::\n~~~\n\n:::\n",
		},
		{
			name:   "at-component with longer fence around a shorter fence",
			source: "@note{type=tip}\n\n````text\n```text\n:::note\n:::\n```\n````\n\n@end\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			native, _, html := renderImportedTable(t, test.source)
			document := mpd.Parse("imported.mpd", []byte(native))

			components := countMPDComponents(document)
			if got := components["note"]; got != 1 {
				t.Fatalf("parsed %d note components, want 1: %#v\n%s", got, components, native)
			}
			code := mpdCodeText(document)
			for _, want := range []string{":::note", "\n:::"} {
				if !strings.Contains(code, want) {
					t.Errorf("code text missing literal %q:\n%s", want, code)
				}
			}
			if !strings.Contains(html, "mpress-admonition") {
				t.Errorf("rendered HTML has no admonition:\n%s", html)
			}
			if !strings.Contains(html, ":::note") {
				t.Errorf("rendered HTML lost the literal fenced example:\n%s", html)
			}
		})
	}
}

// Tab labels and directives inside fences must not split or nest live tabs.
func TestMarkdownToMPDKeepsFencedTabLabelsAndDirectivesLiteral(t *testing.T) {
	backtickFence := "```text\n[Not a tab]\n:::note\nliteral directive\n:::\n```"
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "portable tabs with backtick fence",
			source: ":::tabs\n[First]\n\n" + backtickFence + "\n\n[Second]\n\nSecond body.\n:::\n",
		},
		{
			name:   "at-component tabs with backtick fence",
			source: "@tabs\n[First]\n\n" + backtickFence + "\n\n[Second]\n\nSecond body.\n@end\n",
		},
		{
			name:   "portable tabs with tilde fence",
			source: ":::tabs\n[First]\n\n~~~text\n[Not a tab]\n:::note\nliteral directive\n:::\n~~~\n\n[Second]\n\nSecond body.\n:::\n",
		},
		{
			name:   "portable tabs with longer fence around a shorter fence",
			source: ":::tabs\n[First]\n\n````text\n```text\n[Not a tab]\n:::note\n:::\n```\n````\n\n[Second]\n\nSecond body.\n:::\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			native, _, html := renderImportedTable(t, test.source)
			document := mpd.Parse("imported.mpd", []byte(native))

			components := countMPDComponents(document)
			if got := components["tabs"]; got != 1 {
				t.Fatalf("parsed %d tabs components, want 1: %#v\n%s", got, components, native)
			}
			if got := components["tab"]; got != 2 {
				t.Fatalf("parsed %d tab components, want 2: %#v\n%s", got, components, native)
			}
			if got := components["note"]; got != 0 {
				t.Fatalf("parsed %d note components from fenced text, want 0: %#v\n%s", got, components, native)
			}
			code := mpdCodeText(document)
			for _, want := range []string{"[Not a tab]", ":::note"} {
				if !strings.Contains(code, want) {
					t.Errorf("code text missing literal %q:\n%s", want, code)
				}
			}
			if got := strings.Count(html, `role="tab"`); got != 2 {
				t.Errorf("rendered %d tabs, want 2:\n%s", got, html)
			}
			if !strings.Contains(html, "[Not a tab]") {
				t.Errorf("rendered HTML lost the literal tab label:\n%s", html)
			}
		})
	}
}

// Real nesting outside fences keeps working in both grammars.
func TestMarkdownToMPDConvertsNestedDirectivesAndTabs(t *testing.T) {
	backtickFence := "```text\n[Not a tab]\n:::note\n:::\n```"
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "portable nesting",
			source: ":::note{type=tip}\nOuter body.\n\n:::tabs\n[Alpha]\n\nAlpha body.\n\n[Beta]\n\n" + backtickFence + "\n:::\n:::\n",
		},
		{
			name:   "at-component nesting",
			source: "@note{type=tip}\nOuter body.\n\n@tabs\n[Alpha]\n\nAlpha body.\n\n[Beta]\n\n" + backtickFence + "\n@end\n@end\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			native, _, html := renderImportedTable(t, test.source)
			document := mpd.Parse("imported.mpd", []byte(native))

			components := countMPDComponents(document)
			if got := components["note"]; got != 1 {
				t.Fatalf("parsed %d note components, want 1: %#v\n%s", got, components, native)
			}
			if got := components["tabs"]; got != 1 {
				t.Fatalf("parsed %d tabs components, want 1: %#v\n%s", got, components, native)
			}
			if got := components["tab"]; got != 2 {
				t.Fatalf("parsed %d tab components, want 2: %#v\n%s", got, components, native)
			}
			if got := strings.Count(html, `role="tab"`); got != 2 {
				t.Errorf("rendered %d tabs, want 2:\n%s", got, html)
			}
			if !strings.Contains(mpdCodeText(document), "[Not a tab]") {
				t.Errorf("nested fenced example was not preserved as code:\n%s", native)
			}
		})
	}
}

// A fence cannot act as the closing delimiter of a directive; malformed input
// is still refused rather than silently accepting code text as structure.
func TestMarkdownToMPDStillRejectsUnclosedDirectiveWithFencedExample(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"missing close", ":::note{type=tip}\n\nBody without a closing directive.\n"},
		{"missing component end", "@note{type=tip}\n\nBody without @end.\n"},
		{"fenced close is not a delimiter", ":::note{type=tip}\n\n```text\n:::\n```\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := MarkdownToMPD(test.source); err == nil {
				t.Fatal("converter accepted an unclosed directive")
			} else if !strings.Contains(err.Error(), "unclosed MPress Markdown directive") {
				t.Fatalf("converter raised %q, want the unclosed-directive error", err)
			}
		})
	}
}

func TestMarkdownToMPDKeepsUnbalancedFencedDirectivesLiteral(t *testing.T) {
	for _, literal := range []string{":::note", ":::", ":::endtabs", ":::note\n:::warning\n:::"} {
		t.Run(literal, func(t *testing.T) {
			source := "@note{type=tip}\n\n```text\n" + literal + "\n```\n\n@end\n"
			native, _, _ := renderImportedTable(t, source)
			document := mpd.Parse("imported.mpd", []byte(native))
			if got := countMPDComponents(document)["note"]; got != 1 {
				t.Fatalf("parsed %d real note components, want 1", got)
			}
			if got := strings.TrimSuffix(mpdCodeText(document), "\n"); got != literal {
				t.Fatalf("code example changed: got %q, want %q", got, literal)
			}
		})
	}
}
