package projectfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRelativeReadsRetainBoundariesAndRegularFileGuard(t *testing.T) {
	files, root := openedTestRoot(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "page.md"), []byte("page"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := files.ReadFile("docs/page.md")
	if err != nil || string(data) != "page" {
		t.Fatalf("relative read: %q, %v", data, err)
	}
	directory, err := files.Open("docs")
	if err != nil {
		t.Fatalf("directory open through the relative fast path: %v", err)
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := files.ReadFile("docs"); err == nil {
		t.Fatal("directory passed the regular-file guard")
	}
	if _, err := files.ReadFile("../outside/secret.md"); !errors.Is(err, ErrOutside) {
		t.Fatalf("relative traversal: %v", err)
	}
	if _, err := files.ReadFile(filepath.Join(outside, "secret.md")); !errors.Is(err, ErrOutside) {
		t.Fatalf("absolute outside read: %v", err)
	}
}

func TestRelativeReadsKeepSafeAliasesAndRejectExternalOnes(t *testing.T) {
	files, root := openedTestRoot(t)
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "page.md"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"relative.md": "docs/page.md",
		"absolute.md": filepath.Join(root, "docs", "page.md"),
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(root, "escape.md")); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"relative.md", "absolute.md"} {
		data, err := files.ReadFile(name)
		if err != nil || string(data) != "inside" {
			t.Fatalf("safe alias %s: %q, %v", name, data, err)
		}
		file, err := files.Open(name)
		if err != nil {
			t.Fatalf("Open safe alias %s: %v", name, err)
		}
		data, err = io.ReadAll(file)
		_ = file.Close()
		if err != nil || string(data) != "inside" {
			t.Fatalf("Open safe alias %s: %q, %v", name, data, err)
		}
	}
	if _, err := files.ReadFile("escape.md"); !errors.Is(err, ErrOutside) {
		t.Fatalf("external alias read: %v", err)
	}
}
