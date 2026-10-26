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

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/translate"
)

// stateRollbackFixture converts an MPD triplet to Markdown whose migrated
// translation state is written before a source consumption step fails.
type stateRollbackFixture struct {
	cfg            config.Config
	root           string
	files          *projectfs.FS
	source, target []byte
	originalState  []byte
	generatedState []byte
	statePath      string
	removeErr      error
	failed         bool
}

func setupStateRollbackFixture(t *testing.T) *stateRollbackFixture {
	t.Helper()
	fixture := &stateRollbackFixture{cfg: config.Default(), removeErr: errors.New("source removal failed")}
	fixture.cfg.Site.Languages = []string{"en", "fr"}
	fixture.cfg.Translation.SourceLanguage = "en"
	fixture.root, fixture.files = openConversionProject(t, &fixture.cfg)
	fixture.source, fixture.target, fixture.originalState, fixture.statePath = writeTranslationTriplet(t, fixture.root, &fixture.cfg)
	return fixture
}

// ops fails while consuming the untranslated source, after the translated
// source was moved aside and after the state migration write succeeded.
func (f *stateRollbackFixture) ops() conversionOps {
	ops := defaultConversionOps
	rename := ops.renameLexical
	ops.renameLexical = func(files *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == filepath.Join(f.root, "content", "page.mpd") {
			f.generatedState, _ = files.ReadFile(f.statePath)
			f.failed = true
			return f.removeErr
		}
		return rename(files, old, new)
	}
	return ops
}

func (f *stateRollbackFixture) run(t *testing.T, ops conversionOps) error {
	t.Helper()
	_, err := runConversionWithOps(f.root, &f.cfg, "", "markdown", f.files, f.files, ops)
	return err
}

// injectStateSaveAfterObservation lets the rollback observation read complete
// with its original bytes, then performs a real author save at the state path.
func injectStateSaveAfterObservation(t *testing.T, fixture *stateRollbackFixture, ops *conversionOps, change func(observed []byte) error) *bool {
	t.Helper()
	injected := false
	read := ops.readFile
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		data, err := read(files, name)
		if fixture.failed && !injected && filepath.Clean(name) == fixture.statePath && err == nil {
			injected = true
			if changeErr := change(data); changeErr != nil {
				t.Fatalf("inject state save after observation: %v", changeErr)
			}
		}
		return data, err
	}
	return &injected
}

func authorReviewedState(t *testing.T) []byte {
	t.Helper()
	state := translate.FileState{
		SchemaVersion: 2, Extractor: "mpd", PageKey: "page",
		SourceFile: "page.mpd", SourceLanguage: "en", TargetLanguage: "fr",
		Segments: map[string]translate.SegmentState{
			"segment-1": {SourceHash: "source", TargetHash: "target", Status: "reviewed", Provider: "author", Model: "author-model"},
		},
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func stateBackupGlob(fixture *stateRollbackFixture) string {
	return filepath.Join(fixture.root, "content", recoveryPrefix+"*", filepath.FromSlash(fixture.cfg.Translation.StateDir), "fr", "page.json.recovery")
}

func capturedStateLeaves(t *testing.T, fixture *stateRollbackFixture) []string {
	t.Helper()
	captured, err := filepath.Glob(filepath.Join(fixture.root, "content", recoveryPrefix+"*", currentPrefix+"*", "page.json.recovery"))
	if err != nil {
		t.Fatal(err)
	}
	return captured
}

func requireOriginalStateBackup(t *testing.T, fixture *stateRollbackFixture) string {
	t.Helper()
	copies, err := filepath.Glob(stateBackupGlob(fixture))
	if err != nil || len(copies) != 1 {
		t.Fatalf("original state recovery copy was not retained: %v %v", copies, err)
	}
	data, err := os.ReadFile(copies[0])
	if err != nil || !bytes.Equal(data, fixture.originalState) {
		t.Fatalf("original state recovery copy lost its bytes: %v", err)
	}
	info, err := os.Stat(copies[0])
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("original state recovery copy mode changed: %v", info.Mode().Perm())
	}
	return copies[0]
}

// requireStateRollbackFailure checks that the initial conversion failure is
// reported together with the incomplete rollback, and that no generated
// document target or consumed source was left behind.
func requireStateRollbackFailure(t *testing.T, fixture *stateRollbackFixture, err error, reason string) {
	t.Helper()
	if !errors.Is(err, fixture.removeErr) {
		t.Fatalf("original conversion failure was not preserved: %v", err)
	}
	if !strings.Contains(err.Error(), "rollback incomplete") {
		t.Fatalf("state change during rollback was not reported as incomplete: %v", err)
	}
	if !strings.Contains(err.Error(), fixture.statePath) {
		t.Fatalf("diagnostic does not name the state path: %v", err)
	}
	if reason != "" && !strings.Contains(err.Error(), reason) {
		t.Fatalf("diagnostic does not report %q: %v", reason, err)
	}
	for _, target := range []string{"content/page.md", "content/fr/page.md"} {
		if _, statErr := os.Stat(filepath.Join(fixture.root, target)); !os.IsNotExist(statErr) {
			t.Fatalf("stale generated target %s survived rollback: %v", target, statErr)
		}
	}
	assertConversionSource(t, filepath.Join(fixture.root, "content", "page.mpd"), fixture.source, 0644)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "fr", "page.mpd"), fixture.target, 0644)
}

