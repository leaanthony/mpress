package site

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/knowledge"
	docversion "github.com/leaanthony/mpress/internal/version"
)

const preservationConfig = `site:
  title: Published site
build:
  contentDir: content
  staticDir: static
  outputDir: site
`

func publishedProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", preservationConfig)
	writeFixture(t, root, "content/index.md", "---\ntitle: Original\n---\n\nOriginal home.\n")
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\nOriginal guide.\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	return root
}

func readPublishedIndex(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil {
		t.Fatalf("published index is unreadable: %v", err)
	}
	return string(data)
}

func assertNoStagingLeftovers(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".mpress-build-") || strings.Contains(entry.Name(), ".mpress-previous-") {
			t.Errorf("staging artifact left behind: %s", entry.Name())
		}
	}
}

func TestBuildFailurePreservesPublishedSite(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T, root string)
		options BuildOptions
	}{
		{
			name: "strict parse diagnostic",
			prepare: func(t *testing.T, root string) {
				writeFixture(t, root, "content/broken.mpd", "---\nschema = 1\ntitle = \"Broken\"\n---\n\n```d2\na: {\n```\n")
			},
			options: BuildOptions{Strict: true},
		},
		{
			name: "navigation load",
			prepare: func(t *testing.T, root string) {
				writeFixture(t, root, "content/_nav.yaml", "[\n")
			},
			options: BuildOptions{Strict: true},
		},
		{
			name: "custom css diagnostic",
			prepare: func(t *testing.T, root string) {
				cfg, err := config.Load(root)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Build.CustomCSS = "missing.css"
				if err := config.Save(root, cfg); err != nil {
					t.Fatal(err)
				}
			},
			options: BuildOptions{Strict: true},
		},
		{
			name: "asset minify",
			prepare: func(t *testing.T, root string) {
				writeFixture(t, root, "custom.css", `.broken { content: "nope; }`+"\n")
				cfg, err := config.Load(root)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Build.CustomCSS = "custom.css"
				if err := config.Save(root, cfg); err != nil {
					t.Fatal(err)
				}
			},
			options: BuildOptions{Strict: true, MinifyAssets: true},
		},
		{
			name: "static copy",
			prepare: func(t *testing.T, root string) {
				outside := t.TempDir()
				writeFixture(t, outside, "secret.txt", "outside the project")
				if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "static", "escape.txt")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			},
			options: BuildOptions{Strict: true},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := publishedProject(t)
			published := readPublishedIndex(t, root)
			writeFixture(t, root, "site/obsolete.txt", "last good build")
			writeFixture(t, root, "content/index.md", "---\ntitle: Changed\n---\n\nChanged home.\n")
			test.prepare(t, root)
			if _, err := Build(root, test.options); err == nil {
				t.Fatal("expected the build to fail")
			}
			if got := readPublishedIndex(t, root); got != published {
				t.Fatalf("failed build replaced the published index:\n%s", got)
			}
			if data, err := os.ReadFile(filepath.Join(root, "site", "obsolete.txt")); err != nil || string(data) != "last good build" {
				t.Fatalf("failed build removed published files: %q, %v", data, err)
			}
			assertNoStagingLeftovers(t, root)
		})
	}
}

func TestBuildVersionMountFailurePreservesPublishedSite(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", preservationConfig+`versioning:
  enabled: true
  current: Next
  artifactsDir: .mpress/versions
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Original\n---\n\nOriginal home.\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if err := docversion.Capture(root, "1.0", false); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(root, ".mpress", "versions", "1.0", "index.html")
	data, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshot, append(data, "corrupt"...), 0o644); err != nil {
		t.Fatal(err)
	}
	published := readPublishedIndex(t, root)
	writeFixture(t, root, "site/obsolete.txt", "last good build")
	writeFixture(t, root, "content/index.md", "---\ntitle: Changed\n---\n\nChanged home.\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err == nil {
		t.Fatal("expected version mounting to fail")
	}
	if got := readPublishedIndex(t, root); got != published {
		t.Fatalf("failed version mount replaced the published index:\n%s", got)
	}
	if data, err := os.ReadFile(filepath.Join(root, "site", "obsolete.txt")); err != nil || string(data) != "last good build" {
		t.Fatalf("failed version mount removed published files: %q, %v", data, err)
	}
	assertNoStagingLeftovers(t, root)
}

