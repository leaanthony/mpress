package projectconvert

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/importer"
	"github.com/leaanthony/mpress/internal/projectfs"
)

// injectTargetChangeAfterObservation wraps the rollback read observation seam
// so that, immediately after the generated target is read, an author change
// appears at the same path. The read itself still returns the observed bytes.
func injectTargetChangeAfterObservation(t *testing.T, ops *conversionOps, target, observed string, change func() error) *bool {
	t.Helper()
	injected := false
	read := ops.readFile
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		data, err := read(files, name)
		if filepath.Clean(name) == target && !injected && err == nil && string(data) == observed {
			injected = true
			if err := change(); err != nil {
				t.Fatalf("inject target change: %v", err)
			}
		}
		return data, err
	}
	return &injected
}

type targetCleanupFixture struct {
	cfg          config.Config
	root         string
	files        *projectfs.FS
	a, b         []byte
	aPath, bPath string
	target       string
	removeErr    error
}

// setupTargetCleanupFailure converts a.md/b.md and fails while consuming b, so
// rollback reaches the generated a.mpd target and the original a.md source.
func setupTargetCleanupFailure(t *testing.T) *targetCleanupFixture {
	t.Helper()
	fixture := &targetCleanupFixture{cfg: config.Default(), removeErr: errors.New("source removal failed")}
	fixture.root, fixture.files = openConversionProject(t, &fixture.cfg)
	fixture.aPath = filepath.Join(fixture.root, "content", "a.md")
	fixture.bPath = filepath.Join(fixture.root, "content", "b.md")
	fixture.target = filepath.Join(fixture.root, "content", "a.mpd")
	fixture.a = writeConversionSource(t, fixture.aPath, 0600, "# Alpha\n\nOriginal alpha.\n")
	fixture.b = writeConversionSource(t, fixture.bPath, 0640, "# Beta\n\nOriginal beta.\n")
	return fixture
}

func (f *targetCleanupFixture) ops() conversionOps {
	ops := defaultConversionOps
	ops.renameLexical = func(files *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == f.bPath {
			return f.removeErr
		}
		return files.RenameLexical(old, new)
	}
	return ops
}

func (f *targetCleanupFixture) run(t *testing.T, ops conversionOps) error {
	t.Helper()
	_, err := runConversionWithOps(f.root, &f.cfg, "", "mpd", f.files, f.files, ops)
	return err
}

func (f *targetCleanupFixture) requireDiagnostic(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, f.removeErr) {
		t.Fatalf("original removal failure was not preserved: %v", err)
	}
	if !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), f.target) {
		t.Fatalf("changed generated target was not diagnosed with its path: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(f.root, "content", "b.mpd")); !os.IsNotExist(statErr) {
		t.Fatalf("unchanged generated target survived rollback: %v", statErr)
	}
}

func (f *targetCleanupFixture) requireSourcesRestored(t *testing.T) {
	t.Helper()
	assertConversionSource(t, f.aPath, f.a, 0600)
	assertConversionSource(t, f.bPath, f.b, 0640)
}

func generatedTargetBytes(t *testing.T, source []byte) []byte {
	t.Helper()
	converted, err := importer.MarkdownToMPD(string(source))
	if err != nil {
		t.Fatal(err)
	}
	return []byte(converted)
}

func authorWriteFile(path string, data []byte, mode os.FileMode) error {
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func TestConversionRollbackKeepsLateEditedGeneratedTarget(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("# Alpha\n\nSaved by the author during rollback.\n")
	ops := fixture.ops()
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		return authorWriteFile(fixture.target, author, 0640)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late author save during rollback was not injected")
	}
	fixture.requireDiagnostic(t, runErr)
	assertConversionSource(t, fixture.target, author, 0640)
	fixture.requireSourcesRestored(t)
	copied := recoveryCopies(t, fixture.root)
	if len(copied) != 1 {
		t.Fatalf("original source recovery copy was not retained: %v", copied)
	}
	if got, readErr := os.ReadFile(copied[0]); readErr != nil || string(got) != string(fixture.a) {
		t.Fatalf("retained source recovery copy lost its original bytes: %v %q", readErr, got)
	}
}

