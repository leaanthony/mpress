---
title: M-Press documentation
description: Build modern, rich documentation with Markdown, not MDX.
layout: landing
translationKey: mpress-home
order: 1
---

@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Modern docs.
Rich components.
Just Markdown.
@end

M-Press gives you rich components, private search, reader accessibility controls, versioned releases, and safe AI-assisted translation in one fast Go binary. No Node.js. No MDX. No framework. Even this page is Markdown!

@actions
@button[Build your first site](/tutorials/first-site/){primary}
@button[See what is included](/features/){secondary}
@end
@end

@column{variant=site-screenshot|class=mp-home-site-screenshot-hero}
@image{light="/images/mpress-docs-light.png" dark="/images/mpress-docs-dark.png" alt="The generated M-Press tutorial site" expand}
@end
@end
@end

@section{variant=story|class=mp-home-story-stack}
@columns{variant=story}
@column{variant=story-copy}
## Stop maintaining a frontend just to publish words.

Documentation should not begin with a package manager, a framework upgrade, and a lockfile conflict. M-Press keeps the authoring model deliberately small: one Go binary, one YAML file, and ordinary Markdown.

- **No Node.js runtime.** Install one native executable locally and in CI.
- **No MDX coupling.** Your source stays readable outside M-Press.
- **No hidden build graph.** The generated result is portable static HTML.

@button[See the complete feature reference](/features/){secondary}
@end

@column{variant=story-visual|class=mp-home-stack-visual}
@file-tabs
[index.md]

```markdown
# Hello world

This site was built from Markdown.
```

[mpress.yaml]

```yaml
site:
  title: My Project
  description: My first M-Press site.
build:
  contentDir: docs
  outputDir: site
```

[Output]

@image{src="/images/mpress-hello-output.png" alt="A generated Hello world documentation site" expand}
@end
@end
@end
@end

