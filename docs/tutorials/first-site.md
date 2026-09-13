---
title: Build your first M-Press site
description: Install M-Press, create a site, edit one page, and check the output.
order: 2
---

This tutorial gives you a working documentation site. You need Go 1.26.5 or later.

## Install M-Press

Run this command:

```sh
go install github.com/leaanthony/mpress/cmd/mpress@latest
```

Confirm that the executable is available:

```sh
mpress version
```

## Create the site

Run these commands:

```sh
mpress init product-docs
cd product-docs
mpress dev
```

M-Press shows the local address. Open that address in your browser.

You now have this project:

@filetree
product-docs/
  mpress.yaml  Site configuration
  content/
    index.md  Home page
  static/  Images and other static files
@end

## Change the home page

Open `content/index.md`. The two source files produce the page shown in the
third tab:

@tabs
[content/index.md]
```md {title="content/index.md"}
---
title: Product documentation
description: Learn how to use the product.
---

Welcome to the product documentation.

## Install the product

Follow the installation steps for your operating system.
```

[mpress.yaml]
```yaml {title="mpress.yaml"}
site:
  title: Product documentation
  description: Learn how to use the product.
build:
  contentDir: content
  staticDir: static
  outputDir: site
search:
  enabled: true
accessibility:
  enabled: true
```

[Output]
@image{light="/images/mpress-docs-light.png" dark="/images/mpress-docs-dark.png" alt="The generated documentation site in M-Press" expand}
@end

Save the file. The development server rebuilds the site and refreshes the page.

## Check the release

Stop the development server. Then run these commands:

```sh
mpress build --strict
mpress check
```

The `site/` directory now contains the static website. The strict build reports
unsupported content and duplicate routes as errors. The check reports broken
local links, fragments, scripts, styles, and images.

You have built and checked your first M-Press site.

## Continue

- [Create a custom landing page](/how-to/custom-landing-page/).
- [Define the navigation](/navigation/).
- [Read the authoring reference](/authoring/).
