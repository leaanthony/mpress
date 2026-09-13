---
title: Components
description: A static, accessible component library for serious documentation.
order: 6
---

M-Press ships the complete component library in the main binary. Every
component renders at build time; JavaScript only adds progressive enhancement.

## One component grammar

Every block component starts with `@name{attributes}` and ends with `@end`.
Nested components use the same grammar at every level. Leaf components such as
buttons and images may use their compact single-line forms.

M-Press only treats a registered block component name at the start of a line
as a directive. Compact buttons and images must match their complete
signatures, so email addresses and ordinary mentions are unchanged.

## Notes and details

@note{type="tip" title="Plain Markdown inside"}
Use **Markdown**, links, lists, and code in a note. Types are `info`, `tip`,
`warning`, `caution`, `danger`, and `important`.
@end

@details{title="Source"}
```md
@note{type="tip" title="Plain Markdown inside"}
Use **Markdown**, links, lists, and code in a note. Types are `info`, `tip`,
`warning`, `caution`, `danger`, and `important`.
@end
```
@end

## Tabs

@tabs
[Go]
Run `go test ./...`.

[Shell]
Run `mpress build --strict`.
@end

@details{title="Source"}
```md
@tabs
[Go]
Run `go test ./...`.

[Shell]
Run `mpress build --strict`.
@end
```
@end

Tabs use buttons, tab panels, and ARIA state. The first panel remains readable
when scripts are unavailable.

## Cards and links

@cards{cols="2"}
[Start a project](/getting-started/)
Install one binary and generate a site.

---

[Configure M-Press](/configuration/)
Set navigation, languages, versions, and theme options.
@end

@linkcard{title="Read the authoring guide" href="/authoring/" description="Markdown, HTML, assets, and front matter." icon="→"}

@details{title="Source"}
```md
@cards{cols="2"}
[Start a project](/getting-started/)
Install one binary and generate a site.

---

[Configure M-Press](/configuration/)
Set navigation, languages, versions, and theme options.
@end

@linkcard{title="Read the authoring guide" href="/authoring/" description="Markdown, HTML, assets, and front matter." icon="→"}
@end
```
@end

## Steps

@steps
### Write
Add Markdown or HTML to `docs/`.

### Preview
Run `mpress dev` and edit with live reload.

### Ship
Run `mpress build --strict` and deploy `site/`.
@end

@details{title="Source"}
```md
@steps
### Write
Add Markdown or HTML to `docs/`.

### Preview
Run `mpress dev` and edit with live reload.

### Ship
Run `mpress build --strict` and deploy `site/`.
@end
```
@end

## File trees, badges, and buttons

@filetree
docs/
  index.md  Home page
  components.md  This component catalogue
mpress.yaml  Site configuration
@end

Use {badge.success:stable} for a compact status, or
@button[Open the guide](/getting-started/){secondary} for a clear
action inside prose.

@details{title="Source"}
```md
@filetree
docs/
  index.md  Home page
  components.md  This component catalogue
mpress.yaml  Site configuration
@end

Use {badge.success:stable} for a compact status, or
@button[Open the guide](/getting-started/){secondary} for a clear
action inside prose.
```
@end

## Layout containers

Containers add small, allow-listed layout rules without turning the Markdown
file into a template language.

@container{display=grid|columns=2|gap=1rem}
@tip[Static]
The complete layout exists in the generated HTML.
@end
@info[Responsive]
The default theme collapses dense layouts on small screens.
@end
@end

@details{title="Source"}
```md
@container{display=grid|columns=2|gap=1rem}
@tip[Static]
The complete layout exists in the generated HTML.
@end
@info[Responsive]
The default theme collapses dense layouts on small screens.
@end
@end
```
@end

## Landing page layouts

Landing pages can use nested Markdown layout directives. Use `section`,
`columns`, and `column` to define the page structure. Use `actions` for a group
of buttons. Use `headline` when a large heading needs deliberate line breaks.

```md
@section{variant=hero}
@columns{variant=hero}
@column{variant=hero-copy}
@headline
Modern docs.
Rich components.
Just Markdown.
@end

Write the supporting copy as ordinary Markdown.

@actions
@button[Start the tutorial](/tutorials/first-site/){primary}
@button[Read the guide](/authoring/){secondary}
@end
@end

@column
Add a product example, image, terminal, or another component here.
@end
@end
@end
```

The default theme supplies responsive styles for the named landing variants.
All layout content is rendered at build time. The source does not require HTML,
MDX, or a JavaScript framework.

## Compatibility

Starlight-compatible `Aside`, `Tabs`, `TabItem`, `CardGrid`, `Card`, `LinkCard`,
`Steps`, `FileTree`, `Badge`, and `Image` syntax is understood for migrations.
New content should use the `@` directives because they are portable and easy
to read.
