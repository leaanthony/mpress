---
title: MPress Document specification
description: Draft specification for a fast, deterministic plain-text documentation format with complete Markdown interchange.
order: 13
---

MPress Document is a plain-text document format for technical documentation. Its
file extension is `.mpd`. This document specifies MPress Document version 0.1.

The specification is a draft. MPress uses `.mpd` as its deterministic internal
representation and accepts Markdown as the public authoring format.

The key words **must**, **must not**, **required**, **should**, **should not**,
and **may** describe normative requirements.

## Goals

MPress Document is designed to:

- parse in linear time without backtracking;
- produce exact source ranges for every document node;
- support streaming and incremental parsers;
- remain readable without a renderer;
- express MPress components without MDX or executable source;
- import every supported Markdown construct without losing content;
- export every document as Markdown without generating MDX;
- preserve the original bytes of unchanged imported Markdown;
- make translation and browser editing operate on stable prose ranges; and
- report malformed input at the line where it occurs.

MPress Document is not intended to replace Markdown as the default MPress authoring
format. It is a deterministic alternative and a canonical document model for
imports, transformations, translations, and editing.

## Markdown compatibility contract

Markdown interoperability is part of the format, not a separate convenience
tool. A conforming MPress implementation must provide all of these guarantees:

1. Every Markdown file accepted by the selected profile can be imported.
2. Import never removes unknown content and never executes MDX.
3. An unchanged Markdown file can be exported byte-for-byte in preserve mode.
4. A changed document can be exported as complete, deterministic Markdown.
5. Export never requires MDX, JSX, JavaScript, or a framework runtime.
6. Unsupported extensions survive as exact opaque Markdown with a diagnostic.

The contract covers lexical preservation and document meaning. It does not
claim that every third-party Markdown dialect assigns the same meaning to the
same punctuation. The importer records the selected profile so that conversion
is reproducible.

## Conformance

