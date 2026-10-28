# Reproducing the audit

Run the source-selected Go toolchain from the repository root:

```sh
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
go build -trimpath -ldflags='-s -w' -o /tmp/mpress-audit-bin ./cmd/mpress
/tmp/mpress-audit-bin build --strict --json
/tmp/mpress-audit-bin check
go test ./internal/site -run '^$' -bench '^BenchmarkBuild(Cold|Warm)$' -benchmem -benchtime=3x -count=3
```

The `BenchmarkBuild*` workload writes 200 numbered pages plus a home page.
Its cold case clears the M-Press parse cache and output; the warm case clears
output after priming the parse cache. Both enable asset minification/CSS purge.

For the touch reproduction, this directory is a separate Go module so browser
tooling does not enter the production module or binary. It serves already-built
site output through a temporary local HTTP server and uses installed Chrome:

```sh
cd audit/lean-fast-2026-09-30/tools
go run . /absolute/path/to/mpress/site > mobile-results.json
```

`mobile.go` compares real touch swipes before and after an in-browser diagnostic
CSS injection. The injection does not edit repository source or site output.
The original failing baseline is preserved in the [archived evidence](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/evidence/mobile.json).
The browser currently defaults to `/usr/bin/google-chrome`; adjust that local
path for another installation. This harness is Chromium-only.

The assertion-based popover regression check uses generated production output
without injecting CSS:

```sh
MPRESS_TOUCH_SITE=/absolute/path/to/generated/site go test -count=1 -run '^TestUtilityPopoverInteraction$' -v .
```

Run it against both accessibility-enabled and disabled builds. It checks all
utility popovers on `/configuration/` at 375x667 and 375x375: closed focus/hit
testing, open focus/hit testing, immediate and settled exit transitions, and
page scrolling after closing. Without `MPRESS_TOUCH_SITE`, this test skips;
a skip does not qualify browser behaviour. It is a focused regression, while
the broader overflow-family matrix remains Beads task `mp-b52.2.4`.

For the compact phone table search toolbar, the regression builds its own fixture
with the candidate CLI:

```sh
MPRESS_TABLE_SEARCH_BIN=/absolute/path/to/candidate go test -count=1 -run '^TestCompactMobileTableSearch$' -v .
```

It measures the real toolbar at 390x844 in English and French, requires the
toolbar to track the search field plus normal padding, and checks the 1280x900
row layout is unchanged. It then exercises keyboard search with live status,
pagination, sorting, and a touch-opened column filter. Set
`MPRESS_TABLE_SEARCH_SCREENSHOTS` to an external directory to save screenshots.
Without `MPRESS_TABLE_SEARCH_BIN` the test skips; a skip does not qualify browser
behaviour.

The imported-brand regression builds its own text-title and logo fixtures with
the candidate CLI:

```sh
MPRESS_BRAND_BIN=/absolute/path/to/candidate go test -count=1 -run '^TestImportedBrandTitleStaysInsideHeader$' -v .
```

It requires the measured title box to stay inside the header at 1280 and 1024
while the complete title remains the link's text, checks the short-title and
logo variants, and confirms the phone header still hides the brand. Without
`MPRESS_BRAND_BIN` the test skips; a skip does not qualify browser behaviour.

The imported phone-header regression builds its own English/French site with
the candidate CLI and dispatches real touch events:

```sh
MPRESS_HEADER_NAV_BIN=/absolute/path/to/candidate go test -count=1 -run '^TestImportedPhoneHeaderNavigation$' -v .
```

At 390 and 375 it requires the menu and search centres to hit their own
controls, navigates through the open sidebar, and opens search to a
locale-correct result and taps that result through to its page, in both
menu/search orders with repeated Escape and close dismissals. It also covers
the imported desktop header and a default-theme comparison with a Ctrl+K/Escape
keyboard journey. Chrome defaults to `/usr/bin/google-chrome`; set
`MPRESS_CHROME` for another installation. Without `MPRESS_HEADER_NAV_BIN` or
Chrome the test skips; a skip does not qualify browser behaviour.

The landing primary-navigation regression builds its own small landing site with
20 configured header links and dispatches real touch swipes:

```sh
MPRESS_HEADER_NAV_BIN=/absolute/path/to/candidate go test -count=1 -run '^TestLandingPrimaryNavigationTouch$' -v .
```

At 320x667, 390x844 and 375x375 it requires the 44px primary toggle to receive
its own centre tap, focuses the first link on open, swipes to the last of the 20
links while only the dropdown scrolls, follows it to `/guide/`, and returns home.
It then checks Escape closes the dropdown and restores trigger focus, an outside
tap closes it, and a body swipe still advances the long landing document. A
1280x900 desktop smoke check confirms the toggle is hidden and direct navigation
is rendered, with its first link receiving taps. It does not qualify whether an
arbitrarily long desktop navigation bar fits the viewport.
This is focused phone-primary-navigation coverage, not the full overflow-family
matrix. Chrome defaults to `/usr/bin/google-chrome`; set `MPRESS_CHROME` for
another installation. Without `MPRESS_HEADER_NAV_BIN` or Chrome the test skips;
a skip does not qualify browser behaviour.

