# Changelog

## Unreleased

### Fixed

- Stop rich-table conversion from escaping the rest of a row when a cell contains an unmatched backtick, so valid native tables with literal backticks or adjacent live actions convert in both directions. Keep matched single- and double-backtick code spans containing pipes intact. Beads `mp-b52.3.8`.

- Keep inline code spans that wrap across physical lines inside blockquotes and list items literal during Markdown/MPD conversion, so every example keeps its exact text and container instead of gaining escaped backticks or a soft-wrap backslash. Beads `mp-b52.3.7`.

- Preserve component syntax in inline code examples, including lists and quotes, and refresh previously cached pages on rebuild. Beads `mp-b52.3.7`.

- Preserve expandable images and preview-tab source panels when converting documentation between Markdown and MPD. Beads `mp-b52.3.8`.

- Recheck the selected project directory after a queued export waits for a running build, refusing with actionable restart guidance instead of exporting a different project installed at the old path. Preserve both directories and keep ordinary exports working after restoration. Beads `mp-b52.1.3`.

- Preserve descendant selectors in nested CSS so rules such as `& :not(...)` keep styling child elements after minification. Beads `mp-b52.3.6`.

- Preserve comment-only boundaries during CSS minification so compound selectors keep matching the intended elements. Keep real descendant whitespace, value token separation and license comments. Beads `mp-b52.3.6`.

- Preserve values and references in template interpolations and escaped identifiers during JavaScript minification. Keep binding names when their references cannot be safely analyzed, while still compacting the script. Beads `mp-b52.3.6`.

- Restore phone navigation by preventing invisible search-button text from intercepting taps on nearby controls. Preserve localized search results and keyboard focus. Beads `mp-b52.2.16`.

- Generate imported themes that keep text titles inside the header instead of letting long titles wrap over the page chrome. The full title remains the brand link's accessible name, short titles and logo branding keep their current size, and phone headers stay unchanged. Beads `mp-b52.2.15`.

- Prevent Markdown/MPD round trips from silently deleting pricing plans and testimonials. Preserve plan names, prices, links, features and recommendations alongside quotes, attribution and controls; keep native structured components and literal examples working. Existing projects receive the corrected pages on their next rebuild through the parse-cache epoch bump, without editing source or clearing caches. Beads `mp-b52.3.8`.

- Keep embedded reader-control demos working when importing Markdown: the accessibility demo now stays a real container, so its controls render and the surrounding section, prose and neighbouring components keep their boundaries instead of failing the build. Beads `mp-b52.3.8`.

- Preserve compact buttons inside prose during Markdown/MPD conversion and refresh cached pages so their labels, destinations and variants keep working. Beads `mp-b52.1.6`.

- Preserve compact button actions and appearance during Markdown/MPD conversion, keeping reader accessibility controls working and ordinary link destinations, sizes and icons intact. Retain variant precedence, quoted spaces and pipes, and literal examples. Beads `mp-b52.3.8`.

- Keep directive and tab-label examples inside backtick, tilde and longer code fences literal during Markdown-to-MPD conversion, so a valid `@note`/`:::note` container or tabs group converts instead of failing as unclosed. Real nested directives and tabs still convert, and genuinely unclosed directives are still refused. Beads `mp-b52.3.8`.

- Remove the large empty space around table search on phones so readers can reach the data with less scrolling. Preserve desktop layout and existing table controls. Beads `mp-b52.2.14`.

- Preserve interactive table rows and controls during Markdown/MPD conversion, and stop rendering an extra separator row. Keep literal code-cell pipes and ordinary table alignment intact. Beads `mp-b52.3.8`.

- Reject translation-state migrations when a saved review has changed since planning, and retain saves that collide with publication with recovery details for retry. Beads `mp-b52.1.6`.

- Preserve completed translation review saves through already-open file handles during document conversion, reporting the retained state copy for recovery and retry. Beads `mp-b52.1.6`.

- Recheck consumed documents before discarding conversion recovery copies, preserving completed saves through already-open file handles and reporting where to recover them. Beads `mp-b52.1.6`.

