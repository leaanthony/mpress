package projectconvert

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

const (
	recoveryPrefix = config.ConversionRecoveryPrefix
	recoverySuffix = ".recovery"
)

// conversionOps is the bounded set of filesystem steps a conversion needs.
// Production always uses defaultConversionOps; tests replace individual steps
// to reproduce failures deterministically without process-global hooks.
type conversionOps struct {
	write         func(*projectfs.FS, string, []byte, os.FileMode) error
	writeNew      func(*projectfs.FS, string, []byte, os.FileMode) (bool, error)
	renameLexical func(*projectfs.FS, string, string) error
	linkLexical   func(*projectfs.FS, string, string) error
	remove        func(*projectfs.FS, string) error
	readFile      func(*projectfs.FS, string) ([]byte, error)
	lstat         func(*projectfs.FS, string) (os.FileInfo, error)
	readLink      func(*projectfs.FS, string) (string, error)
	stat          func(*projectfs.FS, string) (os.FileInfo, error)
	mkdirTemp     func(*projectfs.FS, string, string) (string, error)
	removeAll     func(*projectfs.FS, string) error
	saveConfig    func(*projectfs.FS, config.Config) error
}

var defaultConversionOps = conversionOps{
	write: func(files *projectfs.FS, name string, data []byte, mode os.FileMode) error {
		return files.WriteAtomicMode(name, data, mode)
	},
	writeNew: func(files *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		return files.WriteNewAtomicMode(name, data, mode)
	},
	renameLexical: func(files *projectfs.FS, old, new string) error { return files.RenameLexical(old, new) },
	linkLexical:   func(files *projectfs.FS, old, new string) error { return files.LinkLexical(old, new) },
	remove:        func(files *projectfs.FS, name string) error { return files.Remove(name) },
	readFile:      func(files *projectfs.FS, name string) ([]byte, error) { return files.ReadFile(name) },
	lstat:         func(files *projectfs.FS, name string) (os.FileInfo, error) { return files.Lstat(name) },
	readLink:      func(files *projectfs.FS, name string) (string, error) { return files.ReadLink(name) },
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
	mode     os.FileMode
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
	if err := j.ops.renameLexical(files, name, backup); err != nil {
		return 0, err
	}
	j.entries = append(j.entries, recoveryEntry{kind: recoveryMoved, files: files, path: name, backup: backup})
	return len(j.entries) - 1, nil
}

func (j *recoveryJournal) recordCreated(files *projectfs.FS, name string, expected []byte, mode os.FileMode) {
	j.entries = append(j.entries, recoveryEntry{kind: recoveryCreated, files: files, path: name, expected: expected, mode: mode})
}

func (j *recoveryJournal) recordWritten(index int, expected []byte) {
	j.entries[index].expected = expected
}

func sourceChanged(path, reason string) error {
	return fmt.Errorf("source %s %s after planning; refusing to remove it", path, reason)
}

// confirmMovedSource checks the leaf captured by moveAside against the
// planning snapshot. Regular files are checked after the move to detect
// changes captured by it. Relative aliases can become dangling when moved;
// their link text is checked here and their content before source consumption.
func (j *recoveryJournal) confirmMovedSource(index int, item conversion) error {
	entry := j.entries[index]
	info, err := j.ops.lstat(entry.files, entry.backup)
	if err != nil {
		return fmt.Errorf("inspect moved source %s: %w", entry.path, err)
	}
	if item.sourceIsLink {
		if info.Mode()&os.ModeSymlink == 0 {
			return sourceChanged(entry.path, "is no longer a symbolic link")
		}
		target, err := j.ops.readLink(entry.files, entry.backup)
		if err != nil {
			return fmt.Errorf("inspect moved source alias %s: %w", entry.path, err)
		}
		if target != item.sourceAlias {
			return sourceChanged(entry.path, "symbolic link target changed")
		}
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return sourceChanged(entry.path, "was replaced by a symbolic link")
	}
	if info.Mode().Perm() != item.mode {
		return sourceChanged(entry.path, "permissions changed")
	}
	data, err := j.ops.readFile(entry.files, entry.backup)
	if err != nil {
		return fmt.Errorf("read moved source %s: %w", entry.path, err)
	}
	if sha256.Sum256(data) != item.sourceDigest {
		return sourceChanged(entry.path, "content changed")
	}
	return nil
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

func (j *recoveryJournal) restoreMoved(entry recoveryEntry) error {
	if entry.expected != nil {
		return j.ops.renameLexical(entry.files, entry.backup, entry.path)
	}
	// The author may recreate a consumed source after the absence check.
	// Keep the backup until cleanup, and never replace that new leaf.
	if err := j.ops.linkLexical(entry.files, entry.backup, entry.path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("destination appeared during rollback, recovery copy retained: %w", err)
		}
		return fmt.Errorf("restore source: %w", err)
	}
	return nil
}

