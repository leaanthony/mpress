---
title: Import from Starlight
description: Migrate Astro Starlight documentation to M-Press without hiding unsupported content.
order: 11
---

The importer converts the parts of a Starlight project that can remain portable:
pages, frontmatter, navigation, languages, public assets, source images, social
links, branding, and the supported documentation components.

```sh
mpress import --from starlight ../starlight-docs --output ./docs
cd docs
mpress build
```

The source may be a Starlight project root or a directory containing Markdown.
The importer looks for `src/content/docs`, `docs`, and then `src`.

## Migration report

Every import writes `migration-report.md`. Review it before enabling strict
builds. Findings may include:

- custom or unsupported MDX components;
- project-specific JSX imports;
- private underscore-prefixed content;
- unresolved sidebar links;
- styles or components that need a static replacement.

Known components are converted to M-Press directives or static HTML. Unknown
components remain visible in generated pages and make a strict build fail.

## Recommended migration loop

1. Import into a new directory; do not overwrite the original project.
2. Read `migration-report.md` from top to bottom.
3. Run `mpress build --json` and resolve error diagnostics.
4. Run `mpress check` and classify broken links, fragments, and local assets.
5. Compare representative pages at desktop and mobile widths.
6. Enable `mpress build --strict` as the release gate.

@note{type="tip" title="The importer is an adapter"}
Do not preserve framework syntax merely for perfect source compatibility. Convert
important custom presentation into Markdown, ordinary HTML, or a small M-Press
component so the migrated documentation becomes simpler over time.
@end
