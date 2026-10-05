package projectfs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFSRejectsExternalExistingAndMissingTargets(t *testing.T) {
	for _, kind := range []string{"parent", "final", "dangling"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			if err := os.WriteFile(filepath.Join(outside, "sentinel.txt"), []byte("must survive"), 0644); err != nil {
				t.Fatal(err)
			}
			files, err := Open(root)
			if err != nil {
				t.Fatal(err)
			}
			defer files.Close()
			target, name := filepath.Join(outside, "sentinel.txt"), "linked.txt"
			if kind == "parent" {
				target, name = outside, "linked/sentinel.txt"
			}
			if kind == "dangling" {
				target = filepath.Join(outside, "missing.txt")
			}
			link := "linked.txt"
			if kind == "parent" {
				link = "linked"
			}
			if err := os.Symlink(target, filepath.Join(root, link)); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			if _, err := files.ReadFile(name); err == nil {
				t.Error("external read accepted")
			}
			if err := files.WriteAtomic(name, []byte("changed")); err == nil {
				t.Error("external write accepted")
			}
			if err := files.WriteCache(name, []byte("changed cache")); err == nil {
				t.Error("external cache write accepted")
			}
			if err := files.Remove(name); err == nil {
				t.Error("external removal accepted")
			}
			if err := files.RemoveAll(link); err == nil {
				t.Error("external tree removal accepted")
			}
			data, err := os.ReadFile(filepath.Join(outside, "sentinel.txt"))
			if err != nil || string(data) != "must survive" {
				t.Error("outside sentinel changed")
			}
			if _, err := os.Stat(filepath.Join(outside, "missing.txt")); !os.IsNotExist(err) {
				t.Error("external missing target created")
			}
		})
	}
}

func TestFSRetainsSafeAliasesAndAtomicReplacement(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		t.Run(map[bool]string{false: "relative", true: "absolute"}[absolute], func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "real"), 0755); err != nil {
				t.Fatal(err)
			}
			target := "real"
			if absolute {
				target = filepath.Join(root, target)
			}
			if err := os.Symlink(target, filepath.Join(root, "alias")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			files, err := Open(root)
			if err != nil {
				t.Fatal(err)
			}
			defer files.Close()
			if err := files.WriteAtomic("alias/nested/file.txt", []byte("original")); err != nil {
				t.Fatal(err)
			}
			data, err := files.ReadFile("alias/nested/file.txt")
			if err != nil || string(data) != "original" {
				t.Fatalf("alias read: %v", err)
			}
			leafTarget := "nested/file.txt"
			if absolute {
				leafTarget = filepath.Join(root, "real", leafTarget)
			}
			if err := os.Symlink(leafTarget, filepath.Join(root, "real/leaf.txt")); err != nil {
				t.Fatal(err)
			}
			if err := files.WriteAtomic("real/leaf.txt", []byte("replacement")); err != nil {
				t.Fatal(err)
			}
			data, err = files.ReadFile("real/nested/file.txt")
			if err != nil || string(data) != "original" {
				t.Error("atomic replacement overwrote final-link target")
			}
			data, err = files.ReadFile("real/leaf.txt")
			if err != nil || string(data) != "replacement" {
				t.Error("replacement missing")
			}
			entries, err := files.ReadDir("real")
			if err != nil || len(entries) != 2 {
				t.Error("directory listing failed or temporary file leaked")
			}
			if err := files.RemoveAll("alias/nested"); err != nil {
				t.Fatal(err)
			}
			if _, err := files.Stat("real/nested"); !os.IsNotExist(err) {
				t.Error("safe alias cleanup failed")
			}
			if err := files.RemoveAll("."); err == nil {
				t.Error("project root removal accepted")
			}
		})
	}
}

func TestFSRechecksPathsAfterValidation(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "parent"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "sentinel.txt"), []byte("must survive"), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	if _, err := files.Relative("parent/sentinel.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "parent"), filepath.Join(root, "original-parent")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "parent")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := files.ReadFile("parent/sentinel.txt"); !errors.Is(err, ErrOutside) {
		t.Errorf("changed parent read: %v", err)
	}
	if err := files.WriteAtomic("parent/new.txt", []byte("changed")); !errors.Is(err, ErrOutside) {
		t.Errorf("changed parent write: %v", err)
	}
	if err := files.RemoveAll("parent"); !errors.Is(err, ErrOutside) {
		t.Errorf("changed parent removal: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outside, "sentinel.txt"))
	if err != nil || string(data) != "must survive" {
		t.Error("outside sentinel changed")
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); !os.IsNotExist(err) {
		t.Error("outside file created")
	}
}

func TestFSRejectsNonRegularReads(t *testing.T) {
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("native FIFO fixture unavailable")
	}
	root := t.TempDir()
	if err := exec.Command(mkfifo, filepath.Join(root, "fifo")).Run(); err != nil {
		t.Skipf("FIFO fixture unavailable: %v", err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	if _, err := files.ReadFile("fifo"); err == nil {
		t.Error("FIFO read accepted")
	}
	if _, err := files.Open("fifo"); err == nil {
		t.Error("FIFO serving accepted")
	}
}
