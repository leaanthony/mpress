package knowledge

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/projectfs"
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

// ErrResourceLimit identifies a bundle rejected before unbounded allocation.
var ErrResourceLimit = errors.New("knowledge resource limit exceeded")

func readArtifact(files *projectfs.FS, path string) ([]byte, error) {
	return readArtifactLimit(files, path, defaultLoadLimits().artifact)
}

func readArtifactLimit(files *projectfs.FS, path string, maximum int64) ([]byte, error) {
	file, err := files.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular knowledge file: %s", path)
	}
	if maximum < 0 || info.Size() > maximum {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrResourceLimit, path, maximum)
	}
	encoded := &io.LimitedReader{R: file, N: maximum + 1}
	if !strings.HasSuffix(path, ".gz") {
		return readSizedArtifact(encoded, info.Size(), maximum)
	}
	compressed, err := gzip.NewReader(encoded)
	if err != nil {
		return nil, err
	}
	defer compressed.Close()
	data, err := io.ReadAll(io.LimitReader(compressed, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum || encoded.N == 0 {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", ErrResourceLimit, path, maximum)
	}
	return data, nil
}

// A plain file's observed size avoids repeated buffer growth. The size is only
// a hint: shrinking files are read to EOF, and growth still obeys the byte cap.
func readSizedArtifact(reader io.Reader, size, maximum int64) ([]byte, error) {
	data := make([]byte, min(size, maximum)+1)
	n, err := io.ReadFull(reader, data)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	data = data[:n]
	if err == nil && int64(n) <= maximum {
		tail, err := io.ReadAll(io.LimitReader(reader, maximum+1-int64(n)))
		if err != nil {
			return nil, err
		}
		data = append(data, tail...)
	}
	if int64(len(data)) > maximum {
		return nil, ErrResourceLimit
	}
	return data, nil
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
