package projectconvert

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/site"
)

// writeSavedConfiguration writes settings as a separate session would and
// returns the exact bytes so a rejected conversion can be compared with them.
func writeSavedConfiguration(t *testing.T, root string, cfg config.Config, mode os.FileMode) []byte {
	t.Helper()
	data := []byte(matchingConfigurationBytes(t, cfg))
	path := filepath.Join(root, config.Filename)
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestConversionRejectsSavedReferenceOmittedByStaleCaller(t *testing.T) {
	tests := []struct {
		name       string
		caller     func(*config.Config)
		saved      func(*config.Config)
		references []string
	}{
		{
			name:       "added style guide",
			caller:     func(*config.Config) {},
			saved:      func(cfg *config.Config) { cfg.Translation.StyleGuide = "content/guide.md" },
			references: []string{`translation.styleGuide "content/guide.md"`},
		},
		{
			name:       "added glossary",
			caller:     func(*config.Config) {},
			saved:      func(cfg *config.Config) { cfg.Translation.Glossary = "content/glossary.md" },
			references: []string{`translation.glossary "content/glossary.md"`},
		},
		{
			name:   "added style guide and glossary",
			caller: func(*config.Config) {},
			saved: func(cfg *config.Config) {
				cfg.Translation.StyleGuide = "content/guide.md"
				cfg.Translation.Glossary = "content/glossary.md"
			},
			references: []string{`translation.styleGuide "content/guide.md"`, `translation.glossary "content/glossary.md"`},
		},
		{
			name:       "changed reference",
			caller:     func(cfg *config.Config) { cfg.Translation.StyleGuide = "docs/style.md" },
			saved:      func(cfg *config.Config) { cfg.Translation.StyleGuide = "content/guide.md" },
			references: []string{`translation.styleGuide "content/guide.md"`},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := config.Default()
			caller.Site.Title = "Test project"
			test.caller(&caller)
			saved := caller
			saved.Site.Title = "AUTHOR-SAVED-BEFORE-CONVERSION"
			test.saved(&saved)

			root, files := openConversionProject(t, &caller)
			guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
			glossary := writeConversionSource(t, filepath.Join(root, "content", "glossary.md"), 0600, "# Glossary\n\nOriginal glossary.\n")
			page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
			savedBytes := writeSavedConfiguration(t, root, saved, 0600)
			callerBefore := caller

			_, err := runConversionWithOps(root, &caller, "", "mpd", files, files, defaultConversionOps)
			if err == nil || !strings.Contains(err.Error(), "reload") {
				t.Fatalf("stale caller reference was not reported with an actionable conflict: %v", err)
			}
			for _, reference := range test.references {
				if !strings.Contains(err.Error(), reference) {
					t.Fatalf("conflict did not identify %s: %v", reference, err)
				}
			}
			if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
				t.Fatalf("saved configuration changed: %q", got)
			}
			requireConfigurationMode(t, root, 0600)
			if !reflect.DeepEqual(caller, callerBefore) {
				t.Fatalf("caller configuration changed after rejection: %#v", caller)
			}
			assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
			assertConversionSource(t, filepath.Join(root, "content", "glossary.md"), glossary, 0600)
			assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
			assertNoMPDTargets(t, caller.ContentPath(root))
			if copies := recoveryCopies(t, root); len(copies) != 0 {
				t.Fatalf("rejected conversion retained recovery copies: %v", copies)
			}
		})
	}
}

