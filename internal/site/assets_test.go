package site

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestMinifiedScriptPreservesRuntimeBehavior(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required to compare JavaScript execution results")
	}
	programs := []struct {
		name   string
		source string
	}{
		{
			name: "template interpolation reads a local binding",
			source: `(() => {
  const greeting = "hello";
  globalThis.result = ` + "`message: ${greeting}`" + `;
})();`,
		},
		{
			name: "template interpolation reads a parameter",
			source: `(() => {
  const format = (name) => ` + "`hi ${name}`" + `;
  globalThis.result = format("reader");
})();`,
		},
		{
			name:   "escaped ordinary reference keeps its binding",
			source: "(() => { const greeting = \"hello\"; globalThis.result = greet\\u0069ng; })();",
		},
		{
			name: "escaped interpolation reference keeps its binding",
			source: `(() => {
  const greeting = "hello";
  globalThis.result = ` + "`${greet\\u0069ng}`" + `;
})();`,
		},
		{
			name: "direct eval inside interpolation keeps its bindings",
			source: `(() => {
  const greeting = "hello";
  const pieces = ["greet", "ing"];
  globalThis.result = ` + "`${eval(pieces.join(\"\"))}`" + `;
})();`,
		},
		{
			name: "nested template interpolation keeps its bindings",
			source: `(() => {
  const name = "reader";
  const inner = ` + "`inner ${name}`" + `;
  globalThis.result = ` + "`outer ${`${inner}!`}`" + `;
})();`,
		},
		{
			name: "short name does not capture a template reference",
			source: "globalThis.a = \"outer\";\n" + `(() => {
  const description = "local";
  globalThis.result = ` + "`${a}`" + ` + "/" + description;
})();`,
		},
		{
			name: "closure keeps its captured state",
			source: `(() => {
  const makeCounter = (start) => {
    let count = start;
    return () => {
      count += 1;
      return count;
    };
  };
  const counter = makeCounter(4);
  counter();
  globalThis.result = counter();
})();`,
		},
		{
			name: "shorthand property keeps its key",
			source: `(() => {
  const label = "ready";
  const payload = { label };
  globalThis.result = payload.label;
})();`,
		},
		{
			name: "automatic semicolon insertion keeps statement boundaries",
			source: `(() => {
  const make = () => {
    return
    { value: 1 };
  };
  const first = 1
  const second = 2
  globalThis.result = [make() === undefined ? "undefined" : "object", first + second];
})();`,
		},
		{
			name: "regular expression literal keeps its pattern",
			source: `(() => {
  const pattern = /a\/[b c]+/gi;
  const matches = "A/b c".match(pattern);
  globalThis.result = matches.length + ":" + matches[0];
})();`,
		},
		{
			name: "labels keep their target loop",
			source: `(() => {
  let visited = 0;
  outer: for (let row = 0; row < 3; row++) {
    for (let column = 0; column < 3; column++) {
      if (column === 1) continue outer;
      visited += 1;
    }
  }
  globalThis.result = visited;
})();`,
		},
		{
			name: "optional chaining and nullish coalescing keep their values",
			source: `(() => {
  const state = { nested: { ready: true } };
  const missing = undefined;
  globalThis.result = [state?.nested?.ready ?? false, missing?.value ?? "fallback"];
})();`,
		},
	}
	for _, program := range programs {
		t.Run(program.name, func(t *testing.T) {
			minified, err := minifyScript(program.source)
			if err != nil {
				t.Fatalf("minify script: %v", err)
			}
			if minified == strings.TrimSpace(program.source) {
				t.Fatal("script was not minified")
			}
			want, err := executeJavaScript(t, node, program.source)
			if err != nil {
				t.Fatalf("execute original: %v", err)
			}
			got, err := executeJavaScript(t, node, minified)
			if err != nil {
				t.Fatalf("execute minified %q: %v", minified, err)
			}
			if got != want {
				t.Fatalf("minified script behaved differently:\noriginal=%s\nminified=%s\nwant=%s\ngot=%s", program.source, minified, want, got)
			}
		})
	}
}

// executeJavaScript evaluates source in an isolated context and returns the
// serialized globalThis.result. The vm timeout and process deadline bound a
// fixture that fails to terminate.
func executeJavaScript(t *testing.T, node, source string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, node, "-e", jsExecutionHarness)
	command.Env = append(os.Environ(), "MPRESS_JS_PROGRAM="+source)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if ctx.Err() != nil {
		return "", fmt.Errorf("execution timed out: %w", ctx.Err())
	}
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stdout.String()+" "+stderr.String()))
	}
	return stdout.String(), nil
}

