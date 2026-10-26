package projectfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func openedTestRoot(t *testing.T) (*FS, string) {
	t.Helper()
	root := t.TempDir()
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	return files, root
}

func assertNoStaging(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".mpress-write-") {
			t.Fatalf("owned staging file was not removed: %s", entry.Name())
		}
	}
}

func TestPublishNewRefusesExistingRegularFile(t *testing.T) {
	files, root := openedTestRoot(t)
	existing := filepath.Join(root, "existing.mpd")
	if err := os.WriteFile(existing, []byte("existing bytes"), 0640); err != nil {
		t.Fatal(err)
	}
	unrelated := filepath.Join(root, "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("unrelated bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	published, err := files.WriteNewAtomicMode("existing.mpd", []byte("replacement"), 0600)
	if err == nil || published || !errors.Is(err, fs.ErrExist) {
		t.Fatalf("existing target was not refused: published=%v err=%v", published, err)
	}
	if got, err := os.ReadFile(existing); err != nil || string(got) != "existing bytes" {
		t.Fatalf("existing target changed: %v %q", err, got)
	}
	if got, err := os.ReadFile(unrelated); err != nil || string(got) != "unrelated bytes" {
		t.Fatalf("unrelated file changed: %v %q", err, got)
	}
	assertNoStaging(t, root)
}

func TestPublishNewRefusesFinalSymlinkAndDanglingLink(t *testing.T) {
	files, root := openedTestRoot(t)
	if err := os.WriteFile(filepath.Join(root, "real.txt"), []byte("real bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	if err := os.Symlink("missing.txt", filepath.Join(root, "dangling.txt")); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"link.txt", "dangling.txt"} {
		published, err := files.WriteNewAtomicMode(name, []byte("new bytes"), 0644)
		if err == nil || published {
			t.Fatalf("%s was not refused: published=%v err=%v", name, published, err)
		}
		info, statErr := os.Lstat(filepath.Join(root, name))
		if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s symlink changed: %v %v", name, info, statErr)
		}
	}
	if target, err := os.Readlink(filepath.Join(root, "link.txt")); err != nil || target != "real.txt" {
		t.Fatalf("symlink target changed: %q %v", target, err)
	}
	if target, err := os.Readlink(filepath.Join(root, "dangling.txt")); err != nil || target != "missing.txt" {
		t.Fatalf("dangling link target changed: %q %v", target, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "real.txt")); err != nil || string(got) != "real bytes" {
		t.Fatalf("symlink target content changed: %v %q", err, got)
	}
	if _, err := os.Stat(filepath.Join(root, "missing.txt")); !os.IsNotExist(err) {
		t.Fatalf("dangling link target was created: %v", err)
	}
	assertNoStaging(t, root)
}

func TestPublishNewRejectsExternalParents(t *testing.T) {
	files, root := openedTestRoot(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	for _, name := range []string{"escape/new.txt", "../outside.txt"} {
		if published, err := files.WriteNewAtomicMode(name, []byte("escape"), 0644); !errors.Is(err, ErrOutside) || published {
			t.Fatalf("%s was not rejected as outside: published=%v err=%v", name, published, err)
		}
	}
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatalf("outside directory changed: %v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "outside.txt")); !os.IsNotExist(err) {
		t.Fatalf("sibling of the project root was written: %v", err)
	}
}

func TestPublishNewWritesCompleteContentAndMode(t *testing.T) {
	files, root := openedTestRoot(t)
	data := []byte("# Complete\n\nStaged before publication.\n")
	published, err := files.WriteNewAtomicMode("nested/page.mpd", data, 0600)
	if err != nil || !published {
		t.Fatalf("new publication failed: published=%v err=%v", published, err)
	}
	got, err := os.ReadFile(filepath.Join(root, "nested", "page.mpd"))
	if err != nil || string(got) != string(data) {
		t.Fatalf("published content changed: %v %q", err, got)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(root, "nested", "page.mpd"))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatalf("published mode changed: %v %v", info, err)
		}
	}
	assertNoStaging(t, filepath.Join(root, "nested"))
}

func TestPublishNewCleansOwnedStagingAfterWriteFailure(t *testing.T) {
	files, root := openedTestRoot(t)
	writeFailure := errors.New("write failed")
	published, err := files.writeAtomic("partial.mpd", func(*os.File) error { return writeFailure }, 0644, true, false)
	if !errors.Is(err, writeFailure) || published {
		t.Fatalf("write failure was not reported: published=%v err=%v", published, err)
	}
	if _, err := os.Stat(filepath.Join(root, "partial.mpd")); !os.IsNotExist(err) {
		t.Fatalf("partial target became visible: %v", err)
	}
	assertNoStaging(t, root)
}

func TestPublishNewRetainsSymlinkParentAlias(t *testing.T) {
	files, root := openedTestRoot(t)
	if err := os.Mkdir(filepath.Join(root, "real"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(root, "alias")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	published, err := files.WriteNewAtomicMode("alias/page.mpd", []byte("aliased"), 0644)
	if err != nil || !published {
		t.Fatalf("internal alias publication failed: published=%v err=%v", published, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "real", "page.mpd")); err != nil || string(got) != "aliased" {
		t.Fatalf("aliased publication missing: %v %q", err, got)
	}
}

func TestPublishNewRejectsCaseInsensitiveCollision(t *testing.T) {
	files, root := openedTestRoot(t)
	if err := os.WriteFile(filepath.Join(root, "Page.mpd"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "page.mpd")); err == nil {
		published, err := files.WriteNewAtomicMode("page.mpd", []byte("new"), 0644)
		if err == nil || published {
			t.Fatalf("case-insensitive collision was not refused: published=%v err=%v", published, err)
		}
		if got, readErr := os.ReadFile(filepath.Join(root, "Page.mpd")); readErr != nil || string(got) != "existing" {
			t.Fatalf("existing case-variant target changed: %v %q", readErr, got)
		}
		assertNoStaging(t, root)
		return
	}
	t.Skip("filesystem is case-sensitive")
}