func TestConversionAfterReloadingSavedReferences(t *testing.T) {
	caller := config.Default()
	caller.Site.Title = "Test project"
	saved := caller
	saved.Site.Title = "AUTHOR-SAVED-BEFORE-CONVERSION"
	saved.Translation.StyleGuide = "content/guide.md"
	saved.Translation.Glossary = "content/glossary.md"

	root, _ := openConversionProject(t, &caller)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	glossary := writeConversionSource(t, filepath.Join(root, "content", "glossary.md"), 0600, "# Glossary\n\nOriginal glossary.\n")
	writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0644, "# Page\n\nOriginal page.\n")
	savedBytes := writeSavedConfiguration(t, root, saved, 0644)

	if _, err := Run(root, &caller, "", "mpd"); err == nil || !strings.Contains(err.Error(), "reload") {
		t.Fatalf("stale caller was not reported as a configuration conflict: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
		t.Fatalf("saved configuration changed before retry: %q", got)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertConversionSource(t, filepath.Join(root, "content", "glossary.md"), glossary, 0600)
	assertNoMPDTargets(t, caller.ContentPath(root))

	reloaded, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Site.Title != saved.Site.Title || reloaded.Translation.StyleGuide != "content/guide.md" || reloaded.Translation.Glossary != "content/glossary.md" {
		t.Fatalf("saved configuration was not reloaded: %#v %q %q", reloaded.Site, reloaded.Translation.StyleGuide, reloaded.Translation.Glossary)
	}
	if _, err := Run(root, &reloaded, "", "mpd"); err != nil {
		t.Fatalf("conversion after reloading author configuration: %v", err)
	}
	updated, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Site.Title != saved.Site.Title {
		t.Fatalf("conversion lost the author title: %q", updated.Site.Title)
	}
	if updated.Translation.StyleGuide != "content/guide.mpd" || updated.Translation.Glossary != "content/glossary.mpd" {
		t.Fatalf("converted references were not saved: %q %q", updated.Translation.StyleGuide, updated.Translation.Glossary)
	}
	for _, name := range []string{"guide.mpd", "glossary.mpd", "page.mpd"} {
		data, err := os.ReadFile(filepath.Join(root, "content", name))
		if err != nil || len(data) == 0 {
			t.Fatalf("converted document %s is not readable: %v", name, err)
		}
	}
	for _, name := range []string{"guide.md", "glossary.md", "page.md"} {
		if _, err := os.Stat(filepath.Join(root, "content", name)); !os.IsNotExist(err) {
			t.Fatalf("converted source %s was retained: %v", name, err)
		}
	}
	if result, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("strict build after retry: %v; diagnostics: %#v", err, result.Diagnostics)
	}
}

func TestConversionAfterReloadingSavedMPDReference(t *testing.T) {
	caller := config.Default()
	caller.Site.Title = "Test project"
	saved := caller
	saved.Translation.StyleGuide = "content/guide.mpd"

	root, _ := openConversionProject(t, &caller)
	writeConversionSource(t, filepath.Join(root, "content", "guide.mpd"), 0644, "# Guide\n\nUse clear sentences.\n")
	writeConversionSource(t, filepath.Join(root, "content", "page.mpd"), 0600, "# Page\n\nOriginal page.\n")
	writeSavedConfiguration(t, root, saved, 0644)

	if _, err := Run(root, &caller, "", "markdown"); err == nil || !strings.Contains(err.Error(), "reload") {
		t.Fatalf("stale caller was not reported as a configuration conflict: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "guide.mpd")); err != nil {
		t.Fatalf("rejected conversion removed the saved reference source: %v", err)
	}

	reloaded, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(root, &reloaded, "", "markdown"); err != nil {
		t.Fatalf("conversion after reloading author configuration: %v", err)
	}
	updated, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("converted reference was not saved: %q", updated.Translation.StyleGuide)
	}
	data, err := os.ReadFile(filepath.Join(root, "content", "guide.md"))
	if err != nil || len(data) == 0 {
		t.Fatalf("converted reference source is not readable: %v", err)
	}
	if result, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("strict build after retry: %v; diagnostics: %#v", err, result.Diagnostics)
	}
}

func TestConversionNeedsNoConfigurationWithoutReferences(t *testing.T) {
	t.Run("missing configuration", func(t *testing.T) {
		cfg := config.Default()
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0644, "# Page\n\nOriginal page.\n")

		if _, err := Run(root, &cfg, "", "mpd"); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, config.Filename)); !os.IsNotExist(err) {
			t.Fatalf("reference-free conversion created a configuration: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, "content", "page.mpd")); err != nil {
			t.Fatalf("converted document is missing: %v", err)
		}
	})

	t.Run("matching configuration", func(t *testing.T) {
		cfg := config.Default()
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0644, "# Page\n\nOriginal page.\n")
		savedBytes := writeSavedConfiguration(t, root, cfg, 0600)

		if _, err := Run(root, &cfg, "", "mpd"); err != nil {
			t.Fatal(err)
		}
		if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
			t.Fatalf("reference-free conversion rewrote configuration: %q", got)
		}
		requireConfigurationMode(t, root, 0600)
	})

	t.Run("unconverted reference", func(t *testing.T) {
		cfg := config.Default()
		cfg.Translation.StyleGuide = "docs/style.md"
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0644, "# Page\n\nOriginal page.\n")
		savedBytes := writeSavedConfiguration(t, root, cfg, 0644)

		if _, err := Run(root, &cfg, "", "mpd"); err != nil {
			t.Fatalf("reference outside the conversion directory was rejected: %v", err)
		}
		if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
			t.Fatalf("unconverted reference was rewritten: %q", got)
		}
	})

	t.Run("non-regular configuration", func(t *testing.T) {
		cfg := config.Default()
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0644, "# Page\n\nOriginal page.\n")
		if err := os.Mkdir(filepath.Join(root, config.Filename), 0755); err != nil {
			t.Fatal(err)
		}

		if _, err := Run(root, &cfg, "", "mpd"); err != nil {
			t.Fatalf("non-regular configuration blocked reference-free conversion: %v", err)
		}
		if info, err := os.Stat(filepath.Join(root, config.Filename)); err != nil || !info.IsDir() {
			t.Fatalf("non-regular configuration changed: %v %v", info, err)
		}
		if _, err := os.Stat(filepath.Join(root, "content", "page.mpd")); err != nil {
			t.Fatalf("converted document is missing: %v", err)
		}
	})
}

