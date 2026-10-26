package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
)

func testConfigurationRoot(t *testing.T) (*projectfs.FS, string) {
	t.Helper()
	root := t.TempDir()
	files, err := projectfs.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	return files, root
}

func requireConfigurationFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("configuration mode: %v", info.Mode().Perm())
	}
}

func TestCreateRootCreatesMissingConfiguration(t *testing.T) {
	files, root := testConfigurationRoot(t)
	cfg := Default()
	cfg.Site.Title = "Created without replacing"
	if err := CreateRoot(files, cfg); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, Filename)
	requireConfigurationFileMode(t, path, 0644)
	saved, err := Parse(mustReadFile(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != cfg.Site.Title {
		t.Fatalf("created configuration title: %q", saved.Site.Title)
	}
}

func TestCreateRootValidatesBeforeWriting(t *testing.T) {
	files, root := testConfigurationRoot(t)
	cfg := Default()
	cfg.Site.Title = " "
	if err := CreateRoot(files, cfg); err == nil {
		t.Fatal("invalid configuration was accepted")
	}
	if _, err := os.Lstat(filepath.Join(root, Filename)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid configuration was written: %v", err)
	}
}

func TestCreateRootRequiresProjectRoot(t *testing.T) {
	if err := CreateRoot(nil, Default()); err == nil {
		t.Fatal("missing project root was accepted")
	}
}

func TestCreateRootRefusesExistingLeaves(t *testing.T) {
	requireSymlinks := func(t *testing.T) {
		t.Helper()
		probe := filepath.Join(t.TempDir(), "probe")
		if err := os.Symlink("missing", probe); err != nil {
			t.Skipf("native symlinks unavailable: %v", err)
		}
	}
	sentinel := []byte("site:\n  title: Author leaf\n")
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, root string) (leaf, linkTarget string, mode os.FileMode)
	}{
		{
			name: "regular file",
			setup: func(t *testing.T, root string) (string, string, os.FileMode) {
				leaf := filepath.Join(root, Filename)
				if err := os.WriteFile(leaf, sentinel, 0600); err != nil {
					t.Fatal(err)
				}
				return leaf, "", os.FileMode(0600)
			},
		},
		{
			name: "internal alias",
			setup: func(t *testing.T, root string) (string, string, os.FileMode) {
				requireSymlinks(t)
				target := filepath.Join(root, "config", "author.yaml")
				if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(target, sentinel, 0640); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join("config", "author.yaml")
				if err := os.Symlink(link, filepath.Join(root, Filename)); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(root, Filename), link, 0
			},
		},
		{
			name: "dangling alias",
			setup: func(t *testing.T, root string) (string, string, os.FileMode) {
				requireSymlinks(t)
				link := "missing-author.yaml"
				if err := os.Symlink(link, filepath.Join(root, Filename)); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(root, Filename), link, 0
			},
		},
		{
			name: "external alias",
			setup: func(t *testing.T, root string) (string, string, os.FileMode) {
				requireSymlinks(t)
				outside := filepath.Join(t.TempDir(), "author.yaml")
				if err := os.WriteFile(outside, sentinel, 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(root, Filename)); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(root, Filename), outside, 0
			},
		},
		{
			name: "directory",
			setup: func(t *testing.T, root string) (string, string, os.FileMode) {
				leaf := filepath.Join(root, Filename)
				if err := os.MkdirAll(leaf, 0755); err != nil {
					t.Fatal(err)
				}
				return leaf, "", 0
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files, root := testConfigurationRoot(t)
			leaf, linkTarget, mode := tc.setup(t, root)
			err := CreateRoot(files, Default())
			if err == nil {
				t.Fatal("existing configuration leaf was not refused")
			}
			// Regular-file collisions keep the platform-independent existence
			// contract; alias and directory rejection is preserved without
			// assuming one errno on every platform.
			if (tc.name == "regular file" || runtime.GOOS != "windows") && !errors.Is(err, fs.ErrExist) {
				t.Fatalf("existing configuration leaf was not refused with an existence error: %v", err)
			}
			info, statErr := os.Lstat(leaf)
			if statErr != nil {
				t.Fatalf("existing configuration leaf changed: %v", statErr)
			}
			switch {
			case linkTarget != "":
				if info.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("configuration alias was replaced: %v", info.Mode())
				}
				if got, readErr := os.Readlink(leaf); readErr != nil || got != linkTarget {
					t.Fatalf("configuration alias text changed: %q %v", got, readErr)
				}
				switch {
				case filepath.IsAbs(linkTarget):
					if data, readErr := os.ReadFile(linkTarget); readErr != nil || !bytes.Equal(data, sentinel) {
						t.Fatalf("external alias target changed: %v %q", readErr, data)
					}
				case filepath.Clean(linkTarget) == filepath.Join("config", "author.yaml"):
					if data, readErr := os.ReadFile(filepath.Join(root, linkTarget)); readErr != nil || !bytes.Equal(data, sentinel) {
						t.Fatalf("internal alias target changed: %v %q", readErr, data)
					}
				default:
					if _, readErr := os.Lstat(filepath.Join(root, linkTarget)); !errors.Is(readErr, os.ErrNotExist) {
						t.Fatalf("dangling alias target was created or followed: %v", readErr)
					}
				}
			case tc.name == "directory":
				if !info.IsDir() {
					t.Fatalf("configuration directory was replaced: %v", info.Mode())
				}
			default:
				if data, readErr := os.ReadFile(leaf); readErr != nil || !bytes.Equal(data, sentinel) {
					t.Fatalf("configuration bytes changed: %v %q", readErr, data)
				}
				requireConfigurationFileMode(t, leaf, mode)
			}
		})
	}
}

func TestSaveRootStillReplacesExistingConfiguration(t *testing.T) {
	files, root := testConfigurationRoot(t)
	path := filepath.Join(root, Filename)
	if err := os.WriteFile(path, []byte("site:\n  title: Author leaf\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	cfg.Site.Title = "Ordinary editor save"
	if err := SaveRoot(files, cfg); err != nil {
		t.Fatal(err)
	}
	saved, err := Parse(mustReadFile(t, path))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Site.Title != cfg.Site.Title {
		t.Fatalf("ordinary configuration replace: %q", saved.Site.Title)
	}
	requireConfigurationFileMode(t, path, 0644)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
