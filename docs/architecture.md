---
title: Architecture
description: How the small M-Press Go executable turns local content into deterministic static output.
order: 20
---

M-Press is one Go module and one executable. The dependency direction stays
inward and intentionally boring:

```text
CLI -> config -> discovery -> Markdown/components -> navigation -> static output
                                                        |              |
                                                        +-- search     +-- check
                                                        +-- versions   +-- dev server
```

The generator owns deterministic transformation of local files. Generated sites
contain HTML, CSS, JSON, assets, and a small progressive-enhancement script.
JavaScript is not required to read or navigate the documentation.

The Starlight importer is an adapter at the boundary. It converts known syntax
into M-Press's portable authoring format and reports everything else. Supporting
a migration must not complicate the core parser indefinitely.

## Testing boundary

The public repository contains representative tests for contributor confidence.
The private `mpress-tests` repository tests the compiled binary using broader
fixtures, migration corpora, adversarial inputs, and eventually visual baselines.
Releases require both; public CI can be enabled by setting the repository variable
`PRIVATE_CONFORMANCE=true` and the secret `MPRESS_TESTS_TOKEN`.
