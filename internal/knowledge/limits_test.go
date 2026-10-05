package knowledge

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
)

func TestKnowledgeArtifactByteLimits(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(fmt.Sprint(compressed), func(t *testing.T) {
			output := t.TempDir()
			data := bytes.Repeat([]byte("x"), 1024)
			name := "payload.json"
			encoded := data
			if compressed {
				name += ".gz"
				encoded = gzipFixture(t, data)
			}
			if err := os.WriteFile(filepath.Join(output, name), encoded, 0600); err != nil {
				t.Fatal(err)
			}
			root := knowledgeTestRoot(t, output)
			got, err := readArtifactLimit(root, name, 1024)
			if err != nil || !bytes.Equal(got, data) {
				t.Fatalf("exact boundary rejected: %v", err)
			}
			if _, err := readArtifactLimit(root, name, 1023); !errors.Is(err, ErrResourceLimit) {
				t.Fatalf("oversized payload accepted: %v", err)
			}
		})
	}
}

func TestKnowledgePlainReadSizeChanges(t *testing.T) {
	for _, item := range []struct {
		name          string
		hint, maximum int64
		data          string
		limited       bool
	}{
		{"unchanged", 4, 4, "abcd", false},
		{"shrunk", 4, 4, "ab", false},
		{"grown-within-budget", 2, 4, "abcd", false},
		{"grown-outside-budget", 2, 4, "abcde", true},
	} {
		t.Run(item.name, func(t *testing.T) {
			data, err := readSizedArtifact(strings.NewReader(item.data), item.hint, item.maximum)
			if item.limited {
				if !errors.Is(err, ErrResourceLimit) {
					t.Fatalf("growth escaped budget: %v", err)
				}
				return
			}
			if err != nil || string(data) != item.data {
				t.Fatalf("size change lost bytes: %q %v", data, err)
			}
		})
	}
}

func gzipFixture(t *testing.T, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	w := gzip.NewWriter(&buffer)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func knowledgeTestRoot(t *testing.T, output string) *projectfs.FS {
	t.Helper()
	root, err := projectfs.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { root.Close() })
	return root
}

func TestKnowledgeGzipIntegrityAndExpansion(t *testing.T) {
	encoded := gzipFixture(t, bytes.Repeat([]byte("x"), 4096))
	badCRC := append([]byte(nil), encoded...)
	badCRC[len(badCRC)-8] ^= 1
	joined := append(append([]byte(nil), encoded...), encoded...)
	for _, item := range []struct {
		name    string
		data    []byte
		maximum int64
		limit   bool
	}{
		{"truncated-header", encoded[:5], 8192, false},
		{"truncated-trailer", encoded[:len(encoded)-1], 8192, false},
		{"bad-checksum", badCRC, 8192, false},
		{"expansion", encoded, 256, true},
		{"combined-members", joined, 4096, true},
	} {
		t.Run(item.name, func(t *testing.T) {
			output := t.TempDir()
			if err := os.WriteFile(filepath.Join(output, "data.gz"), item.data, 0600); err != nil {
				t.Fatal(err)
			}
			_, err := readArtifactLimit(knowledgeTestRoot(t, output), "data.gz", item.maximum)
			if err == nil || (item.limit && !errors.Is(err, ErrResourceLimit)) {
				t.Fatalf("invalid or oversized stream accepted: %v", err)
			}
		})
	}
}

func TestKnowledgeBundleBudgets(t *testing.T) {
	output := generatedBoundaryBundle(t)
	root := knowledgeTestRoot(t, output)
	files, err := os.ReadDir(filepath.Join(output, Directory))
	if err != nil {
		t.Fatal(err)
	}
	var total, manifestSize, largest int64
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			t.Fatal(err)
		}
		total += info.Size()
		if file.Name() == ManifestFile {
			manifestSize = info.Size()
		} else {
			largest = max(largest, info.Size())
		}
	}
	for _, kind := range []string{"exact", "manifest", "artifact", "bundle", "aggregate"} {
		t.Run(kind, func(t *testing.T) {
			limits := defaultLoadLimits()
			limits.manifest, limits.artifact, limits.bundle, limits.aggregate = manifestSize, largest, total, total
			switch kind {
			case "manifest":
				limits.manifest--
			case "artifact":
				limits.artifact--
			case "bundle":
				limits.bundle--
			case "aggregate":
				limits.aggregate--
			}
			_, err := loadSiteBudget(root, newLoadBudget(limits))
			if kind == "exact" {
				if err != nil {
					t.Fatalf("exact bundle limits rejected: %v", err)
				}
			} else if !errors.Is(err, ErrResourceLimit) {
				t.Fatalf("%s budget not enforced: %v", kind, err)
			}
		})
	}
}

