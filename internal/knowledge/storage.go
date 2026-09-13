package knowledge

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Large artifacts use deterministic gzip files. The manifest retains the same
// logical JSON layout and digest, but schema 2 tells older clients to upgrade.
func encodeArtifacts(pages, chunks, index []byte, threshold int) (map[string][]byte, Artifacts, int, error) {
	files := make(map[string][]byte)
	artifacts := Artifacts{Pages: PagesFile, Chunks: ChunksFile, Index: IndexFile}
	schema := 1
	for _, item := range []struct {
		name *string
		data []byte
	}{
		{&artifacts.Pages, pages}, {&artifacts.Chunks, chunks}, {&artifacts.Index, index},
	} {
		data := item.data
		if len(data) > threshold {
			var buffer bytes.Buffer
			writer := gzip.NewWriter(&buffer)
			if _, err := writer.Write(data); err != nil {
				return nil, Artifacts{}, 0, err
			}
			if err := writer.Close(); err != nil {
				return nil, Artifacts{}, 0, err
			}
			data = buffer.Bytes()
			*item.name += ".gz"
			schema = Schema
		}
		files[*item.name] = data
	}
	return files, artifacts, schema, nil
}

func readArtifact(path string) ([]byte, error) {
	if !strings.HasSuffix(path, ".gz") {
		return os.ReadFile(path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// Rebuilding into the same output must not leave an oversized uncompressed
// artifact behind, or a stale gzip file after the corpus becomes smaller.
func removeStaleArtifacts(directory string, current map[string][]byte) error {
	for _, base := range []string{PagesFile, ChunksFile, IndexFile} {
		for _, suffix := range []string{"", ".gz"} {
			name := base + suffix
			if _, exists := current[name]; exists {
				continue
			}
			if err := os.Remove(filepath.Join(directory, name)); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}
