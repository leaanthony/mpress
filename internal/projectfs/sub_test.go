package projectfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSubConfinesFilesAndPinsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "snapshot"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "snapshot", "index.html"), []byte("snapshot"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret"), []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	snapshot, err := files.Sub("snapshot")
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if _, err := snapshot.ReadFile("../secret"); !errors.Is(err, ErrOutside) {
		t.Errorf("subtree traversal: %v", err)
	}
	// A renamed directory and same-name replacement must not change the pinned
	// handle. Native Windows may deny renaming an open directory.
	if err := os.Rename(filepath.Join(root, "snapshot"), filepath.Join(root, "original")); err != nil {
		t.Skipf("native open-directory rename unavailable: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "snapshot"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "snapshot", "index.html"), []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := snapshot.ReadFile("index.html")
	if err != nil || string(data) != "snapshot" {
		t.Errorf("subtree was redirected: %q, %v", data, err)
	}
}
