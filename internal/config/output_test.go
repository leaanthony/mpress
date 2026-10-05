package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveOutputProtectsConfiguredDataAndAncestors(t *testing.T) {
	for _, test := range []struct {
		name, output, protected string
		configure               func(*Config)
	}{
		{"source ancestor", "docs", "docs/content/keep", func(c *Config) { c.Build.ContentDir = "docs/content" }},
		{"static ancestor", "public", "public/assets/keep", func(c *Config) { c.Build.StaticDir = "public/assets" }},
		{"custom translation state", "state", "state/languages/keep", func(c *Config) { c.Translation.StateDir = "state/languages" }},
		{"custom version store", "history", "history/versions/keep", func(c *Config) { c.Version.Artifacts = "history/versions" }},
		{"custom stylesheet", "theme", "theme/custom.css", func(c *Config) { c.Build.CustomCSS = "theme/custom.css" }},
		{"navigation outside content", "preview", "preview/nav.yaml", func(c *Config) { c.Build.NavFile = "../preview/nav.yaml" }},
		{"localized navigation outside content", "preview", "preview/nav.yaml", func(c *Config) { c.Site.Languages = []string{"en", "fr"}; c.Build.NavFile = "../../preview/nav.yaml" }},
		{"translation glossary", "preview", "preview/glossary.yaml", func(c *Config) { c.Translation.Glossary = "preview/glossary.yaml" }},
		{"translation style guide", "preview", "preview/style.md", func(c *Config) { c.Translation.StyleGuide = "preview/style.md" }},
		{"contributor guide", "preview", "preview/guide.md", func(c *Config) { c.Contribution.Guide = "preview/guide.md" }},
		{"temporary output source conflict", ".mpress/live-preview/build", ".mpress/live-preview/build/content/keep", func(c *Config) { c.Build.ContentDir = ".mpress/live-preview/build/content" }},
		{"temporary output state conflict", ".mpress/export-abc/site", ".mpress/export-abc/site/state/keep", func(c *Config) { c.Translation.StateDir = ".mpress/export-abc/site/state" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			cfg := Default()
			test.configure(&cfg)
			path := filepath.Join(root, test.protected)
			writeOutputSentinel(t, path)
			if err := cfg.RemoveOutput(root, filepath.Join(root, test.output)); err == nil || !strings.Contains(err.Error(), "unsafe output") {
				t.Errorf("expected unsafe output rejection, got %v", err)
			}
			assertOutputSentinel(t, path)
		})
	}
}

func TestRemoveOutputAllowsOnlyDisjointWorkspaces(t *testing.T) {
	for _, output := range []string{"site", "nested/missing/site", ".mpress/export-abc/site", ".mpress/live-preview/revision"} {
		t.Run(output, func(t *testing.T) {
			root := t.TempDir()
			cfg := Default()
			protected := filepath.Join(root, ".mpress/translations/keep")
			writeOutputSentinel(t, protected)
			target := filepath.Join(root, output)
			writeOutputSentinel(t, filepath.Join(target, "obsolete"))
			if err := cfg.RemoveOutput(root, target); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Lstat(target); !os.IsNotExist(err) {
				t.Errorf("old output was not removed: %v", err)
			}
			assertOutputSentinel(t, protected)
			if err := cfg.RemoveOutput(root, target); err != nil {
				t.Fatalf("nonexistent output should remain safe: %v", err)
			}
		})
	}
}

func TestRemoveOutputHandlesSymlinkedPaths(t *testing.T) {
	for _, kind := range []string{"external parent", "external final", "source alias", "state alias", "dangling parent", "safe internal parent", "project root alias"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			cfg := Default()
			protected := filepath.Join(root, "content/keep")
			writeOutputSentinel(t, protected)
			writeOutputSentinel(t, filepath.Join(outside, "site/keep"))
			link, destination, output := filepath.Join(root, "linked"), outside, filepath.Join(root, "linked/site")
			allowed := false
			switch kind {
			case "external final":
				output = link
			case "source alias":
				destination = filepath.Join(root, "content")
			case "state alias":
				destination = filepath.Join(root, ".mpress")
				writeOutputSentinel(t, filepath.Join(destination, "keep"))
			case "dangling parent":
				destination = filepath.Join(outside, "missing")
			case "safe internal parent":
				destination = filepath.Join(root, "generated")
				writeOutputSentinel(t, filepath.Join(destination, "site/obsolete"))
				allowed = true
			case "project root alias":
				link, destination = filepath.Join(outside, "project"), root
				root, output, allowed = link, filepath.Join(link, "site"), true
			}
			if err := os.Symlink(destination, link); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			err := cfg.RemoveOutput(root, output)
			if allowed && err != nil || !allowed && (err == nil || !strings.Contains(err.Error(), "unsafe output")) {
				t.Errorf("allowed=%v, removal error=%v", allowed, err)
			}
			assertOutputSentinel(t, protected)
			assertOutputSentinel(t, filepath.Join(outside, "site/keep"))
			if _, err := os.Readlink(link); err != nil {
				t.Errorf("project alias link changed: %v", err)
			}
		})
	}
}

func TestRemoveOutputRespectsFilesystemCaseSemantics(t *testing.T) {
	root := t.TempDir()
	protected := filepath.Join(root, "content/keep")
	writeOutputSentinel(t, protected)
	alias := filepath.Join(root, "CONTENT")
	_, aliasErr := os.Stat(alias)
	cfg := Default()
	err := cfg.RemoveOutput(root, filepath.Join(alias, "generated"))
	if aliasErr == nil && err == nil {
		t.Fatal("case alias of source directory was accepted")
	}
	if os.IsNotExist(aliasErr) && err != nil {
		t.Fatalf("distinct output on a case-sensitive filesystem was rejected: %v", err)
	}
	assertOutputSentinel(t, protected)
}

func TestRemoveOutputProtectsWorktreeGitFile(t *testing.T) {
	root := t.TempDir()
	protected := filepath.Join(root, ".git")
	writeOutputSentinel(t, protected)
	cfg := Default()
	if err := cfg.RemoveOutput(root, protected); err == nil {
		t.Fatal("worktree Git file was accepted as output")
	}
	assertOutputSentinel(t, protected)
}

func writeOutputSentinel(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("must survive"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertOutputSentinel(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "must survive" {
		t.Errorf("protected file changed: %q, %v", data, err)
	}
}
