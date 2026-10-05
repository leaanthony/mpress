# M-Press backlog

Use Beads (`bd`) to manage the current-feature audit under epic `mp-b52`:

```sh
bd ready
bd show mp-b52
```

The local Dolt database is authoritative and is ignored by Git. Commit the portable
issue export after task updates:

```sh
bd export -o .beads/issues.jsonl
```

To restore issues into an initialized Beads database, run
`bd import .beads/issues.jsonl`. This restores issue data; the JSONL export is not a
full database backup. Use Dolt backup/sync for database history. Keep credentials,
locks and runtime state untracked. The config, project metadata and ignore rules
are shared setup; historical evidence paths in issue notes resolve through the
[archived audit](../audit/lean-fast-2026-09-30/README.md).
