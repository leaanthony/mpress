# Wails Starlight feature corpus

This corpus was selected from Wails `master` at commit `a0fae83ad`. The scan
covered 501 Markdown and MDX files under `docs/src/content/docs`.

The most common Starlight components were:

| Component | Uses | Files |
| --- | ---: | ---: |
| TabItem | 642 | 128 |
| Tabs | 215 | 128 |
| Card | 252 | 68 |
| Aside | 98 | 37 |
| Steps | 74 | 56 |
| CardGrid | 64 | 60 |
| Image | 31 | 19 |
| FileTree | 24 | 24 |
| Badge | 10 | 6 |
| LinkCard | 8 | 4 |
| ShowcaseImage | 2 | 2 |
| MorphText | 2 | 2 |

Seven English pages provide the smallest practical set that covers every
component family and the important nesting combinations:

1. `index.mdx`: splash frontmatter, MorphText, CardGrid, and Card.
2. `quick-start/installation.mdx`: Steps, nested Tabs, Card, and terminal code.
3. `getting-started/your-first-app.mdx`: Steps, FileTree, and deep list nesting.
4. `tutorials/04-self-update-a-wails-app.mdx`: asset imports, Steps, Aside,
   Image, Tabs, and rich code fences.
5. `community/showcase/index.mdx`: ShowcaseImage and generated showcase markup.
6. `experimental/index.mdx`: Aside, CardGrid, and multiline LinkCard.
7. `guides/customising-windows.mdx`: Badge and annotated code fences.

`TestWailsHighCoverageStarlightCorpusConvertsToValidMPD` contains reduced,
source-faithful excerpts from these pages. Every excerpt must convert to MPD
without a parser diagnostic. A separate full-site check imports the complete
Wails checkout and runs `mpress build --strict`.
