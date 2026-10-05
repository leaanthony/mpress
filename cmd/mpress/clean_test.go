package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestCleanPreservesProtectedProjectData(t *testing.T) {
	for _, output := range []string{".", "content", "content/generated", "static", ".git", config.Filename, ".mpress", ".mpress/translations", ".mpress/unknown-state"} {
		t.Run(output, func(t *testing.T) {
			root := t.TempDir()
			cfg := config.Default()
			cfg.Build.OutputDir = output
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			sentinels := []string{"content/keep.md", "static/keep.txt", ".git/keep", ".mpress/translations/keep", ".mpress/unknown-state/keep"}
			for _, path := range sentinels {
				writeCleanSentinel(t, filepath.Join(root, path))
			}
			t.Chdir(root)
			if err := clean(); err == nil || !strings.Contains(err.Error(), "unsafe") {
				t.Errorf("expected unsafe clean rejection, got %v", err)
			}
			for _, path := range sentinels {
				assertCleanSentinel(t, filepath.Join(root, path))
			}
			if _, err := os.Stat(filepath.Join(root, config.Filename)); err != nil {
				t.Errorf("configuration lost: %v", err)
			}
		})
	}
}

func TestCleanRejectsExternalSymlinkParent(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	cfg := config.Default()
	cfg.Build.OutputDir = "linked/site"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(outside, "site/keep")
	writeCleanSentinel(t, path)
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Chdir(root)
	if err := clean(); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Errorf("expected unsafe clean rejection, got %v", err)
	}
	assertCleanSentinel(t, path)
}

func TestCleanRemovesOnlyGeneratedOutput(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(root, "content/keep.md")
	writeCleanSentinel(t, protected)
	writeCleanSentinel(t, filepath.Join(root, "site/obsolete"))
	t.Chdir(root)
	for n := 0; n < 2; n++ {
		if err := clean(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, "site")); !os.IsNotExist(err) {
			t.Errorf("output survived clean: %v", err)
		}
		assertCleanSentinel(t, protected)
	}
}

func writeCleanSentinel(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("must survive"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertCleanSentinel(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "must survive" {
		t.Errorf("sentinel changed: %q, %v", data, err)
	}
}
