package projectfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinkLexicalRestoresLeafWithoutReplacingDestination(t *testing.T) {
	files, root := openedTestRoot(t)
	backup := filepath.Join(root, "backup.recovery")
	if err := os.WriteFile(backup, []byte("original bytes"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := files.LinkLexical("backup.recovery", "restored.md"); err != nil {
		t.Fatalf("restore link failed: %v", err)
	}
	restored, err := os.Stat(filepath.Join(root, "restored.md"))
	if err != nil {
		t.Fatal(err)
	}
	backupInfo, err := os.Stat(backup)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(restored, backupInfo) {
		t.Fatal("restored leaf is not the original inode")
	}
	if got, err := os.ReadFile(filepath.Join(root, "restored.md")); err != nil || string(got) != "original bytes" {
		t.Fatalf("restored bytes changed: %v %q", err, got)
	}
	if runtime.GOOS != "windows" && restored.Mode().Perm() != 0600 {
		t.Fatalf("restored mode changed: %v", restored.Mode().Perm())
	}

	// A second restore must not replace the leaf that already exists.
	existing := filepath.Join(root, "existing.md")
	if err := os.WriteFile(existing, []byte("author bytes"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := files.LinkLexical("backup.recovery", "existing.md"); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("existing destination was not refused: %v", err)
	}
	if got, err := os.ReadFile(existing); err != nil || string(got) != "author bytes" {
		t.Fatalf("existing destination changed: %v %q", err, got)
	}
	info, err := os.Stat(existing)
	if err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("existing destination mode changed: %v %v", info, err)
	}

	// A directory leaf is likewise never replaced.
	if err := os.Mkdir(filepath.Join(root, "directory"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := files.LinkLexical("backup.recovery", "directory"); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("directory destination was not refused: %v", err)
	}
	if info, err := os.Stat(filepath.Join(root, "directory")); err != nil || !info.IsDir() {
		t.Fatalf("directory destination changed: %v %v", info, err)
	}
}

func TestLinkLexicalRetainsDanglingAliasText(t *testing.T) {
	files, root := openedTestRoot(t)
	if err := os.Symlink("missing-target.md", filepath.Join(root, "alias.recovery")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}

	if err := files.LinkLexical("alias.recovery", "alias.md"); err != nil {
		t.Fatalf("dangling alias restore failed: %v", err)
	}
	info, err := os.Lstat(filepath.Join(root, "alias.md"))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("restored leaf is not a symbolic link: %v %v", info, err)
	}
	if target, err := os.Readlink(filepath.Join(root, "alias.md")); err != nil || target != "missing-target.md" {
		t.Fatalf("alias link text changed: %q %v", target, err)
	}
	if _, err := os.Stat(filepath.Join(root, "missing-target.md")); !os.IsNotExist(err) {
		t.Fatalf("dangling target was created: %v", err)
	}

	// A newly written dangling link at the destination is retained too.
	if err := os.Symlink("author-target.md", filepath.Join(root, "author.md")); err != nil {
		t.Fatal(err)
	}
	if err := files.LinkLexical("alias.recovery", "author.md"); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("existing dangling destination was not refused: %v", err)
	}
	if target, err := os.Readlink(filepath.Join(root, "author.md")); err != nil || target != "author-target.md" {
		t.Fatalf("author link text changed: %q %v", target, err)
	}
}

func TestLinkLexicalRejectsExternalParents(t *testing.T) {
	files, root := openedTestRoot(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("must survive"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := files.LinkLexical("escape/sentinel", "stolen"); !errors.Is(err, ErrOutside) {
		t.Errorf("lexical link accepted an external source parent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "owned"), []byte("owned bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := files.LinkLexical("owned", "escape/stolen"); !errors.Is(err, ErrOutside) {
		t.Errorf("lexical link accepted an external destination parent: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(outside, "sentinel")); err != nil || string(got) != "must survive" {
		t.Fatalf("external sentinel changed: %v %q", err, got)
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 1 {
		t.Fatalf("external directory changed: %v %v", entries, err)
	}
}
