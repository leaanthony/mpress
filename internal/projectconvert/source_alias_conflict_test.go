package projectconvert

import (
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConversionPreservesRetargetedSourceAliasWithIdenticalContent(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	for _, name := range []string{"first.txt", "second.txt"} {
		writeConversionSource(t, filepath.Join(root, "assets", name), 0644, "# Shared content\n")
	}
	source := filepath.Join(root, "content", "page.md")
	first := filepath.Join("..", "assets", "first.txt")
	second := filepath.Join("..", "assets", "second.txt")
	if err := os.Symlink(first, source); err != nil {
		t.Fatal(err)
	}
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if err := os.Remove(source); err != nil {
			return false, err
		}
		if err := os.Symlink(second, source); err != nil {
			return false, err
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}
	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if err == nil || !strings.Contains(err.Error(), "changed after planning") {
		t.Fatalf("retargeted source alias was not rejected: %v", err)
	}
	if target, err := os.Readlink(source); err != nil || target != second {
		t.Fatalf("author's retargeted alias was lost: %q %v", target, err)
	}
	assertNoMPDTargets(t, cfg.ContentPath(root))
}
