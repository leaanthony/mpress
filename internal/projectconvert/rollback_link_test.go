package projectconvert

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

// injectSourceCreationDuringRollback replaces the readFile observation seam so
// that, while rollback is deciding whether a consumed source is absent, an
// author creates a new leaf at that path before the absence is reported.
func injectSourceCreationDuringRollback(t *testing.T, ops *conversionOps, source string, create func() error) *bool {
	t.Helper()
	injected := false
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		data, err := files.ReadFile(name)
		if filepath.Clean(name) == source && !injected && errors.Is(err, os.ErrNotExist) {
			injected = true
			if err := create(); err != nil {
				t.Fatalf("inject source creation: %v", err)
			}
		}
		return data, err
	}
	return &injected
}

func TestConversionRollbackKeepsNewRegularWorkAtConsumedSource(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	aPath := filepath.Join(root, "content", "a.md")
	bPath := filepath.Join(root, "content", "b.md")
	a := writeConversionSource(t, aPath, 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, bPath, 0640, "# Beta\n\nOriginal beta.\n")
	newWork := []byte("# New work\n\nThe author recreated this source during rollback.\n")
	removeFailure := errors.New("source removal failed")
	ops := defaultConversionOps
	ops.renameLexical = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == bPath {
			return removeFailure
		}
		return f.RenameLexical(old, new)
	}
	injected := injectSourceCreationDuringRollback(t, &ops, aPath, func() error {
		if err := os.WriteFile(aPath, newWork, 0640); err != nil {
			return err
		}
		return os.Chmod(aPath, 0640)
	})

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !*injected {
		t.Fatal("new source injection during rollback did not occur")
	}
	if !errors.Is(err, removeFailure) {
		t.Fatalf("ordinary removal failure was not retained: %v", err)
	}
	if !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), "destination appeared during rollback") || !strings.Contains(err.Error(), recoveryPrefix) {
		t.Fatalf("collision was not reported as rollback incomplete with its recovery path: %v", err)
	}
	got, readErr := os.ReadFile(aPath)
	if readErr != nil || string(got) != string(newWork) {
		t.Fatalf("newly recreated source bytes changed: %v %q", readErr, got)
	}
	if info, statErr := os.Stat(aPath); statErr != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("newly recreated source mode changed: %v %v", info, statErr)
	}
	assertConversionSource(t, bPath, b, 0640)
	assertNoMPDTargets(t, filepath.Join(root, "content"))

	copies, globErr := filepath.Glob(filepath.Join(root, "content", recoveryPrefix+"*", "content", "a.md.recovery"))
	if globErr != nil || len(copies) != 1 {
		t.Fatalf("original recovery copy was not retained: %v %v", copies, globErr)
	}
	if got, readErr := os.ReadFile(copies[0]); readErr != nil || string(got) != string(a) {
		t.Fatalf("recovery copy lost original bytes: %v %q", readErr, got)
	}
	if info, statErr := os.Stat(copies[0]); statErr != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("recovery copy mode changed: %v %v", info, statErr)
	}
}

func TestConversionRollbackKeepsNewInternalAliasAtConsumedSource(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	aPath := filepath.Join(root, "content", "a.md")
	bPath := filepath.Join(root, "content", "b.md")
	a := writeConversionSource(t, aPath, 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, bPath, 0640, "# Beta\n\nOriginal beta.\n")
	const aliasText = "authored-elsewhere.md"
	removeFailure := errors.New("source removal failed")
	ops := defaultConversionOps
	ops.renameLexical = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == bPath {
			return removeFailure
		}
		return f.RenameLexical(old, new)
	}
	injected := injectSourceCreationDuringRollback(t, &ops, aPath, func() error {
		return os.Symlink(aliasText, aPath)
	})

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !*injected {
		t.Fatal("new alias injection during rollback did not occur")
	}
	if !errors.Is(err, removeFailure) {
		t.Fatalf("ordinary removal failure was not retained: %v", err)
	}
	if !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), "destination appeared during rollback") || !strings.Contains(err.Error(), recoveryPrefix) {
		t.Fatalf("collision was not reported as rollback incomplete with its recovery path: %v", err)
	}
	info, statErr := os.Lstat(aPath)
	if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("newly created internal alias was replaced: %v %v", info, statErr)
	}
	if target, readErr := os.Readlink(aPath); readErr != nil || target != aliasText {
		t.Fatalf("newly created alias text changed: %q %v", target, readErr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "content", aliasText)); !os.IsNotExist(statErr) {
		t.Fatalf("newly created dangling alias target was materialised: %v", statErr)
	}
	assertConversionSource(t, bPath, b, 0640)
	assertNoMPDTargets(t, filepath.Join(root, "content"))

	copies, globErr := filepath.Glob(filepath.Join(root, "content", recoveryPrefix+"*", "content", "a.md.recovery"))
	if globErr != nil || len(copies) != 1 {
		t.Fatalf("original recovery copy was not retained: %v %v", copies, globErr)
	}
	if got, readErr := os.ReadFile(copies[0]); readErr != nil || string(got) != string(a) {
		t.Fatalf("recovery copy lost original bytes: %v %q", readErr, got)
	}
}
