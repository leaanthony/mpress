package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leaanthony/mpress/internal/projectfs"
)

func mountFixture(t *testing.T) string {
	t.Helper()
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "site/index.html"), `<a href="/guide/">saved page</a>`)
	if err := Capture(root, "v1", false); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestMountPreservesInputsBehindNestedLinks(t *testing.T) {
	for _, store := range []bool{true, false} {
		t.Run(map[bool]string{true: "snapshot", false: "missing-store"}[store], func(t *testing.T) {
			root := mountFixture(t)
			if !store {
				if err := os.RemoveAll(filepath.Join(root, ".mpress/versions")); err != nil {
					t.Fatal(err)
				}
			}
			sentinel := filepath.Join(root, "content/private.json")
			writeVersionFile(t, sentinel, "must survive")
			output := filepath.Join(root, "mounted")
			versionSymlink(t, sentinel, filepath.Join(output, "versions/versions.json"))
			if _, err := Mount(root, output); err == nil {
				t.Error("metadata link out of output boundary accepted")
			}
			if data, err := os.ReadFile(sentinel); err != nil || string(data) != "must survive" {
				t.Error("private source changed")
			}
		})
	}
}

func TestMountRetainsNavigationAssetsAndSafeAliases(t *testing.T) {
	root := mountFixture(t)
	writeVersionFile(t, filepath.Join(root, "site/guide/index.html"), `<a href='/'>home</a><img src="/logo.svg"><a href="//example.org">external</a>`+versionMenuStart+"old menu"+versionMenuEnd)
	writeVersionFile(t, filepath.Join(root, "site/data/mpress-version.json"), `{"asset":true}`)
	writeVersionFile(t, filepath.Join(root, "site/logo.svg"), "logo")
	if err := Capture(root, "v1", true); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "mounted")
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "mounted-alias")
	versionSymlink(t, output, alias)
	if count, err := Mount(root, alias); err != nil || count != 1 {
		t.Fatalf("Mount: %d, %v", count, err)
	}
	data, err := os.ReadFile(filepath.Join(output, "versions/v1/guide/index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"href='/versions/v1/'", `src="/versions/v1/logo.svg"`, `href="//example.org"`, `href="/guide/"`, `href="/versions/v1/guide/"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("mounted navigation missing %q", want)
		}
	}
	asset, err := os.ReadFile(filepath.Join(output, "versions/v1/data/mpress-version.json"))
	if err != nil || string(asset) != `{"asset":true}` {
		t.Error("nested manifest-named asset missing")
	}
	if _, err := os.Stat(filepath.Join(output, "versions/v1/mpress-version.json")); !os.IsNotExist(err) {
		t.Error("root snapshot manifest was published")
	}
	metadata, err := os.ReadFile(filepath.Join(output, "versions/versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Versions []string `json:"versions"`
	}
	if err := json.Unmarshal(metadata, &index); err != nil || len(index.Versions) != 1 || index.Versions[0] != "v1" {
		t.Errorf("invalid version index: %s, %v", metadata, err)
	}
}

