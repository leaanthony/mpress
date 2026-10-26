package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildShellControlsDoNotShadowHeadingAnchors(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Test
  defaultLanguage: en
  languages: [en, fr]
  defaultLanguageAtRoot: true
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	for _, filename := range []string{"content/index.md", "content/fr/index.md"} {
		writeFixture(t, root, filename, "---\ntitle: Home\n---\n\n## Search\n\n## Theme\n\n## Accessibility\n\n## Menu\n")
	}

	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}

	for _, filename := range []string{"index.html", "fr/index.html"} {
		t.Run(filename, func(t *testing.T) {
			htmlBytes, err := os.ReadFile(filepath.Join(root, "site", filename))
			if err != nil {
				t.Fatal(err)
			}
			markup := string(htmlBytes)
			for _, id := range []string{"search", "theme", "accessibility", "menu"} {
				if got := strings.Count(markup, `id="`+id+`"`); got != 1 {
					t.Errorf("expected exactly 1 occurrence of id=%q, got %d", id, got)
				}
				if !strings.Contains(markup, `<h2 id="`+id+`"`) {
					t.Errorf("expected heading anchor <h2 id=%q> not found", id)
				}
				controlID := "mpress-" + id + "-toggle"
				if got := strings.Count(markup, `id="`+controlID+`"`); got != 1 {
					t.Errorf("expected exactly 1 shell control id=%q, got %d", controlID, got)
				}
				if !strings.Contains(markup, `<button id="`+controlID+`"`) {
					t.Errorf("shell control %q is not a button", controlID)
				}
			}
		})
	}
}
