package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leaanthony/mpress/internal/mpd"
	"github.com/leaanthony/mpress/internal/site"
)

func TestImportStarlightProducesBuildableProject(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Imported Docs',
  description: 'A fixture',
  locales: { root: { label: 'English', lang: 'en' }, fr: { label: 'Français', lang: 'fr' } },
  sidebar: [{ label: 'Guide', link: '/guide/' }]
})`)
	writeImportFixture(t, source, "src/content/docs/index.mdx", `---
title: Home
---
import { Aside, LinkCard, Card } from '@astrojs/starlight/components';

<Aside type="tip">Keep it small.</Aside>

<LinkCard title="Guide" href="/guide/" description="Read it" />

<Card title="Speed" icon="rocket">Fast builds.</Card>
`)
	writeImportFixture(t, source, "src/content/docs/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n")
	writeImportFixture(t, source, "src/content/docs/tutorials/03-notes.mdx", "---\ntitle: Notes\n---\n\n# Notes\n")
	writeImportFixture(t, source, "public/logo.svg", "<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	result, err := site.Build(output, site.BuildOptions{Strict: true})
	if err != nil {
		converted, _ := os.ReadFile(filepath.Join(output, "content", "index.mpd"))
		t.Fatalf("imported project did not build strictly: %v; %#v\n%s", err, result.Diagnostics, converted)
	}
	if result.Pages < 2 {
		t.Fatalf("expected imported pages, got %d", result.Pages)
	}
	if _, err := os.Stat(filepath.Join(output, "site", "tutorials", "03-notes", "index.html")); err != nil {
		t.Fatalf("importer did not preserve Starlight's ordered-filename route: %v", err)
	}
	page, err := os.ReadFile(filepath.Join(output, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Imported Docs", "mpress-admonition", "mpress-linkcard"} {
		if !strings.Contains(string(page), want) {
			t.Errorf("imported homepage missing %q", want)
		}
	}
	if strings.Contains(string(page), `mpress-card-title">###`) {
		t.Fatal("native MPD card title leaked Markdown heading syntax")
	}
	if _, err := os.Stat(filepath.Join(output, "migration-report.md")); err != nil {
		t.Fatalf("migration report missing: %v", err)
	}
	configFile, err := os.ReadFile(filepath.Join(output, "mpress.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configFile), `fr: "Français"`) {
		t.Fatalf("language labels were not preserved: %s", configFile)
	}
}

