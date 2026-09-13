# Code components

@tabs
[Go]
Run `go test ./...`.

[Shell]
Run `mpress check`.
@end

@terminal{title="Build" frame="macos" prompt="$" comment="#"}
# Validate the site before publishing.
$ # Comments are visible but are not copied.
$ mpress check
No issues found.
@end

@api{method="POST" path="/v1/builds"}
Creates a build.

| Field | Type |
| --- | --- |
| `ref` | string |
@end

@steps
### Write
Add a Markdown file.

### Build
Run the compiler.

### Configure
Set the project options.

### Add navigation
Arrange the pages.

### Add components
Enrich the documentation.

### Check links
Validate every destination.

### Test themes
Check light and dark modes.

### Test mobile
Use a narrow viewport.

### Inspect output
Review the generated HTML.

### Package
Create the deployment archive.

### Publish
Upload the static output.

### Verify deployment
Open the production site.
@end

@diff{title="mpress.yaml" mode="inline"}
search: false
---
search: true
@end

@filetree
docs/
  index.md  Home page
  guide/
    installation.md  Install MPress
    deployment.md  Publish the site
  reference/
    configuration.md  Configuration keys
    components.md  Component catalogue
  assets/
    logo.svg  Site mark
    screenshots/
      home.png  Landing page
  blog/
    index.md  Blog landing page
mpress.yaml  Configuration
@end

@explained
```go
func main() { // (1)
    site.Build() // (2)
}
```

(1) Start with the entry point.
(2) Build the static site.
@end
