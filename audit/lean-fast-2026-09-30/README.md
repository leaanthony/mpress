# Current-feature audit

The feature-freeze audit covers filesystem safety, mobile interactions, build and
translation correctness, and measured performance. Beads epic `mp-b52` is the
active backlog; `.beads/issues.jsonl` is its portable issue export. Consult Beads
for current priorities, dependencies and acceptance criteria rather than keeping
another TODO list here.

## Qualified fixes

- Output/route guards protect source, assets and state before destructive work.
- Rooted IO confines authoring, export, conversion, translation and snapshot
  operations; valid internal aliases and explicit CLI directory selection remain.
- Accessibility and contribution panels scroll in constrained viewports. Browser
  checks cover touch input and focus restoration.
- Knowledge bundles have stable section identities, integrity checks and bounded
  loading. CSS purging retains runtime classes; hero images load eagerly.

The latest conversion/migration checkpoints passed public race tests and vet.
Their generated files matched the preceding checkpoints: 100 self-doc files and
2,165 files from Wails `337a7571b2b78063bdb73c3efb99aae1a1e5e34b`
(`docs/mpress/`, strict build with CSS purging disabled). These are checkpoint
comparisons, not a claim that output matches the initial audit baseline.

Private qualification retains three failures/six assertions. Windows/macOS
cross-compilation is recorded separately from native Linux execution. The private
Wails input was unavailable; public corpus qualification is separate. Conversion
rollback, wider snapshot recovery/coordination, deployment/metadata boundaries and
final qualification remain open. LCP field improvements and live publication are
not established by the laboratory measurements.

## Reproduction and archived evidence

[Browser tools](tools/README.md) remain a separate Go module. Ordinary regression
checks live alongside production packages. Portable historical reproduction
scripts live in `replays/`; their expected failing-before cases are checked against
archived source revisions, with compilation failures rejected as evidence.

Fetch the history used by those scripts before running them:

```sh
git fetch https://github.com/taliesin-ai/mpress.git tag archive/audit-lean-fast-20260930-pre-squash-20261005
go test -race ./...
go vet ./...
python3 audit/lean-fast-2026-09-30/replays/s03-conversion/replay-before.py
python3 audit/lean-fast-2026-09-30/replays/s03-migration/replay-before.py
```

[The archived audit](https://github.com/taliesin-ai/mpress/tree/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30) retains the full reports, measurements and
raw public evidence. Historical paths in Beads notes refer to that archive.
Private fixtures and raw private logs remain outside the public repository.
Store new generated evidence outside Git or in an ignored `evidence/` or `results/`
directory, and record concise conclusions and artifact locations in Beads.
