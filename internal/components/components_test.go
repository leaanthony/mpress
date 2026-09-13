package components

import (
	"strings"
	"testing"
)

// TestNestedNoteInsideTabs verifies that a :::note{type="warning"}...:::
// nested inside :::tabs...:::  is handled correctly.  Before the fix the
// regex was non-greedy and the inner ::: terminated the outer tabs block,
// leaving subsequent [Label] sections outside the tab container.
func TestNestedNoteInsideTabs(t *testing.T) {
	input := `:::tabs

[Bugs]

If you find a bug, let us know.

:::note{type="warning"}
Unexpected behavior isn't necessarily a bug.
:::

[Fixes]

Open a pull request.

[Suggestions]

Post a suggestion.

:::
`
	result, warnings := Process(input)
	t.Logf("warnings: %v", warnings)
	t.Logf("result:\n%s", result)

	// All four tab labels must be present as tab buttons (inside the tablist).
	for _, label := range []string{"Bugs", "Fixes", "Suggestions"} {
		if !strings.Contains(result, `>`+label+`</button>`) {
			t.Errorf("tab button %q not found in output", label)
		}
	}

	// The admonition content must appear.
	if !strings.Contains(result, "Unexpected behavior") {
		t.Error("admonition content 'Unexpected behavior' missing from output")
	}

	// Raw ::: must not appear in the output.
	if strings.Contains(result, ":::") {
		t.Error("raw ::: found in output — component not fully processed")
	}

	// [Fixes] and [Suggestions] must NOT appear as plain paragraphs outside the tabs.
	if strings.Contains(result, "<p>[Fixes]</p>") {
		t.Error("[Fixes] rendered as plain paragraph — tabs closed too early by inner :::")
	}
	if strings.Contains(result, "<p>[Suggestions]</p>") {
		t.Error("[Suggestions] rendered as plain paragraph — tabs closed too early by inner :::")
	}
}

// TestSimpleTabs verifies basic tabs rendering is unaffected by the fix.
func TestSimpleTabs(t *testing.T) {
	input := `:::tabs

[Alpha]

Content in Alpha.

[Beta]

Content in Beta.

:::
`
	result, warnings := Process(input)
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}

	if !strings.Contains(result, `>Alpha</button>`) {
		t.Error("tab button 'Alpha' not found")
	}
	if !strings.Contains(result, `>Beta</button>`) {
		t.Error("tab button 'Beta' not found")
	}
	if !strings.Contains(result, "Content in Alpha") {
		t.Error("'Content in Alpha' not found in output")
	}
	if !strings.Contains(result, "Content in Beta") {
		t.Error("'Content in Beta' not found in output")
	}
	if strings.Contains(result, `id="tabs"`) || !strings.Contains(result, `aria-controls="mpress-tabs-`) || !strings.Contains(result, `aria-labelledby="mpress-tabs-`) {
		t.Errorf("tabs did not render unique ARIA relationships: %s", result)
	}
}

// TestSimpleNote verifies plain admonition rendering is unaffected by the fix.
func TestSimpleNote(t *testing.T) {
	input := `:::note{type="warning"}
Watch out!
:::
`
	result, _ := Process(input)
	if strings.Contains(result, ":::") {
		t.Error("raw ::: in output — note not processed")
	}
	if !strings.Contains(result, "Watch out") {
		t.Error("note content missing from output")
	}
}

// TestContentAfterTabs verifies that content after the closing ::: of a tabs
// block is not swallowed into the last tab.
func TestContentAfterTabs(t *testing.T) {
	input := `:::tabs

[Only]

Tab content.

:::

Paragraph after tabs.
`
	result, _ := Process(input)
	t.Logf("result:\n%s", result)

	if !strings.Contains(result, "Tab content") {
		t.Error("tab content missing")
	}
	if !strings.Contains(result, "Paragraph after tabs") {
		t.Error("content after closing ::: was swallowed into the last tab")
	}
}

func TestComponentSyntaxInsideNestedFenceStaysLiteral(t *testing.T) {
	input := ":::details{title=\"Show syntax\"}\n```md\n:::note{type=\"tip\"}\nLiteral example.\n:::\n```\n:::\n"
	result, warnings := Process(input)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if !strings.Contains(result, "```md\n:::note{type=\"tip\"}\nLiteral example.\n:::\n```") {
		t.Fatalf("fenced component example was not preserved: %s", result)
	}
	if strings.Contains(result, "mpress-admonition") {
		t.Fatalf("fenced component example was incorrectly rendered: %s", result)
	}
}
