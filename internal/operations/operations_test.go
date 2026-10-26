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

func TestOpenDeploymentBorrowsOutputInsideProject(t *testing.T) {
	project := t.TempDir()
	writeOperationFixture(t, project, "mpress.yaml", "build:\n  outputDir: linked\n")
	writeOperationFixture(t, project, "site/index.html", "<h1>Home</h1>")
	if err := os.Symlink("site", filepath.Join(project, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cfg, output, files, err := openDeployment(project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := output.Close(); err != nil {
			t.Errorf("close deployment output: %v", err)
		}
		if err := files.Close(); err != nil {
			t.Errorf("close deployment project: %v", err)
		}
	})
	if cfg.Build.OutputDir != "linked" {
		t.Fatalf("unexpected output config %q", cfg.Build.OutputDir)
	}
	if data, err := output.ReadFile("index.html"); err != nil || string(data) != "<h1>Home</h1>" {
		t.Fatalf("internal output alias was not borrowed: %q, %v", data, err)
	}
}

func TestOpenDeploymentRejectsRedirectedInputs(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		project, outside := t.TempDir(), t.TempDir()
		writeOperationFixture(t, outside, "mpress.yaml", "build:\n  outputDir: site\n")
		if err := os.Symlink(filepath.Join(outside, "mpress.yaml"), filepath.Join(project, "mpress.yaml")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, _, _, err := openDeployment(project); err == nil {
			t.Fatal("external configuration alias was accepted")
		}
	})
	t.Run("output", func(t *testing.T) {
		project, outside := t.TempDir(), t.TempDir()
		writeOperationFixture(t, project, "mpress.yaml", "build:\n  outputDir: linked\n")
		writeOperationFixture(t, outside, "index.html", "<h1>outside</h1>")
		if err := os.Symlink(outside, filepath.Join(project, "linked")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if _, _, _, err := openDeployment(project); err == nil {
			t.Fatal("external output alias was accepted")
		}
	})
}