func TestConversionRollbackKeepsLateTranslationStateSave(t *testing.T) {
	cases := []struct {
		name string
		save func(t *testing.T, path string, data []byte) error
	}{
		{"in place", func(_ *testing.T, path string, data []byte) error {
			return authorWriteFile(path, data, 0644)
		}},
		{"atomic replacement", func(_ *testing.T, path string, data []byte) error {
			replacement := path + ".author"
			if err := authorWriteFile(replacement, data, 0644); err != nil {
				return err
			}
			return os.Rename(replacement, path)
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := setupStateRollbackFixture(t)
			author := authorReviewedState(t)
			ops := fixture.ops()
			injected := injectStateSaveAfterObservation(t, fixture, &ops, func([]byte) error {
				return test.save(t, fixture.statePath, author)
			})

			runErr := fixture.run(t, ops)
			if !*injected {
				t.Fatal("late approval/model save during rollback was not injected")
			}
			requireStateRollbackFailure(t, fixture, runErr, "changed after conversion")
			assertConversionSource(t, fixture.statePath, author, 0644)
			requireOriginalStateBackup(t, fixture)
		})
	}
}

func TestConversionRollbackKeepsTranslationStateModeChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not provide these Unix permission distinctions")
	}
	fixture := setupStateRollbackFixture(t)
	var generated []byte
	ops := fixture.ops()
	injected := injectStateSaveAfterObservation(t, fixture, &ops, func(observed []byte) error {
		generated = append([]byte(nil), observed...)
		return os.Chmod(fixture.statePath, 0640)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("state mode change during rollback was not injected")
	}
	requireStateRollbackFailure(t, fixture, runErr, "permissions changed after conversion")
	assertConversionSource(t, fixture.statePath, generated, 0640)
	requireOriginalStateBackup(t, fixture)
}

func TestConversionRollbackKeepsTranslationStateAlias(t *testing.T) {
	requireSymlinks(t)
	cases := []struct {
		name         string
		alias        string
		materialized bool
	}{
		{"internal", "page-state.json", true},
		{"dangling", "../../author-state.json", false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := setupStateRollbackFixture(t)
			linkText := test.alias
			if test.materialized {
				internal := filepath.Join(filepath.Dir(fixture.statePath), test.alias)
				if err := os.WriteFile(internal, fixture.originalState, 0600); err != nil {
					t.Fatal(err)
				}
			} else if _, err := os.Stat(filepath.Join(filepath.Dir(fixture.statePath), linkText)); !os.IsNotExist(err) {
				t.Fatalf("dangling alias target unexpectedly exists: %v", err)
			}
			ops := fixture.ops()
			injected := injectStateSaveAfterObservation(t, fixture, &ops, func([]byte) error {
				if err := os.Remove(fixture.statePath); err != nil {
					return err
				}
				return os.Symlink(linkText, fixture.statePath)
			})

			runErr := fixture.run(t, ops)
			if !*injected {
				t.Fatal("state alias change during rollback was not injected")
			}
			requireStateRollbackFailure(t, fixture, runErr, "was replaced by a symbolic link")
			info, err := os.Lstat(fixture.statePath)
			if err != nil || info.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("author alias at the state path was not preserved: %v %v", info, err)
			}
			if target, err := os.Readlink(fixture.statePath); err != nil || target != linkText {
				t.Fatalf("author alias text changed: %q %v", target, err)
			}
			if test.materialized {
				assertConversionSource(t, filepath.Join(filepath.Dir(fixture.statePath), test.alias), fixture.originalState, 0600)
			} else if _, err := os.Stat(fixture.statePath); !os.IsNotExist(err) {
				t.Fatalf("dangling author alias was materialised: %v", err)
			}
			requireOriginalStateBackup(t, fixture)
		})
	}
}

