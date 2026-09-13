package content

import (
	"encoding/base64"
	"encoding/xml"
	"regexp"
	"strings"
	"testing"
)

func TestD2FenceRendersStandaloneDiagram(t *testing.T) {
	source := "# Architecture\n\n```d2\nFrontend -> Backend: Call service\n```\n"
	page, _, err := NewRenderer().Parse("architecture.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`src="data:image/svg\+xml;base64,([^"]+)"`).FindStringSubmatch(page.HTML)
	if len(match) != 2 {
		t.Fatal("D2 fence was not rendered as a standalone SVG image")
	}
	svg, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Name    xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(svg, &document); err != nil || document.ViewBox == "" {
		t.Fatalf("invalid SVG image: %v", err)
	}
	for _, label := range []string{"Frontend", "Backend", "Call service"} {
		if !strings.Contains(string(svg), label) {
			t.Errorf("diagram is missing label %q", label)
		}
	}
	if strings.Contains(page.HTML, `class="language-d2"`) {
		t.Error("diagram source was displayed as ordinary code")
	}
	if !strings.Contains(page.HTML, `alt="Frontend, Backend"`) {
		t.Error("diagram alternative text does not describe its nodes")
	}
}

func TestD2FenceTitleProvidesAlternativeText(t *testing.T) {
	page, _, err := NewRenderer().Parse("titled.md", "en", "# Title\n\n```d2 title=\"Client & server\"\nClient -> Server\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.HTML, `alt="Client &amp; server"`) {
		t.Fatal("diagram title was not escaped and used as alternative text")
	}
}

func TestD2FenceRejectsInvalidInput(t *testing.T) {
	for _, test := range []struct{ name, source, message string }{
		{"syntax", "a: {", "D2 diagram"},
		{"file-import", "...@private", "permission denied"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := NewRenderer().Parse(test.name+".md", "en", "# Diagram\n\n```d2\n"+test.source+"\n```\n")
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q for %s, got %v", test.message, test.name, err)
			}
		})
	}
}

func TestD2FenceKeepsItsListContainer(t *testing.T) {
	source := "# Guide\n\n1. Diagram\n\n   ```d2\n   a -> b\n   ```\n\n2. Continue\n"
	page, _, err := NewRenderer().Parse("list.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	start, diagram, end := strings.Index(page.HTML, "<li>"), strings.Index(page.HTML, "mpress-diagram-d2"), strings.Index(page.HTML, "</li>")
	if start < 0 || diagram < start || diagram > end {
		t.Fatal("diagram escaped its list item")
	}
}