func TestKnowledgeMountedBudgets(t *testing.T) {
	output := generatedBoundaryBundle(t)
	snapshot := filepath.Join(output, "versions", "v1", Directory)
	if err := os.MkdirAll(snapshot, 0755); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(filepath.Join(output, Directory))
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, file := range files {
		data := readIdentityFixture(t, filepath.Join(output, Directory, file.Name()))
		total += int64(len(data))
		if err := os.WriteFile(filepath.Join(snapshot, file.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	root := knowledgeTestRoot(t, output)
	for _, kind := range []string{"exact", "aggregate", "versions", "entries"} {
		t.Run(kind, func(t *testing.T) {
			limits := defaultLoadLimits()
			limits.aggregate, limits.versions, limits.entries = total*2, 1, 1
			switch kind {
			case "aggregate":
				limits.aggregate--
			case "versions":
				limits.versions = 0
			case "entries":
				limits.entries = 0
			}
			store, err := loadAllBudget(root, newLoadBudget(limits))
			if kind == "exact" {
				if err != nil || len(store.Pages) != 2 {
					t.Fatalf("exact mounted limits rejected: %v", err)
				}
			} else if !errors.Is(err, ErrResourceLimit) {
				t.Fatalf("%s budget not enforced: %v", kind, err)
			}
		})
	}
}

func TestKnowledgeUnsupportedSchemas(t *testing.T) {
	for _, kind := range []string{"manifest", "index"} {
		t.Run(kind, func(t *testing.T) {
			output := generatedBoundaryBundle(t)
			name := ManifestFile
			if kind == "index" {
				name = IndexFile
			}
			path := filepath.Join(output, Directory, name)
			var value map[string]any
			if err := json.Unmarshal(readIdentityFixture(t, path), &value); err != nil {
				t.Fatal(err)
			}
			value["schema"] = 999
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "index" {
				refreshKnowledgeDigest(t, output)
			}
			if _, err := Load(output); err == nil || !strings.Contains(err.Error(), "schema") {
				t.Fatalf("unsupported %s schema not rejected: %v", kind, err)
			}
		})
	}
}

func TestKnowledgeArtifactReplacement(t *testing.T) {
	output := generatedBoundaryBundle(t)
	want, err := Load(output)
	if err != nil {
		t.Fatal(err)
	}
	files := knowledgeTestRoot(t, filepath.Join(output, Directory))
	original := readIdentityFixture(t, filepath.Join(output, Directory, PagesFile))
	stop, done := replacingKnowledgeFile(files, original)
	for range 30 {
		got, err := LoadAll(output)
		if err == nil && (!reflect.DeepEqual(want.Pages, got.Pages) || !reflect.DeepEqual(want.Chunks, got.Chunks)) {
			t.Error("replacement served an unverified corpus")
		}
	}
	close(stop)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := files.WriteAtomic(PagesFile, original); err != nil {
		t.Fatal(err)
	}
	got, err := LoadAll(output)
	if err != nil || !reflect.DeepEqual(want.Pages, got.Pages) {
		t.Fatalf("restored bundle did not recover: %v", err)
	}
}

func replacingKnowledgeFile(files *projectfs.FS, original []byte) (chan struct{}, chan error) {
	stop, done := make(chan struct{}), make(chan error, 1)
	go func() {
		for {
			select {
			case <-stop:
				done <- nil
				return
			default:
			}
			for _, data := range [][]byte{[]byte("[{"), original} {
				if err := files.WriteAtomic(PagesFile, data); err != nil {
					done <- err
					return
				}
			}
		}
	}()
	return stop, done
}

func refreshKnowledgeDigest(t *testing.T, output string) {
	t.Helper()
	directory := filepath.Join(output, Directory)
	path := filepath.Join(directory, ManifestFile)
	var manifest Manifest
	if err := json.Unmarshal(readIdentityFixture(t, path), &manifest); err != nil {
		t.Fatal(err)
	}
	sum := sha256.New()
	for _, name := range []string{manifest.Artifacts.Pages, manifest.Artifacts.Chunks, manifest.Artifacts.Index} {
		_, _ = sum.Write(readIdentityFixture(t, filepath.Join(directory, name)))
	}
	manifest.Digest = hex.EncodeToString(sum.Sum(nil))
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
