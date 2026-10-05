package version

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

func mount(project, output string) (int, error) {
	files, cfg, err := loadProjectConfig(project)
	if err != nil {
		return 0, err
	}
	defer files.Close()
	if !cfg.Version.Enabled {
		return 0, nil
	}
	if _, err := cfg.SafeOutputPath(project, output); err != nil {
		return 0, err
	}
	if err := cfg.SafeVersionPath(project, cfg.ArtifactsPath(project)); err != nil {
		return 0, err
	}
	if err := files.MkdirAll(output, 0755); err != nil {
		return 0, err
	}
	destination, err := files.Sub(output)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	store, err := files.Sub(cfg.ArtifactsPath(project))
	if os.IsNotExist(err) {
		return 0, writeMountedIndex(destination, ".", cfg.Version.Current, nil)
	}
	if err != nil {
		return 0, err
	}
	defer store.Close()
	unlock, err := lockVersionRead(store)
	if err != nil {
		return 0, err
	}
	defer unlock()
	labels, err := listSnapshots(store)
	if err != nil {
		return 0, err
	}
	for n, label := range labels {
		if err := mountSnapshot(project, store, destination, cfg, label, labels); err != nil {
			return n, err
		}
	}
	return len(labels), writeMountedIndex(destination, ".", cfg.Version.Current, labels)
}

// Keep one pinned snapshot open for both verification and copying, and a narrow
// destination root for every copied file and HTML rewrite.
func mountSnapshot(project string, store, output *projectfs.FS, cfg config.Config, label string, labels []string) error {
	if err := cfg.SafeVersionPath(project, filepath.Join(cfg.ArtifactsPath(project), label)); err != nil {
		return err
	}
	snapshot, err := store.Sub(label)
	if err != nil {
		return err
	}
	defer snapshot.Close()
	if err := verifyPinnedSnapshot(snapshot, label); err != nil {
		return err
	}
	path := filepath.Join("versions", label)
	if err := output.MkdirAll(path, 0755); err != nil {
		return err
	}
	destination, err := output.Sub(path)
	if err != nil {
		return err
	}
	defer destination.Close()
	if err := copyMountedSnapshot(snapshot, destination); err != nil {
		return err
	}
	return rewriteMountedSnapshot(destination, label, cfg.Version.Current, labels)
}

func writeMountedIndex(files *projectfs.FS, output, current string, labels []string) error {
	data, err := json.MarshalIndent(map[string]any{"current": current, "versions": labels}, "", "  ")
	if err != nil {
		return err
	}
	return files.WriteAtomic(filepath.Join(output, "versions/versions.json"), data)
}

func copyMountedSnapshot(snapshot, destination *projectfs.FS) error {
	return snapshot.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || path == "." {
			return walkErr
		}
		if path == "mpress-version.json" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return destination.MkdirAll(path, 0755)
		}
		file, err := snapshot.Open(path)
		if err != nil {
			return err
		}
		err = destination.WriteFrom(path, file)
		closeErr := file.Close()
		if err != nil {
			return fmt.Errorf("mount %s: %w", path, err)
		}
		return closeErr
	})
}

func rewriteMountedSnapshot(files *projectfs.FS, label, current string, labels []string) error {
	return files.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return walkErr
		}
		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}
		rewritten := rewriteRootURLs(string(data), label)
		rewritten = replaceVersionMenu(rewritten, versionRoute(filepath.ToSlash(path)), label, current, labels)
		return files.WriteAtomic(path, []byte(rewritten))
	})
}
