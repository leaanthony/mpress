package knowledge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

func TestKnowledgeRejectsDuplicateArtifactIdentities(t *testing.T) {
	for _, kind := range []string{"page", "term"} {
		t.Run(kind, func(t *testing.T) {
			output := generatedBoundaryBundle(t)
			duplicateKnowledgeIdentity(t, output, kind)
			if _, err := Load(output); err == nil || !strings.Contains(err.Error(), "duplicate") {
				t.Fatalf("digest-valid duplicate %s identity was accepted: %v", kind, err)
			}
		})
	}
}
func duplicateKnowledgeIdentity(t *testing.T, output, kind string) {
	t.Helper()
	root := filepath.Join(output, Directory)
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	var manifest Manifest
	if err := json.Unmarshal(read(ManifestFile), &manifest); err != nil {
		t.Fatal(err)
	}
	names := []string{manifest.Artifacts.Pages, manifest.Artifacts.Chunks, manifest.Artifacts.Index}
	data := [][]byte{read(names[0]), read(names[1]), read(names[2])}
	var value any
	var position int
	switch kind {
	case "page":
		var pages []Page
		if err := json.Unmarshal(data[0], &pages); err != nil {
			t.Fatal(err)
		}
		value = append(pages, pages[0])
		position = 0
	case "term":
		var index Index
		if err := json.Unmarshal(data[2], &index); err != nil {
			t.Fatal(err)
		}
		index.Terms = append(index.Terms, index.Terms[0])
		value = index
		position = 2
	}
	updated, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	data[position] = append(updated, '\n')
	if err := os.WriteFile(filepath.Join(root, names[position]), data[position], 0644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.New()
	for _, artifact := range data {
		_, _ = sum.Write(artifact)
	}
	manifest.Digest = hex.EncodeToString(sum.Sum(nil))
	updated, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ManifestFile), updated, 0644); err != nil {
		t.Fatal(err)
	}
}
func TestKnowledgeSnapshotMatchingCurrentVersionHasDistinctResources(t *testing.T) {
	cfg := config.Default()
	cfg.Version.Current = "v1"
	output := t.TempDir()
	current := &content.Page{Title: "Current", URLPath: "guide", PlainText: "current instructions", HTML: "<p>current instructions</p>"}
	snapshot := &content.Page{Title: "Snapshot", URLPath: "guide", PlainText: "legacy instructions", HTML: "<h2>Alpha</h2><p>legacy instructions</p><h2>Beta</h2><p>legacy second section</p>"}
	for dir, page := range map[string]*content.Page{output: current, filepath.Join(output, "versions", "v1"): snapshot} {
		if err := Generate(dir, cfg, map[string][]*content.Page{"en": {page}}); err != nil {
			t.Fatal(err)
		}
	}
	store, err := LoadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	assertKnowledgeResourceSeparation(t, store)
	again, err := LoadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	if store.Pages[0].ResourceURI != again.Pages[0].ResourceURI || store.Pages[1].ResourceURI != again.Pages[1].ResourceURI {
		t.Fatal("mounted resource identities are not deterministic")
	}
	// Advancing the current release must not change already-mounted resource IDs.
	oldLegacy := store.Search(SearchOptions{Query: "legacy"})[0]
	oldSnapshot := oldLegacy.PageResource
	cfg.Version.Current = "v2"
	if err := Generate(output, cfg, map[string][]*content.Page{"en": {current}}); err != nil {
		t.Fatal(err)
	}
	advanced, err := LoadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	legacy := advanced.Search(SearchOptions{Query: "legacy"})
	if len(legacy) != 1 || (legacy[0].PageResource != oldSnapshot || legacy[0].ResourceURI != oldLegacy.ResourceURI) {
		t.Fatal("snapshot resource identity changed when current release advanced")
	}

}

func assertKnowledgeResourceSeparation(t *testing.T, store *Store) {
	t.Helper()
	if len(store.Pages) != 2 || len(store.Chunks) != 3 {
		t.Fatalf("lost current or snapshot content: %d pages/%d chunks", len(store.Pages), len(store.Chunks))
	}
	for _, page := range store.Pages {
		resolved, ok := store.Page(page.ID)
		if !ok || resolved.URL != page.URL || resolved.Title != page.Title {
			t.Fatalf("page identity resolves to another version: %+v -> %+v", page, resolved)
		}
	}
	for _, chunk := range store.Chunks {
		resolved, ok := store.Chunk(chunk.ID)
		if !ok || resolved.URL != chunk.URL || resolved.Text != chunk.Text {
			t.Fatalf("chunk identity resolves to another version: %+v -> %+v", chunk, resolved)
		}
	}
	if store.Pages[0].ResourceURI == store.Pages[1].ResourceURI || store.Chunks[0].ResourceURI == store.Chunks[1].ResourceURI {
		t.Fatal("current and same-labelled snapshot share resource URIs")
	}
	for _, query := range []string{"current", "legacy"} {
		result := store.Search(SearchOptions{Query: query})
		if len(result) != 1 {
			t.Fatalf("search %s did not distinguish current/snapshot: %+v", query, result)
		}
	}
}

