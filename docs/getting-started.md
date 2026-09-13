---
title: Installation and your first site
description: Install M-Press and build a documentation site in a few commands.
order: 2
---

M-Press requires only the `mpress` executable at runtime.

## Build from source

Until packaged release binaries are published, install from the Go module:

```sh
go install github.com/leaanthony/mpress/cmd/mpress@latest
```

You can also build a repository checkout:

```sh
git clone https://github.com/leaanthony/mpress.git
cd mpress
go build -o mpress ./cmd/mpress
```

@note{type="info" title="Release packaging"}
Pre-built Linux, macOS, and Windows binaries are part of the 0.1 release plan.
The source build is the current installation path.
@end

## Create a site

```sh
mpress init my-docs
cd my-docs
mpress dev
```

Open `http://localhost:3000`. The starter contains:

@filetree
my-docs/
  mpress.yaml  Site configuration
  content/
    index.md  Home page
  static/  Images and other static files
@end

Edit `content/index.md` and save it. The development server rebuilds the site
and refreshes connected browser tabs.

## Build for release

```sh
mpress build --strict
mpress check
```

`--strict` rejects duplicate routes, unsupported components, and other error
diagnostics. `mpress check` verifies local links, fragments, images, scripts,
and stylesheets in the generated HTML.

The default output directory is `site/`. It can be served directly:

```sh
python3 -m http.server --directory site 8000
```

## Use M-Press inside an existing Go checkout

M-Press itself uses this approach. Put `mpress.yaml` at the repository root,
set `build.contentDir` to your documentation directory, and run:

```sh
go run ./cmd/mpress build --strict
go run ./cmd/mpress check
```
