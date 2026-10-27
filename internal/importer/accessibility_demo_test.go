package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

// The accessibility demo is a registered container. Markdown conversion must
// keep its own @end so following prose and neighbouring components stay outside
// it instead of being consumed as the enclosing section's body.
func TestMarkdownToMPDAccessibilityDemoStandaloneKeepsBoundaries(t *testing.T) {
	source := "# Reader controls\n\n@accessibility-demo\n@end\n\nFollowing prose after the demo.\n"
	native, _, html := renderImportedTable(t, source)

	document := mpd.Parse("imported.mpd", []byte(native))
	components := countMPDComponents(document)
	if got := components["accessibility-demo"]; got != 1 {
		t.Fatalf("parsed %d accessibility-demo components, want 1: %#v\n%s", got, components, native)
	}
	for _, want := range []string{
		`data-a11y-demo`,
		`data-a11y-choice="siteWidth"`,
		`data-open-accessibility`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered demo missing %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "@accessibility-demo") {
		t.Errorf("rendered HTML still contains the unparsed directive:\n%s", html)
	}
	demoIndex := strings.Index(html, `data-a11y-demo`)
	proseIndex := strings.Index(html, "Following prose after the demo.")
	if demoIndex < 0 || proseIndex < demoIndex {
		t.Errorf("prose after the demo was not rendered after it:\n%s", html)
	}
}

// A demo nested in section/columns must keep every container boundary, leave
// the section open for following prose, and close before the next heading.
func TestMarkdownToMPDAccessibilityDemoNestedKeepsSectionOpen(t *testing.T) {
	source := `@section{variant=story}
## Reader controls

@columns{variant=story}
@column{variant=story-visual}
@accessibility-demo
@end
@end
@end

After the demo, still in the section.
@end

## Following heading
`
	native, _, html := renderImportedTable(t, source)

	document := mpd.Parse("imported.mpd", []byte(native))
	components := countMPDComponents(document)
	for name, want := range map[string]int{
		"section":            1,
		"columns":            1,
		"column":             1,
		"accessibility-demo": 1,
	} {
		if got := components[name]; got != want {
			t.Fatalf("parsed %d %s components, want %d: %#v\n%s", got, name, want, components, native)
		}
	}
	var sectionIndex uint32
	sectionFound, headingAtRoot := false, false
	roots := document.Children(document.Root)
	for {
		index, node, ok := roots.Next()
		if !ok {
			break
		}
		if node.Kind == mpd.KindComponent && string(document.Text(node.Name)) == "section" {
			sectionIndex, sectionFound = index, true
		}
		if node.Kind == mpd.KindHeading && strings.Contains(string(document.Text(node.Source)), "Following heading") {
			headingAtRoot = true
		}
	}
	if !sectionFound || !headingAtRoot {
		t.Fatal("enclosing section and following heading must remain root siblings")
	}
	proseInSection := false
	children := document.Children(sectionIndex)
	for {
		_, child, ok := children.Next()
		if !ok {
			break
		}
		if child.Kind == mpd.KindParagraph && strings.Contains(string(document.Text(child.Source)), "After the demo, still in the section.") {
			proseInSection = true
		}
	}
	if !proseInSection {
		t.Fatal("prose following the demo must remain inside its enclosing section")
	}
	if strings.Contains(html, "@accessibility-demo") {
		t.Errorf("rendered HTML still contains the unparsed directive:\n%s", html)
	}
	if !strings.Contains(html, `data-a11y-demo`) {
		t.Errorf("nested demo did not render:\n%s", html)
	}
	demoIndex := strings.Index(html, `data-a11y-demo`)
	proseIndex := strings.Index(html, "After the demo, still in the section.")
	headingIndex := strings.Index(html, "Following heading")
	if demoIndex < 0 || proseIndex < demoIndex || headingIndex < proseIndex {
		t.Errorf("section prose or following heading was misordered:\n%s", html)
	}
}

// Fenced source examples remain inert and never become live controls.
func TestMarkdownToMPDAccessibilityDemoFenceStaysInert(t *testing.T) {
	source := "# Reader controls\n\n```md\n@accessibility-demo\n@end\n```\n\nThe fenced example stays inert.\n"
	native, _, html := renderImportedTable(t, source)

	document := mpd.Parse("imported.mpd", []byte(native))
	if got := countMPDComponents(document)["accessibility-demo"]; got != 0 {
		t.Fatalf("parsed %d accessibility-demo components from a fenced example, want 0\n%s", got, native)
	}
	if !strings.Contains(mpdCodeText(document), "@accessibility-demo") {
		t.Errorf("fenced example text was lost:\n%s", native)
	}
	if strings.Contains(html, `data-a11y-demo`) {
		t.Errorf("fenced example rendered live reader controls:\n%s", html)
	}
	if !strings.Contains(html, "@accessibility-demo") {
		t.Errorf("rendered HTML lost the literal fenced example:\n%s", html)
	}
}

// Converting the demo must not disturb neighbouring live components.
func TestMarkdownToMPDAccessibilityDemoKeepsNeighbourComponents(t *testing.T) {
	source := "# Reader controls\n\n@accessibility-demo\n@end\n\n@callout{title=\"Next steps\"}\nRead the guide.\n@end\n"
	native, _, html := renderImportedTable(t, source)

	document := mpd.Parse("imported.mpd", []byte(native))
	components := countMPDComponents(document)
	for name, want := range map[string]int{"accessibility-demo": 1, "callout": 1} {
		if got := components[name]; got != want {
			t.Fatalf("parsed %d %s components, want %d: %#v\n%s", got, name, want, components, native)
		}
	}
	demoIndex := strings.Index(html, `data-a11y-demo`)
	calloutIndex := strings.Index(html, "mpress-compact-callout")
	if demoIndex < 0 || calloutIndex < demoIndex {
		t.Errorf("neighbouring callout did not render after the demo:\n%s", html)
	}
	if !strings.Contains(html, "Next steps") || !strings.Contains(html, "Read the guide.") {
		t.Errorf("neighbouring callout lost its content:\n%s", html)
	}
}
