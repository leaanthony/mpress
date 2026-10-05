package version

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func versionBoundaryFixture(t *testing.T, artifacts string) string {
	t.Helper()
	root := t.TempDir()
	writeVersionFile(t, filepath.Join(root, "mpress.yaml"), fmt.Sprintf("site:\n  title: Versions\nversioning:\n  enabled: true\n  artifactsDir: %q\n", artifacts))
	return root
}

func versionSymlink(t *testing.T, target, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.Symlink(target, probe); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	if err := os.Symlink(target, name); err != nil {
		t.Fatal(err)
	}
}

func versionManifest(t *testing.T, dir, label, name, body string) {
	t.Helper()
	sum := sha256.Sum256([]byte(body))
	data, err := json.Marshal(Manifest{SchemaVersion: 1, Version: label, Files: map[string]string{name: hex.EncodeToString(sum[:])}})
	if err != nil {
		t.Fatal(err)
	}
	writeVersionFile(t, filepath.Join(dir, "mpress-version.json"), string(data))
}

func TestVersionRemoveRejectsExternalArtifactParent(t *testing.T) {
	root, outside := versionBoundaryFixture(t, ".mpress/versions"), t.TempDir()
	writeVersionFile(t, filepath.Join(outside, "v1", "sentinel"), "must survive")
	versionSymlink(t, outside, filepath.Join(root, ".mpress", "versions"))
	if err := Remove(root, "v1"); err == nil {
		t.Error("external snapshot removal accepted")
	}
	data, err := os.ReadFile(filepath.Join(outside, "v1", "sentinel"))
	if err != nil || string(data) != "must survive" {
		t.Error("external sentinel was removed")
	}
}

func TestVersionRemovePreservesProjectInputs(t *testing.T) {
	for _, base := range []string{".", "content", "static", "site", ".git", ".mpress", ".mpress/cache", ".mpress/backups"} {
		t.Run(base, func(t *testing.T) {
			root := versionBoundaryFixture(t, base)
			name := filepath.Join(root, base, "v1", "sentinel")
			writeVersionFile(t, name, "must survive")
			if err := Remove(root, "v1"); err == nil {
				t.Error("removal with overlapping artifact store accepted")
			}
			data, err := os.ReadFile(name)
			if err != nil || string(data) != "must survive" {
				t.Error("project input/state sentinel was removed")
			}
		})
	}
}

func TestVersionListAndVerifyRejectExternalLinks(t *testing.T) {
	for _, kind := range []string{"store", "snapshot", "manifest", "file", "dangling"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := versionBoundaryFixture(t, ".mpress/versions"), t.TempDir()
			dir := filepath.Join(root, ".mpress/versions/v1")
			writeVersionFile(t, filepath.Join(outside, "index.html"), "external secret")
			versionManifest(t, outside, "v1", "index.html", "external secret")
			switch kind {
			case "store":
				versionManifest(t, filepath.Join(outside, "v1"), "v1", "index.html", "external secret")
				writeVersionFile(t, filepath.Join(outside, "v1/index.html"), "external secret")
				versionSymlink(t, outside, filepath.Dir(dir))
			case "snapshot":
				versionSymlink(t, outside, dir)
			case "manifest":
				writeVersionFile(t, filepath.Join(dir, "index.html"), "external secret")
				versionSymlink(t, filepath.Join(outside, "mpress-version.json"), filepath.Join(dir, "mpress-version.json"))
			default:
				versionManifest(t, dir, "v1", "index.html", "external secret")
				target := filepath.Join(outside, "index.html")
				if kind == "dangling" {
					target = filepath.Join(outside, "missing")
				}
				versionSymlink(t, target, filepath.Join(dir, "index.html"))
			}
			if err := Verify(root, "v1"); err == nil {
				t.Error("verification followed an external snapshot input")
			}
			if kind == "store" || kind == "manifest" {
				labels, err := List(root)
				if err == nil && len(labels) != 0 {
					t.Errorf("List published external manifest metadata: %v", labels)
				}
			}
		})
	}
}

func TestVersionVerifyRejectsTraversalLabel(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	dir := filepath.Join(root, ".mpress/victim")
	versionManifest(t, dir, "../victim", "index.html", "secret")
	writeVersionFile(t, filepath.Join(dir, "index.html"), "secret")
	if err := Verify(root, "../victim"); err == nil {
		t.Error("verification accepted traversal label")
	}
}

