package components

import (
	"strings"
	"testing"
)

func TestCanonicalFencedButton(t *testing.T) {
	input := `:::button{href="/start/" variant="primary" icon="play"}
Get started
:::
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`class="mpress-button mpress-button-primary"`, `href="/start/"`, `Get started`, `lucide-play`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestCanonicalFencedThemeImage(t *testing.T) {
	input := `:::image{light="/images/light.png" dark="/images/dark.png" alt="Architecture"}
:::
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`mpress-theme-image-light`, `/images/light.png`, `mpress-theme-image-dark`, `/images/dark.png`, `alt="Architecture"`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestCanonicalFencesNestWithoutChangingDelimiter(t *testing.T) {
	input := `:::section{variant="hero"}
:::columns{variant="hero"}
:::column{variant="hero-copy"}
:::actions
:::button{href="/start/" variant="primary"}
Start
:::
:::
:::
:::
:::
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`mpress-section-hero`, `mpress-columns-hero`, `mpress-column-hero-copy`, `mp-home-actions`, `mpress-button-primary`} {
		if !strings.Contains(got, want) {
			t.Errorf("nested output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, ":::") {
		t.Fatalf("canonical fence leaked into output:\n%s", got)
	}
}

func TestTutorialCheckboxIncludesVisibleCheckIcon(t *testing.T) {
	tutorial := &Tutorial{
		Meta:  map[string]string{"title": "Check it"},
		Items: []TutorialStep{{Title: "Verify", Content: "Run the checks."}},
	}

	got, err := tutorial.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="mpress-tutorial-checkmark"`, `lucide-check`, `aria-hidden="true"`} {
		if !strings.Contains(got, want) {
			t.Errorf("tutorial checkbox output missing %q:\n%s", want, got)
		}
	}
}
