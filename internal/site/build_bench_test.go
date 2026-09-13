package site

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// benchProject writes a synthetic project with the given number of content
// pages. Each page carries enough Markdown to exercise the parser, renderer, and
// CSS purge realistically rather than measuring fixed per-build overhead.
func benchProject(tb testing.TB, pages int) string {
	tb.Helper()
	root := tb.TempDir()
	must := func(err error) {
		if err != nil {
			tb.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(root, "mpress.yaml"), []byte(`site:
  title: Bench
  description: Benchmark site
  defaultLanguage: en
  languages: [en]
build:
  contentDir: content
  staticDir: static
  outputDir: site
search:
  enabled: true
`), 0o644))
	must(os.MkdirAll(filepath.Join(root, "content"), 0o755))
	must(os.MkdirAll(filepath.Join(root, "static"), 0o755))
	var body string
	for section := 0; section < 24; section++ {
		body += fmt.Sprintf("## Section %d\n\nParagraph with **bold**, `code`, and a [link](/page-0001/).\n\n"+
			"```go\nfunc sample%d() {}\n```\n\n- one\n- two\n- three\n\n", section, section)
	}
	must(os.WriteFile(filepath.Join(root, "content", "index.md"), []byte("---\ntitle: Home\n---\n\n# Home\n\n"+body), 0o644))
	for i := 0; i < pages; i++ {
		src := fmt.Sprintf("---\ntitle: Page %d\ndescription: Description %d\norder: %d\n---\n\n# Page %d\n\n%s", i, i, i, i, body)
		must(os.WriteFile(filepath.Join(root, "content", fmt.Sprintf("page-%04d.md", i)), []byte(src), 0o644))
	}
	return root
}

// BenchmarkBuildCold measures a from-scratch production build with no parse
// cache: parse, render, and CSS optimization all run in full.
func BenchmarkBuildCold(b *testing.B) {
	root := benchProject(b, 200)
	opts := BuildOptions{MinifyAssets: true, PurgeUnusedCSS: true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_ = os.RemoveAll(filepath.Join(root, ".mpress"))
		_ = os.RemoveAll(filepath.Join(root, "site"))
		b.StartTimer()
		if _, err := Build(root, opts); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBuildWarm measures a rebuild with a populated parse cache, isolating
// the render and CSS optimization phases from Markdown parsing.
func BenchmarkBuildWarm(b *testing.B) {
	root := benchProject(b, 200)
	opts := BuildOptions{MinifyAssets: true, PurgeUnusedCSS: true}
	if _, err := Build(root, opts); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		_ = os.RemoveAll(filepath.Join(root, "site"))
		b.StartTimer()
		if _, err := Build(root, opts); err != nil {
			b.Fatal(err)
		}
	}
}
