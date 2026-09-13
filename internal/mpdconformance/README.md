# MPD conformance corpus

This package contains the executable conformance corpus for MPress Document.
Each fixture has three files under `testdata/corpus/`:

- `.mpd` is the native MPress Document source.
- `.md` is the equivalent Markdown interchange source.
- `.html` is the exact output from the production MPress renderer.

Native MPD puts line-scoped attributes after the directive name, for example
`@note type="tip"`. Leaf directives use a separated trailing marker, for
example `@image src="hero.png" /`. The Markdown interchange files retain the
deployed MPress Markdown component syntax, including its braced attributes.
This lexical difference is intentional. Both sources represent the same
document model and HTML output.

Raw HTML uses the dedicated `@rawHTML` block. It has no attributes. The removed
generic `@raw format="html"` spelling is not canonical MPD.

`testdata/manifest.json` records the category, level, features, and component
coverage for every fixture. Top-level fixtures cover every registered MPress
component. Nested fixtures cover the component container families.

Run this command from the repository root after an intentional renderer or
fixture change:

```sh
go run ./cmd/mpd-fixtures -root .
```

The command refreshes every HTML golden and rebuilds
`testdata/viewer/index.html`. The viewer is a complete static MPress site. It
uses the production CSS and JavaScript and includes system, light, and dark
preview controls.
