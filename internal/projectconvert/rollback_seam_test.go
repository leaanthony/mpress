package projectconvert

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/translate"
)

func openConversionProject(t *testing.T, cfg *config.Config) (string, *projectfs.FS) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(cfg.ContentPath(root), 0755); err != nil {
		t.Fatal(err)
	}
	files, err := projectfs.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	return root, files
}

func recoveryCopies(t *testing.T, root string) []string {
	t.Helper()
	var copies []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.Contains(path, recoveryPrefix) {
			copies = append(copies, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return copies
}

func writeTranslationTriplet(t *testing.T, root string, cfg *config.Config) (source, target, encoded []byte, statePath string) {
	t.Helper()
	source = []byte("Read the guide.\n")
	target = []byte("Lisez le guide.\n")
	document, err := translate.ExtractMPD("page.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	segment := document.Segments[0]
	state := translate.FileState{SchemaVersion: 1, PageKey: "page", SourceFile: "page.mpd", SourceLanguage: "en", TargetLanguage: "fr", Segments: map[string]translate.SegmentState{segment.ID: {SourceHash: segment.SourceHash, TargetHash: translate.Hash(string(target[:len(target)-1])), Status: "final", Provider: "original", Model: "original-model"}}}
	encoded, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	statePath = filepath.Join(root, cfg.Translation.StateDir, "fr", "page.json")
	writeConversionSource(t, filepath.Join(root, "content", "page.mpd"), 0644, string(source))
	writeConversionSource(t, filepath.Join(root, "content", "fr", "page.mpd"), 0644, string(target))
	writeConversionSource(t, statePath, 0600, string(encoded))
	return source, target, encoded, statePath
}

func requireSymlinks(t *testing.T) {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.Symlink("missing", probe); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
}

func TestConversionRestoresSourcesWhenSourceRemovalFails(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0640, "# Beta\n\nOriginal beta.\n")
	removeFailure := errors.New("source removal failed")
	ops := defaultConversionOps
	ops.rename = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == filepath.Join(root, "content", "b.md") {
			return removeFailure
		}
		return f.Rename(old, new)
	}

	if _, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops); !errors.Is(err, removeFailure) {
		t.Fatalf("source removal failure was not reported: %v", err)
	}

	assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
	assertConversionSource(t, filepath.Join(root, "content", "b.md"), b, 0640)
	assertNoMPDTargets(t, filepath.Join(root, "content"))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful rollback retained recovery copies: %v", copies)
	}
}

func TestConversionRestoresTranslationStateWhenStateWriteFails(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Translation.SourceLanguage = "en"
	root, files := openConversionProject(t, &cfg)
	source, target, encoded, statePath := writeTranslationTriplet(t, root, &cfg)
	stateWriteFailure := errors.New("state write failed")
	ops := defaultConversionOps
	ops.write = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) error {
		if filepath.Clean(name) == statePath {
			return stateWriteFailure
		}
		return f.WriteAtomicMode(name, data, mode)
	}

	if _, err := runConversionWithOps(root, &cfg, "", "markdown", files, files, ops); !errors.Is(err, stateWriteFailure) {
		t.Fatalf("state write failure was not reported: %v", err)
	}

	got, err := os.ReadFile(statePath)
	if err != nil || string(got) != string(encoded) {
		t.Fatalf("translation state was not restored: %v %q", err, got)
	}
	if info, err := os.Stat(statePath); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("translation state mode changed: %v %v", info, err)
	}
	for _, path := range []string{"content/page.mpd", "content/fr/page.mpd"} {
		assertConversionSource(t, filepath.Join(root, path), map[string][]byte{"content/page.mpd": source, "content/fr/page.mpd": target}[path], 0644)
	}
	for _, path := range []string{"content/page.md", "content/fr/page.md"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Fatalf("converted target survived state failure: %v", err)
		}
	}
}

func TestConversionRestoresTranslationStateThroughInternalAlias(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Translation.SourceLanguage = "en"
	cfg.Translation.StateDir = "translation-state"
	root, files := openConversionProject(t, &cfg)
	if err := os.MkdirAll(filepath.Join(root, ".mpress", "translations"), 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "translation-state")
	if err := os.Symlink(filepath.Join(".mpress", "translations"), link); err != nil {
		t.Fatal(err)
	}
	_, _, encoded, statePath := writeTranslationTriplet(t, root, &cfg)
	stateWriteFailure := errors.New("state write failed")
	ops := defaultConversionOps
	ops.write = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) error {
		if filepath.Clean(name) == statePath {
			return stateWriteFailure
		}
		return f.WriteAtomicMode(name, data, mode)
	}

	if _, err := runConversionWithOps(root, &cfg, "", "markdown", files, files, ops); !errors.Is(err, stateWriteFailure) {
		t.Fatalf("state write failure was not reported: %v", err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("internal state alias was not restored: %v %v", info, err)
	}
	if got, err := os.ReadFile(statePath); err != nil || string(got) != string(encoded) {
		t.Fatalf("state behind alias was not restored: %v %q", err, got)
	}
}

