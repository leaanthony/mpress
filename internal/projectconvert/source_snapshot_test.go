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

func TestConversionRejectsSourceEditedDuringPublication(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/page.md"
	root, files := openConversionProject(t, &cfg)
	sourcePath := filepath.Join(root, "content", "page.md")
	writeConversionSource(t, sourcePath, 0600, "# Original\n\nPlanning snapshot.\n")
	edited := []byte("# Edited\n\nThe author changed this during conversion.\n")
	target := filepath.Join(root, "content", "page.mpd")
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if filepath.Clean(name) == target {
			if err := os.WriteFile(sourcePath, edited, 0640); err != nil {
				return false, err
			}
			if err := os.Chmod(sourcePath, 0640); err != nil {
				return false, err
			}
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if got, err := os.ReadFile(sourcePath); err != nil || string(got) != string(edited) {
		t.Fatalf("author edit was not preserved: %v %q", err, got)
	}
	if err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("edited source was not rejected: %v", err)
	}
	if info, err := os.Stat(sourcePath); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("author mode was not preserved: %v %v", info, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("generated target survived source rejection: %v", err)
	}
	if cfg.Translation.StyleGuide != "content/page.md" {
		t.Fatalf("caller configuration changed: %q", cfg.Translation.StyleGuide)
	}
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful rollback retained recovery copies: %v", copies)
	}
}

func TestConversionRejectsAtomicSourceReplacementDuringPublication(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	sourcePath := filepath.Join(root, "content", "page.md")
	writeConversionSource(t, sourcePath, 0600, "# Original\n\nPlanning snapshot.\n")
	edited := []byte("# Replaced\n\nA new inode replaced this document.\n")
	target := filepath.Join(root, "content", "page.mpd")
	replacement := filepath.Join(root, "content", "page.md.replacement")
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if filepath.Clean(name) == target {
			if err := os.WriteFile(replacement, edited, 0640); err != nil {
				return false, err
			}
			if err := os.Rename(replacement, sourcePath); err != nil {
				return false, err
			}
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if got, err := os.ReadFile(sourcePath); err != nil || string(got) != string(edited) {
		t.Fatalf("replacement bytes were not preserved: %v %q", err, got)
	}
	if err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("atomically replaced source was not rejected: %v", err)
	}
	if info, err := os.Stat(sourcePath); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("replacement mode was not preserved: %v %v", info, err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("generated target survived source rejection: %v", err)
	}
}

func TestConversionRejectsLateSourceEditAfterEarlierSourceMoved(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	bPath := filepath.Join(root, "content", "b.md")
	writeConversionSource(t, bPath, 0640, "# Beta\n\nOriginal beta.\n")
	edited := []byte("# Beta edited\n\nChanged after alpha was consumed.\n")
	injected := false
	ops := defaultConversionOps
	ops.renameLexical = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == bPath {
			injected = true
			if err := os.WriteFile(bPath, edited, 0644); err != nil {
				return err
			}
			if err := os.Chmod(bPath, 0644); err != nil {
				return err
			}
		}
		return f.RenameLexical(old, new)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !injected {
		t.Fatal("late source edit injection did not occur")
	}
	if got, err := os.ReadFile(bPath); err != nil || string(got) != string(edited) {
		t.Fatalf("late author edit was not preserved: %v %q", err, got)
	}
	if err == nil || !strings.Contains(err.Error(), "changed after planning") || !strings.Contains(err.Error(), "b.md") {
		t.Fatalf("late source edit was not rejected: %v", err)
	}
	assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
	if info, err := os.Stat(bPath); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0644 {
		t.Fatalf("late author mode was not preserved: %v %v", info, err)
	}
	assertNoMPDTargets(t, filepath.Join(root, "content"))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful rollback retained recovery copies: %v", copies)
	}
}

func TestConversionRetainsAliasWhenTargetSortsBeforeLink(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	writeConversionSource(t, filepath.Join(root, "content", "real.md"), 0644, "# Real\n\nAliased document.\n")
	if err := os.Symlink("real.md", filepath.Join(root, "content", "z-alias.md")); err != nil {
		t.Fatal(err)
	}
	result, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, defaultConversionOps)
	if err != nil || result.Count != 2 {
		t.Fatalf("alias conversion failed: %v %+v", err, result)
	}
	for _, name := range []string{"real.mpd", "z-alias.mpd"} {
		if _, err := os.Stat(filepath.Join(root, "content", name)); err != nil {
			t.Fatalf("converted alias target %s missing: %v", name, err)
		}
	}
	for _, name := range []string{"real.md", "z-alias.md"} {
		if _, err := os.Lstat(filepath.Join(root, "content", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("converted source %s retained: %v", name, err)
		}
	}
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful alias conversion retained recovery copies: %v", copies)
	}
}

func TestConversionRejectsSourcePermissionChangeDuringPublication(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not provide these Unix permission distinctions")
	}
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	sourcePath := filepath.Join(root, "content", "page.md")
	original := writeConversionSource(t, sourcePath, 0600, "# Original\n\nPlanning snapshot.\n")
	target := filepath.Join(root, "content", "page.mpd")
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if filepath.Clean(name) == target {
			if err := os.Chmod(sourcePath, 0640); err != nil {
				return false, err
			}
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if info, statErr := os.Stat(sourcePath); statErr != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("author permission change was not preserved: %v %v", info, statErr)
	}
	if got, readErr := os.ReadFile(sourcePath); readErr != nil || string(got) != string(original) {
		t.Fatalf("source bytes changed: %v %q", readErr, got)
	}
	if err == nil || !strings.Contains(err.Error(), "permissions changed") {
		t.Fatalf("source permission change was not rejected: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("generated target survived source rejection: %v", err)
	}
}

func TestConversionRejectsLateSourceEditAndRestoresTranslationState(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Translation.SourceLanguage = "en"
	root, files := openConversionProject(t, &cfg)
	source, _, encoded, statePath := writeTranslationTriplet(t, root, &cfg)
	translated := filepath.Join(root, "content", "fr", "page.mpd")
	edited := []byte("Lisez le guide modifie.\n")
	ops := defaultConversionOps
	ops.renameLexical = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == translated {
			if err := os.WriteFile(translated, edited, 0640); err != nil {
				return err
			}
			if err := os.Chmod(translated, 0640); err != nil {
				return err
			}
		}
		return f.RenameLexical(old, new)
	}

	_, err := runConversionWithOps(root, &cfg, "", "markdown", files, files, ops)
	if got, readErr := os.ReadFile(statePath); readErr != nil || string(got) != string(encoded) {
		t.Fatalf("translation state was not restored: %v %q", readErr, got)
	}
	assertConversionSource(t, filepath.Join(root, "content", "page.mpd"), source, 0644)
	if got, readErr := os.ReadFile(translated); readErr != nil || string(got) != string(edited) {
		t.Fatalf("late translated edit was not preserved: %v %q", readErr, got)
	}
	if info, statErr := os.Stat(translated); statErr != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0640 {
		t.Fatalf("late translated mode was not preserved: %v %v", info, statErr)
	}
	for _, name := range []string{"content/page.md", "content/fr/page.md"} {
		if _, statErr := os.Stat(filepath.Join(root, name)); !os.IsNotExist(statErr) {
			t.Fatalf("converted target survived source rejection: %v", statErr)
		}
	}
	if err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("late translated edit was not rejected: %v", err)
	}
}