func TestVersionVerifyConfinesManifestToSnapshot(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	dir := filepath.Join(root, ".mpress/versions/v1")
	writeVersionFile(t, filepath.Join(root, "secret"), "secret")
	versionManifest(t, dir, "v1", "index.html", "secret")
	versionSymlink(t, filepath.Join(root, "secret"), filepath.Join(dir, "index.html"))
	if err := Verify(root, "v1"); err == nil {
		t.Error("manifest followed a file outside the selected snapshot")
	}
}

func TestVersionVerifyRejectsUnsupportedSchema(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, ".mpress/versions/v1/mpress-version.json"), `{"schemaVersion":999,"version":"v1","files":{}}`)
	if err := Verify(root, "v1"); err == nil {
		t.Error("unsupported manifest schema accepted")
	}
}

func TestVersionOperationsSharePortableLabels(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	for _, label := range []string{"", ".", "..", "../v1", "v1/other", `v1\other`, "v1.", "CON", "LPT1", "v1%2fother"} {
		t.Run(label, func(t *testing.T) {
			for _, operation := range []struct {
				name string
				run  func() error
			}{
				{"capture", func() error { return Capture(root, label, false) }},
				{"verify", func() error { return Verify(root, label) }},
				{"remove", func() error { return Remove(root, label) }},
			} {
				if err := operation.run(); err == nil {
					t.Errorf("%s accepted unsafe label", operation.name)
				}
			}
		})
	}
}

func TestVersionVerifyRejectsNoncanonicalManifestPaths(t *testing.T) {
	for _, name := range []string{"nested/../index.html", "./index.html", "index.html/", `/index.html`, `C:\index.html`, `nested\index.html`} {
		t.Run(name, func(t *testing.T) {
			if _, err := manifestFilePath("snapshot", name); err == nil {
				t.Error("noncanonical/host-dependent manifest path accepted")
			}
		})
	}
}

func TestVersionRemoveRejectsSnapshotAliasToSource(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	writeVersionFile(t, filepath.Join(root, "content", "index.md"), "must survive")
	versionSymlink(t, filepath.Join(root, "content"), filepath.Join(root, ".mpress/versions/v1"))
	if err := Remove(root, "v1"); err == nil {
		t.Error("snapshot alias to protected source accepted")
	}
	data, err := os.ReadFile(filepath.Join(root, "content", "index.md"))
	if err != nil || string(data) != "must survive" {
		t.Error("source changed")
	}
	if _, err := os.Lstat(filepath.Join(root, ".mpress/versions/v1")); err != nil {
		t.Error("rejected source alias was removed")
	}
}

func TestVersionSafeInternalStoreAndRemoval(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	dir := filepath.Join(root, "snapshots/v1")
	writeVersionFile(t, filepath.Join(dir, "index.html"), "ordinary snapshot")
	versionManifest(t, dir, "v1", "index.html", "ordinary snapshot")
	versionSymlink(t, "index.html", filepath.Join(dir, "alias.html"))
	data, err := os.ReadFile(filepath.Join(dir, "mpress-version.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Files["alias.html"] = manifest.Files["index.html"]
	data, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeVersionFile(t, filepath.Join(dir, "mpress-version.json"), string(data))
	writeVersionFile(t, filepath.Join(root, "snapshots/unrelated"), "must survive")
	versionSymlink(t, filepath.Join(root, "snapshots"), filepath.Join(root, ".mpress/versions"))
	if err := Verify(root, "v1"); err != nil {
		t.Fatal(err)
	}
	labels, err := List(root)
	if err != nil || len(labels) != 1 || labels[0] != "v1" {
		t.Fatalf("List: %v, %v", labels, err)
	}
	alias := filepath.Join(root, ".mpress/versions/alias")
	versionSymlink(t, dir, alias)
	if err := Remove(root, "alias"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(alias); !os.IsNotExist(err) {
		t.Error("selected snapshot alias remains")
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("removing alias changed its target: %v", err)
	}
	if err := Remove(root, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("selected snapshot remains")
	}
	if data, err := os.ReadFile(filepath.Join(root, "snapshots/unrelated")); err != nil || string(data) != "must survive" {
		t.Error("unrelated store file changed")
	}
}

func TestVersionMissingStoreAndConfiguration(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	if labels, err := List(root); err != nil || len(labels) != 0 {
		t.Fatalf("missing store listing: %v, %v", labels, err)
	}
	if err := Remove(root, "v1"); err != nil {
		t.Fatalf("missing snapshot removal: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "mpress.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := List(root); err == nil {
		t.Error("missing configuration swallowed by List")
	}
	if err := Remove(root, "v1"); err == nil {
		t.Error("missing configuration swallowed by Remove")
	}
}
