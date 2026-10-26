package projectconvert

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

const (
	recoveryPrefix = ".mpress-conversion-recovery-"
	recoverySuffix = ".recovery"
)

// conversionOps is the bounded set of filesystem steps a conversion needs.
// Production always uses defaultConversionOps; tests replace individual steps
// to reproduce failures deterministically without process-global hooks.
type conversionOps struct {
	write         func(*projectfs.FS, string, []byte, os.FileMode) error
	rename        func(*projectfs.FS, string, string) error
	renameLexical func(*projectfs.FS, string, string) error
	remove        func(*projectfs.FS, string) error
	readFile      func(*projectfs.FS, string) ([]byte, error)
	stat          func(*projectfs.FS, string) (os.FileInfo, error)
	mkdirTemp     func(*projectfs.FS, string, string) (string, error)
	removeAll     func(*projectfs.FS, string) error
	saveConfig    func(*projectfs.FS, config.Config) error
}

var defaultConversionOps = conversionOps{
	write: func(files *projectfs.FS, name string, data []byte, mode os.FileMode) error {
		return files.WriteAtomicMode(name, data, mode)
	},
	rename:        func(files *projectfs.FS, old, new string) error { return files.Rename(old, new) },
	renameLexical: func(files *projectfs.FS, old, new string) error { return files.RenameLexical(old, new) },
	remove:        func(files *projectfs.FS, name string) error { return files.Remove(name) },
	readFile:      func(files *projectfs.FS, name string) ([]byte, error) { return files.ReadFile(name) },
	stat:          func(files *projectfs.FS, name string) (os.FileInfo, error) { return files.Stat(name) },
	mkdirTemp: func(files *projectfs.FS, parent, prefix string) (string, error) {
		return files.MkdirTemp(parent, prefix)
	},
	removeAll:  func(files *projectfs.FS, name string) error { return files.RemoveAll(name) },
	saveConfig: config.SaveRoot,
}

type recoveryKind int

const (
	recoveryCreated recoveryKind = iota
	recoveryMoved
)

type fileObservation int

const (
	fileAbsent fileObservation = iota
	fileUnchanged
	fileChanged
)

type recoveryEntry struct {
	kind     recoveryKind
	files    *projectfs.FS
	path     string
	backup   string
	expected []byte
}

func (e recoveryEntry) detail() string {
	if e.backup == "" {
		return e.path
	}
	return fmt.Sprintf("%s (recovery copy %s)", e.path, e.backup)
}

// recoveryJournal moves changed files into exclusively owned recovery
// directories before changing them, so an ordinary later error can restore
// exact bytes and modes. A failed rollback leaves those copies in place.
type recoveryJournal struct {
	ops              conversionOps
	contentFiles     *projectfs.FS
	contentDirectory string
	backups          map[*projectfs.FS]string
	entries          []recoveryEntry
}

func newRecoveryJournal(ops conversionOps, contentFiles *projectfs.FS, contentDirectory string) *recoveryJournal {
	return &recoveryJournal{ops: ops, contentFiles: contentFiles, contentDirectory: contentDirectory, backups: map[*projectfs.FS]string{}}
}

func (j *recoveryJournal) backupRoot(files *projectfs.FS) (string, error) {
	if root, ok := j.backups[files]; ok {
		return root, nil
	}
	parent := "."
	if files == j.contentFiles {
		parent = j.contentDirectory
	}
	root, err := j.ops.mkdirTemp(files, parent, recoveryPrefix)
	if err != nil {
		return "", fmt.Errorf("allocate conversion recovery directory: %w", err)
	}
	j.backups[files] = root
	return root, nil
}

func (j *recoveryJournal) backupPath(files *projectfs.FS, name string) (string, error) {
	// Backup names stay lexical so distinct internal aliases that resolve to
	// one file keep separate recovery copies.
	if !filepath.IsAbs(name) {
		name = filepath.Join(files.Project(), name)
	}
	rel, err := filepath.Rel(files.Project(), filepath.Clean(name))
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("recovery path leaves the conversion boundary: %s", name)
	}
	root, err := j.backupRoot(files)
	if err != nil {
		return "", err
	}
	backup := filepath.Join(root, rel+recoverySuffix)
	if err := files.MkdirAll(filepath.Dir(backup), 0700); err != nil {
		return "", err
	}
	return backup, nil
}

