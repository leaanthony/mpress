package projectconvert

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/translate"
)

func TestConversionMigratesTranslationTripletsBeforeRemovingSources(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(map[bool]string{false: "verified", true: "unknown extractor"}[broken], func(t *testing.T) {
			root := t.TempDir()
			cfg := config.Default()
			cfg.Site.Languages = []string{"en", "fr"}
			cfg.Translation.SourceLanguage = "en"
			cfg.Build.ContentDir = "content"
			source := []byte("Read the guide.\n")
			target := []byte("Lisez le guide.\n")
			doc, err := translate.ExtractMPD("page.mpd", source)
			if err != nil {
				t.Fatal(err)
			}
			segment := doc.Segments[0]
			state := translate.FileState{SchemaVersion: 1, PageKey: "page", SourceFile: "page.mpd", SourceLanguage: "en", TargetLanguage: "fr", Segments: map[string]translate.SegmentState{segment.ID: {SourceHash: segment.SourceHash, TargetHash: translate.Hash(string(target[:len(target)-1])), Status: "final", Provider: "original", Model: "original-model"}}}
			if broken {
				state.Extractor = "unknown-future-extractor"
			}
			encoded, _ := json.Marshal(state)
			statePath := filepath.Join(cfg.Translation.StateDir, "fr/page.json")
			for name, data := range map[string][]byte{"content/page.mpd": source, "content/fr/page.mpd": target, statePath: encoded} {
				path := filepath.Join(root, name)
				if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(path, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			result, err := Run(root, &cfg, "", "markdown")
			if broken {
				if err == nil {
					t.Fatal("converted incompatible state")
				}
				if _, err = os.Stat(filepath.Join(root, "content/page.mpd")); err != nil {
					t.Fatal("original removed before migration validation")
				}
				if _, err = os.Stat(filepath.Join(root, "content/page.md")); !os.IsNotExist(err) {
					t.Fatal("partial conversion written")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Count != 2 || result.MigratedStates != 1 || result.MigrationReview != 0 {
				t.Fatalf("result: %#v", result)
			}
			saved, err := os.ReadFile(filepath.Join(root, statePath))
			if err != nil {
				t.Fatal(err)
			}
			var migrated translate.FileState
			if err = json.Unmarshal(saved, &migrated); err != nil {
				t.Fatal(err)
			}
			if migrated.SchemaVersion != 2 || migrated.SourceFile != "page.md" || migrated.Extractor == "" {
				t.Fatalf("state: %#v", migrated)
			}
			report, err := translate.NewEngine(root, cfg, nil).Run(context.Background(), translate.Options{Language: "fr", DryRun: true})
			if err != nil {
				t.Fatal(err)
			}
			if report.Pending != 0 || report.Files[0].States["final"] != 1 {
				t.Fatalf("lost approved translation: %#v", report)
			}
		})
	}
}