@section{variant=story|class="mp-home-story-components mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Rich components from simple Markdown keywords.

Readers expect tabs, steps, terminals, diagrams, code annotations, file trees, and responsive navigation. Authors should not need MDX or a JavaScript package for each one. Write a clear keyword such as `@terminal`, `@steps`, or `@note`. M-Press turns it into an accessible, interactive component.

- **Search is automatic and private.** M-Press builds a static index for every language. Queries stay in the browser.
- **One keyword creates the rich component.** The source remains short, readable Markdown.
- **Components are included.** There is no package to install, import, or update for each pattern.
- **The output works without a framework runtime.** Host it anywhere that serves static files.

@button[Explore every component](/components/){secondary}
@end

@column{variant=story-visual|class=mp-home-components-visual}
@preview-tabs{source}
[Terminal]
@terminal{title="Build the documentation" language=bash frame=macos}
$ mpress build
Built 60 pages in 47 ms
@end

[Steps]
@steps
### Write Markdown

Keep the source readable in every editor.

### Build the site

Generate static HTML with one command.
@end

[Note]
@note{type=tip title="Included by default"}
The component, its theme, and its browser behaviour ship with M-Press.
@end

[File tree]
@filetree
docs/
  index.md  Landing page
  guide.md  Product guide
mpress.yaml  Site configuration
@end

[Cards + Links]
@cards{cols=2}
[Start a project](/tutorials/first-site/)
Create a complete site with one binary.

---

[Explore components](/components/)
See every component included with M-Press.
@end

@linkcard{title="Read the authoring guide" href="/authoring/" description="Learn the small set of Markdown conventions." icon="book-open"}

[Layout]
@container{display=grid|columns=2|gap=.75rem}
@tip[Static output]
The complete layout is generated as HTML.
@end
@info[Responsive]
Columns collapse cleanly on smaller screens.
@end
@end

[Explained code]
@explained
```go
func main() { // (1)
    mpress.Build() // (2)
}
```

(1) Start with a regular Go entry point.

(2) Build the complete static site.
@end

[QR]
@qr{url="https://m-press.me" size="150" label="Open the M-Press website"}

[Rich tables]
@table{header search filter sort paginate column-separators page-size=4}
| Page | Status | Updated |
| Installation | Published | Today |
| Configuration | Published | Yesterday |
| First project | In review | 2 days ago |
| Deployment | Published | 3 days ago |
| Accessibility | Draft | 5 days ago |
| Components | Published | 1 week ago |
@end
@end
@end
@end
@end

@section{variant=story|class="mp-home-story-accessibility mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## Accessibility belongs in every documentation site.

One presentation cannot suit every reader. The accessibility menu is enabled by default and stores each visitor's preferences only in their browser. Site owners do not receive health or preference data.

- **Reading controls.** Change text size, page width, font, spacing, and word emphasis.
- **Focus controls.** Dim distractions, follow a reading guide, and reduce motion.
- **Vision controls.** Increase contrast, underline links, and select colour profiles.

@button[Try the accessibility options](#){primary|action=accessibility}
@button[Read the accessibility reference](/accessibility/){secondary}
@end

@column{variant=story-visual|class=mp-home-accessibility-carousel}
@carousel{label="Accessibility options"}
## Reading that fits you

Choose a text size, readable font, relaxed spacing, bionic emphasis, and a page width that is comfortable on your screen.

- Text size: Default, Large, Larger
- Page width: Relative or fixed
- Readable font and relaxed spacing
---
## Keep your place

Reduce visual distraction without changing the content.

- Focus mode dims navigation
- Reading guide follows the pointer
- Reduced motion stops non-essential animation
---
## Improve colour and contrast

Make links and controls easier to identify.

- High contrast
- Underlined links
- Colour profiles for common colour-vision needs
@end
@end
@end
@end

@section{variant=story|class=mp-home-story-speed}
@columns{variant=story}
@column{variant=story-copy}
## Keep writing while the site keeps up.

Slow builds break concentration. M-Press watches your content, rebuilds automatically, and refreshes the browser. The same binary also checks links, audits quality, edits configuration, exports production files, and prepares deployment.

- **Immediate feedback.** Save a file and see the generated page update.
- **Visible evidence.** Every build and validation stage reports its real duration.
- **Quality in the same workflow.** Run strict checks and a Lighthouse audit before release.

@button[See how M-Press stays fast](/explanation/performance/){secondary}
@end

@column{variant=story-visual|class="mp-home-speed-visual mp-home-build-evidence"}
### Measured on this documentation set

**60 pages · 82 output files · 47 to 66 ms**

@capabilities
- **Build** 47 to 66 ms across two clean production builds
- **Search** Index generated with the pages
- **Validation** Links and assets collected during rendering
- **Optimisation** Native CSS and JavaScript minification
@end

**One binary, complete tooling**

Development server, configuration editor, checks, Lighthouse, ZIP export, deployment helpers, migration, translation, and version capture are ready from the same command.
@end
@end
@end

@section{variant=story|class=mp-home-story-global}
@columns{variant=story}
@column{variant=story-copy}
## Languages and versions belong in the core.

Translations become risky when code fences, component syntax, and links are treated as ordinary prose. Versions become confusing when old pages quietly change. M-Press models both concerns directly.

- **Safe translation.** Protect structure, use glossaries and style guides, and track stale, manual, conflicted, and reviewed text.
- **Independent language sites.** Build routes, navigation, and search for each language.
- **Immutable versions.** Capture, checksum, verify, and mount complete releases with a reader-facing selector.

@button[Read about translation](/translation/){secondary}
@button[Read about versioning](/versioning/){secondary}
@end

@column{variant=story-visual|class="mp-home-story-image mp-home-global-screenshot"}
@image{light="/images/mpress-language-version-light.png" dark="/images/mpress-language-version-dark.png" alt="The Cymraeg translation of an M-Press documentation site with language and version controls" expand}
@end
@end
@end

@section{variant=story|class="mp-home-story-custom mp-home-story-reversed"}
@columns{variant=story}
@column{variant=story-copy}
## A complete documentation workshop, ready in development mode.

The development site exposes the work that is usually hidden across scripts and services. Edit every supported setting with a structured form, inspect the generated file tree, run checks, review build performance, start a Lighthouse audit, export a ZIP, and prepare deployment without leaving the documentation.

- **One complete configuration.** Control identity, languages, links, files, theme, accessibility, layout, versions, translation, and deployment.
- **Fully customisable.** Set colour, typography, light and dark logos, navigation, social links, and custom CSS.
- **Responsive by default.** The same Markdown content supports a full product landing page and a focused mobile reading experience by default.

@button[Configure a site](/configuration/){secondary}
@end

@column{variant=story-visual|class="mp-home-story-image mp-home-config-screenshot"}
@image{light="/images/mpress-config-light.png" dark="/images/mpress-config-dark.png" alt="The structured M-Press configuration editor in development mode" expand}
@end
@end
@end

@section{variant=migration}
@columns{variant=migration}
@column
## Keep the content. Remove the JavaScript toolchain.
@end

@column
The importer brings across pages, supported components, navigation, languages, frontmatter, public assets, and configuration. It writes a report for every pattern that needs a human decision.

@button[Read the migration guide](/starlight/){secondary}
@end
@end
@end

@section{variant=final}
@column
## Build rich documentation without Node.

Create a working M-Press project. Accessibility, versions, translations, search, checks, components, and production optimisation are ready from the first build.
@end

@actions
@button[Start the tutorial](/tutorials/first-site/){primary}
@button[View the source](https://github.com/leaanthony/mpress){secondary}
@end
@end
