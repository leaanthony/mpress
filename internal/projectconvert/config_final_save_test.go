package projectconvert

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

// saveConfigAtFinalPublication injects a real author settings save immediately
// before conversion's final configuration publication. The wrapper keeps the
// ordinary save otherwise, so the seam only moves the author write.
func saveConfigAtFinalPublication(t *testing.T, save func() error) conversionOps {
	t.Helper()
	injected := false
	ops := defaultConversionOps
	ops.saveConfig = func(files *projectfs.FS, cfg config.Config) error {
		if !injected {
			injected = true
			if err := save(); err != nil {
				return err
			}
		}
		return defaultConversionOps.saveConfig(files, cfg)
	}
	t.Cleanup(func() {
		if !injected {
			t.Error("final author configuration save injection did not run")
		}
	})
	return ops
}

type finalSaveFixture struct {
	cfg      config.Config
	root     string
	files    *projectfs.FS
	guide    []byte
	page     []byte
	original []byte
}

func newFinalSaveFixture(t *testing.T, existing bool) finalSaveFixture {
	t.Helper()
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	fixture := finalSaveFixture{cfg: cfg, root: root, files: files, guide: guide, page: page}
	if existing {
		if err := config.Save(root, cfg); err != nil {
			t.Fatal(err)
		}
		fixture.original = readConfigurationBytes(t, root)
	} else if _, err := os.Lstat(filepath.Join(root, config.Filename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("initial configuration was not absent: %v", err)
	}
	return fixture
}

func (f finalSaveFixture) configurationPath() string {
	return filepath.Join(f.root, config.Filename)
}

func (f finalSaveFixture) requireConflict(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("author configuration save at final publication was not reported")
	}
	if runtime.GOOS == "windows" && !errors.Is(err, fs.ErrExist) {
		if !strings.Contains(err.Error(), config.Filename) {
			t.Fatalf("rejected configuration leaf was not identified: %v", err)
		}
		return
	}
	if !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("configuration conflict was not actionable: %v", err)
	}
}

// requireOriginals asserts the caller configuration and original documents
// survived a rejected conversion.
func (f finalSaveFixture) requireOriginals(t *testing.T) {
	t.Helper()
	if f.cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", f.cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(f.root, "content", "guide.md"), f.guide, 0644)
	assertConversionSource(t, filepath.Join(f.root, "content", "page.md"), f.page, 0600)
	assertNoMPDTargets(t, f.cfg.ContentPath(f.root))
}

// requireAuthorLeaf asserts the author's exact configuration leaf and mode
// survived a rejected conversion.
func (f finalSaveFixture) requireAuthorLeaf(t *testing.T, authorBytes []byte, mode os.FileMode, styleGuide string) {
	t.Helper()
	if got := readConfigurationBytes(t, f.root); !bytes.Equal(got, authorBytes) {
		t.Fatalf("author configuration changed: %q", got)
	}
	requireConfigurationMode(t, f.root, mode)
	saved, err := config.Load(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Translation.StyleGuide != styleGuide {
		t.Fatalf("author configuration reference changed: %q", saved.Translation.StyleGuide)
	}
	f.requireOriginals(t)
}

// requireOriginalRecovery asserts the pre-conversion configuration was kept in
// named recovery storage when rollback could not restore it.
func (f finalSaveFixture) requireOriginalRecovery(t *testing.T, runErr error) {
	t.Helper()
	if !strings.Contains(runErr.Error(), recoveryPrefix) {
		t.Fatalf("retained original configuration was not named: %v", runErr)
	}
	copies := recoveryCopies(t, f.root)
	if len(copies) == 0 {
		t.Fatal("original configuration recovery copy was not retained")
	}
	for _, copyPath := range copies {
		if filepath.Base(copyPath) != config.Filename+recoverySuffix {
			continue
		}
		if data, err := os.ReadFile(copyPath); err == nil && bytes.Equal(data, f.original) {
			return
		}
	}
	t.Fatalf("original configuration bytes were not retained in recovery copies: %v", copies)
}

func TestConversionPreservesConfigurationSavedAtFinalPublication(t *testing.T) {
	inPlaceBytes := []byte("site:\n  title: Author in-place settings at final publication\nbuild:\n  contentDir: content\n  outputDir: site\n")
	for _, tc := range []struct {
		name     string
		existing bool
		inPlace  bool
	}{
		{name: "existing atomic", existing: true},
		{name: "existing in-place", existing: true, inPlace: true},
		{name: "absent atomic"},
		{name: "absent in-place", inPlace: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newFinalSaveFixture(t, tc.existing)
			author := fixture.cfg
			author.Site.Title = "Author settings saved at final publication"
			var authorBytes []byte
			mode := os.FileMode(0644)
			styleGuide := "content/guide.md"
			title := author.Site.Title
			if tc.inPlace {
				authorBytes = inPlaceBytes
				mode = 0600
				styleGuide = ""
				title = "Author in-place settings at final publication"
			}
			ops := saveConfigAtFinalPublication(t, func() error {
				if tc.inPlace {
					return os.WriteFile(fixture.configurationPath(), authorBytes, mode)
				}
				if err := config.Save(fixture.root, author); err != nil {
					return err
				}
				data, err := os.ReadFile(fixture.configurationPath())
				authorBytes = data
				return err
			})

			_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
			fixture.requireConflict(t, err)
			if !errors.Is(err, fs.ErrExist) {
				t.Fatalf("regular configuration conflict did not retain an existence error: %v", err)
			}
			fixture.requireAuthorLeaf(t, authorBytes, mode, styleGuide)
			saved, loadErr := config.Load(fixture.root)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if saved.Site.Title != title {
				t.Fatalf("author site title changed: %q", saved.Site.Title)
			}
			if tc.existing {
				fixture.requireOriginalRecovery(t, err)
			} else if !strings.Contains(err.Error(), recoveryPrefix) {
				t.Fatalf("absent configuration conflict did not name retained recovery evidence: %v", err)
			}
		})
	}
}

