package projectconvert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/importer"
	"github.com/leaanthony/mpress/internal/mpd"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/leaanthony/mpress/internal/translate"
)

// Result describes a successfully validated project content conversion.
type Result struct {
	Count           int    `json:"count"`
	Directory       string `json:"directory"`
	SourceFormat    string `json:"sourceFormat"`
	TargetFormat    string `json:"targetFormat"`
	MigratedStates  int    `json:"migratedStates"`
	MigrationReview int    `json:"migrationReview"`
}

type conversion struct {
	source string
	target string
	mode   os.FileMode
	data   []byte
}

// Run converts every document of the source format and removes the originals
// only after every target has converted and validated successfully.
func Run(root string, cfg *config.Config, directory, format string) (Result, error) {
	projectFiles, err := projectfs.Open(root)
	if err != nil {
		return Result{}, err
	}
	defer projectFiles.Close()
	contentFiles := projectFiles
	// An explicit CLI selection owns a separate directory boundary. Default
	// project content and borrowed authoring operations stay inside the project.
	if strings.TrimSpace(directory) != "" {
		selected := conversionDirectory(root, cfg, directory)
		if _, err := projectFiles.Relative(selected); errors.Is(err, projectfs.ErrOutside) {
			contentFiles, err = projectfs.Open(selected)
			if err != nil {
				return Result{}, err
			}
			defer contentFiles.Close()
		}
	}
	return runConversion(root, cfg, directory, format, projectFiles, contentFiles)
}

// RunRoot confines an authoring conversion to its borrowed project boundary.
func RunRoot(root string, files *projectfs.FS, cfg *config.Config, directory, format string) (Result, error) {
	if files == nil {
		return Result{}, errors.New("conversion project root is required")
	}
	return runConversion(root, cfg, directory, format, files, files)
}

func conversionDirectory(root string, cfg *config.Config, directory string) string {
	if strings.TrimSpace(directory) == "" {
		return cfg.ContentPath(root)
	}
	if !filepath.IsAbs(directory) {
		return filepath.Join(root, filepath.FromSlash(directory))
	}
	return directory
}

func runConversion(root string, cfg *config.Config, directory, format string, projectFiles, contentFiles *projectfs.FS) (Result, error) {
	return runConversionWithOps(root, cfg, directory, format, projectFiles, contentFiles, defaultConversionOps)
}

