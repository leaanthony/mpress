package site

import (
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"net/url"
	"strings"
	"testing"
)

func TestLocalizedDocumentLinksPreserveQueriesAndMapHeadings(t *testing.T) {
	cfg := config.Default()
	cfg.Site.BaseURL = "https://docs.example"
	cfg.Site.DefaultLanguage = "en"
	cfg.Site.DefaultAtRoot = true
	en := &content.Page{URLPath: "guide", Headings: []content.Heading{{ID: "install", Level: 2}}}
	fr := &content.Page{URLPath: "guide", Headings: []content.Heading{{ID: "installer", Level: 2}}}
	sources := map[string]*content.Page{"guide": en}
	targets := map[string]*content.Page{"guide": fr}
	base, _ := url.Parse("https://docs.example/start/")
	for _, test := range []struct{ link, want string }{
		{"/guide/#install", "/fr/guide/#installer"},
		{"../guide/?tab=linux#install", "/fr/guide/?tab=linux#installer"},
		{"https://docs.example/guide/#install", "https://docs.example/fr/guide/#installer"},
		{"#current", "#current"}, {"/guide/#explicit-id", "/fr/guide/#explicit-id"},
		{"/ja/guide/", "/ja/guide/"}, {"/fr/guide/", "/fr/guide/"},
		{"/assets/guide.svg", "/assets/guide.svg"}, {"https://other.example/guide/", "https://other.example/guide/"},
		{"mailto:user@example.com", "mailto:user@example.com"}, {"/missing/", "/missing/"},
	} {
		got, err := localizedDocumentLink(cfg, "fr", base, test.link, sources, targets)
		if err != nil || got != test.want {
			t.Fatalf("%s => %q,%v; want %q", test.link, got, err, test.want)
		}
	}
	fr.Headings = nil
	if _, err := localizedDocumentLink(cfg, "fr", base, "/guide/#install", sources, targets); err == nil {
		t.Fatal("accepted an unaligned heading")
	}
}

func TestLocalizePageLinksKeepsCodeAndCachedContentUnchanged(t *testing.T) {
	cfg := config.Default()
	cfg.Site.BaseURL = ""
	cfg.Site.DefaultAtRoot = true
	page := &content.Page{Language: "fr", URLPath: "start", HTML: `<p><a title="Read &amp; learn" href="/guide/">Guide</a></p><pre>&lt;a href="/guide/"&gt;</pre><script>const link='<a href="/guide/">';</script>`}
	original := page.HTML
	routes := map[string]*content.Page{"guide": {URLPath: "guide"}}
	output, err := localizePageLinks(cfg, page, routes, routes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `href="/fr/guide/"`) || !strings.Contains(output, `<pre>&lt;a href="/guide/"&gt;</pre>`) || !strings.Contains(output, `const link='<a href="/guide/">'`) {
		t.Fatal(output)
	}
	if page.HTML != original {
		t.Fatal("modified cached page")
	}
}

func TestLocalizedLinksRespectBasePathsAndLanguageRoots(t *testing.T) {
	cfg := config.Default()
	cfg.Site.BaseURL = "https://docs.example/docs"
	cfg.Site.DefaultAtRoot = false
	base, _ := url.Parse("https://docs.example/docs/en/start/")
	routes := map[string]*content.Page{"": {URLPath: ""}, "guide": {URLPath: "guide"}}
	for _, test := range []struct{ link, want string }{{"/docs/en/", "/docs/fr/"}, {"/docs/en/guide/", "/docs/fr/guide/"}, {"/docs-other/en/guide/", "/docs-other/en/guide/"}} {
		got, err := localizedDocumentLink(cfg, "fr", base, test.link, routes, routes)
		if err != nil || got != test.want {
			t.Fatalf("%q,%v want %q", got, err, test.want)
		}
	}
}