func TestBuildSuccessReplacesPublishedSiteAndKeepsDirectoryMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not meaningful on Windows")
	}
	root := publishedProject(t)
	writeFixture(t, root, "site/obsolete.txt", "stale output")
	output := filepath.Join(root, "site")
	if err := os.Chmod(output, 0o750); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "content/index.md", "---\ntitle: Changed\n---\n\nChanged home.\n")
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if got := readPublishedIndex(t, root); !strings.Contains(got, "Changed home.") {
		t.Fatalf("successful build did not publish new content:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(output, "obsolete.txt")); !os.IsNotExist(err) {
		t.Fatalf("successful build kept stale output: %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o750 {
		t.Fatalf("output directory mode changed from 0750 to %04o", perm)
	}
	assertNoStagingLeftovers(t, root)
}

func TestBuildCollectorIndexesFinalOutputWithVersionsAndKnowledge(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", preservationConfig+`versioning:
  enabled: true
  current: Next
  artifactsDir: .mpress/versions
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n[Guide](/guide/)\n")
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\n[Home](/)\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	if err := docversion.Capture(root, "1.0", false); err != nil {
		t.Fatal(err)
	}
	collector := check.NewCollector()
	result, err := Build(root, BuildOptions{Strict: true, LinkCollector: collector})
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "site")
	if result.Files != countFiles(output) {
		t.Fatalf("collector counted %d files, output has %d", result.Files, countFiles(output))
	}
	if broken := collector.Finalize(); len(broken) != 0 {
		t.Fatalf("collector found broken links against staged paths: %#v", broken)
	}
	if broken, err := check.Run(output); err != nil || len(broken) != 0 {
		t.Fatalf("standalone check disagreed: %v, %#v", err, broken)
	}
	assertExists(t, filepath.Join(output, "versions", "1.0", "index.html"))
	assertExists(t, filepath.Join(output, knowledge.Directory, knowledge.IndexFile))
	assertNoStagingLeftovers(t, root)
}

func newPromotionFixture(t *testing.T) (*stagedBuildOutput, string, string) {
	t.Helper()
	root := t.TempDir()
	final := filepath.Join(root, "site")
	staging := filepath.Join(root, ".site.mpress-build-test")
	writeFixture(t, root, "site/index.html", "previous output")
	writeFixture(t, root, ".site.mpress-build-test/index.html", "new output")
	stage := &stagedBuildOutput{
		final: final, staging: staging, project: root, mode: 0o755,
		rename: os.Rename, removeAll: os.RemoveAll,
		mkdirTemp: func(parent, prefix string) (string, error) { return os.MkdirTemp(parent, prefix) },
		chmod:     os.Chmod,
	}
	return stage, final, staging
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func findBackup(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".mpress-previous-") {
			return filepath.Join(root, entry.Name())
		}
	}
	return ""
}