func TestConversionRejectsSavedReferenceThroughInternalConfigurationAlias(t *testing.T) {
	requireSymlinks(t)
	caller := config.Default()
	caller.Site.Title = "Test project"
	saved := caller
	saved.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &caller)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	base := writeConversionSource(t, filepath.Join(root, "config", "base.yaml"), 0640, matchingConfigurationBytes(t, saved))
	if err := os.Symlink(filepath.Join("config", "base.yaml"), filepath.Join(root, config.Filename)); err != nil {
		t.Fatal(err)
	}

	_, err := runConversionWithOps(root, &caller, "", "mpd", files, files, defaultConversionOps)
	if err == nil || !strings.Contains(err.Error(), "reload") || !strings.Contains(err.Error(), "translation.styleGuide") {
		t.Fatalf("saved alias reference was not reported with an actionable conflict: %v", err)
	}
	info, err := os.Lstat(filepath.Join(root, config.Filename))
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("internal configuration alias changed: %v %v", info, err)
	}
	assertConversionSource(t, filepath.Join(root, "config", "base.yaml"), base, 0640)
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, caller.ContentPath(root))
}

func TestConversionRejectsUninspectableSavedConfiguration(t *testing.T) {
	tests := []struct {
		name string
		data func(t *testing.T, saved config.Config) []byte
	}{
		{
			name: "malformed yaml",
			data: func(t *testing.T, saved config.Config) []byte {
				return []byte("translation:\n  styleGuide: content/guide.md\nunknownField: [\n")
			},
		},
		{
			name: "unknown field",
			data: func(t *testing.T, saved config.Config) []byte {
				return []byte(matchingConfigurationBytes(t, saved) + "unexpectedField: true\n")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := config.Default()
			saved := caller
			saved.Translation.StyleGuide = "content/guide.md"
			root, files := openConversionProject(t, &caller)
			guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
			page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
			savedBytes := test.data(t, saved)
			if err := os.WriteFile(filepath.Join(root, config.Filename), savedBytes, 0644); err != nil {
				t.Fatal(err)
			}

			_, err := runConversionWithOps(root, &caller, "", "mpd", files, files, defaultConversionOps)
			if err == nil || !strings.Contains(err.Error(), "inspect saved configuration "+config.Filename) || !strings.Contains(err.Error(), "parse "+config.Filename) {
				t.Fatalf("uninspectable saved settings were not rejected with the parse error: %v", err)
			}
			if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
				t.Fatalf("invalid saved configuration changed: %q", got)
			}
			assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
			assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
			assertNoMPDTargets(t, caller.ContentPath(root))
			if copies := recoveryCopies(t, root); len(copies) != 0 {
				t.Fatalf("rejected conversion retained recovery copies: %v", copies)
			}
		})
	}
}

func TestConversionSurfacesSavedConfigurationReadFailure(t *testing.T) {
	caller := config.Default()
	root, files := openConversionProject(t, &caller)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	if err := config.Save(root, caller); err != nil {
		t.Fatal(err)
	}
	savedBytes := readConfigurationBytes(t, root)
	readFailure := errors.New("configuration read failed")
	ops := defaultConversionOps
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		if filepath.Clean(name) == config.Filename {
			return nil, readFailure
		}
		return defaultConversionOps.readFile(files, name)
	}

	_, err := runConversionWithOps(root, &caller, "", "mpd", files, files, ops)
	if !errors.Is(err, readFailure) || !strings.Contains(err.Error(), config.Filename) {
		t.Fatalf("configuration read failure was not surfaced: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
		t.Fatalf("saved configuration changed after read failure: %q", got)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
	assertNoMPDTargets(t, caller.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("rejected conversion retained recovery copies: %v", copies)
	}
}
