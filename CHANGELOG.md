# Changelog

## Unreleased

### Fixed

- Preserve literal class and ID tokens from inline page scripts during production CSS purging, including class lists and template strings. Pages with runtime animation classes can retain their styles while removing unused CSS.
- Mark frontmatter hero images as eager, high-priority resources with asynchronous decoding for both light and dark themes.

### Previously merged

- Closed utility popovers no longer intercept mobile touch scrolling or keyboard focus. Thanks to [@dogukanoklu](https://github.com/dogukanoklu), M-Press's first community contributor, for [PR #6](https://github.com/leaanthony/mpress/pull/6).
