package site

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const repeatedComponentsSource = `---
title: Repeated components
---

# Repeated components

:::tabs{sync-key=os alpha="one" zeta="last"}
[One]

Same tab content.

[Two]

Same tab content again.
:::

:::tabs{sync-key=os alpha="one" zeta="last"}
[One]

Same tab content.

[Two]

Same tab content again.
:::

:::carousel{label="Demo"}
First slide.
---
Second slide.
:::

:::carousel{label="Demo"}
First slide.
---
Second slide.
:::

:::table{header=true search=true filter=true}
| Name | Value |
| A | 1 |
| B | 2 |
:::

:::table{header=true search=true filter=true}
| Name | Value |
| A | 1 |
| B | 2 |
:::

:::api-playground{method="GET" path="/health" baseUrl="http://127.0.0.1:1"}
Local fixture only.
:::

:::api-playground{method="GET" path="/health" baseUrl="http://127.0.0.1:1"}
Local fixture only.
:::
`

// Cold and warm builds must emit identical bytes for repeated component
// instances. The warm build reads the cached page, so this also guards the
// serialized identity scope.
func TestBuildRepeatedComponentsAreStableAcrossColdAndWarmBuild(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Repeated components
build:
  contentDir: content
  staticDir: static
  outputDir: site
knowledge:
  enabled: false
search:
  enabled: false
`)
	writeFixture(t, root, "content/index.md", repeatedComponentsSource)
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("cold build: %v", err)
	}
	cold, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}

	// A warm build reuses the parse cache populated by the cold build.
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("warm build: %v", err)
	}
	warm, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cold, warm) {
		t.Fatalf("cold and warm builds differ:\ncold:\n%s\nwarm:\n%s", cold, warm)
	}

	for _, want := range []string{
		`id="mpress-tabs-`,
		`-2-tab-1"`,
		`-2" class="mpress-carousel"`,
		`-2-filter-0"`,
		`id="api-pg-`,
		`-2" data-method="GET"`,
	} {
		if !bytes.Contains(warm, []byte(want)) {
			t.Errorf("warm build output missing repeated-instance marker %q", want)
		}
	}
}
