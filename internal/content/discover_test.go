package content

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
)

func writeDiscoveryFixture(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func requireDiscoverySymlinks(t *testing.T) {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.Symlink("missing", probe); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
}

func TestDiscoverSkipsNestedConversionRecoveryTrees(t *testing.T) {
	requireDiscoverySymlinks(t)
	root := t.TempDir()
	content := filepath.Join(root, "content")
	writeDiscoveryFixture(t, content, "index.md", "# Home\n")
	writeDiscoveryFixture(t, content, "guide.md", "# Guide\n")
	writeDiscoveryFixture(t, content, "fr/index.md", "# Accueil\n")

	// Ordinary hidden author content, names that merely resemble the reserved
	// prefix, and recovery-prefixed regular files remain discoverable.
	writeDiscoveryFixture(t, content, ".authors/draft.md", "# Draft\n")
	writeDiscoveryFixture(t, content, ".mpress-conversion-recovery/note.md", "# Similar\n")
	writeDiscoveryFixture(t, content, ".mpress-conversion-recoveryx/note.md", "# Similar\n")
	writeDiscoveryFixture(t, content, ".mpress-conversion-recovery-file.md", "# File\n")
	if err := os.Symlink("index.md", filepath.Join(content, "alias.md")); err != nil {
		t.Fatal(err)
	}

	recovery := filepath.Join(content, ".mpress-conversion-recovery-retained")
	writeDiscoveryFixture(t, recovery, "content/a.mpd.recovery/keep.md", "# Keep\n\nAuthor evidence.\n")
	writeDiscoveryFixture(t, recovery, "content/a.mpd.recovery/keep.mpd", "# Keep\n\nAuthor evidence.\n")
	writeDiscoveryFixture(t, recovery, "content/index.md", "---\ntitle: Collision\nslug: /\n---\n\n# Collision\n")
	writeDiscoveryFixture(t, recovery, "content/malformed.md", "---\ntitle: [unterminated\n---\n\n# Malformed\n")
	if err := os.Symlink("keep.md", filepath.Join(recovery, "content", "a.mpd.recovery", "alias.md")); err != nil {
		t.Fatal(err)
	}
	// A recovery tree nested under ordinary hidden authoring content is also
	// owned evidence, not published content.
	writeDiscoveryFixture(t, content, ".authors/.mpress-conversion-recovery-nested/leak.md", "# Leak\n")

	want := map[string][]string{
		"en": {
			".authors/draft.md",
			".mpress-conversion-recovery-file.md",
			".mpress-conversion-recovery/note.md",
			".mpress-conversion-recoveryx/note.md",
			"alias.md",
			"guide.md",
			"index.md",
		},
		"fr": {"index.md"},
	}

	t.Run("default", func(t *testing.T) {
		got, err := Discover(content+string(filepath.Separator), []string{"en", "fr"}, "en")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("discovered content = %#v, want %#v", got, want)
		}
	})

	t.Run("borrowed root", func(t *testing.T) {
		files, err := projectfs.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = files.Close() }()
		got, err := DiscoverRoot(files, "content", []string{"en", "fr"}, "en")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("discovered borrowed content = %#v, want %#v", got, want)
		}
	})
}

func TestDiscoverRecoveryNamedContentRootStaysAuthoritative(t *testing.T) {
	requireDiscoverySymlinks(t)
	root := t.TempDir()
	name := ".mpress-conversion-recovery-content"
	content := filepath.Join(root, name)
	writeDiscoveryFixture(t, content, "index.md", "# Home\n")
	writeDiscoveryFixture(t, content, "fr/index.md", "# Accueil\n")
	if err := os.Symlink("index.md", filepath.Join(content, "alias.md")); err != nil {
		t.Fatal(err)
	}
	writeDiscoveryFixture(t, content, ".mpress-conversion-recovery-retained/leak.md", "# Leak\n")

	want := map[string][]string{
		"en": {"alias.md", "index.md"},
		"fr": {"index.md"},
	}

	for _, dir := range []string{content, content + string(filepath.Separator)} {
		got, err := Discover(dir, []string{"en", "fr"}, "en")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("discovered content under %q = %#v, want %#v", dir, got, want)
		}
	}

	files, err := projectfs.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	got, err := DiscoverRoot(files, name, []string{"en", "fr"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discovered borrowed content = %#v, want %#v", got, want)
	}
}

func TestDiscoverKeepsRecoveryPrefixRegularFiles(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	for _, name := range []string{
		".mpress-conversion-recovery-retained.md",
		".mpress-conversion-recovery-retained.mpd",
	} {
		writeDiscoveryFixture(t, content, name, "# Authored\n")
	}
	got, err := Discover(content, []string{"en"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".mpress-conversion-recovery-retained.md", ".mpress-conversion-recovery-retained.mpd"}
	if !reflect.DeepEqual(got["en"], want) {
		t.Fatalf("discovered content = %#v, want %#v", got["en"], want)
	}
}

func TestDiscoverSkipsRecoveryDirectoryNamedWithPrefixOnly(t *testing.T) {
	root := t.TempDir()
	content := filepath.Join(root, "content")
	writeDiscoveryFixture(t, content, "index.md", "# Home\n")
	writeDiscoveryFixture(t, content, "keep.md", "# Keep\n")
	writeDiscoveryFixture(t, content, ".mpress-conversion-recovery-/hidden.md", "# Hidden\n")
	got, err := Discover(content, []string{"en"}, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"index.md", "keep.md"}
	if !reflect.DeepEqual(got["en"], want) {
		t.Fatalf("discovered content = %#v, want %#v", got["en"], want)
	}
}
