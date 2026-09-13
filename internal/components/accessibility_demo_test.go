package components

import (
	"strings"
	"testing"
)

func TestAccessibilityDemoUsesLiveReaderControlContracts(t *testing.T) {
	demo := &AccessibilityDemo{}
	got, err := demo.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`data-a11y-demo`, `data-a11y-choice="siteWidth"`, `data-a11y-toggle="readable"`, `data-open-accessibility`} {
		if !strings.Contains(got, want) {
			t.Fatalf("accessibility demo missing %q: %s", want, got)
		}
	}
}

func TestAccessibilityDemoNativeSyntaxRendersInsideColumn(t *testing.T) {
	got, warnings := Process("@column{variant=story-visual}\n@accessibility-demo\n@end\n@end\n")
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(got, `data-a11y-demo`) || strings.Contains(got, `@accessibility-demo`) {
		t.Fatalf("accessibility demo did not render: %s", got)
	}
}
