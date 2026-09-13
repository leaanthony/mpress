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
)

// Result describes a complete, atomic project content conversion.
type Result struct {
	Count        int    `json:"count"`
	Directory    string `json:"directory"`
	SourceFormat string `json:"sourceFormat"`
	TargetFormat string `json:"targetFormat"`
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
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "mpd" && format != "markdown" {
		return Result{}, errors.New("target format must be mpd or markdown")
	}
	if strings.TrimSpace(directory) == "" {
		directory = cfg.ContentPath(root)
	} else if !filepath.IsAbs(directory) {
		directory = filepath.Join(root, filepath.FromSlash(directory))
	}

	var conversions []conversion
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, walkErr error) error {
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
		if _, statErr := os.Stat(target); statErr == nil {
			return fmt.Errorf("refusing to replace existing target file %s", target)
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		source, readErr := os.ReadFile(path)
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
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		conversions = append(conversions, conversion{source: path, target: target, mode: info.Mode().Perm(), data: converted})
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

	for _, item := range conversions {
		temporary := item.target + ".tmp"
		if err := os.WriteFile(temporary, item.data, item.mode); err != nil {
			return Result{}, err
		}
		if err := os.Rename(temporary, item.target); err != nil {
			_ = os.Remove(temporary)
			return Result{}, err
		}
	}
	for _, item := range conversions {
		if err := os.Remove(item.source); err != nil {
			return Result{}, err
		}
	}

	configChanged := false
	for _, reference := range []*string{&cfg.Translation.StyleGuide, &cfg.Translation.Glossary} {
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
				return Result{}, relErr
			}
			*reference = filepath.ToSlash(relative)
			configChanged = true
			break
		}
	}
	if configChanged {
		if err := config.Save(root, *cfg); err != nil {
			return Result{}, fmt.Errorf("update converted configuration paths: %w", err)
		}
	}

	sourceLabel, targetLabel := "Markdown", "MPD"
	if format == "markdown" {
		sourceLabel, targetLabel = "MPD", "Markdown"
	}
	return Result{Count: len(conversions), Directory: directory, SourceFormat: sourceLabel, TargetFormat: targetLabel}, nil
}
