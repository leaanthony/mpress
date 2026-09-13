---
title: Data and product components
description: Present comparisons, releases, status, dates, plans, and proof.
order: 8
---

## Comparison matrix

@matrix{highlight="M-Press"}
| Capability | M-Press | Typical Node SSG |
| --- | --- | --- |
| Single binary | ✓ | ✗ |
| Markdown and HTML | ✓ | ✓ |
| Runtime required | ✗ | ✓ |
| Built-in versions | ✓ | ~ |
@end

@details{title="Source"}
```md
@matrix{highlight="M-Press"}
| Capability | M-Press | Typical Node SSG |
| --- | --- | --- |
| Single binary | ✓ | ✗ |
| Markdown and HTML | ✓ | ✓ |
| Runtime required | ✗ | ✓ |
| Built-in versions | ✓ | ~ |
@end
```
@end

## Service status

@status{service="Build service" state="operational"}
All systems normal
@end

@status{service="Translation queue" state="degraded"}
Jobs may take longer than usual
@end

@details{title="Source"}
```md
@status{service="Build service" state="operational"}
All systems normal
@end

@status{service="Translation queue" state="degraded"}
Jobs may take longer than usual
@end
```
@end

## Calendar

@calendar{month="2026-08" style="compact"}
Use the arrows to browse months.
@end

@details{title="Source"}
```md
@calendar{month="2026-08" style="compact"}
Use the arrows to browse months.
@end
```
@end

## Changelog

@changelog
### v0.2.0 (2026-08-01)
#### Added
- Complete component library
- Interactive terminal copy
#### Fixed
- Theme icon alignment
- Table-of-contents rhythm

### v0.1.0 (2026-07-30)
#### Added
- Initial static site compiler
@end

@details{title="Source"}
```md
@changelog
### v0.2.0 (2026-08-01)
#### Added
- Complete component library
- Interactive terminal copy
#### Fixed
- Theme icon alignment
- Table-of-contents rhythm

### v0.1.0 (2026-07-30)
#### Added
- Initial static site compiler
@end
```
@end

## Release notes

@release{version="0.2.0" date="2026-08-01" type="minor"}
### Highlights
- M-Press now documents every supported component.
### New Features
- Terminal, diff, API, calendar, pricing, and tutorial renderers.
### Bug Fixes
- Consistent component spacing and dark-mode colours.
@end

@details{title="Source"}
```md
@release{version="0.2.0" date="2026-08-01" type="minor"}
### Highlights
- M-Press now documents every supported component.
### New Features
- Terminal, diff, API, calendar, pricing, and tutorial renderers.
### Bug Fixes
- Consistent component spacing and dark-mode colours.
@end
```
@end

## Pricing

The pricing component displays example plans. Checkout and subscription
management require an external service. These are fictional product plans.

@pricing{cols="2"}
### Personal
$0
[Get started](/getting-started/)
- ✓ One project
- ✓ Community support
---
### Team
$15/month
[Learn more](/data-components/)
recommended
- ✓ Shared projects
- ✓ Priority support
@end

@details{title="Source"}
```md
@pricing{cols="2"}
### Personal
$0
[Get started](/getting-started/)
- ✓ One project
- ✓ Community support
---
### Team
$15/month
[Learn more](/data-components/)
recommended
- ✓ Shared projects
- ✓ Priority support
@end
```
@end

## Testimonials

@testimonials{autoplay="0"}
“The content stays ordinary Markdown.”
Documentation author
---
“The output works without a client framework.”
Platform engineer
@end

@details{title="Source"}
```md
@testimonials{autoplay="0"}
“The content stays ordinary Markdown.”
Documentation author
---
“The output works without a client framework.”
Platform engineer
@end
```
@end

## QR codes

QR codes are generated into the page at build time. There is no external image
service.

@qr{url="https://github.com/leaanthony/mpress" size="120" label="M-Press repository"}

@details{title="Source"}
```md
@qr{url="https://github.com/leaanthony/mpress" size="120" label="M-Press repository"}
```
@end
