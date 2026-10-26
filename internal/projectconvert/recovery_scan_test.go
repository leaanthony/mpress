package projectconvert

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/knowledge"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/site"
)

type conversionTreeEntry struct {
	Path string
	Mode os.FileMode
	Data string
	Link string
}

func snapshotConversionTree(t *testing.T, root string) []conversionTreeEntry {
	t.Helper()
	var entries []conversionTreeEntry
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		record := conversionTreeEntry{Path: path, Mode: info.Mode()}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			if record.Link, err = os.Readlink(path); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			record.Data = string(data)
		}
		entries = append(entries, record)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func requireConversionTarget(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ordinary document was not converted to %s: %v", path, err)
	}
}

func TestConversionDefaultScanSkipsRecoveryTrees(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	root, _ := openConversionProject(t, &cfg)
	content := cfg.ContentPath(root)

	// Ordinary hidden author directories and names that merely resemble the
	// reserved prefix still convert; only nested reserved directories are
	// recovery evidence.
	writeConversionSource(t, filepath.Join(content, ".authors", "draft.md"), 0644, "# Draft\n\nHidden author document.\n")
	writeConversionSource(t, filepath.Join(content, ".mpress-conversion-recovery", "note.md"), 0644, "# Similar\n\nNo trailing dash.\n")
	writeConversionSource(t, filepath.Join(content, ".mpress-conversion-recoveryx", "note.md"), 0644, "# Similar\n\nNo dash at all.\n")
	writeConversionSource(t, filepath.Join(content, recoveryPrefix+"file.md"), 0644, "# File\n\nRecovery-prefixed file, not directory.\n")

	recovery := filepath.Join(content, recoveryPrefix+"retained")
	writeConversionSource(t, filepath.Join(recovery, "content", "keep.md"), 0600, "# Keep\n\nAuthor evidence.\n")
	writeConversionSource(t, filepath.Join(recovery, "content", "keep.mpd"), 0640, "# Collision\n\nExisting target.\n")
	writeConversionSource(t, filepath.Join(recovery, "content", "nested", "note.md"), 0644, "# Nested\n\nNested evidence.\n")
	if err := os.Symlink("keep.md", filepath.Join(recovery, "content", "alias.md")); err != nil {
		t.Fatal(err)
	}
	before := snapshotConversionTree(t, recovery)

	result, err := Run(root, &cfg, "", "mpd")
	if err != nil || result.Count != 4 {
		t.Fatalf("default recursive conversion failed: %v %+v", err, result)
	}
	for _, target := range []string{
		filepath.Join(content, ".authors", "draft.mpd"),
		filepath.Join(content, ".mpress-conversion-recovery", "note.mpd"),
		filepath.Join(content, ".mpress-conversion-recoveryx", "note.mpd"),
		filepath.Join(content, recoveryPrefix+"file.mpd"),
	} {
		requireConversionTarget(t, target)
	}
	if after := snapshotConversionTree(t, recovery); !reflect.DeepEqual(before, after) {
		t.Fatalf("retry scan changed the named recovery tree:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestConversionDefaultReverseScanSkipsRecoveryTrees(t *testing.T) {
	requireSymlinks(t)
	cfg := config.Default()
	root, _ := openConversionProject(t, &cfg)
	content := cfg.ContentPath(root)
	writeConversionSource(t, filepath.Join(content, ".authors", "draft.mpd"), 0644, "# Draft\n\nHidden author document.\n")

	recovery := filepath.Join(content, recoveryPrefix+"retained")
	writeConversionSource(t, filepath.Join(recovery, "content", "keep.mpd"), 0600, "# Keep\n\nAuthor evidence.\n")
	writeConversionSource(t, filepath.Join(recovery, "content", "keep.md"), 0640, "# Collision\n\nExisting target.\n")
	writeConversionSource(t, filepath.Join(recovery, "content", "nested", "note.mpd"), 0644, "# Nested\n\nNested evidence.\n")
	if err := os.Symlink("keep.mpd", filepath.Join(recovery, "content", "alias.mpd")); err != nil {
		t.Fatal(err)
	}
	before := snapshotConversionTree(t, recovery)

	result, err := Run(root, &cfg, "", "markdown")
	if err != nil || result.Count != 1 {
		t.Fatalf("default reverse conversion failed: %v %+v", err, result)
	}
	requireConversionTarget(t, filepath.Join(content, ".authors", "draft.md"))
	if after := snapshotConversionTree(t, recovery); !reflect.DeepEqual(before, after) {
		t.Fatalf("reverse scan changed the named recovery tree:\nbefore %+v\nafter  %+v", before, after)
	}
}

func TestConversionExplicitRecoveryNamedRootsStayAuthoritative(t *testing.T) {
	t.Run("external directory", func(t *testing.T) {
		project := t.TempDir()
		cfg := config.Default()
		selected := filepath.Join(t.TempDir(), recoveryPrefix+"chosen")
		if err := os.MkdirAll(selected, 0755); err != nil {
			t.Fatal(err)
		}
		writeConversionSource(t, filepath.Join(selected, "page.md"), 0600, "# Selected\n\nConvert this document.\n")
		result, err := Run(project, &cfg, selected, "mpd")
		if err != nil || result.Count != 1 || result.Directory != selected {
			t.Fatalf("explicit recovery-named directory failed: %v %+v", err, result)
		}
		requireConversionTarget(t, filepath.Join(selected, "page.mpd"))
	})

	t.Run("configured content directory", func(t *testing.T) {
		cfg := config.Default()
		cfg.Build.ContentDir = recoveryPrefix + "content"
		root, _ := openConversionProject(t, &cfg)
		writeConversionSource(t, filepath.Join(cfg.ContentPath(root), "page.md"), 0600, "# Content\n\nConvert this document.\n")
		result, err := Run(root, &cfg, "", "mpd")
		if err != nil || result.Count != 1 {
			t.Fatalf("recovery-named content directory failed: %v %+v", err, result)
		}
		requireConversionTarget(t, filepath.Join(cfg.ContentPath(root), "page.mpd"))
	})

	t.Run("borrowed nested directory", func(t *testing.T) {
		project := t.TempDir()
		cfg := config.Default()
		selected := filepath.Join(cfg.ContentPath(project), recoveryPrefix+"chosen")
		writeConversionSource(t, filepath.Join(selected, "page.md"), 0600, "# Borrowed\n\nConvert this document.\n")
		files, err := projectfs.Open(project)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = files.Close() }()
		result, err := RunRoot(project, files, &cfg, filepath.Join("content", recoveryPrefix+"chosen"), "mpd")
		if err != nil || result.Count != 1 {
			t.Fatalf("borrowed recovery-named directory failed: %v %+v", err, result)
		}
		if _, err := files.Stat(filepath.Join("content", recoveryPrefix+"chosen", "page.mpd")); err != nil {
			t.Fatalf("borrowed conversion did not publish inside its boundary: %v", err)
		}
	})
}

func TestConversionRetryAfterFailurePreservesNamedRecoveryTree(t *testing.T) {
	cfg := config.Default()
	root, files := openConversionProject(t, &cfg)
	if err := os.WriteFile(filepath.Join(root, config.Filename), []byte("site:\n  title: Recovery retry\nbuild:\n  contentDir: content\n  outputDir: site\n"), 0644); err != nil {
		t.Fatal(err)
	}
	content := cfg.ContentPath(root)
	a := writeConversionSource(t, filepath.Join(content, "a.md"), 0600, "# Alpha\n\nOriginal alpha.\n")
	b := writeConversionSource(t, filepath.Join(content, "b.md"), 0640, "# Beta\n\nOriginal beta.\n")
	aTarget := filepath.Join(content, "a.mpd")
	publicationFailure := errors.New("beta target publication failed")
	canary := "ZQXCANARYTOKEN7F3A"
	keep := []byte("# Keep this author evidence\n\nDo not convert.\n\n" + canary + "\n")
	injected := false
	ops := defaultConversionOps
	ops.writeNew = func(f *projectfs.FS, name string, data []byte, mode os.FileMode) (bool, error) {
		if filepath.Clean(name) == filepath.Join(content, "b.mpd") {
			return false, publicationFailure
		}
		return f.WriteNewAtomicMode(name, data, mode)
	}
	// Replace the published target with an author directory after rollback
	// reads it, reproducing a failure that retains a named recovery directory.
	ops.readFile = func(f *projectfs.FS, name string) ([]byte, error) {
		data, err := f.ReadFile(name)
		if filepath.Clean(name) == aTarget && !injected && err == nil {
			injected = true
			if err := os.Remove(aTarget); err != nil {
				return nil, err
			}
			if err := os.Mkdir(aTarget, 0755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(filepath.Join(aTarget, "keep.md"), keep, 0600); err != nil {
				return nil, err
			}
		}
		return data, err
	}

	_, err := runConversionWithOps(root, &cfg, "", "mpd", files, files, ops)
	if !injected {
		t.Fatal("recovery directory injection did not occur")
	}
	if !errors.Is(err, publicationFailure) || !strings.Contains(err.Error(), recoveryPrefix) {
		t.Fatalf("failure did not report the retained recovery directory: %v", err)
	}
	assertConversionSource(t, filepath.Join(content, "a.md"), a, 0600)
	assertConversionSource(t, filepath.Join(content, "b.md"), b, 0640)

	roots, globErr := filepath.Glob(filepath.Join(content, recoveryPrefix+"*"))
	if globErr != nil || len(roots) != 1 {
		t.Fatalf("named recovery directory was not retained: %v %v", roots, globErr)
	}
	captured := filepath.Join(roots[0], "content", "a.mpd.recovery")
	before := snapshotConversionTree(t, roots[0])

	result, err := Run(root, &cfg, "", "mpd")
	if err != nil || result.Count != 2 {
		t.Fatalf("retry after failure failed: %v %+v", err, result)
	}
	for _, target := range []string{aTarget, filepath.Join(content, "b.mpd")} {
		requireConversionTarget(t, target)
	}
	if _, err := os.Stat(filepath.Join(content, "a.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retry retained a converted source: %v", err)
	}
	assertConversionSource(t, filepath.Join(captured, "keep.md"), keep, 0600)
	if after := snapshotConversionTree(t, roots[0]); !reflect.DeepEqual(before, after) {
		t.Fatalf("retry changed retained recovery evidence:\nbefore %+v\nafter  %+v", before, after)
	}
	if _, err := os.Stat(filepath.Join(captured, "keep.mpd")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retry converted a retained recovery document: %v", err)
	}

	// A later reader build must publish the converted pages without reading the
	// retained originals.
	beforeBuild := snapshotConversionTree(t, roots[0])
	buildResult, err := site.Build(root, site.BuildOptions{Strict: true})
	if err != nil {
		t.Fatalf("build after retry: %v; diagnostics: %#v", err, buildResult.Diagnostics)
	}
	output := filepath.Join(root, "site")
	err = filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.HasSuffix(path, ".gz") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), canary) {
			t.Errorf("generated artifact %s contains retained recovery content", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	store, err := knowledge.Load(output)
	if err != nil {
		t.Fatal(err)
	}
	if results := store.Search(knowledge.SearchOptions{Query: canary}); len(results) != 0 {
		t.Errorf("knowledge search found retained recovery content: %#v", results)
	}
	if after := snapshotConversionTree(t, roots[0]); !reflect.DeepEqual(beforeBuild, after) {
		t.Fatalf("build changed retained recovery evidence:\nbefore %+v\nafter  %+v", beforeBuild, after)
	}
}
