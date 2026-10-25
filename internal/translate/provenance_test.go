package translate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestIncrementalUpdateSynchronisesProtectedCodeWithoutProvider(t *testing.T) {
	root, cfg := translationProject(t)
	sourcePath := filepath.Join(root, "content/index.md")
	source := "# Welcome\n\nUse this example.\n\n```go\noldExample()\n```\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(root, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(root, "content/fr/index.md")
	target, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	target = []byte(strings.Replace(string(target), "FR Use this example.", "Exemple relu manuellement.", 1))
	if err := os.WriteFile(targetPath, target, 0644); err != nil {
		t.Fatal(err)
	}
	source = strings.Replace(source, "oldExample()", "safeExample()", 1)
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	engine.Provider = nil
	report, err := engine.Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pending != 0 || report.Written != 1 {
		t.Fatalf("protected update was skipped: %#v", report)
	}
	updated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "safeExample()") || strings.Contains(string(updated), "oldExample()") || !strings.Contains(string(updated), "Exemple relu manuellement.") {
		t.Fatalf("lost protected update or manual prose:\n%s", updated)
	}
	report, err = engine.Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pending != 0 || report.Written != 0 {
		t.Fatalf("unchanged translation was rewritten: %#v", report)
	}
}

func TestIncrementalUpdateCreatesProtectedOnlyTarget(t *testing.T) {
	root, cfg := translationProject(t)
	sourcePath := filepath.Join(root, "content/index.md")
	source := []byte("```go\nsafeExample()\n```\n")
	if err := os.WriteFile(sourcePath, source, 0644); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(root, cfg, nil)
	report, runErr := engine.Run(context.Background(), Options{Language: "fr"})
	target, readErr := os.ReadFile(filepath.Join(root, "content/fr/index.md"))
	if runErr != nil || readErr != nil || report.Pending != 0 || report.Written != 1 || string(target) != string(source) {
		t.Fatalf("protected-only target was not created: run=%v read=%v report=%#v target=%q", runErr, readErr, report, target)
	}
}
