package site

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// staticAssetsProject creates the ordinary assets used by the dangling-link
// regressions. real.txt and real-dir/file.txt sort after linked.txt, so a walk
// that aborts at the dangling link would silently lose them.
func staticAssetsProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", preservationConfig)
	writeFixture(t, root, "content/index.md", "---\ntitle: Original\n---\n\nOriginal home.\n")
	writeFixture(t, root, "static/asset.txt", "asset bytes\n")
	writeFixture(t, root, "static/real.txt", "real bytes\n")
	writeFixture(t, root, "static/real-dir/file.txt", "nested real bytes\n")
	return root
}

func TestBuildRejectsDanglingStaticAssetAndPreservesPublishedSite(t *testing.T) {
	root := staticAssetsProject(t)
	if err := os.Symlink("asset.txt", filepath.Join(root, "static", "alias.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("baseline build: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "site", "real.txt")); err != nil || string(data) != "real bytes\n" {
		t.Fatalf("baseline ordinary asset: %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "site", "real-dir", "file.txt")); err != nil || string(data) != "nested real bytes\n" {
		t.Fatalf("baseline nested ordinary asset: %q, %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "site", "alias.txt")); err != nil || string(data) != "asset bytes\n" {
		t.Fatalf("baseline internal static alias: %q, %v", data, err)
	}
	writeFixture(t, root, "site/previous-only.txt", "last good build")
	before := snapshotRetainedTree(t, filepath.Join(root, "site"))
	writeFixture(t, root, "content/index.md", "---\ntitle: Changed\n---\n\nChanged home.\n")
	if err := os.Symlink("missing-target.txt", filepath.Join(root, "static", "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, test := range []struct {
		name    string
		options BuildOptions
	}{
		{name: "strict", options: BuildOptions{Strict: true}},
		{name: "normal", options: BuildOptions{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Build(root, test.options); err == nil {
				t.Fatal("build with a dangling static asset succeeded")
			} else if !strings.Contains(err.Error(), "static") {
				t.Fatalf("build error does not identify the static copy: %v", err)
			}
			if after := snapshotRetainedTree(t, filepath.Join(root, "site")); !reflect.DeepEqual(before, after) {
				t.Fatal("failed build changed published names, modes, aliases or bytes")
			}
			assertNoStagingLeftovers(t, root)
			if target, err := os.Readlink(filepath.Join(root, "static", "linked.txt")); err != nil || target != "missing-target.txt" {
				t.Fatalf("failed build changed the dangling source alias: %q %v", target, err)
			}
			if data, err := os.ReadFile(filepath.Join(root, "static", "real.txt")); err != nil || string(data) != "real bytes\n" {
				t.Fatalf("failed build changed the static source: %q, %v", data, err)
			}
			target, err := os.Readlink(filepath.Join(root, "static", "alias.txt"))
			if err != nil || target != "asset.txt" {
				t.Fatalf("failed build changed the internal static alias: %q, %v", target, err)
			}
		})
	}
}

func TestBuildRejectsDanglingStaticAssetWithoutPublishingPartialSite(t *testing.T) {
	root := staticAssetsProject(t)
	if err := os.Symlink("missing-target.txt", filepath.Join(root, "static", "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err == nil {
		t.Fatal("build with a dangling static asset succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, "site")); !os.IsNotExist(err) {
		t.Fatalf("failed build published a partial site: %v", err)
	}
	assertNoStagingLeftovers(t, root)
}

func TestBuildPublishesCorrectedStaticAliasOnRetry(t *testing.T) {
	root := staticAssetsProject(t)
	linked := filepath.Join(root, "static", "linked.txt")
	if err := os.Symlink("missing-target.txt", linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err == nil {
		t.Fatal("build with a dangling static asset succeeded")
	}
	if err := os.Remove(linked); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real.txt", linked); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("retry after correcting the static alias: %v", err)
	}
	for name, want := range map[string]string{
		"asset.txt":         "asset bytes\n",
		"linked.txt":        "real bytes\n",
		"real.txt":          "real bytes\n",
		"real-dir/file.txt": "nested real bytes\n",
	} {
		data, err := os.ReadFile(filepath.Join(root, "site", filepath.FromSlash(name)))
		if err != nil || string(data) != want {
			t.Fatalf("published %s = %q, %v; want %q", name, data, err, want)
		}
	}
}

func TestBuildWithoutStaticDirectoryStillSucceeds(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", preservationConfig)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nHome.\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("build without an optional static directory: %v", err)
	}
	assertExists(t, filepath.Join(root, "site", "index.html"))
}

func TestBuildPublishesCorrectedStaticRootOnRetry(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", preservationConfig)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nHome.\n")
	writeFixture(t, root, "assets/real.txt", "real bytes\n")
	static := filepath.Join(root, "static")
	if err := os.Symlink("missing-static", static); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err == nil {
		t.Fatal("build with a dangling static root succeeded")
	}
	if _, err := os.Stat(filepath.Join(root, "site")); !os.IsNotExist(err) {
		t.Fatalf("failed build published a partial site: %v", err)
	}
	if err := os.Remove(static); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("assets", static); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatalf("retry after correcting the static root: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "site", "real.txt")); err != nil || string(data) != "real bytes\n" {
		t.Fatalf("corrected static root was not published: %q, %v", data, err)
	}
}
