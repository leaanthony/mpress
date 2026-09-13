package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildNativeMPDWithD2Diagram(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "site:\n  title: Diagrams\nbuild:\n  contentDir: content\n  outputDir: site\n")
	writeFixture(t, root, "content/index.mpd", "---\nschema = 1\ntitle = \"Architecture\"\n---\n\n```d2\nFrontend -> Backend: Call service\n```\n")
	// This cache was produced by the released v1.0.1 binary for the exact
	// source above. An upgrade must not reuse its unrendered D2 code frame.
	entries, err := os.ReadDir("testdata/d2-v1.0.1-cache")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join("testdata/d2-v1.0.1-cache", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, filepath.Join(".mpress/cache/parse", entry.Name()), string(data))
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(filepath.Join(root, "site/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `class="mpress-diagram mpress-diagram-d2"`) || strings.Contains(string(output), `class="language-d2"`) {
		t.Fatal("native MPD build did not render the D2 diagram")
	}
	writeFixture(t, root, "content/index.mpd", "---\nschema = 1\ntitle = \"Broken\"\n---\n\n```d2\na: {\n```\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err == nil {
		t.Fatal("strict build accepted a malformed D2 diagram")
	}
}
