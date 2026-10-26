package projectconvert

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func writeConversionSource(t *testing.T, path string, mode os.FileMode, data string) []byte {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	return []byte(data)
}

func assertConversionSource(t *testing.T, path string, want []byte, mode os.FileMode) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("original source was not restored: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("original source bytes changed: %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != mode {
		t.Fatalf("original source mode changed: %v", info.Mode().Perm())
	}
}

func assertNoMPDTargets(t *testing.T, directory string) {
	t.Helper()
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && filepath.Ext(path) == ".mpd" {
			t.Fatalf("conversion left target %s after failure", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestConversionRestoresSourcesAndConfigurationWhenConfigSaveFails(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Translation.StyleGuide = "content/guide.md"
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, filepath.Join(root, "content", "b.md"), 0640, "# Beta\n\nOriginal beta.\n")
	guide := writeConversionSource(t, filepath.Join(root, "content", "guide.md"), 0644, "# Guide\n\nOriginal guide.\n")
	// A directory at the configuration destination makes the final atomic
	// rename fail after every target and source deletion has been attempted.
	if err := os.MkdirAll(filepath.Join(root, config.Filename), 0755); err != nil {
		t.Fatal(err)
	}

	if _, err := Run(root, &cfg, "", "mpd"); err == nil {
		t.Fatal("configuration save failure was not reported")
	}

	assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
	assertConversionSource(t, filepath.Join(root, "content", "b.md"), b, 0640)
	assertConversionSource(t, filepath.Join(root, "content", "guide.md"), guide, 0644)
	assertNoMPDTargets(t, filepath.Join(root, "content"))
	if cfg.Translation.StyleGuide != "content/guide.md" {
		t.Fatalf("caller configuration changed after failure: %q", cfg.Translation.StyleGuide)
	}
	info, err := os.Stat(filepath.Join(root, config.Filename))
	if err != nil || !info.IsDir() {
		t.Fatalf("configuration destination changed after failure: %v %v", info, err)
	}
}

func TestConversionRemovesEarlierTargetsWhenALaterWriteFails(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	a := writeConversionSource(t, filepath.Join(root, "content", "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	locked := filepath.Join(root, "content", "locked")
	b := writeConversionSource(t, filepath.Join(locked, "b.md"), 0644, "# Beta\n\nOriginal beta.\n")
	if err := os.Chmod(locked, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0755) })
	probe := filepath.Join(locked, "probe")
	if err := os.WriteFile(probe, []byte("x"), 0644); err == nil {
		_ = os.Remove(probe)
		t.Skip("filesystem permissions do not prevent writes in this environment")
	}

	if _, err := Run(root, &cfg, "", "mpd"); err == nil {
		t.Fatal("later target write failure was not reported")
	}

	assertConversionSource(t, filepath.Join(root, "content", "a.md"), a, 0600)
	assertConversionSource(t, filepath.Join(locked, "b.md"), b, 0644)
	if _, err := os.Stat(filepath.Join(root, "content", "a.mpd")); !os.IsNotExist(err) {
		t.Fatalf("earlier target survived a later write failure: %v", err)
	}
}

func TestConversionRejectsDuplicatePlannedTargets(t *testing.T) {
	cases := []struct {
		name, format, first, second, target string
	}{
		{"markdown extensions", "mpd", "foo.md", "foo.markdown", "foo.mpd"},
		{"source extension case", "mpd", "Page.md", "Page.MD", "Page.mpd"},
		{"reverse source extension case", "markdown", "page.mpd", "page.MPD", "page.md"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			cfg := config.Default()
			first := writeConversionSource(t, filepath.Join(root, "content", test.first), 0600, "First document.\n")
			if _, err := os.Stat(filepath.Join(root, "content", test.second)); err == nil {
				t.Skip("filesystem does not support distinct case-variant source names")
			}
			second := writeConversionSource(t, filepath.Join(root, "content", test.second), 0640, "Second document.\n")

			if _, err := Run(root, &cfg, "", test.format); err == nil {
				t.Fatal("duplicate planned target was not rejected")
			}

			assertConversionSource(t, filepath.Join(root, "content", test.first), first, 0600)
			assertConversionSource(t, filepath.Join(root, "content", test.second), second, 0640)
			if _, err := os.Stat(filepath.Join(root, "content", test.target)); !os.IsNotExist(err) {
				t.Fatalf("duplicate target was written: %v", err)
			}
		})
	}
}
