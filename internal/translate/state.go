package translate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stateSchemaVersion = 1

type SegmentState struct {
	SourceHash    string `json:"sourceHash"`
	TargetHash    string `json:"targetHash"`
	MachineHash   string `json:"machineHash,omitempty"`
	MachineText   string `json:"machineText,omitempty"`
	Status        string `json:"status"`
	SourcePreview string `json:"sourcePreview,omitempty"`
	Provider      string `json:"provider,omitempty"`
	Model         string `json:"model,omitempty"`
	PromptVersion string `json:"promptVersion,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

type FileState struct {
	SchemaVersion  int                     `json:"schemaVersion"`
	PageKey        string                  `json:"pageKey"`
	SourceFile     string                  `json:"sourceFile"`
	SourceLanguage string                  `json:"sourceLanguage"`
	TargetLanguage string                  `json:"targetLanguage"`
	Segments       map[string]SegmentState `json:"segments"`
}

func newFileState(sourceFile, sourceLanguage, targetLanguage, pageKey string) *FileState {
	if pageKey == "" {
		pageKey = Hash(filepath.ToSlash(sourceFile))[:20]
	}
	return &FileState{
		SchemaVersion: stateSchemaVersion, PageKey: pageKey,
		SourceFile: filepath.ToSlash(sourceFile), SourceLanguage: sourceLanguage,
		TargetLanguage: targetLanguage, Segments: map[string]SegmentState{},
	}
}

func loadState(path, sourceFile, sourceLanguage, targetLanguage, pageKey string) (*FileState, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return newFileState(sourceFile, sourceLanguage, targetLanguage, pageKey), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read translation state: %w", err)
	}
	var state FileState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse translation state %s: %w", path, err)
	}
	if state.SchemaVersion != stateSchemaVersion {
		return nil, fmt.Errorf("translation state %s uses unsupported schema %d", path, state.SchemaVersion)
	}
	if state.Segments == nil {
		state.Segments = map[string]SegmentState{}
	}
	return &state, nil
}

func saveState(path string, state *FileState) error {
	state.SchemaVersion = stateSchemaVersion
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mpress-translation-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func findStateByPageKey(root, pageKey string) (string, error) {
	if pageKey == "" {
		return "", nil
	}
	var match string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".json" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var identity struct {
			PageKey string `json:"pageKey"`
		}
		if json.Unmarshal(data, &identity) == nil && identity.PageKey == pageKey {
			if match != "" {
				return fmt.Errorf("translationKey %q is used by more than one translation state file", pageKey)
			}
			match = path
		}
		return nil
	})
	return match, err
}

func statePath(root, stateDir, language, sourceFile string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(sourceFile))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("translation source path must stay inside the project")
	}
	stateFile := strings.TrimSuffix(clean, filepath.Ext(clean)) + ".json"
	return filepath.Join(root, filepath.FromSlash(stateDir), language, stateFile), nil
}

func preview(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len([]rune(value)) <= 100 {
		return value
	}
	runes := []rune(value)
	return string(runes[:100]) + "…"
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339) }
