package projectconvert

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

// requireOpenFileRename skips a holder whose filesystem cannot rename a file
// that is still open, which is the precondition for an author descriptor to
// follow a source consumed by conversion.
func requireOpenFileRename(t *testing.T) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "open.md")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("cannot open a file for the descriptor check: %v", err)
	}
	defer func() { _ = file.Close() }()
	if err := os.Rename(path, filepath.Join(directory, "moved.md")); err != nil {
		t.Skipf("filesystem cannot rename an open file: %v", err)
	}
}

// requireSourceRecoveryCopy names the single recovery copy of a consumed
// project document.
func requireSourceRecoveryCopy(t *testing.T, root, name string) string {
	t.Helper()
	copies, err := filepath.Glob(filepath.Join(root, "content", recoveryPrefix+"*", "content", name+recoverySuffix))
	if err != nil {
		t.Fatal(err)
	}
	if len(copies) != 1 {
		t.Fatalf("expected one recovery copy of %s, got %v", name, copies)
	}
	return copies[0]
}

type openSourceFixture struct {
	cfg      config.Config
	root     string
	files    *projectfs.FS
	path     string
	original []byte
	note     []byte
	saved    []byte
	file     *os.File
}

func newOpenSourceFixture(t *testing.T) openSourceFixture {
	t.Helper()
	requireOpenFileRename(t)
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/page.md"
	root, files := openConversionProject(t, &cfg)
	path := filepath.Join(root, "content", "page.md")
	original := writeConversionSource(t, path, 0600, "# Page\n\nOriginal page.\n")
	// The author already has the source open for append before conversion
	// starts, so the descriptor follows the inode into recovery.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	note := []byte("Author save completed before cleanup.\n")
	return openSourceFixture{
		cfg:      cfg,
		root:     root,
		files:    files,
		path:     path,
		original: original,
		note:     note,
		saved:    append(append([]byte{}, original...), note...),
		file:     file,
	}
}

func TestConversionCommitKeepsSourceSavedThroughOpenDescriptor(t *testing.T) {
	fixture := newOpenSourceFixture(t)
	var recoveryPath string
	ops := saveConfigAtFinalPublication(t, func() error {
		if _, err := fixture.file.Write(fixture.note); err != nil {
			return err
		}
		if err := fixture.file.Sync(); err != nil {
			return err
		}
		if err := fixture.file.Close(); err != nil {
			return err
		}
		recoveryPath = requireSourceRecoveryCopy(t, fixture.root, "page.md")
		inRecovery, err := os.ReadFile(recoveryPath)
		if err != nil {
			return err
		}
		if !bytes.Equal(inRecovery, fixture.saved) {
			t.Fatalf("completed save was not in recovery before cleanup: %q", inRecovery)
		}
		return nil
	})

	result, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
	if err == nil {
		t.Fatal("completed source save was discarded silently")
	}
	for _, want := range []string{"refusing to discard recovery copies", "content changed", recoveryPrefix, recoveryPath} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic missing %q: %v", want, err)
		}
	}
	if result.Count != 1 {
		t.Fatalf("conversion result changed: %+v", result)
	}
	got, readErr := os.ReadFile(recoveryPath)
	if readErr != nil || !bytes.Equal(got, fixture.saved) {
		t.Fatalf("author save did not survive cleanup: %v %q", readErr, got)
	}
	info, statErr := os.Lstat(recoveryPath)
	if statErr != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("author save mode was not retained: %v %v", info, statErr)
	}

	// The retained copy is usable author work: retry from it after removing
	// the target this completed run published.
	target := filepath.Join(fixture.root, "content", "page.mpd")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(recoveryPath, fixture.path); err != nil {
		t.Fatal(err)
	}
	if _, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, defaultConversionOps); err != nil {
		t.Fatalf("retry from the retained recovery copy failed: %v", err)
	}
	converted, readErr := os.ReadFile(target)
	if readErr != nil || !strings.Contains(string(converted), "Author save completed before cleanup.") {
		t.Fatalf("retry did not convert the saved bytes: %v %q", readErr, converted)
	}
}

