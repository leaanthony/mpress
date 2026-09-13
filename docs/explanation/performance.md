---
title: Why M-Press is fast
description: How one Go process keeps builds, rebuilds, checks, and generated pages small and responsive.
translationKey: performance-design
order: 31
---

M-Press treats short feedback time as a product feature. Authors should be able
to save Markdown, inspect the result, and continue writing without waiting for
a JavaScript toolchain to start or rebuild an application.

## One native process does the work

M-Press is one Go executable. A build does not start Node.js, resolve packages,
transpile application code, or contact a network service. The same executable
loads configuration, reads content, renders pages, creates search indexes,
optimises assets, and writes the static site.

This design reduces setup time and removes communication between separate build
processes.

## Parsing is cached and parallel

M-Press fingerprints each source page and stores its parsed form under
`.mpress/cache/parse/`. An unchanged page can reuse that result on the next
build. A changed page receives a new fingerprint and cannot use stale content.

Markdown parsing uses several workers when the machine has available processor
cores. Results return to their original order, so parallel work does not make
the generated site non-deterministic.

## Validation starts during generation

Strict checks do not need to reopen every generated page to discover its links.
The build collects routes, links, and assets while it renders and copies files.
The final validation stage resolves that in-memory index against the completed
output.

The development menu shows a timing chart for each check stage. The chart makes
slow content discovery, parsing, rendering, optimisation, or link validation
visible instead of hiding the total behind one duration.

## Production optimisation is built in

Production builds use M-Press's Go-native CSS and JavaScript minifier. They also
scan the generated HTML and runtime script to remove class and ID rules that are
clearly unused. Responsive rules are filtered recursively. Rules that custom
HTML or runtime components may need remain intact.

This optimisation needs no Node.js package, lockfile, or external command.
Pass `--no-purge-css` when a project must retain every selector.

## Generated pages stay static

The browser receives static HTML first. Navigation, content, components, and
code examples do not wait for a client framework to render. A small script adds
progressive features such as search, tabs, theme selection, and accessibility
preferences.

Static output also lets the host cache the same files for every visitor. There
is no server-side rendering process to warm, scale, or monitor.

## Measure your project

Run a production build with machine-readable output:

```sh
mpress build --strict --json
```

The result reports the page count, file count, total build duration, and
diagnostics. Run **Run checks** in the development menu to inspect the timing of
each stage. Use **Lighthouse audit** to measure the current page on mobile or
desktop.

Build time depends on page count, content size, local storage, processor count,
and the number of static assets. M-Press reports the measurements instead of
promising one duration for every project.
