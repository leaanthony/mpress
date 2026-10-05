# Changelog

Changes to existing M-Press behaviour, tests and performance are recorded here.

## Unreleased

### Fixed — 2026-10-02

- Translation checks now share one pinned root across discovery, coverage, audit and hash-pinned exceptions. Coverage and exception readers reject external files while preserving internal aliases and reviewed hashes. Borrowed handles remain owned by callers. Beads `mp-b52.1.3`; migration/conversion, deployment and metadata qualification continues.
- Authoring conversion now confines document and translation-state IO, deletion and configuration saves to its pinned project root. Exclusive temporary writes preserve permissions and unrelated `.tmp` files; explicit CLI directory selection remains supported. Five authenticated regressions reproduce external reads/overwrites and temporary-file loss. Beads `mp-b52.1.3`/`mp-b52.1.6`; whole-conversion rollback remains open.
- Translation-state migration now pins the selected previous snapshot separately from the current project, confining configuration, triplet reads and state writes. Nine boundary regressions include an external state overwrite; valid aliases, original approvals, dry-run/idempotency and borrowed ownership remain supported. Beads `mp-b52.1.3`; deployment and metadata delegates remain under audit.

- Model comparison and workload estimation now share the pinned project filesystem for source and provider inputs. Five authenticated failing-before cases include comparison sending external content to both local test providers; safe aliases, read-only sampling and workload estimates remain supported. Removed obsolete ambient input readers from production. Beads `mp-b52.1.3` remains open for check, migration/conversion, deploy and metadata delegates.

- Translation audit and refinement now confine source, target, state and provider inputs through the same pinned root, including independent review and final re-audit. Fifteen failing-before cases cover external reads and three external overwrites; internal aliases and clean-segment preservation remain supported. Removed the unused ambient atomic writer. Beads `mp-b52.1.3`; remaining delegates stay open.

- Translation planning and execution now share the pinned project filesystem across source, target, state, provider inputs and relocation cleanup. External links and provider-time parent replacements are rejected; safe internal aliases remain supported. Style-guide/glossary limits apply before allocation. Generated documentation is unchanged. Beads `mp-b52.1.3` remains in progress for remaining delegates; see [archived S03.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S03.md).

### Fixed — 2026-10-01

- Translation review now confines source/target/state reads, page-key discovery, review-state writes and old-sidecar cleanup to the pinned project root. Authenticated regressions reject five external boundaries, including an external state overwrite; eight internal alias cases preserve reviewed state and target bytes. Other translation operations remain under audit. Beads `mp-b52.1.3`; see [archived S03.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S03.md).

- Knowledge loading now bounds manifest/artifact bytes, gzip expansion, each bundle, the combined mounted corpus and version enumeration. Truncated/checksum-invalid streams and unsupported index schemas fail; concurrent replacement/recovery is qualified. Full current and mounted Wails bundles remain supported, with generated files unchanged and loading cost measured. Beads `mp-b52.1.8`; limits are documented in `docs/knowledge.md`.

- Knowledge sections now have distinct deterministic identities for repeated or unnamed headings. Legacy bundles are repaired in memory after digest verification; duplicate page/term IDs fail explicitly. Mounted resources remain distinct from a current release with the same version and stable across release changes. Legacy reindexing cost is measured in [archived S08.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S08.md); resource budgets are qualified. Beads `mp-b52.1.8` is closed.

- Knowledge reads now pin project/site/version/bundle boundaries and validate all artifact filenames before reading, rejecting external links and traversal from the authenticated authoring endpoint. Internal aliases, compressed digests and version-resource behavior remain supported; generated docs are byte-identical. Beads `mp-b52.1.3`/`mp-b52.1.8`; resource and identity qualification is recorded in [archived S08.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S08.md).

- Contribution dialogs now scroll on phones and short windows, containing touch gestures while expanded setup and instructions remain reachable. Six native touch cases qualify commands, translation, copy and Escape/close focus; embedded viewer assets were regenerated. Beads `mp-b52.2.3`; see [archived B03.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/B03.md).

- Accessibility settings now use one scrollable panel on phones and short windows. The body can shrink at other sizes; focus scroll padding keeps controls clear of clipping edges. Real touch/keyboard checks cover all settings, colour options, width/font preferences, reset and close across ten viewport/text/theme cases, including 200% text simulation. Beads `mp-b52.2.2`; see [archived B02.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/B02.md).

- Snapshot mounting now pins the verified snapshot and output boundaries, rejecting linked destinations that overwrite external or protected files. Shared reader locks coordinate list/verify/mount with capture/removal. Legacy read-only stores, version navigation and safe aliases remain supported; nested manifest-named assets are published. Ordinary and mounted docs remain byte-identical. Beads `mp-b52.1.7`; recovery/platform and build-wide coordination qualification remains open.

- Snapshot capture now confines reads and writes, owns a unique workspace and preserves the previous snapshot across copy, manifest and promotion failures. Capture/removal writers coordinate across processes; failed rollback retains a recovery backup. Capturing `v1` preserves `v1.tmp`, and nested manifest-named assets are indexed correctly. Full-corpus snapshot checksums remain identical; capture performance cost is recorded in [archived S07.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S07.md). Beads `mp-b52.1.7` remains open for mounting and reader coordination.

