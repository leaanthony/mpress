# Wails external-link investigation and review evidence

Investigation baseline recorded 2026-10-31. The report below captures the
pre-release review for the M-Press v1.0.27 implementation. Wails
integration and PR validation are tracked separately. Historical document excerpts
are explicitly requested regression
fixtures with their original MIT attribution.

## Authority and existing behavior

- M-Press baseline `4b98efbe146666e9b8ee9b9b9c98517ca97d7e84` (v1.0.26
  source plus its release-evidence ledger commit). Read root `AGENTS.md`.
- Wails baseline `e4452516a0adfefe0487e0a93cddf009166c8120`. Read root and
  `docs/AGENTS.md`, plus `docs/mpress/CONTRIBUTING.md`.
- GitHub PR [6515](https://github.com/wailsapp/wails/pull/6515) was OPEN/unmerged,
  head `7b24545b02659d44a9f404e8197ebd0a11fb3ebd`, linked to
  [issue 6514](https://github.com/wailsapp/wails/issues/6514), also OPEN.
  Its base equals the inspected Wails revision. Its proposed URL and exception
  edits were not presumed merged and were not copied into Wails.

M-Press `internal/check/check.go` deliberately skips URI schemes and
protocol-relative references. `cmd/mpress/main.go:checkSite` uses the local
build/link operation in `internal/operations/operations.go`. The newer
`internal/check/rendered.go:Rendered` publication check explicitly makes no
network requests; external hosts are skipped. These local checks are useful
and remain unchanged.

Translation `internal/translate/check.go`, `audit.go`, `engine.go` and
`markdown_protection.go` protect link destinations and translated structure,
not external availability. Whole-segment source hashes mean a URL edit becomes
stale, or conflicts with manual translation. A missing-only update attempts
reuse and rejects mismatched protected URLs. The ordinary stale scope can call
a translation provider; it is not a provider-free URL propagation mechanism.

Inspected Wails `docs/mpress/scripts/{build.sh,check_site.py,check_translations.py}`,
`docs/Taskfile.yml`, `.github/workflows/docs-mpress.yml`,
`.github/workflows/nightly-release-v3.yml` and
`.github/workflows/weekly-release-v2.yml`, plus `v3/tasks/release/release.go`.
The docs pipeline downloads the latest stable tool, checks translations, builds,
checks local links/assets, runs Python tests and validates rendered output.
The v3 release workflow tests a source SHA and publishes generated release notes
and the changelog; v2 has its separate historical documentation workflow.
None gives continuous external-link availability coverage after branch deletion.

## Implemented scope

`mpress check external` extends the existing check command family and reuses its
HTML collector. It explicitly opts into network access against an existing
build, independent of local/publication checks. It collects every generated
locale, deduplicates fragment-free initial destination URLs, and retains sorted
HTML page paths. JSON includes status, HTTP code, final URL and attempts.

Exit 0: selected destinations returned 2xx. Exit 1: confirmed HTTP 404/410
(takes precedence), or an operational/usage error. Exit 2: inconclusive results
without confirmed breakage. Timeouts, access denials, rate limits, TLS/DNS and
server failures remain inconclusive. Default timeout 10s per attempt, one
additional transient retry, one-second delay, maximum ten HTTP(S) redirects.
A Retry-After response stops further attempts for that URL until a later audit.
No persistent results/exception cache is created.

GET availability cannot prove correct content, remote anchors, authentication,
or file/directory semantics. HTML href/src scanning does not cover CSS/JS link
discovery, srcset or commands inside code. Host filters apply to initial URLs;
redirects may leave that host. Audit only trusted build output.

`mpress translate sync-links --lang CODE --file PAGE --previous-source FILE`
previews by default; `--write` applies. It compares the complete old/new English
Markdown and accepts only simple inline HTTP(S) destination changes. It validates
the old target, replaces matching parsed protected destinations, validates the
new target and writes atomically. Real French/Korean labels, prose, whitespace
and formatting survive byte for byte. Human-diverged URLs, structural damage,
mixed prose/layout corrections and ambiguous/unsupported syntax are refused.
State sidecars and audit exceptions are never written. Existing stale/manual
states can remain and need explicit review; the helper grants no new approval.

Wails adds `.github/workflows/docs-external-links.yml`: daily, branch-deletion
and manual runs audit a validated build of current master. The independent job
fails for broken or inconclusive results, retains JSON and produces a readable
Actions summary with affected pages. Maintainers can inspect the artifact and
use their existing Actions notification preferences. This catches deletion after
release without introducing external HTTP checks into normal build/deploy gates.
URL-prefix filters limit the Wails job to repository tree/blob destinations
rather than thousands of changelog/profile URLs. It audits current master, not
a crawl of the live deployment. GitHub scheduling
and deletion-event limits still apply; daily runs provide a backstop.

The build script gains an opt-in audit flag and an absolute MPRESS_BIN override
for candidate testing. Normal download/checksum/version verification remains
intact. At investigation time, released v1.0.26 lacked the command. The Wails rollout
must use v1.0.27 or newer; release publication is handled separately from this
investigation. No silent compatibility bypass is added.

## Historical regression and independent verification

`testdata/wails-links/README.md` records exact Wails source revision/paths,
excerpt boundaries, MIT attribution, tag commit and tree/blob identities.
Fixtures retain real frontmatter, headings, list and inline-code links from
English/French/Korean notification, window options and multi-window pages.
Two additional controls link to an actual directory and file at v3.0.0.

Separate GitHub GETs returned 404 for the three historical release-branch URLs,
and for multi-window even after switching to the tag. Both controls returned
200; the contents API confirmed directory/file types, the branch ref API
returned 404 and Git trees confirmed the objects. Tag v3.0.0 points to
`7b11001b92b712210b27d8cb53430e83f67ad8e4`. The existing replacement example
`v3/examples/window` is tree `f38677e8ca5ae26a370aa1b70715e8ac560b2cac`;
its main.go creates additional windows (lines 116, 126 and onward). The checker
does not infer this semantic replacement.

The automated regression uses controlled transport with the real GitHub URL
shapes, never live GitHub. Local HTTP-server tests cover actual redirect,
rate-limit, timeout and retry behavior. Subprocess tests verify actual exit
statuses 0/1/2. The historical command test uses only APIs that existed at the
baseline, so the identical test runs before and after the patch.

## Commands and results

From `/var/home/lea/projects/mpress-external-link-audit`:

```sh
go test ./cmd/mpress -run '^TestWailsExternalCommandRegression$' -count=1
go test ./internal/check -run 'TestExternal|TestWails' -count=1
go test ./internal/translate -run 'TestWailsURLCorrection|TestSyncLinks' -count=1
go test ./cmd/mpress -run TestExternalExitProcess -count=1
go test -race ./...
go vet ./...
go run ./cmd/mpress build --strict
go run ./cmd/mpress check
go build -buildvcs=false -o /tmp/mpress-external-link-audit ./cmd/mpress
```

All passed; M-Press generated 62 pages/100 files with no broken local references.
The local buildvcs override works around an unrelated parent-directory VCS
stamping problem, not a source/CI setting. Final affected packages were also
checked with `go test -race ./internal/check ./internal/translate ./cmd/mpress`.

Before change, in detached `/tmp/mpress-before-external-audit` at the baseline,
copy the same fixture tree and `cmd/mpress/external_test.go`, then run:

```sh
go test ./cmd/mpress -run '^TestWailsExternalCommandRegression$' -count=1
```

Expected FAIL reproduced: local validation passed, but no external report was
available (`usage: mpress check [site ...]`). After change the identical test
passes, reporting three broken destinations across all three fixture locales
and accepting both controls. A second regression ensures the nonexistent
multi-window directory remains broken after branch-to-tag substitution.

From `/var/home/lea/projects/wails-docs-external-link-audit`:

```sh
MPRESS_BIN=/tmp/mpress-external-link-audit bash docs/mpress/scripts/build.sh
python3 -B -m unittest discover -s docs/mpress/scripts -p 'test_*.py'
bash -n docs/mpress/scripts/build.sh
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/docs-external-links.yml
```

Passed: 2,050 pages/2,217 files, no broken local links/assets, no rendered-site
errors, 25 Python tests and workflow syntax validation. Pipeline tests run
offline with controlled tools: normal builds make no audit call; opt-in audit
results retain all locale paths and propagate exit 1/2 and JSON artifacts.
The actual new CLI/transport behavior is exercised by the M-Press tests above.
Hosted scheduled jobs have not run because these are unpublished local changes.

Both candidate and released v1.0.26 translation checks report zero errors and
1,471 warning identities; none added or removed. Eighteen message strings can
name a different missing protected token because an existing audit map's
iteration order is nondeterministic; page/segment/code identities are identical.
These pre-existing warnings include protected-content diagnostics and remain
warnings under Wails' existing policy; no exceptions were refreshed.

From Wails `v3/`:

```sh
go test scripts/validate-changelog.go scripts/validate-changelog_test.go
go test ./tasks/release ./tasks/releasegate
python3 -B -m unittest discover -s scripts -p test_generate_release_changelog.py
```

Passed (including 14 Python changelog tests). No Wails runtime source was changed.

Additional review: `golangci-lint run ./internal/check/... ./internal/translate/...
./cmd/mpress/...` reports the same 21 pre-existing findings at baseline and
candidate after normalizing shifted line numbers; no new findings. Ripwire
quality delta has zero gating findings (non-gating complexity/test-discovery
advisories remain), and existing command signatures are unchanged. Both
worktrees pass `git diff --check`.

Raw logs, GitHub verification JSON and comparison evidence stay outside Git in
`/var/home/lea/.codex/experiments/mpress-external-links-20261031/`.
