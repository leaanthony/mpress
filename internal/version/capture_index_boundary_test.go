package version

import (
	"errors"
	"github.com/leaanthony/mpress/internal/projectfs"
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureRejectsPrivateIndexWithoutReplacingPrevious(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	index := filepath.Join(root, "site/index.html")
	writeVersionFile(t, index, "previous usable version")
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}
	writeVersionFile(t, filepath.Join(root, "private/index.html"), "private index content")
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	versionSymlink(t, filepath.Join("..", "private", "index.html"), index)
	if err := Capture(root, "v1", true); !errors.Is(err, projectfs.ErrOutside) {
		t.Fatalf("unsafe index should report its boundary error: %v", err)
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/index.html")); err != nil || string(data) != "previous usable version" {
		t.Fatalf("private index replaced the verified version: %v %q", err, data)
	}
}