- Version listing, verification and removal now use rooted artifact/snapshot boundaries, reject unsafe labels and manifest paths/schemas, and preserve source/state/output when the artifact store overlaps them. Authenticated REST/MCP regressions reproduce the original external deletion. Ordinary and version-mounted documentation output remains identical. Beads `mp-b52.1.7` is now P0 and stays in progress for capture, overwrite recovery, concurrency and mounting; see [archived S07.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S07.md).

- Export now rejects external project-state parents before creating a workspace, uses rooted workspace cleanup and archive reads, and retains safe internal state aliases. An authenticated HTTP regression catches temporary external writes; full Wails ZIP entry contents remain unchanged. Beads `mp-b52.1.3`; export/snapshot qualification continues.

- Confined direct REST/MCP authoring file operations, uploads, backups, trees and preview/site serving to the project using rooted IO; protected preview cleanup from source, state and published-output overlap. Delegated source/cache/static/CSS/navigation inputs now use the same boundary. Safe aliases and ordinary artifacts remain covered. Beads `mp-b52.1.3` stays in progress for the remaining delegates; see [archived S03.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S03.md).
- Extended shared build/clean/preview guards to preserve configured navigation, translation glossary, writing guide and contributor guide files outside the main content directory. Added failing-before data-preservation checks for these cases.

### Fixed — 2026-09-30

- Page routes and language components now reject traversal and nonportable path forms. Canonical route/output preflight catches duplicate, multilingual, generated-file, static and snapshot collisions before replacing the current site; cached pages and direct page writes use the same contract. Root/nested/Unicode routes and supported static overrides remain covered. Beads `mp-b52.1.2`; see [archived S02.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S02.md) for failing-before tests, artifact equivalence and remaining qualification.
- Build and clean now share canonical output validation and rooted removal, rejecting output that overlaps source, assets, configuration, Git or persistent state. Safe internal aliases, missing output directories, export and live preview remain supported; final output symlinks are replaced without deleting their targets. Sentinel regressions and identical ordinary-doc artifact hashes qualify Beads `mp-b52.1.1`; see [archived S01.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/S01.md).
- Retained literal runtime classes from inline scripts during production CSS purging, including whitespace-separated class lists and template literals. Added failing-before regression coverage at the purge and build boundaries.
- Marked both frontmatter hero theme images eager, high priority and asynchronously decoded. Prepared in [PR #7](https://github.com/leaanthony/mpress/pull/7); public CI passed, publication and field qualification remain pending.
- Prepared Wails documentation fixes for blocking CSS and reported legacy carousel/article images. The released M-Press 1.0.18 production build now safely purges CSS with page-local animation styles: 191,986 → 123,740 bytes. Repeated cold throttled v3 LCP medians improved 9–10%; this is laboratory evidence, not a claim that the reported field outliers are resolved.
- Closed utility popovers no longer intercept mobile touch scrolling or keyboard focus; open panels remain interactive. Thanks to [@dogukanoklu](https://github.com/dogukanoklu), M-Press's first community contributor, for [PR #6](https://github.com/leaanthony/mpress/pull/6).
- Regenerated the embedded MPD viewer CSS and its HTML cache keys for the popover fix, restoring viewer reproducibility tests.
- Added browser checks against generated production CSS for popover focus, hit testing, closing transitions and touch scrolling, including builds with accessibility disabled.

### Maintenance — 2026-10-05

- Moved generated audit evidence and detailed reports to the preserved archive; retained a concise summary, portable replay scripts and browser checks. Kept the Beads export and minimal setup, removed tracked Ripwire acknowledgements, and ignored future generated audit output.

### Maintenance — 2026-10-01

- Restored the historical authoring-confinement replay after later config/server seams changed, preserving original tested behavior and requiring exact failures across all three stages. Beads `mp-b52.5.8`; the final audit gate remains open.

- Qualified image and contribution focus restoration with 60 native/script activation and close cases, each repeated twice. Corrected the private image keyboard test to use Enter and await closure; its false failure is removed while three private failing tests remain. Production modal behavior is unchanged. Beads `mp-b52.2.5`; see [archived B05.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/B05.md).

### Maintenance — 2026-09-30

- Audited current `main` at `f6ca003f255fe7636e97d63ba80e4bfcd9cffa32` under a feature freeze.
- Established Beads epic `mp-b52` with 48 prioritized work items, five workstreams, dependencies and acceptance criteria.
- Recorded public race/coverage, vet, vulnerability, strict-build, link-check, fuzz, browser touch, filesystem-safety and full Wails corpus results in the [archived audit](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30).
- Established build time/allocation, generated asset size, binary size and startup baselines for measured optimization.
- Added user-reported LCP task `mp-b52.4.12`, controlled browser measurements and theme/no-JavaScript/autoplay regression checks. See [archived LCP.md](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/LCP.md) for implementation, evidence and remaining deployment work.

The audit baseline remains preserved; verified product fixes are recorded separately above.