- Preserve completed settings saves through already-open file handles during document conversion, reporting the recovery copy when cleanup cannot safely proceed. Beads `mp-b52.1.6`.

- Preserve saved settings and guide/glossary references during document conversion, including references added outside the editor. Report unreadable or invalid saved settings before any files change; reload valid settings to retry. Beads `mp-b52.1.6`.

- Refuse development-server operations when the selected project directory has moved, been replaced or its alias has changed targets. Report how to restart instead of rebuilding another project at the old path; restoring the original directory permits retry. Beads `mp-b52.1.3`.

- Return optional configuration maps and lists as empty objects and arrays instead of JSON `null` from the authoring MCP server and REST settings endpoint, so reading, editing the title and saving the complete configuration works without client-side normalisation. Beads `mp-b52.3.10`.

- Refuse builds and exports whose static tree contains an unreadable entry, such as a dangling asset link, and keep the previous published site or archive intact instead of silently publishing an incomplete result. Projects without an optional static directory still build normally. Beads `mp-b52.1.4` (audit S04).

- Keep site settings saved at the very end of a format conversion: the final configuration publication never replaces a newer author save. The conflict is reported and the newer settings, original configuration and documents are retained. Beads `mp-b52.1.6`.

- Detect conflicting site-settings saves when converting style-guide or glossary paths, preserving newer settings and restoring the original documents. Beads `mp-b52.1.6`.

- Preserve translation approval and model changes saved during failed conversion recovery, with diagnostics that identify retained recovery copies. Beads `mp-b52.1.6`.

- Keep retained conversion recovery documents out of generated pages and reader indexes, including live preview. Beads `mp-b52.1.10`.

- Leave a failed conversion's named recovery directory untouched when the author retries: retained originals and any documents inside them keep their names, bytes, modes and aliases instead of being converted again. Ordinary hidden author directories and an explicitly selected conversion directory still convert. Beads `mp-b52.1.6`.

- Preserve edits, replaced aliases and permission changes made to generated documents during failed conversion cleanup. Report any retained recovery copy instead of deleting newly saved work. Beads `mp-b52.1.6`.

- Preserve source files and aliases recreated while a failed conversion rolls back. If restoring a consumed source encounters newly saved work, leave that work intact and report the retained recovery copy. Beads `mp-b52.1.6`.

- Refresh authoring settings and the served preview after successful configuration changes, including moving the document root when `outputDir` changes. Invalid or failed builds preserve the last usable preview, while guarded configuration snapshots and revision checks protect concurrent settings operations. Beads `mp-b52.3.2`.

- Refresh live preview after creating, deleting, renaming or editing files even when timestamps do not advance, including changes within large assets and configured navigation outside the content directory. Save bursts are coalesced, scanning errors are reported, and generated output and unrelated files do not trigger rebuilds. Beads `mp-b52.3.1`.

- Keep the localized phone search launcher correctly sized and clickable: normalize its mobile button box so the 44px target stays in the header, and keep theme, accessibility and language controls visible and usable after search closes while focus returns to the launcher. Beads `mp-b52.2.13`.

- Give repeated interactive components on one documentation page independent identities. Tabs, carousels, data tables, preview/file tabs and API playgrounds allocate document-scoped IDs during rendering, keep local ARIA and popover references paired, and report duplicate authored IDs without rewriting them. Repeated builds and concurrent page renders now produce stable bytes. Beads `mp-b52.2.7`.

- Confine version captures to the selected build output subtree. Capture pins the output directory once and reads, walks and copies only through that pinned source, so an output asset link cannot package project-private files from elsewhere in the project while a rejected capture preserves the previous verified snapshot. Internal aliases within the output and internally aliased output directories still work. Beads `mp-b52.1.7`.

- Keep version capture and removal checks inside the existing store lock, preventing transient path errors while another capture replaces the same version. Beads `mp-b52.1.7`.

## v1.0.25 — 2026-10-26

### Fixed

- Generate the configured local RSS feed from published blog posts instead of preserving an empty imported placeholder. Include escaped titles and descriptions, canonical item links and GUIDs, dates and tags; exclude drafts, future posts and translated duplicates. Beads `mp-b52.13`.

