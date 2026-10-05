package version

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/routes"
)

var errMissingStore = errors.New("version artifacts directory missing")

func validateLabel(label string) error {
	if !labelRE.MatchString(label) || routes.Component(label) != nil {
		return fmt.Errorf("invalid version label %q", label)
	}
	return nil
}

func loadProjectConfig(project string) (*projectfs.FS, config.Config, error) {
	files, err := projectfs.Open(project)
	if err != nil {
		return nil, config.Config{}, err
	}
	cfg, err := config.LoadWithReadFile(project, files.ReadFile)
	if err != nil {
		files.Close()
		return nil, cfg, err
	}
	return files, cfg, nil
}

func loadVersionProject(project string) (*projectfs.FS, config.Config, error) {
	files, cfg, err := loadProjectConfig(project)
	if err != nil {
		return nil, cfg, err
	}
	if err := cfg.SafeVersionPath(project, cfg.ArtifactsPath(project)); err != nil {
		files.Close()
		return nil, cfg, err
	}
	return files, cfg, nil
}

// Legacy read-only stores may predate the stable lock file. Reading them must
// not require write permission; stores written by Capture always have a lock.
func lockVersionRead(store *projectfs.FS) (func(), error) {
	lock, err := store.LockShared(storeLockFile)
	if errors.Is(err, os.ErrPermission) {
		if _, statErr := store.Stat(storeLockFile); os.IsNotExist(statErr) {
			return func() {}, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return func() { lock.Close() }, nil
}

func openVersionStore(project string) (*projectfs.FS, config.Config, error) {
	files, cfg, err := loadVersionProject(project)
	if err != nil {
		return nil, cfg, err
	}
	defer files.Close()
	store, err := files.Sub(cfg.ArtifactsPath(project))
	if os.IsNotExist(err) {
		err = fmt.Errorf("%w: %w", errMissingStore, err)
	}
	return store, cfg, err
}

const storeLockFile = ".mpress-version-lock"

func listSnapshots(files *projectfs.FS) ([]string, error) {
	entries, err := files.ReadDir(".")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() && validateLabel(e.Name()) == nil {
			complete, err := snapshotHasManifest(files, e.Name())
			if err != nil {
				return nil, err
			}
			if complete {
				out = append(out, e.Name())
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// Listing requires a manifest file, not a full checksum verification. Keep
// incomplete snapshots hidden, while diagnosing unsafe or unreadable inputs.
func snapshotHasManifest(files *projectfs.FS, path string) (bool, error) {
	snapshot, err := files.Sub(path)
	if err != nil {
		return false, err
	}
	defer snapshot.Close()
	info, err := snapshot.Stat("mpress-version.json")
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}
