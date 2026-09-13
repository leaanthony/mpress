package site

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

func TestMinifyStylesheetPreservesCSSSemantics(t *testing.T) {
	source := `/* removable comment */
:root { --accent: #5375f6; }
.card, .card:hover { color: var(--accent); background: url("/assets/card image.svg"); }
@media (max-width: 40rem) { .card { padding: 1rem; } }`
	minified, err := minifyStylesheet(source)
	if err != nil {
		t.Fatalf("minify stylesheet: %v", err)
	}
	if len(minified) >= len(source) {
		t.Fatalf("stylesheet was not reduced: %d -> %d bytes", len(source), len(minified))
	}
	for _, want := range []string{"--accent:#5375f6", ".card:hover", `url("/assets/card image.svg")`, "@media"} {
		if !strings.Contains(minified, want) {
			t.Errorf("minified stylesheet lost %q: %s", want, minified)
		}
	}
}

func TestMinifyScriptPreservesModernSyntax(t *testing.T) {
	source := `// removable comment
const state = { ready: true };
const enabled = state?.ready ?? false;
document.querySelector('[data-ready]')?.classList.toggle('active', enabled);`
	minified, err := minifyScript(source)
	if err != nil {
		t.Fatalf("minify script: %v", err)
	}
	if len(minified) >= len(source) {
		t.Fatalf("script was not reduced: %d -> %d bytes", len(source), len(minified))
	}
	for _, want := range []string{"ready:true", "?.ready", "??", "classList.toggle"} {
		if !strings.Contains(minified, want) {
			t.Errorf("minified script lost %q: %s", want, minified)
		}
	}
}

func TestMinifyStylesheetPreservesSelectorAndValueWhitespace(t *testing.T) {
	source := `@media (min-width: 40rem) {
  article :not(pre) > code { --space: calc(100% - 2rem); content: "/* literal */"; }
}`
	minified, err := minifyStylesheet(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"article :not", "calc(100% - 2rem)", `"/* literal */"`} {
		if !strings.Contains(minified, want) {
			t.Errorf("minified stylesheet lost required syntax %q: %s", want, minified)
		}
	}
}

