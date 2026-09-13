package operations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckReportsOrderedPerformanceTimeline(t *testing.T) {
	project := t.TempDir()
	writeOperationFixture(t, project, "mpress.yaml", `site:
  title: Timed checks
  defaultLanguage: en
  languages: [en]
build:
  contentDir: content
  staticDir: static
  outputDir: site
search:
  enabled: true
`)
	writeOperationFixture(t, project, "content/index.md", "---\ntitle: Home\n---\n\n# Home\n\n[Guide](/guide/)\n")
	writeOperationFixture(t, project, "content/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n")
	if err := os.MkdirAll(filepath.Join(project, "static"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := Check(project, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"discover", "parse", "prepare", "render", "optimize", "finalize", "links"}
	if len(result.Performance.Steps) != len(want) {
		t.Fatalf("expected %d performance steps, got %#v", len(want), result.Performance.Steps)
	}
	for index, name := range want {
		step := result.Performance.Steps[index]
		if step.Name != name || step.Status != "passed" || step.OffsetMS < 0 || step.DurationMS < 0 {
			t.Fatalf("unexpected performance step %d: %#v", index, step)
		}
		if index > 0 && step.OffsetMS < result.Performance.Steps[index-1].OffsetMS {
			t.Fatalf("performance timeline is out of order: %#v", result.Performance.Steps)
		}
	}
	if got := result.Performance.Steps[3].Label; got != "Render pages, search, and link index" {
		t.Fatalf("render step should explain incremental indexing, got %q", got)
	}
	if got := result.Performance.Steps[6].Label; got != "Resolve links and assets" {
		t.Fatalf("link step should only resolve the completed index, got %q", got)
	}
	if result.Performance.StartedAt.IsZero() || result.Performance.DurationMS <= 0 {
		t.Fatalf("performance summary is incomplete: %#v", result.Performance)
	}
}

func writeOperationFixture(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