func TestVersionReadersWaitForReplacement(t *testing.T) {
	for _, operation := range []string{"list", "verify", "mount"} {
		t.Run(operation, func(t *testing.T) {
			root := mountFixture(t)
			store, err := projectfs.Open(filepath.Join(root, ".mpress/versions"))
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			lock, err := store.Lock(storeLockFile)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Close()
			// Model the replacement interval with the old snapshot safely retained.
			if err := store.Rename("v1", ".previous-test"); err != nil {
				t.Fatal(err)
			}
			result := make(chan error, 1)
			go func() {
				switch operation {
				case "list":
					labels, err := List(root)
					if err == nil && (len(labels) != 1 || labels[0] != "v1") {
						err = os.ErrNotExist
					}
					result <- err
				case "verify":
					result <- Verify(root, "v1")
				case "mount":
					_, err := Mount(root, filepath.Join(root, "mounted"))
					result <- err
				}
			}()
			select {
			case err := <-result:
				t.Fatalf("reader observed unfinished replacement: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			if err := store.Rename(".previous-test", "v1"); err != nil {
				t.Fatal(err)
			}
			if err := lock.Close(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-result:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("reader did not resume after replacement")
			}
		})
	}
}

func TestVersionReadersSupportLegacyReadOnlyStore(t *testing.T) {
	root := mountFixture(t)
	store := filepath.Join(root, ".mpress/versions")
	lock := filepath.Join(store, storeLockFile)
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(store, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(store, 0755)
	probe := filepath.Join(store, "permission-probe")
	if err := os.WriteFile(probe, nil, 0600); err == nil {
		os.Remove(probe)
		t.Skip("native permissions do not enforce a read-only store")
	}
	if labels, err := List(root); err != nil || len(labels) != 1 {
		t.Fatalf("List: %v, %v", labels, err)
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatal(err)
	}
	if n, err := Mount(root, filepath.Join(root, "mounted")); err != nil || n != 1 {
		t.Fatalf("Mount: %d, %v", n, err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Error("read-only legacy store changed")
	}
}

func TestMountRejectsTamperedSnapshot(t *testing.T) {
	root := mountFixture(t)
	writeVersionFile(t, filepath.Join(root, ".mpress/versions/v1/index.html"), "tampered")
	output := filepath.Join(root, "mounted")
	if n, err := Mount(root, output); err == nil || n != 0 {
		t.Fatalf("tampered snapshot mounted: %d, %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(output, "versions/v1/index.html")); !os.IsNotExist(err) {
		t.Error("tampered bytes published")
	}
}

func TestMountReadOnlyDestinationPreservesFiles(t *testing.T) {
	root := mountFixture(t)
	output := filepath.Join(root, "mounted")
	destination := filepath.Join(output, "versions/v1")
	sentinel := filepath.Join(destination, "index.html")
	writeVersionFile(t, sentinel, "last output")
	if err := os.Chmod(destination, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(destination, 0755)
	probe := filepath.Join(destination, "permission-probe")
	if err := os.WriteFile(probe, nil, 0600); err == nil {
		os.Remove(probe)
		t.Skip("native permissions do not enforce a read-only directory")
	}
	if _, err := Mount(root, output); err == nil {
		t.Error("read-only mounting destination accepted")
	}
	if data, err := os.ReadFile(sentinel); err != nil || string(data) != "last output" {
		t.Error("previous output was changed")
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("failed mounting changed snapshot: %v", err)
	}
}

func TestMountRejectsExternalDestinations(t *testing.T) {
	for _, kind := range []string{"output", "versions", "snapshot", "metadata"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := mountFixture(t), t.TempDir()
			output := filepath.Join(root, "mounted")
			sentinel := filepath.Join(outside, "index.html")
			writeVersionFile(t, sentinel, "must survive")
			link := output
			switch kind {
			case "versions":
				link = filepath.Join(output, "versions")
			case "snapshot":
				link = filepath.Join(output, "versions/v1")
			case "metadata":
				link = filepath.Join(output, "versions/versions.json")
			}
			target := outside
			if kind == "metadata" {
				target = sentinel
			}
			versionSymlink(t, target, link)
			if _, err := Mount(root, output); err == nil {
				t.Error("external mounting destination accepted")
			}
			if data, err := os.ReadFile(sentinel); err != nil || string(data) != "must survive" {
				t.Error("mounting overwrote external sentinel")
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 1 {
				t.Error("mounting created external files or directories")
			}
		})
	}
}

func TestMountPreservesProjectInputs(t *testing.T) {
	for _, name := range []string{".", "content", "static", ".git", ".mpress", ".mpress/cache", ".mpress/versions"} {
		t.Run(name, func(t *testing.T) {
			root := mountFixture(t)
			output := filepath.Join(root, name)
			sentinel := filepath.Join(output, "versions/v1/index.html")
			writeVersionFile(t, sentinel, "must survive")
			if _, err := Mount(root, output); err == nil {
				t.Error("mounting into protected project inputs accepted")
			}
			if data, err := os.ReadFile(sentinel); err != nil || string(data) != "must survive" {
				t.Error("project input changed")
			}
		})
	}
}
