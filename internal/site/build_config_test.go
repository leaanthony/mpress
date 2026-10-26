package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestBuildReportsConfigurationItUsed(t *testing.T) {
	root := t.TempDir()
	source := []byte("site:\n  title: Used by build\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n")
	if err := os.WriteFile(filepath.Join(root, config.Filename), source, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "index.md"), []byte("# Home\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Build(root, BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Site.Title != "Used by build" {
		t.Fatalf("build did not report the configuration it used: %#v", result.Config.Site)
	}
	if result.ConfigRevision != config.SourceRevision(source) {
		t.Fatalf("build reported revision %q, want %q", result.ConfigRevision, config.SourceRevision(source))
	}
	if result.Config.OutputPath(root) != filepath.Join(root, "site") {
		t.Fatalf("reported configuration points at %q", result.Config.OutputPath(root))
	}
}
