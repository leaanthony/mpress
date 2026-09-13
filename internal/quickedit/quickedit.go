package quickedit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/translate"
)

const (
	Version      = 1
	maxDraftSize = 256 << 10
)

// Segment is a source-aware piece of visible Markdown text. The shared type
// keeps parse-cache and browser-draft data aligned with translation ranges.
type Segment = translate.EditableSegment

type Document struct {
	Version  int       `json:"version" glint:"version"`
	Revision string    `json:"revision" glint:"revision"`
	Segments []Segment `json:"segments" glint:"segments"`
}

type Change struct {
	ID       string `json:"id"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Original string `json:"original"`
	Markdown string `json:"markdown"`
}

type Draft struct {
	Version  int      `json:"version"`
	Source   string   `json:"source"`
	Route    string   `json:"route,omitempty"`
	Revision string   `json:"revision"`
	Changes  []Change `json:"changes"`
}

// Extract reuses the translation parser's visible source ranges and segment
// identities. It omits frontmatter because quick editing is intentionally
// limited to visible page content.
func Extract(source []byte) (Document, error) {
	segments, err := translate.ExtractEditable(source)
	if err != nil {
		return Document{}, err
	}
	return Document{Version: Version, Revision: Revision(source), Segments: segments}, nil
}

func Revision(source []byte) string {
	sum := sha256.Sum256(source)
	return hex.EncodeToString(sum[:])
}

func Marshal(draft Draft) ([]byte, error) {
	if draft.Version == 0 {
		draft.Version = Version
	}
	data, err := json.Marshal(draft)
	if err != nil {
		return nil, err
	}
	if len(data) > maxDraftSize {
		return nil, errors.New("quick-edit draft is too large")
	}
	return data, nil
}

func Unmarshal(data []byte) (Draft, error) {
	if len(data) > maxDraftSize {
		return Draft{}, errors.New("quick-edit draft is too large")
	}
	var draft Draft
	if err := json.Unmarshal(data, &draft); err != nil {
		return Draft{}, errors.New("quick-edit draft is not valid")
	}
	if draft.Version != Version {
		return Draft{}, fmt.Errorf("quick-edit draft version %d is not supported", draft.Version)
	}
	return draft, nil
}

// ResolveDraftFile finds a downloaded draft by explicit path, current working
// directory, or the user's configured Downloads directory.
func ResolveDraftFile(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("quick-edit draft file is required")
	}
	candidates := []string{name}
	if !filepath.IsAbs(name) && filepath.Base(name) == name {
		if home, err := os.UserHomeDir(); err == nil {
			for _, directory := range downloadDirectories(home) {
				candidates = append(candidates, filepath.Join(directory, name))
			}
		}
	}
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		absolute, err := filepath.Abs(candidate)
		if err != nil || seen[absolute] {
			continue
		}
		seen[absolute] = true
		if info, statErr := os.Stat(absolute); statErr == nil && info.Mode().IsRegular() {
			return absolute, nil
		}
	}
	return "", fmt.Errorf("could not find downloaded draft %q; keep it in Downloads or pass its complete path", name)
}

func downloadDirectories(home string) []string {
	directories := make([]string, 0, 3)
	appendDirectory := func(value string) {
		value = strings.TrimSpace(strings.Trim(value, `"`))
		value = strings.ReplaceAll(value, "${HOME}", home)
		value = strings.ReplaceAll(value, "$HOME", home)
		if value == "" || value == home {
			return
		}
		for _, existing := range directories {
			if existing == value {
				return
			}
		}
		directories = append(directories, value)
	}
	appendDirectory(os.Getenv("XDG_DOWNLOAD_DIR"))
	if configDir, err := os.UserConfigDir(); err == nil {
		if data, readErr := os.ReadFile(filepath.Join(configDir, "user-dirs.dirs")); readErr == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if value, ok := strings.CutPrefix(strings.TrimSpace(line), "XDG_DOWNLOAD_DIR="); ok {
					appendDirectory(value)
					break
				}
			}
		}
	}
	appendDirectory(filepath.Join(home, "Downloads"))
	return directories
}

// ApplyFile validates a downloaded browser draft against the checked-out
// source revision and writes exact Markdown ranges from the end to the start.
func ApplyFile(project, expectedSource, draftPath string) (int, error) {
	data, err := os.ReadFile(draftPath)
	if err != nil {
		return 0, err
	}
	draft, err := Unmarshal(data)
	if err != nil {
		return 0, err
	}
	draft.Source = filepath.ToSlash(filepath.Clean(strings.TrimSpace(draft.Source)))
	expectedSource = filepath.ToSlash(filepath.Clean(strings.TrimSpace(expectedSource)))
	if draft.Source == "." || draft.Source == "" || draft.Source != expectedSource {
		return 0, errors.New("quick-edit draft does not belong to this page")
	}
	ext := strings.ToLower(filepath.Ext(draft.Source))
	if ext != ".md" && ext != ".markdown" {
		return 0, errors.New("quick editing is available only for Markdown files")
	}
	root, err := filepath.Abs(project)
	if err != nil {
		return 0, err
	}
	path := filepath.Join(root, filepath.FromSlash(draft.Source))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return 0, errors.New("quick-edit source must stay inside the project")
	}
	source, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if draft.Revision == "" || draft.Revision != Revision(source) {
		return 0, errors.New("the page changed after this browser draft was created; review it before applying")
	}
	changes := append([]Change(nil), draft.Changes...)
	if len(changes) == 0 {
		return 0, errors.New("quick-edit draft contains no changes")
	}
	for _, change := range changes {
		if change.Start < 0 || change.End < change.Start || change.End > len(source) {
			return 0, errors.New("quick-edit draft contains an invalid source range")
		}
		if string(source[change.Start:change.End]) != change.Original {
			return 0, errors.New("quick-edit draft no longer matches the Markdown source")
		}
	}
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Start == changes[j].Start {
			return changes[i].End > changes[j].End
		}
		return changes[i].Start > changes[j].Start
	})
	previousStart := len(source) + 1
	updated := append([]byte(nil), source...)
	for _, change := range changes {
		if change.End > previousStart {
			return 0, errors.New("quick-edit draft contains overlapping changes")
		}
		previousStart = change.Start
		next := make([]byte, 0, len(updated)-(change.End-change.Start)+len(change.Markdown))
		next = append(next, updated[:change.Start]...)
		next = append(next, change.Markdown...)
		next = append(next, updated[change.End:]...)
		updated = next
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".mpress-quick-edit-*")
	if err != nil {
		return 0, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = temporary.Write(updated); err == nil {
		err = temporary.Chmod(0o644)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return 0, err
	}
	return len(changes), nil
}
