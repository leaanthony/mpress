---
title: Translate documentation
description: Create, update, and review translated M-Press Flavoured Markdown without changing code or document structure.
translationKey: translation-guide
order: 8
---

M-Press treats translation as part of the content workflow. It can
create translated M-Press Flavoured Markdown with an OpenRouter, OpenAI, or OpenAI-compatible
model. The static site does not need the provider or an API key.

M-Press does not charge for the translation workflow, language routing,
freshness tracking, glossary checks, or review state. The selected model
provider may charge for API usage.

## Add target languages

Add every published language to `mpress.yaml`:

@explained
```yaml
site: # (1)
  defaultLanguage: en
  languages: [en, fr, ja] # (2)
  languageLabels: # (3)
    en: English
    fr: Français
    ja: 日本語
  defaultLanguageAtRoot: true
  missingTranslation: link-to-default # (4)
```

(1) Put language settings in the site section of the mpress.yaml file.

(2) List the language codes that M-Press must publish. The first build can use
only the default language. Add another code when its translated content is
ready.

(3) Set the names that readers see in the language selector.

(4) Send readers to the default-language page when the selected translation
does not exist. Set this value to omit to hide unavailable languages instead.
@end

The default language lives directly in the content directory. M-Press writes
other languages to a language directory:

@explained
```text
content/ # (1)
├── index.md # (2)
├── installation.md
├── _nav.yaml
├── fr/ # (3)
│   ├── index.md
│   ├── installation.md
│   └── _nav.yaml
└── ja/ # (4)
    ├── index.md
    └── _nav.yaml
```

(1) The content directory is the default location. Change build.contentDir when
the project uses another location.

(2) Keep default-language pages and navigation directly in the content
directory.

(3) Give each translated language its own directory. Use the same relative
paths as the default language so M-Press can match the pages.

(4) A language directory can contain fewer pages. Here, Japanese does not have
installation.md, so the configured missing-translation policy controls what
the reader sees.
@end

Missing pages are not copied into the generated site. `link-to-default` sends a
visitor to the default-language page and marks the translation as unavailable.
`omit` removes that language from the page selector.

## Configure a provider

This example uses OpenRouter:

```yaml
translation:
  provider: openrouter
  model: openai/gpt-5.4-mini
  baseURL: https://openrouter.ai/api/v1
  apiKeyEnv: OPENROUTER_API_KEY
  sourceLanguage: en
  glossary: glossary.yaml
  styleGuide: docs/writing-standard.md
  stateDir: .mpress/translations
  dataCollection: deny
  requireParameters: true
```

Set the key in the process environment. Do not put it in `mpress.yaml`:

```sh
export OPENROUTER_API_KEY="your-key"
```

For OpenAI, use `provider: openai`, `https://api.openai.com/v1`, and
`OPENAI_API_KEY`. For another compatible provider, use
`provider: openai-compatible` and set its base URL and environment-variable
name.

For local-first translation, use an installed coding agent instead of an API
key:

```yaml
translation:
  provider: codex # or claude
  model: local
  command: codex # optional; defaults to codex or claude
```

M-Press sends the structured request to the command on standard input. The
command must return a JSON object with a `translations` array containing the
same `id` values. This works with an authenticated local Codex CLI or Claude
Code installation and keeps documentation content and credentials on the
machine. A custom wrapper command is supported when the local CLI uses a
different invocation.

Override the configured provider for one run when you do not want to edit the
project configuration:

```sh
mpress translate --lang ja --harness codex
mpress translate --lang ja --harness claudecode
```

The second command invokes the `claude` executable used by Claude Code.
Harness selection is temporary. It does not modify `mpress.yaml`, provider
credentials, or model settings stored in the project.

Select an exact model for one run with `--model`:

```sh
mpress translate --lang ja --harness codex --model gpt-5.6-terra
mpress translate --lang cy --harness claudecode --model claude-opus-5
```

Codex receives the model through `codex exec --model`. Claude Code receives it
through `claude --model`. If `--model` is absent, the harness uses its own
configured default. Claude Code also accepts aliases such as `sonnet` and
`opus`, but a complete model ID gives reproducible translation state.

`requireParameters` tells OpenRouter to select only a backend that supports the
structured response. `dataCollection: deny` excludes providers that may store
the supplied documentation.

Model quality varies by language and documentation set. In the development
tool, use the model comparison workflow to translate the same protected sample
with two candidates and choose the better result without seeing their names.
For French and other European languages, start by comparing
`mistralai/mistral-small-2603`, `openai/gpt-5.4-mini`, and
`google/gemini-3.5-flash-lite`. Treat these as candidates, not a permanent
ranking.

## Check before you translate

Run a status check. This command does not contact the provider:

```sh
mpress translate status --lang fr
mpress translate status --lang fr --file installation.md --json
```

Estimate the paid work before you start it:

```sh
mpress translate --estimate --lang fr
mpress translate --estimate --lang fr --file installation.md --json
```

The estimate reports the planned page targets, segments, provider requests,
and approximate input and output tokens. It exits without contacting the
provider and without writing configuration, translated pages, or state.

To include an approximate USD total, record the configured model's current
price per million tokens:

```yaml
translation:
  inputPricePerMillion: 0.25
  outputPricePerMillion: 2.00
```