func TestMinifyStylesheetPreservesEscapedDelimiters(t *testing.T) {
	source := `.icon { background: url(icon\)name.svg); content: "}"; }`
	minified, err := minifyStylesheet(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(minified, `url(icon\)name.svg)`) {
		t.Fatalf("escaped delimiter changed: %s", minified)
	}
}

func TestMinifyScriptPreservesRegexTemplatesAndASI(t *testing.T) {
	source := "const ratio = total / count;\n" +
		"const pattern = /a\\/[b c]+/gi;\n" +
		"const message = `outer ${`inner ${value}`}`;\n" +
		"function boundary() { return\n{ok: true}; }\n" +
		"let first = 1\nlet second = 2\n"
	minified, err := minifyScript(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/a\\/[b c]+/gi", "`outer ${`inner ${value}`}`", "return\n", "1\nlet"} {
		if !strings.Contains(minified, want) {
			t.Errorf("minified script lost required syntax %q: %s", want, minified)
		}
	}
}

func TestMinifyScriptDoesNotCrossMangleRepeatedBindings(t *testing.T) {
	source := `(() => {
  const index = ['page'];
  const positions = [1, 2].map((value, index) => value + index);
  globalThis.result = index[0] + positions.join('');
})();`
	minified, err := minifyScript(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(minified, "const index=") || !strings.Contains(minified, ",index)=>") {
		t.Fatalf("repeated binding was unsafely mangled: %s", minified)
	}
}

func TestMinifyScriptPreservesASIAndLabels(t *testing.T) {
	source := `const object = {}
run(object)
outer: for (;;) { break outer }`
	minified, err := minifyScript(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(minified, "}\nrun") || !strings.Contains(minified, "outer:") || !strings.Contains(minified, "break outer") {
		t.Fatalf("ASI or label semantics changed: %s", minified)
	}
}

func TestMinifiersRejectUnterminatedInput(t *testing.T) {
	if _, err := minifyStylesheet(`.broken { content: "nope; }`); err == nil {
		t.Fatal("CSS minifier accepted an unterminated string")
	}
	if _, err := minifyScript(`const broken = /nope;`); err == nil {
		t.Fatal("JavaScript minifier accepted an unterminated regular expression")
	}
}

func TestPurgeUnusedCSSKeepsRuntimeAndAtRules(t *testing.T) {
	source := `:root { --accent: blue; }
.used { color: var(--accent); }
.used, .unused-sibling { font-weight: 600; }
.active { opacity: 1; }
.js .used { display: block; }
.unused { display: none; }
@media (max-width: 40rem) { .unused-in-media { display: none; } }
@keyframes pulse { from { opacity: 0; } to { opacity: 1; } }`
	purged := purgeUnusedCSS(source, []string{`<main class="used"></main>`}, `document.body.classList.add('runtime-state', 'active')`)
	if strings.Contains(purged, ".unused {\n") || strings.Contains(purged, ".unused-in-media") {
		t.Fatalf("unused selector was retained: %s", purged)
	}
	for _, want := range []string{":root", ".used", ".unused-sibling", ".active", ".js .used", "@keyframes", "pulse"} {
		if !strings.Contains(purged, want) {
			t.Errorf("purge removed required %q: %s", want, purged)
		}
	}
}

func TestBuildOptimizesProductionAssets(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Optimized
build:
  contentDir: content
  staticDir: static
  outputDir: site
  customCSS: custom.css
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n\n<div class=\"keep-production\">Ready</div>\n")
	writeFixture(t, root, "custom.css", ".keep-production { color: green; }\n.remove-production { display: none; }\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true, MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.css"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(css), "remove-production") || !strings.Contains(string(css), "keep-production") {
		t.Fatalf("production CSS purge result is incorrect: %s", css)
	}
	if strings.Contains(string(css), "\n") {
		t.Fatal("production CSS was not minified")
	}
	js, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.js"))
	if err != nil {
		t.Fatal(err)
	}
	if len(js) >= len(defaultThemeJS) {
		t.Fatalf("production JavaScript was not minified: %d -> %d bytes", len(defaultThemeJS), len(js))
	}
	html, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	cssVersion := fmt.Sprintf("%x", sha256.Sum256(css))[:12]
	jsVersion := fmt.Sprintf("%x", sha256.Sum256(js))[:12]
	for _, want := range []string{"mpress.css?v=" + cssVersion, "mpress.js?v=" + jsVersion} {
		if !strings.Contains(string(html), want) {
			t.Errorf("production HTML does not reference final asset %q: %s", want, html)
		}
	}
}

func BenchmarkMinifyProductionStylesheet(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(defaultThemeCSS)))
	for b.Loop() {
		if _, err := minifyStylesheet(defaultThemeCSS); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMinifyProductionScript(b *testing.B) {
	b.ReportAllocs()
	b.SetBytes(int64(len(defaultThemeJS)))
	for b.Loop() {
		if _, err := minifyScript(defaultThemeJS); err != nil {
			b.Fatal(err)
		}
	}
}

func TestProductionAssetsPreferBrotli(t *testing.T) {
	stylesheet, err := minifyStylesheet(defaultThemeCSS)
	if err != nil {
		t.Fatal(err)
	}
	script, err := minifyScript(defaultThemeJS)
	if err != nil {
		t.Fatal(err)
	}
	assets := []byte(stylesheet + script)

	var gzipOutput bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&gzipOutput, gzip.BestCompression)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gzipWriter.Write(assets); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}

	var brotliOutput bytes.Buffer
	brotliWriter := brotli.NewWriterLevel(&brotliOutput, brotli.BestCompression)
	if _, err := brotliWriter.Write(assets); err != nil {
		t.Fatal(err)
	}
	if err := brotliWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if brotliOutput.Len() >= gzipOutput.Len() {
		t.Fatalf("Brotli no longer produces the smallest production assets: br=%d gzip=%d", brotliOutput.Len(), gzipOutput.Len())
	}
	t.Logf("production assets: identity=%d gzip=%d brotli=%d", len(assets), gzipOutput.Len(), brotliOutput.Len())
}
