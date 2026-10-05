# Current-feature quality and performance work

The user has frozen new product features. Test, fix and optimize existing behaviour.
Do not remove an existing capability to obtain a smaller binary or faster benchmark.

The active backlog is Beads epic `mp-b52`. The concise audit summary and archive
link live in `audit/lean-fast-2026-09-30/README.md`. Beads stores authoritative
statuses, dependencies and acceptance criteria; `.beads/issues.jsonl` is the
portable export. Keep reusable checks in source control and generated logs,
profiles and tool reports outside Git. Historical paths in task notes refer to
the archived audit linked from the summary.

At each bounded change:

1. Read `bd ready` and `bd show ID`; address confirmed P0/P1 bugs before optimizations.
2. Claim the issue with `bd update ID --status in_progress --assignee taliesin-ai`.
3. Reproduce defects and add meaningful behaviour regression checks. For an
   optimization, record a representative baseline and profile first.
4. Implement a bounded change. Preserve current contracts and unrelated user work.
5. Run affected tests, then appropriate build/race/browser/link checks. Measure
   performance on the same workload and machine. A source pattern alone is not
   evidence that an optimization helps.
6. Update `CHANGELOG.md`, record evidence and commit in Beads notes, and close
   only when the acceptance criteria are met. An investigation can be resolved
   with a documented evidence-based no-change decision; confirmed bugs require fixes.
7. Export the ledger to `.beads/issues.jsonl` and keep the repository snapshot
   current. Dolt remains the authoritative database; JSONL is interchange,
   not a full database backup.

Use `gh` for GitHub operations and the `taliesin-ai` profile for commits:
`taliesin-ai <281494191+taliesin-ai@users.noreply.github.com>`.
Scope Git identity to this worktree; do not change the identity of sibling branches.
Private conformance fixtures and raw private logs stay outside the public repository.

Only claim performance improvements demonstrated by repeatable before/after
measurements. Record unavailable browser/device/native-platform checks honestly.
Do not close the parent epic or final qualification task while required child work remains.
