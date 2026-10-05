package version

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/routes"
)

func capture(project, label string, force bool) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	files, cfg, err := loadVersionProject(project)
	if err != nil {
		return err
	}
	defer files.Close()
	source := cfg.OutputPath(project)
	if _, err := cfg.SafeOutputPath(project, source); err != nil {
		return err
	}
	info, err := files.Stat(filepath.Join(source, "index.html"))
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("build output missing or not regular; run mpress build first")
	}
	base := cfg.ArtifactsPath(project)
	if err := cfg.SafeVersionPath(project, filepath.Join(base, label)); err != nil {
		return err
	}
	if err := files.MkdirAll(base, 0755); err != nil {
		return err
	}
	store, err := files.Sub(base)
	if err != nil {
		return err
	}
	defer store.Close()
	lock, err := store.Lock(storeLockFile)
	if err != nil {
		return err
	}
	defer lock.Close()
	if _, err := store.Stat(label); err == nil && !force {
		return fmt.Errorf("version %s already exists", label)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	temporary, err := store.MkdirTemp(".", ".capture-")
	if err != nil {
		return err
	}
	defer store.RemoveAll(temporary)
	staged, err := store.Sub(temporary)
	if err != nil {
		return err
	}
	defer staged.Close()
	checksums, err := copySnapshot(files, source, staged)
	if err != nil {
		return err
	}
	manifest := Manifest{SchemaVersion: 1, Version: label, CreatedAt: time.Now().UTC().Format(time.RFC3339), Files: checksums}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := staged.WriteAtomic("mpress-version.json", data); err != nil {
		return err
	}
	if err := staged.Chmod(".", 0755); err != nil {
		return err
	}
	// Close the child directory before moving it, including on Windows.
	if err := staged.Close(); err != nil {
		return err
	}
	return promoteSnapshot(store, temporary, label, force, store.Rename)
}

func copySnapshot(files *projectfs.FS, source string, staged *projectfs.FS) (map[string]string, error) {
	checksums := map[string]string{}
	err := files.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil || rel == "." {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "versions/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err := routes.Output(rel); err != nil {
			return err
		}
		if entry.IsDir() {
			return staged.MkdirAll(rel, 0755)
		}
		if rel == "mpress-version.json" {
			return nil
		}
		sum, err := copySnapshotFile(files, path, staged, rel)
		if err != nil {
			return fmt.Errorf("capture %s: %w", rel, err)
		}
		checksums[rel] = sum
		return nil
	})
	return checksums, err
}

func copySnapshotFile(files *projectfs.FS, source string, staged *projectfs.FS, name string) (string, error) {
	file, err := files.Open(source)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	err = staged.WriteFrom(name, io.TeeReader(file, hash))
	closeErr := file.Close()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), closeErr
}

// The caller holds the store lock. Keep the old snapshot in an owned backup
// until promotion succeeds, and retain that backup if rollback itself fails.
func promoteSnapshot(store *projectfs.FS, staged, label string, force bool, rename func(string, string) error) error {
	_, err := store.Stat(label)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	backup := ""
	if err == nil {
		if !force {
			return fmt.Errorf("version %s already exists", label)
		}
		backup, err = store.MkdirTemp(".", ".previous-")
		if err != nil {
			return err
		}
		if err := rename(label, filepath.Join(backup, "snapshot")); err != nil {
			store.RemoveAll(backup)
			return err
		}
	}
	if err := rename(staged, label); err != nil {
		if backup != "" {
			if restoreErr := rename(filepath.Join(backup, "snapshot"), label); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("rollback failed; previous snapshot retained at %s: %w", filepath.Join(backup, "snapshot"), restoreErr))
			}
			store.RemoveAll(backup)
		}
		return err
	}
	if backup != "" {
		if err := store.RemoveAll(backup); err != nil {
			return fmt.Errorf("version captured, but previous snapshot cleanup failed at %s: %w", backup, err)
		}
	}
	return nil
}
