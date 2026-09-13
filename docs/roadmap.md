---
title: After the MVP
description: The ordered work beyond the verified M-Press 0.1 release contract.
order: 22
---

The M-Press 0.1 contract is complete. Component parity is in place for the
retained component suite, the real Wails corpus imports and builds cleanly, and
the public and private release gates pass. API chat remains deliberately
deferred. The work below hardens the product on the path to 1.0; it is not a
blocker for the first MVP release.

## 1. Wails rollout

The generator-side migration gate is complete: 514 Markdown sources import as
524 content pages and 673 files, strict build succeeds, and the generated site has
no broken links or assets. The remaining work is the project rollout.

- Review the explicit migration findings with the Wails maintainers.
- Compare the generated route manifest with the routes deployed in production.
- Add any rollout-only failures to the private conformance corpus.
- Run a staged deployment before changing the public documentation origin.

## 2. Translation hardening

The generator now provides structural machine translation,
OpenRouter and OpenAI-compatible providers, glossary enforcement, stable page
keys, source fingerprints, manual-edit detection, navigation translation, and a
secured development workflow. The remaining work broadens proof and review.

- Run the Wails multilingual corpus through private provider and browser gates.
- Add a pseudo-language test mode to expose untranslated interface strings and
  layouts that fail with longer text.
- Add XLIFF import and export for professional translation tools.
- Measure translation quality with human MQM review. Treat automated scores as
  signals, not release gates.

Optional external services can provide shared translation memory, assignments, budgets, pull
requests, vendor workflows, and organisation audit history.

## 3. Versioning hardening

Version capture, checksums, listing, verification, and mounting exist. They now
need deployment-grade subpath testing.

- Define stable ordering, display labels, aliases such as `latest`, and behaviour
  when the current version changes.
- Test multilingual version snapshots and non-root `baseURL` deployments.
- Make capture atomic and verify that interrupted captures never replace a valid
  artifact.

The local immutable artifact format stays portable. Managed storage, preview
URLs, retention, promotion, and rollback can use optional external services.

## 4. Development and theme polish

- Add import planning, rename and move operations, asset handling, undo, and a
  recoverable project reset to the authoring protocol. Build state, the
  development bar, configuration forms, onboarding, checks, live reload, and
  direct Cloudflare deployment are in place.
- Add build-time syntax highlighting with no browser runtime dependency.
- Add copy controls to ordinary fenced code blocks, not only terminals.
- Add heading permalink controls and make deep links obvious on focus or hover.
- Add structured data for specialised reference and blog pages.
- Improve developer errors with source lines, suggestions, and a preview-server
  error page that does not destroy the last successful build.
- Finish responsive, print, reduced-motion, high-contrast, and long-content
  testing for the full component catalogue.

## 5. Release engineering

- Produce signed Linux, macOS, and Windows binaries from tagged releases.
- Publish checksums and a documented install and upgrade path.
- Stabilise the configuration schema, CLI exit codes, JSON diagnostics, and the
  compatibility policy before calling the format `1.0`.
- Add private browser baselines, accessibility checks, migration fixtures, and
  adversarial parser cases to the release pipeline.
- Establish performance budgets for cold build, rebuild, generated page weight,
  and search-index size.
