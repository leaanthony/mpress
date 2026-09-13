package mpdconformance

import (
	_ "embed"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/site"
)

var (
	//go:embed testdata/assets/mp4.mp4
	fixtureVideo []byte
	//go:embed testdata/assets/mp4.webp
	fixtureVideoPoster []byte
)

// Regenerate writes every expected HTML fragment and rebuilds the checked-in
// visual viewer from the production MPress renderer and theme.
func Regenerate(repositoryRoot string) error {
	packageRoot := filepath.Join(repositoryRoot, "internal", "mpdconformance")
	corpus, err := LoadFS(os.DirFS(packageRoot), "testdata", false)
	if err != nil {
		return err
	}
	if err := writeHTMLGoldens(packageRoot, corpus); err != nil {
		return err
	}
	corpus, err = LoadFS(os.DirFS(packageRoot), "testdata", true)
	if err != nil {
		return err
	}
	return GenerateViewer(corpus, filepath.Join(packageRoot, "testdata", "viewer"))
}

func writeHTMLGoldens(packageRoot string, corpus *Corpus) error {
	renderer := content.NewRenderer()
	for index := range corpus.Fixtures {
		fixture := &corpus.Fixtures[index]
		page, diagnostics, err := renderer.Parse(fixture.ID+".md", "en", fixture.Markdown)
		if err != nil {
			return fmt.Errorf("render Markdown reference %s: %w", fixture.ID, err)
		}
		for _, diagnostic := range diagnostics {
			if diagnostic.Severity == "error" {
				return fmt.Errorf("render Markdown reference %s: %s: %s", fixture.ID, diagnostic.Code, diagnostic.Message)
			}
		}
		fixture.HTML = page.HTML
		target := filepath.Join(packageRoot, "testdata", "corpus", filepath.FromSlash(fixture.ID)+".html")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(page.HTML), 0o644); err != nil {
			return fmt.Errorf("write HTML fixture %s: %w", fixture.ID, err)
		}
	}
	return nil
}

// GenerateViewer builds a standalone MPress site that displays every checked-in
// fixture with MPD and Markdown source tabs and the real rendered HTML.
func GenerateViewer(corpus *Corpus, outputDir string) error {
	projectDir, err := os.MkdirTemp("", "mpress-mpd-viewer-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(projectDir)

	for _, directory := range []string{"docs", "static"} {
		if err := os.MkdirAll(filepath.Join(projectDir, directory), 0o755); err != nil {
			return err
		}
	}
	config := `site:
  title: MPD Fixture Viewer
  description: MPress Document conformance fixtures rendered with the current MPress theme.
  defaultLanguage: en
  languages: [en]
theme:
  colorScheme: system
  layout:
    preset: wide
build:
  contentDir: docs
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
  customCSS: viewer.css
search:
  enabled: false
accessibility:
  enabled: false
`
	files := map[string]string{
		"mpress.yaml":                     config,
		"viewer.css":                      viewerCSS,
		"docs/index.md":                   viewerMarkdown(corpus),
		"static/assets/example-light.svg": fixtureImageLight,
		"static/assets/example-dark.svg":  fixtureImageDark,
		"static/assets/logo.svg":          fixtureLogo,
		"docs/_nav.yaml": `- label: Fixture viewer
  link: /
`,
	}
	for name, data := range files {
		target := filepath.Join(projectDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, []byte(data), 0o644); err != nil {
			return err
		}
	}
	for name, data := range map[string][]byte{
		"static/assets/mp4.mp4":  fixtureVideo,
		"static/assets/mp4.webp": fixtureVideoPoster,
	} {
		target := filepath.Join(projectDir, filepath.FromSlash(name))
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("write fixture asset %s: %w", name, err)
		}
	}
	buildOutput := filepath.Join(projectDir, "site")
	result, err := site.Build(projectDir, site.BuildOptions{Strict: true, OutputDir: buildOutput})
	if err != nil {
		return fmt.Errorf("build MPD fixture viewer: %w", err)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == "error" {
			return fmt.Errorf("build MPD fixture viewer: %s: %s", diagnostic.Code, diagnostic.Message)
		}
	}
	if err := os.RemoveAll(outputDir); err != nil {
		return err
	}
	return copyDirectory(buildOutput, outputDir)
}

