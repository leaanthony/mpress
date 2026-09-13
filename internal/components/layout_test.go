package components

import (
	"strings"
	"testing"
)

func TestMarkdownLayoutComponentsRenderNestedContent(t *testing.T) {
	input := `@section{variant=hero|aria-label="Product introduction"}
@columns{variant=hero}
@column{variant=hero-copy}
# Modern documentation

Plain Markdown stays readable.

@actions
@button[Start](/start/){primary}
@end
@end

@column
@docs-preview{brand="Product docs"|nav="START HERE:Overview*,Install"}
## Build your first site

Create the project.
@end
@end
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{
		`<section class="mpress-section mpress-section-hero mp-home-hero"`,
		`aria-label="Product introduction"`,
		`class="mpress-columns mpress-columns-hero mp-home-hero-grid"`,
		`<h1 id="modern-documentation">Modern documentation</h1>`,
		`class="mpress-actions mp-home-actions"`,
		`mpress-button-primary`,
		`class="mpress-doc-preview mpress-doc-preview-compact mp-home-product"`,
		`<h2 id="build-your-first-site">Build your first site</h2>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@end") || strings.Contains(got, "@section") {
		t.Fatalf("native layout syntax leaked into output: %s", got)
	}
	if strings.Contains(got, `<div class="mpress-actions mp-home-actions"><p>`) || strings.Contains(got, `</a></p>`) {
		t.Fatalf("action buttons were wrapped in paragraphs:\n%s", got)
	}
}

