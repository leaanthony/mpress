---
title: M-Press 0.1 MVP
description: The deliberately narrow release contract for the first M-Press version.
order: 21
---

## Outcome

M-Press 0.1 is successful when Wails can replace its Starlight build with one
Go binary while authors continue to write Markdown, ordinary HTML, and a small
documented set of portable content components.

It is a documentation generator, not a general web application framework.

## Verification status

The 0.1 release contract is implemented and exercised against both the public
test suite and the private conformance suite. The current Wails documentation corpus
contains 514 Markdown sources, imports 524 content pages and publishes 673 files, builds in
strict mode, and passes the generated link and asset checker. Desktop and 375px
browser checks cover the landing page, documentation shell, search, accessibility
controls, project tools, onboarding, and configuration in dark and light themes.

## Release contract

### Authoring and build

- One executable on Linux, macOS, and Windows.
- `mpress init`, `dev`, `build`, `clean`, `check`, and `deploy`.
- CommonMark plus tables, task lists, footnotes, heading IDs, and ordinary HTML.
- YAML frontmatter with stable routes, drafts, titles, descriptions, and ordering.
- Deterministic offline builds with no Node.js runtime or network requirement.
- Unknown MDX never disappears: normal builds render an obvious marker and
  strict builds fail with a file-level diagnostic.

### Documentation experience

- A polished responsive default theme, light/dark/system colour modes, navigation,
  table of contents, previous/next links, and accessible keyboard landmarks.
- Static client-side search with a per-language index.
- Notes, tabs, cards, link cards, steps, file trees, badges, and images.
- Custom CSS for projects that need their own visual identity.
- A development-only project menu, guided setup, structured configuration,
  repository and work-branch preparation, a draggable live-preview panel,
  release checks, direct Cloudflare previews, and automatic browser reloads.

### Translation

- Explicit language list and stable language-prefixed routes.
- The default language may live at `/`.
- Translation switcher shows availability and links missing translations back
  to the corresponding default-language page.
- Missing pages are not silently duplicated into translated output.
- Per-language navigation and search indexes.
- Structural machine translation through OpenRouter, OpenAI, a compatible
  provider, or an authenticated local Codex/Claude Code command without
  changing code, links, Markdown syntax, or components.
- An outcome-first translation wizard for adding, updating, and reviewing
  languages without exposing provider details as the primary workflow.
- Local freshness state, manual-edit and conflict detection, glossary checks,
  page review state, and translated navigation.

Shared translation memory, assignments, budgets, pull requests, vendor
workflows, and organisation audit history are service features.

### Versioning

- Capture a completed static build under a label.
- A checksum manifest makes captured versions tamper-evident.
- Verify, list, remove, and mount captured versions into later builds.
- Version artifacts remain deployable static files and never require the service.
- Generated output includes host-neutral `_headers` rules: version snapshots
  are immutable-cacheable, assets use stale-while-revalidate, and HTML remains
  revalidatable.

### Deployment

- Publish the checked static output to Cloudflare Pages or Netlify from the CLI
  or the development menu.
- Preview deploys are isolated draft deploys. Production requires an explicit
  flag and a credential kept outside the project.
- ZIP export remains available for any other static host.

Managed artifact storage, retention policies, release automation, preview URLs,
and a version dashboard are service features.

### Starlight migration

- Import pages, supported components, navigation, languages, frontmatter, public
  assets, and relevant presentation settings.
- Generate a migration report for every pattern requiring human review.
- Validate the importer against both curated private fixtures and the real Wails
  documentation corpus.

## Deliberately outside 0.1

- Arbitrary Astro/React/Vue/Svelte component execution.
- A plugin runtime, theme marketplace, or general template language.
- Ecommerce, authentication, server-side rendering, and application state.
- Built-in Git hosting, managed deployment history, analytics, or translation vendors.
- Compatibility with every Starlight plugin.

## Release gates

1. Public unit and end-to-end fixture tests pass.
2. Private conformance and migration corpus pass against the release binary.
3. A Wails import builds without parser failure; intentional migration findings
   are reviewed and documented.
4. Generated pages pass link/asset checks and browser smoke tests at desktop and
   mobile widths.
5. A captured version verifies and mounts into a clean build.

All five gates are now covered. Migration findings that require editorial
judgement remain explicit in `migration-report.md`; they are not silently
discarded by the importer.