func viewerMarkdown(corpus *Corpus) string {
	var out strings.Builder
	out.WriteString("---\ntitle: MPD Fixture Viewer\ndescription: Visual conformance corpus for MPress Document.\nlayout: landing\n---\n\n")
	out.WriteString(`<div class="mpd-viewer"><header class="mpd-viewer-hero"><p class="mpd-kicker">MPress Document · conformance corpus</p><h1>Every construct, rendered.</h1><p>Review the native MPD source, its Markdown interchange form, and the production HTML contract. Every preview uses the current MPress CSS and runtime.</p><dl class="mpd-stats"><div><dt>Fixtures</dt><dd>`)
	fmt.Fprintf(&out, "%d", len(corpus.Fixtures))
	out.WriteString(`</dd></div><div><dt>Top level</dt><dd>`)
	fmt.Fprintf(&out, "%d", corpus.CountLevel(LevelTop))
	out.WriteString(`</dd></div><div><dt>Nested</dt><dd>`)
	fmt.Fprintf(&out, "%d", corpus.CountLevel(LevelNested))
	out.WriteString(`</dd></div><div><dt>Categories</dt><dd>`)
	fmt.Fprintf(&out, "%d", len(corpus.Categories()))
	out.WriteString(`</dd></div></dl></header>`)
	out.WriteString(`<section class="mpd-toolbar" aria-label="Fixture controls"><label class="mpd-search"><span>Find a fixture</span><input type="search" data-fixture-search placeholder="Search fixtures and features"></label><div class="mpd-filter-group" role="group" aria-label="Fixture depth"><button type="button" class="active" data-level="all">All</button><button type="button" data-level="top">Top level</button><button type="button" data-level="nested">Nested</button></div><div class="mpd-filter-group" role="group" aria-label="Preview theme"><button type="button" data-preview-theme="system">System</button><button type="button" data-preview-theme="light">Light</button><button type="button" data-preview-theme="dark">Dark</button></div></section>`)
	out.WriteString(`<nav class="mpd-categories" aria-label="Fixture categories"><button type="button" class="active" data-category="all">All</button>`)
	for _, category := range corpus.Categories() {
		fmt.Fprintf(&out, `<button type="button" data-category="%s">%s</button>`, html.EscapeString(category), html.EscapeString(categoryLabel(category)))
	}
	out.WriteString(`</nav><p class="mpd-result-count" role="status" aria-live="polite"></p><div class="mpd-fixtures">`)
	for _, fixture := range corpus.Fixtures {
		features := strings.Join(fixture.Features, " ")
		fmt.Fprintf(&out, `<article class="mpd-fixture" id="fixture-%s" data-category="%s" data-level="%s" data-search="%s"><header class="mpd-fixture-heading"><div><p>%s · %s</p><h2>%s</h2></div><a href="#fixture-%s" aria-label="Link to %s">#</a></header>`, html.EscapeString(slugID(fixture.ID)), html.EscapeString(fixture.Category), html.EscapeString(fixture.Level), html.EscapeString(strings.ToLower(fixture.Title+" "+fixture.ID+" "+features)), html.EscapeString(categoryLabel(fixture.Category)), html.EscapeString(fixture.Level), html.EscapeString(fixture.Title), html.EscapeString(slugID(fixture.ID)), html.EscapeString(fixture.Title))
		out.WriteString(`<div class="mpd-source"><div class="mpd-source-tabs" role="tablist"><button type="button" role="tab" aria-selected="true" data-source-tab="mpd">MPD</button><button type="button" role="tab" aria-selected="false" data-source-tab="markdown">Markdown</button></div><pre data-source-panel="mpd"><code>`)
		out.WriteString(escapedPreformatted(formatMPDForDisplay(fixture.MPD)))
		out.WriteString(`</code></pre><pre data-source-panel="markdown" hidden><code>`)
		out.WriteString(escapedPreformatted(fixture.Markdown))
		out.WriteString(`</code></pre></div><div class="mpd-preview"><p class="mpd-preview-label">Expected HTML</p><div class="mpd-preview-body">`)
		out.WriteString(fixture.HTML)
		out.WriteString(`</div></div></article>`)
	}
	out.WriteString(`</div><p class="mpd-empty" hidden>No fixtures match these filters.</p></div>`)
	out.WriteString(`<script>
(() => {
  const root = document.querySelector('.mpd-viewer');
  if (!root) return;
  const fixtures = [...root.querySelectorAll('.mpd-fixture')];
  const query = root.querySelector('[data-fixture-search]');
  const status = root.querySelector('.mpd-result-count');
  const empty = root.querySelector('.mpd-empty');
  let category = 'all';
  let level = 'all';
  const filter = () => {
    const needle = query.value.trim().toLowerCase();
    let visible = 0;
    fixtures.forEach(fixture => {
      const show = (category === 'all' || fixture.dataset.category === category) && (level === 'all' || fixture.dataset.level === level) && (!needle || fixture.dataset.search.includes(needle));
      fixture.hidden = !show;
      if (show) visible++;
    });
    status.textContent = visible + (visible === 1 ? ' fixture' : ' fixtures');
    empty.hidden = visible !== 0;
  };
  query.addEventListener('input', filter);
  root.querySelectorAll('[data-category]').forEach(button => button.addEventListener('click', () => {
    category = button.dataset.category;
    root.querySelectorAll('[data-category]').forEach(item => item.classList.toggle('active', item === button));
    filter();
  }));
  root.querySelectorAll('[data-level]').forEach(button => button.addEventListener('click', () => {
    level = button.dataset.level;
    root.querySelectorAll('[data-level]').forEach(item => item.classList.toggle('active', item === button));
    filter();
  }));
  root.querySelectorAll('[data-preview-theme]').forEach(button => button.addEventListener('click', () => {
    const wanted = button.dataset.previewTheme;
    const theme = document.querySelector('#theme');
    for (let attempt = 0; theme && document.documentElement.dataset.theme !== wanted && attempt < 3; attempt++) theme.click();
    root.querySelectorAll('[data-preview-theme]').forEach(item => item.classList.toggle('active', item.dataset.previewTheme === document.documentElement.dataset.theme));
  }));
  root.querySelectorAll('.mpd-source').forEach(source => source.querySelectorAll('[data-source-tab]').forEach(button => button.addEventListener('click', () => {
    const selected = button.dataset.sourceTab;
    source.querySelectorAll('[data-source-tab]').forEach(item => item.setAttribute('aria-selected', String(item === button)));
    source.querySelectorAll('[data-source-panel]').forEach(panel => panel.hidden = panel.dataset.sourcePanel !== selected);
  })));
  const selectedTheme = document.documentElement.dataset.theme || 'system';
  root.querySelector('[data-preview-theme="' + selectedTheme + '"]')?.classList.add('active');
  filter();
})();
</script>`)
	return out.String()
}

