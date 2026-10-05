package projectfs

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock holds an advisory exclusive process lock until the returned file closes.
// The stable lock file is never truncated or removed; replacing it could let
// separate processes acquire locks on different inodes for the same store.
func (f *FS) Lock(name string) (*os.File, error) {
	return f.lock(name, true)
}

// LockShared coordinates readers with an exclusive writer. Multiple readers
// may hold the stable lock at once; closing the returned file releases it.
func (f *FS) LockShared(name string) (*os.File, error) {
	return f.lock(name, false)
}

func (f *FS) lock(name string, exclusive bool) (*os.File, error) {
	parent, leaf, err := f.destination(name)
	if err != nil {
		return nil, err
	}
	rel := filepath.Join(parent, leaf)
	file, err := f.openLockFile(rel)
	if err != nil {
		return nil, err
	}
	if err := lockFile(file, exclusive); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func (f *FS) openLockFile(name string) (*os.File, error) {
	for range 100 {
		info, err := f.root.Lstat(name)
		if os.IsNotExist(err) {
			file, err := f.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
			if os.IsExist(err) {
				continue
			}
			return file, err
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("not a regular project lock file: %s", name)
		}
		// Exclusive locks need read access, not permission to alter this file.
		return f.root.Open(name)
	}
	return nil, fmt.Errorf("cannot open a stable project lock file: %s", name)
}
