---
title: Versioning
description: Capture, verify, and mount immutable documentation releases.
order: 9
---

M-Press versions are completed static builds, not alternate source trees. A
captured version remains deployable even when the current documentation changes.
The complete version workflow and reader selector are built in.

## What is included

M-Press can:

- capture a completed static build under a release label;
- write a SHA-256 manifest for every captured file;
- verify that a release is complete and unchanged;
- mount several releases under stable routes;
- show available releases in the generated navigation bar;
- list and remove local release artifacts.

The published versions remain static files. Readers do not need an M-Press
account, hosted service, or server-side application.

## Enable versions

```yaml
versioning:
  enabled: true
  current: next
  artifactsDir: .mpress/versions
```

## Capture a release

```sh
mpress build --strict
mpress check
mpress versions capture v1.0
mpress versions verify v1.0
```

Capture copies the current output and writes `mpress-version.json` containing a
SHA-256 checksum for every file. `verify` fails if a captured file is missing or
has changed.

Run `mpress build` again after capturing. Enabled versions are mounted beneath
`/versions/<label>/`, `versions.json` is generated, and the current site displays
a version selector.

```text
site/
├── index.html
└── versions/
    ├── versions.json
    └── v1.0/
        └── index.html
```

Use `mpress versions list` and `mpress versions remove <label>` to manage local
artifacts. `capture --force <label>` intentionally replaces an existing label.

@note{type="warning" title="Commit or store the artifacts"}
If versions must survive a clean checkout, keep `.mpress/versions` in durable
storage or commit it according to your release policy.
@end
