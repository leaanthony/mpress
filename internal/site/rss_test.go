package site

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leaanthony/mpress/internal/config"
)

func TestBuildRSSReplacesImportedPlaceholder(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: News & releases
  description: Updates <for> everyone
  baseURL: https://example.test/docs
  defaultLanguage: en
  defaultLanguageAtRoot: true
  languages: [en, fr]
social:
  rss: /blog/rss.xml
`)
	writeFixture(t, root, "content/blog/index.md", "---\ntitle: Blog\nlayout: blog-index\n---\n")
	writeFixture(t, root, "content/blog/new.md", "---\ntitle: New & improved\ndescription: A < B & C\ndate: 2026-01-02\ntags: [release]\n---\nNew post\n")
	writeFixture(t, root, "content/blog/old.md", "---\ntitle: Old\ndate: 2025-01-01\n---\nOld post\n")
	writeFixture(t, root, "content/blog/draft.md", "---\ntitle: Draft\ndraft: true\n---\nPrivate\n")
	writeFixture(t, root, "content/blog/future.md", "---\ntitle: Future\ndate: 2999-01-01\n---\nFuture\n")
	writeFixture(t, root, "content/fr/blog/index.md", "---\ntitle: Blog français\nlayout: blog-index\n---\n")
	writeFixture(t, root, "content/fr/blog/new.md", "---\ntitle: Nouveau\ndate: 2026-01-02\n---\nFrench\n")
	writeFixture(t, root, "static/blog/rss.xml", "<rss><channel></channel></rss>")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "site/blog/rss.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var feed rssDocument
	if err := xml.Unmarshal(data, &feed); err != nil {
		t.Fatal(err)
	}
	if feed.Version != "2.0" || feed.Channel.Title != "News & releases" || feed.Channel.Link != "https://example.test/docs/blog/" {
		t.Fatalf("invalid channel: %+v", feed)
	}
	if len(feed.Channel.Items) != 2 {
		t.Fatalf("want two published English items, got %s", data)
	}
	first := feed.Channel.Items[0]
	if first.Title != "New & improved" || first.Description != "A < B & C" || first.Link != "https://example.test/docs/blog/new/" || first.GUID != first.Link || len(first.Categories) != 1 {
		t.Fatalf("invalid item: %+v", first)
	}
	if _, err := time.Parse(time.RFC1123Z, first.PubDate); err != nil {
		t.Fatal(err)
	}
	if feed.Channel.Items[1].Title != "Old" {
		t.Fatal("feed is not newest first")
	}
}

func TestRSSOutputPaths(t *testing.T) {
	for _, value := range []string{"/../outside.xml", "/%2e%2e/outside.xml", "/blog/rss.xml?x=1", "/blog/rss.xml#fragment", "/blog/CON"} {
		t.Run(value, func(t *testing.T) {
			cfg := config.Config{Social: config.SocialConfig{RSS: value}}
			if _, err := rssOutputPath(cfg); err == nil {
				t.Fatalf("accepted unsafe output %q", value)
			}
		})
	}
	for _, value := range []string{"", "https://external.test/rss.xml", "//external.test/rss.xml"} {
		cfg := config.Config{Social: config.SocialConfig{RSS: value}}
		if name, err := rssOutputPath(cfg); err != nil || name != "" {
			t.Fatalf("external/disabled feed: %q %v", name, err)
		}
	}
}

func TestBuildRSSRejectsPageCollision(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "social:\n  rss: /blog/index.html\n")
	writeFixture(t, root, "content/blog/index.md", "---\ntitle: Blog\nlayout: blog-index\n---\n")
	if _, err := Build(root, BuildOptions{}); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("want output collision, got %v", err)
	}
}

func TestBuildRSSRejectsEngineCollision(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", "social:\n  rss: /sitemap.xml\n")
	writeFixture(t, root, "content/index.md", "# Home\n")
	if _, err := Build(root, BuildOptions{}); err == nil || !strings.Contains(err.Error(), "collision") {
		t.Fatalf("want output collision, got %v", err)
	}
}
