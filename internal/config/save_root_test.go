package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
)

func TestRootedConfigSaveRetainsSerialization(t *testing.T) {
	project := t.TempDir()
	cfg := Default()
	if err := Save(project, cfg); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(project, Filename))
	if err != nil {
		t.Fatal(err)
	}
	files, err := projectfs.Open(project)
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	if err := SaveRoot(files, cfg); err != nil {
		t.Fatal(err)
	}
	after, err := files.ReadFile(Filename)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("rooted configuration serialization changed: %v", err)
	}
}
