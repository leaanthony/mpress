package components

import (
	"strings"
	"testing"
)

func TestChangelogRendersSemanticReleaseFeed(t *testing.T) {
	component := &Changelog{}
	if err := component.Parse(`### v2.0.0 (2026-08-12)
#### Added
- Native MPD rendering
#### Fixed
- Stable fixture output`); err != nil {
		t.Fatal(err)
	}
	rendered, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="mpress-changelog" role="list"`,
		`<article class="mpress-changelog-entry mpress-changelog-major" role="listitem">`,
		`<header class="mpress-changelog-header"><span class="mpress-changelog-version">v2.0.0</span>`,
		`<time class="mpress-changelog-date" datetime="2026-08-12">2026-08-12</time>`,
		`<div class="mpress-changelog-body">`,
		`<section class="mpress-changelog-section mpress-badge-feature">`,
		`<h4 class="mpress-changelog-category"><span aria-hidden="true"></span>Added</h4>`,
		`<section class="mpress-changelog-section mpress-badge-fix">`,
		`<li>Stable fixture output</li>`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered changelog missing %q:\n%s", want, rendered)
		}
	}
	for _, unwanted := range []string{"mpress-timeline", "mpress-changelog-dot"} {
		if strings.Contains(rendered, unwanted) {
			t.Errorf("rendered changelog still contains timeline styling %q:\n%s", unwanted, rendered)
		}
	}
}

func TestChangelogKeepsUncategorisedChangesAccessible(t *testing.T) {
	component := &Changelog{}
	if err := component.Parse("### v1.2.3\n- Small correction"); err != nil {
		t.Fatal(err)
	}
	rendered, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`mpress-changelog-category-visually-hidden">Changes</h4>`,
		`<li>Small correction</li>`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered changelog missing %q:\n%s", want, rendered)
		}
	}
}
