---
title: Everything a documentation site should do
description: Rich authoring, reader controls, languages, versions, search, checks, and deployment in one fast workflow.
layout: landing
translationKey: mpress-features
order: 5
---

@section{variant=hero|class=mp-features-hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Write Markdown.
Ship a complete product.
@end

M-Press turns ordinary Markdown into fast, searchable, accessible documentation. Rich components, translations, versions, checks, and deployment work together from the first build.

@actions
@button[Build your first site](/tutorials/first-site/){primary}
@button[Explore components](/components/){secondary}
@end
@end

@column{variant=story-visual}
@terminal{title="A complete production build" language=bash frame=macos}
$ mpress build --strict
Parsed 511 pages
Built search indexes for 8 languages
Checked links and assets
Generated static site in 3.2 s
@end
@end
@end
@end

@section{variant=story}
@columns{variant=story}
@column{variant=story-copy}
## Rich documentation without a frontend project.

Use short, readable keywords for the patterns technical readers expect. M-Press supplies the markup, behaviour, responsive layout, and theme.

- `@terminal` creates a copy-safe command block.
- `@steps` explains a procedure with clear progress.
- `@note` adds semantic guidance with the right visual weight.
- Tabs, file trees, diffs, annotated code, tables, video, and API views are ready too.

@button[See the component showcase](/components/){secondary}
@end

@column{variant=story-visual}
@preview-tabs
[Terminal]
@terminal{title="Start the development server" language=bash frame=macos}
$ mpress dev
Ready at http://127.0.0.1:4174
@end

[Steps]
@steps
### Write normally

Keep content readable in any text editor.

### Build confidently

M-Press checks the site before it ships.
@end

[Guidance]
@note{type=tip title="One keyword, complete behaviour"}
The component is keyboard accessible, responsive, and themed by default.
@end
@end
@end
@end
@end

@section{variant=story|class=mp-features-accessibility}
@columns{variant=story}
@column{variant=story-copy}
## Let every reader shape the experience.

Accessibility is not a separate build or a hidden preference page. The reader menu stays in the navbar and saves choices only in that browser.

- Resize text and the reading column.
- Switch between a full-width and fixed-width site shell.
- Change type, spacing, contrast, link treatment, and colour profiles.
- Reduce motion, dim distractions, or follow a reading guide.

@button[Open the live accessibility menu](#){primary|action=accessibility}
@end

@column{variant=story-visual|class=mp-features-a11y-preview}
@accessibility-demo
@end
@end
@end
@end

@section{variant=story|class=mp-home-story-reversed}
@columns{variant=story}
@column{variant=story-copy}
## Languages and versions are first-class content.

A translated site is more than copied paragraphs. M-Press keeps navigation, search, routes, components, code, links, metadata, and missing-page policy coherent for every language.

Version releases are immutable, checksummed static artifacts. Readers switch language or version from the same navbar that they already understand.

@button[Translate documentation](/translation/){secondary}
@button[Publish versioned docs](/versioning/){secondary}
@end

@column{variant=story-visual}
@filetree
docs/ — Source language
  en/ — English
    quick-start.md
    configuration.md
  fr/ — French
    quick-start.md
    configuration.md
.mpress/versions/ — Verified releases
  v1.0.0/
  v1.1.0/
@end
@end
@end
@end

@section{variant=story}
@columns{variant=story}
@column{variant=story-copy}
## The development server is your control room.

Edit content with fast rebuilds and browser refresh. Use the project workspace to configure the complete YAML file, run checks, inspect timings, audit Lighthouse scores, export a ZIP, translate content, create versions, and deploy.

Strict builds catch duplicate routes, unsupported MDX, broken links, missing assets, and invalid configuration before release.

@button[See why builds are fast](/explanation/performance/){secondary}
@end

@column{variant=story-visual}
@image{light="/images/mpress-config-light.png" dark="/images/mpress-config-dark.png" alt="M-Press project settings with structured controls for the complete site configuration"}
@end
@end
@end

@section{variant=story|class=mp-home-story-reversed}
@columns{variant=story}
@column{variant=story-copy}
## Static output that stays portable.

The production result is HTML, CSS, JavaScript, JSON, and assets. Serve it from any static host. No M-Press process, application server, model provider, or database is needed after the build.

Export a production ZIP, deploy from the project workspace, or run the same deterministic command in CI.

@button[Choose a deployment path](/deployment/){primary}
@end

@column{variant=story-visual}
@terminal{title="Publish the same verified output" language=bash frame=macos}
$ mpress build --strict
$ mpress export site.zip
$ mpress deploy production
@end
@end
@end
@end
