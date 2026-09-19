package translate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestTranslationStateMigration(t *testing.T) {
	for _, format := range []string{"legacy-markdown", "mpd", "mpd-code-punctuation"} {
		t.Run(format, func(t *testing.T) {
			original, cfg := translationProject(t)
			current := t.TempDir()
			if err := config.Save(original, cfg); err != nil {
				t.Fatal(err)
			}
			if err := config.Save(current, cfg); err != nil {
				t.Fatal(err)
			}
			file := "index.md"
			source := []byte("Read the [guide](/guide/) and run `tool`.\n")
			target := []byte("Lisez le [guide](/guide/) et lancez `tool`.\n")
			extractor := "markdown-text-v1"
			if strings.HasPrefix(format, "mpd") {
				file = "index.mpd"
				extractor = "mpd-v1"
				if format == "mpd-code-punctuation" {
					source = []byte("`tool`:\n")
					target = []byte("`tool` :\n")
				}
			}
			sourceDoc, err := extractHistorical(file, cfg.Build.NavFile, source, extractor)
			if err != nil {
				t.Fatal(err)
			}
			targetDoc, err := extractHistorical(file, cfg.Build.NavFile, target, extractor)
			if err != nil {
				t.Fatal(err)
			}
			state := newFileState(file, "en", "fr", "")
			state.Extractor = ""
			state.SchemaVersion = 1
			targets := migrationSegments(targetDoc)
			for _, s := range sourceDoc.Segments {
				text := targets[s.ID].Original
				state.Segments[s.ID] = SegmentState{SourceHash: s.SourceHash, TargetHash: Hash(text), MachineHash: Hash(text), MachineText: text, Status: "final", Provider: "old-provider", Model: "old-model", PromptVersion: "old-prompt", UpdatedAt: "2025-01-01T00:00:00Z"}
			}
			data, _ := json.Marshal(state)
			baselineSource, err := convertMigrationDocument(file, "index.md", source, nil)
			if err != nil {
				t.Fatal(err)
			}
			baselineTarget, err := convertMigrationDocument(file, "index.md", target, nil)
			if err != nil {
				t.Fatal(err)
			}
			write := func(root, path string, data []byte) {
				t.Helper()
				name := filepath.Join(root, path)
				if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			write(original, "content/"+file, source)
			write(original, "content/fr/"+file, target)
			write(current, "content/index.md", baselineSource)
			write(current, "content/fr/index.md", baselineTarget)
			oldPath, _ := statePath(original, cfg.Translation.StateDir, "fr", file)
			write(original, strings.TrimPrefix(oldPath, original+"/"), data)
			path, _ := statePath(current, cfg.Translation.StateDir, "fr", "index.md")
			write(current, strings.TrimPrefix(path, current+"/"), data)
			engine := NewEngine(current, cfg, nil)
			report, err := engine.MigrateState(original, "fr", "index.md", false)
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Files) != 1 || report.Files[0].Conflicts != 0 || report.Written != 0 {
				t.Fatalf("plan: %s", report.JSON())
			}
			unchanged, _ := os.ReadFile(path)
			if string(unchanged) != string(data) {
				t.Fatal("dry run wrote state")
			}
			report, err = engine.MigrateState(original, "fr", "index.md", true)
			if err != nil {
				t.Fatal(err)
			}
			if report.Written != 1 {
				t.Fatalf("write: %s", report.JSON())
			}
			migrated, err := loadState(path, "index.md", "en", "fr", "")
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range migrated.Segments {
				if s.Status != "final" || s.Provider != "old-provider" || s.Model != "old-model" || s.PromptVersion != "old-prompt" {
					t.Fatalf("lost evidence: %#v", s)
				}
			}
			saved, _ := os.ReadFile(path)
			report, err = engine.MigrateState(original, "fr", "index.md", true)
			if err != nil {
				t.Fatal(err)
			}
			again, _ := os.ReadFile(path)
			if report.Written != 0 || string(saved) != string(again) {
				t.Fatal("migration was not idempotent")
			}
			status, err := engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
			if err != nil {
				t.Fatal(err)
			}
			if status.Pending != 0 || status.Files[0].States["final"] == 0 && status.Files[0].States["protected"] == 0 {
				t.Fatalf("migrated status: %#v", status)
			}
			actual, _ := os.ReadFile(filepath.Join(current, "content/fr/index.md"))
			if string(actual) != string(baselineTarget) {
				t.Fatal("migration rewrote target")
			}
		})
	}
}

func TestMigrateSegmentRefusesStaleAndDowngradesChangedApproval(t *testing.T) {
	source := Segment{ID: "one", Original: "Source", SourceHash: Hash("Source")}
	target := Segment{ID: "one", Original: "Traduction"}
	states := map[string]SegmentState{"one": {SourceHash: source.SourceHash, TargetHash: Hash("previous translation"), Status: "final"}}
	sources, targets := map[string]Segment{"one": source}, map[string]Segment{"one": target}
	entry, ok := migrateSegment([]string{"one"}, states, sources, targets)
	if !ok || entry.Status != "manual" {
		t.Fatalf("changed approved target: %#v, %v", entry, ok)
	}
	stale := states["one"]
	stale.SourceHash = Hash("old source")
	states["one"] = stale
	if _, ok = migrateSegment([]string{"one"}, states, sources, targets); ok {
		t.Fatal("stale source acquired current approval")
	}
}

func TestIncompatibleSidecarBlocksProviderAndReportsMigration(t *testing.T) {
	root, cfg := translationProject(t)
	provider := &fakeProvider{}
	engine := NewEngine(root, cfg, provider)
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	path, _ := statePath(root, cfg.Translation.StateDir, "fr", "index.md")
	state, err := loadState(path, "index.md", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	state.Extractor = "markdown-text-v1"
	if err = saveState(path, state); err != nil {
		t.Fatal(err)
	}
	before := provider.calls
	report, err := engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["migration-required"] == 0 {
		t.Fatalf("status: %#v", report)
	}
	if _, err = engine.Run(context.Background(), Options{Language: "fr"}); err == nil {
		t.Fatal("accepted incompatible state")
	}
	if provider.calls != before {
		t.Fatal("called provider before migration")
	}
	if _, err = engine.Mark("fr", "index.md", "final"); err == nil {
		t.Fatal("approved incompatible state")
	}
	state.Extractor = currentExtractor("index.md")
	for id, entry := range state.Segments {
		entry.Status = "migration-review"
		state.Segments[id] = entry
	}
	if err = saveState(path, state); err != nil {
		t.Fatal(err)
	}
	report, err = engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["migration-review"] == 0 {
		t.Fatalf("review status: %#v", report)
	}
	if _, err = engine.Run(context.Background(), Options{Language: "fr"}); err == nil {
		t.Fatal("updated unreviewed migration")
	}
	if provider.calls != before {
		t.Fatal("called provider before reviewing migration")
	}
	if _, err = engine.Mark("fr", "index.md", "reviewed"); err != nil {
		t.Fatal(err)
	}
	report, err = engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["migration-review"] != 0 || report.Files[0].States["reviewed"] == 0 {
		t.Fatalf("review did not clear migration requirement: %#v", report)
	}
}
