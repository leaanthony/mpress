package knowledge

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

func TestCompressedKnowledgeRoundTrip(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "ja"}
	pages := map[string][]*content.Page{
		"en": {{Title: "Install", URLPath: "install", PlainText: "Install the compiler.", HTML: "<h1>Install</h1><p>Install the compiler.</p>"}},
		"ja": {{Title: "インストール", URLPath: "install", PlainText: "コンパイラをインストールします。", HTML: "<h1>インストール</h1><p>コンパイラをインストールします。</p>"}},
	}
	plain, compressed, repeated := t.TempDir(), t.TempDir(), t.TempDir()
	if err := Generate(plain, cfg, pages); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{compressed, repeated} {
		if err := generate(output, cfg, pages, 1); err != nil {
			t.Fatal(err)
		}
	}
	original, err := Load(plain)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Load(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if original.Manifest.Schema != 1 || restored.Manifest.Schema != Schema {
		t.Fatal("wrong compatibility schema")
	}
	if original.Manifest.Digest != restored.Manifest.Digest || !reflect.DeepEqual(original.Pages, restored.Pages) || !reflect.DeepEqual(original.Chunks, restored.Chunks) {
		t.Fatal("compression changed knowledge content")
	}
	for _, options := range []SearchOptions{{Query: "compiler", Language: "en"}, {Query: "インストール", Language: "ja"}} {
		want, got := original.Search(options), restored.Search(options)
		if len(got) == 0 || !reflect.DeepEqual(want, got) {
			t.Fatalf("search mismatch for %s", options.Language)
		}
	}
	for _, name := range []string{ManifestFile, PagesFile + ".gz", ChunksFile + ".gz", IndexFile + ".gz"} {
		first, err := os.ReadFile(filepath.Join(compressed, Directory, name))
		if err != nil {
			t.Fatal(err)
		}
		second, err := os.ReadFile(filepath.Join(repeated, Directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, second) {
			t.Fatalf("%s is not deterministic", name)
		}
	}
	// Corruption must still fail the digest even when the gzip stream is valid.
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte("[]\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(compressed, Directory, ChunksFile+".gz"), buffer.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(compressed); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("tampering returned %v", err)
	}
}

func TestKnowledgeCompressionThreshold(t *testing.T) {
	files, artifacts, schema, err := encodeArtifacts([]byte("[]\n"), []byte("[]\n"), []byte("{\"terms\":[]}\n"), 3)
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.Pages != PagesFile || artifacts.Chunks != ChunksFile || artifacts.Index != IndexFile+".gz" || schema != Schema {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
	output := t.TempDir()
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(output, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := readArtifact(filepath.Join(output, artifacts.Index))
	if err != nil {
		t.Fatal(err)
	}
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	if _, err := readArtifact(filepath.Join(output, "missing.gz")); err == nil {
		t.Fatal("missing gzip accepted")
	}
	if err := os.WriteFile(filepath.Join(output, "invalid.gz"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readArtifact(filepath.Join(output, "invalid.gz")); err == nil {
		t.Fatal("invalid gzip accepted")
	}
}

func TestRegenerationRemovesObsoleteArtifactEncoding(t *testing.T) {
	cfg := config.Default()
	pages := map[string][]*content.Page{"en": {{Title: "Home", PlainText: "Home", HTML: "<p>Home</p>"}}}
	output := t.TempDir()
	for _, threshold := range []int{1 << 20, 1, 1 << 20} {
		if err := generate(output, cfg, pages, threshold); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(output); err != nil {
			t.Fatal(err)
		}
		files, err := os.ReadDir(filepath.Join(output, Directory))
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 4 {
			t.Fatalf("obsolete artifact left behind: %v", files)
		}
	}
}
