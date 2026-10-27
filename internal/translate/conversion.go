package translate

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

// ConvertedDocument holds both sides of a planned project format conversion.
// Paths are relative to the configured content directory.
type ConvertedDocument struct {
	SourceFile, TargetFile string
	Source, Target         []byte
}
type ConvertedState struct {
	Path           string
	Data           []byte
	RequiresReview int
	// SourceDigest fingerprints the exact raw state bytes decoded to plan Data.
	// Publishing must refuse a state file whose current bytes differ from it.
	SourceDigest [sha256.Size]byte
}

// PlanConversionStatesRoot borrows the already pinned project state boundary.
func PlanConversionStatesRoot(files *projectfs.FS, root string, cfg config.Config, documents []ConvertedDocument, markdownToMPD func(string) (string, error)) ([]ConvertedState, error) {
	if files == nil {
		return nil, fmt.Errorf("conversion state project root is required")
	}
	byFile := map[string]ConvertedDocument{}
	for _, doc := range documents {
		byFile[doc.SourceFile] = doc
	}
	var result []ConvertedState
	for _, doc := range documents {
		for _, language := range cfg.Site.Languages {
			if language == cfg.Translation.SourceLanguage {
				continue
			}
			target, ok := byFile[filepath.ToSlash(filepath.Join(language, doc.SourceFile))]
			if !ok {
				continue
			}
			path, err := statePath(root, cfg.Translation.StateDir, language, doc.SourceFile)
			if err != nil {
				return nil, err
			}
			if _, err = files.Stat(path); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return nil, err
			}
			raw, readErr := files.ReadFile(path)
			old, err := decodeState(path, raw, readErr, doc.SourceFile, cfg.Translation.SourceLanguage, language, "")
			if err != nil {
				return nil, err
			}
			if filepath.ToSlash(old.SourceFile) != doc.SourceFile || old.TargetLanguage != language || old.SourceLanguage != cfg.Translation.SourceLanguage {
				return nil, fmt.Errorf("translation state %s does not match the conversion triplet; migrate or restore it first", path)
			}
			if strings.TrimPrefix(target.TargetFile, language+"/") != doc.TargetFile {
				return nil, fmt.Errorf("translation conversion paths do not align for %s", doc.SourceFile)
			}
			migrated, review, err := migrateDocuments(migrationPair{doc.SourceFile, cfg.Build.NavFile, doc.Source, target.Source}, migrationPair{doc.TargetFile, cfg.Build.NavFile, doc.Target, target.Target}, old, cfg.Translation.SourceLanguage, markdownToMPD)
			if err != nil {
				return nil, fmt.Errorf("migrate conversion state %s: %w", path, err)
			}
			data, err := json.MarshalIndent(migrated, "", "  ")
			if err != nil {
				return nil, err
			}
			result = append(result, ConvertedState{
				Path:           path,
				Data:           append(data, '\n'),
				RequiresReview: review,
				SourceDigest:   sha256.Sum256(raw),
			})
		}
	}
	return result, nil
}