func TestConversionCommitKeepsSourceChangedBeforeCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit changes require Unix mode semantics")
	}
	fixture := newOpenSourceFixture(t)
	var recoveryPath string
	ops := saveConfigAtFinalPublication(t, func() error {
		recoveryPath = requireSourceRecoveryCopy(t, fixture.root, "page.md")
		return os.Chmod(recoveryPath, 0644)
	})

	_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
	if err == nil || !strings.Contains(err.Error(), "permissions changed") || !strings.Contains(err.Error(), recoveryPrefix) {
		t.Fatalf("changed source mode was not reported with its recovery path: %v", err)
	}
	info, statErr := os.Stat(recoveryPath)
	if statErr != nil || info.Mode().Perm() != 0644 {
		t.Fatalf("changed source mode was not retained: %v %v", info, statErr)
	}
}

func TestConversionCommitRefusesCleanupAfterRecoveryReadError(t *testing.T) {
	fixture := newOpenSourceFixture(t)
	readFailure := errors.New("recovery copy read failed")
	var recoveryPath string
	armed := false
	ops := saveConfigAtFinalPublication(t, func() error {
		recoveryPath = requireSourceRecoveryCopy(t, fixture.root, "page.md")
		armed = true
		return nil
	})
	readFile := ops.readFile
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		if armed && filepath.Clean(name) == filepath.Clean(recoveryPath) {
			return nil, readFailure
		}
		return readFile(files, name)
	}

	_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
	if !errors.Is(err, readFailure) || !strings.Contains(err.Error(), "refusing to discard recovery copies") || !strings.Contains(err.Error(), recoveryPath) {
		t.Fatalf("uninspectable recovery was treated as safe to discard: %v", err)
	}
	if _, statErr := os.Stat(recoveryPath); statErr != nil {
		t.Fatalf("recovery evidence was removed after a read error: %v", statErr)
	}
}

func TestConversionCommitRefusesCleanupWhenRecoveryCopyMissing(t *testing.T) {
	fixture := newOpenSourceFixture(t)
	var recoveryPath string
	ops := saveConfigAtFinalPublication(t, func() error {
		recoveryPath = requireSourceRecoveryCopy(t, fixture.root, "page.md")
		return os.Remove(recoveryPath)
	})

	_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
	if err == nil || !strings.Contains(err.Error(), "refusing to discard recovery copies") || !strings.Contains(err.Error(), recoveryPath) {
		t.Fatalf("missing recovery copy was treated as safe to discard: %v", err)
	}
}

func TestConversionCommitCheckKeepsOrdinaryOutcomes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := config.Default()
		root, files := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0644, "# Alpha\n")
		result, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, defaultConversionOps)
		if err != nil || result.Count != 1 {
			t.Fatalf("conversion failed: %v %+v", err, result)
		}
		if copies := recoveryCopies(t, root); len(copies) != 0 {
			t.Fatalf("successful conversion retained recovery copies: %v", copies)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		cfg := config.Default()
		root, files := openConversionProject(t, &cfg)
		a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n")
		b := writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0640, "# Beta\n")
		removeFailure := errors.New("source removal failed")
		ops := defaultConversionOps
		ops.renameLexical = func(f *projectfs.FS, old, new string) error {
			if filepath.Clean(old) == filepath.Join(root, "content", "b.md") {
				return removeFailure
			}
			return f.RenameLexical(old, new)
		}
		if _, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops); !errors.Is(err, removeFailure) {
			t.Fatalf("source removal failure was not reported: %v", err)
		}
		assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
		assertConversionSource(t, filepath.Join(root, "content", "b.md"), b, 0640)
		if copies := recoveryCopies(t, root); len(copies) != 0 {
			t.Fatalf("successful rollback retained recovery copies: %v", copies)
		}
	})

	t.Run("alias", func(t *testing.T) {
		requireSymlinks(t)
		cfg := config.Default()
		root, files := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "real.md"), 0644, "# Real\n")
		if err := os.Symlink("real.md", filepath.Join(root, "content", "z-alias.md")); err != nil {
			t.Fatal(err)
		}
		result, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, defaultConversionOps)
		if err != nil || result.Count != 2 {
			t.Fatalf("alias conversion failed: %v %+v", err, result)
		}
		if copies := recoveryCopies(t, root); len(copies) != 0 {
			t.Fatalf("successful alias conversion retained recovery copies: %v", copies)
		}
	})
}
