package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func writeRenderedFixture(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func renderedFixture(t *testing.T) (string, config.Config) {
	t.Helper()
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.BaseURL = "https://docs.example.test/manual"
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.DefaultLanguage = "en"
	cfg.Site.DefaultAtRoot = true
	cfg.Build.ContentDir = "content"
	cfg.Build.OutputDir = "site"
	cfg.Search.Enabled = false
	cfg.Knowledge.Enabled = false
	writeRenderedFixture(t, root, "content/index.md", "# Home")
	for _, lang := range cfg.Site.Languages {
		prefix := languagePrefix(cfg, lang)
		body := `<html lang="` + lang + `"><link rel="canonical" href="` + cfg.Site.BaseURL + `/` + prefix + `"><h2 id="café">Welcome</h2>`
		writeRenderedFixture(t, root, "site/"+prefix+"index.html", body)
	}
	for _, name := range []string{"sitemap.xml", "robots.txt", "llms.txt"} {
		writeRenderedFixture(t, root, "site/"+name, "generated")
	}
	return root, cfg
}

func TestRenderedPublicationChecks(t *testing.T) {
	cases := []struct{ name, extra, want string }{
		{"clean", "", ""},
		{"absolute internal", `<a href="https://docs.example.test/manual/missing/">Bad</a>`, "missing link"},
		{"external", `<a href="https://elsewhere.test/missing/">External</a>`, ""},
		{"encoded anchor", `<a href="#caf%C3%A9">Good</a>`, ""},
		{"missing anchor", `<a href="#missing">Bad</a>`, "missing anchor"},
		{"image", `<img src="missing.png">`, "missing link"},
		{"D2", `<pre><code class="language-d2">a -&gt; b</code></pre>`, "D2 diagram"},
		{"fence", "<details>```go\nfunc main() {}\n```</details>", "Markdown code fence"},
		{"literal fence", "<pre><code>```go</code></pre><script>\"```\"</script><svg><text>```</text></svg>", ""},
		{"source escape", `<meta name="mpress:source" content="private.txt">`, "invalid contribution source"},
		{"valid source", `<meta name="mpress:source" content="content/index.md">`, ""},
		{"canonical", `<link rel="canonical" href="https://old.example/">`, "incorrect canonical"},
		{"bad URL", `<a href="/%GG">Bad</a>`, "invalid link"},
		{"encoded escape", `<a href="/manual/%2e%2e/private.txt">Bad</a>`, "link escapes site"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, cfg := renderedFixture(t)
			writeRenderedFixture(t, root, "private.txt", "private")
			path := filepath.Join(root, "site/index.html")
			data, _ := os.ReadFile(path)
			writeRenderedFixture(t, root, "site/index.html", string(data)+tc.extra)
			report, err := Rendered(root, cfg, RenderedOptions{})
			if err != nil {
				t.Fatal(err)
			}
			problems := strings.Join(report.Errors, "\n")
			if tc.want == "" && len(report.Errors) > 0 || tc.want != "" && !strings.Contains(problems, tc.want) {
				t.Fatalf("errors: %s; want %q", problems, tc.want)
			}
		})
	}
}

func TestRenderedRedirects(t *testing.T) {
	for _, tc := range []struct{ name, redirects, want string }{
		{"chain", "/manual/old/ /manual/new/ 301\n/manual/new/ /manual/ 302", ""},
		{"external", "/manual/old/ https://external.test/ 301", ""},
		{"relative target", "/manual/old/ ../ 301", ""},
		{"cycle", "/manual/old/ /manual/new/ 301\n/manual/new/ /manual/old/ 302", "redirect cycle"},
		{"missing", "/manual/old/ /manual/missing/ 301", "missing link"},
		{"invalid status", "/manual/old/ /manual/ 200", "expected an exact path"},
		{"malformed", "invalid", "expected an exact path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, cfg := renderedFixture(t)
			writeRenderedFixture(t, root, "site/_redirects", tc.redirects)
			path := filepath.Join(root, "site/index.html")
			data, _ := os.ReadFile(path)
			writeRenderedFixture(t, root, "site/index.html", string(data)+`<a href="/manual/old/#caf%C3%A9">Redirect</a>`)
			report, err := Rendered(root, cfg, RenderedOptions{})
			if err != nil {
				t.Fatal(err)
			}
			problems := strings.Join(report.Errors, "\n")
			if tc.want == "" && len(report.Errors) > 0 || tc.want != "" && !strings.Contains(problems, tc.want) {
				t.Fatalf("errors: %s; want %q", problems, tc.want)
			}
		})
	}
}

