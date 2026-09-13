package translate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestResolveAPIKeyReadsPrivateProviderFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission behavior")
	}
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("OPENROUTER_API_KEY", "")
	directory := filepath.Join(root, "mpress")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "openrouter.env"), []byte("export OPENROUTER_API_KEY='private-key'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	if got := ResolveAPIKey(cfg); got != "private-key" {
		t.Fatalf("key = %q", got)
	}
}

func TestResolveAPIKeyRejectsPublicProviderFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission behavior")
	}
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("OPENROUTER_API_KEY", "")
	directory := filepath.Join(root, "mpress")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "openrouter.env"), []byte("OPENROUTER_API_KEY=exposed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ResolveAPIKey(config.Default()); got != "" {
		t.Fatalf("public file key = %q", got)
	}
}
