package knowledge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/routes"
)

func generatedBoundaryBundle(t *testing.T) string {
	t.Helper()
	output := t.TempDir()
	pages := map[string][]*content.Page{"en": {{Title: "Home", PlainText: "compiler instructions", HTML: "<p>compiler instructions</p>"}}}
	if err := Generate(output, config.Default(), pages); err != nil {
		t.Fatal(err)
	}
	return output
}
func TestKnowledgeRejectsNonportableArtifactNames(t *testing.T) {
	for _, item := range []struct{ label, name string }{
		{"traversal", "../pages.json"}, {"noncanonical", "nested/../pages.json"}, {"absolute", "/pages.json"}, {"volume", "C:/pages.json"}, {"backslash", `nested\pages.json`}, {"repeated-slash", "nested//pages.json"}, {"empty", ""}, {"device", "NUL.json"}, {"trailing-dot", "pages.json."}, {"trailing-space", "pages.json "},
	} {
		t.Run(item.label, func(t *testing.T) {
			name := item.name
			output := generatedBoundaryBundle(t)
			manifestPath := filepath.Join(output, Directory, ManifestFile)
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			var manifest Manifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				t.Fatal(err)
			}
			manifest.Artifacts.Pages = name
			data, err = json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifestPath, data, 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(output); !errors.Is(err, routes.ErrUnsafe) {
				t.Fatalf("manifest name %q was not rejected before artifact IO: %v", name, err)
			}
		})
	}
}
func TestKnowledgeRetainsInternalArtifactAliases(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		t.Run(map[bool]string{false: "relative", true: "absolute"}[absolute], func(t *testing.T) {
			output := generatedBoundaryBundle(t)
			before, err := Load(output)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(output, Directory, PagesFile)
			if err := os.Rename(path, path+".original"); err != nil {
				t.Fatal(err)
			}
			target := PagesFile + ".original"
			if absolute {
				target = path + ".original"
			}
			knowledgeSymlink(t, target, path)
			after, err := LoadAll(output)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.Pages, after.Pages) || !reflect.DeepEqual(before.Search(SearchOptions{Query: "compiler"}), after.Search(SearchOptions{Query: "compiler"})) {
				t.Fatal("internal alias changed knowledge content or search")
			}
		})
	}
}
func TestKnowledgeRejectsArtifactsOutsideBundleWithinSite(t *testing.T) {
	output := generatedBoundaryBundle(t)
	path := filepath.Join(output, Directory, PagesFile)
	if err := os.Rename(path, filepath.Join(output, "private.json")); err != nil {
		t.Fatal(err)
	}
	knowledgeSymlink(t, "../private.json", path)
	if _, err := Load(output); !errors.Is(err, projectfs.ErrOutside) {
		t.Fatalf("artifact read escaped its bundle: %v", err)
	}
}
func TestKnowledgeMountedMissingArtifactsRemainFatal(t *testing.T) {
	output := generatedBoundaryBundle(t)
	snapshot := filepath.Join(output, "versions", "v1")
	pages := map[string][]*content.Page{"en": {{Title: "Snapshot", PlainText: "snapshot", HTML: "<p>snapshot</p>"}}}
	if err := Generate(snapshot, config.Default(), pages); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(snapshot, Directory, ChunksFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAll(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing version artifact was silently skipped: %v", err)
	}
	if err := os.Remove(filepath.Join(snapshot, Directory, ManifestFile)); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAll(output); err != nil {
		t.Fatalf("version without a knowledge manifest must remain optional: %v", err)
	}
}

func knowledgeSymlink(t *testing.T, target, name string) {
	t.Helper()
	if err := os.Symlink(target, name); err != nil {
		probe := t.TempDir()
		if probeErr := os.Symlink("target", filepath.Join(probe, "link")); probeErr == nil {
			t.Fatalf("native symlinks work but fixture link failed: %v", err)
		}
		t.Skipf("native symlinks unavailable: %v", err)
	}
}

// Optional large-corpus qualification; a skipped run is not qualification.
func TestExplicitKnowledgeCorpus(t *testing.T) {
	output := os.Getenv("MPRESS_KNOWLEDGE_CORPUS")
	if output == "" {
		t.Skip("set MPRESS_KNOWLEDGE_CORPUS to a built pinned corpus")
	}
	store, err := LoadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Pages) < len(store.Manifest.Pages) || len(store.Chunks) == 0 {
		t.Fatal("corpus did not retain its manifest pages and chunks")
	}
	if len(store.Search(SearchOptions{Query: "wails"})) == 0 {
		t.Fatal("full Wails corpus search lost its results")
	}
	t.Logf("verified digest %s: %d pages, %d chunks", store.Manifest.Digest, len(store.Pages), len(store.Chunks))
}

func BenchmarkExplicitKnowledgeLoadAll(b *testing.B) {
	output := os.Getenv("MPRESS_KNOWLEDGE_CORPUS")
	if output == "" {
		b.Skip("set MPRESS_KNOWLEDGE_CORPUS to a built pinned corpus")
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		store, err := LoadAll(output)
		if err != nil {
			b.Fatal(err)
		}
		if len(store.Pages) == 0 {
			b.Fatal("empty corpus")
		}
	}
}