const jsExecutionHarness = `
const vm = require('node:vm');
const context = vm.createContext({});
try {
  vm.runInContext(process.env.MPRESS_JS_PROGRAM, context, { timeout: 2000, filename: 'fixture.js' });
  process.stdout.write(JSON.stringify(context.result));
} catch (error) {
  process.stdout.write('threw ' + error.name + ': ' + error.message);
  process.exit(1);
}
`

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

func TestPurgeUnusedCSSKeepsInlineScriptClasses(t *testing.T) {
	css := `.title-line { display: inline-flex; }
.word-title { display: inline-block; }
.word-out { opacity: 0; }
.ready .word-title { opacity: 1; }
.unused-widget { display: none; }`
	pages := []string{`<main><h1>Build apps</h1></main>
<SCRIPT type="module">
const line = document.createElement('span');
line.className = 'title-line word-title';
line.classList.add("word-out");
document.body.classList.add(` + "`ready`" + `);
</SCRIPT >`}
	got := purgeUnusedCSS(css, pages, "")
	for _, selector := range []string{".title-line", ".word-title", ".word-out", ".ready"} {
		if !strings.Contains(got, selector) {
			t.Errorf("inline script state %s was removed: %s", selector, got)
		}
	}
	if strings.Contains(got, ".unused-widget") {
		t.Fatalf("unreferenced rule was retained: %s", got)
	}
}

// TestBuildPreservesCustomCSSDuringProductionPurge reproduces a site whose
// configured stylesheet serves classes created at runtime by a separate
// static script. Production purging scans generated HTML and inline scripts,
// not static JavaScript, so it must not prune the author's stylesheet.
func TestBuildPreservesCustomCSSDuringProductionPurge(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Custom CSS
build:
  contentDir: content
  staticDir: static
  outputDir: site
  customCSS: custom.css
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n\n<div class=\"author-static\">Ready</div>\n<script src=\"/custom.js\" defer></script>\n")
	writeFixture(t, root, "custom.css", `.author-static { color: green; }
.runtime-created-class { color: blue; }
.md\:flex { display: flex; }
@layer author { .author-static { font-weight: 600; } }
@property --author-tone { syntax: "<color>"; inherits: false; initial-value: green; }
`)
	writeFixture(t, root, "static/custom.js", `document.documentElement.classList.add("runtime-created-class");`)
	if _, err := Build(root, BuildOptions{Strict: true, MinifyAssets: true, PurgeUnusedCSS: true}); err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(filepath.Join(root, "site", "assets", "mpress.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"author-static", "runtime-created-class", `md\:flex`, "@layer", "@property"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("production CSS dropped configured custom rule %q: %s", want, css)
		}
	}
	// Built-in theme pruning must remain active; this header button variant is
	// absent from both the fixture configuration and its rendered markup.
	if strings.Contains(string(css), "header-link-button-custom") {
		t.Fatalf("production purge no longer removes unused built-in theme rules: %s", css)
	}
	if strings.Contains(string(css), "\n") {
		t.Fatal("production CSS was not minified")
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
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n\n<div class=\"keep-production\">Ready</div>\n<script>document.body.classList.add('dynamic-production');</script>\n")
	writeFixture(t, root, "custom.css", ".keep-production { color: green; }\n.dynamic-production { color: blue; }\n.remove-production { display: none; }\n")
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
	if !strings.Contains(string(css), "keep-production") {
		t.Fatalf("production CSS purge removed a used author rule: %s", css)
	}
	// Configured custom CSS is preserved verbatim: the purge scan cannot prove
	// that author rules are unused when separate scripts can create markup at
	// runtime. Built-in theme pruning still removes unused selectors.
	if !strings.Contains(string(css), "remove-production") {
		t.Fatalf("production purge dropped configured custom CSS: %s", css)
	}
	if strings.Contains(string(css), "header-link-button-custom") {
		t.Fatalf("production purge no longer removes unused built-in theme rules: %s", css)
	}
	if !strings.Contains(string(css), "dynamic-production") {
		t.Fatalf("production purge removed an inline-script class: %s", css)
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
