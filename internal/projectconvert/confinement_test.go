package projectconvert

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

func TestConversionRetainsExplicitDirectoryScopeAndPermissions(t *testing.T) {
	project, selected := t.TempDir(), t.TempDir()
	cfg := config.Default()
	source := filepath.Join(selected, "page.md")
	if err := os.WriteFile(source, []byte("# Selected directory\n\nKeep this document.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := Run(project, &cfg, selected, "mpd")
	if err != nil || result.Count != 1 || result.Directory != selected {
		t.Fatalf("explicit directory failed: %v %+v", err, result)
	}
	info, err := os.Stat(filepath.Join(selected, "page.mpd"))
	if err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("converted mode changed: %v %v", info, err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("successful conversion retained the selected source: %v", err)
	}
	if entries, err := os.ReadDir(project); err != nil || len(entries) != 0 {
		t.Fatalf("outside selection changed the project: %v %v", entries, err)
	}
}

func TestBorrowedConversionRootRetainsOwnership(t *testing.T) {
	project := t.TempDir()
	cfg := config.Default()
	if err := os.MkdirAll(cfg.ContentPath(project), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.ContentPath(project), "page.md"), []byte("# Hello\n\nKeep this document.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := projectfs.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	result, err := RunRoot(project, files, &cfg, "", "mpd")
	if err != nil || result.Count != 1 {
		t.Fatalf("borrowed conversion failed: %v %+v", err, result)
	}
	if _, err := files.Stat("content/page.mpd"); err != nil {
		t.Fatalf("conversion closed the caller's root: %v", err)
	}
}

func TestConversionUpdatesWritingGuideConfiguration(t *testing.T) {
	project := t.TempDir()
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	if err := os.MkdirAll(cfg.ContentPath(project), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, cfg.Translation.StyleGuide), []byte("# Writing guide\n\nUse clear sentences.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(project, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(project, &cfg, "", "mpd"); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Load(project)
	if err != nil || cfg.Translation.StyleGuide != "content/guide.mpd" || saved.Translation.StyleGuide != cfg.Translation.StyleGuide {
		t.Fatalf("converted guide reference was not persisted: %v %q %q", err, cfg.Translation.StyleGuide, saved.Translation.StyleGuide)
	}
}