func TestConversionRollbackRestoresMissingTranslationState(t *testing.T) {
	fixture := setupStateRollbackFixture(t)
	ops := fixture.ops()
	injected := injectStateSaveAfterObservation(t, fixture, &ops, func([]byte) error {
		return os.Remove(fixture.statePath)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("removal of the state destination was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) {
		t.Fatalf("original conversion failure was not preserved: %v", runErr)
	}
	if strings.Contains(runErr.Error(), "rollback incomplete") {
		t.Fatalf("missing state destination prevented a clean rollback: %v", runErr)
	}
	assertConversionSource(t, fixture.statePath, fixture.originalState, 0600)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "page.mpd"), fixture.source, 0644)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "fr", "page.mpd"), fixture.target, 0644)
	for _, target := range []string{"content/page.md", "content/fr/page.md"} {
		if _, statErr := os.Stat(filepath.Join(fixture.root, target)); !os.IsNotExist(statErr) {
			t.Fatalf("stale generated target %s survived rollback: %v", target, statErr)
		}
	}
	if copies := recoveryCopies(t, fixture.root); len(copies) != 0 {
		t.Fatalf("clean rollback retained recovery copies: %v", copies)
	}
}

func TestConversionRestoresTranslationStateAliasAfterOrdinaryFailure(t *testing.T) {
	requireSymlinks(t)
	fixture := setupStateRollbackFixture(t)
	aliasTarget := filepath.Join(filepath.Dir(fixture.statePath), "page-state.json")
	if err := os.Rename(fixture.statePath, aliasTarget); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("page-state.json", fixture.statePath); err != nil {
		t.Fatal(err)
	}

	runErr := fixture.run(t, fixture.ops())
	if !errors.Is(runErr, fixture.removeErr) {
		t.Fatalf("original conversion failure was not preserved: %v", runErr)
	}
	if strings.Contains(runErr.Error(), "rollback incomplete") {
		t.Fatalf("unchanged rollback did not complete: %v", runErr)
	}
	info, err := os.Lstat(fixture.statePath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("original state alias was not restored: %v %v", info, err)
	}
	if target, err := os.Readlink(fixture.statePath); err != nil || target != "page-state.json" {
		t.Fatalf("original state alias text changed: %q %v", target, err)
	}
	assertConversionSource(t, aliasTarget, fixture.originalState, 0600)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "page.mpd"), fixture.source, 0644)
	assertConversionSource(t, filepath.Join(fixture.root, "content", "fr", "page.mpd"), fixture.target, 0644)
	for _, target := range []string{"content/page.md", "content/fr/page.md"} {
		if _, statErr := os.Stat(filepath.Join(fixture.root, target)); !os.IsNotExist(statErr) {
			t.Fatalf("stale generated target %s survived rollback: %v", target, statErr)
		}
	}
	if copies := recoveryCopies(t, fixture.root); len(copies) != 0 {
		t.Fatalf("clean rollback retained recovery copies: %v", copies)
	}
}

func TestConversionRollbackNeverReplacesTranslationStateRecreatedAfterCapture(t *testing.T) {
	fixture := setupStateRollbackFixture(t)
	newer := []byte("{\"schemaVersion\": 2, \"segments\": {\"again\": {\"status\": \"reviewed\"}}}\n")
	ops := fixture.ops()
	link := ops.linkLexical
	ops.linkLexical = func(files *projectfs.FS, old, new string) error {
		if filepath.Base(old) == "page.json.recovery" && !strings.Contains(old, currentPrefix) {
			if err := authorWriteFile(new, newer, 0600); err != nil {
				return err
			}
		}
		return link(files, old, new)
	}

	runErr := fixture.run(t, ops)
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, os.ErrExist) {
		t.Fatalf("recreated state destination collision was not reported: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "rollback incomplete") {
		t.Fatalf("recreated state destination was not diagnosed as incomplete rollback: %v", runErr)
	}
	assertConversionSource(t, fixture.statePath, newer, 0600)
	captured := capturedStateLeaves(t, fixture)
	if len(captured) != 1 {
		t.Fatalf("captured state leaf was not retained beside the newer save: %v", captured)
	}
	assertConversionSource(t, captured[0], fixture.generatedState, 0644)
	requireOriginalStateBackup(t, fixture)
}

