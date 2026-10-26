package projectfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// CheckRoot follows the pinned directory across ordinary content changes and
// reports moves, replacements and retargeted aliases.
func TestCheckRootTracksPinnedDirectory(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "project")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()

	if err := files.CheckRoot(); err != nil {
		t.Fatalf("fresh root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.md"), []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); err != nil {
		t.Fatalf("ordinary file change: %v", err)
	}

	moved := filepath.Join(parent, "preserved")
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); !errors.Is(err, ErrRootChanged) {
		t.Fatalf("moved root: %v", err)
	}

	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); !errors.Is(err, ErrRootChanged) {
		t.Fatalf("replacement root: %v", err)
	}

	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); !errors.Is(err, ErrRootChanged) {
		t.Fatalf("missing root: %v", err)
	}

	if err := os.Rename(moved, root); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); err != nil {
		t.Fatalf("restored root: %v", err)
	}
}

func TestCheckRootFollowsStableAliasAndRejectsRetarget(t *testing.T) {
	parent := t.TempDir()
	first := filepath.Join(parent, "first")
	second := filepath.Join(parent, "second")
	for _, path := range []string{first, second} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(parent, "project")
	if err := os.Symlink(first, alias); err != nil {
		if probeErr := os.Symlink(first, filepath.Join(parent, "probe")); probeErr != nil {
			t.Skipf("symlinks unavailable: %v", probeErr)
		}
		t.Fatal(err)
	}
	files, err := Open(alias)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	if err := files.CheckRoot(); err != nil {
		t.Fatalf("stable alias: %v", err)
	}

	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(second, alias); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); !errors.Is(err, ErrRootChanged) {
		t.Fatalf("retargeted alias: %v", err)
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(first, alias); err != nil {
		t.Fatal(err)
	}
	if err := files.CheckRoot(); err != nil {
		t.Fatalf("restored alias: %v", err)
	}
}