// rollbackCreated captures the current leaf before deciding to remove it.
// Inspecting the captured leaf avoids unlinking a new file at the original name.
func (j *recoveryJournal) rollbackCreated(entry recoveryEntry) error {
	captured, err := j.backupPath(entry.files, entry.path)
	if err != nil {
		return err
	}
	if _, err := j.ops.lstat(entry.files, captured); err == nil {
		return fmt.Errorf("recovery path %s already exists", captured)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := j.ops.renameLexical(entry.files, entry.path, captured); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("capture current output for recovery: %w", err)
	}
	return j.settleCreated(entry, captured)
}

// settleCreated inspects the leaf captured from a path this run created and
// discards it only when it is still the verified regular generated output.
// Author bytes, changed permissions, aliases or directories are preserved
// under the original name without replacing a leaf that appeared meanwhile.
func (j *recoveryJournal) settleCreated(entry recoveryEntry, captured string) error {
	info, err := j.ops.lstat(entry.files, captured)
	if err != nil {
		return fmt.Errorf("inspect captured output at %s: %w", captured, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return j.preserveCreated(entry, captured, "was replaced by a symbolic link", true)
	}
	if !info.Mode().IsRegular() {
		return j.preserveCreated(entry, captured, "was replaced by a non-regular file", false)
	}
	if info.Mode().Perm() != entry.mode.Perm() {
		return j.preserveCreated(entry, captured, "permissions changed after conversion", true)
	}
	data, err := j.ops.readFile(entry.files, captured)
	if err != nil {
		return fmt.Errorf("read captured output at %s: %w", captured, err)
	}
	if !bytes.Equal(data, entry.expected) {
		return j.preserveCreated(entry, captured, "changed after conversion", true)
	}
	if err := j.ops.remove(entry.files, captured); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("discard unchanged output copy at %s: %w", captured, err)
	}
	return nil
}

// preserveCreated restores a captured author leaf without replacing anything.
// A leaf that cannot be relinked (a directory) or whose original name was
// recreated stays at the explicit recovery path named in the error.
func (j *recoveryJournal) preserveCreated(entry recoveryEntry, captured, reason string, linkable bool) error {
	if !linkable {
		return fmt.Errorf("%s; preserved at %s", reason, captured)
	}
	if err := j.ops.linkLexical(entry.files, captured, entry.path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s; original name recreated during rollback, preserved at %s: %w", reason, captured, err)
		}
		return fmt.Errorf("%s; preserved at %s: %w", reason, captured, err)
	}
	if err := j.ops.remove(entry.files, captured); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s; restored to the original name but recovery copy retained at %s: %w", reason, captured, err)
	}
	return fmt.Errorf("%s; restored to the original name", reason)
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
			if err := j.restoreMoved(entry); err != nil {
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
			if err := j.rollbackCreated(entry); err != nil {
				problems = append(problems, fmt.Errorf("%s: %w", entry.detail(), err))
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