const fixtureImageLight = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 720 360"><rect width="720" height="360" fill="#f7f8fa"/><rect x="48" y="48" width="624" height="264" rx="12" fill="#fff" stroke="#d8dde8"/><path d="M96 112h244M96 152h416M96 192h352" stroke="#667085" stroke-width="12" stroke-linecap="round"/><circle cx="596" cy="120" r="34" fill="#5574ff"/></svg>`

const fixtureImageDark = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 720 360"><rect width="720" height="360" fill="#0d1119"/><rect x="48" y="48" width="624" height="264" rx="12" fill="#151b26" stroke="#313b4c"/><path d="M96 112h244M96 152h416M96 192h352" stroke="#a7b0c0" stroke-width="12" stroke-linecap="round"/><circle cx="596" cy="120" r="34" fill="#7890ff"/></svg>`

const fixtureLogo = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 240" role="img" aria-labelledby="title desc" color="#5574ff"><title id="title">M-PRESS crown M icon</title><desc id="desc">Geometric M with straight side pillars and an integrated three-point crown in negative space</desc><path fill="currentColor" fill-rule="evenodd" d="M20 30H220V210H20Z M66 30L120 89L174 30Z M68 210V134L96 185H99L120 113L141 185H144L172 134V210Z"/></svg>`

func escapedPreformatted(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = html.EscapeString(strings.TrimSuffix(value, "\n"))
	// Hide directive markers from the production component preprocessor. The
	// browser decodes the entity back to the exact source character.
	value = strings.ReplaceAll(value, "@", "&#64;")
	return strings.ReplaceAll(value, "\n", "&#10;")
}

func formatMPDForDisplay(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	formatted := make([]string, 0, len(lines))
	depth := 0
	fenceWidth := 0
	literalBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceWidth > 0 {
			formatted = append(formatted, displayIndent(depth, line))
			if run := leadingBackticks(trimmed); run >= fenceWidth && strings.TrimSpace(trimmed[run:]) == "" {
				fenceWidth = 0
			}
			continue
		}
		if literalBlock {
			if trimmed == "@end" {
				if depth > 0 {
					depth--
				}
				formatted = append(formatted, displayIndent(depth, line))
				literalBlock = false
			} else {
				formatted = append(formatted, displayIndent(depth, line))
			}
			continue
		}
		if run := leadingBackticks(trimmed); run >= 3 {
			formatted = append(formatted, displayIndent(depth, line))
			fenceWidth = run
			continue
		}
		if trimmed == "@end" {
			if depth > 0 {
				depth--
			}
			formatted = append(formatted, displayIndent(depth, line))
			continue
		}

		formatted = append(formatted, displayIndent(depth, line))
		name, directive := displayDirectiveName(trimmed)
		if !directive || name == "metadata" || mpdLeafDirectives[name] {
			continue
		}
		depth++
		literalBlock = name == "rawHTML" || name == "comment"
	}
	return strings.Join(formatted, "\n")
}

var mpdLeafDirectives = map[string]bool{
	"computed": true,
	"hr":       true,
	"image":    true,
	"import":   true,
	"include":  true,
	"input":    true,
	"link":     true,
	"qr":       true,
}

func displayIndent(depth int, line string) string {
	if strings.TrimSpace(line) == "" {
		return ""
	}
	return strings.Repeat("  ", depth) + line
}

func displayDirectiveName(line string) (string, bool) {
	if len(line) < 2 || line[0] != '@' || line[1] == '@' {
		return "", false
	}
	end := 1
	for end < len(line) {
		char := line[end]
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' {
			end++
			continue
		}
		break
	}
	return line[1:end], end > 1
}

func leadingBackticks(line string) int {
	count := 0
	for count < len(line) && line[count] == '`' {
		count++
	}
	return count
}

func categoryLabel(value string) string {
	words := strings.ReplaceAll(value, "-", " ")
	if words == "" {
		return words
	}
	return strings.ToUpper(words[:1]) + words[1:]
}

func slugID(value string) string {
	return strings.NewReplacer("/", "-", "_", "-").Replace(value)
}

func copyDirectory(source, target string) error {
	return filepath.WalkDir(source, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
}

// FileSet returns the relative files below root for deterministic comparisons.
func FileSet(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	sort.Strings(files)
	return files, err
}
