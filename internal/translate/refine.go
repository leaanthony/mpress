package translate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RefinementReport describes targeted changes made after an independent audit.
type RefinementReport struct {
	Language string `json:"language"`
	Files    int    `json:"files"`
	Segments int    `json:"segments"`
	Written  int    `json:"written"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

// RefineWithProvider repairs only segments named by audit findings. Clean
// translations are reused byte for byte rather than being sent for rewriting.
func (e *Engine) RefineWithProvider(ctx context.Context, language, sourceFile string, findings []AuditFinding, provider Provider) (RefinementReport, error) {
	if provider == nil {
		return RefinementReport{}, errors.New("translation refinement provider is not configured")
	}
	languages, err := e.targetLanguages(language)
	if err != nil {
		return RefinementReport{}, err
	}
	if len(languages) != 1 {
		return RefinementReport{}, errors.New("translation refinement requires one target language")
	}
	language = languages[0]
	files, err := e.sourceFiles(sourceFile)
	if err != nil {
		return RefinementReport{}, err
	}
	requested := map[string]map[string][]string{}
	for _, finding := range findings {
		if finding.Segment == "" || (finding.Severity != "error" && finding.Severity != "warning") {
			continue
		}
		if requested[finding.File] == nil {
			requested[finding.File] = map[string][]string{}
		}
		requested[finding.File][finding.Segment] = append(requested[finding.File][finding.Segment], finding.Message)
	}
	report := RefinementReport{Language: language, Provider: provider.Name(), Model: provider.Model()}
	if len(requested) == 0 {
		return report, nil
	}
	styleGuide, err := readOptionalProjectFile(e.Project, e.Config.Translation.StyleGuide, 128<<10)
	if err != nil {
		return report, fmt.Errorf("read translation style guide: %w", err)
	}
	glossaries, err := loadGlossary(e.Project, e.Config.Translation.Glossary)
	if err != nil {
		return report, err
	}
	contentDir := e.Config.ContentPath(e.Project)
	for _, file := range files {
		notes := requested[file]
		if len(notes) == 0 {
			continue
		}
		report.Files++
		sourceBytes, readErr := os.ReadFile(filepath.Join(contentDir, filepath.FromSlash(file)))
		if readErr != nil {
			return report, readErr
		}
		targetPath := filepath.Join(contentDir, filepath.FromSlash(language), filepath.FromSlash(file))
		targetBytes, readErr := os.ReadFile(targetPath)
		if readErr != nil {
			return report, readErr
		}
		sourceDoc, parseErr := extractFile(file, e.Config.Build.NavFile, sourceBytes)
		if parseErr != nil {
			return report, parseErr
		}
		targetDoc, parseErr := extractTargetFile(file, e.Config.Build.NavFile, sourceBytes, targetBytes)
		if parseErr != nil {
			return report, parseErr
		}
		stateFile, stateErr := statePath(e.Project, e.Config.Translation.StateDir, language, file)
		if stateErr != nil {
			return report, stateErr
		}
		state, stateErr := loadState(stateFile, file, e.Config.Site.DefaultLanguage, language, sourceDoc.TranslationKey)
		if stateErr != nil {
			return report, stateErr
		}

		if stateErr = checkStateForUpdate(state, file); stateErr != nil {
			return report, stateErr
		}
		targets := make(map[string]Segment, len(targetDoc.Segments))
		for _, segment := range targetDoc.Segments {
			targets[segment.ID] = segment
		}
		var pending []RequestSegment
		for segmentIndex, segment := range sourceDoc.Segments {
			messages := notes[segment.ID]
			if len(messages) == 0 || segment.Protected {
				continue
			}
			target, ok := targets[segment.ID]
			if !ok {
				continue
			}
			current, prepareErr := prepareExistingForFormat(sourceDoc.Format, &segment, target)
			if prepareErr == nil && segment.Text != sourceDoc.Segments[segmentIndex].Text {
				// Refinement still starts from English source tokens. Unrefined segments
				// below retain the target representation; never send it as source prose.
				segment = sourceDoc.Segments[segmentIndex]
				current = target.Original
				messages = append(append([]string(nil), messages...), "The current translation uses equivalent formatting. Rebuild from the source while preserving every source placeholder.")
			} else {
				sourceDoc.Segments[segmentIndex] = segment
			}
			if prepareErr != nil {
				current = target.Original
				messages = append(append([]string(nil), messages...), "The existing translation has damaged protected content. Rebuild this segment from the source, preserving every source placeholder exactly. "+prepareErr.Error())
			}
			pending = append(pending, RequestSegment{ID: segment.ID, Kind: segment.Kind, Section: segment.Section, Text: segment.Text, CurrentText: current, ReviewNotes: strings.Join(messages, "; ")})
		}
		if len(pending) == 0 {
			continue
		}
		refinementBatchSize := 3_000
		if provider.Name() == "codex" || provider.Name() == "claude" {
			refinementBatchSize = 12_000
		}
		refined := make(map[string]string, len(pending))
		for _, batch := range segmentBatches(pending, refinementBatchSize) {
			before, after := surroundingContext(sourceDoc.Segments, batch)
			request := TranslationRequest{Mode: "refine", Format: sourceDoc.Format, SourceLanguage: e.Config.Translation.SourceLanguage, TargetLanguage: language, LocaleGuidance: localeGuidance(language), PageTitle: sourceDoc.Title, Outline: sourceDoc.Outline, PreviousContext: before, FollowingContext: after, StyleGuide: styleGuide, Glossary: glossaries[language], Segments: batch}
			requestContext, cancel := context.WithTimeout(ctx, 4*time.Minute)
			batchRefined, refineErr := provider.Translate(requestContext, request)
			cancel()
			if refineErr != nil {
				return report, fmt.Errorf("refine %s: %w", file, refineErr)
			}
			for id, value := range batchRefined {
				refined[id] = value
			}
		}
		values := make(map[string]string, len(sourceDoc.Segments))
		for segmentIndex, segment := range sourceDoc.Segments {
			if value, ok := refined[segment.ID]; ok {
				values[segment.ID] = value
				continue
			}
			target, ok := targets[segment.ID]
			if !ok {
				return report, fmt.Errorf("refine %s: target segment %s is missing", file, segment.ID)
			}
			value, prepareErr := prepareExistingForFormat(sourceDoc.Format, &segment, target)
			sourceDoc.Segments[segmentIndex] = segment
			if prepareErr != nil {
				return report, prepareErr
			}
			values[segment.ID] = value
		}
		output, applyErr := Apply(sourceDoc, values)
		if applyErr != nil {
			return report, applyErr
		}
		if validateErr := validateFile(file, e.Config.Build.NavFile, sourceDoc, output); validateErr != nil {
			return report, validateErr
		}
		if filepath.ToSlash(file) != filepath.ToSlash(e.Config.Build.NavFile) {
			output, applyErr = rewriteLocalFragments(file, e.Config.Site.DefaultLanguage, language, sourceBytes, output)
			if applyErr != nil {
				return report, applyErr
			}
		}
		if writeErr := writeAtomic(targetPath, output); writeErr != nil {
			return report, writeErr
		}
		for _, segment := range sourceDoc.Segments {
			value, changed := refined[segment.ID]
			if !changed {
				if _, tracked := state.Segments[segment.ID]; !tracked {
					state.Segments[segment.ID] = SegmentState{SourceHash: segment.SourceHash, TargetHash: Hash(targets[segment.ID].Original), Status: "migration-review", SourcePreview: preview(segment.Original)}
				}
				continue
			}
			restored, restoreErr := restore(segment, value, preservesMarkdownSyntax(sourceDoc.Format))
			if restoreErr != nil {
				return report, restoreErr
			}
			entry := state.Segments[segment.ID]
			entry.SourceHash = segment.SourceHash
			entry.TargetHash = Hash(restored)
			entry.MachineHash = Hash(restored)
			entry.MachineText = restored
			entry.Status = "machine-refined"
			entry.Provider = provider.Name()
			entry.Model = providerModelForLanguage(provider, language)
			entry.PromptVersion = promptVersion
			entry.UpdatedAt = nowString()
			state.Segments[segment.ID] = entry
			report.Segments++
		}
		if stateErr = saveState(stateFile, state); stateErr != nil {
			return report, stateErr
		}
		report.Written++
	}
	return report, nil
}
