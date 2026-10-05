package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	docversion "github.com/leaanthony/mpress/internal/version"
	"gopkg.in/yaml.v3"
)

func TestBuildRejectsUnsafeRoutesBeforeReplacingOutput(t *testing.T) {
	for _, slug := range []string{"../escaped", "guide/../../escaped", "guide/./topic", `..\escaped`, `C:\escaped`, "C:/escaped", `\\server\share`, "guide:stream", "guide/CON.txt", "guide/LPT²", "guide/end.", "guide/end /topic", "%2e%2e/escaped", "guide%2f..%2fescaped", "%252e%252e/escaped", "%2e%2e%5cescaped", "%ff", "%zz", "%00", "guide?query", "guide#fragment", "bad\x00name"} {
		for _, ext := range []string{"md", "mpd"} {
			for _, strict := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/strict=%v", slug, ext, strict), func(t *testing.T) {
					root := routeSafetyProject(t)
					body := routeSafetySource(t, slug, ext)
					name := "content/unsafe." + ext
					writeFixture(t, root, name, body)
					if _, err := Build(root, BuildOptions{Strict: strict}); err == nil {
						t.Error("unsafe route accepted")
					}
					assertRouteSafetyPreserved(t, root)
					data, err := os.ReadFile(filepath.Join(root, name))
					if err != nil || string(data) != body {
						t.Errorf("source changed: %v", err)
					}
				})
			}
		}
	}
}

func TestBuildAllowsRoutesForDisabledGeneratedOutputs(t *testing.T) {
	root := routeSafetyProject(t)
	cfg := config.Default()
	cfg.Knowledge.Enabled, cfg.Search.Enabled, cfg.Contribution.Enabled, cfg.Version.Enabled = false, false, false, false
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	for i, slug := range []string{"knowledge/index.json", "search-index.json", "contribute.sh", "versions/versions.json", "assets/guide", "knowledge/manifest.json.gz"} {
		writeFixture(t, root, fmt.Sprintf("content/page-%d.md", i), routeSafetySource(t, slug, "md"))
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	for _, filename := range []string{"knowledge/index.json/index.html", "search-index.json/index.html", "contribute.sh/index.html", "versions/versions.json/index.html", "assets/guide/index.html", "knowledge/manifest.json.gz/index.html"} {
		assertExists(t, filepath.Join(root, "site", filepath.FromSlash(filename)))
	}
}

func TestBuildRejectsRouteCollisionsBeforeReplacingOutput(t *testing.T) {
	for _, pair := range [][2]string{{"guide/topic", "guide//topic"}, {"guide", "%67uide"}, {"Guide", "guide"}, {"caf\u00e9", "cafe\u0301"}, {"guide", "guide/index.html/topic"}} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/strict=%v", pair, strict), func(t *testing.T) {
				root := routeSafetyProject(t)
				for i, slug := range pair {
					writeFixture(t, root, fmt.Sprintf("content/page-%d.md", i), routeSafetySource(t, slug, "md"))
				}
				if _, err := Build(root, BuildOptions{Strict: strict}); err == nil {
					t.Error("colliding routes accepted")
				}
				assertRouteSafetyPreserved(t, root)
			})
		}
	}
}

func TestBuildRejectsGeneratedFileAndStaticRouteCollisions(t *testing.T) {
	for _, slug := range []string{"assets/mpress.css", "assets/mpress.js/topic", "knowledge/index.json", "knowledge/pages.json.gz", "search-index.json", "sitemap.xml", "llms.txt", "robots.txt", "mpress-manifest.json", "404.html", "_headers"} {
		t.Run(slug, func(t *testing.T) {
			root := routeSafetyProject(t)
			writeFixture(t, root, "content/page.md", routeSafetySource(t, slug, "md"))
			if _, err := Build(root, BuildOptions{}); err == nil {
				t.Error("generated file collision accepted")
			}
			assertRouteSafetyPreserved(t, root)
		})
	}
	for _, name := range []string{"guide", "guide/index.html", "guide/index.html/child.txt"} {
		t.Run("static/"+name, func(t *testing.T) {
			root := routeSafetyProject(t)
			writeFixture(t, root, "content/guide.md", "# Guide\n")
			writeFixture(t, root, "static/"+name, "static source must survive")
			if _, err := Build(root, BuildOptions{}); err == nil {
				t.Error("static collision accepted")
			}
			assertRouteSafetyPreserved(t, root)
		})
	}
}

func TestBuildRejectsGlobalLocaleRouteCollision(t *testing.T) {
	root := routeSafetyProject(t)
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/fr/index.mpd", routeSafetySource(t, "/", "mpd"))
	writeFixture(t, root, "content/collision.md", routeSafetySource(t, "fr", "md"))
	if _, err := Build(root, BuildOptions{}); err == nil {
		t.Error("default page and localized home collision accepted")
	}
	assertRouteSafetyPreserved(t, root)
}

