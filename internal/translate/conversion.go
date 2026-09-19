package translate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
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
}

// PlanConversionStates validates and serializes sidecars before a project
// converter writes documents or removes the original triplets.
func PlanConversionStates(root string, cfg config.Config, documents []ConvertedDocument, markdownToMPD func(string) (string, error)) ([]ConvertedState, error) {
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
			if _, err = os.Stat(path); os.IsNotExist(err) {
				continue
			} else if err != nil {
				return nil, err
			}
			old, err := loadState(path, doc.SourceFile, cfg.Translation.SourceLanguage, language, "")
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
			result = append(result, ConvertedState{path, append(data, '\n'), review})
		}
	}
	return result, nil
}