func TestKnowledgeLegacyDuplicateChunkIDsRemainReadable(t *testing.T) {
	for _, threshold := range []int{1, 20 << 20} {
		t.Run(fmt.Sprint(threshold), func(t *testing.T) {
			output, expected := legacySectionBundle(t, threshold)
			manifestPath := filepath.Join(output, Directory, ManifestFile)
			before := readIdentityFixture(t, manifestPath)
			store, err := Load(output)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(expected, store.Chunks) {
				t.Fatal("legacy repair does not match canonical generated identities/content")
			}
			first := store.Search(SearchOptions{Query: "first"})
			second := store.Search(SearchOptions{Query: "second"})
			if len(first) != 1 || len(second) != 1 || first[0].ChunkID == second[0].ChunkID || !strings.Contains(first[0].Snippet, "first text") || !strings.Contains(second[0].Snippet, "second text") {
				t.Fatalf("legacy index still aliases sections: %+v / %+v", first, second)
			}
			again, err := Load(output)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(store.Chunks, again.Chunks) {
				t.Fatal("legacy repaired identities are not deterministic")
			}
			after := readIdentityFixture(t, manifestPath)
			if !bytes.Equal(before, after) {
				t.Fatal("legacy loader changed the artifact manifest")
			}
		})
	}
}
func legacySectionBundle(t *testing.T, threshold int) (string, []Chunk) {
	t.Helper()
	output := t.TempDir()
	page := &content.Page{Title: "Sections", URLPath: "guide", PlainText: "first second", HTML: `<h2>First</h2><p>first text</p><h2>Second</h2><p>second text</p>`}
	if err := Generate(output, config.Default(), map[string][]*content.Page{"en": {page}}); err != nil {
		t.Fatal(err)
	}
	site, err := Load(output)
	if err != nil {
		t.Fatal(err)
	}
	expected := append([]Chunk(nil), site.Chunks...)
	// Recreate the old generator's exact ambiguous IDs and corresponding index.
	for i := range site.Chunks {
		chunk := &site.Chunks[i]
		chunk.ID = stableID("chunk", chunk.PageID, chunk.HeadingID, fmt.Sprint(chunk.Part))
		chunk.ResourceURI = "mpress://knowledge/section/" + chunk.ID
	}
	pages, err := marshalArtifact(site.Pages)
	if err != nil {
		t.Fatal(err)
	}
	chunks, err := marshalArtifact(site.Chunks)
	if err != nil {
		t.Fatal(err)
	}
	index, err := marshalCompactArtifact(makeIndex(site.Chunks))
	if err != nil {
		t.Fatal(err)
	}
	files, artifacts, schema, err := encodeArtifacts(pages, chunks, index, threshold)
	if err != nil {
		t.Fatal(err)
	}
	manifest := site.Manifest
	manifest.Artifacts = artifacts
	manifest.Schema = schema
	sum := sha256.New()
	for _, data := range [][]byte{pages, chunks, index} {
		_, _ = sum.Write(data)
	}
	manifest.Digest = hex.EncodeToString(sum.Sum(nil))
	files[ManifestFile], err = marshalArtifact(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(output, Directory, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return output, expected
}
func TestGeneratedRepeatedAndUnnamedSectionsHaveDistinctChunkIDs(t *testing.T) {
	cfg := config.Default()
	output := t.TempDir()
	page := &content.Page{Title: "Sections", URLPath: "guide", PlainText: "first second third", HTML: `<h2>First</h2><p>first text</p><h2>Second</h2><p>second text</p><h2 id="repeat">Third</h2><p>third text</p><h2 id="repeat">Fourth</h2><p>fourth text</p>`}
	if err := Generate(output, cfg, map[string][]*content.Page{"en": {page}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(output, Directory, ChunksFile))
	if err != nil {
		t.Fatal(err)
	}
	var chunks []Chunk
	if err := json.Unmarshal(data, &chunks); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, chunk := range chunks {
		if seen[chunk.ID] {
			t.Fatalf("generated duplicate section ID %s", chunk.ID)
		}
		seen[chunk.ID] = true
	}
	if len(chunks) != 4 {
		t.Fatalf("generated %d chunks, want four", len(chunks))
	}
	store, err := Load(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Chunks) != 4 {
		t.Fatal("loader lost a generated section")
	}
}

func readIdentityFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