func TestImportStarlightPublishesGeneratedSectionIndexes(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  editLink: { baseUrl: 'https://github.com/example/docs/edit/main/docs' },
  sidebar: []
})`)
	writeImportFixture(t, source, "src/content/docs/guides/install.md", "---\ntitle: Install\n---\n\n[All guides](/guides/)\n")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	if _, err := site.Build(output, site.BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "site", "guides", "index.html")); err != nil {
		t.Fatalf("generated section index was not published: %v", err)
	}
	generatedHTML, err := os.ReadFile(filepath.Join(output, "site", "guides", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generatedHTML), "Edit page") {
		t.Fatalf("synthetic section index advertised a source file that does not exist:\n%s", generatedHTML)
	}
}

func TestImportStarlightDoesNotDuplicateExistingSectionOrLocaleIndexes(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  locales: { root: { label: 'English', lang: 'en' }, id: { label: 'Bahasa Indonesia', lang: 'id' } },
  sidebar: []
})`)
	writeImportFixture(t, source, "src/content/docs/experimental/index.mdx", "---\ntitle: Experimental\n---\n\n# Experimental\n")
	writeImportFixture(t, source, "src/content/docs/experimental/wake.mdx", "---\ntitle: Wake\n---\n\n# Wake\n")
	writeImportFixture(t, source, "src/content/docs/blog/index.mdx", "---\ntitle: Blog\n---\n\n# Blog\n")
	writeImportFixture(t, source, "src/content/docs/blog/first-post.mdx", "---\ntitle: First post\n---\n\n# First post\n")
	writeImportFixture(t, source, "src/content/docs/id/index.mdx", "---\ntitle: Beranda\n---\n\n# Beranda\n")
	writeImportFixture(t, source, "src/content/docs/id/guide/start.mdx", "---\ntitle: Mulai\n---\n\n# Mulai\n")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	if _, err := site.Build(output, site.BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	for _, unexpected := range []string{
		"experimental-migrated.mpd",
		"blog-migrated.mpd",
		"id-migrated.mpd",
	} {
		if _, err := os.Stat(filepath.Join(output, "content", unexpected)); !os.IsNotExist(err) {
			t.Fatalf("existing section or locale index produced duplicate %s", unexpected)
		}
	}
	report, err := os.ReadFile(filepath.Join(output, "migration-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(report), "index.mpd` — route-collision") {
		t.Fatalf("generated section index collided with an existing page:\n%s", report)
	}
}

func TestImportStarlightPreservesSourceModificationTime(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  editLink: { baseUrl: 'https://github.com/example/docs/edit/main/docs' },
  sidebar: []
})`)
	pagePath := filepath.Join(source, "src", "content", "docs", "guide", "start.mdx")
	writeImportFixture(t, source, "src/content/docs/guide/start.mdx", "---\ntitle: Start\n---\n\n# Start\n")
	modified := time.Date(2024, time.March, 4, 5, 6, 7, 0, time.UTC)
	if err := os.Chtimes(pagePath, modified, modified); err != nil {
		t.Fatal(err)
	}

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	converted := filepath.Join(output, "content", "guide", "start.mpd")
	info, err := os.Stat(converted)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(modified) {
		t.Fatalf("converted page modification time = %s, want %s", info.ModTime(), modified)
	}
	if _, err := site.Build(output, site.BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(output, "site", "guide", "start", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), `datetime="2024-03-04T05:06:07Z"`) {
		t.Fatalf("generated page did not preserve the source timestamp:\n%s", html)
	}
	if !strings.Contains(string(html), `href="https://github.com/example/docs/edit/main/docs/src/content/docs/guide/start.mdx"`) {
		t.Fatalf("generated edit link did not preserve the original Starlight source path:\n%s", html)
	}
}

func TestImportStarlightResolvesRouteCollisionsAndWailsHero(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Wails',
  locales: { root: { label: 'English', lang: 'en' } },
  sidebar: []
})`)
	writeImportFixture(t, source, "src/content/docs/index.mdx", `---
title: Home
template: splash
hero:
  tagline: Build desktop apps
---
import MorphText from "@components/MorphText.astro";
import { LinkButton } from '@astrojs/starlight/components';

<MorphText />

## Start
Welcome.

