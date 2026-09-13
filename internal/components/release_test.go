package components

import (
	"strings"
	"testing"
)

func TestReleaseRendersSemanticReleaseRecord(t *testing.T) {
	component := &Release{Meta: map[string]string{
		"version": "1.4.0",
		"title":   "The Performance Update",
		"date":    "2026-08-12",
		"type":    "minor",
		"latest":  "true",
	}}
	if err := component.Parse("### Highlights\n- Faster **MPD** compilation\n### Bug Fixes\n- Stable output"); err != nil {
		t.Fatal(err)
	}
	rendered, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<article class="mpress-release mpress-release-minor">`,
		`<span class="mpress-release-type">Latest release</span>`,
		`<h3 class="mpress-release-version"><span>v1.4.0</span><span class="mpress-release-title">The Performance Update</span></h3>`,
		`<time class="mpress-release-date" datetime="2026-08-12">`,
		`<section class="mpress-release-section mpress-release-highlights">`,
		`<strong>MPD</strong>`,
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered release missing %q:\n%s", want, rendered)
		}
	}
}

func TestReleaseAssetsUseFunctionalDownloadIcon(t *testing.T) {
	component := &Release{Meta: map[string]string{"version": "1.0.0"}}
	if err := component.Parse("### Assets\n- [Source code](/release.zip)"); err != nil {
		t.Fatal(err)
	}
	rendered, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`mpress-release-assets`, `mpress-release-asset-icon`, `lucide-download`, `href="/release.zip"`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered release asset missing %q:\n%s", want, rendered)
		}
	}
}

func TestReleaseDoesNotDuplicateVersionPrefix(t *testing.T) {
	component := &Release{Meta: map[string]string{"version": "v2.0.0"}}
	if err := component.Parse("- First release"); err != nil {
		t.Fatal(err)
	}
	rendered, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered, "vv2.0.0") || !strings.Contains(rendered, ">v2.0.0</span></h3>") {
		t.Fatalf("release version prefix is incorrect:\n%s", rendered)
	}
}
