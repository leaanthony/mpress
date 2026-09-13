---
title: Deployment
description: Publish M-Press output to any static host.
order: 10
---

An M-Press build is a directory of static files. No M-Press process, Go runtime,
database, or Node.js server is required after the build.

## Release checklist

```sh
mpress build --strict
mpress check
```

Publish the configured output directory, `site/` by default, as the web root.

## Deploy from M-Press

M-Press can create a Cloudflare Pages project and publish the completed static
build. Configure the target once:

```sh
mpress deploy configure cloudflare \
  --account YOUR_ACCOUNT_ID \
  --project your-documentation
```

Keep the credential outside `mpress.yaml`:

```sh
export CLOUDFLARE_API_TOKEN=your-token
```

The token needs permission to edit Cloudflare Pages projects. Create a preview
first, then publish the same checked output when it is ready:

```sh
mpress deploy
mpress deploy --production
```

Netlify uses an existing site ID, domain, or site name. Set a team slug only
when the site belongs to a team:

```sh
mpress deploy configure netlify \
  --site docs.example.com \
  --account your-team
export NETLIFY_AUTH_TOKEN=your-token
mpress deploy
mpress deploy --production
```

Netlify previews are draft atomic deploys. They do not replace the published
site. Netlify tokens stay in the environment and are never written to
`mpress.yaml`.

In development, select **Deploy** from the M-Press menu at the bottom of the
generated site. Choose Cloudflare Pages or Netlify. The form edits the same
configuration and runs the same release checks as the command line. It never
writes an API token to the project.

## Base URL and sitemap

Set the public origin before a production build:

```yaml
site:
  baseURL: https://docs.example.com
```

M-Press then generates `sitemap.xml` using that origin. Routes remain directory
URLs ending in `/`, so hosts should serve each directory's `index.html`.

## Select a host

Use any host that serves static files:

- For GitHub Pages, upload `site/` with a Pages workflow.
- For Cloudflare Pages, publish `site/` or upload a completed build.
- For object storage, copy `site/` to the bucket and configure the CDN.
- For your own server, copy `site/` to the web root for Apache, Caddy, or Nginx.

Configure the host to serve `index.html` for directory routes. If you publish the
site below a path prefix, make sure that root-relative links use the correct origin.

Because search is a static JSON index, it works on every one of these hosts
without a server-side search service.
