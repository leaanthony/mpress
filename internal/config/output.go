package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// SafeOutputPath resolves an output directory and rejects paths that can erase
// project inputs or persistent state. Missing output directories are resolved
// through their nearest existing ancestor, including any symbolic links. The
// final component is retained so removal unlinks an output symlink rather than
// deleting the directory it points to.
func (c Config) SafeOutputPath(project, output string) (string, error) {
	return c.safeOutputPath(project, output, false)
}

// SafePreviewPool applies the same input/state overlap policy to the dev
// server's managed pool, whose startup cleanup removes the pool itself.
func (c Config) SafePreviewPool(project string) (string, error) {
	return c.safeOutputPath(project, filepath.Join(project, ".mpress", "live-preview"), true)
}

func (c Config) safeOutputPath(project, output string, previewPool bool) (string, error) {
	root, err := projectfs.CanonicalPath(project)
	if err != nil {
		return "", fmt.Errorf("unsafe output directory %s: resolve project: %w", output, err)
	}
	resolved, err := projectfs.CanonicalPath(output)
	if err != nil {
		return "", fmt.Errorf("unsafe output directory %s: %w", output, err)
	}
	if !pathBelow(root, resolved) {
		return "", fmt.Errorf("unsafe output directory %s: must stay inside the project", output)
	}
	protected := append(c.protectedInputs(project), c.ArtifactsPath(project))
	if previewPool {
		protected = append(protected, c.OutputPath(project))
	}
	for _, path := range protected {
		canonical, err := projectfs.CanonicalPath(path)
		if err != nil {
			return "", fmt.Errorf("unsafe output directory %s: resolve protected path %s: %w", output, path, err)
		}
		if pathsOverlap(resolved, canonical) {
			return "", fmt.Errorf("unsafe output directory %s: overlaps protected path %s", output, path)
		}
	}
	state, err := projectfs.CanonicalPath(filepath.Join(project, ".mpress"))
	if err != nil {
		return "", fmt.Errorf("unsafe output directory %s: resolve project state: %w", output, err)
	}
	managedPool := previewPool && resolved == filepath.Join(state, "live-preview")
	if pathsOverlap(resolved, state) && !temporaryOutput(state, resolved) && !managedPool {
		return "", fmt.Errorf("unsafe output directory %s: overlaps project state", output)
	}
	parent, err := projectfs.CanonicalPath(filepath.Dir(output))
	if err != nil {
		return "", fmt.Errorf("unsafe output directory %s: resolve parent: %w", output, err)
	}
	return filepath.Join(parent, filepath.Base(output)), nil
}

func (c Config) protectedInputs(project string) []string {
	protected := []string{
		c.ContentPath(project), c.StaticPath(project), filepath.Join(project, Filename),
		filepath.Join(c.ContentPath(project), c.Build.NavFile),
		filepath.Join(project, ".git"),
		filepath.Join(project, c.Translation.StateDir),
		filepath.Join(project, ".mpress", "cache"), filepath.Join(project, ".mpress", "backups"),
		filepath.Join(project, ".mpress", "onboarding"),
	}
	for _, language := range c.Site.Languages {
		if language != c.Site.DefaultLanguage {
			protected = append(protected, filepath.Join(c.ContentPath(project), language, c.Build.NavFile))
		}
	}
	for _, path := range []string{c.Translation.Glossary, c.Translation.StyleGuide, c.Contribution.Guide} {
		if path != "" {
			protected = append(protected, filepath.Join(project, path))
		}
	}
	if c.Build.CustomCSS != "" {
		protected = append(protected, filepath.Join(project, c.Build.CustomCSS))
	}
	return protected
}

// RemoveOutput applies the shared build/clean policy immediately before
// removal. Root.RemoveAll also confines deletion if a parent changes to an
// external symlink after validation; it never removes the project root itself.
func (c Config) RemoveOutput(project, output string) error {
	resolved, err := c.SafeOutputPath(project, output)
	if err != nil {
		return err
	}
	root, err := projectfs.CanonicalPath(project)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == "." || !filepath.IsLocal(relative) {
		return fmt.Errorf("unsafe output directory %s", output)
	}
	directory, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.RemoveAll(relative)
}

func pathBelow(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != "." && filepath.IsLocal(relative)
}

func pathsOverlap(first, second string) bool {
	forward, err := filepath.Rel(first, second)
	if err == nil && filepath.IsLocal(forward) {
		return true
	}
	backward, err := filepath.Rel(second, first)
	if err == nil && filepath.IsLocal(backward) {
		return true
	}
	// EvalSymlinks need not normalize spelling on a case-insensitive volume.
	// Compare existing directory identities so a case alias cannot hide overlap.
	return physicalAncestor(first, second) || physicalAncestor(second, first)
}

func physicalAncestor(parent, child string) bool {
	info, err := os.Stat(parent)
	if err != nil {
		return false
	}
	for {
		other, err := os.Stat(child)
		if err == nil && os.SameFile(info, other) {
			return true
		}
		next := filepath.Dir(child)
		if next == child {
			return false
		}
		child = next
	}
}

// Only the existing export and live-preview workspaces may contain output
// inside .mpress. Never remove .mpress or a workspace's containing directory.
func temporaryOutput(state, output string) bool {
	relative, err := filepath.Rel(state, output)
	if err != nil || !filepath.IsLocal(relative) {
		return false
	}
	parts := strings.Split(relative, string(filepath.Separator))
	return len(parts) >= 2 && (parts[0] == "live-preview" || (strings.HasPrefix(parts[0], "export-") && len(parts[0]) > len("export-")))
}
