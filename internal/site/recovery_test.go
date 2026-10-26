package site

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/knowledge"
)

const retainedRecoveryName = ".mpress-conversion-recovery-retained"
const retainedRecoveryCanary = "ZQXCANARYTOKEN7F3A"

type retainedTreeEntry struct {
	Path string
	Mode os.FileMode
	Data string
	Link string
}

func snapshotRetainedTree(t *testing.T, root string) []retainedTreeEntry {
	t.Helper()
	var entries []retainedTreeEntry
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		record := retainedTreeEntry{Path: path, Mode: info.Mode()}
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

func assertRecoveryExcluded(t *testing.T, output string, canary string, pages int) {
	t.Helper()
	err := filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(output, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.Contains(rel, retainedRecoveryName) {
			t.Errorf("generated output contains a retained recovery path: %s", rel)
		}
		if strings.HasSuffix(rel, ".gz") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), canary) {
			t.Errorf("generated output %s contains retained recovery content", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"search-index.json", "fr/search-index.json", "sitemap.xml", "llms.txt", "mpress-manifest.json", "robots.txt"} {
		data, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("reader artifact %s: %v", name, err)
		}
		if strings.Contains(string(data), canary) || strings.Contains(string(data), retainedRecoveryName) {
			t.Errorf("reader artifact %s exposes retained recovery content: %s", name, data)
		}
	}
	store, err := knowledge.Load(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Pages) != pages {
		t.Errorf("knowledge bundle described %d pages, want %d: %#v", len(store.Pages), pages, store.Pages)
	}
	for _, page := range store.Pages {
		if strings.Contains(page.Source, retainedRecoveryName) || strings.Contains(page.Text, canary) {
			t.Errorf("knowledge page exposes retained recovery content: %#v", page)
		}
	}
	for _, chunk := range store.Chunks {
		if strings.Contains(chunk.Source, retainedRecoveryName) || strings.Contains(chunk.Text, canary) {
			t.Errorf("knowledge chunk exposes retained recovery content: %#v", chunk)
		}
	}
	if results := store.Search(knowledge.SearchOptions{Query: canary}); len(results) != 0 {
		t.Errorf("knowledge search found retained recovery content: %#v", results)
	}
}

func TestBuildExcludesRetainedConversionRecoveryTrees(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "mpress.yaml", `site:
  title: Recovery test
  description: Reader artifacts must omit retained conversion originals
  baseURL: https://docs.example.test
  defaultLanguage: en
  languages: [en, fr]
  defaultLanguageAtRoot: true
build:
  contentDir: content
  staticDir: static
  outputDir: site
`)
	writeFixture(t, root, "content/index.md", "---\ntitle: Home\n---\n\n# Welcome\n\nPublished home page.\n")
	writeFixture(t, root, "content/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n\nPublished guide page.\n")
	writeFixture(t, root, "content/fr/index.md", "---\ntitle: Accueil\n---\n\n# Bonjour\n\nPage française publiée.\n")

	// Ordinary hidden author content, similar names and recovery-prefixed
	// regular files remain ordinary published content.
	writeFixture(t, root, "content/.authors/notes.md", "---\ntitle: Author notes\n---\n\n# Author notes\n\nHIDDEN-AUTHOR-MARKER\n")
	writeFixture(t, root, "content/.mpress-conversion-recovery/note.md", "---\ntitle: Similar\n---\n\n# Similar\n\nSIMILAR-NO-DASH-MARKER\n")
	writeFixture(t, root, "content/.mpress-conversion-recoveryx/note.md", "---\ntitle: Similar X\n---\n\n# Similar X\n\nSIMILAR-X-MARKER\n")
	writeFixture(t, root, "content/.mpress-conversion-recovery-file.md", "---\ntitle: Prefix file\n---\n\n# Prefix file\n\nPREFIX-FILE-MARKER\n")

	recovery := filepath.Join(root, "content", retainedRecoveryName)
	writeFixture(t, root, "content/"+retainedRecoveryName+"/content/a.mpd.recovery/keep.md", "---\ntitle: Retained\n---\n\n# Retained\n\n"+retainedRecoveryCanary+"\n")
	writeFixture(t, root, "content/"+retainedRecoveryName+"/content/guide.md", "---\ntitle: Collision\nslug: guide\n---\n\n# Collision\n\n"+retainedRecoveryCanary+"\n")
	writeFixture(t, root, "content/"+retainedRecoveryName+"/content/malformed.md", "---\ntitle: [unterminated\n---\n\n# Malformed\n\n"+retainedRecoveryCanary+"\n")
	if err := os.Symlink("a.mpd.recovery/keep.md", filepath.Join(recovery, "content", "alias.md")); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	before := snapshotRetainedTree(t, recovery)

	builds := []struct {
		name    string
		options BuildOptions
		output  string
	}{
		{name: "strict", options: BuildOptions{Strict: true}, output: "site"},
		{name: "draft preview", options: BuildOptions{IncludeDrafts: true, OutputDir: "preview"}, output: "preview"},
	}
	for _, build := range builds {
		t.Run(build.name, func(t *testing.T) {
			result, err := Build(root, build.options)
			if err != nil {
				t.Fatalf("build: %v; diagnostics: %#v", err, result.Diagnostics)
			}
			if result.Pages != 7 {
				t.Fatalf("published %d pages, want 7: %#v", result.Pages, result.Diagnostics)
			}
			home, err := os.ReadFile(filepath.Join(root, build.output, "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(home), "Published home page.") {
				t.Errorf("ordinary page is not readable: %s", home)
			}
			french, err := os.ReadFile(filepath.Join(root, build.output, "fr", "index.html"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(french), "Page française publiée.") {
				t.Errorf("ordinary translated page is not readable: %s", french)
			}
			for _, marker := range []string{
				".authors/notes/index.html",
				".mpress-conversion-recovery/note/index.html",
				".mpress-conversion-recoveryx/note/index.html",
				".mpress-conversion-recovery-file/index.html",
			} {
				path := filepath.Join(root, build.output, filepath.FromSlash(marker))
				if _, err := os.Stat(path); err != nil {
					t.Errorf("ordinary authored page %s is missing: %v", marker, err)
				}
			}
			assertRecoveryExcluded(t, filepath.Join(root, build.output), retainedRecoveryCanary, 7)
		})
	}
	if after := snapshotRetainedTree(t, recovery); !reflect.DeepEqual(before, after) {
		t.Fatalf("build changed retained recovery evidence:\nbefore %+v\nafter  %+v", before, after)
	}
}