For responsive article navigation, run the generated default site and the Wails
release site through the same browser regression:

```sh
MPRESS_TOC_SITE=/absolute/path/to/generated/site go test -count=1 -run '^TestResponsiveTOC$' -v .
MPRESS_TOC_SITE=/absolute/path/to/wails/docs/mpress/site MPRESS_TOC_ROUTE=/quick-start/installation/ go test -count=1 -run '^TestResponsiveTOC$' -v .
```

It checks native touch opening, keyboard selection and focus, Escape, reaching
the last section in a scrolling chooser, site-menu overlap, desktop resizing,
both themes, and the no-JavaScript fallback. Run against purged and unpurged
production CSS. Set `MPRESS_TOC_SCREENSHOTS` to an external directory to save
screenshots. Without `MPRESS_TOC_SITE` the test skips.

Historical public evidence is linked from the [audit summary](../README.md);
keep new generated reports outside Git or in an ignored results directory. Raw private conformance logs and private
fixtures are deliberately kept in the private test environment. To run that
suite, select the exact candidate explicitly with `MPRESS_SOURCE`/`MPRESS_BIN`
and supply installed browser/Axe prerequisites. A skipped check is not a pass.

To recreate the large workload, archive the pinned Wails `docs/mpress/` revision
listed in the [audit summary](../README.md) into a disposable directory, then build there with
`--strict --no-purge-css` and run `check`. Do not modify the user's Wails checkout
or substitute a different revision without recording its provenance.

For controlled LCP/resource measurements, use the separate `lcp` program:

```sh
MPRESS_LCP_GZIP=1 go run ./lcp /absolute/path/to/generated/site 3 > lcp-results.json
MPRESS_LCP_GZIP=1 MPRESS_LCP_ROUTE=/docs/community/showcase/portfall/ go run ./lcp /absolute/path/to/legacy/build 2 > article-results.json
MPRESS_LCP_GZIP=1 go run ./lcp https://public-pages-host.example/ 1 > live-results.json
MPRESS_V3_SITE=/absolute/path/to/v3/site MPRESS_V2_SITE=/absolute/path/to/v2/build go test -count=1 -run '^TestLandingVisualContracts$' -v ./lcp
```

Each measurement uses a fresh Chrome context, disabled cache, 150ms latency,
200 KiB/s download, 4x CPU slowdown and 1365px/390px viewports in both themes.
The local server can gzip text assets; use the same setting for both builds.
It observes LCP, layout shifts, long tasks and resource timings until four
seconds after navigation finishes. An autoplay carousel may produce multiple
LCP candidates: the last observed candidate is used, not a cherry-picked first
paint. These are lab samples, not field percentiles or an entire browsing visit.

`TestLandingVisualContracts` checks generated styles, both themes, translated
no-JavaScript hero rendering, and carousel eager/lazy loading plus autoplay with
stable height at desktop/mobile widths. Missing site environment variables skip
the test; a skip does not qualify the change. See [LCP report](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/LCP.md) for pinned build
sources and measurement evidence. Browser dependencies remain outside M-Press's
production module.

Accessibility and contribution scrolling regressions also use production builds:

```sh
MPRESS_A11Y_SITE=/absolute/path/to/generated/site go test -count=1 -run '^TestAccessibilityViewportReachability$' -v .
MPRESS_CONTRIBUTE_SITE=/absolute/path/to/generated/site go test -count=1 -run '^TestContributionDialogPhoneReachability$' -v .
```

Both require the complete public documentation fixture with `/configuration/`.
The contribution fixture must have contributions enabled with this repository
configured, so its script/source-path assertions verify the actual site handoff.
They exercise native touch gestures and keyboard input, share clipping/gesture
primitives, and enforce independent scroll-owner checks. Missing environment
variables skip these checks. Optional `MPRESS_A11Y_REPORT` and
`MPRESS_CONTRIBUTE_REPORT` paths save JSON evidence. See [B02 report](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/B02.md) and [B03 report](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/B03.md)
for viewport coverage, original-CSS replay and test limitations.

The focus matrix builds its own small public fixture with the candidate CLI:

```sh
MPRESS_DIALOG_BIN=/absolute/path/to/candidate go test -race -count=1 -run '^TestDialogOpenerFocus$' -v .
```

It distinguishes native keyboard/mouse/touch activation from script clicks and
waits for focus restoration after all four supported closing methods. See
[B05 report](https://github.com/taliesin-ai/mpress/blob/archive/audit-lean-fast-20260930-pre-squash-20261005/audit/lean-fast-2026-09-30/B05.md) for its no-change diagnosis and stale-oracle replay.
