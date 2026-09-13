---
title: Configuration
description: Configure site metadata, build paths, search, theme, accessibility, contribution, languages, links, and versions.
order: 4
---

Every project has one `mpress.yaml`. Unknown fields are rejected so misspellings
cannot silently change a release.

```yaml
site:
  title: Acme Documentation
  description: Learn how to build with Acme.
  baseURL: https://docs.example.com
  defaultLanguage: en
  languages: [en, fr]
  languageLabels:
    en: English
    fr: Français
  defaultLanguageAtRoot: true
  missingTranslation: link-to-default
  logoLight: logo-light.svg
  logoDark: logo-dark.svg
  logoWidth: 160px
  favicon: favicon.svg
  socialImage: social-card.png
  headerLinks:
    - label: Guides
      url: /guides/
    - label: API
      url: /reference/
    - label: Support
      url: /support/
      type: button
      variant: outline

social:
  github: https://github.com/example/acme
  discord: https://discord.gg/example
  x: https://x.com/example
  editURL: https://github.com/example/acme/edit/main/docs

contribution:
  enabled: true
  repository: https://github.com/example/acme.git
  branch: main
  quickEdit: false
  guide: CONTRIBUTING.md

theme:
  accentColor: "#5375f6"
  hoverColorLight: "#315bd6"
  hoverColorDark: "#ffffff"
  colorScheme: system
  layout:
    preset: starlight

build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
  customCSS: custom.css

knowledge:
  enabled: true

blog:
  showTags: true
  headingSize: default
  imageMode: panel
  imageFit: cover
  imageWidth: 100
  imageBackground: ""

search:
  enabled: true
  shortcut: Mod+K
  placeholder: Search documentation
  maxResults: 12
  rememberRecent: true

accessibility:
  enabled: true
  shortcut: Mod+A

versioning:
  enabled: false
  current: next
  artifactsDir: .mpress/versions

translation:
  provider: openrouter
  model: openai/gpt-5-mini
  reasoningEffort: none
  languageModels:
    ja:
      model: qwen/qwen3.5-397b-a17b
      reasoningEffort: none
  baseURL: https://openrouter.ai/api/v1
  apiKeyEnv: OPENROUTER_API_KEY
  sourceLanguage: en
  glossary: glossary.yaml
  styleGuide: docs/writing-standard.md
  stateDir: .mpress/translations
  dataCollection: deny
  requireParameters: true
```

Navigation groups can provide translated labels in `_nav.yaml`. Page links use
the translated page title automatically. A missing group translation is marked
as a fallback in the generated HTML instead of being silently presented as
translated content.

```yaml
- label: Guides
  labels:
    fr: Guides
    de: Anleitungen
    ja: ガイド
  items:
    - label: Installation
      link: /quick-start/installation/
```

## Site

`title` is required. `description` becomes the fallback page description.
`baseURL` enables canonical URLs, `sitemap.xml`, the sitemap entry in
`robots.txt`, and absolute social sharing metadata. Leave it empty for
local-only builds.

Logos, the favicon, and paths referenced from Markdown should exist under the
configured static directory. Static files are copied into the output root.

Use `logoLight` for a logo that has sufficient contrast in light mode. Use
`logoDark` for the dark-mode logo. M-Press changes the logo when the visitor
changes the colour mode. M-Press preserves the logo's aspect ratio and fits it
to the navbar automatically. `logoWidth` controls the rendered width and accepts
a positive `px`, `rem`, or `ch` value. In dev mode, the Brand settings page can
upload and edit SVG, PNG, JPEG, WebP, or AVIF files in the project's
`static/brand` directory.

`socialImage` is optional. Use an image close to 1200 by 630 pixels for rich
Open Graph and Twitter previews. M-Press always writes titles, descriptions,
and page types. It adds the image metadata when both `baseURL` and
`socialImage` are set.

M-Press generates a useful `404.html` and a permissive `robots.txt`. Add either
file to the static directory when the project needs a custom version.

`headerLinks` adds short text links next to the site identity. Keep this list
small. Search, version, language, social, and theme controls remain in the
right-hand control group. A normal entry uses `type: link`. Use `type: button`
for one compact call to action. Its `variant` can be `primary`, `secondary`,
`outline`, or `custom`. A custom button also requires a six-digit `color`:

```yaml
- label: Sponsor
  url: /sponsor/
  type: button
  variant: custom
  color: "#7c3aed"
```

## Contribution

Set `contribution.enabled` to add a **Contribute** action to generated pages.
`repository` is the Git repository that contains `mpress.yaml`. `branch` is the
source branch to clone and defaults to `main`. `guide` is an optional project
path such as `CONTRIBUTING.md`. M-Press shows this guide before a contributor
edits a local checkout.

The generated page publishes repository and source-file metadata. It never
publishes credentials. Private repositories use the reader's existing Git
authentication. See [Contribute from a published page](/how-to/enable-site-contributions/) for the
reader workflow and checkout safety rules.

## MCP knowledge base

`knowledge.enabled` controls the portable, read-only knowledge bundle. It is
enabled by default. Every production build writes deterministic page, section,
and search artifacts under `site/knowledge/`. The `mpress knowledge` command
serves those verified artifacts to MCP clients. It does not expose editing,
deployment, or file-system tools.

Disable the bundle only when the generated files are not required:

```yaml
knowledge:
  enabled: false
```

## Blog

The Blog settings page controls the default presentation of the generated blog
archive. Posts can override these values in frontmatter. `showTags` controls
the tag row, `headingSize` may be `default`, `compact`, or `large`, and
`imageMode` may be `panel` or `floating`. `imageFit` may be `cover` or
`contain`. `imageWidth` controls floating images from 50 to 100 percent.
`imageBackground` accepts a six-digit hex colour and is ignored for floating
images.

## Theme

`colorScheme` may be `system`, `light`, or `dark`. Visitors can change it using
the theme control; their choice is stored locally in the browser.

`accentColor` changes the main interactive colour. `hoverColorLight` and
`hoverColorDark` set the colour used when the pointer is over navigation icons
in each theme. M-Press derives a contrast-safe foreground from each configured
colour. The older `hoverColor` field remains a fallback for existing
projects. For deeper customisation, set `build.customCSS` to a stylesheet
relative to the project root. It is copied after the default stylesheet so your
rules can override the theme.

The `starlight` layout preset is the default. It fixes the navigation to the
left and centres the article and table of contents as one unit. The `wide`
preset gives reference pages more room. The `reading` preset uses a shorter
line length for prose.

Use `custom` when the project needs exact measurements:

```yaml
theme:
  layout:
    preset: custom
    contentWidth: 720px
    wideContentWidth: 1024px
    sidebarWidth: 300px
    tocWidth: 256px
    contentTocGap: 40px
    alignment: cluster
    toc: right
```

Article widths accept `px`, `rem`, `ch`, or a percentage from `40%` to `100%`.
Other measurements accept `px`, `rem`, or `ch`. Use `alignment: left` to keep
the content unit near the navigation. Use `toc: hidden` to remove the desktop
table of contents. The layout always collapses for smaller screens.

Add `layout: wide` to page frontmatter when a table or reference page needs the
configured wide-page width.

## Accessibility

The accessibility menu is enabled by default. It lets each visitor change text
size and spacing, use a readable font or bionic reading, reduce distractions
and motion, add a reading guide, increase contrast, underline links, and select
a colour profile.

Preferences stay in the visitor's browser. M-Press does not send them to a
server. Set `accessibility.enabled: false` to remove the control, panel,
stylesheet rules, and script from the generated site.

## Keyboard shortcuts

`search.shortcut` opens search. `accessibility.shortcut` opens the accessibility
menu. `Mod` means Command on Apple devices and Ctrl on other platforms. Each
shortcut must contain at least one modifier and one key. Set a shortcut to
`None` to disable it. M-Press does not intercept shortcuts while the reader is
editing an input, textarea, select box, or editable region.

## Build paths and safety

Relative build paths are resolved from the project containing `mpress.yaml`.
The output directory must be a child of that project. M-Press refuses to clean
or replace the project root, its parent, or an external directory.

## Search

Static search produces one small JSON index per language. Search runs entirely
in the browser and sends no queries to a service.

## Translation

Translation settings apply only when you run `mpress translate` or use the
Translations development tool. Static builds do not contact a provider.

`apiKeyEnv` names an environment variable. Do not put the credential itself in
the configuration. `stateDir` stores committed freshness and review metadata.
`glossary` and `styleGuide` are optional paths inside the project. See
[Translate documentation](/translation/) for the full workflow.