- Restore original documents, permissions, internal aliases, translation state and configuration when a format conversion encounters an ordinary filesystem error. Reject duplicate planned targets before changing files, publish new document targets with exclusive no-replace staging so a destination created while conversion runs is never overwritten, and reject a source whose content changed after planning so an author edit made during conversion is preserved instead of discarded. If rollback detects changed content or cannot restore an original, report the retained recovery copies. Filesystems without hard-link support fail safely during new-target publication. This recovery does not provide concurrent-writer isolation or crash recovery. Beads `mp-b52.1.6`.
- Rename the header menu, search, theme and accessibility control IDs to `mpress-menu-toggle`, `mpress-search-toggle`, `mpress-theme-toggle` and `mpress-accessibility-toggle` so they cannot collide with Markdown heading anchors, updating the matching CSS and runtime selectors. Beads `mp-b52.2.6`.
- Keep the last published site in place when a build fails. Builds now generate into a validated staging directory and replace the published output only after generation and strict diagnostics succeed, restoring the previous output if promotion or post-promotion indexing fails. Version and knowledge indexing still uses final output paths, and the output directory permissions are preserved. Beads `mp-b52.1.5`.
- Preserve configured custom CSS during production purging, including runtime-created classes, escaped selectors and modern at-rules. Keep cascade order and minification while removing unused built-in styles. Beads `mp-b52.3.5`.
- Confine deployment assets to the selected output and reject files changed after hashing. Reject symbolic links and other non-regular output entries, and keep deployment configuration and output reads within the project. Beads `mp-b52.1.3`.

## v1.0.24 — 2026-10-26

### Fixed

- Synchronise source-owned code blocks and layout in existing translations even when no prose needs translating. Preserve manual prose, avoid provider requests for protected-only updates, create code-only target pages, and retain no-op runs without rewriting files. Beads `mp-b52.12`.

## v1.0.23 — 2026-10-26

### Fixed

- Repair preserved bold inline-code spans beside CJK text without treating visible code as an empty span. Missing code and genuinely empty spans remain rejected. Beads `mp-b52.11`.

- Preserve the intrinsic proportions of same-origin interactive SVG objects in articles, including tall contributor grids, while keeping explicit author sizing and safely ignoring inaccessible cross-origin documents. Beads `mp-b52.10`.

## v1.0.22 — 2026-10-25

### Fixed

- Automatically repair preserved bold delimiters that stop parsing beside CJK punctuation by emitting equivalent strong tags at the same placeholder boundaries. Missing, duplicated or reordered markers and empty bold spans remain rejected, and full protected-content validation still applies. Beads `mp-b52.8`.

## v1.0.21 — 2026-10-25

### Fixed

- Use each page’s frontmatter image for Open Graph and X cards, falling back to the site social image. Emit matching absolute image URLs and select large-image cards whenever an image is available. Beads `mp-b52.7`.

## v1.0.20 — 2026-10-25

### Fixed

- Restore mobile and tablet article navigation with a native section disclosure, keyboard focus and Escape handling, scrollable touch targets, and anchors below the fixed header. Keep the article bar behind site navigation, including existing Starlight-imported themes. Beads `mp-b52.2.12`; Wails [#6338](https://github.com/wailsapp/wails/issues/6338).

### Security

- Build releases with Go 1.26.10 and update the Go networking dependencies to clear the current vulnerability gate. Beads `mp-b52.5.9`.

## Earlier changes

### Fixed

- Preserve literal class and ID tokens from inline page scripts during production CSS purging, including class lists and template strings. Pages with runtime animation classes can retain their styles while removing unused CSS.
- Mark frontmatter hero images as eager, high-priority resources with asynchronous decoding for both light and dark themes.

### Previously merged

- Closed utility popovers no longer intercept mobile touch scrolling or keyboard focus. Thanks to [@dogukanoklu](https://github.com/dogukanoklu), M-Press's first community contributor, for [PR #6](https://github.com/leaanthony/mpress/pull/6).
Changes to existing M-Press behaviour, tests and performance are recorded here.

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
