package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildClosesNativeLayoutAroundLeafImage(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Native layout
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", `---
title: Home
layout: landing
---
@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Build clearly.
Ship confidently.
@end
@end

@column{variant=site-screenshot}
@image{light="/preview-light.svg" dark="/preview-dark.svg" alt="Generated documentation preview"}
@end
@end
@end
`)
	writeFixture(t, root, "static/preview-light.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="18"><rect width="32" height="18" fill="white"/></svg>`)
	writeFixture(t, root, "static/preview-dark.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="32" height="18"><rect width="32" height="18" fill="black"/></svg>`)

	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(page)
	for _, want := range []string{
		`<section class="mpress-section mpress-section-hero mp-home-hero"`,
		`class="mpress-theme-image"`,
		`src="/preview-light.svg"`,
		`src="/preview-dark.svg"`,
		`</section>`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("landing page missing %q: %s", want, markup)
		}
	}
	for _, leaked := range []string{"@section", "@columns", "@column", "@image", "@end"} {
		if strings.Contains(markup, leaked) {
			t.Fatalf("native layout syntax %q leaked into generated HTML: %s", leaked, markup)
		}
	}
}
