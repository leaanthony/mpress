package version

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// An output asset link must not let a capture read bytes from elsewhere in the
// project. The selected output subtree is the only permitted source boundary.
func TestCaptureRejectsProjectPrivateAssetAlias(t *testing.T) {
	for _, kind := range []string{"relative", "absolute"} {
		t.Run(kind, func(t *testing.T) {
			root := versionBoundaryFixture(t, ".mpress/versions")
			private := filepath.Join(root, "private", "draft.txt")
			writeVersionFile(t, private, "project-private sentinel")
			writeVersionFile(t, filepath.Join(root, "site/index.html"), "previous verified snapshot")
			if err := Capture(root, "v1", false); err != nil {
				t.Fatal(err)
			}

			writeVersionFile(t, filepath.Join(root, "site/index.html"), "replacement")
			target := filepath.Join("..", "private", "draft.txt")
			if kind == "absolute" {
				target = private
			}
			versionSymlink(t, target, filepath.Join(root, "site/leak.txt"))

			if err := Capture(root, "v1", true); err == nil {
				verified := Verify(root, "v1") == nil
				data, readErr := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/leak.txt"))
				if verified && readErr == nil && string(data) == "project-private sentinel" {
					t.Fatal("capture published project-private bytes into a verified snapshot")
				}
				t.Fatal("capture accepted an output asset link to project-private data")
			}
			if err := Verify(root, "v1"); err != nil {
				t.Fatalf("previous verified snapshot not intact: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/index.html"))
			if err != nil || string(data) != "previous verified snapshot" {
				t.Errorf("rejected capture changed the previous snapshot: %q, %v", data, err)
			}
			if _, err := os.Stat(filepath.Join(root, ".mpress/versions/v1/leak.txt")); !os.IsNotExist(err) {
				t.Error("rejected capture published a partial snapshot")
			}
			if data, err := os.ReadFile(private); err != nil || string(data) != "project-private sentinel" {
				t.Error("project-private sentinel changed")
			}
		})
	}
}

// A file alias that stays inside the selected output subtree is an ordinary
// output asset and must keep working.
func TestCapturePreservesInternalAssetAlias(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/assets/real.css"), "shared stylesheet")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "index")
	versionSymlink(t, filepath.Join("assets", "real.css"), filepath.Join(root, "site/styles.css"))

	if err := Capture(root, "v1", false); err != nil {
		t.Fatalf("internal output alias rejected: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/styles.css"))
	if err != nil || string(data) != "shared stylesheet" {
		t.Errorf("internal output alias not captured: %q, %v", data, err)
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("snapshot with internal alias does not verify: %v", err)
	}
}

// The configured output directory may itself be an internal symlink. Capture
// must follow the alias and still confine bytes to the resolved subtree.
func TestCaptureAliasedOutputRoot(t *testing.T) {
	t.Run("accepts internal selection", func(t *testing.T) {
		root := versionBoundaryFixture(t, ".mpress/versions")
		writeVersionFile(t, filepath.Join(root, "build/actual/index.html"), "index")
		versionSymlink(t, filepath.Join("build", "actual"), filepath.Join(root, "site"))

		if err := Capture(root, "v1", false); err != nil {
			t.Fatalf("internal output root alias rejected: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/index.html"))
		if err != nil || string(data) != "index" {
			t.Errorf("aliased output root not captured: %q, %v", data, err)
		}
	})

	t.Run("rejects escape from alias", func(t *testing.T) {
		root := versionBoundaryFixture(t, ".mpress/versions")
		writeVersionFile(t, filepath.Join(root, "build/actual/index.html"), "index")
		private := filepath.Join(root, "private", "draft.txt")
		writeVersionFile(t, private, "project-private sentinel")
		versionSymlink(t, filepath.Join("build", "actual"), filepath.Join(root, "site"))
		versionSymlink(t, filepath.Join("..", "..", "private", "draft.txt"), filepath.Join(root, "build/actual/leak.txt"))

		if err := Capture(root, "v1", false); err == nil {
			data, readErr := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/leak.txt"))
			if readErr == nil && string(data) == "project-private sentinel" {
				t.Fatal("capture escaped an aliased output root")
			}
			t.Fatal("capture accepted an asset link outside an aliased output root")
		}
		if _, err := os.Stat(filepath.Join(root, ".mpress/versions/v1")); !os.IsNotExist(err) {
			t.Error("rejected capture published a snapshot")
		}
	})
}

// A replaced output directory must not redirect a capture that already pinned
// the selected subtree: copied bytes stay with the pinned generation.
func TestCaptureReadsPinnedOutputAcrossRootSwap(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "generation one")

	files, err := projectfs.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	output, err := files.Sub(filepath.Join(root, "site"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = output.Close() }()

	// Native Windows may deny renaming an open directory; elsewhere the open
	// pinned root must keep reading the original generation.
	if err := os.Rename(filepath.Join(root, "site"), filepath.Join(root, "site-old")); err != nil {
		t.Skipf("native open-directory rename unavailable: %v", err)
	}
	writeVersionFile(t, filepath.Join(root, "site/index.html"), "generation two")
	if err := files.MkdirAll(filepath.Join(root, ".mpress/versions"), 0755); err != nil {
		t.Fatal(err)
	}
	store, err := files.Sub(filepath.Join(root, ".mpress/versions"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	if err := capturePinned(output, store, "v1", false); err != nil {
		t.Fatalf("pinned capture failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/index.html"))
	if err != nil || string(data) != "generation one" {
		t.Errorf("capture copied bytes across a root swap: %q, %v", data, err)
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("pinned snapshot does not verify: %v", err)
	}
}
