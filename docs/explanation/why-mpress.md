---
title: Why M-Press exists
description: The design reasons for a small static documentation generator that uses Markdown, HTML, and one Go executable.
order: 30
---

M-Press is for teams that want documentation without a JavaScript application
toolchain. It accepts Markdown and standard HTML. It produces files that any
static web server can publish.

## Portable content is the primary constraint

Documentation usually lives longer than its first generator. Framework-specific
components make migration expensive because content becomes application source.
M-Press keeps its authoring model small so teams can read and transform the source
with common tools.

HTML is the escape hatch. Authors can use semantic HTML when Markdown is not
sufficient. A landing page can use a custom layout without changing the format of
the technical documentation.

## Strict builds make migration safer

A migration tool must not hide content that it cannot convert. The M-Press
Starlight importer converts supported syntax and writes a migration report.
Unsupported components remain visible and cause a strict build error.

This behaviour can make the first migration build fail. The failure is useful:
it identifies work that a silent conversion would lose.

## Local tools and shared services have different jobs

The M-Press binary owns deterministic local work. This includes parsing,
navigation, private search, accessibility controls, language routes, structural
translation, version snapshots, checks, production optimisation, and static
output.

Optional external services can coordinate people and hosted data. M-Press
builds and publishes complete static sites without a subscription.

See the [M-Press feature showcase](/features/) for the complete workflow.
