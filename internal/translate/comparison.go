package translate

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/leaanthony/mpress/internal/config"
)

// ProjectEstimate describes the approximate provider workload for one complete
// target language and for the missing or stale work across configured targets.
// Token counts are estimates because OpenRouter models use different tokenizers.
type ProjectEstimate struct {
	Files                 int `json:"files"`
	Segments              int `json:"segments"`
	Requests              int `json:"requests"`
	InputTokens           int `json:"inputTokens"`
	OutputTokens          int `json:"outputTokens"`
	TargetLanguages       int `json:"targetLanguages"`
	RemainingSegments     int `json:"remainingSegments"`
	RemainingInputTokens  int `json:"remainingInputTokens"`
	RemainingOutputTokens int `json:"remainingOutputTokens"`
}

// ComparisonCandidate is one model used in a blinded sample comparison.
type ComparisonCandidate struct {
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoningEffort,omitempty"`
}

type ComparisonText struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Section string `json:"section,omitempty"`
	Text    string `json:"text"`
}

type ComparisonOutput struct {
	Model      string           `json:"model"`
	Texts      []ComparisonText `json:"texts"`
	DurationMS int64            `json:"durationMs"`
}

type ModelComparison struct {
	SourceFile            string             `json:"sourceFile"`
	TargetLanguage        string             `json:"targetLanguage"`
	Source                []ComparisonText   `json:"source"`
	Outputs               []ComparisonOutput `json:"outputs"`
	EstimatedInputTokens  int                `json:"estimatedInputTokens"`
	EstimatedOutputTokens int                `json:"estimatedOutputTokens"`
}

// EstimateProject measures the actual translatable segments and then estimates
// tokens for the same JSON and schema shape used by the translation provider.
func EstimateProject(project string, cfg config.Config) (ProjectEstimate, error) {
	engine := NewEngine(project, cfg, nil)
	files, err := engine.sourceFiles("")
	if err != nil {
		return ProjectEstimate{}, err
	}
	styleGuide, err := readOptionalProjectFile(project, cfg.Translation.StyleGuide, 128<<10)
	if err != nil {
		return ProjectEstimate{}, err
	}
	var estimate ProjectEstimate
	for _, sourceFile := range files {
		data, readErr := os.ReadFile(filepath.Join(cfg.ContentPath(project), filepath.FromSlash(sourceFile)))
		if readErr != nil {
			return ProjectEstimate{}, readErr
		}
		document, extractErr := extractFile(sourceFile, cfg.Build.NavFile, data)
		if extractErr != nil {
			return ProjectEstimate{}, extractErr
		}
		estimate.Files++
		estimate.Segments += len(document.Segments)
		segments := make([]RequestSegment, 0, len(document.Segments))
		for _, segment := range document.Segments {
			segments = append(segments, RequestSegment{ID: segment.ID, Kind: segment.Kind, Section: segment.Section, Text: segment.Text})
		}
		for _, batch := range segmentBatches(segments, 18_000) {
			before, after := surroundingContext(document.Segments, batch)
			request := TranslationRequest{
				SourceLanguage: cfg.Translation.SourceLanguage, TargetLanguage: "target",
				PageTitle: document.Title, Outline: document.Outline, StyleGuide: styleGuide,
				PreviousContext: before, FollowingContext: after, Segments: batch,
			}
			payload, _ := json.Marshal(request)
			schema, _ := json.Marshal(translationSchema(batch))
			estimate.Requests++
			estimate.InputTokens += estimatedTokens(len(translationSystemPrompt("markdown")) + len(payload) + len(schema))
			outputCharacters := 32
			for _, segment := range batch {
				outputCharacters += utf8.RuneCountInString(segment.Text) + len(segment.ID) + 24
			}
			// Translated text can tokenize more densely than the English source,
			// especially for CJK languages, so use three characters per token.
			estimate.OutputTokens += int(math.Ceil(float64(outputCharacters) / 3.0))
		}
	}
	targets, err := engine.targetLanguages("")
	if err != nil {
		// Model setup is still useful before the first target is added.
		targets = nil
	}
	estimate.TargetLanguages = len(targets)
	if len(targets) == 0 || estimate.Segments == 0 {
		return estimate, nil
	}
	report, err := engine.Run(context.Background(), Options{Scope: "stale", DryRun: true})
	if err != nil {
		return ProjectEstimate{}, err
	}
	estimate.RemainingSegments = report.Pending
	allTargetSegments := estimate.Segments * len(targets)
	ratio := 0.0
	if allTargetSegments > 0 {
		ratio = float64(report.Pending) / float64(allTargetSegments)
	}
	estimate.RemainingInputTokens = int(math.Ceil(float64(estimate.InputTokens*len(targets)) * ratio))
	estimate.RemainingOutputTokens = int(math.Ceil(float64(estimate.OutputTokens*len(targets)) * ratio))
	return estimate, nil
}

