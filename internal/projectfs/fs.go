// Package projectfs confines project file operations to the selected directory.
package projectfs

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ErrOutside = errors.New("path leaves the project")

type FS struct {
	project string
	root    *os.Root
}

func Open(project string) (*FS, error) {
	canonical, err := CanonicalPath(project)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(canonical)
	if err != nil {
		return nil, err
	}
	return &FS{project: canonical, root: root}, nil
}

func (f *FS) Close() error { return f.root.Close() }

// Sub pins a directory inside this boundary and confines subsequent operations
// to it. A snapshot manifest cannot use an internal link to read another part
// of the project through this narrower root.
func (f *FS) Sub(name string) (*FS, error) {
	rel, err := f.Relative(name)
	if err != nil {
		return nil, err
	}
	root, err := f.root.OpenRoot(rel)
	if err != nil {
		return nil, err
	}
	return &FS{project: filepath.Join(f.project, rel), root: root}, nil
}

func (f *FS) Open(name string) (*os.File, error) {
	rel, err := f.readPath(name, true)
	if err != nil {
		return nil, err
	}
	return f.root.Open(rel)
}

// Relative validates physical containment, including missing descendants.
// Subsequent IO uses Root, so an external link substituted after this check
// cannot redirect the operation. Resolve first to retain safe absolute links
// within the project, which Root itself does not follow.
func (f *FS) Relative(name string) (string, error) {
	if !filepath.IsAbs(name) {
		name = filepath.Join(f.project, name)
	}
	canonical, err := CanonicalPath(name)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(f.project, canonical)
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("%w: %s", ErrOutside, name)
	}
	return rel, nil
}

func (f *FS) ReadFile(name string) ([]byte, error) {
	rel, err := f.readPath(name, false)
	if err != nil {
		return nil, err
	}
	return f.root.ReadFile(rel)
}

func (f *FS) readPath(name string, allowDirectory bool) (string, error) {
	rel, err := f.Relative(name)
	if err != nil {
		return "", err
	}
	info, err := f.root.Stat(rel)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() && !(allowDirectory && info.IsDir()) {
		return "", fmt.Errorf("not a regular project file: %s", name)
	}
	return rel, nil
}

func (f *FS) Stat(name string) (fs.FileInfo, error) {
	rel, err := f.Relative(name)
	if err != nil {
		return nil, err
	}
	return f.root.Stat(rel)
}

func (f *FS) ReadDir(name string) ([]os.DirEntry, error) {
	rel, err := f.Relative(name)
	if err != nil {
		return nil, err
	}
	return fs.ReadDir(f.root.FS(), filepath.ToSlash(rel))
}

func (f *FS) MkdirAll(name string, mode fs.FileMode) error {
	rel, err := f.Relative(name)
	if err != nil {
		return err
	}
	return f.root.MkdirAll(rel, mode)
}

func (f *FS) Chmod(name string, mode fs.FileMode) error {
	rel, err := f.Relative(name)
	if err != nil {
		return err
	}
	return f.root.Chmod(rel, mode)
}

// MkdirTemp exclusively allocates a private directory through the project root.
func (f *FS) MkdirTemp(parent, prefix string) (string, error) {
	if strings.ContainsAny(prefix, `/\`) {
		return "", errors.New("temporary directory prefix must not contain a separator")
	}
	rel, err := f.Relative(parent)
	if err != nil {
		return "", err
	}
	if err := f.root.MkdirAll(rel, 0755); err != nil {
		return "", err
	}
	for range 100 {
		name := filepath.Join(rel, prefix+rand.Text())
		if err := f.root.Mkdir(name, 0700); err == nil {
			return filepath.Join(f.project, name), nil
		} else if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", errors.New("cannot allocate a unique project temporary directory")
}

// WalkDir enumerates through the pinned project root, retaining logical names.
// As with filepath.WalkDir, directory symlinks are not followed.
func (f *FS) WalkDir(name string, visit fs.WalkDirFunc) error {
	rel, err := f.Relative(name)
	if err != nil {
		return err
	}
	return fs.WalkDir(f.root.FS(), filepath.ToSlash(rel), func(path string, entry fs.DirEntry, err error) error {
		tail, relErr := filepath.Rel(rel, filepath.FromSlash(path))
		if relErr != nil {
			return relErr
		}
		return visit(filepath.Join(name, tail), entry, err)
	})
}

// destination validates a final target but retains its leaf name. Atomic
// replacement and removal unlink a final internal symlink; they must not
// delete or rewrite the unrelated file it points to.
func (f *FS) destination(name string) (parent, leaf string, err error) {
	rel, err := f.Relative(name)
	if err != nil {
		return "", "", err
	}
	if rel == "." {
		return "", "", errors.New("cannot modify the project root")
	}
	parent, err = f.Relative(filepath.Dir(name))
	return parent, filepath.Base(name), err
}

func (f *FS) Remove(name string) error {
	parent, leaf, err := f.destination(name)
	if err != nil {
		return err
	}
	return f.root.Remove(filepath.Join(parent, leaf))
}

func (f *FS) RemoveAll(name string) error {
	parent, leaf, err := f.destination(name)
	if err != nil {
		return err
	}
	return f.root.RemoveAll(filepath.Join(parent, leaf))
}

// Rename retains both leaf names and performs the move through the pinned root.
func (f *FS) Rename(old, new string) error {
	oldParent, oldLeaf, err := f.destination(old)
	if err != nil {
		return err
	}
	newParent, newLeaf, err := f.destination(new)
	if err != nil {
		return err
	}
	return f.root.Rename(filepath.Join(oldParent, oldLeaf), filepath.Join(newParent, newLeaf))
}

// WriteAtomic uses a pinned parent and an exclusively created temporary file.
// Neither directory creation, temporary writes nor promotion use ambient paths.
func (f *FS) WriteAtomic(name string, data []byte) error {
	return f.WriteAtomicMode(name, data, 0644)
}

// WriteAtomicMode preserves the requested permissions before promotion.
func (f *FS) WriteAtomicMode(name string, data []byte, mode fs.FileMode) error {
	return f.writeAtomic(name, func(file *os.File) error { _, err := file.Write(data); return err }, mode.Perm(), true)
}

// WriteCache atomically replaces recomputable data without a durability sync.
func (f *FS) WriteCache(name string, data []byte) error {
	return f.writeAtomic(name, func(file *os.File) error { _, err := file.Write(data); return err }, 0600, false)
}

// WriteFrom atomically copies a stream without a durability sync. It supports
// owned staging files without allocating the entire input in memory.
func (f *FS) WriteFrom(name string, source io.Reader) error {
	return f.writeAtomic(name, func(file *os.File) error { _, err := io.Copy(file, source); return err }, 0644, false)
}

func (f *FS) writeAtomic(name string, write func(*os.File) error, mode fs.FileMode, durable bool) error {
	parent, leaf, err := f.destination(name)
	if err != nil {
		return err
	}
	if err := f.root.MkdirAll(parent, 0755); err != nil {
		return err
	}
	dir, err := f.root.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer dir.Close()
	tmpName := ".mpress-write-" + rand.Text()
	tmp, err := dir.OpenFile(tmpName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	promoted := false
	defer func() {
		if !promoted {
			_ = dir.Remove(tmpName)
		}
	}()
	if err = write(tmp); err == nil && mode != 0600 {
		err = tmp.Chmod(mode)
	}
	if err == nil && durable {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := dir.Rename(tmpName, leaf); err != nil {
		return err
	}
	promoted = true
	return nil
}
