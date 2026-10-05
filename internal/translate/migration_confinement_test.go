package translate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

func migrationBoundaryFixture(t *testing.T) (*Engine, string) {
	t.Helper()
	current, previous := t.TempDir(), t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Translation.SourceLanguage = "en"
	source, target := "Read the guide.\n", "Lisez le guide.\n"
	doc, err := ExtractMPD("index.mpd", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	segment := doc.Segments[0]
	state := newFileState("index.mpd", "en", "fr", "")
	state.SchemaVersion, state.Extractor = 1, ""
	state.Segments[segment.ID] = SegmentState{SourceHash: segment.SourceHash, TargetHash: Hash("Lisez le guide."), Status: "final", Provider: "original"}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{current, previous} {
		if err := config.Save(root, cfg); err != nil {
			t.Fatal(err)
		}
		for name, data := range map[string][]byte{"content/index.mpd": []byte(source), "content/fr/index.mpd": []byte(target), filepath.Join(cfg.Translation.StateDir, "fr/index.json"): encoded} {
			path := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return NewEngine(current, cfg, nil), previous
}

func migrationBoundaryPath(engine *Engine, previous, boundary string) string {
	root := engine.Project
	if len(boundary) > 4 && boundary[:4] == "old-" {
		root = previous
		boundary = boundary[4:]
	} else {
		boundary = boundary[len("current-"):]
	}
	switch boundary {
	case "config":
		return filepath.Join(root, config.Filename)
	case "source":
		return filepath.Join(root, "content/index.mpd")
	case "target":
		return filepath.Join(root, "content/fr/index.mpd")
	case "state-file":
		return filepath.Join(root, engine.Config.Translation.StateDir, "fr/index.json")
	default:
		return filepath.Join(root, engine.Config.Translation.StateDir, "fr")
	}
}

func migrationSymlinkProbe(t *testing.T, project string) {
	t.Helper()
	probe := filepath.Join(project, "native-symlink-probe")
	if err := os.Symlink(config.Filename, probe); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationRejectsExternalTripletBoundaries(t *testing.T) {
	for _, boundary := range []string{"old-config", "old-source", "old-target", "old-state-file", "old-state-parent", "current-source", "current-target", "current-state-file", "current-state-parent"} {
		t.Run(boundary, func(t *testing.T) {
			engine, previous := migrationBoundaryFixture(t)
			migrationSymlinkProbe(t, engine.Project)
			path := migrationBoundaryPath(engine, previous, boundary)
			outside := filepath.Join(t.TempDir(), filepath.Base(path))
			if err := os.Rename(path, outside); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, path); err != nil {
				t.Fatal(err)
			}
			sentinel := outside
			if filepath.Base(path) == "fr" {
				sentinel = filepath.Join(outside, "index.json")
			}
			before, err := os.ReadFile(sentinel)
			if err != nil {
				t.Fatal(err)
			}
			statePath := migrationBoundaryPath(engine, previous, "current-state-file")
			stateBefore, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatal(err)
			}
			if report, err := engine.MigrateState(previous, "fr", "index.mpd", true); err == nil {
				t.Errorf("migration accepted external %s: %+v", boundary, report)
			}
			after, err := os.ReadFile(sentinel)
			if err != nil || string(after) != string(before) {
				t.Errorf("external migration sentinel changed: %v", err)
			}
			stateAfter, err := os.ReadFile(statePath)
			if err != nil || string(stateAfter) != string(stateBefore) {
				t.Errorf("rejected migration changed destination state: %v", err)
			}
		})
	}
}

func TestMigrationRetainsInternalAliasesAndSeparateSnapshot(t *testing.T) {
	for _, boundary := range []string{"old-config", "old-source", "old-target", "old-state-file", "old-state-parent", "current-source", "current-target", "current-state-file", "current-state-parent"} {
		t.Run(boundary, func(t *testing.T) {
			engine, previous := migrationBoundaryFixture(t)
			migrationSymlinkProbe(t, engine.Project)
			path := migrationBoundaryPath(engine, previous, boundary)
			saved := path + ".saved"
			if err := os.Rename(path, saved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(saved, path); err != nil {
				t.Fatal(err)
			}
			report, err := engine.MigrateState(previous, "fr", "index.mpd", true)
			if err != nil || report.Written != 1 || len(report.Files) != 1 || report.Files[0].Conflicts != 0 {
				t.Fatalf("safe alias/separate snapshot migration failed: %v %+v", err, report)
			}
			statePath, err := statePath(engine.Project, engine.Config.Translation.StateDir, "fr", "index.mpd")
			if err != nil {
				t.Fatal(err)
			}
			state, err := loadState(statePath, "index.mpd", "en", "fr", "")
			if err != nil || state.SchemaVersion != 2 {
				t.Fatalf("migration state changed: %v %+v", err, state)
			}
			for _, segment := range state.Segments {
				if segment.Status != "final" || segment.Provider != "original" {
					t.Fatalf("migration lost approved evidence: %+v", segment)
				}
			}
			for _, root := range []string{engine.Project, previous} {
				data, err := os.ReadFile(filepath.Join(root, "content/fr/index.mpd"))
				if err != nil || string(data) != "Lisez le guide.\n" {
					t.Fatalf("migration rewrote target: %v %q", err, data)
				}
			}
		})
	}
}

func TestMigrationBorrowedRootRetainsOwnership(t *testing.T) {
	engine, previous := migrationBoundaryFixture(t)
	root, err := projectfs.Open(engine.Project)
	if err != nil {
		t.Fatal(err)
	}
	borrowed, err := engine.BorrowRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	_, migrationErr := borrowed.MigrateState(previous, "fr", "index.mpd", false)
	_, statErr := root.Stat(config.Filename)
	closeErr := root.Close()
	if migrationErr != nil || statErr != nil || closeErr != nil {
		t.Fatalf("migration borrowed ownership changed: migration=%v stat=%v close=%v", migrationErr, statErr, closeErr)
	}
	if _, err := engine.MigrateState(previous, "fr", "index.mpd", true); err != nil {
		t.Fatalf("original engine retained closed root: %v", err)
	}
}
