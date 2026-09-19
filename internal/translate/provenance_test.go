package translate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestIncrementalUpdatePreservesProvenanceAndStaleBaseline(t *testing.T) {
	root, cfg := translationProject(t)
	engine := NewEngine(root, cfg, &fakeProvider{})
	source := filepath.Join(root, "content/index.md")
	if err := os.WriteFile(source, []byte("First sentence.\n\nSecond sentence.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	path, _ := statePath(root, cfg.Translation.StateDir, "fr", "index.md")
	original, err := loadState(path, "index.md", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	for id, entry := range original.Segments {
		entry.Provider = "original-provider"
		entry.Model = "original-model"
		entry.PromptVersion = "original-prompt"
		entry.UpdatedAt = "2025-01-01T00:00:00Z"
		original.Segments[id] = entry
	}
	if err = saveState(path, original); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(source, []byte("Changed first sentence.\n\nSecond sentence.\n\nAdded third sentence.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = engine.Run(context.Background(), Options{Language: "fr", Scope: "missing"}); err != nil {
		t.Fatal(err)
	}
	updated, err := loadState(path, "index.md", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	for id, old := range original.Segments {
		got := updated.Segments[id]
		if got.SourceHash != old.SourceHash || got.Provider != old.Provider || got.Model != old.Model || got.PromptVersion != old.PromptVersion || got.UpdatedAt != old.UpdatedAt {
			t.Fatalf("relabelled untouched translation %s: %#v", id, got)
		}
	}
	report, err := engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["stale"] != 1 {
		t.Fatalf("missing-only run erased stale baseline: %#v", report)
	}
}

func TestLanguageModelIsRecordedInState(t *testing.T) {
	root, cfg := translationProject(t)
	cfg.Translation.Model = "default-model"
	cfg.Translation.LanguageModels = map[string]config.TranslationLanguageModel{"fr": {Model: "french-model"}}
	provider := &configuredProvider{cfg: cfg, providers: map[string]Provider{"french-model\x00": &fakeProvider{}}}
	if _, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	path, _ := statePath(root, cfg.Translation.StateDir, "fr", "index.md")
	state, err := loadState(path, "index.md", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range state.Segments {
		if entry.Model != "french-model" {
			t.Fatalf("wrong model: %#v", entry)
		}
	}
}
