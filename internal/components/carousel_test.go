package components

import (
	"strings"
	"testing"
)

func TestCarouselRendersMarkdownSlidesAndControls(t *testing.T) {
	component := &Carousel{Meta: map[string]string{"label": "Reading options"}}
	if err := component.Parse("## Reading\n\nChange the type.\n\n---\n\n## Focus\n\nReduce distractions."); err != nil {
		t.Fatal(err)
	}
	got, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="mpress-carousel"`, `aria-label="Reading options"`, `data-carousel-slide`, `>Reading</h2>`, `>Focus</h2>`, `data-carousel-previous`, `data-carousel-next`, `data-carousel-status`, `data-carousel-dot="1"`, `lucide-chevron-left`, `lucide-chevron-right`} {
		if !strings.Contains(got, want) {
			t.Errorf("carousel output missing %q:\n%s", want, got)
		}
	}
	if count := strings.Count(got, `data-carousel-slide`); count != 2 {
		t.Fatalf("expected two slides, got %d", count)
	}
	if count := strings.Count(got, ` hidden`); count != 1 {
		t.Fatalf("expected only the second slide to be hidden, got %d", count)
	}
}

func TestCarouselRequiresContent(t *testing.T) {
	component := &Carousel{}
	if _, err := component.Render(); err == nil {
		t.Fatal("expected an empty carousel to fail")
	}
}
