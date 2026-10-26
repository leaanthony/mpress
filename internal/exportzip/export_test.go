package exportzip

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestCreateBuildsProductionSiteIntoPortableArchive(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nPublished page.\n")
	writeFixture(t, root, "content/draft.md", "---\ntitle: Draft\ndraft: true\n---\n\nDraft page.\n")
	writeFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n")
	writeFixture(t, root, "static/robots.txt", "User-agent: *\n")
	archivePath := filepath.Join(t.TempDir(), "documentation.zip")

	result, err := Create(root, archivePath, Options{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Archive != archivePath || result.Pages != 1 || result.Files == 0 || result.Bytes == 0 {
		t.Fatalf("unexpected export result: %#v", result)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	files := map[string]string{}
	for _, item := range archive.File {
		reader, openErr := item.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		data, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[item.Name] = string(data)
	}
	if !strings.Contains(files["index.html"], "Published page") || files["robots.txt"] != "User-agent: *\n" {
		t.Fatalf("archive lost generated content: %#v", files)
	}
	if _, included := files["draft/index.html"]; included {
		t.Fatal("production archive included a draft page")
	}
	if _, err := Create(root, archivePath, Options{}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite protection, got %v", err)
	}
	if _, err := Create(root, archivePath, Options{Overwrite: true}); err != nil {
		t.Fatalf("forced export did not replace the archive: %v", err)
	}
}

func writeFixture(t *testing.T, root, name, value string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCreateRetainsInternalStateAliasAndRemovesOwnedWorkspace(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/index.md", "# Home\n")
	writeFixture(t, root, "state/sentinel", "must survive")
	if err := os.Symlink("state", filepath.Join(root, ".mpress")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	archive := filepath.Join(t.TempDir(), "docs.zip")
	if _, err := Create(root, archive, Options{Strict: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "state", "sentinel"))
	if err != nil || string(data) != "must survive" {
		t.Error("export cleanup changed unrelated state")
	}
	entries, err := os.ReadDir(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "export-") {
			t.Error("export left its workspace behind")
		}
	}
	if _, err := os.Readlink(filepath.Join(root, ".mpress")); err != nil {
		t.Error("export replaced the safe state alias")
	}
}

func archiveFiles(t *testing.T, path string) map[string]string {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := archive.Close(); err != nil {
			t.Errorf("close archive: %v", err)
		}
	}()
	files := map[string]string{}
	for _, item := range archive.File {
		reader, err := item.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(reader)
		_ = reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[item.Name] = string(data)
	}
	return files
}

func TestCreateRejectsDanglingStaticAssetAndRetainsPreviousArchive(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nPublished page.\n")
	writeFixture(t, root, "static/asset.txt", "asset bytes\n")
	writeFixture(t, root, "static/real.txt", "real bytes\n")
	writeFixture(t, root, "static/real-dir/file.txt", "nested real bytes\n")
	archive := filepath.Join(t.TempDir(), "documentation.zip")

	if _, err := Create(root, archive, Options{Strict: true}); err != nil {
		t.Fatalf("baseline export: %v", err)
	}
	before := archiveFiles(t, archive)
	for _, name := range []string{"real.txt", "real-dir/file.txt"} {
		if _, exists := before[name]; !exists {
			t.Fatalf("baseline archive lost ordinary asset %q", name)
		}
	}
	beforeBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink("missing-target.txt", filepath.Join(root, "static", "linked.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Create(root, archive, Options{Strict: true, Overwrite: true}); err == nil {
		t.Fatal("forced export with a dangling static asset succeeded")
	}
	afterBytes, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeBytes, afterBytes) {
		t.Fatal("failed forced export changed the previous archive bytes")
	}
	if after := archiveFiles(t, archive); !reflect.DeepEqual(before, after) {
		t.Fatalf("failed forced export changed previous archive entries: %v", after)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".mpress"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "export-") {
			t.Error("failed export left its workspace behind")
		}
	}
}

func TestCreateWithoutStaticDirectoryStillExports(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\nPublished page.\n")
	archive := filepath.Join(t.TempDir(), "documentation.zip")
	if _, err := Create(root, archive, Options{Strict: true}); err != nil {
		t.Fatalf("export without an optional static directory: %v", err)
	}
	if _, exists := archiveFiles(t, archive)["index.html"]; !exists {
		t.Fatal("archive without a static directory is missing index.html")
	}
}
