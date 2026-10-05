package projectfs

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestMkdirTempAllocatesDistinctPrivateDirectoriesThroughInternalAlias(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "state"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "state"), filepath.Join(root, "alias")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	const count = 24
	names, failures := make(chan string, count), make(chan error, count)
	var workers sync.WaitGroup
	for range count {
		workers.Go(func() {
			name, err := files.MkdirTemp("alias", "export-")
			if err != nil {
				failures <- err
				return
			}
			names <- name
		})
	}
	workers.Wait()
	close(names)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	seen := map[string]bool{}
	for name := range names {
		if seen[name] || filepath.Dir(name) != filepath.Join(root, "state") || !strings.HasPrefix(filepath.Base(name), "export-") {
			t.Errorf("duplicate or unconfined workspace: %s", name)
		}
		seen[name] = true
		info, err := files.Stat(name)
		if err != nil || !info.IsDir() {
			t.Errorf("workspace unavailable: %v", err)
			continue
		}
		if os.PathSeparator != '\\' && info.Mode().Perm() != 0700 {
			t.Error("workspace is not private")
		}
		if err := files.RemoveAll(name); err != nil {
			t.Error(err)
		}
	}
	if len(seen) != count {
		t.Errorf("allocated %d distinct directories, want %d", len(seen), count)
	}
	entries, err := files.ReadDir("state")
	if err != nil || len(entries) != 0 {
		t.Error("temporary workspace cleanup left files behind")
	}
}

func TestMkdirTempRejectsExternalParentsAndInvalidPrefixes(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	if _, err := files.MkdirTemp("linked", "export-"); err == nil {
		t.Error("external temporary directory accepted")
	}
	for _, prefix := range []string{"../escape-", `..\escape-`, "/absolute-"} {
		if _, err := files.MkdirTemp("missing", prefix); err == nil {
			t.Error("invalid temporary prefix accepted")
		}
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Error("external directory changed")
	}
	if _, err := files.Stat("missing"); !os.IsNotExist(err) {
		t.Error("invalid prefix created its parent before rejection")
	}
}
