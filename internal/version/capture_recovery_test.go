package version

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestCapturePromotionFailurePreservesPreviousSnapshot(t *testing.T) {
	for _, kind := range []string{"old-move", "promotion", "rollback"} {
		t.Run(kind, func(t *testing.T) {
			root := versionBoundaryFixture(t, ".mpress/versions")
			writeVersionFile(t, filepath.Join(root, "site/index.html"), "previous verified snapshot")
			if err := Capture(root, "v1", false); err != nil {
				t.Fatal(err)
			}
			store, _, err := openVersionStore(root)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			lock, err := store.Lock(storeLockFile)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			staged, err := store.MkdirTemp(".", ".capture-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer store.RemoveAll(staged)
			writeVersionFile(t, filepath.Join(staged, "index.html"), "replacement")
			versionManifest(t, staged, "v1", "index.html", "replacement")
			calls := 0
			injected := errors.New("injected filesystem move failure")
			rename := func(old, new string) error {
				calls++
				if kind == "old-move" && calls == 1 || kind == "promotion" && calls == 2 || kind == "rollback" && calls >= 2 {
					return injected
				}
				return store.Rename(old, new)
			}
			if err := promoteSnapshot(store, staged, "v1", true, rename); !errors.Is(err, injected) {
				t.Fatalf("expected injected failure: %v", err)
			}
			previous := "v1"
			entries, err := store.ReadDir(".")
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".previous-") {
					if kind != "rollback" {
						t.Error("recoverable failure left a backup")
					}
					previous = filepath.Join(entry.Name(), "snapshot")
				}
			}
			snapshot, err := store.Sub(previous)
			if err != nil {
				t.Fatalf("previous snapshot lost: %v", err)
			}
			defer snapshot.Close()
			if err := verifyPinnedSnapshot(snapshot, "v1"); err != nil {
				t.Fatal(err)
			}
			data, err := snapshot.ReadFile("index.html")
			if err != nil || string(data) != "previous verified snapshot" {
				t.Error("failure changed the previous snapshot")
			}
		})
	}
}

func TestCaptureCopyAndManifestFailuresPreservePrevious(t *testing.T) {
	for _, kind := range []string{"external-file", "manifest-directory"} {
		t.Run(kind, func(t *testing.T) {
			root := versionBoundaryFixture(t, ".mpress/versions")
			writeVersionFile(t, filepath.Join(root, "site/index.html"), "previous")
			if err := Capture(root, "v1", false); err != nil {
				t.Fatal(err)
			}
			writeVersionFile(t, filepath.Join(root, "site/index.html"), "replacement")
			if kind == "external-file" {
				outside := t.TempDir()
				writeVersionFile(t, filepath.Join(outside, "secret"), "secret")
				versionSymlink(t, filepath.Join(outside, "secret"), filepath.Join(root, "site/linked"))
			} else {
				writeVersionFile(t, filepath.Join(root, "site/mpress-version.json/nested"), "directory at manifest path")
			}
			if err := Capture(root, "v1", true); err == nil {
				t.Error("expected capture failure")
			}
			if err := Verify(root, "v1"); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/index.html"))
			if err != nil || string(data) != "previous" {
				t.Error("failed capture changed the previous snapshot")
			}
			entries, err := os.ReadDir(filepath.Join(root, ".mpress/versions"))
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".capture-") || strings.HasPrefix(entry.Name(), ".previous-") {
					t.Error("failed capture leaked its workspace")
				}
			}
		})
	}
}

func TestCaptureIndexesNestedManifestNamedAsset(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "ordinary output")
	writeVersionFile(t, filepath.Join(root, "site/data/mpress-version.json"), "ordinary asset")
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("nested ordinary asset missing from manifest: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(root, ".mpress/versions/v1"))
		if err != nil || info.Mode().Perm() != 0755 {
			t.Error("snapshot directory lost its ordinary read permissions")
		}
	}
}

func TestCaptureConcurrentWriters(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "no-force", true: "force"}[force], func(t *testing.T) {
			root := versionBoundaryFixture(t, ".mpress/versions")
			writeVersionFile(t, filepath.Join(root, "site/index.html"), strings.Repeat("snapshot", 8192))
			start := make(chan struct{})
			results := make(chan error, 6)
			var group sync.WaitGroup
			for range 6 {
				group.Go(func() { <-start; results <- Capture(root, "v1", force) })
			}
			close(start)
			group.Wait()
			close(results)
			successes := 0
			for err := range results {
				if err == nil {
					successes++
				} else if force || !strings.Contains(err.Error(), "already exists") {
					t.Errorf("unexpected capture failure: %v", err)
				}
			}
			want := 1
			if force {
				want = 6
			}
			if successes != want {
				t.Errorf("successful captures: %d, want %d", successes, want)
			}
			if err := Verify(root, "v1"); err != nil {
				t.Fatalf("concurrent captures produced an invalid snapshot: %v", err)
			}
		})
	}
}

func TestCaptureReadOnlyStorePreservesPrevious(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "previous")
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, ".mpress/versions")
	if err := os.Chmod(base, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(base, 0755)
	if err := os.WriteFile(filepath.Join(base, "permission-probe"), nil, 0600); err == nil {
		os.Remove(filepath.Join(base, "permission-probe"))
		t.Skip("native directory permissions do not prevent writes")
	}
	if err := Capture(root, "v1", true); err == nil {
		t.Error("read-only snapshot replacement accepted")
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("read-only verification failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(base, "v1/index.html"))
	if err != nil || string(data) != "previous" {
		t.Error("read-only failure changed the previous snapshot")
	}
}