func TestRenderedTranslationRoutes(t *testing.T) {
	root, cfg := renderedFixture(t)
	if err := os.Remove(filepath.Join(root, "site/fr/index.html")); err != nil {
		t.Fatal(err)
	}
	report, err := Rendered(root, cfg, RenderedOptions{})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "fr/index.html: missing translated page") {
		t.Fatalf("%+v %v", report, err)
	}
	writeRenderedFixture(t, root, "site/fr/index.html", `<html lang="en"><link rel="canonical" href="https://docs.example.test/manual/fr/">`)
	report, err = Rendered(root, cfg, RenderedOptions{})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "translation fallback") {
		t.Fatalf("%+v %v", report, err)
	}
	// A non-English default and language-prefixed output must work too.
	cfg.Site.DefaultLanguage = "fr"
	cfg.Site.DefaultAtRoot = false
	writeRenderedFixture(t, root, "site/fr/index.html", `<html lang="fr"><link rel="canonical" href="https://docs.example.test/manual/fr/">`)
	writeRenderedFixture(t, root, "site/en/index.html", `<html lang="en"><link rel="canonical" href="https://docs.example.test/manual/en/">`)
	os.Remove(filepath.Join(root, "site/index.html"))
	report, err = Rendered(root, cfg, RenderedOptions{})
	if err != nil || len(report.Errors) > 0 {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestRenderedKnowledgeAndLimits(t *testing.T) {
	root, cfg := renderedFixture(t)
	cfg.Knowledge.Enabled = true
	names := map[string]string{"pages": "pages.json", "chunks": "chunks.json.gz", "index": "index.json.gz"}
	for _, name := range names {
		writeRenderedFixture(t, root, "site/knowledge/"+name, "data")
	}
	data, _ := json.Marshal(map[string]any{"artifacts": names})
	writeRenderedFixture(t, root, "site/knowledge/manifest.json", string(data))
	report, err := Rendered(root, cfg, RenderedOptions{CloudflarePages: true})
	if err != nil || len(report.Errors) > 0 {
		t.Fatalf("%+v %v", report, err)
	}
	writeRenderedFixture(t, root, "site/knowledge/manifest.json", `{"artifacts":{"pages":"../llms.txt","chunks":"/etc/passwd","index":""}}`)
	report, err = Rendered(root, cfg, RenderedOptions{})
	if err != nil || len(report.Errors) != 3 {
		t.Fatalf("%+v %v", report, err)
	}
	cfg.Knowledge.Enabled = false
	file, err := os.Create(filepath.Join(root, "site/large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(25*1024*1024 + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	report, err = Rendered(root, cfg, RenderedOptions{CloudflarePages: true})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "25 MiB") {
		t.Fatalf("%+v %v", report, err)
	}
	report, err = Rendered(root, cfg, RenderedOptions{})
	if err != nil || len(report.Errors) > 0 {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestRenderedRejectsSymlinkEscapes(t *testing.T) {
	root, cfg := renderedFixture(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "content/escape.md")); err != nil {
		t.Skip(err)
	}
	writeRenderedFixture(t, root, "site/index.html", `<html lang="en"><link rel="canonical" href="https://docs.example.test/manual/"><meta name="mpress:source" content="content/escape.md">`)
	report, err := Rendered(root, cfg, RenderedOptions{})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "invalid contribution source") {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestRenderedEscapedPagePathsDoNotPanic(t *testing.T) {
	root, cfg := renderedFixture(t)
	cfg.Site.BaseURL = ""
	cfg.Site.Languages = []string{"en"}
	if err := os.RemoveAll(filepath.Join(root, "site/fr")); err != nil {
		t.Fatal(err)
	}
	writeRenderedFixture(t, root, "site/100%/index.html", `<html lang="en"><a href="../">Home</a>`)
	report, err := Rendered(root, cfg, RenderedOptions{})
	if err != nil || len(report.Errors) > 0 {
		t.Fatalf("%+v %v", report, err)
	}
}

func TestRenderedCloudflareFileCountBoundary(t *testing.T) {
	root, cfg := renderedFixture(t)
	// The fixture has two pages and three generated assets.
	for i := 5; i < 20000; i++ {
		writeRenderedFixture(t, root, fmt.Sprintf("site/assets/%d.txt", i), "")
	}
	report, err := Rendered(root, cfg, RenderedOptions{CloudflarePages: true})
	if err != nil || len(report.Errors) != 0 || report.Files != 20000 {
		t.Fatalf("%+v %v", report, err)
	}
	writeRenderedFixture(t, root, "site/assets/overflow.txt", "")
	report, err = Rendered(root, cfg, RenderedOptions{CloudflarePages: true})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "20,000") {
		t.Fatalf("%+v %v", report, err)
	}
}