func TestActionsRenderAdjacentButtonsWithoutParagraphs(t *testing.T) {
	input := `@actions
@button{href="/start/" variant="primary"}
Start
@end
@button{href="/guide/" variant="secondary"}
Read the guide
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	if !strings.Contains(got, `>Start</a>`) || !strings.Contains(got, `>Read the guide</a>`) {
		t.Fatalf("actions missing buttons:\n%s", got)
	}
	if strings.Contains(got, `<p>`) || strings.Contains(got, `</p>`) {
		t.Fatalf("actions contain paragraph wrappers:\n%s", got)
	}
}

func TestNativeLeafImageDoesNotConsumeParentLayoutEnd(t *testing.T) {
	input := `@section{variant=story|class="story-image story-reversed"}
@columns{variant=story}
@column{variant=story-visual}
@image{light="/light.png" dark="/dark.png" alt="Product documentation"}
@end
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{
		`<section class="mpress-section mpress-section-story story-image story-reversed mp-home-section mp-home-story-section"`,
		`class="mpress-theme-image"`,
		`src="/light.png"`,
		`src="/dark.png"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@section") || strings.Contains(got, "@end") {
		t.Fatalf("native layout syntax leaked into output: %s", got)
	}
}

func TestLandingHeroWithActionsAndLeafImageClosesSection(t *testing.T) {
	input := `@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Modern docs.
Rich components.
Just Markdown.
@end

M-Press keeps the source ordinary Markdown.

@actions
@button[Build your first site](/tutorials/first-site/){primary}
@button[See what is included](/features/){secondary}
@end
@end

@column{variant=site-screenshot|class=mp-home-site-screenshot-hero}
@image{light="/images/light.png" dark="/images/dark.png" alt="The generated tutorial site"}
@end
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{
		`<section class="mpress-section mpress-section-hero mp-home-hero"`,
		`class="mpress-columns mpress-columns-hero mp-home-hero-grid"`,
		`class="mpress-actions mp-home-actions"`,
		`class="mpress-theme-image"`,
		`</section>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@section") || strings.Contains(got, "@end") {
		t.Fatalf("native landing syntax leaked into output: %s", got)
	}
}

func TestMarkdownLandingDataComponents(t *testing.T) {
	input := `@timeline
1. **Install M-Press** Place one executable on your path.
2. **Publish** Copy the output to a static host.
@end

@capabilities
- **Search** Build the search index.
- **Translations** Generate language-aware routes.
@end

@resources
[01 · Tutorial](/tutorial/)
## Build a site
Create working documentation.
---
[02 · How-to](/how-to/)
## Add a language
Generate translated routes.
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{`class="mpress-timeline mp-home-timeline"`, `class="mpress-timeline-entry mpress-step"`, `class="mpress-timeline-marker mpress-step-number"`, `class="mpress-timeline-body mpress-step-body"`, `class="mpress-step-title"`, "Install M-Press", "mp-home-capabilities", "language-aware routes", "mp-home-resources", `href="/tutorial/"`, "Build a site"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestMarkdownSalesStoryLayout(t *testing.T) {
	input := `@section{variant=story|class="story-muted story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Leave the toolchain behind

Write Markdown.
@end

@column{variant=story-visual}
~~~yaml
site:
  title: Product docs
~~~
@end
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{
		`mp-home-story-section`,
		`story-muted`,
		`story-reversed`,
		`mp-home-story`,
		`mp-home-story-copy`,
		`mp-home-story-visual`,
		`<code class="language-yaml">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestMarkdownLayoutLeafComponentsDoNotConsumeParentEnd(t *testing.T) {
	input := `@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
# Product documentation
@end
@column{variant=story-visual|class="story-image story-reversed"}
@image{light="/light.png" dark="/dark.png" alt="Documentation preview"}
@end
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{
		`<section class="mpress-section mpress-section-hero mp-home-hero"`,
		`class="mpress-column mpress-column-story-visual story-image story-reversed mp-home-story-visual"`,
		`class="mpress-theme-image"`,
		`src="/light.png"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@section") || strings.Contains(got, "@end") {
		t.Fatalf("native layout syntax leaked into output:\n%s", got)
	}
}

func TestPreviewTabsAndCompactCallout(t *testing.T) {
	input := `@preview-tabs
[Terminal]

~~~sh
mpress build
~~~

[Steps]

One, two, three.

[File tree]

docs/
@end

@callout{title="Built in"}
No framework is required.
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{"mpress-preview-tabs", "mp-home-demo-tabs", `role="tab"`, `aria-selected="true"`, `role="tabpanel"`, "Terminal", "Steps", "File tree", "mpress build", "mpress-compact-callout", "Built in", "No framework is required"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if panelCount := strings.Count(got, `role="tabpanel"`); panelCount != 3 {
		t.Fatalf("expected 3 preview tab panels, got %d:\n%s", panelCount, got)
	}
}

func TestPreviewTabsCanShowInertMarkdownSource(t *testing.T) {
	input := `@preview-tabs{source}
[Terminal]
@terminal{title="Build"}
$ mpress build
@end

[Explained]
@explained
~~~go
func main() { // (1)
}
~~~

(1) Entry point.
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	if count := strings.Count(got, `class="mpress-preview-source"`); count != 2 {
		t.Fatalf("expected a source panel for both tabs, got %d:\n%s", count, got)
	}
	for _, want := range []string{
		`class="mpress-preview-source-label">Markdown`,
		`@terminal{title=&#34;Build&#34;}`,
		`~~~go`,
		`class="mpress-terminal mpress-terminal-macos"`,
		`class="mpress-explained"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestFileTabsRenderSourceFilesAndOutput(t *testing.T) {
	input := `@file-tabs
[index.md]

~~~markdown
# Hello world
~~~

[mpress.yaml]

~~~yaml
site:
  title: Hello world
~~~

[Output]

@image{src="/hello.png" alt="Generated hello world site" expand}
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{`class="mpress-tabs mpress-file-tabs"`, `>index.md</button>`, `>mpress.yaml</button>`, `>Output</button>`, `class="mpress-image-expand"`, `src="/hello.png"`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if panelCount := strings.Count(got, `role="tabpanel"`); panelCount != 3 {
		t.Fatalf("expected 3 file tab panels, got %d:\n%s", panelCount, got)
	}
}

func TestFileTabsKeepMultilineMarkdownAsLiteralCode(t *testing.T) {
	input := `@file-tabs
[index.md]

~~~markdown
---
title: Hello world
---

# Hello world

This site was built from Markdown.
~~~

[Output]

Ready.
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	for _, want := range []string{
		`<pre><code class="language-markdown">`,
		`# Hello world`,
		`This site was built from Markdown.`,
		`# Hello world&#10;&#10;This site was built from Markdown.`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{`<h1`, `<p>This site was built from Markdown.`} {
		if strings.Contains(got, unwanted) {
			t.Errorf("file source became live page markup %q:\n%s", unwanted, got)
		}
	}
}

func TestHeadlinePreservesAuthoredLines(t *testing.T) {
	input := `@headline
Modern docs.
Rich components.
Just Markdown.
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v\n%s", warnings, got)
	}
	if want := `<h1 class="mpress-headline">Modern docs.<br>Rich components.<br>Just Markdown.</h1>`; !strings.Contains(got, want) {
		t.Fatalf("output missing %q:\n%s", want, got)
	}
}
