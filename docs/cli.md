---
title: Command-line reference
description: Commands for creating, developing, validating, importing, and versioning M-Press sites.
order: 3
---

Run `mpress help` to print the compact command summary.

## Project commands

| Command | Purpose |
| --- | --- |
| `mpress init [directory]` | Create a starter project. The default directory is the current directory. |
| `mpress dev [directory] [--port 3000]` | Build drafts, serve the output, watch source files, and live reload. |
| `mpress dev --repo URL --checkout DIRECTORY [--branch NAME]` | Clone a repository, create a safe work branch, and start its development UI. |
| `mpress contribute SITE_URL_OR_REPOSITORY [--branch BRANCH] [--checkout DIRECTORY] [--draft-file FILE] [--no-open]` | Discover a page or use a configured repository, apply an optional downloaded browser draft, prepare a local checkout, and open the contribution wizard. |
| `mpress build [--strict] [--drafts] [--json] [--no-purge-css]` | Generate the production site with minified CSS and JavaScript. |
| `mpress export [archive.zip] [--strict] [--drafts] [--force] [--json]` | Build and package the complete static site as a ZIP archive. |
| `mpress clean` | Remove the configured output directory. |
| `mpress check` | Check generated local links, fragments, and assets. |
| `mpress check site [--output DIRECTORY] [--cloudflare-pages] [--json]` | Validate an existing build for publication. |
| `mpress knowledge [directory] [--transport stdio\|http]` | Serve the generated documentation as a read-only MCP knowledge base. |
| `mpress knowledge evaluate [directory] --suite FILE [--json]` | Measure retrieval recall, ranking, and citation coverage against a repeatable question suite. |
| `mpress translate status [--lang CODE] [--file PAGE] [--json]` | Report missing, stale, manual, and conflicting translations without using an API. |
| `mpress translate --estimate [--lang CODE] [--file PAGE] [--scope missing\|stale\|all] [--json]` | Estimate provider requests, tokens, and configured USD cost without contacting the provider or writing files. |
| `mpress translate [--lang CODE] [--file PAGE] [--scope missing\|stale\|all] [--harness codex\|claudecode] [--model MODEL] [--force]` | Translate documentation and navigation with the configured provider or a selected local harness and model. |
| `mpress translate audit --lang CODE [--file PAGE] [--harness codex\|claudecode] [--model MODEL] [--json]` | Audit translated structure and prose locally, with an optional independent coding-agent review. |
| `mpress translate check [--lang CODE] [--exceptions FILE] [--exclude-audit PAGE] [--json]` | Require complete translations and a clean local structural and linguistic audit. |
| `mpress translate review --lang CODE --file PAGE [--status reviewed\|final]` | Record review state for a translated page. |
| `mpress translate --repo URL --checkout DIRECTORY [--branch NAME] ...` | Clone a missing local checkout, then run translation in it. |
| `mpress version` | Print the installed M-Press version. |

M-Press searches the current directory and its parents for `mpress.yaml`, so
commands may be run from a nested content directory.

Start an existing project without changing your shell directory:

```sh
mpress dev ../product-docs
```

To begin from a remote repository, use one command. Git authentication comes
from your existing credential helper or `gh` login. M-Press clones the source,
creates a non-default work branch, builds the site, and prints the local UI URL:

```sh
mpress dev \
  --repo git@github.com:example/product-docs.git \
  --checkout ../product-docs \
  --branch docs/add-french
```

Open the printed URL and use **Translations** in the development bar. The
editing tools recognise the prepared branch, so they do not ask you to prepare
the repository a second time.

Start from a published M-Press page when the site has enabled contributions:

```sh
mpress contribute https://docs.example.com/guide/install/
```

M-Press discovers the repository and matching source file from the page. It
uses `~/mpress-contributions/<owner>-<repository>` by default, creates a safe
contribution branch, selects an available port, and opens the local wizard. Use
`--checkout` to choose another directory or `--no-open` to keep the browser
closed.

Use `--goal page` or `--goal translate` to open the matching local workflow
directly. Use `--draft-file` only with a development-only browser draft.

## Build modes

@tabs
[Normal]
`mpress build` emits diagnostics but completes the build. Unsupported content is
made visible in the output instead of disappearing.

[Strict]
`mpress build --strict` writes the output and returns a failure when error-level
diagnostics exist. Use this in CI and before every release.

[Machine-readable]
`mpress build --json` prints page counts, file counts, build duration, and
diagnostics as JSON. This is useful for integrations and private release gates.
@end

Draft pages are excluded unless `--drafts` is supplied. The development server
includes them automatically.

Production builds minify generated CSS and JavaScript with M-Press's built-in
Go minifier. It removes comments and redundant whitespace and shortens safe,
non-repeated local JavaScript bindings. It does not need Node or an external
minification tool. Production builds also remove clearly unused class and ID
rules after scanning every generated page and the runtime script. Responsive
at-rules are filtered recursively, while animation, font, and unknown at-rules
are retained. Selectors without class or ID tokens are retained because custom
HTML and runtime components can activate them later. Pass `--no-purge-css` when
a build must retain every selector.

## Publication checks

```sh
mpress check site --json
mpress check site --output release/site --cloudflare-pages --json
mpress translate check --json
mpress translate check --lang fr --exceptions translation/audit-exceptions.json --json
```

Both commands are read-only and run locally without Python, a translation
provider, or network access. They return a non-zero exit status on validation
failures. `--json` writes one report to stdout; the error summary goes to stderr.
Paths supplied by these flags are relative to the project root containing
`mpress.yaml`, unless absolute.

