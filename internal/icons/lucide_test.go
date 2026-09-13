package icons

import (
	"strings"
	"testing"
)

func TestContributionLucideIconsDoNotUseTheFallbackCircle(t *testing.T) {
	for _, name := range []string{"arrow-up-down", "copy", "list-filter", "pencil"} {
		markup := Lucide(name, 16)
		if !strings.Contains(markup, "lucide-"+name) || strings.Contains(markup, "lucide-circle") {
			t.Fatalf("Lucide(%q) used the fallback: %s", name, markup)
		}
	}
}

func TestAccessibilityUsesTheTracedMonochromeArtwork(t *testing.T) {
	markup := Accessibility(19)
	for _, want := range []string{
		`class="mpress-accessibility-icon"`,
		`viewBox="0 0 512 512"`,
		`id="mpress-accessibility-cutout"`,
		`mask="url(#mpress-accessibility-cutout)"`,
		`fill="black"`,
		`width="19" height="19"`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("accessibility icon is missing %q: %s", want, markup)
		}
	}
	if strings.Contains(markup, `id="title"`) || strings.Contains(markup, `fill="#111"`) || strings.Contains(markup, `fill="#fff"`) {
		t.Fatalf("accessibility icon leaked source-only metadata or fixed colours: %s", markup)
	}
}