func TestConversionRollbackPreservesSourceRecreatedAtFailure(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0640, "# Beta\n\nOriginal beta.\n")
	c := writeConversionSource(t, filepath.Join(root, "content", "c.md"), 0644, "# Gamma\n\nOriginal gamma.\n")
	external := []byte("# Recreated\n\nExternal writer content.\n")
	removeFailure := errors.New("source removal failed")
	ops := defaultConversionOps
	ops.rename = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == filepath.Join(root, "content", "c.md") {
			if err := os.WriteFile(filepath.Join(root, "content", "a.md"), external, 0644); err != nil {
				return err
			}
			return removeFailure
		}
		return f.Rename(old, new)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(err, removeFailure) || !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), "changed after conversion") {
		t.Fatalf("clobbered rollback was not reported: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(root, "content", "a.md"))
	if readErr != nil || string(got) != string(external) {
		t.Fatalf("recreated source was overwritten: %v %q", readErr, got)
	}
	assertConversionSource(t, filepath.Join(root, "content", "b.md"), b, 0640)
	assertConversionSource(t, filepath.Join(root, "content", "c.md"), c, 0644)
	copies, globErr := filepath.Glob(filepath.Join(root, "content", recoveryPrefix+"*", "content", "a.md.recovery"))
	if globErr != nil || len(copies) != 1 {
		t.Fatalf("original recovery copy was not retained: %v %v", copies, globErr)
	}
	if got, readErr := os.ReadFile(copies[0]); readErr != nil || string(got) != string(a) {
		t.Fatalf("recovery copy lost original bytes: %v %q", readErr, got)
	}
}

func TestConversionRollbackPreservesReplacedGeneratedTarget(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0644, "# Beta\n\nOriginal beta.\n")
	external := []byte("External replacement of the generated target.\n")
	writeFailure := errors.New("target write failed")
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if filepath.Clean(name) == filepath.Join(root, "content", "b.mpd") {
			if err := os.WriteFile(filepath.Join(root, "content", "a.mpd"), external, 0644); err != nil {
				return false, err
			}
			return false, writeFailure
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(err, writeFailure) || !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), "left in place") {
		t.Fatalf("replaced target was not reported: %v", err)
	}
	got, readErr := os.ReadFile(filepath.Join(root, "content", "a.mpd"))
	if readErr != nil || string(got) != string(external) {
		t.Fatalf("replaced generated target was overwritten: %v %q", readErr, got)
	}
	assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
	assertConversionSource(t, filepath.Join(root, "content", "b.md"), b, 0644)
}

func TestConversionReportsRetainedRecoveryAfterCleanupFailure(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0644, "# Alpha\n\nOriginal alpha.\n")
	cleanupFailure := errors.New("cleanup failed")
	ops := defaultConversionOps
	ops.removeAll = func(*projectfs.FS, string) error { return cleanupFailure }

	result, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(err, cleanupFailure) || result.Count != 1 {
		t.Fatalf("cleanup failure was not reported with the successful result: %v %+v", err, result)
	}
	if !strings.Contains(err.Error(), "conversion succeeded") || !strings.Contains(err.Error(), recoveryPrefix) || strings.Contains(err.Error(), "restored") {
		t.Fatalf("cleanup failure message is misleading: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "a.md")); !os.IsNotExist(err) {
		t.Fatalf("converted source retained: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "a.mpd")); err != nil {
		t.Fatalf("converted target missing: %v", err)
	}
	if copies := recoveryCopies(t, root); len(copies) == 0 {
		t.Fatal("retained recovery copy was not reported by path")
	}
}

func TestConversionRestoresConfigurationAliasWhenSaveFails(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	base := writeConversionSource(t, filepath.Join(root, "config", "base.yaml"), 0644, "site:\n  title: Base\n")
	if err := os.Symlink(filepath.Join("config", "base.yaml"), filepath.Join(root, config.Filename)); err != nil {
		t.Fatal(err)
	}
	saveFailure := errors.New("configuration save failed")
	ops := defaultConversionOps
	ops.saveConfig = func(*projectfs.FS, config.Config) error { return saveFailure }

	_, runErr := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(runErr, saveFailure) {
		t.Fatalf("configuration save failure was not reported: %v", runErr)
	}
	info, err := os.Lstat(filepath.Join(root, config.Filename))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("configuration alias was not restored: %v %v (run error: %v)", info, err, runErr)
	}
	if target, err := os.Readlink(filepath.Join(root, config.Filename)); err != nil || target != filepath.Join("config", "base.yaml") {
		t.Fatalf("configuration alias changed: %q %v", target, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "config", "base.yaml")); err != nil || string(got) != string(base) {
		t.Fatalf("configuration alias target changed: %v %q", err, got)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
}

func TestConversionRestoresDistinctAliasesSharingOneSource(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	real := writeConversionSource(t, filepath.Join(root, "content", "real.md"), 0644, "# Real\n\nOriginal real document.\n")
	for _, name := range []string{"alias-one.md", "alias-two.md"} {
		if err := os.Symlink("real.md", filepath.Join(root, "content", name)); err != nil {
			t.Fatal(err)
		}
	}
	removeFailure := errors.New("source removal failed")
	ops := defaultConversionOps
	ops.rename = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == filepath.Join(root, "content", "real.md") {
			return removeFailure
		}
		return f.Rename(old, new)
	}

	_, runErr := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(runErr, removeFailure) {
		t.Fatalf("source removal failure was not reported: %v", runErr)
	}
	for _, name := range []string{"alias-one.md", "alias-two.md"} {
		path := filepath.Join(root, "content", name)
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("source alias %s was not restored: %v %v (run error: %v)", name, info, err, runErr)
		}
		if target, err := os.Readlink(path); err != nil || target != "real.md" {
			t.Fatalf("source alias %s changed: %q %v", name, target, err)
		}
	}
	assertConversionSource(t, filepath.Join(root, "content", "real.md"), real, 0644)
}