func TestStagedOutputPromotionRollsBackOnFailure(t *testing.T) {
	t.Run("install failure restores previous output", func(t *testing.T) {
		stage, final, staging := newPromotionFixture(t)
		injected := errors.New("injected rename failure")
		stage.rename = func(old, new string) error {
			if old == staging {
				return injected
			}
			return os.Rename(old, new)
		}
		if err := stage.promote(); !errors.Is(err, injected) {
			t.Fatalf("expected injected promotion failure, got %v", err)
		}
		if got := readFixture(t, filepath.Join(final, "index.html")); got != "previous output" {
			t.Fatalf("previous output was not restored: %q", got)
		}
		if got := readFixture(t, filepath.Join(staging, "index.html")); got != "new output" {
			t.Fatalf("staging output was lost: %q", got)
		}
		if backup := findBackup(t, filepath.Dir(final)); backup != "" {
			t.Fatalf("successful rollback left a backup: %s", backup)
		}
	})

	t.Run("rollback failure retains previous output", func(t *testing.T) {
		stage, _, staging := newPromotionFixture(t)
		injected := errors.New("injected rename failure")
		stage.rename = func(old, new string) error {
			if old == staging || strings.Contains(old, ".mpress-previous-") {
				return injected
			}
			return os.Rename(old, new)
		}
		if err := stage.promote(); !errors.Is(err, injected) {
			t.Fatalf("expected injected promotion failure, got %v", err)
		}
		backup := findBackup(t, filepath.Dir(stage.final))
		if backup == "" {
			t.Fatal("failed rollback discarded the previous output")
		}
		if got := readFixture(t, filepath.Join(backup, "output", "index.html")); got != "previous output" {
			t.Fatalf("retained previous output changed: %q", got)
		}
	})

	t.Run("post promotion rollback restores previous output", func(t *testing.T) {
		stage, final, staging := newPromotionFixture(t)
		if err := stage.promote(); err != nil {
			t.Fatal(err)
		}
		if got := readFixture(t, filepath.Join(final, "index.html")); got != "new output" {
			t.Fatalf("promotion did not install new output: %q", got)
		}
		if err := stage.rollback(); err != nil {
			t.Fatal(err)
		}
		if got := readFixture(t, filepath.Join(final, "index.html")); got != "previous output" {
			t.Fatalf("post-promotion rollback did not restore the previous output: %q", got)
		}
		if got := readFixture(t, filepath.Join(staging, "index.html")); got != "new output" {
			t.Fatalf("post-promotion rollback lost the new output: %q", got)
		}
		stage.discard()
		if _, err := os.Stat(staging); !os.IsNotExist(err) {
			t.Fatalf("discard did not remove staging: %v", err)
		}
	})

	t.Run("finish removes the retained backup", func(t *testing.T) {
		stage, final, _ := newPromotionFixture(t)
		warning, err := stage.finishBuild(nil, config.Default())
		if err != nil {
			t.Fatal(err)
		}
		if warning != nil {
			t.Fatalf("unexpected warning: %+v", *warning)
		}
		if got := readFixture(t, filepath.Join(final, "index.html")); got != "new output" {
			t.Fatalf("finish changed the promoted output: %q", got)
		}
		if backup := findBackup(t, filepath.Dir(final)); backup != "" {
			t.Fatalf("finish left a backup: %s", backup)
		}
	})

	t.Run("backup cleanup failure warns after publication", func(t *testing.T) {
		stage, final, _ := newPromotionFixture(t)
		injected := errors.New("injected cleanup failure")
		stage.removeAll = func(path string) error {
			if strings.Contains(path, ".mpress-previous-") {
				return injected
			}
			return os.RemoveAll(path)
		}
		warning, err := stage.finishBuild(nil, config.Default())
		if err != nil {
			t.Fatalf("published build failed on cleanup: %v", err)
		}
		if warning == nil || warning.Severity != "warning" || warning.Code != "output-cleanup" {
			t.Fatalf("expected output-cleanup warning, got %+v", warning)
		}
		if !strings.Contains(warning.Message, ".mpress-previous-") {
			t.Fatalf("warning does not retain the backup path: %s", warning.Message)
		}
		if got := readFixture(t, filepath.Join(final, "index.html")); got != "new output" {
			t.Fatalf("cleanup warning disturbed the new output: %q", got)
		}
		if backup := findBackup(t, filepath.Dir(final)); backup == "" {
			t.Fatal("cleanup failure did not retain the backup")
		}
	})
}

func TestBuildStagingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not meaningful on Windows")
	}
	t.Run("discard read-only staging after failed promotion", func(t *testing.T) {
		stage, _, staging := newPromotionFixture(t)
		if err := os.Chmod(staging, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(staging, 0o700) })
		stage.discard()
		if _, err := os.Lstat(staging); !os.IsNotExist(err) {
			t.Fatalf("discard left read-only staging behind: %v", err)
		}
	})
	t.Run("first build uses 0755", func(t *testing.T) {
		root := t.TempDir()
		writeFixture(t, root, "mpress.yaml", preservationConfig)
		writeFixture(t, root, "content/index.md", "# Home\n")
		if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(filepath.Join(root, "site"))
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o755 {
			t.Fatalf("first build output mode is %04o, want 0755", perm)
		}
	})
	t.Run("read-only published output rebuilds with retained mode", func(t *testing.T) {
		root := publishedProject(t)
		output := filepath.Join(root, "site")
		if err := os.Chmod(output, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(output, 0o755) })
		writeFixture(t, root, "content/index.md", "---\ntitle: Changed\n---\n\nChanged home.\n")
		result, err := Build(root, BuildOptions{Strict: true})
		if err != nil {
			t.Fatalf("rebuild over read-only output: %v", err)
		}
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Code == "output-cleanup" {
				t.Fatalf("unexpected cleanup diagnostic: %+v", diagnostic)
			}
		}
		if got := readPublishedIndex(t, root); !strings.Contains(got, "Changed home.") {
			t.Fatalf("read-only output was not replaced:\n%s", got)
		}
		info, err := os.Stat(output)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o555 {
			t.Fatalf("output mode changed from 0555 to %04o", perm)
		}
	})
}
