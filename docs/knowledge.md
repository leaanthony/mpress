---
title: MCP knowledge base
description: Give agents precise, cited access to an M-Press documentation site without exposing authoring controls.
order: 5
---

M-Press turns every production site into a portable knowledge base. The same
build creates the reader-facing HTML and a deterministic set of agent-facing
artifacts. There is no crawler, database, Node process, or external indexing
service.

## What the build creates

The `knowledge/` directory contains four JSON files:

| File | Purpose |
| --- | --- |
| `manifest.json` | Site identity, languages, version, pages, and a digest for the complete bundle. |
| `pages.json` | Complete plain-text pages with source paths and canonical URLs. |
| `chunks.json` | Sections split at semantic headings with stable IDs and exact citation URLs. |
| `index.json` | A deterministic lexical index for local search. |

Long sections are split at word boundaries. Every part retains its page,
heading path, language, version, tags, source file, and citation. M-Press also
loads captured version artifacts when they are available, so an agent can
filter a search to a specific documentation release.

## Connect an MCP client

Run the server over standard input and output:

```sh
mpress knowledge /path/to/project
```

Configure a client to start that command:

```json
{
  "mcpServers": {
    "product-docs": {
      "command": "mpress",
      "args": ["knowledge", "/path/to/project"]
    }
  }
}
```

The server exposes three tools:

| Tool | Purpose |
| --- | --- |
| `site_info` | Read the site identity, languages, current version, artifact digest, and content counts. |
| `search` | Search sections with optional language, version, tag, and result-limit filters. |
| `get_page` | Read one complete page by its stable ID, route, or canonical URL. |

Pages are also normal MCP resources. Search results link to section resources,
which lets an agent retrieve only the supporting passage and preserve its exact
web citation.

## Security boundary

The knowledge server is separate from the development server. All its tools are
declared read-only, idempotent, non-destructive, and closed-world. It has no
editing, shell, Git, configuration, deployment, or translation tools. Document
content is explicitly treated as untrusted source material rather than agent
instructions.

For local use, prefer standard input and output. For HTTP, bind to loopback:

```sh
mpress knowledge --transport http --host 127.0.0.1 --port 3100
```

M-Press refuses to expose HTTP on another interface unless you provide a bearer
token with `--token`. A public hosted service should add proper user identity,
access control, rate limits, and audit logs in front of the same portable bundle.

## Measure retrieval quality

Keep representative user questions in a JSON evaluation suite. Each case names
the expected page routes and can restrict the language, version, or tags. Run it
after documentation or ranking changes:

```sh
mpress knowledge evaluate . --suite knowledge-evaluation.json
```

The report includes Recall@1, Recall@3, Recall@5, mean reciprocal rank, citation
coverage, and the top result for every failed case. Add `--json` for CI or a
dashboard. The M-Press repository includes a 30-question Wails v3 suite as a
working example.
