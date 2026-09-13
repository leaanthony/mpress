package check

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunFindsMissingTargetsAndFragments(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index.html", `<a href="guide/#present">yes</a><a href="guide/#absent">bad fragment</a><img src="missing.png">`)
	write("guide/index.html", `<h2 id="present">Guide</h2>`)
	broken, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(broken) != 2 {
		t.Fatalf("expected 2 failures, got %#v", broken)
	}
}

func TestRunIgnoresEscapedHTMLExamples(t *testing.T) {
	root := t.TempDir()
	body := `<pre><code>&lt;img src="/not-a-real-image.png"&gt;</code></pre>`
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(broken) != 0 {
		t.Fatalf("escaped example was treated as a live element: %#v", broken)
	}
}

func TestSkipRecognisesExternalURIsAndMalformedEscapes(t *testing.T) {
	for _, href := range []string{
		"https://docs.example.test/guide/",
		"mailto:docs@example.test",
		"custom+scheme:payload",
		"//cdn.example.test/app.js",
		"bad%zz",
	} {
		if !skip(href) {
			t.Fatalf("expected %q to be skipped", href)
		}
	}
	for _, href := range []string{"guide/?tab=one#install", "space%20page/", "#section"} {
		if skip(href) {
			t.Fatalf("expected %q to remain a local reference", href)
		}
	}
}

func TestRunResolvesRelativeRootDirectoryAssetsQueriesAndEscapedPaths(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("index.html", `<h1 id="home">Home</h1><a href="guide">guide</a><a href="/assets/app.js">asset</a><a href="/assets/app.js#asset">asset fragment</a><a href="space%20page/?q=1#space%20heading">escaped</a><a href="#home">self</a><a href="guide/#missing">bad fragment</a><img src="/assets/missing.png"><a href="guide/#missing">duplicate</a><a href="//cdn.example.test/app.js">external</a>`)
	write("guide/index.html", `<h2 id="present">Guide</h2>`)
	write("space page/index.html", `<h2 id="space heading">Escaped</h2>`)
	write("assets/app.js", `console.log("ok") /* id="asset" */`)

	broken, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []BrokenLink{
		{File: "index.html", Href: "guide/#missing", Reason: "fragment not found"},
		{File: "index.html", Href: "/assets/missing.png", Reason: "target not found"},
		{File: "index.html", Href: "guide/#missing", Reason: "fragment not found"},
	}
	if len(broken) != len(want) {
		t.Fatalf("expected %#v, got %#v", want, broken)
	}
	for i := range want {
		if broken[i] != want[i] {
			t.Fatalf("expected broken[%d] %#v, got %#v", i, want[i], broken[i])
		}
	}
}

func TestRunPreservesFragmentIDsWithHTMLAttributes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<a href="page/#entity">ok</a><a href="page/#missing">bad</a>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "page"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "page", "index.html"), []byte(`<h2 id = "entity">Entity</h2>`), 0o644); err != nil {
		t.Fatal(err)
	}
	broken, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(broken) != 1 || broken[0].Href != "page/#missing" {
		t.Fatalf("unexpected broken links: %#v", broken)
	}
}

func TestCollectorIndexesGeneratedPagesBeforeTheyReachDisk(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Static content is indexed first, exactly as it is during a build. The
	// generated index replaces the static HTML entry without retaining its old
	// reference.
	write("index.html", `<a href="missing.html">stale static page</a>`)
	write("assets/app.js", `console.log("ok")`)
	collector := NewCollector()
	collector.Reset(root)
	if err := collector.IndexRemaining(); err != nil {
		t.Fatal(err)
	}

	generatedIndex := []byte(`<a href="guide/#ready">Guide</a><img src="assets/app.js">`)
	generatedGuide := []byte(`<h2 id="ready">Ready</h2>`)
	write("index.html", string(generatedIndex))
	write("guide/index.html", string(generatedGuide))
	collector.AddHTML("guide/index.html", generatedGuide)
	collector.AddHTML("index.html", generatedIndex)
	if err := collector.IndexRemaining(); err != nil {
		t.Fatal(err)
	}

	got := collector.Finalize()
	if len(got) != 0 {
		t.Fatalf("incremental collector found unexpected failures: %#v", got)
	}
	fromDisk, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(fromDisk) != 0 {
		t.Fatalf("full disk scan disagreed with collector: %#v", fromDisk)
	}
}

func TestCollectorAcceptsWorkerParsedHTMLWithoutChangingHealthResults(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "guide"), 0o755); err != nil {
		t.Fatal(err)
	}
	index := []byte(`<a href="guide/#ready">Guide</a><img src="missing.png">`)
	guide := []byte(`<h2 id="ready">Ready</h2>`)
	if err := os.WriteFile(filepath.Join(root, "index.html"), index, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "guide", "index.html"), guide, 0o644); err != nil {
		t.Fatal(err)
	}

	collector := NewCollector()
	collector.Reset(root)
	collector.AddParsedHTML("guide/index.html", ParseHTML(guide))
	collector.AddParsedHTML("index.html", ParseHTML(index))
	got := collector.Finalize()
	want, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("worker parsed health output differs: got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("worker parsed health output differs at %d: got %#v want %#v", i, got[i], want[i])
		}
	}
}

func BenchmarkRunIndexedSite(b *testing.B) {
	root := b.TempDir()
	var body strings.Builder
	for i := 0; i < 160; i++ {
		body.WriteString(`<a href="/shared/#shared">shared</a>`)
		body.WriteString(`<img src="/assets/app.js">`)
	}
	for i := 0; i < 500; i++ {
		name := filepath.Join(root, "pages", "page-"+strconv.Itoa(i), "index.html")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(name, []byte("<h1 id=\"page\">Page</h1>"+body.String()), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "shared"), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "shared", "index.html"), []byte(`<h1 id="shared">Shared</h1>`), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broken, err := Run(root)
		if err != nil {
			b.Fatal(err)
		}
		if len(broken) != 0 {
			b.Fatalf("unexpected broken links: %#v", broken[:min(len(broken), 3)])
		}
	}
}