func runConversionWithOps(root string, cfg *config.Config, directory, format string, projectFiles, contentFiles *projectfs.FS, ops conversionOps) (Result, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "mpd" && format != "markdown" {
		return Result{}, errors.New("target format must be mpd or markdown")
	}
	directory = conversionDirectory(root, cfg, directory)

	var conversions []conversion
	var documents []translate.ConvertedDocument
	err := contentFiles.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if (format == "mpd" && extension != ".md" && extension != ".markdown") || (format == "markdown" && extension != ".mpd") {
			return nil
		}
		targetExtension := ".mpd"
		if format == "markdown" {
			targetExtension = ".md"
		}
		target := strings.TrimSuffix(path, filepath.Ext(path)) + targetExtension
		if _, statErr := contentFiles.Stat(target); statErr == nil {
			return fmt.Errorf("refusing to replace existing target file %s", target)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		source, readErr := contentFiles.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var converted []byte
		if format == "mpd" {
			result, convertErr := importer.MarkdownToMPD(string(source))
			if convertErr != nil {
				return fmt.Errorf("convert %s: %w", path, convertErr)
			}
			converted = []byte(result)
			document := mpd.Parse(target, converted)
			for _, diagnostic := range document.Diagnostics {
				if diagnostic.Severity == mpd.SeverityError {
					return fmt.Errorf("convert %s: line %d: %s", path, diagnostic.Position.Line, diagnostic.Message)
				}
			}
			if _, renderErr := mpd.Markdown(document); renderErr != nil {
				return fmt.Errorf("render converted %s: %w", path, renderErr)
			}
		} else {
			document := mpd.Parse(path, source)
			result, renderErr := mpd.Markdown(document)
			if renderErr != nil {
				return fmt.Errorf("convert %s: %w", path, renderErr)
			}
			converted = result
		}
		info, infoErr := contentFiles.Stat(path)
		if infoErr != nil {
			return infoErr
		}
		conversions = append(conversions, conversion{source: path, target: target, mode: info.Mode().Perm(), data: converted})
		relative, err := filepath.Rel(cfg.ContentPath(root), path)
		if err != nil {
			return err
		}
		targetRelative, err := filepath.Rel(cfg.ContentPath(root), target)
		if err != nil {
			return err
		}
		if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			documents = append(documents, translate.ConvertedDocument{SourceFile: filepath.ToSlash(relative), TargetFile: filepath.ToSlash(targetRelative), Source: source, Target: converted})
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	if len(conversions) == 0 {
		if format == "mpd" {
			return Result{}, errors.New("no Markdown documents were found")
		}
		return Result{}, errors.New("no MPD documents were found")
	}

	states, err := translate.PlanConversionStatesRoot(projectFiles, root, *cfg, documents, importer.MarkdownToMPD)
	if err != nil {
		return Result{}, err
	}
	count := len(conversions)
	review := 0
	for _, state := range states {
		conversions = append(conversions, conversion{source: state.Path, target: state.Path, mode: 0644, data: state.Data})
		review += state.RequiresReview
	}
	plannedTargets := make(map[string]string, len(conversions))
	for _, item := range conversions {
		target := filepath.Clean(item.target)
		if previous, exists := plannedTargets[target]; exists {
			return Result{}, fmt.Errorf("refusing to convert %s and %s to the same target %s", previous, item.source, item.target)
		}
		plannedTargets[target] = item.source
	}

	journal := newRecoveryJournal(ops, contentFiles, directory)
	fail := func(err error) (Result, error) {
		if rollbackErr := journal.rollback(); rollbackErr != nil {
			return Result{}, fmt.Errorf("%w; %w", err, rollbackErr)
		}
		return Result{}, err
	}

	for _, item := range conversions {
		if item.source == item.target {
			index, err := journal.moveAside(projectFiles, item.target)
			if err != nil {
				return fail(err)
			}
			if err := ops.write(projectFiles, item.target, item.data, item.mode); err != nil {
				return fail(err)
			}
			journal.recordWritten(index, item.data)
			continue
		}
		published, err := ops.writeNew(contentFiles, item.target, item.data, item.mode)
		if published {
			journal.recordCreated(contentFiles, item.target, item.data)
		}
		if err != nil {
			return fail(err)
		}
	}
	for _, item := range conversions {
		if item.source == item.target {
			continue
		}
		if _, err := journal.moveAside(contentFiles, item.source); err != nil {
			return fail(err)
		}
	}

	updated := *cfg
	configChanged := false
	for _, reference := range []*string{&updated.Translation.StyleGuide, &updated.Translation.Glossary} {
		if strings.TrimSpace(*reference) == "" {
			continue
		}
		absolute := *reference
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(root, filepath.FromSlash(absolute))
		}
		for _, item := range conversions {
			if filepath.Clean(absolute) != filepath.Clean(item.source) {
				continue
			}
			relative, relErr := filepath.Rel(root, item.target)
			if relErr != nil {
				return fail(relErr)
			}
			*reference = filepath.ToSlash(relative)
			configChanged = true
			break
		}
	}
	if configChanged {
		info, statErr := ops.stat(projectFiles, config.Filename)
		switch {
		case statErr == nil && info.Mode().IsRegular():
			if _, err := journal.moveAside(projectFiles, config.Filename); err != nil {
				return fail(err)
			}
		case statErr == nil:
			// Keep a non-regular destination in place so the save reports it
			// instead of silently replacing an unrelated directory.
		case errors.Is(statErr, os.ErrNotExist):
			journal.recordCreated(projectFiles, config.Filename, nil)
		default:
			return fail(statErr)
		}
		if err := ops.saveConfig(projectFiles, updated); err != nil {
			return fail(fmt.Errorf("update converted configuration paths: %w", err))
		}
		*cfg = updated
	}

	sourceLabel, targetLabel := "Markdown", "MPD"
	if format == "markdown" {
		sourceLabel, targetLabel = "MPD", "Markdown"
	}
	result := Result{Count: count, Directory: directory, SourceFormat: sourceLabel, TargetFormat: targetLabel, MigratedStates: len(states), MigrationReview: review}
	if err := journal.commit(); err != nil {
		return result, fmt.Errorf("conversion succeeded but %w", err)
	}
	return result, nil
}