func TestConversionRollbackKeepsReplacedGeneratedTarget(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("# Replaced\n\nA new inode replaced the generated output.\n")
	ops := fixture.ops()
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		replacement := fixture.target + ".author"
		if err := authorWriteFile(replacement, author, 0640); err != nil {
			return err
		}
		return os.Rename(replacement, fixture.target)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("generated target replacement was not injected")
	}
	fixture.requireDiagnostic(t, runErr)
	assertConversionSource(t, fixture.target, author, 0640)
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackKeepsGeneratedTargetSymlink(t *testing.T) {
	requireSymlinks(t)
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	const aliasText = "author-elsewhere.md"
	ops := fixture.ops()
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		replacement := fixture.target + ".author"
		if err := os.Symlink(aliasText, replacement); err != nil {
			return err
		}
		return os.Rename(replacement, fixture.target)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("generated target alias replacement was not injected")
	}
	fixture.requireDiagnostic(t, runErr)
	info, statErr := os.Lstat(fixture.target)
	if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("author alias at the generated target was not preserved: %v %v", info, statErr)
	}
	if target, readErr := os.Readlink(fixture.target); readErr != nil || target != aliasText {
		t.Fatalf("author alias text changed: %q %v", target, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(fixture.root, "content", aliasText)); !os.IsNotExist(statErr) {
		t.Fatalf("dangling author alias target was materialised: %v", statErr)
	}
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackKeepsGeneratedTargetPermissionChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not provide these Unix permission distinctions")
	}
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	ops := fixture.ops()
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		return os.Chmod(fixture.target, 0644)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("generated target permission change was not injected")
	}
	fixture.requireDiagnostic(t, runErr)
	assertConversionSource(t, fixture.target, generated, 0644)
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackRetainsReplacedDirectoryTarget(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("author directory content\n")
	ops := fixture.ops()
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		if err := os.Remove(fixture.target); err != nil {
			return err
		}
		if err := os.Mkdir(fixture.target, 0750); err != nil {
			return err
		}
		return authorWriteFile(filepath.Join(fixture.target, "author.txt"), author, 0600)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("generated target directory replacement was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !strings.Contains(runErr.Error(), "rollback incomplete") {
		t.Fatalf("replaced directory was not diagnosed as incomplete rollback: %v", runErr)
	}
	captured, globErr := filepath.Glob(filepath.Join(fixture.root, "content", recoveryPrefix+"*", "content", "a.mpd.recovery"))
	if globErr != nil || len(captured) != 1 {
		t.Fatalf("replaced directory was not retained with a named recovery path: %v %v", captured, globErr)
	}
	if !strings.Contains(runErr.Error(), captured[0]) {
		t.Fatalf("retained directory path was not diagnosed: %v", runErr)
	}
	info, statErr := os.Stat(captured[0])
	if statErr != nil || !info.IsDir() {
		t.Fatalf("retained directory leaf changed type: %v %v", info, statErr)
	}
	assertConversionSource(t, filepath.Join(captured[0], "author.txt"), author, 0600)
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackReportsCaptureReadFailure(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("author bytes captured before an inspection failure\n")
	captureReadFailure := errors.New("captured output read failed")
	ops := fixture.ops()
	read := ops.readFile
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		if strings.Contains(name, recoveryPrefix) && filepath.Base(name) == "a.mpd.recovery" {
			return nil, captureReadFailure
		}
		return read(files, name)
	}
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		return authorWriteFile(fixture.target, author, 0600)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late author save during rollback was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, captureReadFailure) {
		t.Fatalf("capture inspection failure was not reported with the original cause: %v", runErr)
	}
	captured, globErr := filepath.Glob(filepath.Join(fixture.root, "content", recoveryPrefix+"*", "content", "a.mpd.recovery"))
	if globErr != nil || len(captured) != 1 {
		t.Fatalf("captured author bytes were not retained: %v %v", captured, globErr)
	}
	assertConversionSource(t, captured[0], author, 0600)
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackLeavesTargetWhenCaptureRenameFails(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("author bytes that capture could not move\n")
	captureFailure := errors.New("capture rename failed")
	ops := fixture.ops()
	rename := ops.renameLexical
	ops.renameLexical = func(files *projectfs.FS, old, new string) error {
		if strings.Contains(new, recoveryPrefix) && filepath.Base(new) == "a.mpd.recovery" {
			return captureFailure
		}
		return rename(files, old, new)
	}
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		return authorWriteFile(fixture.target, author, 0640)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late author save during rollback was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, captureFailure) {
		t.Fatalf("capture failure was not reported with the original cause: %v", runErr)
	}
	assertConversionSource(t, fixture.target, author, 0640)
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackRetainsTargetWhenRestoreFails(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("author bytes retained at the recovery path\n")
	restoreFailure := errors.New("no-replace restore failed")
	ops := fixture.ops()
	link := ops.linkLexical
	ops.linkLexical = func(files *projectfs.FS, old, new string) error {
		if strings.Contains(old, recoveryPrefix) && filepath.Base(old) == "a.mpd.recovery" {
			return restoreFailure
		}
		return link(files, old, new)
	}
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		return authorWriteFile(fixture.target, author, 0640)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late author save during rollback was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, restoreFailure) {
		t.Fatalf("restore failure was not reported with the original cause: %v", runErr)
	}
	captured, globErr := filepath.Glob(filepath.Join(fixture.root, "content", recoveryPrefix+"*", "content", "a.mpd.recovery"))
	if globErr != nil || len(captured) != 1 {
		t.Fatalf("author bytes were not retained at a named recovery path: %v %v", captured, globErr)
	}
	assertConversionSource(t, captured[0], author, 0640)
	fixture.requireSourcesRestored(t)
}

func TestConversionRollbackNeverReplacesTargetRecreatedAfterCapture(t *testing.T) {
	fixture := setupTargetCleanupFailure(t)
	generated := generatedTargetBytes(t, fixture.a)
	author := []byte("author bytes captured before restoration\n")
	newer := []byte("author bytes saved again at the original name\n")
	ops := fixture.ops()
	link := ops.linkLexical
	ops.linkLexical = func(files *projectfs.FS, old, new string) error {
		if strings.Contains(old, recoveryPrefix) && filepath.Base(old) == "a.mpd.recovery" {
			if err := authorWriteFile(new, newer, 0600); err != nil {
				return err
			}
		}
		return link(files, old, new)
	}
	injected := injectTargetChangeAfterObservation(t, &ops, fixture.target, string(generated), func() error {
		return authorWriteFile(fixture.target, author, 0640)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late author save during rollback was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, os.ErrExist) {
		t.Fatalf("recreated destination collision was not reported: %v", runErr)
	}
	assertConversionSource(t, fixture.target, newer, 0600)
	captured, globErr := filepath.Glob(filepath.Join(fixture.root, "content", recoveryPrefix+"*", "content", "a.mpd.recovery"))
	if globErr != nil || len(captured) != 1 {
		t.Fatalf("captured author bytes were not retained beside the newer save: %v %v", captured, globErr)
	}
	assertConversionSource(t, captured[0], author, 0640)
	fixture.requireSourcesRestored(t)
}