A conforming parser must accept the grammar in this specification and must
produce the specified node kinds. A conforming serializer must emit canonical
MPress Document. A conforming Markdown adapter must implement the interchange
requirements in [Markdown interchange](#markdown-interchange).

An implementation may impose documented limits on file size, line length,
attribute count, and nesting depth. It must report a diagnostic when a limit is
exceeded. It must not silently discard content.

## Encoding and lines

An MPress Document document must use UTF-8. A byte order mark is permitted only at
the start of a file and is not document content.

The parser must recognise LF, CRLF, and CR line endings. It must retain the
original line-ending bytes in source ranges. A canonical serializer writes LF.

A blank line contains no characters other than spaces or tabs. Blank lines
separate block constructs. More than one blank line has the same document
meaning as one blank line, but the parser retains every byte as source trivia.

## Minimal document

MPress Document has no required signature or preamble. This is a complete
schema 1 document:

```text
Hello world.
```

A document without metadata uses schema 1. The `.mpd` extension identifies the
format. The metadata `schema` field identifies a different document schema when
one is required.

## Metadata

Page metadata uses an optional delimited block. The opening `---` must be the
first line of the document and must contain no other characters. No blank line,
space, or comment may precede it. An optional UTF-8 byte order mark is not
document content. A closing `---` must occupy its own line.

The metadata body is MPress Document metadata, not YAML. Each field occupies
one line and uses `key = value` syntax.

```text
---
schema = 1
title = "Install Wails"
description = "Install the tools required to build a Wails application."
slug = "quick-start/installation"
order = 20
draft = false
tags = ["installation", "beginner"]
---
```

A metadata key is an ASCII identifier. It may contain letters, digits, `_`,
`-`, and `.`, and must begin with a letter or `_`. A value must be a JSON
string, number, Boolean, null, array, or object on the same logical line.

When a metadata block is present, `schema` is required, must be its first field,
and must be a positive integer. A parser must reject an unsupported schema
before parsing document content. The following bare keys are permanently
reserved by schema 1:

`schema`, `title`, `translationKey`, `description`, `draft`, `layout`, `slug`,
`order`, `date`, `template`, `banner`, `hero`, `tags`, `author`, `authors`,
`image`, `imageMode`, `imageFit`, `imageBackground`, `imageWidth`, `showTags`,
and `headingSize`.

These names are case-sensitive. The schema 1 reserved set is frozen. A later
schema must not assign MPress meaning to another unprefixed key because an
existing document may already use that key as custom metadata.

All keys beginning with `mpress.` are reserved for present and future MPress
features. Custom metadata may use any other unreserved key. Namespaced custom
keys such as `wails.section`, `seo.image`, and `analytics.campaign` are
recommended. Unknown custom keys are retained exactly and made available to
the application. They must never be silently reinterpreted by a later schema.

Duplicate keys are an error. A first-line `---` always opens metadata, so a
missing closing delimiter is an error. A `---` line anywhere else is plain
text.

## Blocks

A block begins at the start of a logical line. Except inside a list or quote,
leading indentation does not change the block type. A block construct must be
separated from surrounding paragraphs by a blank line.

A parser must decide a block type from the current line. It must not reinterpret
earlier lines after reading later content.

### Paragraphs

One or more ordinary lines form a paragraph. A blank line ends the paragraph.
The `@end` directive for the current container can also end it. Another block
marker without a preceding blank line remains paragraph text.

```text
MPress builds complete documentation from plain text.
This line starts on a new rendered line.

This sentence is \
continued without a rendered line break.

This is a new paragraph.
```

A single line ending inside a paragraph creates a hard line break. A backslash
immediately before the line ending consumes that line ending. The text on the
next physical line continues in the same inline flow. Any space before the
backslash remains content, so prose can wrap without joining adjacent words.

Text cannot accidentally become another block after a paragraph has started.
For example, `# text` on the second line of a paragraph remains text.

### Headings

A heading starts with one to six `#` characters followed by one ASCII space.
It must occupy one line.

```text
# Install MPress

## Verify the installation
```

Closing `#` characters have no special meaning. Setext headings are not part of
MPress Document.

### Rules

The reserved directive `@hr` creates a thematic break.

```text
Before the break.

@hr

After the break.
```

### Lists

An unordered item begins with `- `. An ordered item begins with one or more
ASCII digits followed by `. `. List indentation must be a multiple of two
spaces. Tabs are not permitted for structural indentation.

```text
- Install Go.
- Install MPress.
  - Build the documentation.
  - Run the checks.

1. Create the project.
2. Preview the site.
```

The first item determines whether a list is ordered. Every item at that depth
must use the same marker type. Ordered item numbers are retained as source
metadata. A canonical serializer starts at the first number and increments
subsequent numbers.

Continuation lines must align two spaces beyond their item marker. There are no
lazy continuation lines. A nested block or list must also begin two spaces
beyond its parent item.

```text
- First paragraph in the item.

  Second paragraph in the same item.

  @note type="tip"
  A component in the item.
  @end
```

A task item places `[ ]`, `[x]`, or `[X]` immediately after its list marker.

```text
- [ ] Write the guide.
- [x] Run the checks.
```

### Quotes

A quote line begins with `> ` or consists only of `>`. Every line in the quote
must carry the prefix. Repeating the prefix creates a nested quote.

```text
> Documentation is part of the product.
>
> > Clear examples make adoption easier.
```

The parser removes one prefix at a time and applies the block grammar to the
remaining text.

### Code blocks

A code block uses a fence of three or more backticks. The opening fence may be
followed by a language identifier and an attribute list. The closing fence must
contain at least as many backticks as the opening fence and no other
non-whitespace characters.

`````text
````go {title="Build the site" lineNumbers=true}
site, err := mpress.Build(config)
@end
```
````
`````

The body is opaque. Every byte between the opening and closing fences is code.
Directives, inline roles, links, and formatting are not recognised there. An
`@`, an exact `@end` line, and a shorter backtick fence are all literal content.
An author must use a longer opening and closing fence when the body contains a
line that would otherwise close the block. No `@` escaping is permitted or
required inside a code block.

The attributes `language`, `title`, `filename`, `highlight`, and `lineNumbers`
have the same meanings as their MPress Markdown component equivalents.

### Raw HTML

Raw HTML is explicit.

```text
@rawHTML
<details><summary>More</summary>Content</details>
@end
```

The body is not parsed. It is HTML and the directive has no attributes.

Raw HTML is permitted because raw HTML is part of CommonMark. JSX, imports,
exports, expressions, and other MDX constructs are not executable and must not
be generated by a Markdown exporter.

### Comments

Comments use an opaque block and are retained in the document tree.

```text
@comment
Check this claim before the next release.
@end
```

Comments are not rendered. A Markdown serializer writes them as HTML comments
with unsafe `--` sequences escaped.

## Inline content

Inline parsing is confined to one paragraph, heading, table cell, or component
label. It must not depend on a definition that appears later in the document.

An inline delimiter can be escaped with `\`. An unmatched delimiter is literal
text. Delimiters must be properly nested and must not cross.

### Emphasis and strong text

Single `_` delimiters create emphasis. Single `*` delimiters create strong
text.

```text
Use _emphasis_ sparingly and make *important text* clear.
```

A delimiter may open when it is followed by a non-whitespace byte. It may close
when it is preceded by a non-whitespace byte. If the matching delimiter is not
at the top of the delimiter stack, the byte is literal text.

This rule does not use Unicode categories, flanking rules, or doubled markers.

### Code spans

A run of one or more backticks opens a code span. The next run of the same
length closes it. A different run length is content. An unclosed span continues
to the end of its inline container and produces a warning.

```text
Run `mpress build` or use ``a ` character`` in an example.
```

Code span content is literal. A canonical serializer chooses a delimiter one
character longer than the longest backtick run in the content.

### Links and images

An inline link uses `[label](destination)`. An image adds `!` before the label.

```text
[Read the guide](/getting-started/)
![Application window](/images/application.png)
```

The label may contain inline content with properly nested brackets. The
destination must occur on the same logical line. It is either a sequence with
balanced parentheses and backslash escapes or a JSON-quoted string.

Reference links use `[label][identifier]`. The second pair is required, so link
recognition is local. Identifiers are case-sensitive.

```text
[Download Go][go-download]

@link id="go-download" destination="https://go.dev/dl/"
```

An undefined reference remains a link node and produces a diagnostic. Shortcut
reference syntax such as `[identifier]` is not part of MPress Document.

### Automatic links

An absolute `http`, `https`, or `mailto` URI between angle brackets creates an
automatic link.

```text
<https://m-press.me>
```

Plain text that resembles a URL remains text. Automatic linking of bare text is
a renderer option, not source syntax.

### Emoji shortcodes

A recognised emoji name between colons creates an `emoji` node.

```text
Build complete :white_check_mark: Ship it :rocket:
```

An emoji name contains one or more ASCII letters, digits, `_`, `-`, or `+`.
Names are case-sensitive and canonical names are lowercase. Schema 1 uses a
fixed snapshot of the GitHub-compatible emoji shortcode registry published with
the conformance suite. A parser must use that local registry. It must not query
a network service or allow a registry update to change the meaning of an
existing schema 1 document.

Only a name in the schema registry creates an `emoji` node. An unknown name,
including its colons, remains literal text and does not produce a diagnostic.
A backslash before the first colon prevents recognition, so `\:rocket:` renders
as the literal text `:rocket:`.

Emoji shortcodes are recognised in headings, paragraphs, link labels, table
cells, and component prose. They are not recognised in code spans, code fences,
raw HTML blocks, comments, metadata values, attribute values, or link
destinations. An HTML renderer writes the registry's native Unicode sequence.
It must not require an image service, script, or client-side replacement.

Canonical MPress Document writes the canonical shortcode name. A Markdown
importer using the `mpress-markdown-1` profile recognises the same registry.
Canonical MPress Markdown retains the shortcode. Portable CommonMark and GFM
export write the native Unicode sequence because emoji shortcodes are not part
of those specifications.

### Additional inline roles

Less common inline semantics use an explicit role. This avoids assigning common
punctuation more than one meaning.

```text
Water is H@sub[2]O.
Press @kbd[Command K].
This is @mark[important] and @delete[obsolete].
```

The grammar is `@name[content]`. `name` is an ASCII identifier. Brackets inside
the content must be balanced or escaped. The roles defined by version 0.1 are
`sub`, `sup`, `mark`, `insert`, `delete`, `kbd`, `var`, and `cite`.

An unknown role remains an `inline-role` node. It must not disappear.

### Metadata references

The reserved inline role `metadata` inserts a metadata value as text at the
reference position.

```text
This page is called @metadata[title].
The previous location was @metadata[wails.redirect].
```

The content between brackets is one complete, case-sensitive metadata key.
Dots are part of the key, so `wails.redirect` performs an exact lookup of that
namespaced key. It does not traverse an object. Reserved and custom metadata
use the same lookup rules.

A string inserts its contents without quotation marks. Numbers, Booleans, and
null use their canonical JSON spelling. Arrays and objects use compact canonical
JSON. Object keys retain metadata source order. The inserted value is text, not
markup, and must not be parsed again for roles, formatting, HTML, or components.
Typographic substitutions must not change it. An HTML renderer must escape it
as ordinary text.

A metadata reference is recognised in every inline-content context, including
headings, paragraphs, link labels, table cells, and component prose. It is not
recognised inside code fences, raw HTML blocks, comments, metadata values, attribute
values, or link destinations.

If the key does not exist, the renderer retains the literal
`@metadata[key]` source and produces an `mpd-metadata-reference` diagnostic. It
must not insert an empty string or silently choose another key.

### Escapes and entities

A backslash before ASCII punctuation makes the next byte literal. A backslash
before any other byte remains a backslash.

MPress Document does not interpret HTML entities. Write the UTF-8 character itself.
The Markdown adapter retains entity spelling as provenance and decodes it for
the document model.

## Components

A component is either a container or a leaf declared by the schema registry.
Its opening line begins with `@`. A container body uses ordinary MPress Document,
and `@end` closes it.

```text
@note type="tip" title="No MDX required"
Components contain *ordinary document content*.
@end
```

Components can nest. The parser uses a stack and the closest `@end` closes the
current container component.

The schema component registry declares each component as a container or a leaf.
A leaf component consists only of its opening line and does not use `@end`.

```text
@image light="/light.png" dark="/dark.png" alt="Architecture"
```

MPress controls the complete component registry. User-defined components are
not part of MPress Document. An unknown directive produces an `mpd-directive`
diagnostic and remains literal text. It does not open a container.

### Attributes

Attributes follow the component name on the opening line. At least one ASCII
space separates the name from its first attribute, and one or more ASCII spaces
separate adjacent attributes. Braces are not part of MPress Document component
syntax.

The opening line uses this grammar:

```text
opening    = "@" name [spaces attributes] line-end
attributes = attribute *(spaces attribute)
attribute  = name | name "=" value
value      = string | number | boolean | null | array | object
spaces     = 1*(ASCII space)
```

Names use the metadata key grammar. Values use single-line JSON syntax. Bare
string values are not permitted. A name without `=` is shorthand for
`name=true` and is valid only for a declared boolean attribute. Duplicate
attributes are an error.

Attribute parsing is confined to the opening line. JSON strings, arrays, and
objects can contain spaces. The parser determines their ends from JSON syntax,
not by splitting the line at every space.

The schema registry determines whether an opening line is complete or requires
a matching `@end`. The parser does not infer this from source punctuation. A
slash inside a quoted value, such as the slashes in a URL, is ordinary value
content. A canonical serializer writes one space between the component name and
each attribute.

These are canonical opening lines:

```text
@note type="tip" title="Read this first"
@table header search page-size=20
@image src="/images/overview.png" alt="Product overview"
@hr
```

### Literal directive lines

At the start of a logical line, a backslash escapes a directive's initial `@`.
This is an application of the normal punctuation escape rule, not a separate
escape mechanism.

This rule does not apply inside code blocks. Code block bodies are opaque and
retain every `@` without an escape.

```text
\@note is displayed as @note.
```

## Tables

A table is an explicit block. Each row begins and ends with `|`. A backslash
escapes a literal pipe.

```text
@table header=true
| Name | Purpose |
| Markdown | Portable authoring |
| MPress Document | Deterministic authoring |
@end
```

Every row must contain the same number of cells. Leading and trailing ASCII
spaces inside a cell are formatting trivia and are not cell content. Cells use
the inline grammar.

Column alignment uses an optional `align` array with `left`, `centre`, `right`,
or `default` values.

```text
@table header=true align=["left", "right"]
| Stage | Time |
| Parse | 83 ms |
@end
```

Tables can provide reader controls. Each control is optional and disabled by
default:

- `search` adds one case-insensitive search field for all cells.
- `filter` adds an exact-value filter menu to each column header.
- `sort` makes each header sortable in ascending or descending order.
- `paginate` divides matching rows into pages. `page-size` sets the number of
  rows per page and defaults to `10`.
- `column-separators` draws each column boundary through the header and body.

```text
@table header search filter sort paginate column-separators page-size=10
| Package | Platform | Downloads |
| Core | Linux | 1840 |
| Studio | macOS | 920 |
| Core | Windows | 1260 |
@end
```

Search, filters, sorting, and pagination combine. A search or filter change
returns to the first page. Sorting is stable, locale-aware, and compares
embedded numbers numerically. Generated controls must have accessible names,
sortable headers must expose `aria-sort`, and the visible row count must be an
`aria-live` status. Without script support, every row remains visible and the
underlying table remains readable.

## Footnotes

An inline footnote reference uses `[^identifier]`. Its definition is an
explicit block and can contain any block content.

```text
The result is reproducible.[^benchmark]

@footnote id="benchmark"
The benchmark uses twenty paired cold builds.
@end
```

Identifiers are case-sensitive. A duplicate definition is an error. An
undefined reference produces a diagnostic without deleting the reference.

## Imports and includes

An include reads another MPress Document file or fragment. An import invokes a
format adapter.

```text
@include src="shared/prerequisites.mpd"

@import src="README.md" format="markdown" profile="gfm"

@import src="api/openapi.yaml" format="openapi"
```

`src` is resolved relative to the containing file. Resolution must remain
inside the configured project root unless the project explicitly permits an
additional root. Symlinks must be resolved before this check.

Remote imports are disabled by default. A remote adapter must pin immutable
content by digest and must not retrieve content during an offline build.

The parser creates an import node without opening the target. Import resolution
is a separate build phase. This keeps parsing deterministic and permits caching.

An adapter returns the same document node model as the native parser. Every
imported node records its source format, source file, byte range when available,
and adapter version.

An include cycle is an error. Implementations must report the complete cycle.

### Import modes

The optional `mode` attribute has these values:

| Mode | Result |
| --- | --- |
| `fragment` | Insert the imported block nodes at the directive. This is the default. |
| `document` | Import metadata and body. Metadata conflicts are errors unless mapped explicitly. |
| `code` | Import bytes as a code block. |
| `data` | Make structured data available to a registered component without adding body nodes. |

## Markdown interchange

Markdown is a required adapter. MDX is not.

Version 0.1 defines these profiles:

- `commonmark-0.31.2`;
- `gfm` for tables, task lists, strikethrough, and automatic links; and
- `mpress-markdown-1` for the complete MPress Markdown component grammar,
  footnotes, typographic substitutions, and page frontmatter.

The default profile is `mpress-markdown-1`.

### Import guarantee

The adapter must retain every byte of a Markdown source file. It must never
discard an unknown extension or an unsupported construct.

Known constructs become typed document nodes. Raw HTML becomes a raw HTML node.
MDX-like syntax becomes an opaque Markdown node and a diagnostic. It is never
executed.

If a construct is valid Markdown but has no native MPress Document equivalent, the
adapter creates an `opaque-markdown` node containing its exact source bytes.
This is the lossless fallback, not an error-recovery shortcut.

Every imported node records:

- the original byte range;
- leading and trailing trivia;
- the original delimiter spelling;
- the source line endings;
- the Markdown profile; and
- whether the node or any child has changed.

### Export guarantee

A Markdown exporter has three modes.

| Mode | Guarantee |
| --- | --- |
| `preserve` | An unchanged imported Markdown document is byte-for-byte identical. Changed nodes are spliced into untouched original ranges. |
| `canonical` | Emit deterministic Markdown representing the complete document model. Formatting can change, but content and structure must not. |
| `portable` | Emit CommonMark or GFM without MPress components where a portable representation exists. Report components that cannot be lowered without loss. |

The exporter must generate Markdown, not MDX. It must not emit imports, exports,
JSX elements, JavaScript expressions, or framework component calls.

MPress components export with the MPress `@component ... @end` Markdown
extension. A generic Markdown reader therefore sees plain text and the component
body rather than executable syntax.

Canonical MPress Markdown retains `@metadata[key]`. Portable Markdown resolves
the reference and writes its plain-text value. Preserve mode retains the
original source bytes.

### Markdown mapping

| Markdown construct | MPress Document representation |
| --- | --- |
| Paragraph | Paragraph |
| ATX or Setext heading | Single-line `#` heading |
| Emphasis | `_text_` |
| Strong emphasis | `*text*` |
| Inline code | Code span |
| Fenced or indented code | Backtick code fence |
| Block quote | Prefixed quote lines |
| Ordered or unordered list | Strict list |
| Task list | Strict list with task state |
| Link or image | Link or image node |
| Reference link | Reference link and `@link` definition |
| Emoji shortcode | `:emoji_name:` |
| Thematic break | `@hr` |
| GFM table | `@table` |
| Footnote | `@footnote` |
| Raw HTML | `@rawHTML` |
| MPress component | Component or leaf component |
| Metadata reference | `@metadata[key]` |
| YAML frontmatter | Delimited MPD metadata with values converted without loss |
| Unsupported extension | Exact `opaque-markdown` node |

### Persisted provenance

When Markdown remains the source file, provenance refers directly to that file.
When `mpress convert` writes a standalone `.mpd` file, canonical Markdown export
requires no additional file.

Byte-identical `preserve` export after conversion requires a provenance sidecar
named `<document>.mpd-source`. The sidecar contains the original bytes, their
digest, the Markdown profile, and node ranges. It is a build artefact and must
not contain executable code.

An exporter must verify the digest before using a sidecar. If it is missing or
does not match, the exporter must use canonical mode or stop with an error. It
must not claim byte preservation.

## Document model

The normative model is an ordered tree of nodes. Every node contains:

- `kind`;
- `source` with file, start byte, end byte, start line, and start column;
- ordered attributes;
- ordered children where the kind permits them;
- source provenance when imported; and
- diagnostics attached to that node.

Text content refers to immutable source slices where possible. A parser is not
required to allocate one string for every text node.

The core block kinds are `document`, `metadata`, `paragraph`, `heading`, `rule`,
`list`, `item`, `quote`, `code`, `raw-html`, `comment`, `table`, `table-row`,
`table-cell`, `footnote`, `component`, `import`, `include`, and
`opaque-markdown`.

The core inline kinds are `text`, `soft-break`, `hard-break`, `emphasis`,
`strong`, `code-span`, `link`, `image`, `automatic-link`, `emoji`,
`footnote-reference`, `metadata-reference`, and `inline-role`.

## Parser requirements

A conforming native parser must:

1. scan input from start to finish without backtracking;
2. use bounded lookahead within the current line;
3. recognise blocks before parsing their inline content;
4. protect opaque code fences before resolving components;
5. resolve components with a stack;
6. resolve inline delimiters with a stack;
7. parse references without consulting later definitions;
8. keep import resolution outside the parser;
9. recover at a blank line or the current container's `@end`; and
10. emit source ranges even when a node contains a diagnostic.

For `n` input bytes and nesting depth `d`, native parsing must be `O(n)` time and
`O(d + r)` auxiliary memory, where `r` is the number of references retained by
the document. The document tree itself is not auxiliary memory.

A conforming parser must not require regular expressions, HTML tokenisation,
Unicode category lookup, entity lookup, network access, or component-specific
parsing to discover the document structure.

## Diagnostics and recovery

A diagnostic contains a stable code, severity, message, source range, and an
optional repair suggestion.

Required diagnostic codes include:

| Code | Meaning |
| --- | --- |
| `mpd-version` | Unsupported document schema |
| `mpd-metadata` | Invalid, duplicate, or unterminated metadata |
| `mpd-metadata-reference` | Undefined metadata reference |
| `mpd-utf8` | Invalid UTF-8 |
| `mpd-unclosed` | Unclosed component, code fence, or raw HTML block |
| `mpd-end` | Unexpected `@end` |
| `mpd-attribute` | Invalid or duplicate attribute |
| `mpd-directive` | Unknown or unavailable directive |
| `mpd-indent` | Invalid structural indentation |
| `mpd-reference` | Undefined or duplicate reference |
| `mpd-import-cycle` | Include or import cycle |
| `mpd-import-root` | Import escapes an allowed root |
| `mpd-opaque` | Markdown construct retained as opaque content |

An unexpected `@end` becomes literal text after producing a diagnostic. An
unclosed ordinary component closes at the end of its containing block. An
unclosed raw HTML block consumes the remaining file and produces a diagnostic. No
recovery rule may discard source bytes.

## Canonical serialization

Canonical MPress Document:

- uses LF line endings;
- omits metadata when the document has no metadata and uses schema 1;
- writes `---` on the first line when metadata is present;
- writes `schema` as the first metadata field;
- writes metadata in source order;
- uses one blank line between block siblings;
- uses two spaces for each list depth;
- uses JSON quoting for attribute values;
- writes attributes in source order;
- uses a code fence one backtick longer than the longest potential closing
  fence in the body, with a minimum length of three;
- uses lowercase Boolean and null values;
- writes explicit `@end` lines for container components;
- omits `@end` for leaf components; and
- ends with one line ending.

Canonical output must parse to a document model equivalent to the input model.

## Conformance tests

The public conformance suite must include:

- one example for every normative syntax rule;
- malformed and adversarial inputs for every recovery rule;
- deeply nested components up to the documented limit;
- mixed LF, CRLF, and CR Markdown imports;
- all CommonMark 0.31.2 examples;
- the applicable GitHub Flavoured Markdown examples;
- every MPress Markdown component fixture;
- Markdown import followed by preserve export with byte equality;
- Markdown import followed by canonical export and semantic re-import;
- MPress Document export to Markdown followed by semantic re-import;
- fuzz tests asserting linear progress and no panics; and
- corpus benchmarks using the complete Wails v3 documentation.

The initial Go implementation should meet these non-normative performance
targets on the Wails v3 corpus:

- at least twice Goldmark's parse throughput for equivalent native content;
- no more than half Goldmark's parser allocations;
- source-range extraction during the native parse, with no second parse; and
- unchanged generated HTML compared with equivalent MPress Markdown fixtures.

Performance claims must include the corpus revision, Go version, CPU count,
sample count, before and after distributions, and generated-output comparison.

## Versioning

The metadata `schema` number changes when existing valid source can acquire a
different document meaning or when the component registry changes. A document
without metadata is schema 1. Additive node kinds, roles, adapters, and
attributes that do not change directive classification do not require a new
schema when an older parser can retain them as unknown nodes.

The first-line metadata delimiter and the `schema` field grammar are stable
across schema versions. This allows a parser to select the schema before it
parses the document body. A future schema must use the `mpress.*` namespace for
new engine-owned metadata rather than claiming an existing custom key.

During version 0.x, this document can change after implementation experiments.
MPress must not describe `.mpd` as stable until the grammar, Markdown adapter,
and conformance suite have shipped together.
