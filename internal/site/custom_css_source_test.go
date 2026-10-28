package site

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildAttributesCustomCSSMinifyFailureToConfiguredSource reproduces a
// production build whose configured stylesheet is lexically invalid. The error
// must name the configured file and an offset relative to the authored bytes,
// a failed build must leave the published site and sources untouched, and the
// non-minifying build must keep accepting the same authored bytes.
func TestBuildAttributesCustomCSSMinifyFailureToConfiguredSource(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Custom CSS source
build:
  contentDir: content
  staticDir: static
  outputDir: site
  customCSS: styles/reader.css
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n\n<div class=\"reader-tone\">Ready</div>\n")
	validCSS := "/* reader overrides */\n.reader-tone { color: green; }\n"
	writeFixture(t, root, "styles/reader.css", validCSS)
	if _, err := Build(root, BuildOptions{Strict: true, MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
		t.Fatal(err)
	}
	cssPath := filepath.Join(root, "site", "assets", "mpress.css")
	publishedBefore, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatal(err)
	}
	indexBefore := readPublishedIndex(t, root)

	invalidCSS := "/* reader overrides */\n\n.reader-tone { color: \"red; }\n"
	writeFixture(t, root, "styles/reader.css", invalidCSS)
	_, err = Build(root, BuildOptions{Strict: true, MinifyAssets: true, PurgeUnusedCSS: true})
	if err == nil {
		t.Fatal("production build accepted a custom stylesheet with an unterminated string")
	}
	quote := strings.Index(invalidCSS, `"`)
	newline := quote + strings.Index(invalidCSS[quote:], "\n")
	for _, want := range []string{"styles/reader.css", fmt.Sprintf("byte %d", newline)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("minify error does not identify %q: %v", want, err)
		}
	}
	if !strings.Contains(err.Error(), "configured custom stylesheet") {
		t.Errorf("minify error does not distinguish the configured stylesheet: %v", err)
	}
	if strings.Contains(err.Error(), "generated stylesheet") {
		t.Errorf("configured custom stylesheet error is reported as generated CSS: %v", err)
	}
	if indexNow := readPublishedIndex(t, root); indexNow != indexBefore {
		t.Error("failed build replaced the published page")
	}
	if cssNow, readErr := os.ReadFile(cssPath); readErr != nil || string(cssNow) != string(publishedBefore) {
		t.Errorf("failed build replaced the published stylesheet (read error: %v)", readErr)
	}
	if source, readErr := os.ReadFile(filepath.Join(root, "styles", "reader.css")); readErr != nil || string(source) != invalidCSS {
		t.Errorf("failed build modified the authored stylesheet (read error: %v)", readErr)
	}

	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("non-minifying build rejected the authored stylesheet: %v", err)
	}
	published, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(published), invalidCSS) {
		t.Errorf("non-minifying build did not publish the authored bytes unchanged: %s", published)
	}

	correctedCSS := "/* reader overrides */\n\n.reader-tone { color: blue; }\n"
	writeFixture(t, root, "styles/reader.css", correctedCSS)
	if _, err := Build(root, BuildOptions{Strict: true, MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
		t.Fatalf("build did not recover after correcting the stylesheet: %v", err)
	}
	published, err = os.ReadFile(cssPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reader-tone", "blue"} {
		if !strings.Contains(string(published), want) {
			t.Errorf("recovered stylesheet lost intended style %q: %s", want, published)
		}
	}
	if !strings.Contains(readPublishedIndex(t, root), "reader-tone") {
		t.Error("recovered build lost the page using the custom stylesheet")
	}
	assertNoStagingLeftovers(t, root)
}