<LinkButton href="/tutorials/01-first-step">Start tutorial</LinkButton>
`)
	writeImportFixture(t, source, "src/content/docs/tutorials/01-first-step.mdx", "---\ntitle: First step\n---\n\n# First step\n")
	writeImportFixture(t, source, "src/content/docs/contributing.mdx", "---\ntitle: Contributing\n---\n\nCanonical page.\n")
	writeImportFixture(t, source, "src/content/docs/contributing/index.mdx", "---\ntitle: Technical overview\n---\n\nColliding page.\n")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	result, err := site.Build(output, site.BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("imported project did not build strictly: %v; %#v", err, result.Diagnostics)
	}
	page, err := os.ReadFile(filepath.Join(output, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "mpress-unsupported") || !strings.Contains(string(page), "Start") || !strings.Contains(string(page), "mpress-frontmatter-hero") || !strings.Contains(string(page), "landing-page") {
		t.Fatalf("Wails hero fallback was not portable: %s", page)
	}
	if !strings.Contains(string(page), `titleLine.className = 'morph-title-line'`) || !strings.Contains(string(page), `wordWrap.className = 'morph-word-title'`) {
		t.Fatalf("Wails morphing hero title lost its inline first line: %s", page)
	}
	if strings.Contains(string(page), `.innerHTML`) || !strings.Contains(string(page), `title.replaceChildren`) {
		t.Fatalf("Wails morphing hero must construct trusted DOM nodes without reinterpreting text as HTML: %s", page)
	}
	if !strings.Contains(string(page), `href="/tutorials/01-first-step/"`) {
		t.Fatalf("converted HTML link did not use the canonical M-Press route: %s", page)
	}
	report, err := os.ReadFile(filepath.Join(output, "migration-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(report), "route-collision") {
		t.Fatalf("migration report lost review findings: %s", report)
	}
	if _, err := os.Stat(filepath.Join(output, "site", "contributing-migrated", "index.html")); err != nil {
		t.Fatalf("colliding source page was not preserved at an alternate route: %v", err)
	}
}

func TestImportStarlightPreservesCodeExamplesAndConvertsLinkCards(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  locales: { root: { label: 'English', lang: 'en' } },
  sidebar: []
})`)
	writeImportFixture(t, source, "src/content/docs/index.mdx", `---
title: Fixture
---
import { CardGrid, LinkCard } from '@astrojs/starlight/components';

<CardGrid>
  <LinkCard title="Agent control" description="Inspect a running application." href="/agent/" />
  <LinkCard title="Agent reference" description="Read the complete reference." href="/agent/" />
</CardGrid>

<LinkCard title="Standalone card" description="A leaf component outside a grid." href="/agent/" />

`+"```javascript"+`
import { Events } from '@wailsio/runtime'

export function View() {
  return <Router><Route path="/" /></Router>
}
`+"```"+`
`)
	writeImportFixture(t, source, "src/content/docs/agent.md", "---\ntitle: Agent\n---\n\nAgent docs.\n")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	converted, err := os.ReadFile(filepath.Join(output, "content", "index.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(converted)
	for _, want := range []string{
		"import { Events } from '@wailsio/runtime'",
		`return <Router><Route path="/" /></Router>`,
		`@linkcard title="Agent control" href="/agent/" description="Inspect a running application."`,
		`@linkcard title="Agent reference" href="/agent/" description="Read the complete reference."`,
		`@linkcard title="Standalone card" href="/agent/" description="A leaf component outside a grid."`,
		"Inspect a running application.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("converted document lost %q:\n%s", want, text)
		}
	}
	report, err := os.ReadFile(filepath.Join(output, "migration-report.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(report), "@wailsio/runtime") || strings.Contains(string(report), "<Router>") {
		t.Fatalf("code examples were incorrectly reported as MDX: %s", report)
	}
	result, err := site.Build(output, site.BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("converted LinkCard did not build strictly: %v; %#v", err, result.Diagnostics)
	}
	htmlPage, err := os.ReadFile(filepath.Join(output, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(htmlPage), "mpress-linkcard") {
		t.Fatalf("converted LinkCard did not render: %s", htmlPage)
	}
	if strings.Contains(string(htmlPage), "<p>@end</p>") {
		t.Fatalf("converted LinkCard leaked a component closer: %s", htmlPage)
	}
	if closers := strings.Count(text, "\n@end\n"); closers != 1 {
		t.Fatalf("leaf LinkCards must not emit component closers; got %d closers:\n%s", closers, text)
	}
}

func TestImportStarlightKeepsLocalizedSlugRoutes(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  locales: {
    root: { label: 'English', lang: 'en' },
    id: { label: 'Bahasa Indonesia', lang: 'id' }
  },
  sidebar: []
})`)
	for _, fixture := range []struct{ name, title string }{
		{"src/content/docs/blog/release.md", "Release"},
		{"src/content/docs/id/blog/release.md", "Rilis"},
	} {
		writeImportFixture(t, source, fixture.name, "---\nslug: blog/release\ntitle: "+fixture.title+"\n---\n\n# "+fixture.title+"\n")
	}
	writeImportFixture(t, source, "src/content/docs/id/index.md", "---\ntitle: Beranda\n---\n\n# Beranda\n")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(output, "content", "blog", "release.mpd"),
		filepath.Join(output, "content", "id", "blog", "release.mpd"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("localized slug route missing at %s: %v", path, err)
		}
	}
	result, err := site.Build(output, site.BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("localized slug routes did not build: %v; %#v", err, result.Diagnostics)
	}
	for _, path := range []string{
		filepath.Join(output, "site", "blog", "release", "index.html"),
		filepath.Join(output, "site", "id", "blog", "release", "index.html"),
		filepath.Join(output, "site", "id", "index.html"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("localized built route missing at %s: %v", path, err)
		}
	}
}

func TestImportStarlightConvertsReactStyleAttributes(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  locales: { root: { label: 'English', lang: 'en' } },
  sidebar: []
})`)
	writeImportFixture(t, source, "src/content/docs/index.mdx", `---
title: Sponsors
---

<img
  src="/sponsor.png"
  style={{ margin: "auto", width: "100%", maxWidth: "800px" }}
  alt="Sponsors"
/>
`)
	writeImportFixture(t, source, "public/sponsor.png", "fixture")

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	converted, err := os.ReadFile(filepath.Join(output, "content", "index.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	want := `style="margin: auto; width: 100%; max-width: 800px"`
	if !strings.Contains(string(converted), want) || strings.Contains(string(converted), "style={{") {
		t.Fatalf("React style attribute was not converted:\n%s", converted)
	}
	result, err := site.Build(output, site.BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("converted image did not build strictly: %v; %#v", err, result.Diagnostics)
	}
	page, err := os.ReadFile(filepath.Join(output, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "style={{") || !strings.Contains(string(page), "max-width: 800px") {
		t.Fatalf("React style leaked into generated HTML:\n%s", page)
	}
}

func TestImportStarlightPreservesStepsAroundNestedTabs(t *testing.T) {
	source := t.TempDir()
	output := t.TempDir()
	writeImportFixture(t, source, "astro.config.mjs", `starlight({
  title: 'Fixture',
  locales: { root: { label: 'English', lang: 'en' } },
  sidebar: []
})`)
	writeImportFixture(t, source, "src/content/docs/index.mdx", `---
title: Install
---
import { Steps, Tabs, TabItem } from '@astrojs/starlight/components';

<Steps>

1. **Install Go**

   Use the installer.

   <Tabs syncKey="os">
     <TabItem label="Windows">
       Run the Windows installer.
     </TabItem>
     <TabItem label="Linux">
       Use your package manager.
     </TabItem>
   </Tabs>

2. **Verify**

   Run the check.

</Steps>
`)

	if err := ImportStarlight(source, output); err != nil {
		t.Fatal(err)
	}
	converted, err := os.ReadFile(filepath.Join(output, "content", "index.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(converted), "@steps") || !strings.Contains(string(converted), `@tabs sync-key="os"`) || !strings.Contains(string(converted), `@tab label="Windows"`) {
		t.Fatalf("nested step and tab structure was not preserved:\n%s", converted)
	}
	result, err := site.Build(output, site.BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("nested steps and tabs did not build strictly: %v; %#v", err, result.Diagnostics)
	}
	page, err := os.ReadFile(filepath.Join(output, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`class="mpress-timeline mpress-steps"`, `class="mpress-timeline-entry mpress-step"`, `class="mpress-tabs"`, `role="tablist"`} {
		if !strings.Contains(string(page), want) {
			t.Fatalf("built page lost %q:\n%s", want, page)
		}
	}
}

func TestConvertStarlightLandingTerminals(t *testing.T) {
	input := strings.Join([]string{
		"---",
		"title: Wails",
		"template: splash",
		"---",
		"",
		"## Quickstart",
		"",
		"```bash",
		"# Install the Wails CLI",
		"wails3 setup",
		"```",
		"",
	}, "\n")
	got := convertStarlightLandingTerminals(input)
	for _, want := range []string{`@terminal{frame=macos|prompt=none}`, "# Install the Wails CLI", "wails3 setup", "@end"} {
		if !strings.Contains(got, want) {
			t.Errorf("converted landing page missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "```bash") {
		t.Fatalf("bash fence was not converted:\n%s", got)
	}
}

func TestStarlightCustomCSSUsesStrongAccent(t *testing.T) {
	source := t.TempDir()
	writeImportFixture(t, source, "src/stylesheets/extra.css", `
:root {
  --sl-color-accent: #e90000;
  --sl-color-accent-high: #f53d3d;
}

:root[data-theme='dark'] {
  --sl-color-accent: #ef2222;
  --sl-color-accent-high: #ff5555;
}
`)
	got := starlightCustomCSS(source)
	for _, want := range []string{
		`--wails-accent-strong: #f53d3d`,
		`--wails-accent-strong: #ff5555`,
		`--wails-accent-link: #e90000`,
		`--wails-accent-link: #ff5555`,
		`:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .primary-links { order: 4; height: 32px; margin-right: -.9rem; padding-inline: .65rem; justify-content: center; border-left: 1px solid var(--border); }`,
		`:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .header-utility-cluster { order: 5; }`,
		`:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .header-utility-cluster > .header-group::before { display: none; }`,
		`:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .accessibility-select,`,
		`:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-toggle { width: 34px; min-width: 34px; height: 34px; min-height: 34px; justify-content: center; }`,
		`:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .language-select .utility-menu-trigger { width: 34px; min-width: 34px; justify-content: center; }`,
		`background: var(--wails-accent-strong)`,
		`.docs-page .sidebar { z-index: 1; padding: 0;`,
		`.docs-page .sidebar nav > * + * { margin-top: 8px; }`,
		`.docs-page .mpress-step-title { margin: 0 0 8px; color: var(--muted); font-size: 21px; font-weight: 600; line-height: 30px; }`,
		`background: var(--accent-fill)`,
		`color: var(--accent-fill-text)`,
		`color: var(--wails-toc-muted); font-size: 13px; line-height: 16.25px`,
		`.docs-page .layout {
  gap: 0;`,
		`.docs-page .docs-stage { container-type: inline-size; grid-template-columns: minmax(0, 1fr) minmax(0, var(--layout-content-width)) minmax(0, 1fr) minmax(0, var(--layout-toc-width)); width: 100%; }`,
		`.docs-page .docs-stage > main { grid-column: 2; }`,
		`.docs-page .docs-stage > .toc { grid-column: 4; }`,
		`.docs-page article > h1 { display: block; max-width: none; margin: 0 0 12px;`,
		`.docs-page article > h1 + .page-lead { position: relative; box-sizing: border-box; width: calc(100% + 48px + max(0px, (100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2));`,
		`.docs-page article > h1:not(:has(+ .page-lead)) { position: relative; box-sizing: border-box; width: calc(100% + 48px + max(0px, (100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2));`,
		`.docs-page article > h1:not(:has(+ .page-lead))::after { content: ""; position: absolute; right: 100%; bottom: -1px; width: 100vw;`,
		`.docs-page article > .page-lead + h2 { margin-top: 0; }`,
		`.docs-page .toc { min-width: 0; gap: 0; padding: 16px 17px; overflow-x: hidden; border-left: 1px solid var(--surface); }`,
		`.docs-page .mpress-admonition-warning,`,
		`--notice-bg: hsl(41, 39%, 22%); --notice-ink: hsl(41, 82%, 87%);`,
		`.docs-page .mpress-admonition-titlebar { gap: 8px; margin: 0; color: var(--notice-ink);`,
		`.docs-page .mpress-admonition-body :is(a, em) { color: var(--notice-ink); }`,
		`.docs-page .mpress-admonition-body code:not(pre code) { background: color-mix(in srgb, var(--notice-ink) 12%, transparent); color: var(--notice-text); }`,
		`html[data-theme="light"] .docs-page .mpress-admonition-warning,`,
		`--notice-bg: hsl(41, 90%, 88%); --notice-ink: hsl(41, 80%, 25%);`,
		"@media (max-width: 1050px) {\n  .docs-page .layout { grid-template-columns: 295px minmax(0, 1fr); gap: 0; }",
		`.docs-page .docs-stage { display: block; width: 100%; }`,
		`.docs-page article > h1 + .page-lead { width: calc(100% + 32px); margin-left: -16px; padding-right: 16px; padding-left: 16px; }`,
		`.docs-page .toc { position: fixed; z-index: 8; top: 64px; right: 0; left: 295px; display: flex;`,
		`.landing-page .mpress-frontmatter-hero-tagline { margin-top: 16px; }`,
		`.mpress-frontmatter-hero-action:not(.mpress-frontmatter-hero-action-primary) { border-color: var(--text); background: transparent; color: var(--text); }`,
		`.landing-page > footer { width: min(1080px, calc(100% - 3rem)); max-width: none; padding-inline: 0; }`,
		`@media (max-width: 420px) {`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("custom CSS missing %q", want)
		}
	}
	if strings.Contains(got, `.primary-links { order: 4; height: 32px; margin-right: -.9rem; padding-inline: .65rem; justify-content: center; border-right:`) {
		t.Error("primary navigation still draws a duplicate right separator")
	}
	if strings.Contains(got, `grid-template-columns: 300px minmax(0, 710px)`) {
		t.Error("Starlight import still pins the article to a fixed left-aligned grid track")
	}
}

func TestContributionFromEditURL(t *testing.T) {
	repository, branch := contributionFromEditURL("https://github.com/wailsapp/wails/edit/master/docs/src/content/docs")
	if repository != "https://github.com/wailsapp/wails.git" || branch != "master" {
		t.Fatalf("unexpected contribution target: %q %q", repository, branch)
	}
	if repository, branch := contributionFromEditURL("https://example.com/docs/edit/main"); repository != "" || branch != "" {
		t.Fatalf("non-GitHub target should not be inferred: %q %q", repository, branch)
	}
}

func TestNormaliseImportedHeadingLevelsSkipsCodeFences(t *testing.T) {
	input := "# Introduction\n\n#### Missing command\n\n```md\n# Literal title\n###### Literal example\n```\n\n#### Another problem\n"
	got := normaliseImportedHeadingLevels(input)
	for _, want := range []string{"## Introduction", "### Missing command", "# Literal title", "###### Literal example", "#### Another problem"} {
		if !strings.Contains(got, want) {
			t.Fatalf("normalised headings missing %q:\n%s", want, got)
		}
	}
}

func TestMarkdownToMPDNormalizesStarlightStructures(t *testing.T) {
	source := `---
title: Import fixture
---

:::note{type="tip" title="Quoted title with spaces"}
Useful.
:::

:::filetree
    - root/
        - child.txt
:::

3. Add:
   * Unit tests
   * A regression test if you touched
     the bindings generator

` + "```go title=\"main.go\" {2-3} ins={3}\nfunc main() {}\n```\n" + `
@contributor fixed this.
` + "\nUse `Command`+` for the shortcut.\n" + "\n@terminal{frame=macos|prompt=none}\nmpress build --strict\n@end\n"
	got, err := markdownToMPD(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`@note type="tip" title="Quoted title with spaces"`,
		"@filetree\n- root/\n  - child.txt",
		"3. Add:\n  - Unit tests\n  - A regression test if you touched the bindings generator",
		"```go {title=\"main.go\" highlight=\"2-3\" ins=\"3\"}",
		`\@contributor fixed this.`,
		"Use `Command`+\\` for the shortcut.",
		"@terminal frame=\"macos\" prompt=\"none\"\nmpress build --strict\n@end",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("converted MPD missing %q:\n%s", want, got)
		}
	}
}

func TestMarkdownToMPDCanonicalizesTildeFencesAndTableAlignment(t *testing.T) {
	source := `~~~yaml
site:
  links:
    - Blog
~~~

| Left | Centre | Right |
| :--- | :----: | ----: |
| one | two | three |
`
	got, err := MarkdownToMPD(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"```yaml\nsite:\n  links:\n    - Blog\n```",
		`@table header=true align=["left","center","right"]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("converted MPD missing %q:\n%s", want, got)
		}
	}
	document := mpd.Parse("converted.mpd", []byte(got))
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("converted MPD is invalid: %#v\n%s", document.Diagnostics, got)
		}
	}
}

func TestWailsHighCoverageStarlightCorpusConvertsToValidMPD(t *testing.T) {
	tests := []struct {
		path     string
		features []string
		source   string
		want     []string
	}{
		{
			path:     "index.mdx",
			features: []string{"landing frontmatter", "MorphText", "CardGrid", "Card"},
			source: `---
title: Wails
template: splash
hero:
  title: Build desktop applications using Go
---
<MorphText />
<CardGrid>
  <Card title="Performance That Users Notice" icon="rocket">Native performance.</Card>
</CardGrid>
`,
			want: []string{"template = \"splash\"", "@cards", `@card title="🚀 Performance That Users Notice"`},
		},
		{
			path:     "quick-start/installation.mdx",
			features: []string{"Steps", "nested Tabs", "Card", "terminal fence"},
			source: `---
title: Installation
---
<Steps>
1. **Install Go**
   <Tabs syncKey="os">
     <TabItem label="Linux">
       ` + "```bash\ngo version\n```" + `
     </TabItem>
   </Tabs>
2. **Build an app**
   <Card title="First app" icon="rocket">Continue.</Card>
</Steps>
`,
			want: []string{"@steps", `@step title="Install Go"`, "@tabs", `@tab label="Linux"`, "```bash", "@card"},
		},
		{
			path:     "getting-started/your-first-app.mdx",
			features: []string{"Steps", "FileTree", "deep nested lists"},
			source: `---
title: Your first app
---
<Steps>
1. **Explore the project**
    <FileTree>
    - build/
        - darwin/
            - Info.plist
    - frontend/
    </FileTree>
</Steps>
`,
			want: []string{"@steps", "@filetree", "- build/", "  - darwin/", "    - Info.plist"},
		},
		{
			path:     "tutorials/04-self-update-a-wails-app.mdx",
			features: []string{"asset import", "Steps", "Aside", "Image", "Tabs"},
			source: `---
title: Self update
---
import updateReady from '../../assets/update-ready.png';
<Steps>
1. **Configure updates**
   <Aside type="caution" title="Run it in a goroutine">Do not block startup.</Aside>
   <Tabs><TabItem label="Go">Use the service.</TabItem></Tabs>
   <Image src={updateReady} alt="Update ready" />
</Steps>
`,
			want: []string{"@steps", `@note type="caution" title="Run it in a goroutine"`, "@tabs", "![Update ready](/assets/update-ready.png)"},
		},
		{
			path:     "community/showcase/index.mdx",
			features: []string{"ShowcaseImage", "Steps", "generated showcase grid"},
			source: `---
title: Showcase
---
<ShowcaseImage
  entries={[
    {
      thumbnail: import("../../../../assets/showcase-images/app.webp"),
      href: "/community/showcase/app",
      title: "Application",
    },
  ]}
/>
<Steps>
1. **Submit an app**
   Open a pull request.
</Steps>
`,
			want: []string{"@rawHTML", "mpress-showcase-grid", "@steps", `@step title="Submit an app"`},
		},
		{
			path:     "experimental/index.mdx",
			features: []string{"Aside", "CardGrid", "LinkCard"},
			source: `---
title: Experimental
---
<Aside type="caution" title="Here be experiments">APIs can change.</Aside>
<CardGrid>
  <LinkCard title="New API" href="/experimental/new/" description="Try it" />
</CardGrid>
`,
			want: []string{"@note", "@container", "@linkcard"},
		},
		{
			path:     "guides/customising-windows.mdx",
			features: []string{"Badge", "fence title", "line highlights"},
			source: `---
title: Custom windows
---
<Badge text="Windows" variant="tip" />
` + "```go title=\"main.go\" {2-3}\nfunc main() {\n}\n```\n",
			want: []string{"@rawHTML", "```go {title=\"main.go\" highlight=\"2-3\"}"},
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			converted := convertStarlightContentWithRoutes(test.source, "/", nil)
			mpdSource, err := markdownToMPD(converted)
			if err != nil {
				t.Fatal(err)
			}
			document := mpd.Parse(test.path, []byte(mpdSource))
			if len(document.Diagnostics) != 0 {
				t.Fatalf("features %v produced MPD diagnostics: %#v\n%s", test.features, document.Diagnostics, mpdSource)
			}
			for _, want := range test.want {
				if !strings.Contains(mpdSource, want) {
					t.Errorf("features %v missing %q:\n%s", test.features, want, mpdSource)
				}
			}
		})
	}
}

func writeImportFixture(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
