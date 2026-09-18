# Contributing to M-Press

No issues: open a pull request with a reproducible test that demonstrates a gap
in M-Press. Include the expected behaviour, actual behaviour, and the exact test
command and output.

If you can fix it too, brilliant. If you have only the failing test, open a draft
PR and say that a fix is still needed. Keep each PR focused on one gap.

See the [contribution guide](docs/contributing.md) for setup and validation.
Report suspected vulnerabilities privately using [SECURITY.md](SECURITY.md).

## Publishing the documentation

Every push to `main` publishes the documentation after the public CI checks pass.
CI builds M-Press from that commit and runs `mpress deploy --target cloudflare
--production`. That command performs a strict documentation build, checks links
and assets, and uploads to the `mpress-docs` Cloudflare Pages project configured
in `mpress.yaml`. Pull requests and forks do not publish production documentation.
Superseded commits are skipped so older CI runs cannot replace newer docs.

An upstream repository administrator must set the Actions repository secret
`CLOUDFLARE_API_TOKEN` to a Cloudflare API token with Account → Cloudflare Pages →
Edit permission for the configured account. Keep credentials out of the project
configuration. To retry publication, run the CI workflow manually on `main`.
