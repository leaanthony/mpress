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

func readConfigurationBytes(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func requireConfigurationMode(t *testing.T, root string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(filepath.Join(root, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != want {
		t.Fatalf("configuration mode changed: %v", info.Mode().Perm())
	}
}

// matchingConfigurationBytes renders a configuration exactly as config.Save
// would, so an internal alias fixture describes the same settings as its
// caller instead of introducing an unrelated stale configuration.
func matchingConfigurationBytes(t *testing.T, cfg config.Config) string {
	t.Helper()
	directory := t.TempDir()
	if err := config.Save(directory, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func saveConfigAfterPublication(t *testing.T, save func() error) conversionOps {
	t.Helper()
	injected := false
	ops := defaultConversionOps
	ops.writeNew = func(files *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		published, err := files.WriteNewAtomicMode(name, data, mode)
		if err != nil {
			return published, err
		}
		if !injected {
			injected = true
			if err := save(); err != nil {
				return published, err
			}
		}
		return published, nil
	}
	t.Cleanup(func() {
		if !injected {
			t.Error("author configuration save injection did not run")
		}
	})
	return ops
}

func TestConversionRejectsAtomicConfigurationSaveAfterPublication(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	author := cfg
	author.Site.Title = "Author settings saved during conversion"
	ops := saveConfigAfterPublication(t, func() error { return config.Save(root, author) })

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("author configuration save was not reported as a conflict: %v", err)
	}
	saved, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != author.Site.Title || saved.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("author configuration was not preserved: %#v", saved.Site)
	}
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
	assertNoMPDTargets(t, cfg.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("rejected conversion retained recovery copies: %v", copies)
	}
}

func TestConversionRejectsInPlaceConfigurationSaveAfterPublication(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	authorBytes := []byte("site:\n  title: Author in-place settings save\nbuild:\n  contentDir: content\n  outputDir: site\n")
	ops := saveConfigAfterPublication(t, func() error {
		return os.WriteFile(filepath.Join(root, config.Filename), authorBytes, 0644)
	})

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("in-place configuration save was not reported as a conflict: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(authorBytes) {
		t.Fatalf("in-place author configuration changed: %q", got)
	}
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, cfg.ContentPath(root))
}

func TestConversionRejectsConfigurationModeChangeAfterPublication(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not provide these Unix permission distinctions")
	}
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	original := readConfigurationBytes(t, root)
	ops := saveConfigAfterPublication(t, func() error {
		return os.Chmod(filepath.Join(root, config.Filename), 0600)
	})

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("configuration mode change was not reported as a conflict: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(original) {
		t.Fatalf("configuration bytes changed: %q", got)
	}
	requireConfigurationMode(t, root, 0600)
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, cfg.ContentPath(root))
}

func TestConversionPreservesConfigurationCreatedAfterInitialAbsence(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	if _, err := os.Stat(filepath.Join(root, config.Filename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("initial configuration was not absent: %v", err)
	}
	author := cfg
	author.Site.Title = "Author settings created during conversion"
	ops := saveConfigAfterPublication(t, func() error { return config.Save(root, author) })

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("newly created configuration was not reported as a conflict: %v", err)
	}
	saved, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != author.Site.Title {
		t.Fatalf("created author configuration was not preserved: %q", saved.Site.Title)
	}
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, cfg.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("rejected conversion retained recovery copies: %v", copies)
	}
}

func TestConversionCreatesConfigurationAfterInitialAbsence(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, _ := openConversionProject(t, &cfg)
	writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")

	if _, err := Run(root, &cfg, "", "mpd"); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Translation.StyleGuide != "content/guide.mpd" || cfg.Translation.StyleGuide != "content/guide.mpd" {
		t.Fatalf("converted reference was not saved: %q %q", saved.Translation.StyleGuide, cfg.Translation.StyleGuide)
	}
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful conversion retained recovery copies: %v", copies)
	}
}

func TestConversionKeepsUnchangedConfigurationAndUpdatesReferences(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, _ := openConversionProject(t, &cfg)
	writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(root, &cfg, "", "mpd"); err != nil {
		t.Fatal(err)
	}
	saved, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != cfg.Site.Title || saved.Translation.StyleGuide != "content/guide.mpd" {
		t.Fatalf("configuration was not updated in place: %#v %q", saved.Site, saved.Translation.StyleGuide)
	}
	if cfg.Translation.StyleGuide != "content/guide.mpd" {
		t.Fatalf("caller configuration was not updated: %q", cfg.Translation.StyleGuide)
	}
	requireConfigurationMode(t, root, 0644)
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful conversion retained recovery copies: %v", copies)
	}
}

func TestConversionRestoresConfigurationWhenReferenceSaveFails(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	original := readConfigurationBytes(t, root)
	saveFailure := errors.New("configuration reference save failed")
	ops := defaultConversionOps
	ops.saveConfig = func(*projectfs.FS, config.Config) error { return saveFailure }

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !errors.Is(err, saveFailure) {
		t.Fatalf("configuration save failure was not reported: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(original) {
		t.Fatalf("configuration bytes changed after failure: %q", got)
	}
	requireConfigurationMode(t, root, 0644)
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
	assertNoMPDTargets(t, cfg.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful rollback retained recovery copies: %v", copies)
	}
}

func TestConversionPreservesConfigurationReplacedBeforeReplacingRename(t *testing.T) {
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	author := cfg
	author.Site.Title = "Author settings saved before the replacing rename"
	injected := false
	ops := defaultConversionOps
	ops.renameLexical = func(f *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == config.Filename && !injected {
			injected = true
			if err := config.Save(root, author); err != nil {
				return err
			}
		}
		return f.RenameLexical(old, new)
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !injected {
		t.Fatal("configuration replacement injection did not run")
	}
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("captured configuration change was not reported: %v", err)
	}
	saved, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != author.Site.Title || saved.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("captured author configuration was not restored: %#v %q", saved.Site, saved.Translation.StyleGuide)
	}
	requireConfigurationMode(t, root, 0644)
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, cfg.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("restored conflict retained recovery copies: %v", copies)
	}
}

func TestConversionUpdatesInternalConfigurationAlias(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, _ := openConversionProject(t, &cfg)
	writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	base := writeConversionSource(t, filepath.Join(root, "config", "base.yaml"), 0640, matchingConfigurationBytes(t, cfg))
	if err := os.Symlink(filepath.Join("config", "base.yaml"), filepath.Join(root, config.Filename)); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(root, &cfg, "", "mpd"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(root, config.Filename))
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		t.Fatalf("internal configuration alias was not replaced safely: %v %v", info, err)
	}
	saved, err := config.Load(root)
	if err != nil || saved.Translation.StyleGuide != "content/guide.mpd" {
		t.Fatalf("converted reference was not saved through the alias: %v %q", err, saved.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "config", "base.yaml"), base, 0640)
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("successful conversion retained recovery copies: %v", copies)
	}
}

func TestConversionRejectsExternalConfigurationAlias(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, _ := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	outside := filepath.Join(t.TempDir(), "base.yaml")
	external := writeConversionSource(t, outside, 0640, "site:\n  title: External\n")
	if err := os.Symlink(outside, filepath.Join(root, config.Filename)); err != nil {
		t.Fatal(err)
	}

	_, err := Run(root, &cfg, "", "mpd")
	if !errors.Is(err, projectfs.ErrOutside) {
		t.Fatalf("external configuration alias was not rejected: %v", err)
	}
	assertConversionSource(t, outside, external, 0640)
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after rejection: %q", cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, cfg.ContentPath(root))
}