`mpress check site` reads `build.outputDir`, or the directory selected by
`--output`. It checks:

- Local links and assets, same-host absolute links, and HTML fragments.
- Exact redirects in `_redirects`, including chains, cycles, and missing targets.
  The supported rule format is `SOURCE TARGET 301` or `SOURCE TARGET 302`;
  wildcard rules, rewrites, and other statuses are reported as unsupported.
  An absent `_redirects` file is valid. External redirect targets are not fetched.
- Page language and canonical URL against `site.baseURL`. Canonical checks are
  skipped when no base URL is configured; the generated `404.html` is exempt.
- A page in every configured target language for every default-language route,
  with no default-language fallback. This is a complete-translation publication
  gate even when the builder permits missing translations.
- Contribution source metadata pointing to an existing file inside
  `build.contentDir`, including checks against symlink escapes.
- D2 source blocks or Markdown fences accidentally displayed as prose.
- Sitemap, robots, and LLM text assets; per-language search indexes when search
  is enabled; knowledge manifest artifacts when knowledge is enabled, including
  compressed filenames recorded in the manifest.

`--cloudflare-pages` additionally enforces a maximum of 20,000 output files and
25 MiB per file. These hosting limits are opt-in. Output symlinks and other
non-regular files are rejected. The JSON report contains `files`, `html_pages`,
`languages`, `redirects`, and a sorted `errors` array.

`mpress translate check` checks all configured target languages, or just `--lang`.
It discovers Markdown and MPD source pages using the translation engine and
checks translated navigation when source navigation exists. Missing, empty,
and orphaned translations fail. Local audit warnings also fail this command,
unlike `translate audit`, which fails only for error-level findings.

`--exceptions` accepts a JSON array using this format:

```json
[
  {
    "language": "fr",
    "file": "guide.md",
    "segment": "title",
    "code": "untranslated",
    "reason": "The reviewed heading is the same term in both languages.",
    "sourceSHA256": "<SHA-256 of the complete source file>",
    "targetSHA256": "<SHA-256 of the complete translated file>"
  }
]
```

The language, content-relative filename, segment ID, and code must match the
finding. The reason must be non-empty and both hashes must still match. Only
`untranslated` and `requirement-language` findings can be accepted this way;
missing content and structural damage cannot be waived by an exception.

`--exclude-audit PAGE` is an explicit escape hatch for known legacy audit
baselines. It excludes all audit findings for that exact source filename,
including structural findings, but retains missing, empty, and orphan coverage
checks. Repeat the flag for multiple files. No filenames are excluded by
default; unknown filenames are errors. The JSON report contains `languages`,
`errors`, the number of `accepted` findings, and `excluded_audit` filenames.

## ZIP exports

Run `mpress export` to create `<project-directory>.zip` in the project root. The
archive contains the production HTML, CSS, JavaScript, search indexes, and
static assets at its root, ready for any static host. It does not contain source
Markdown, the development bar, or authoring endpoints.

Use an explicit destination when another tool expects a fixed artifact name:

```sh
mpress export --strict docs-release.zip
```

M-Press refuses to replace an existing archive unless `--force` is supplied.
Use `--drafts` only for private review archives.

## Migration

```sh
mpress import --from starlight ../old-docs --output ./new-docs
```

Options may appear before or after the source path. The importer writes a
`migration-report.md` next to the new project configuration.

## Version artifacts

```sh
mpress versions capture v1.0
mpress versions capture --force v1.0
mpress versions list
mpress versions verify v1.0
mpress versions remove v1.0
```

Version labels may contain letters, numbers, dots, underscores, and hyphens.

## Translation

`mpress translate` reads its API key from the environment variable named by
`translation.apiKeyEnv`. It never writes the key to project files. Normal runs
translate missing and stale segments and preserve manual edits. See
[Translate documentation](/translation/) for provider, glossary, state, and
review details.

Run `mpress translate --estimate --lang fr` before a paid translation. This
command uses the real translation plan but does not initialise or contact the
provider. Token totals are approximate because provider tokenisers and target
language lengths differ. Add `inputPricePerMillion` and
`outputPricePerMillion` to the translation configuration to include a USD
estimate. Use the current prices published for the configured model.

`--harness codex` and `--harness claudecode` override the translation provider
for one command without changing `mpress.yaml`. They invoke the authenticated
`codex` or `claude` executable already installed on the machine. The harness
receives structured translation requests through standard input. M-Press still
validates every returned segment before it writes a target page.

`--model` overrides the configured model for the same command. M-Press passes
the value directly to the selected harness. For example:

```sh
mpress translate --lang ja --harness codex --model gpt-5.6-terra
mpress translate --lang ja --harness claudecode --model claude-opus-5
```

`mpress translate audit --lang CODE` performs a read-only quality check and
returns a non-zero exit status when it finds an error. Add `--harness` and an
optional `--model` to run a second semantic review. The reviewer reports
findings only. It cannot rewrite translated pages.

## MCP knowledge base

Connect a local MCP client directly to a project:

```json
{
  "mcpServers": {
    "product-docs": {
      "command": "mpress",
      "args": ["knowledge", "/path/to/product-docs"]
    }
  }
}
```

The command builds the site before it starts. Pass `--no-build` when CI has
already produced and verified the knowledge artifacts.

Use HTTP when the client cannot start a local process:

```sh
mpress knowledge --transport http --host 127.0.0.1 --port 3100
```

The endpoint is `http://127.0.0.1:3100/mcp`. A non-loopback address requires a
bearer token supplied with `--token`. Keep the endpoint private unless it sits
behind a production authentication layer.