Prices are not built into M-Press because providers can change them. Update
these values when you change the model. The estimate is a planning ceiling,
not an invoice. Retries and provider-specific tokenisation can change the final
charge.

Then translate missing and stale text:

```sh
mpress translate --lang fr
mpress translate --lang fr --file installation.md
```

For a new language, add it to the site configuration and translate the complete
site with one command:

```sh
mpress translate --lang fr --add-language --label "Français"
```

This writes the language and reader-facing label to `mpress.yaml`, translates
every default-language Markdown page plus navigation, and writes the
result under the language directory. The operation is resumable. Repeating the
command translates only missing or stale segments.

M-Press uses four concurrent provider requests by default. For a provider with
higher rate limits, use `--workers` with a value from 1 to 16:

```sh
mpress translate --lang fr --workers 8
```

Use a lower value when the provider returns rate-limit errors. Each page is
validated completely before M-Press replaces its target file and state.

If the project is not checked out on this machine, let M-Press create a local
checkout first:

```sh
mpress translate \
  --repo git@github.com:example/docs.git \
  --checkout ../docs-translation \
  --branch main \
  --lang fr
```

M-Press uses the existing Git credential helper or `gh` authentication. It does
not copy a GitHub token into the project configuration. The destination must be
empty, and the cloned repository must contain `mpress.yaml`.

Use `--scope missing` to fill gaps only. Use `--scope all --force` only when you
intend to replace existing machine text and manual conflicts.

You can run the same workflow from the **Translations** item in the development
bar. Start with the outcome you need: add a language, translate the current
page, update a complete language, or mark the current page as reviewed. M-Press
then shows the target, scope, affected pages, and conflict policy before it
writes content. Remote writes require the development server authoring token.

## What M-Press protects

M-Press parses M-Press Flavoured Markdown into a structured document. It sends
exact prose ranges to the provider and patches the returned text into the
original bytes. It does not regenerate the source document.

The translator protects:

- fenced and inline code;
- link destinations and automatic links;
- HTML and component declarations;
- component directives, identifiers, escaped characters, metadata references, and
  unknown emoji shortcodes;
- visible component attributes such as titles, labels, descriptions, and
  alternative text are translated without exposing their JSON syntax;
- configuration placeholders such as `{name}` and `${HOME}`;
- Markdown punctuation and line structure;
- navigation links and YAML nesting.

Requests use structured output keyed by a stable segment ID. Long pages are
split into bounded requests with the page title, outline, nearby text, style
guide, and glossary as context. M-Press rejects incomplete responses and any
result that changes protected structure.

Each heading, paragraph, and table cell is one translation unit.
Inline code and other immutable constructs become placeholders inside that
unit. The model can therefore move an inline command to the position required
by the target language without separating it from the sentence. In a
`@filetree`, paths stay protected and human-readable descriptions are
translated.

## Track freshness and human edits

Commit `.mpress/translations/` to the repository. Each sidecar file records
source and target hashes, provider, model, prompt version, and review state. It
does not contain an API key.

M-Press reports these states:

| State | Meaning |
| --- | --- |
| `missing` | No target text exists. |
| `machine-translated` | The target matches the last provider output. |
| `stale` | The source changed after machine translation. |
| `manual` | A person edited the target while the source stayed unchanged. |
| `conflict` | Both the source and the manually edited target changed. |
| `reviewed` or `final` | A review tool recorded an approved state. |

Normal translation runs preserve manual text and stop short of conflicts. Use
`--force` only after review.

Mark a reviewed page from the development tool or the CLI:

```sh
mpress translate review --lang fr --file installation.md
mpress translate review --lang fr --file installation.md --status final
```

If the source or target changes later, M-Press replaces the approval with the
correct stale, manual, or conflict state.

Add a permanent `translationKey` to frontmatter when a page may move or change
its filename:

```yaml
---
title: Install M-Press
translationKey: installation
---
```

M-Press uses this key to move the target and its state without sending unchanged
text to the provider again. Keys must be unique across the site.

## Use a glossary

Create a YAML glossary when a product or technical term must be consistent:

```yaml
terms:
  - source: M-Press
    translations:
      fr: M-Press
      ja: M-Press
    note: Product name. Do not translate it.
  - source: development server
    translations:
      fr: serveur de développement
```

M-Press includes applicable terms in each request. It also rejects a result
when a required translation is missing.

## Review generated files

Audit a translated language before human review:

```sh
mpress translate audit --lang fr
mpress translate audit --lang fr --file installation.md
```

The audit is local and does not contact a provider. It checks that translated
files and segments exist, validates document structure and glossary terms,
finds unchanged source prose, and warns when French requirement language may
have lost its force. An error gives the command a non-zero exit status, so the
same check can run in CI.

Use a separate coding-agent model for an independent semantic review:

```sh
mpress translate audit \
  --lang fr \
  --harness claudecode \
  --model claude-opus-5
```

The reviewer receives paired source and target segments. It cannot change the
files. M-Press accepts findings only when they refer to a real file and segment,
then labels them with an `ai-` code. This stage can incur model usage charges.
Use a different model family from the translator when you want an independent
assessment.

Translated pages are normal Markdown files. Review and edit them with the same
tools as the source pages. Then run:

```sh
mpress build --strict
mpress check
```

The local tool provides translation, state, glossary enforcement,
and validation. A hosted product can add shared translation memory, assignments,
budgets, pull requests, vendor workflows, and organisation audit history.
