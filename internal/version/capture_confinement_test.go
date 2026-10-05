package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCapturePreservesTmpNamedSnapshot(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "old version")
	if err := Capture(root, "v1.tmp", false); err != nil {
		t.Fatal(err)
	}
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "new version")
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}
	if err := Verify(root, "v1.tmp"); err != nil {
		t.Fatalf("capture destroyed the existing tmp-named snapshot: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1.tmp/index.html"))
	if err != nil || string(data) != "old version" {
		t.Error("tmp-named snapshot content changed")
	}
}

func TestCaptureRejectsNonRegularOutputFile(t *testing.T) {
	mkfifo, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("native FIFO fixture unavailable")
	}
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "ordinary output")
	if err := exec.Command(mkfifo, filepath.Join(root, "site/fifo")).Run(); err != nil {
		t.Fatal(err)
	}
	if err := Capture(root, "v1", false); err == nil {
		t.Error("non-regular output asset accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".mpress/versions/v1")); !os.IsNotExist(err) {
		t.Error("non-regular input published a snapshot")
	}
}

func TestCaptureRejectsExternalPaths(t *testing.T) {
	for _, kind := range []string{"store", "output", "file"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := versionBoundaryFixture(t, ".mpress/versions"), t.TempDir()
			writeVersionFile(t, filepath.Join(root, "site/index.html"), "ordinary output")
			writeVersionFile(t, filepath.Join(outside, "index.html"), "external secret")
			switch kind {
			case "store":
				writeVersionFile(t, filepath.Join(outside, "v1/index.html"), "must survive")
				versionSymlink(t, outside, filepath.Join(root, ".mpress/versions"))
			case "output":
				if err := os.RemoveAll(filepath.Join(root, "site")); err != nil {
					t.Fatal(err)
				}
				versionSymlink(t, outside, filepath.Join(root, "site"))
			case "file":
				versionSymlink(t, filepath.Join(outside, "index.html"), filepath.Join(root, "site/linked.html"))
			}
			if err := Capture(root, "v1", true); err == nil {
				t.Error("capture accepted an external path")
			}
			data, err := os.ReadFile(filepath.Join(outside, "index.html"))
			if err != nil || string(data) != "external secret" {
				t.Error("outside source changed")
			}
			if kind == "store" {
				data, err := os.ReadFile(filepath.Join(outside, "v1/index.html"))
				if err != nil || string(data) != "must survive" {
					t.Error("outside snapshot overwritten")
				}
			}
		})
	}
}
