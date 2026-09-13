package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaptureAndVerifyDetectsMutation(t *testing.T) {
	root := t.TempDir()
	writeVersionFile(t, filepath.Join(root, "mpress.yaml"), `site:
  title: Versions
build:
  contentDir: content
  staticDir: static
  outputDir: site
versioning:
  enabled: true
  artifactsDir: .mpress/versions
`)
	writeVersionFile(t, filepath.Join(root, "site", "index.html"), "version one")
	if err := Capture(root, "v1.0", false); err != nil {
		t.Fatal(err)
	}
	if err := Verify(root, "v1.0"); err != nil {
		t.Fatal(err)
	}
	writeVersionFile(t, filepath.Join(root, ".mpress", "versions", "v1.0", "index.html"), "tampered")
	if err := Verify(root, "v1.0"); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
}

func TestListIgnoresIncompleteVersionDirectories(t *testing.T) {
	root := t.TempDir()
	writeVersionFile(t, filepath.Join(root, "mpress.yaml"), `site:
  title: Versions
versioning:
  enabled: true
  artifactsDir: .mpress/versions
`)
	writeVersionFile(t, filepath.Join(root, ".mpress", "versions", "1.0", "mpress-version.json"), `{"version":"1.0"}`)
	writeVersionFile(t, filepath.Join(root, ".mpress", "versions", "2.0.tmp", "index.html"), "partial")
	writeVersionFile(t, filepath.Join(root, ".mpress", "versions", "broken", "index.html"), "no manifest")

	labels, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(labels) != 1 || labels[0] != "1.0" {
		t.Fatalf("List() = %#v, want [1.0]", labels)
	}
}

func TestVerifyRejectsUnexpectedFiles(t *testing.T) {
	root := t.TempDir()
	writeVersionFile(t, filepath.Join(root, "mpress.yaml"), `site:
  title: Versions
build:
  outputDir: site
versioning:
  enabled: true
  artifactsDir: .mpress/versions
`)
	writeVersionFile(t, filepath.Join(root, "site", "index.html"), "version one")
	if err := Capture(root, "1.0", false); err != nil {
		t.Fatal(err)
	}
	writeVersionFile(t, filepath.Join(root, ".mpress", "versions", "1.0", "injected.html"), "not in manifest")
	if err := Verify(root, "1.0"); err == nil || !strings.Contains(err.Error(), "unexpected file") {
		t.Fatalf("expected unexpected-file error, got %v", err)
	}
}

func writeVersionFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
