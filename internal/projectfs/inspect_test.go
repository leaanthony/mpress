package projectfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLeafInspectionConfinesParentsAndPreservesLinkText(t *testing.T) {
	files, root := openedTestRoot(t)
	outside := t.TempDir()
	target := filepath.Join(outside, "sentinel")
	if err := os.WriteFile(target, []byte("outside content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "leaf")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	info, err := files.Lstat("leaf")
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("final symlink was followed: %v %v", info, err)
	}
	if text, err := files.ReadLink("leaf"); err != nil || text != target {
		t.Fatalf("link text changed: %q %v", text, err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"escape/sentinel", target} {
		if _, err := files.Lstat(name); !errors.Is(err, ErrOutside) {
			t.Errorf("inspection accepted external parent %s: %v", name, err)
		}
		if _, err := files.ReadLink(name); !errors.Is(err, ErrOutside) {
			t.Errorf("link read accepted external parent %s: %v", name, err)
		}
	}
}
