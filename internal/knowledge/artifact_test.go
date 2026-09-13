package knowledge

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

func TestGenerateLoadAndSearchKnowledgeArtifact(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Title = "Acme Knowledge"
	cfg.Site.Description = "Build Acme applications."
	cfg.Site.BaseURL = "https://docs.acme.test"
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	pages := map[string][]*content.Page{
		"en": {{SourcePath: "guide.md", Language: "en", URLPath: "guide", Title: "Install Acme", Description: "Install the command line.", PlainText: "Install Acme Linux package apt install acme Configure a token.", HTML: `<h1 id="install-acme">Install Acme</h1><p>Choose the Linux package.</p><h2 id="configure">Configure</h2><p>Configure a token.</p>`, Meta: content.Frontmatter{Tags: []string{"setup", "cli"}}}},
		"fr": {{SourcePath: "guide.md", Language: "fr", URLPath: "guide", Title: "Installer Acme", PlainText: "Installer Acme avec le paquet Linux.", HTML: `<h1 id="installer-acme">Installer Acme</h1><p>Utilisez le paquet Linux.</p>`, Meta: content.Frontmatter{Tags: []string{"setup"}}}},
	}
	first := t.TempDir()
	second := t.TempDir()
	if err := Generate(first, cfg, pages); err != nil {
		t.Fatal(err)
	}
	if err := Generate(second, cfg, pages); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{ManifestFile, PagesFile, ChunksFile, IndexFile} {
		left, err := os.ReadFile(filepath.Join(first, Directory, name))
		if err != nil {
			t.Fatal(err)
		}
		right, err := os.ReadFile(filepath.Join(second, Directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(left, right) {
			t.Fatalf("%s is not deterministic", name)
		}
		if name == IndexFile && strings.Contains(string(left), "\n ") {
			t.Fatalf("%s must use compact JSON for static-host compatibility", name)
		}
	}
	store, err := Load(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Pages) != 2 || len(store.Chunks) != 3 {
		t.Fatalf("pages=%d chunks=%d", len(store.Pages), len(store.Chunks))
	}
	results := store.Search(SearchOptions{Query: "configure token", Language: "en", Tags: []string{"setup"}})
	if len(results) == 0 || results[0].Title != "Configure" || results[0].URL != "https://docs.acme.test/guide/#configure" {
		t.Fatalf("unexpected search results: %#v", results)
	}
	if strings.Contains(results[0].ResourceURI, "guide") || !strings.HasPrefix(results[0].ResourceURI, "mpress://knowledge/section/") {
		t.Fatalf("resource URI is not a stable opaque identifier: %q", results[0].ResourceURI)
	}
	if filtered := store.Search(SearchOptions{Query: "Linux", Language: "de"}); len(filtered) != 0 {
		t.Fatalf("language filter returned %#v", filtered)
	}
}

func TestLoadRejectsTamperedArtifact(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Title = "Tamper test"
	page := &content.Page{SourcePath: "index.md", Language: "en", Title: "Home", PlainText: "Home page", HTML: "<h1 id=\"home\">Home</h1><p>Page.</p>"}
	output := t.TempDir()
	if err := Generate(output, cfg, map[string][]*content.Page{"en": {page}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(output, Directory, ChunksFile)
	if err := os.WriteFile(path, []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(output); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("tampered artifact returned %v", err)
	}
}

func TestTokensNormaliseEnglishPluralsAndChinesePhrases(t *testing.T) {
	if got := strings.Join(tokens("application menus"), ","); got != "application,menu" {
		t.Fatalf("English tokens=%q", got)
	}
	got := tokens("构建系统")
	if strings.Join(got, ",") != "构建,建系,系统" {
		t.Fatalf("Chinese tokens=%q", got)
	}
}

func TestLoadAllIncludesMountedVersionsWithDistinctResources(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Title = "Versioned"
	cfg.Site.BaseURL = "https://docs.example.test"
	page := &content.Page{SourcePath: "guide.md", Language: "en", URLPath: "guide", Title: "Guide", PlainText: "Legacy installation guide.", HTML: "<h1 id=\"install\">Install</h1><p>Legacy installation guide.</p>"}
	output := t.TempDir()
	if err := Generate(output, cfg, map[string][]*content.Page{"en": {page}}); err != nil {
		t.Fatal(err)
	}
	versionRoot := filepath.Join(output, "versions", "1.0")
	if err := os.MkdirAll(versionRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Generate(versionRoot, cfg, map[string][]*content.Page{"en": {page}}); err != nil {
		t.Fatal(err)
	}
	store, err := LoadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Pages) != 2 || store.Pages[0].ResourceURI == store.Pages[1].ResourceURI {
		t.Fatalf("version resources were not separated: %#v", store.Pages)
	}
	results := store.Search(SearchOptions{Query: "legacy installation", Version: "1.0"})
	if len(results) != 1 || results[0].Version != "1.0" || results[0].URL != "https://docs.example.test/versions/1.0/guide/#install" {
		t.Fatalf("unexpected version search: %#v", results)
	}
}
