package projectfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingStream struct{ read bool }

func (r *failingStream) Read(buffer []byte) (int, error) {
	if r.read {
		return 0, errors.New("injected stream read failure")
	}
	r.read = true
	return copy(buffer, "partial replacement"), nil
}

func TestWriteFromPreservesDestinationOnReadFailure(t *testing.T) {
	root := t.TempDir()
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	if err := files.WriteAtomic("file", []byte("previous")); err != nil {
		t.Fatal(err)
	}
	if err := files.WriteFrom("file", &failingStream{}); err == nil {
		t.Error("stream failure ignored")
	}
	data, err := files.ReadFile("file")
	if err != nil || string(data) != "previous" {
		t.Error("failed stream changed the destination")
	}
	entries, err := files.ReadDir(".")
	if err != nil || len(entries) != 1 {
		t.Error("failed stream left a temporary file")
	}
	if err := files.WriteFrom("nested/file", strings.NewReader("streamed")); err != nil {
		t.Fatal(err)
	}
	file, err := files.Open("nested/file")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err = io.ReadAll(file)
	if err != nil || string(data) != "streamed" {
		t.Error("ordinary stream copy failed")
	}
}

func TestRenameRejectsExternalPathsAndRetainsAliasLeaf(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("must survive"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "external")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	files, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	if err := files.WriteAtomic("incoming", []byte("replacement")); err != nil {
		t.Fatal(err)
	}
	if err := files.Rename("external/sentinel", "stolen"); err == nil {
		t.Error("external rename source accepted")
	}
	if err := files.Rename("incoming", "external/sentinel"); err == nil {
		t.Error("external rename target accepted")
	}
	if err := files.WriteAtomic("original", []byte("must survive")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("original", filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}
	if err := files.Rename("incoming", "alias"); err != nil {
		t.Fatal(err)
	}
	data, err := files.ReadFile("original")
	if err != nil || string(data) != "must survive" {
		t.Error("rename overwrote the alias target")
	}
	data, err = os.ReadFile(filepath.Join(outside, "sentinel"))
	if err != nil || string(data) != "must survive" {
		t.Error("rename changed the external sentinel")
	}
}
