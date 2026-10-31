# Historical Wails external-link regression

Small attributed excerpts from `wailsapp/wails`, MIT © 2018–Present Lea Anthony
(see LICENSE). Copied explicitly as regression fixtures; no implementation is
imported. Source revision: `e4452516a0adfefe0487e0a93cddf009166c8120`, the
verified base of [PR #6515](https://github.com/wailsapp/wails/pull/6515), linked to
[issue #6514](https://github.com/wailsapp/wails/issues/6514). The PR was OPEN and
unmerged when inspected on 2026-10-31; its head was
`7b24545b02659d44a9f404e8197ebd0a11fb3ebd`.

For each of English (no prefix), `fr/` and `ko/`, these fixture paths retain the
original frontmatter and the indicated body excerpt from
`docs/mpress/content/<locale>/<path>`:

- `features/notifications/overview.md`: Examples heading, paragraph and list.
- `features/windows/options.md`: inline-code source-file link paragraph.
- `features/windows/multiple.md`: final horizontal rule and Questions paragraph,
  including its Discord and nonexistent multi-window example links.

Home `index.md` files are minimal test scaffolding. `controls.md` is authored
control data, not an excerpt: it uses the verified published-tag file/directory
URLs. No fixture URLs are rewritten to localhost. A controlled Go transport
models their exact GitHub paths; local HTTP-server tests cover real client
redirect, timeout and rate-limit behavior. Tests never contact GitHub.

Live verification was performed separately on 2026-10-31 using GitHub HTTP GET
and the GitHub contents/ref APIs:

| Destination suffix after `https://github.com/wailsapp/wails/` | GET |
| --- | --- |
| `tree/releases/v3.0.0/v3/examples/notifications` | 404 |
| `blob/releases/v3.0.0/v3/pkg/application/webview_window_options.go` | 404 |
| `tree/releases/v3.0.0/v3/examples/multi-window` | 404 |
| `tree/v3.0.0/v3/examples/multi-window` | 404 |
| `tree/v3.0.0/v3/examples/notifications` | 200 |
| `blob/v3.0.0/v3/pkg/application/webview_window_options.go` | 200 |

GitHub `git/ref/heads/releases/v3.0.0` returned 404. The `v3.0.0` tag resolved to
commit `7b11001b92b712210b27d8cb53430e83f67ad8e4`. The contents API independently
returned an array for `v3/examples/notifications` (directory), `type: file` for
`v3/pkg/application/webview_window_options.go`, and 404 for
`v3/examples/multi-window`. The Git tree agrees: notifications is tree
`6186fcb7528354f6e18c8251a471a77660d30f57`; window options is blob
`4f7c75f0c29646432f0054598d4d158cf0b1e5b7`.

Availability alone does not establish that a different example is an appropriate
replacement. Tests deliberately still reject `multi-window` after replacing the
branch with the valid tag; they do not silently infer `window` as a repair.

Before/after reproduction (same command test, offline):

```sh
go test ./cmd/mpress -run '^TestWailsExternalCommandRegression$' -count=1
```

Copy this fixture directory and `cmd/mpress/external_test.go` to a clean detached
worktree at M-Press `4b98efbe146666e9b8ee9b9b9c98517ca97d7e84` and run that
command. The original `check` succeeds, but the regression fails because no
external audit/report exists. On the implementation it passes, finding three
missing destinations, each referenced by EN/FR/KO, and accepting both controls.

Additional regression commands:

```sh
go test ./internal/check -run 'TestExternal|TestWails' -count=1
go test ./internal/translate -run 'TestWailsURLCorrection|TestSyncLinks' -count=1
go test ./cmd/mpress -run TestExternalExitProcess -count=1
```