// moveAside relocates a file that must be changed or removed and returns the
// entry index so a later successful write can record its expected content.
func (j *recoveryJournal) moveAside(files *projectfs.FS, name string) (int, error) {
	backup, err := j.backupPath(files, name)
	if err != nil {
		return 0, err
	}
	if err := j.ops.rename(files, name, backup); err != nil {
		return 0, err
	}
	j.entries = append(j.entries, recoveryEntry{kind: recoveryMoved, files: files, path: name, backup: backup})
	return len(j.entries) - 1, nil
}

func (j *recoveryJournal) recordCreated(files *projectfs.FS, name string, expected []byte) {
	j.entries = append(j.entries, recoveryEntry{kind: recoveryCreated, files: files, path: name, expected: expected})
}

func (j *recoveryJournal) recordWritten(index int, expected []byte) {
	j.entries[index].expected = expected
}

// observe compares a current path with the content this operation wrote.
// A nil expectation means the path must be absent for a clean rollback.
func (j *recoveryJournal) observe(files *projectfs.FS, name string, expected []byte) (fileObservation, error) {
	data, err := j.ops.readFile(files, name)
	if errors.Is(err, os.ErrNotExist) {
		return fileAbsent, nil
	}
	if err != nil {
		return fileChanged, err
	}
	if expected != nil && bytes.Equal(data, expected) {
		return fileUnchanged, nil
	}
	return fileChanged, nil
}

// rollback undoes recorded changes in reverse order. Every entry is attempted.
// A path changed by another writer is left untouched and its recovery copy is
// retained. Reaching an error retains all recovery directories and names them.
func (j *recoveryJournal) rollback() error {
	var problems []error
	for i := len(j.entries) - 1; i >= 0; i-- {
		entry := j.entries[i]
		observation, err := j.observe(entry.files, entry.path, entry.expected)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: inspect failed: %w", entry.detail(), err))
			continue
		}
		switch entry.kind {
		case recoveryMoved:
			if observation == fileChanged {
				problems = append(problems, fmt.Errorf("%s: changed after conversion, recovery copy retained", entry.detail()))
				continue
			}
			if err := j.ops.renameLexical(entry.files, entry.backup, entry.path); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", entry.detail(), err))
			}
		case recoveryCreated:
			if observation == fileAbsent {
				continue
			}
			if observation == fileChanged {
				problems = append(problems, fmt.Errorf("%s: changed after conversion, left in place", entry.path))
				continue
			}
			if err := j.ops.remove(entry.files, entry.path); err != nil && !errors.Is(err, os.ErrNotExist) {
				problems = append(problems, fmt.Errorf("%s: %w", entry.path, err))
			}
		}
	}
	if len(problems) > 0 {
		if roots := j.roots(); len(roots) > 0 {
			return fmt.Errorf("rollback incomplete: %w; recovery copies retained at %s", errors.Join(problems...), strings.Join(roots, ", "))
		}
		return fmt.Errorf("rollback incomplete: %w", errors.Join(problems...))
	}
	if err := j.cleanup(); err != nil {
		return fmt.Errorf("rollback completed but recovery copies were retained at %s: %w", strings.Join(j.roots(), ", "), err)
	}
	return nil
}

func (j *recoveryJournal) commit() error {
	if err := j.cleanup(); err != nil {
		return fmt.Errorf("recovery copies were retained at %s: %w", strings.Join(j.roots(), ", "), err)
	}
	return nil
}

func (j *recoveryJournal) cleanup() error {
	var problems []error
	for files, root := range j.backups {
		if err := j.ops.removeAll(files, root); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", root, err))
		}
	}
	return errors.Join(problems...)
}

func (j *recoveryJournal) roots() []string {
	roots := make([]string, 0, len(j.backups))
	for _, root := range j.backups {
		roots = append(roots, root)
	}
	return roots
}
