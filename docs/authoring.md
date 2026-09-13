---
title: Markdown and HTML
description: Structure M-Press content with Markdown, HTML, frontmatter, links, assets, and drafts.
order: 5
---

Content files use `.md` or `.markdown`. M-Press supports CommonMark plus tables,
task lists, strikethrough, footnotes, typographic substitutions, and automatic
heading IDs.

## Development workflow

Run `mpress dev` and open the generated site. A development-only bar appears at
the bottom of every page. Use its M-Press menu to change common configuration,
run release checks, create a deployment, or restart the project guide.

The first editing action opens **Setup** when the current project is not on a
work branch. Continue in the current checkout, clone a repository, or create and
clone a GitHub fork. M-Press creates a work branch before it opens the editing
tools.

For the shortest existing-repository workflow, clone and start it from one CLI
command:

```sh
mpress dev --repo git@github.com:example/docs.git \
  --checkout ../docs-work \
  --branch docs/improve-site
```

The development UI opens editing tools directly because the checkout is already
on the prepared branch. If you clone or fork from **Setup** instead, copy the
single `mpress dev "/path/to/checkout"` command shown when preparation finishes.

A published site can provide an even shorter reader workflow. The **Contribute**
action gives the reader one command that downloads M-Press and runs
`mpress contribute <page-url>`. M-Press finds the repository and source file,
prepares a local branch, opens the matching page, and asks what the reader wants
to improve. See [Contribute from a published page](/how-to/enable-site-contributions/).

Configuration opens in a draggable glass panel on desktop and a contained sheet
on mobile. The page stays visible and scrollable behind it. Site title, colour
scheme, accent colour, and hover colour update as you change their fields.
Closing the panel restores unsaved values. Saving validates `mpress.yaml`,
rebuilds the site, and keeps the new values.

Edit Markdown in your normal text editor. M-Press watches project files, rebuilds
the site, and reloads the browser after a successful change. The development bar
and write endpoints are not included in `mpress build` output.

## Write in the browser

Open **Blog**, select a post, and use **Write** for visual editing or **Markdown**
for the source. Both editors accept dropped image files and pasted screenshots.
The image tool provides the same workflow when you prefer to choose a file.

Drop one image to use it in every colour scheme. Drop two images whose filenames
contain `light` and `dark` to create a theme-aware pair. M-Press lets you confirm
the image description, crop, position, brightness, and filename before it saves
the files under `static/images/`.

The editor saves an ordinary Markdown image for one file. It saves the native
`@image` form for a light and dark pair. You can move between **Write** and
**Markdown** without losing that source.

## Test a page with Lighthouse

Open the M-Press menu and select **Lighthouse audit**. M-Press tests the page
that is open in the browser and reports scores for performance, accessibility,
best practices, and SEO. You can use the mobile or desktop test profile.

The audit is optional. Building and serving an M-Press site does not require
Node.js. The audit uses a globally installed `lighthouse` command when one is
available. Otherwise, it can run Lighthouse through `npx` after you explicitly
start the audit. Current Lighthouse releases require Node.js 22 or later and a
local Chrome or Chromium installation.

Set `MPRESS_LIGHTHOUSE` to the path or command name of a Lighthouse executable
when it is installed in a non-standard location.

## Connect an agent with MCP

`mpress dev` includes an MCP server in the same binary. The terminal prints the
endpoint and a random server token when development starts:

```text
M-Press MCP:    http://localhost:3000/__mpress/mcp
MCP token:      4c21...
```

Configure an MCP client to use Streamable HTTP and send the token as a bearer
credential:

```json
{
  "mcpServers": {
    "mpress": {
      "url": "http://localhost:3000/__mpress/mcp",
      "headers": {
        "Authorization": "Bearer 4c21..."
      }
    }
  }
}
```

The MCP server can inspect the project, list and edit source files, update the
complete configuration, run checks, capture versions, and deploy configured
targets. File and configuration writes require the revision returned by the
corresponding read tool. M-Press creates a backup and rebuilds the site after a
successful change.

The MCP endpoint does not accept the token in its URL. Send the same token on
every request using the `Authorization` header. Restarting `mpress dev` creates
a new token unless you supply a stable token with `mpress dev --token VALUE`.

## Frontmatter

```yaml
---
title: Build an application
description: Create and package your first application.
slug: guides/first-application
order: 20
draft: false
layout: landing
tags: [guide, beginner]
author: Documentation team
---
```

`title` is recommended. Without a title, M-Press uses the first heading and then
the filename. `slug` overrides the route. `order` controls generated navigation
ordering. Drafts appear in `mpress dev` and `mpress build --drafts`. They do not
appear in a normal release build.

The default layout is a documentation article. Set `layout: landing` to replace
the documentation columns with an authored Markdown layout. The common header
remains. M-Press does not add an automatic heading or page links to a landing
layout. See [Create a Markdown landing page](/how-to/custom-landing-page/).

## Routes

| Source | Route |
| --- | --- |
| `index.md` | `/` |
| `installation.md` | `/installation/` |
| `guides/index.md` | `/guides/` |
| `01-guides/02-build.md` | `/guides/build/` |

Numeric prefixes help order files without becoming part of the public URL.
`guide.md` and `guide/index.md` therefore collide; strict builds report the
problem rather than picking one silently.

## Links and assets

Root-relative documentation links are usually the clearest:

```md
[Install M-Press](/getting-started/)
![Application window](/images/application.png)
```

Place the image at `static/images/application.png`. Run `mpress check` after
building to catch missing targets and fragments.

## Light and dark images

Use the native image component when one image does not work in both colour
modes:

```md
@image{light="/images/architecture-light.png" dark="/images/architecture-dark.png" alt="System architecture"}
```

Both files remain ordinary static assets. M-Press shows the image that matches
the visitor's selected colour mode. Always provide useful alternative text.

Add the `expand` option when readers need to inspect a larger version. Clicking
the image opens it in a centred overlay. Readers can close the overlay with its
close button, the background, or the Escape key.

```md
@image{src="/images/architecture.png" alt="System architecture" expand}
```

## Ordinary HTML

HTML passes through the renderer when Markdown is not sufficient:

```html
<details>
  <summary>Show advanced details</summary>
  This remains ordinary, portable HTML.
</details>
```

M-Press does not execute JSX or framework components. An unknown uppercase MDX
component such as `&lt;ProductDemo /&gt;` becomes a visible warning marker and an
error diagnostic. This prevents migration content from vanishing unnoticed.
