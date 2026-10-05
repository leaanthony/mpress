package version

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// Verification uses one pinned snapshot for its manifest, checksums and
// enumeration; callers must keep that boundary open throughout the operation.
func verifyPinnedSnapshot(snapshot *projectfs.FS, label string) error {
	data, err := snapshot.ReadFile("mpress-version.json")
	if err != nil {
		return err
	}
	var m Manifest
	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported version manifest schema %d", m.SchemaVersion)
	}
	if m.Version != label {
		return fmt.Errorf("manifest version %q does not match %q", m.Version, label)
	}
	for rel, want := range m.Files {
		path, pathErr := manifestFilePath(".", rel)
		if pathErr != nil {
			return pathErr
		}
		data, err = snapshot.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			return fmt.Errorf("checksum mismatch: %s", rel)
		}
	}
	if err = snapshot.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, relErr := filepath.Rel(".", path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "mpress-version.json" {
			return nil
		}
		if _, ok := m.Files[rel]; !ok {
			return fmt.Errorf("unexpected file: %s", rel)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