func estimatedTokens(characters int) int {
	if characters <= 0 {
		return 0
	}
	return int(math.Ceil(float64(characters) / 4.0))
}

// CompareProjectModels translates one random, compact source sample with both
// candidates. It validates the sample in its original Markdown page but never
// writes translated content or translation state.
func CompareProjectModels(ctx context.Context, project string, cfg config.Config, language string, candidates []ComparisonCandidate) (ModelComparison, error) {
	return CompareProjectModelsForFile(ctx, project, cfg, language, "", candidates)
}

// CompareProjectModelsForFile compares a compact random sample from one page.
// An empty sourceFile keeps the site-wide random sampling behaviour.
func CompareProjectModelsForFile(ctx context.Context, project string, cfg config.Config, language, sourceFile string, candidates []ComparisonCandidate) (ModelComparison, error) {
	if len(candidates) != 2 || strings.TrimSpace(candidates[0].Model) == "" || strings.TrimSpace(candidates[1].Model) == "" {
		return ModelComparison{}, errors.New("choose exactly two translation models")
	}
	if candidates[0].Model == candidates[1].Model {
		return ModelComparison{}, errors.New("choose two different translation models")
	}
	engine := NewEngine(project, cfg, nil)
	languages, err := engine.targetLanguages(language)
	if err != nil || len(languages) != 1 {
		if err != nil {
			return ModelComparison{}, err
		}
		return ModelComparison{}, errors.New("choose one target language")
	}
	document, sourceFile, sample, err := randomComparisonSample(project, cfg, sourceFile)
	if err != nil {
		return ModelComparison{}, err
	}
	styleGuide, err := readOptionalProjectFile(project, cfg.Translation.StyleGuide, 128<<10)
	if err != nil {
		return ModelComparison{}, err
	}
	glossary, err := loadGlossary(project, cfg.Translation.Glossary)
	if err != nil {
		return ModelComparison{}, err
	}
	before, after := surroundingContext(document.Segments, sample)
	request := TranslationRequest{
		SourceLanguage: cfg.Translation.SourceLanguage, TargetLanguage: language,
		PageTitle: document.Title, Outline: document.Outline, StyleGuide: styleGuide,
		Glossary: glossary[language], PreviousContext: before, FollowingContext: after, Segments: sample,
	}
	payload, _ := json.Marshal(request)
	schema, _ := json.Marshal(translationSchema(sample))
	comparison := ModelComparison{
		SourceFile: sourceFile, TargetLanguage: language,
		EstimatedInputTokens: estimatedTokens(len(translationSystemPrompt("markdown")) + len(payload) + len(schema)),
	}
	segmentByID := map[string]Segment{}
	for _, segment := range document.Segments {
		segmentByID[segment.ID] = segment
	}
	for _, segment := range sample {
		original := segmentByID[segment.ID]
		comparison.Source = append(comparison.Source, ComparisonText{ID: segment.ID, Kind: segment.Kind, Section: segment.Section, Text: original.Original})
		comparison.EstimatedOutputTokens += estimatedTokens(utf8.RuneCountInString(segment.Text)*4 + len(segment.ID) + 24)
	}

	type result struct {
		index  int
		output ComparisonOutput
		err    error
	}
	results := make(chan result, 2)
	var group sync.WaitGroup
	for index, candidate := range candidates {
		group.Add(1)
		go func(index int, candidate ComparisonCandidate) {
			defer group.Done()
			provider, providerErr := NewOpenAICompatibleProvider(ProviderOptions{
				Name: cfg.Translation.Provider, Model: candidate.Model, BaseURL: cfg.Translation.BaseURL,
				APIKey: ResolveAPIKey(cfg), RequireParameters: true,
				DataCollection: cfg.Translation.DataCollection, HTTPReferer: cfg.Site.BaseURL,
				ApplicationTitle: "M-Press model comparison", ReasoningEffort: candidate.ReasoningEffort,
			})
			if providerErr != nil {
				results <- result{index: index, err: providerErr}
				return
			}
			started := time.Now()
			values, translateErr := provider.Translate(ctx, request)
			output := ComparisonOutput{Model: candidate.Model, DurationMS: time.Since(started).Milliseconds()}
			if translateErr == nil {
				candidatePage, applyErr := Apply(document, values)
				if applyErr == nil {
					applyErr = validateFile(sourceFile, cfg.Build.NavFile, document, candidatePage)
				}
				if applyErr != nil {
					translateErr = applyErr
				}
			}
			if translateErr == nil {
				for _, segment := range sample {
					restored, restoreErr := restore(segmentByID[segment.ID], values[segment.ID], document.Format == "mpd")
					if restoreErr != nil {
						translateErr = restoreErr
						break
					}
					output.Texts = append(output.Texts, ComparisonText{ID: segment.ID, Kind: segment.Kind, Section: segment.Section, Text: restored})
				}
			}
			results <- result{index: index, output: output, err: translateErr}
		}(index, candidate)
	}
	group.Wait()
	close(results)
	comparison.Outputs = make([]ComparisonOutput, 2)
	for item := range results {
		if item.err != nil {
			return ModelComparison{}, fmt.Errorf("compare model %s: %w", candidates[item.index].Model, item.err)
		}
		comparison.Outputs[item.index] = item.output
	}
	return comparison, nil
}