func TestConversionKeepsNamedRecoveryCopiesWhenRollbackFails(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0644, "# Beta\n\nOriginal beta.\n")
	removeFailure := errors.New("source removal failed")
	restoreFailure := errors.New("source restore failed")
	ops := defaultConversionOps
	ops.rename = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == filepath.Join(root, "content", "b.md") {
			return removeFailure
		}
		return f.Rename(old, new)
	}
	ops.renameLexical = func(f *projectfs.FS, old, new string) error {
		if strings.Contains(old, recoveryPrefix) {
			return restoreFailure
		}
		return f.RenameLexical(old, new)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(err, removeFailure) || !errors.Is(err, restoreFailure) || !strings.Contains(err.Error(), "rollback incomplete") || !strings.Contains(err.Error(), recoveryPrefix) {
		t.Fatalf("rollback failure was not reported with its recovery path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "a.md")); !os.IsNotExist(err) {
		t.Fatalf("failed rollback unexpectedly restored the source: %v", err)
	}
	copies, err := filepath.Glob(filepath.Join(root, "content", recoveryPrefix+"*", "content", "a.md.recovery"))
	if err != nil || len(copies) != 1 {
		t.Fatalf("named recovery copy was not retained: %v %v", copies, err)
	}
	if got, err := os.ReadFile(copies[0]); err != nil || string(got) != string(a) {
		t.Fatalf("recovery copy lost original bytes: %v %q", err, got)
	}
	for _, target := range []string{"content/a.mpd", "content/b.mpd"} {
		if _, err := os.Stat(filepath.Join(root, target)); !os.IsNotExist(err) {
			t.Fatalf("converted target survived rollback failure: %v", err)
		}
	}
}

func TestSuccessfulConversionLeavesNoRecoveryCopies(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0644, "# Alpha\n\nOriginal alpha.\n")
	result, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, defaultConversionOps)
	if err != nil || result.Count != 1 {
		t.Fatalf("conversion failed: %v %+v", err, result)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "a.mpd")); err != nil {
		t.Fatalf("converted target missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "a.md")); !os.IsNotExist(err) {
		t.Fatalf("converted source retained: %v", err)
	}
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful conversion retained recovery copies: %v", copies)
	}
}

func TestConversionRejectsDestinationCreatedAfterPlanning(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0640, "# Beta\n\nOriginal beta.\n")
	external := []byte("# Newly authored\n\nThis document appeared while conversion was running.\n")
	target := filepath.Join(root, "content", "b.mpd")
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if filepath.Clean(name) == target {
			if err := os.WriteFile(target, external, 0644); err != nil {
				return false, err
			}
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("new destination created after planning was not rejected: %v", err)
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != string(external) {
		t.Fatalf("newly authored destination changed: %v %q", readErr, got)
	}
	if info, err := os.Stat(target); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0644 {
		t.Fatalf("newly authored destination mode changed: %v %v", info, err)
	}
	assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
	assertConversionSource(t, filepath.Join(root, "content", "b.md"), b, 0640)
	if _, err := os.Stat(filepath.Join(root, "content", "a.mpd")); !os.IsNotExist(err) {
		t.Fatalf("earlier generated target survived the rejection: %v", err)
	}
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("rejected conversion retained recovery copies: %v", copies)
	}
}

func TestConversionRollsBackTargetPublishedBeforeCleanupError(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	source := filepath.Join(root, "content", "page.md")
	original := writeConversionSource(t, source, 0600, "# Keep this original\n")
	cleanupError := errors.New("temporary cleanup failed after publication")
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		published, err := f.WriteNewAtomicMode(name, data, mode)
		if err != nil {
			return published, err
		}
		return published, cleanupError
	}
	if _, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops); !errors.Is(err, cleanupError) {
		t.Fatalf("post-publication error was not reported: %v", err)
	}
	assertConversionSource(t, source, original, 0600)
	assertNoMPDTargets(t, cfg.ContentPath(root))
}
