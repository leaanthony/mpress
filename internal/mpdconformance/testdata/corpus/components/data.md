# Data components

@matrix{highlight="MPress"}
| Capability | MPress | Other |
| --- | --- | --- |
| Single binary | Yes | No |
| Runtime | No | Yes |
@end

@status{service="Build service" state="operational"}
All systems normal
@end
@status{service="Translation queue" state="degraded"}
Jobs may take longer
@end

@calendar{month="2026-08" style="compact" today="2026-08-11"}
Release dates
@end

@changelog
### v2.0.0 (2026-08-12)
#### Added
- Native MPD rendering
- Accessible reader controls
#### Breaking changes
- Removed the legacy component preamble
### v1.4.0 (2026-08-01)
#### Improved
- Faster document compilation
#### Fixed
- Stable fixture output
#### Documentation
- Expanded the authoring reference
@end

@release{version="1.1.0" title="The Conformance Update" date="2026-08-01" type="minor" latest=true}
### What's changed
- A complete conformance corpus.
### Bug Fixes
- Deterministic fixtures.
### Assets
- [Source code (zip)](/downloads/mpress-v1.1.0.zip)
- [Source code (tar.gz)](/downloads/mpress-v1.1.0.tar.gz)
@end

@pricing{cols="2"}
### Starter
$0
[Start](/start/)
- ✓ One project
---
### Team
$15
[Upgrade](/team/)
recommended
- ✓ Shared projects
@end

@testimonials
“The source stays portable.”
Documentation author
---
“The output is static.”
Platform engineer
@end

@qr{url="https://m-press.me" size="120" label="MPress website"}

@tutorial{title="Publish a site"}
### Check
Run `mpress check`.

### Build
Run `mpress build`.
@end
