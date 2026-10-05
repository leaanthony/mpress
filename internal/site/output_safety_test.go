package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestBuildPreservesInputsWhenOutputOverlapsProjectData(t *testing.T) {
	for _, output := range []string{".", "content", "content/generated", "static", "static/generated", config.Filename, ".git", ".git/generated", ".mpress", ".mpress/cache", ".mpress/translations", ".mpress/versions", ".mpress/backups", ".mpress/onboarding", ".mpress/unknown-state"} {
		t.Run(output, func(t *testing.T) {
			root := t.TempDir()
			cfg := config.Default()
			cfg.Build.OutputDir = output
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			files := outputSafetySentinels(t, root)
			if _, err := Build(root, BuildOptions{Strict: true}); err == nil || !strings.Contains(err.Error(), "unsafe output") {
				t.Errorf("expected unsafe output rejection, got %v", err)
			}
			assertOutputSafetySentinels(t, root, files)
		})
	}
}

func TestBuildOverrideCannotDeleteContent(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	files := outputSafetySentinels(t, root)
	if _, err := Build(root, BuildOptions{OutputDir: filepath.Join(root, "content")}); err == nil || !strings.Contains(err.Error(), "unsafe output") {
		t.Errorf("override should be rejected, got %v", err)
	}
	assertOutputSafetySentinels(t, root, files)
}

func TestBuildRejectsOutputThroughExternalParent(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	files := outputSafetySentinels(t, root)
	writeFixture(t, outside, "site/sentinel.txt", "outside must survive")
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Build(root, BuildOptions{OutputDir: "linked/site"}); err == nil || !strings.Contains(err.Error(), "unsafe output") {
		t.Errorf("external parent should be rejected, got %v", err)
	}
	assertOutputSafetySentinels(t, root, files)
	data, err := os.ReadFile(filepath.Join(outside, "site/sentinel.txt"))
	if err != nil || string(data) != "outside must survive" {
		t.Fatalf("external sentinel changed: %q, %v", data, err)
	}
}

func TestBuildAllowsSafeOutputAliasesAndMissingParents(t *testing.T) {
	for _, kind := range []string{"missing parents", "internal parent alias", "project alias"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			cfg := config.Default()
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			files := outputSafetySentinels(t, root)
			project, output, generated := root, "generated/missing/site", filepath.Join(root, "generated/missing/site")
			if kind == "internal parent alias" {
				writeFixture(t, root, "generated/site/obsolete", "replace old output")
				if err := os.Symlink(filepath.Join(root, "generated"), filepath.Join(root, "linked")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				output, generated = "linked/site", filepath.Join(root, "generated/site")
			}
			if kind == "project alias" {
				project = filepath.Join(t.TempDir(), "project")
				if err := os.Symlink(root, project); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				output, generated = "site", filepath.Join(root, "site")
			}
			if _, err := Build(project, BuildOptions{Strict: true, OutputDir: output}); err != nil {
				t.Fatal(err)
			}
			assertExists(t, filepath.Join(generated, "index.html"))
			assertOutputSafetySentinels(t, root, files)
		})
	}
}

func TestBuildReplacesFinalOutputLinkWithoutDeletingItsTarget(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	files := outputSafetySentinels(t, root)
	writeFixture(t, root, "generated/keep", "link target must survive")
	output := filepath.Join(root, "site")
	if err := os.Symlink(filepath.Join(root, "generated"), output); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Build(root, BuildOptions{Strict: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "generated/keep"))
	if err != nil || string(data) != "link target must survive" {
		t.Errorf("final output link was followed during deletion: %q, %v", data, err)
	}
	info, err := os.Lstat(output)
	if err != nil || !info.IsDir() {
		t.Errorf("output link was not replaced with a directory: %v, %v", info, err)
	}
	assertExists(t, filepath.Join(output, "index.html"))
	assertOutputSafetySentinels(t, root, files)
}

func outputSafetySentinels(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{
		"content/index.md": "# Source must survive\n", "static/asset.txt": "asset must survive",
		".git/sentinel": "git must survive", ".mpress/cache/sentinel": "cache must survive",
		".mpress/translations/state.json": "state must survive", ".mpress/versions/prior/sentinel": "version must survive",
		".mpress/backups/sentinel": "backup must survive", ".mpress/onboarding": "onboarding must survive",
		".mpress/unknown-state/sentinel": "future state must survive",
	}
	for name, value := range files {
		writeFixture(t, root, name, value)
	}
	data, err := os.ReadFile(filepath.Join(root, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	files[config.Filename] = string(data)
	return files
}

func assertOutputSafetySentinels(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, want := range files {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(data) != want {
			t.Errorf("protected %s changed (got %d bytes, error %v)", name, len(data), err)
		}
	}
}
