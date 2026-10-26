package projectconvert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/site"
)

func TestConversionRejectsStaleCallerConfigurationBeforeDocumentMutation(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Title = "Test project"
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	author := cfg
	author.Site.Title = "AUTHOR-SAVED-BEFORE-CONVERSION"
	if err := config.Save(root, author); err != nil {
		t.Fatal(err)
	}
	savedBytes := readConfigurationBytes(t, root)

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, defaultConversionOps)
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("stale caller configuration was not reported as a conflict: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(savedBytes) {
		t.Fatalf("author configuration changed: %q", got)
	}
	requireConfigurationMode(t, root, 0644)
	if cfg.Site.Title != "Test project" || cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after rejection: %#v %q", cfg.Site, cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
	assertNoMPDTargets(t, cfg.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("rejected conversion retained recovery copies: %v", copies)
	}

	reloaded, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Site.Title != author.Site.Title {
		t.Fatalf("author title was not reloaded: %q", reloaded.Site.Title)
	}
	if _, err := Run(root, &reloaded, "", "mpd"); err != nil {
		t.Fatalf("conversion after reloading author configuration: %v", err)
	}
	saved, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != author.Site.Title || saved.Translation.StyleGuide != "content/guide.mpd" {
		t.Fatalf("conversion lost author settings or reference: %#v %q", saved.Site, saved.Translation.StyleGuide)
	}
	if buildResult, err := site.Build(root, site.BuildOptions{Strict: true}); err != nil {
		t.Fatalf("strict build after retry: %v; diagnostics: %#v", err, buildResult.Diagnostics)
	}
}

// saveConfigAtSnapshotCapture injects a normal author configuration save at
// the initial snapshot capture, after planning and before document writes.
func saveConfigAtSnapshotCapture(t *testing.T, save func() error) conversionOps {
	t.Helper()
	injected := false
	ops := defaultConversionOps
	ops.stat = func(files *projectfs.FS, name string) (os.FileInfo, error) {
		if filepath.Clean(name) == config.Filename && !injected {
			injected = true
			if err := save(); err != nil {
				return nil, err
			}
		}
		return defaultConversionOps.stat(files, name)
	}
	t.Cleanup(func() {
		if !injected {
			t.Error("configuration save injection at snapshot capture did not run")
		}
	})
	return ops
}

func TestConversionRejectsConfigurationSavedAtSnapshotCapture(t *testing.T) {
	cfg := config.Default()
	cfg.Site.Title = "Test project"
	cfg.Translation.StyleGuide = "content/guide.md"
	root, files := openConversionProject(t, &cfg)
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	page := writeConversionSource(t, filepath.Join(root, "content", "page.md"), 0600, "# Page\n\nOriginal page.\n")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	author := cfg
	author.Site.Title = "AUTHOR-SAVED-AT-SNAPSHOT-CAPTURE"
	authorBytes := []byte(matchingConfigurationBytes(t, author))
	ops := saveConfigAtSnapshotCapture(t, func() error { return config.Save(root, author) })

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if err == nil || !strings.Contains(err.Error(), "configuration mpress.yaml changed during conversion") {
		t.Fatalf("configuration saved at capture was not reported as a conflict: %v", err)
	}
	if got := readConfigurationBytes(t, root); string(got) != string(authorBytes) {
		t.Fatalf("author configuration changed: %q", got)
	}
	requireConfigurationMode(t, root, 0644)
	if cfg.Site.Title != "Test project" || cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after rejection: %#v %q", cfg.Site, cfg.Translation.StyleGuide)
	}
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertConversionSource(t, filepath.Join(root, "content", "page.md"), page, 0600)
	assertNoMPDTargets(t, cfg.ContentPath(root))
	if copies := recoveryCopies(t, root); len(copies) != 0 {
		t.Fatalf("rejected conversion retained recovery copies: %v", copies)
	}
}

func TestConversionAcceptsEquivalentCallerConfiguration(t *testing.T) {
	t.Run("formatting and comments", func(t *testing.T) {
		cfg := config.Default()
		cfg.Translation.StyleGuide = "content/guide.md"
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
		if err := config.Save(root, cfg); err != nil {
			t.Fatal(err)
		}
		original := readConfigurationBytes(t, root)
		formatted := []byte("# Author comment that config.Save does not write.\n\n" + string(original))
		if err := os.WriteFile(filepath.Join(root, config.Filename), formatted, 0644); err != nil {
			t.Fatal(err)
		}

		if _, err := Run(root, &cfg, "", "mpd"); err != nil {
			t.Fatalf("formatting-only configuration change was rejected: %v", err)
		}
		if cfg.Site.Title != "Documentation" || cfg.Translation.StyleGuide != "content/guide.mpd" {
			t.Fatalf("caller configuration was not updated: %#v %q", cfg.Site, cfg.Translation.StyleGuide)
		}
		saved, err := config.Load(root)
		if err != nil || saved.Translation.StyleGuide != "content/guide.mpd" {
			t.Fatalf("converted reference was not saved: %v %q", err, saved.Translation.StyleGuide)
		}
	})

	t.Run("nil and empty optional collections", func(t *testing.T) {
		cfg := config.Default()
		cfg.Translation.StyleGuide = "content/guide.md"
		cfg.Site.LanguageLabels = nil
		cfg.Site.HeaderLinks = nil
		cfg.Translation.LanguageModels = nil
		cfg.Deploy.Targets = nil
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
		persisted := "site:\n  title: Documentation\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\ntranslation:\n  styleGuide: content/guide.md\n"
		if err := os.WriteFile(filepath.Join(root, config.Filename), []byte(persisted), 0644); err != nil {
			t.Fatal(err)
		}

		if _, err := Run(root, &cfg, "", "mpd"); err != nil {
			t.Fatalf("nil caller collections were not treated as empty: %v", err)
		}
		saved, err := config.Load(root)
		if err != nil || saved.Translation.StyleGuide != "content/guide.mpd" {
			t.Fatalf("converted reference was not saved: %v %q", err, saved.Translation.StyleGuide)
		}
	})
}