func TestBuildRejectsActiveOptionalFileRouteCollisions(t *testing.T) {
	for _, slug := range []string{"contribute.sh", "contribute.ps1", "mpress-contribute.sh", "mpress-contribute.ps1", "versions/versions.json", "fr/search-index.json"} {
		t.Run(slug, func(t *testing.T) {
			root := routeSafetyProject(t)
			cfg := config.Default()
			cfg.Contribution.Enabled, cfg.Contribution.Repository = true, "https://github.com/example/docs"
			cfg.Version.Enabled, cfg.Site.Languages = true, []string{"en", "fr"}
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, "content/page.mpd", routeSafetySource(t, slug, "mpd"))
			if _, err := Build(root, BuildOptions{}); err == nil {
				t.Error("active optional file collision accepted")
			}
			assertRouteSafetyPreserved(t, root)
		})
	}
}

func routeSafetyProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/index.md", "# Home\n")
	writeFixture(t, root, "site/last-good", "previous output must survive")
	writeFixture(t, root, "escaped/index.html", "outside output must survive")
	return root
}

func routeSafetySource(t *testing.T, slug, ext string) string {
	t.Helper()
	if ext == "mpd" {
		encoded, err := json.Marshal(slug)
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf("---\nschema = 1\ntitle = \"Page\"\nslug = %s\n---\n# Page\n", encoded)
	}
	data, err := yaml.Marshal(map[string]string{"slug": slug, "title": "Page"})
	if err != nil {
		t.Fatal(err)
	}
	return "---\n" + string(data) + "---\n# Page\n"
}

func TestBuildRejectsUnsafeCachedRoutes(t *testing.T) {
	for _, page := range []content.Page{
		{URLPath: "guide", OutputPath: "../escaped/index.html"},
		{URLPath: "guide", OutputPath: "/guide/index.html"},
		{URLPath: "guide", OutputPath: `guide\index.html`},
		{URLPath: "guide", OutputPath: "guide/../guide/index.html"},
		{URLPath: "../escaped", OutputPath: "../escaped/index.html"},
		{URLPath: "guide//topic", OutputPath: "guide/topic/index.html"},
	} {
		t.Run(page.URLPath+"/"+page.OutputPath, func(t *testing.T) {
			root := routeSafetyProject(t)
			source := []byte("# Guide\n")
			writeFixture(t, root, "content/guide.md", string(source))
			page.SourcePath, page.Language, page.Title = "guide.md", "en", "Guide"
			cache := newParseCache(root)
			key := cache.key(parseJob{lang: "en", rel: "guide.md"}, source)
			cache.save(key, &page, nil)
			if _, _, hit := cache.load(key); !hit {
				t.Fatal("fixture did not populate the active parse cache")
			}
			if _, err := Build(root, BuildOptions{}); err == nil {
				t.Error("unsafe cache output accepted")
			}
			assertRouteSafetyPreserved(t, root)
		})
	}
}

func TestBuildAllowsSafeNormalizedAndLocalizedRoutes(t *testing.T) {
	for _, atRoot := range []bool{false, true} {
		t.Run(fmt.Sprintf("default-at-root=%v", atRoot), func(t *testing.T) {
			root := routeSafetyProject(t)
			cfg := config.Default()
			cfg.Site.Languages, cfg.Site.DefaultAtRoot = []string{"en", "zh-Hans"}, atRoot
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, "content/01-guide/02-topic.markdown", "# Topic\n")
			writeFixture(t, root, "content/custom.mpd", routeSafetySource(t, "/nested//caf%C3%A9/", "mpd"))
			writeFixture(t, root, "content/knowledge.md", "# Knowledge\n")
			writeFixture(t, root, "content/zh-Hans/index.mpd", routeSafetySource(t, "/", "mpd"))
			writeFixture(t, root, "content/zh-Hans/guide.md", routeSafetySource(t, "指南/入门", "md"))
			for i := 0; i < 2; i++ {
				if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
					t.Fatal(err)
				}
				prefix := ""
				if !atRoot {
					prefix = "en/"
				}
				for _, filename := range []string{prefix + "index.html", prefix + "guide/topic/index.html", prefix + "nested/café/index.html", prefix + "knowledge/index.html", "zh-Hans/index.html", "zh-Hans/指南/入门/index.html", "knowledge/index.json"} {
					assertExists(t, filepath.Join(root, "site", filepath.FromSlash(filename)))
				}
			}
		})
	}
}

func TestBuildRejectsVersionSnapshotRouteCollision(t *testing.T) {
	root := routeSafetyProject(t)
	cfg := config.Default()
	cfg.Version.Enabled = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "site/guide/index.html", "historical page")
	writeFixture(t, root, "site/index.html", "historical home")
	if err := docversion.Capture(root, "1.0", false); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/page.md", routeSafetySource(t, "versions/1.0/guide", "md"))
	if _, err := Build(root, BuildOptions{}); err == nil {
		t.Error("snapshot collision accepted")
	}
	assertRouteSafetyPreserved(t, root)
	data, err := os.ReadFile(filepath.Join(cfg.ArtifactsPath(root), "1.0/guide/index.html"))
	if err != nil || string(data) != "historical page" {
		t.Fatal("snapshot source changed")
	}
}

func assertRouteSafetyPreserved(t *testing.T, root string) {
	t.Helper()
	assertOutputSafetySentinels(t, root, map[string]string{"content/index.md": "# Home\n", "site/last-good": "previous output must survive", "escaped/index.html": "outside output must survive"})
}
