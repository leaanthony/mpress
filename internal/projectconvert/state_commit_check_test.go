package projectconvert

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/translate"
)

// reviewedState adds a completed final-review save to the archived original
// translation state without changing its identity or schema.
func reviewedState(t *testing.T, original []byte) []byte {
	t.Helper()
	var state translate.FileState
	if err := json.Unmarshal(original, &state); err != nil {
		t.Fatal(err)
	}
	for id, segment := range state.Segments {
		segment.Status = "reviewed"
		segment.Provider = "author"
		segment.Model = "author-model"
		segment.UpdatedAt = "2026-10-27T00:00:00Z"
		state.Segments[id] = segment
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func TestConversionCommitKeepsTranslationStateSavedThroughOpenDescriptor(t *testing.T) {
	requireOpenFileRename(t)
	fixture := setupStateRollbackFixture(t)
	author := reviewedState(t, fixture.originalState)
	file, err := os.OpenFile(fixture.statePath, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })

	var recoveryPath string
	injected := false
	ops := defaultConversionOps
	rename := ops.renameLexical
	// The first original document rename runs after the state was moved into
	// recovery and its migration was published.
	ops.renameLexical = func(files *projectfs.FS, old, new string) error {
		if !injected && filepath.Clean(old) == filepath.Join(fixture.root, "content", "page.mpd") {
			injected = true
			recoveryPath = requireRecoveryCopy(t, fixture.root, filepath.Join(fixture.cfg.Translation.StateDir, "fr", "page.json"))
			if n, err := file.WriteAt(author, 0); err != nil || n != len(author) {
				t.Fatalf("state save incomplete: %d bytes, %v", n, err)
			}
			if err := file.Truncate(int64(len(author))); err != nil {
				return err
			}
			if err := file.Sync(); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
			inRecovery, err := os.ReadFile(recoveryPath)
			if err != nil {
				return err
			}
			if !bytes.Equal(inRecovery, author) {
				t.Fatalf("completed state save was not in recovery before cleanup: %q", inRecovery)
			}
		}
		return rename(files, old, new)
	}

	result, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "markdown", fixture.files, fixture.files, ops)
	if !injected {
		t.Fatal("completed translation state save was not injected")
	}
	if err == nil {
		t.Fatal("completed translation state save was discarded silently")
	}
	for _, want := range []string{"refusing to discard recovery copies", "content changed", fixture.statePath, recoveryPath} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("diagnostic missing %q: %v", want, err)
		}
	}
	if result.MigratedStates != 1 {
		t.Fatalf("conversion result changed: %+v", result)
	}

	// The completed save is retained verbatim, with its original 0600 mode,
	// beside the generated 0644 migration.
	got, readErr := os.ReadFile(recoveryPath)
	if readErr != nil || !bytes.Equal(got, author) {
		t.Fatalf("author state save did not survive cleanup: %v %q", readErr, got)
	}
	info, statErr := os.Lstat(recoveryPath)
	if statErr != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("author state save mode was not retained: %v %v", info, statErr)
	}
	generated, readErr := os.ReadFile(fixture.statePath)
	if readErr != nil || bytes.Equal(generated, author) {
		t.Fatalf("generated migration was not kept at the state path: %v %q", readErr, generated)
	}
	if stateInfo, statErr := os.Stat(fixture.statePath); statErr != nil || runtime.GOOS != "windows" && stateInfo.Mode().Perm() != 0644 {
		t.Fatalf("generated state mode changed: %v %v", stateInfo, statErr)
	}
	var restored translate.FileState
	if err := json.Unmarshal(got, &restored); err != nil {
		t.Fatalf("retained state save is not valid: %v", err)
	}
	if !hasReviewedProvenance(restored.Segments) {
		t.Fatalf("retained state save lost its provenance: %+v", restored.Segments)
	}

	// The retained copies are usable author work: restore the completed save
	// and the consumed documents, then retry the conversion.
	if err := os.Remove(fixture.statePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(recoveryPath, fixture.statePath); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"page.mpd", filepath.Join("fr", "page.mpd")} {
		copyPath := requireRecoveryCopy(t, fixture.root, filepath.Join("content", name))
		if err := os.Rename(copyPath, filepath.Join(fixture.root, "content", name)); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(fixture.root, "content", strings.TrimSuffix(name, ".mpd")+".md")); err != nil {
			t.Fatal(err)
		}
	}
	retry, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "markdown", fixture.files, fixture.files, defaultConversionOps)
	if err != nil {
		t.Fatalf("retry from the retained state save failed: %v", err)
	}
	if retry.MigratedStates != 1 {
		t.Fatalf("retry did not migrate the restored state: %+v", retry)
	}
	migratedData, readErr := os.ReadFile(fixture.statePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	var migrated translate.FileState
	if err := json.Unmarshal(migratedData, &migrated); err != nil {
		t.Fatalf("retry produced an unreadable state: %v", err)
	}
	if migrated.SchemaVersion != 2 || migrated.SourceFile != "page.md" {
		t.Fatalf("retry did not migrate the restored state: %+v", migrated)
	}
	if !hasReviewedProvenance(migrated.Segments) {
		t.Fatalf("retry lost the restored review provenance: %+v", migrated.Segments)
	}
}

// hasReviewedProvenance reports whether the saved author review survived.
func hasReviewedProvenance(segments map[string]translate.SegmentState) bool {
	if len(segments) == 0 {
		return false
	}
	for _, segment := range segments {
		if segment.Status != "reviewed" || segment.Provider != "author" || segment.Model != "author-model" || segment.UpdatedAt != "2026-10-27T00:00:00Z" {
			return false
		}
	}
	return true
}

func TestConversionFailsBeforePublishingStateWhenOriginalCannotBeRead(t *testing.T) {
	fixture := setupStateRollbackFixture(t)
	readFailure := errors.New("state checkpoint read failed")
	ops := defaultConversionOps
	read := ops.readFile
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		if filepath.Clean(name) == fixture.statePath {
			return nil, readFailure
		}
		return read(files, name)
	}

	_, err := runConversionWithOps(fixture.root, &fixture.cfg, "", "markdown", fixture.files, fixture.files, ops)
	if !errors.Is(err, readFailure) {
		t.Fatalf("state checkpoint read failure was not reported: %v", err)
	}
	if !strings.Contains(err.Error(), fixture.statePath) {
		t.Fatalf("state checkpoint read failure did not name the source: %v", err)
	}
	assertConversionSource(t, fixture.statePath, fixture.originalState, 0600)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "page.mpd"), fixture.source, 0644)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "fr", "page.mpd"), fixture.target, 0644)
	for _, target := range []string{"content/page.md", "content/fr/page.md"} {
		if _, statErr := os.Stat(filepath.Join(fixture.root, target)); !os.IsNotExist(statErr) {
			t.Fatalf("stale generated target %s survived the failure: %v", target, statErr)
		}
	}
	if copies := recoveryCopies(t, fixture.root); len(copies) != 0 {
		t.Fatalf("rollback retained recovery copies: %v", copies)
	}
}