func TestConversionRollbackReportsTranslationStateCaptureReadFailure(t *testing.T) {
	fixture := setupStateRollbackFixture(t)
	author := authorReviewedState(t)
	captureReadFailure := errors.New("captured state read failed")
	ops := fixture.ops()
	read := ops.readFile
	ops.readFile = func(files *projectfs.FS, name string) ([]byte, error) {
		if strings.Contains(name, currentPrefix) && filepath.Base(name) == "page.json.recovery" {
			return nil, captureReadFailure
		}
		return read(files, name)
	}
	injected := injectStateSaveAfterObservation(t, fixture, &ops, func([]byte) error {
		return os.WriteFile(fixture.statePath, author, 0644)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late approval/model save during rollback was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, captureReadFailure) {
		t.Fatalf("capture inspection failure was not reported with its cause: %v", runErr)
	}
	captured := capturedStateLeaves(t, fixture)
	if len(captured) != 1 {
		t.Fatalf("captured author state was not retained: %v", captured)
	}
	if !strings.Contains(runErr.Error(), captured[0]) {
		t.Fatalf("retained capture path was not diagnosed: %v", runErr)
	}
	assertConversionSource(t, captured[0], author, 0644)
	requireOriginalStateBackup(t, fixture)
}

func TestConversionRollbackLeavesTranslationStateWhenCaptureRenameFails(t *testing.T) {
	fixture := setupStateRollbackFixture(t)
	author := authorReviewedState(t)
	captureFailure := errors.New("state capture rename failed")
	ops := fixture.ops()
	rename := ops.renameLexical
	ops.renameLexical = func(files *projectfs.FS, old, new string) error {
		if filepath.Clean(old) == fixture.statePath && strings.Contains(new, currentPrefix) {
			return captureFailure
		}
		return rename(files, old, new)
	}
	injected := injectStateSaveAfterObservation(t, fixture, &ops, func([]byte) error {
		return authorWriteFile(fixture.statePath, author, 0644)
	})

	runErr := fixture.run(t, ops)
	if !*injected {
		t.Fatal("late approval/model save during rollback was not injected")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, captureFailure) {
		t.Fatalf("capture failure was not reported with its cause: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "rollback incomplete") {
		t.Fatalf("capture failure was not diagnosed as incomplete rollback: %v", runErr)
	}
	assertConversionSource(t, fixture.statePath, author, 0644)
	requireOriginalStateBackup(t, fixture)
}

func TestConversionRollbackRetainsTranslationStateWhenRestoreFails(t *testing.T) {
	fixture := setupStateRollbackFixture(t)
	restoreFailure := errors.New("state no-replace restore failed")
	ops := fixture.ops()
	link := ops.linkLexical
	ops.linkLexical = func(files *projectfs.FS, old, new string) error {
		if filepath.Base(old) == "page.json.recovery" && !strings.Contains(old, currentPrefix) {
			return restoreFailure
		}
		return link(files, old, new)
	}

	runErr := fixture.run(t, ops)
	if len(fixture.generatedState) == 0 {
		t.Fatal("migrated state was not observed before the failure")
	}
	if !errors.Is(runErr, fixture.removeErr) || !errors.Is(runErr, restoreFailure) {
		t.Fatalf("restore failure was not reported with its cause: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "rollback incomplete") {
		t.Fatalf("restore failure was not diagnosed as incomplete rollback: %v", runErr)
	}
	if _, err := os.Stat(fixture.statePath); !os.IsNotExist(err) {
		t.Fatalf("failed restore unexpectedly materialised the state path: %v", err)
	}
	captured := capturedStateLeaves(t, fixture)
	if len(captured) != 1 {
		t.Fatalf("verified generated state copy was not retained: %v", captured)
	}
	assertConversionSource(t, captured[0], fixture.generatedState, 0644)
	requireOriginalStateBackup(t, fixture)
}

func TestWrittenStatePermissionsMatchPlatformContract(t *testing.T) {
	for _, test := range []struct {
		name            string
		actual, planned os.FileMode
		goos            string
		want            bool
	}{
		{"Windows writable state", 0o666, 0o644, "windows", true},
		{"Windows read-only author change", 0o444, 0o644, "windows", false},
		{"Windows planned read-only state", 0o444, 0o400, "windows", true},
		{"Unix unchanged state", 0o644, 0o644, "linux", true},
		{"Unix changed group permissions", 0o640, 0o644, "linux", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := sameWrittenPermissions(test.actual, test.planned, test.goos); got != test.want {
				t.Fatalf("permissions match = %v, want %v", got, test.want)
			}
		})
	}
}