func TestConversionPreservesConfigurationLeafRecreatedAtFinalPublication(t *testing.T) {
	requireSymlinks(t)
	for _, tc := range []struct {
		name   string
		leaf   func(t *testing.T, fixture finalSaveFixture) string
		verify func(t *testing.T, fixture finalSaveFixture, link string)
	}{
		{
			name: "internal alias",
			leaf: func(t *testing.T, fixture finalSaveFixture) string {
				if err := os.MkdirAll(filepath.Join(fixture.root, "config"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(fixture.root, "config", "author.yaml"), []byte("site:\n  title: Author alias\n"), 0640); err != nil {
					t.Fatal(err)
				}
				return filepath.Join("config", "author.yaml")
			},
			verify: func(t *testing.T, fixture finalSaveFixture, link string) {
				data, err := os.ReadFile(filepath.Join(fixture.root, link))
				if err != nil || string(data) != "site:\n  title: Author alias\n" {
					t.Fatalf("internal author alias target changed: %v %q", err, data)
				}
			},
		},
		{
			name: "dangling alias",
			leaf: func(t *testing.T, fixture finalSaveFixture) string {
				return "missing-author.yaml"
			},
			verify: func(t *testing.T, fixture finalSaveFixture, link string) {
				if _, err := os.Lstat(filepath.Join(fixture.root, link)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("dangling author alias target was created or followed: %v", err)
				}
			},
		},
		{
			name: "external alias",
			leaf: func(t *testing.T, fixture finalSaveFixture) string {
				outside := filepath.Join(t.TempDir(), "author.yaml")
				if err := os.WriteFile(outside, []byte("site:\n  title: External author\n"), 0644); err != nil {
					t.Fatal(err)
				}
				return outside
			},
			verify: func(t *testing.T, fixture finalSaveFixture, link string) {
				data, err := os.ReadFile(link)
				if err != nil || string(data) != "site:\n  title: External author\n" {
					t.Fatalf("external author alias target changed: %v %q", err, data)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newFinalSaveFixture(t, true)
			link := tc.leaf(t, fixture)
			ops := saveConfigAtFinalPublication(t, func() error {
				return os.Symlink(link, fixture.configurationPath())
			})

			_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
			fixture.requireConflict(t, err)
			if runtime.GOOS != "windows" && !errors.Is(err, fs.ErrExist) {
				t.Fatalf("alias configuration conflict did not retain an existence error: %v", err)
			}
			info, statErr := os.Lstat(fixture.configurationPath())
			if statErr != nil || info.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("author configuration alias was not preserved: %v %v", info, statErr)
			}
			if got, readErr := os.Readlink(fixture.configurationPath()); readErr != nil || got != link {
				t.Fatalf("author configuration alias text changed: %q %v", got, readErr)
			}
			tc.verify(t, fixture, link)
			fixture.requireOriginals(t)
			fixture.requireOriginalRecovery(t, err)
		})
	}
}

func TestConversionPreservesConfigurationDirectoryRecreatedAtFinalPublication(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not provide the Unix directory collision distinction")
	}
	fixture := newFinalSaveFixture(t, true)
	ops := saveConfigAtFinalPublication(t, func() error {
		return os.MkdirAll(fixture.configurationPath(), 0755)
	})

	_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "mpd", fixture.files, fixture.files, ops)
	fixture.requireConflict(t, err)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("directory configuration conflict did not retain an existence error: %v", err)
	}
	info, statErr := os.Stat(fixture.configurationPath())
	if statErr != nil || !info.IsDir() {
		t.Fatalf("author configuration directory was not preserved: %v %v", info, statErr)
	}
	fixture.requireOriginals(t)
	fixture.requireOriginalRecovery(t, err)
}
