---
title: Contribute to M-Press
description: Open a pull request with a test that demonstrates a gap in M-Press. A fix is welcome too.
order: 35
---

M-Press contributions start with a pull request and a test that demonstrates a
gap. We do not accept issues. If something is broken or missing, show the
expected behaviour in a reproducible test.

If you can fix it too, brilliant. A PR with only a failing test is also welcome:
it gives us a concrete example to work from.

## Demonstrate the gap

1. Fork [M-Press](https://github.com/leaanthony/mpress) and create a branch from `main`.
2. Add the smallest test that demonstrates the missing or incorrect behaviour.
   Put it beside the relevant code in a `*_test.go` file and use the existing
   package's test helpers and fixtures.
3. Run the test against the current implementation. Confirm it fails because of
   the gap, rather than a setup error.
4. If you can, fix the implementation and confirm the same test passes.

Use the Go version specified in `go.mod` or newer. The public tests run entirely
from this repository; contributors do not need access to private test suites.

## Open the pull request

Include:

- What you expected and what happens instead.
- The test that demonstrates the gap and the exact command to run it.
- The observed failure before a fix, and the result afterwards if you included one.

Keep the PR focused on one gap. If you are submitting only the failing test,
open a **draft PR** and say that a fix is still needed. The failing check is the
reproduction; the PR can become ready to merge once the gap is fixed and checks
pass.

For an implementation fix, run the affected package tests, then the public checks:

```sh
go test -race ./...
go vet ./...
```

If the change affects documentation or generated output, also run:

```sh
go run ./cmd/mpress build --strict
go run ./cmd/mpress check
```

## Security reports

Report suspected vulnerabilities privately using the
[security policy](https://github.com/leaanthony/mpress/blob/main/SECURITY.md).
Do not put credentials or exploit details in a public PR.

## Contributions to a documentation site

To enable reader contributions on a site you build with M-Press, see
[Enable site contributions](/how-to/enable-site-contributions/).
