---
title: Code and terminal
description: Commands, source explanations, diffs, and API endpoints.
order: 7
---

## Terminal

The terminal distinguishes commands, comments, and output. Set the `prompt`
and `comment` characters in metadata. The copy button copies executable
commands only. It does not copy prompt characters, complete comment lines, or
command output. Set `title` and `frame` (`macos`, `windows`, `linux`, or
`plain`) as metadata.

@terminal{title="Build the documentation" frame="macos" prompt="$" comment="#"}
# Check the site before the production build.
$ mpress check
No issues found.
$ mpress build --strict
Built 15 pages in 24ms.
@end

@details{title="Source"}
```md
@terminal{title="Build the documentation" frame="macos" prompt="$" comment="#"}
# Check the site before the production build.
$ mpress check
No issues found.
$ mpress build --strict
Built 15 pages in 24ms.
@end
```
@end

## D2 diagrams

Use a `d2` fence to render a diagram as an SVG image during the build:

```text
Frontend -> Backend: Call service
Backend -> Database: Query
```

```d2
Frontend -> Backend: Call service
Backend -> Database: Query
```

The SVG includes its fonts and works without JavaScript, a diagram service,
or a separate D2 installation. M-Press uses the Dagre layout engine. Keep each
diagram self-contained: file imports are disabled. Invalid D2 fails the build
and reports the compiler error instead of displaying the source as code.
Add `title="Description of the diagram"` after `d2` in the opening fence to
provide alternative text. Without a title, the diagram's node labels are used.
Wide diagrams scroll within their frame so their text remains readable.

## Diffs

@diff{title="mpress.yaml" mode="inline"}
colorScheme: light
search: false
---
colorScheme: system
search: true
@end

@details{title="Source"}
```md
@diff{title="mpress.yaml" mode="inline"}
colorScheme: light
search: false
---
colorScheme: system
search: true
@end
```
@end

Use the default `mode="side-by-side"` for a two-column comparison.

## Explained code

@explained
```go
func main() { // (1)
    site.Build() // (2)
}
```

(1) The program starts with a regular Go entry point.

(2) One call turns the content tree into the static site.
@end

@details{title="Source"}
````md
@explained
```go
func main() { // (1)
    site.Build() // (2)
}
```

(1) The program starts with a regular Go entry point.

(2) One call turns the content tree into the static site.
@end
````
@end

Point to or focus a numbered code line to open its explanation. Select the line
on a touch screen.

## API endpoints

@api{method="POST" path="/v1/builds"}
Creates a documentation build.

| Field | Type | Required |
| --- | --- | --- |
| `ref` | string | yes |
| `strict` | boolean | no |
@end

@details{title="Source"}
```md
@api{method="POST" path="/v1/builds"}
Creates a documentation build.

| Field | Type | Required |
| --- | --- | --- |
| `ref` | string | yes |
| `strict` | boolean | no |
@end
```
@end