func randomComparisonSample(project string, cfg config.Config, sourceFile string) (*Document, string, []RequestSegment, error) {
	engine := NewEngine(project, cfg, nil)
	files, err := engine.sourceFiles(sourceFile)
	if err != nil {
		return nil, "", nil, err
	}
	type page struct {
		file     string
		document *Document
		starts   []int
	}
	var pages []page
	for _, sourceFile := range files {
		if filepath.ToSlash(sourceFile) == filepath.ToSlash(cfg.Build.NavFile) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(cfg.ContentPath(project), filepath.FromSlash(sourceFile)))
		if readErr != nil {
			return nil, "", nil, readErr
		}
		document, extractErr := Extract(data)
		if extractErr != nil {
			return nil, "", nil, extractErr
		}
		var starts []int
		for index, segment := range document.Segments {
			if segment.Kind != "frontmatter" && utf8.RuneCountInString(strings.TrimSpace(segment.Original)) >= 24 {
				starts = append(starts, index)
			}
		}
		if len(starts) > 0 {
			pages = append(pages, page{file: sourceFile, document: document, starts: starts})
		}
	}
	if len(pages) == 0 {
		return nil, "", nil, errors.New("the site has no suitable prose sample")
	}
	pageIndex, err := secureIndex(len(pages))
	if err != nil {
		return nil, "", nil, err
	}
	selected := pages[pageIndex]
	startIndex, err := secureIndex(len(selected.starts))
	if err != nil {
		return nil, "", nil, err
	}
	start := selected.starts[startIndex]
	var sample []RequestSegment
	characters := 0
	for index := start; index < len(selected.document.Segments) && len(sample) < 6; index++ {
		segment := selected.document.Segments[index]
		if segment.Kind == "frontmatter" || utf8.RuneCountInString(strings.TrimSpace(segment.Original)) < 8 {
			continue
		}
		sample = append(sample, RequestSegment{ID: segment.ID, Kind: segment.Kind, Section: segment.Section, Text: segment.Text})
		characters += utf8.RuneCountInString(segment.Original)
		if characters >= 520 && len(sample) >= 2 {
			break
		}
	}
	if len(sample) == 0 {
		return nil, "", nil, errors.New("the site has no suitable prose sample")
	}
	return selected.document, filepath.ToSlash(selected.file), sample, nil
}

func secureIndex(length int) (int, error) {
	if length <= 1 {
		return 0, nil
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}
