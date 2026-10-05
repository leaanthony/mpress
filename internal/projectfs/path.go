package projectfs

import (
	"errors"
	"os"
	"path/filepath"
)

// CanonicalPath resolves existing ancestors and symlinks without requiring the
// final path to exist. A dangling symlink remains an error.
func CanonicalPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var missing []string
	for {
		_, err := os.Lstat(absolute)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(absolute)
			if err != nil {
				return "", err
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return "", err
		}
		missing = append(missing, filepath.Base(absolute))
		absolute = parent
	}
}
