---
title: Interactive components
description: Progressive enhancement without a JavaScript framework.
order: 9
---

Interactive components are static HTML first. The default theme adds a small,
dependency-free script for state, network requests, and presentation.

## Tutorials

@tutorial{title="Publish a site"}
### Check the content
Run `mpress check`.

### Build strictly
Run `mpress build --strict`.

### Deploy the output
Upload the generated `site/` directory.
@end

@details{title="Source"}
```md
@tutorial{title="Publish a site"}
### Check the content
Run `mpress check`.

### Build strictly
Run `mpress build --strict`.

### Deploy the output
Upload the generated `site/` directory.
@end
```
@end

Progress is stored locally and the reset button clears it.

## Audience filtering

The selector is added automatically when a page contains audience blocks.

@audience{role="developer"}
Developers can extend the Markdown processing pipeline in Go.
@end

@audience{role="writer"}
Writers only need Markdown, HTML, and the preview command.
@end

@details{title="Source"}
```md
@audience{role="developer"}
Developers can extend the Markdown processing pipeline in Go.
@end

@audience{role="writer"}
Writers only need Markdown, HTML, and the preview command.
@end
```
@end

## Conditional content

Conditional blocks read a URL parameter, then remember it locally. Visit this
page with `?framework=go` to select the second block.

@if{param="framework" value="plain" default="true"}
This is the default, framework-neutral guidance.
@end

@if{param="framework" value="go"}
This guidance is selected for Go users.
@end

@details{title="Source"}
```md
@if{param="framework" value="plain" default="true"}
This is the default, framework-neutral guidance.
@end

@if{param="framework" value="go"}
This guidance is selected for Go users.
@end
```
@end

## Reactive values

@input{name="projects" type="range" min="1" max="20" value="4" label="Projects"}

@input{name="seats" type="number" value="3" label="Editors"}

@computed{expr="projects * seats" deps="projects,seats" label="Project seats" format="%d"}

@details{title="Source"}
```md
@input{name="projects" type="range" min="1" max="20" value="4" label="Projects"}
@end

@input{name="seats" type="number" value="3" label="Editors"}
@end

@computed{expr="projects * seats" deps="projects,seats" label="Project seats" format="%d"}
@end
```
@end

## API playground

The playground sends a real browser request only when the reader presses the
button. This example targets the local preview server and is safe to inspect.

@api-playground{method="GET" path="/search-index.json" baseUrl="http://127.0.0.1:4174"}
Inspect the generated search index.
@end

@details{title="Source"}
```md
@api-playground{method="GET" path="/search-index.json" baseUrl="http://127.0.0.1:4174"}
Inspect the generated search index.
@end
```
@end

## Variants

`@variant{name="react"}` is resolved during the build rather than in the
browser. Set the active variant in the build configuration to publish one set
of product-specific content from a shared source.
