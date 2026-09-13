# M-Press

M-Press is an MIT-licensed documentation static-site generator written in Go. Authors
write Markdown and ordinary HTML; a single binary builds a searchable static
site without Node.js.

Static generation, local search, structural translation, versioning,
accessibility, contribution tools, and static deployment are included.

```sh
mpress init docs
cd docs
mpress dev
mpress build --strict
```

## Install and upgrade

Download the archive for your OS and architecture and `checksums.txt` from the
[v1.0.0 release](https://github.com/leaanthony/mpress/releases/tag/v1.0.0).
Archives are available for Linux, macOS, and Windows on AMD64 and ARM64.
Verify the archive's SHA-256 digest against `checksums.txt` before extracting it:

```sh
# Linux
sha256sum mpress-linux-amd64.tar.gz
# macOS
shasum -a 256 mpress-darwin-arm64.tar.gz
```

```powershell
# Windows PowerShell
Get-FileHash .\mpress-windows-amd64.zip -Algorithm SHA256
```

Place `mpress` (or `mpress.exe`) in a directory on your `PATH`, then run
`mpress version`. To upgrade, verify and extract the new release and replace
the existing executable. Keep your previous binary for rollback.

## This documentation builds itself

The user manual in `docs/` is an M-Press site. From this repository, run:

```sh
go run ./cmd/mpress build --strict
go run ./cmd/mpress check
go run ./cmd/mpress dev
```

The generated site is written to `site/`. CI runs the strict self-build and link
check, so changes to M-Press must keep its own documentation buildable.

## Principles

- No network access during builds.
- No silent content loss.
- One readable `mpress.yaml` configuration file.
- Plain static output that works on any host.
- Multilingual routing, structural machine translation, and documentation
  versioning remain available in OSS.
- Visitor accessibility tools are included by default and store preferences
  only in the browser.
- Development mode can audit the current page with Google Lighthouse without
  adding Node.js to the M-Press build path.

## Commands

- `mpress init [directory]`
- `mpress build [--strict]`
- `mpress export [archive.zip] [--strict] [--force]`
- `mpress dev [directory] [--port 3000]`
- `mpress dev --repo URL --checkout DIRECTORY [--branch docs/mpress]`
- `mpress contribute <site-url-or-repository> [--branch BRANCH] [--checkout DIRECTORY]`
- `mpress clean`
- `mpress check`
- `mpress import --from starlight <source> --output <directory>`
- `mpress translate [status] [--lang CODE] [--file PAGE]`
- `mpress versions capture|list|verify|remove`
- `mpress deploy [target] [--production]`
- `mpress version`

The [initial MVP contract](docs/mvp.md) records the original project scope. The implementation
shape and private/public test split are documented in
[Architecture](docs/architecture.md).

## Contributing

No issues: open a PR with tests that demonstrate a gap. A fix is welcome too;
if you only have the failing test, submit a draft PR. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities privately.

## License

[MIT](LICENSE)
