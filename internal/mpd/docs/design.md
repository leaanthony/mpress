---
title: Why MPress Document is different
description: The reasoning behind a deterministic native format that continues to treat Markdown as a first-class source and interchange format.
order: 32
---

Markdown is the correct default for MPress. It is familiar, portable, and easy
to adopt. Its flexibility also makes exact source editing more expensive than
rendering alone suggests.

MPress currently parses Markdown to render a page and parses source structure
again to find safe translation and quick-edit ranges. Reusing a transformed
render tree changes source semantics in edge cases. Removing the second parse
without a replacement would weaken byte-for-byte editing guarantees.

MPress Document explores a different trade-off. It is a small native language whose
syntax is chosen around deterministic parsing, stable source ranges, explicit
components, and complete Markdown interchange.

## What we take from Djot

Djot demonstrates that a Markdown-like format can be easier to parse when it
removes non-local and ambiguous rules. MPress Document adopts several principles:

- parsing must be linear and must not backtrack;
- references must look like references before their definitions are known;
- emphasis must not require CommonMark's complete flanking algorithm;
- raw HTML must be explicit;
- one construct should have one preferred syntax; and
- arbitrary containers should not require an embedded programming language.

MPress Document does not copy Djot's grammar. MPress has different product needs.
Its format must understand MPress components, preserve imported Markdown bytes,
resolve project imports safely, expose exact edit ranges, and support a Go-only
single-binary implementation.

## The native parser stays small

The parser can recognise a block from its current line. It does not need to ask
whether a future reference exists, whether an HTML tag name is valid, or how an
arbitrary indentation width affects an earlier list item.

Components put line-scoped attributes after their names and close with `@end`.
They do not wrap attributes in braces. Leaf directives end with a separated
`/`. List indentation is exact. Raw HTML blocks do not run the inline parser.
Imports create nodes but do not perform I/O during parsing.

These rules support a streaming event parser. A tree builder, renderer,
translation extractor, or syntax highlighter can consume the same events.

## Markdown remains a complete boundary format

The native format must not become a migration trap. MPress therefore treats the
Markdown adapter as part of the specification rather than an optional converter.

An imported Markdown page keeps its original bytes and source trivia. If no
content changes, preserve export returns those bytes exactly. If one node
changes, the exporter rewrites that node and keeps untouched source ranges.

Canonical export creates deterministic Markdown when lexical preservation is
not required. Components use MPress's plain `@component ... @end` extension.
The exporter never produces MDX.

Opaque Markdown nodes are intentional. They let an unknown extension survive
an import and export cycle instead of disappearing or being guessed at. A
diagnostic makes the limitation visible.

## Imports are document adapters

An import does not paste text before parsing. It asks a registered adapter to
produce the common document model. Markdown, MPress Document, OpenAPI, HTML, and
future formats can therefore enter through separate, testable boundaries.

The parser records an import node without opening a file. The build resolves it
later, checks project-root safety, detects cycles, and caches the result by
content digest and adapter version.

This separation prevents network and filesystem behaviour from changing parser
results. It also lets a project rebuild only the documents affected by an
imported source.

## Translation and editing become parser outputs

Every native text node has a source range when it is first parsed. Translation
and quick editing do not need to rediscover prose with a second Markdown parse.

Code, link destinations, attributes, component names, and raw HTML already
have distinct node kinds. The translation layer can select prose nodes and
protect everything else without reconstructing the source grammar.

Browser editing can identify a node by document, source range, kind, and source
digest. Saving still verifies the digest before applying a change.

## Why this is not a template language

MPress Document has no variables, loops, expressions, module code, JSX, or runtime
component execution. Those features would make evaluation, security, caching,
and export substantially harder.

Imports compose documents. Components describe semantic content. Configuration
controls the site. Go code implements rendering. Each concern has one boundary.

## Adoption should be earned

The format should remain experimental until a prototype proves four things:

1. Five representative Wails pages remain pleasant to author.
2. Their generated HTML matches the Markdown versions.
3. Markdown import and export pass the public conformance suite.
4. Native parsing produces a material end-to-end build improvement.

If MPress Document is fast but unpleasant to write, it should remain an internal
document model. If it is pleasant but not materially faster, it can remain an
optional authoring format. It should become a headline feature only if it
improves authoring, editing, translation, and build performance together.

See the [MPress Document specification](/mpress-document/) for the normative grammar and
interchange requirements.
