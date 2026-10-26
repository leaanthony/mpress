package version

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// runWithTransientLabelOwner holds the store's writer lock while making the
// label path a transient external alias, then releases ownership after
// restoring the label. A writer must validate that mutable label path only
// after it owns the lock; returning early means validation raced the owner.
func runWithTransientLabelOwner(t *testing.T, root, label, name string, operation func() error) {
	t.Helper()
	store, _, err := openVersionStore(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	owner, err := store.Lock(storeLockFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = owner.Close() }()

	versions := filepath.Join(root, ".mpress", "versions")
	backup := filepath.Join(root, "owner-backup-"+label)
	if err := os.Rename(filepath.Join(versions, label), backup); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	versionSymlink(t, outside, filepath.Join(versions, label))

	done := make(chan error, 1)
	go func() { done <- operation() }()
	select {
	case err := <-done:
		t.Fatalf("%s returned before the store owner released the lock: %v", name, err)
	case <-time.After(200 * time.Millisecond):
	}

	if err := os.Remove(filepath.Join(versions, label)); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, filepath.Join(versions, label)); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s after ownership release: %v", name, err)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("%s did not finish after ownership release", name)
	}
}

func TestCaptureWaitsForStoreOwnerBeforeLabelCheck(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "output")
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}

	runWithTransientLabelOwner(t, root, "v1", "capture", func() error {
		return Capture(root, "v1", true)
	})
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("capture did not replace the restored snapshot: %v", err)
	}
}

func TestRemoveWaitsForStoreOwnerBeforeLabelCheck(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "output")
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}

	runWithTransientLabelOwner(t, root, "v1", "remove", func() error {
		return Remove(root, "v1")
	})
	if _, err := os.Stat(filepath.Join(root, ".mpress", "versions", "v1")); !os.IsNotExist(err) {
		t.Errorf("remove did not delete the restored snapshot: %v", err)
	}
}
