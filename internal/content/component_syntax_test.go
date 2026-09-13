package content

import (
	"os"
	"strings"
	"testing"
)

func TestCanonicalComponentGrammarRendersThroughContentPipeline(t *testing.T) {
	source := `# Landing

:::section{variant="hero"}
:::actions
:::button{href="/start/" variant="primary"}
Start
:::
:::
:::image{light="/light.png" dark="/dark.png" alt="Preview"}
:::
:::
`
	page, diagnostics, err := NewRenderer().Parse("landing.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %#v", diagnostics)
	}
	for _, want := range []string{"mpress-section-hero", "mpress-button-primary", "mpress-theme-image-light"} {
		if !strings.Contains(page.HTML, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, page.HTML)
		}
	}
}

func TestFeatureShowcaseRendersAccessibilityDemo(t *testing.T) {
	source, err := os.ReadFile("../../docs/features.md")
	if err != nil {
		t.Fatal(err)
	}
	page, diagnostics, err := NewRenderer().Parse("features.md", "en", string(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			t.Fatalf("diagnostic: %#v", diagnostic)
		}
	}
	if !strings.Contains(page.HTML, `data-a11y-demo`) || strings.Contains(page.HTML, `@accessibility-demo`) {
		t.Fatalf("feature showcase did not render accessibility demo")
	}
}

func TestAccessibilityDemoRendersInsideLandingColumn(t *testing.T) {
	source := `@section{variant=story}
@columns{variant=story}
@column{variant=story-visual}
@accessibility-demo
@end
@end
@end
@end
`
	page, diagnostics, err := NewRenderer().Parse("features.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "title-fallback" {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if !strings.Contains(page.HTML, `data-a11y-demo`) || strings.Contains(page.HTML, `@accessibility-demo`) {
		t.Fatalf("accessibility demo did not render through content pipeline: %s", page.HTML)
	}
}

func TestUnparsedComponentDirectiveIsAnError(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("broken.md", "en", "# Broken\n\n@section{variant=hero}\nMissing end marker.\n")
	if err != nil {
		t.Fatal(err)
	}
	if page == nil {
		t.Fatal("expected parsed page")
	}
	want := Diagnostic{Severity: "error", Code: "unparsed-component", File: "broken.md", Message: "component directive was not parsed: @section{variant=hero}"}
	for _, diagnostic := range diagnostics {
		if diagnostic == want {
			return
		}
	}
	t.Fatalf("diagnostics missing %#v: %#v", want, diagnostics)
}

func TestDanglingComponentEndIsAnError(t *testing.T) {
	_, diagnostics, err := NewRenderer().Parse("broken.mpd", "en", "# Broken\n\n@end\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "unparsed-component" && strings.Contains(diagnostic.Message, "@end") {
			return
		}
	}
	t.Fatalf("dangling @end was not diagnosed: %#v", diagnostics)
}

func TestFenceProtectionRequiresMatchingBacktickRun(t *testing.T) {
	source := "# Example\n\n````text\n@rawHTML\n```\n@end\n````\n"
	page, diagnostics, err := NewRenderer().Parse("fence.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %#v", diagnostics)
	}
	if !strings.Contains(page.HTML, "@rawHTML") || !strings.Contains(page.HTML, "@end") {
		t.Fatalf("long fence did not preserve literal directives:\n%s", page.HTML)
	}
}
