package config

import (
	"fmt"
	"path/filepath"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// SafeVersionPath protects project inputs, state and current output from a
// misconfigured artifact store or a redirected snapshot directory. Validate
// the store itself as well as individual mutation targets before removal.
func (c Config) SafeVersionPath(project, path string) error {
	root, err := projectfs.CanonicalPath(project)
	if err != nil {
		return err
	}
	resolved, err := projectfs.CanonicalPath(path)
	if err != nil {
		return fmt.Errorf("unsafe version path %s: %w", path, err)
	}
	if !pathBelow(root, resolved) {
		return fmt.Errorf("unsafe version path %s: must stay inside the project", path)
	}
	protected := append(c.protectedInputs(project), c.OutputPath(project), filepath.Join(project, ".mpress", "live-preview"))
	for _, input := range protected {
		canonical, err := projectfs.CanonicalPath(input)
		if err != nil {
			return fmt.Errorf("unsafe version path %s: resolve protected path %s: %w", path, input, err)
		}
		if pathsOverlap(resolved, canonical) {
			return fmt.Errorf("unsafe version path %s: overlaps protected path %s", path, input)
		}
	}
	return nil
}
