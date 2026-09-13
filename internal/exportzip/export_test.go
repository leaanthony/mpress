package exportzip

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
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
