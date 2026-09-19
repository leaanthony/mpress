package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/mpd"
	"gopkg.in/yaml.v3"
)

type Options struct {
	Language string
	File     string
	Scope    string
	DryRun   bool
	Force    bool
	Workers  int
}

type FileReport struct {
	SourceFile       string         `json:"sourceFile"`
	TargetFile       string         `json:"targetFile"`
	TargetLanguage   string         `json:"targetLanguage"`
	Segments         int            `json:"segments"`
	NeedsTranslation int            `json:"needsTranslation"`
	Translated       int            `json:"translated"`
	States           map[string]int `json:"states"`
	Written          bool           `json:"written"`
	Estimate         UsageEstimate  `json:"estimate,omitempty"`
}

type Report struct {
	Files    []FileReport  `json:"files"`
	Segments int           `json:"segments"`
	Pending  int           `json:"pending"`
	Written  int           `json:"written"`
	Estimate UsageEstimate `json:"estimate,omitempty"`
}

// UsageEstimate is a conservative, non-billable estimate of the provider work
// described by a dry-run translation plan. Token counts are approximate because
// provider tokenizers and translated text lengths differ by model and language.
type UsageEstimate struct {
	Requests          int     `json:"requests"`
	SourceCharacters  int     `json:"sourceCharacters"`
	InputTokens       int     `json:"inputTokens"`
	OutputTokens      int     `json:"outputTokens"`
	InputCostUSD      float64 `json:"inputCostUSD,omitempty"`
	OutputCostUSD     float64 `json:"outputCostUSD,omitempty"`
	TotalCostUSD      float64 `json:"totalCostUSD,omitempty"`
	PricingConfigured bool    `json:"pricingConfigured"`
}

func (estimate *UsageEstimate) add(other UsageEstimate) {
	estimate.Requests += other.Requests
	estimate.SourceCharacters += other.SourceCharacters
	estimate.InputTokens += other.InputTokens
	estimate.OutputTokens += other.OutputTokens
	estimate.InputCostUSD += other.InputCostUSD
	estimate.OutputCostUSD += other.OutputCostUSD
	estimate.TotalCostUSD += other.TotalCostUSD
	estimate.PricingConfigured = estimate.PricingConfigured || other.PricingConfigured
}

func (e *Engine) Mark(language, sourceFile, status string) (FileReport, error) {
	if status != "reviewed" && status != "final" {
		return FileReport{}, errors.New("translation review status must be reviewed or final")
	}
	languages, err := e.targetLanguages(language)
	if err != nil {
		return FileReport{}, err
	}
	if len(languages) != 1 {
		return FileReport{}, errors.New("one target language is required")
	}
	files, err := e.sourceFiles(sourceFile)
	if err != nil {
		return FileReport{}, err
	}
	if len(files) != 1 {
		return FileReport{}, errors.New("one source page is required")
	}
	sourceFile = files[0]
	contentDir := e.Config.ContentPath(e.Project)
	sourceBytes, err := os.ReadFile(filepath.Join(contentDir, filepath.FromSlash(sourceFile)))
	if err != nil {
		return FileReport{}, err
	}
	sourceDoc, err := extractFile(sourceFile, e.Config.Build.NavFile, sourceBytes)
	if err != nil {
		return FileReport{}, err
	}
	stateFile, err := statePath(e.Project, e.Config.Translation.StateDir, language, sourceFile)
	if err != nil {
		return FileReport{}, err
	}
	statePathToLoad := stateFile
	if sourceDoc.TranslationKey != "" {
		if found, findErr := findStateByPageKey(filepath.Join(e.Project, filepath.FromSlash(e.Config.Translation.StateDir), language), sourceDoc.TranslationKey); findErr != nil {
			return FileReport{}, findErr
		} else if found != "" {
			statePathToLoad = found
		}
	}
	state, err := loadState(statePathToLoad, sourceFile, e.Config.Translation.SourceLanguage, language, sourceDoc.TranslationKey)
	if err != nil {
		return FileReport{}, err
	}
	targetBytes, err := os.ReadFile(filepath.Join(contentDir, filepath.FromSlash(language), filepath.FromSlash(sourceFile)))
	if err != nil {
		return FileReport{}, errors.New("translate the page before marking its review state")
	}
	targetDoc, err := extractTargetFile(sourceFile, e.Config.Build.NavFile, sourceBytes, targetBytes)
	if err != nil {
		return FileReport{}, err
	}
	if err := checkStateExtractor(state, sourceFile); err != nil {
		return FileReport{}, err
	}
	if err := validateFile(sourceFile, e.Config.Build.NavFile, sourceDoc, targetBytes); err != nil {
		return FileReport{}, err
	}
	targetByID := map[string]Segment{}
	for _, segment := range targetDoc.Segments {
		targetByID[segment.ID] = segment
	}
	matches := matchSegments(sourceDoc.Segments, state.Segments)
	layout := inferTargetLayout(matches, state.Segments, targetByID)
	reviewed := make(map[string]SegmentState, len(sourceDoc.Segments))
	report := FileReport{SourceFile: sourceFile, TargetFile: filepath.ToSlash(filepath.Join(language, sourceFile)), TargetLanguage: language, Segments: len(sourceDoc.Segments), States: map[string]int{status: len(sourceDoc.Segments)}}
	for _, segment := range sourceDoc.Segments {
		oldID := matches[segment.ID]
		old, ok := state.Segments[oldID]
		target, err := reusableTarget(segment.ID, oldID, old, targetByID, layout)
		if err != nil {
			return FileReport{}, err
		}
		targetText := target.Original
		if !ok || targetText == "" {
			return FileReport{}, fmt.Errorf("segment %s has not been translated", segment.ID)
		}
		if old.SourceHash != segment.SourceHash {
			return FileReport{}, fmt.Errorf("segment %s is stale and cannot be marked %s", segment.ID, status)
		}
		old.Status = status
		old.TargetHash = Hash(targetText)
		old.SourceHash = segment.SourceHash
		old.UpdatedAt = nowString()
		reviewed[segment.ID] = old
	}
	state.Segments = reviewed
	state.SourceFile = filepath.ToSlash(sourceFile)
	state.PageKey = newFileState(sourceFile, e.Config.Translation.SourceLanguage, language, sourceDoc.TranslationKey).PageKey
	if err := saveState(stateFile, state); err != nil {
		return FileReport{}, err
	}
	if statePathToLoad != stateFile {
		_ = os.Remove(statePathToLoad)
	}
	return report, nil
}

type Engine struct {
	Project  string
	Config   config.Config
	Provider Provider
	// Supplied by CLI/project conversion to keep the translation core independent of importers.
	MarkdownToMPD func(string) (string, error)
}

func NewEngine(project string, cfg config.Config, provider Provider) *Engine {
	return &Engine{Project: project, Config: cfg, Provider: provider}
}

func NewConfiguredEngine(project string, cfg config.Config) (*Engine, error) {
	provider := &configuredProvider{cfg: cfg}
	return NewEngine(project, cfg, provider), nil
}

type configuredProvider struct {
	cfg       config.Config
	mu        sync.Mutex
	providers map[string]Provider
	errors    map[string]error
}

func (p *configuredProvider) Name() string  { return p.cfg.Translation.Provider }
func (p *configuredProvider) Model() string { return p.cfg.Translation.Model }
func providerModelForLanguage(provider Provider, language string) string {
	if configured, ok := provider.(*configuredProvider); ok {
		if selection, exists := configured.cfg.Translation.LanguageModels[language]; exists {
			return selection.Model
		}
	}
	return provider.Model()
}
func (p *configuredProvider) Translate(ctx context.Context, request TranslationRequest) (map[string]string, error) {
	model := p.cfg.Translation.Model
	reasoningEffort := p.cfg.Translation.ReasoningEffort
	if selection, ok := p.cfg.Translation.LanguageModels[request.TargetLanguage]; ok {
		model = selection.Model
		reasoningEffort = selection.ReasoningEffort
	}
	key := model + "\x00" + reasoningEffort
	p.mu.Lock()
	if p.providers == nil {
		p.providers = map[string]Provider{}
		p.errors = map[string]error{}
	}
	provider, found := p.providers[key]
	providerErr := p.errors[key]
	if !found && providerErr == nil {
		if p.cfg.Translation.Provider == "codex" || p.cfg.Translation.Provider == "claude" {
			provider, providerErr = NewLocalProvider(p.cfg.Translation.Provider, model, p.cfg.Translation.Command)
		} else {
			provider, providerErr = NewOpenAICompatibleProvider(ProviderOptions{
				Name: p.cfg.Translation.Provider, Model: model, BaseURL: p.cfg.Translation.BaseURL,
				APIKey: ResolveAPIKey(p.cfg), RequireParameters: p.cfg.Translation.RequireParameters,
				DataCollection: p.cfg.Translation.DataCollection, HTTPReferer: p.cfg.Site.BaseURL,
				ApplicationTitle: "M-Press", ReasoningEffort: reasoningEffort,
			})
		}
		if providerErr != nil {
			p.errors[key] = providerErr
		} else {
			p.providers[key] = provider
		}
	}
	p.mu.Unlock()
	if providerErr != nil {
		return nil, providerErr
	}
	return provider.Translate(ctx, request)
}

func (e *Engine) Run(ctx context.Context, options Options) (Report, error) {
	if options.Scope == "" {
		options.Scope = "stale"
	}
	if options.Scope != "missing" && options.Scope != "stale" && options.Scope != "all" {
		return Report{}, fmt.Errorf("unknown translation scope %q", options.Scope)
	}
	languages, err := e.targetLanguages(options.Language)
	if err != nil {
		return Report{}, err
	}
	files, err := e.sourceFiles(options.File)
	if err != nil {
		return Report{}, err
	}
	styleGuide, err := readOptionalProjectFile(e.Project, e.Config.Translation.StyleGuide, 128<<10)
	if err != nil {
		return Report{}, fmt.Errorf("read translation style guide: %w", err)
	}
	glossary, err := loadGlossary(e.Project, e.Config.Translation.Glossary)
	if err != nil {
		return Report{}, err
	}

	type translationJob struct {
		index      int
		language   string
		sourceFile string
	}
	var jobs []translationJob
	for _, language := range languages {
		for _, sourceFile := range files {
			jobs = append(jobs, translationJob{index: len(jobs), language: language, sourceFile: sourceFile})
		}
	}
	requestWorkers := options.Workers
	if requestWorkers <= 0 {
		requestWorkers = 1
	}
	workers := requestWorkers
	workers = min(workers, len(jobs))
	type translationResult struct {
		index  int
		report FileReport
		err    error
	}
	jobChannel := make(chan translationJob)
	resultChannel := make(chan translationResult, len(jobs))
	workerContext, cancel := context.WithCancel(ctx)
	defer cancel()
	requestSlots := make(chan struct{}, requestWorkers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for job := range jobChannel {
				var fileReport FileReport
				var jobErr error
				for attempt := 0; attempt < 3; attempt++ {
					fileReport, jobErr = e.translateFile(workerContext, job.language, job.sourceFile, options, styleGuide, glossary[job.language], requestSlots)
					var structural *structuralTranslationFailure
					if jobErr == nil || !errors.As(jobErr, &structural) || attempt == 2 {
						break
					}
				}
				if jobErr != nil {
					jobErr = fmt.Errorf("translate %s to %s: %w", job.sourceFile, job.language, jobErr)
				}
				resultChannel <- translationResult{index: job.index, report: fileReport, err: jobErr}
				if jobErr != nil {
					cancel()
				}
			}
		}()
	}
	go func() {
		defer close(jobChannel)
		for _, job := range jobs {
			select {
			case jobChannel <- job:
			case <-workerContext.Done():
				return
			}
		}
	}()
	go func() { group.Wait(); close(resultChannel) }()
	ordered := make([]translationResult, len(jobs))
	received := 0
	var firstErr error
	for result := range resultChannel {
		ordered[result.index] = result
		received++
		if firstErr == nil && result.err != nil {
			firstErr = result.err
		}
	}
	var report Report
	for _, result := range ordered {
		if result.report.SourceFile == "" {
			continue
		}
		report.Files = append(report.Files, result.report)
		report.Segments += result.report.Segments
		report.Pending += result.report.NeedsTranslation
		if result.report.Written {
			report.Written++
		}
		report.Estimate.add(result.report.Estimate)
	}
	if firstErr != nil {
		return report, firstErr
	}
	if received != len(jobs) {
		return report, workerContext.Err()
	}
	return report, nil
}

func (e *Engine) targetLanguages(requested string) ([]string, error) {
	if requested != "" {
		if requested == e.Config.Site.DefaultLanguage {
			return nil, errors.New("target language must differ from the default language")
		}
		for _, language := range e.Config.Site.Languages {
			if language == requested {
				return []string{requested}, nil
			}
		}
		return nil, fmt.Errorf("target language %q is not configured in site.languages", requested)
	}
	var result []string
	for _, language := range e.Config.Site.Languages {
		if language != e.Config.Site.DefaultLanguage {
			result = append(result, language)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("no target languages are configured")
	}
	return result, nil
}

func (e *Engine) sourceFiles(requested string) ([]string, error) {
	discovered, err := content.Discover(e.Config.ContentPath(e.Project), e.Config.Site.Languages, e.Config.Site.DefaultLanguage)
	if err != nil {
		return nil, err
	}
	files := discovered[e.Config.Site.DefaultLanguage]
	navFile := filepath.ToSlash(e.Config.Build.NavFile)
	if _, statErr := os.Stat(filepath.Join(e.Config.ContentPath(e.Project), filepath.FromSlash(navFile))); statErr == nil {
		files = append(files, navFile)
	}
	if requested == "" {
		sort.Strings(files)
		return files, nil
	}
	wanted := filepath.ToSlash(filepath.Clean(filepath.FromSlash(requested)))
	contentPrefix := strings.TrimSuffix(filepath.ToSlash(e.Config.Build.ContentDir), "/") + "/"
	wanted = strings.TrimPrefix(wanted, contentPrefix)
	for _, language := range e.Config.Site.Languages {
		wanted = strings.TrimPrefix(wanted, language+"/")
	}
	for _, file := range files {
		if filepath.ToSlash(file) == wanted {
			return []string{file}, nil
		}
	}
	return nil, fmt.Errorf("source page %q was not found", requested)
}

type matchedSegment struct {
	Segment
	OldID      string
	Old        SegmentState
	TargetText string
	Target     Segment
	State      string
}

func (e *Engine) translateFile(ctx context.Context, language, sourceFile string, options Options, styleGuide string, glossary []GlossaryTerm, requestSlots chan struct{}) (FileReport, error) {
	contentDir := e.Config.ContentPath(e.Project)
	sourcePath := filepath.Join(contentDir, filepath.FromSlash(sourceFile))
	targetPath := filepath.Join(contentDir, filepath.FromSlash(language), filepath.FromSlash(sourceFile))
	stateFile, err := statePath(e.Project, e.Config.Translation.StateDir, language, sourceFile)
	if err != nil {
		return FileReport{}, err
	}
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return FileReport{}, err
	}
	sourceDoc, err := extractFile(sourceFile, e.Config.Build.NavFile, sourceBytes)
	if err != nil {
		return FileReport{}, err
	}
	statePathToLoad := stateFile
	if sourceDoc.TranslationKey != "" {
		if matchedPath, findErr := findStateByPageKey(filepath.Join(e.Project, filepath.FromSlash(e.Config.Translation.StateDir), language), sourceDoc.TranslationKey); findErr != nil {
			return FileReport{}, findErr
		} else if matchedPath != "" {
			statePathToLoad = matchedPath
		}
	}
	state, err := loadState(statePathToLoad, sourceFile, e.Config.Site.DefaultLanguage, language, sourceDoc.TranslationKey)
	if err != nil {
		return FileReport{}, err
	}

	targetReadPath := targetPath
	if state.SourceFile != "" && filepath.ToSlash(state.SourceFile) != filepath.ToSlash(sourceFile) {
		if _, statErr := os.Stat(targetPath); errors.Is(statErr, os.ErrNotExist) {
			targetReadPath = filepath.Join(contentDir, filepath.FromSlash(language), filepath.FromSlash(state.SourceFile))
		}
	}
	report := FileReport{SourceFile: filepath.ToSlash(sourceFile), TargetFile: filepath.ToSlash(filepath.Join(language, sourceFile)), TargetLanguage: language, Segments: len(sourceDoc.Segments), States: map[string]int{}}
	targetDoc, err := e.readTranslationTarget(sourceFile, targetReadPath, sourceBytes, state, options)
	if options.DryRun && (errors.Is(err, errUntrackedTranslation) || errors.Is(err, errTranslationMigration) || errors.Is(err, errMigrationReview)) {
		stateName := "untracked"
		if errors.Is(err, errTranslationMigration) {
			stateName = "migration-required"
		}
		if errors.Is(err, errMigrationReview) {
			for _, entry := range state.Segments {
				if entry.Status == "migration-review" {
					report.States["migration-review"]++
				}
			}
		} else {
			report.States[stateName] = report.Segments
		}
		return report, nil
	}
	if err != nil {
		return FileReport{}, err
	}
	targetByID := map[string]Segment{}
	if targetDoc != nil {
		for _, segment := range targetDoc.Segments {
			targetByID[segment.ID] = segment
		}
	}

	matches := matchSegments(sourceDoc.Segments, state.Segments)
	layout := inferTargetLayout(matches, state.Segments, targetByID)
	items := make([]matchedSegment, 0, len(sourceDoc.Segments))
	for _, segment := range sourceDoc.Segments {
		oldID := matches[segment.ID]
		old, exists := state.Segments[oldID]
		target, targetErr := reusableTarget(segment.ID, oldID, old, targetByID, layout)
		if targetErr != nil {
			if !options.Force || options.Scope != "all" {
				return FileReport{}, targetErr
			}
			target = targetByID[segment.ID]
		}
		targetText := ""
		if exists {
			targetText = target.Original
			if targetText == "" {
				targetText = old.MachineText
			}
		}
		status := segmentStatus(segment, old, exists, targetText)
		if segment.Protected {
			target = segment
			targetText = segment.Original
			status = "protected"
		}
		report.States[status]++
		items = append(items, matchedSegment{Segment: segment, OldID: oldID, Old: old, TargetText: targetText, Target: target, State: status})
	}

	var pending []RequestSegment
	for _, item := range items {
		if shouldTranslate(item.State, options.Scope, options.Force) {
			pending = append(pending, RequestSegment{ID: item.ID, Kind: item.Kind, Section: item.Section, Text: item.Text})
		}
	}
	report.NeedsTranslation = len(pending)
	report.Estimate = estimateUsage(e.Config, sourceDoc, language, pending, styleGuide, glossary)
	if options.DryRun {
		return report, nil
	}
	needsRelocation := targetReadPath != targetPath || statePathToLoad != stateFile
	if len(pending) == 0 && !needsRelocation {
		return report, nil
	}
	if len(pending) > 0 && e.Provider == nil {
		return report, errors.New("translation provider is not configured")
	}

	batchSize := 3_000
	if e.Provider != nil && (e.Provider.Name() == "codex" || e.Provider.Name() == "claude") {
		batchSize = 12_000
	}
	batches := segmentBatches(pending, batchSize)
	batchResults := make([]map[string]string, len(batches))
	batchErrors := make([]error, len(batches))
	var batchGroup sync.WaitGroup
	for index, batch := range batches {
		index, batch := index, batch
		batchGroup.Add(1)
		go func() {
			defer batchGroup.Done()
			select {
			case requestSlots <- struct{}{}:
				defer func() { <-requestSlots }()
			case <-ctx.Done():
				batchErrors[index] = ctx.Err()
				return
			}
			before, after := surroundingContext(sourceDoc.Segments, batch)
			batchResults[index], batchErrors[index] = e.translateBatch(ctx, TranslationRequest{
				Format: sourceDoc.Format, SourceLanguage: e.Config.Translation.SourceLanguage, TargetLanguage: language,
				LocaleGuidance: localeGuidance(language),
				PageTitle:      sourceDoc.Title, Outline: sourceDoc.Outline, StyleGuide: styleGuide,
				Glossary: glossary, PreviousContext: before, FollowingContext: after, Segments: batch,
			})
		}()
	}
	batchGroup.Wait()
	translated := map[string]string{}
	for index, batchErr := range batchErrors {
		if batchErr != nil {
			return report, fmt.Errorf("translation batch %d of %d: %w", index+1, len(batches), batchErr)
		}
		for id, value := range batchResults[index] {
			translated[id] = value
		}
	}
	values := map[string]string{}
	newState := newFileState(sourceFile, e.Config.Translation.SourceLanguage, language, sourceDoc.TranslationKey)
	newState.Migration = state.Migration
	for itemIndex, item := range items {
		value, newlyTranslated := translated[item.ID]
		status := item.State
		machineText := item.Old.MachineText
		machineHash := item.Old.MachineHash
		if newlyTranslated {
			restored, restoreErr := restore(item.Segment, value, preservesMarkdownSyntax(sourceDoc.Format))
			if restoreErr != nil {
				return report, restoreErr
			}
			if glossaryErr := validateGlossary(item.Original, restored, glossary); glossaryErr != nil {
				return report, fmt.Errorf("segment %s: %w", item.ID, glossaryErr)
			}
			machineText = restored
			machineHash = Hash(restored)
			status = "machine-translated"
			report.Translated++
		} else if item.TargetText != "" {
			target := item.Target
			target.Original = item.TargetText
			value, err = prepareExistingForFormat(sourceDoc.Format, &item.Segment, target)
			sourceDoc.Segments[itemIndex] = item.Segment
			if err != nil {
				return report, err
			}
		} else {
			return report, fmt.Errorf("segment %s has no reusable target text", item.ID)
		}
		values[item.ID] = value
		restored, restoreErr := restore(item.Segment, value, preservesMarkdownSyntax(sourceDoc.Format))
		if restoreErr != nil {
			return report, restoreErr
		}
		providerName, providerModel := item.Old.Provider, item.Old.Model
		if newlyTranslated && e.Provider != nil {
			providerName, providerModel = e.Provider.Name(), providerModelForLanguage(e.Provider, language)
		}
		newState.Segments[item.ID] = SegmentState{
			SourceHash: item.SourceHash, TargetHash: Hash(restored), MachineHash: machineHash,
			MachineText: machineText, Status: status, SourcePreview: preview(item.Original),
			Provider: providerName, Model: providerModel, PromptVersion: item.Old.PromptVersion, UpdatedAt: item.Old.UpdatedAt,
		}
		if !newlyTranslated && !item.Protected {
			entry := newState.Segments[item.ID]
			entry.SourceHash = item.Old.SourceHash
			entry.SourcePreview = item.Old.SourcePreview
			newState.Segments[item.ID] = entry
		}
		if newlyTranslated {
			entry := newState.Segments[item.ID]
			entry.PromptVersion = promptVersion
			entry.UpdatedAt = nowString()
			newState.Segments[item.ID] = entry
		}
	}
	output, err := Apply(sourceDoc, values)
	if err != nil {
		return report, err
	}
	if err := validateFile(sourceFile, e.Config.Build.NavFile, sourceDoc, output); err != nil {
		return report, err
	}
	if filepath.ToSlash(sourceFile) != filepath.ToSlash(e.Config.Build.NavFile) {
		output, err = rewriteLocalFragments(sourceFile, e.Config.Site.DefaultLanguage, language, sourceBytes, output)
		if err != nil {
			return report, err
		}
	}
	if err := writeAtomic(targetPath, output); err != nil {
		return report, err
	}
	if err := saveState(stateFile, newState); err != nil {
		return report, err
	}
	if needsRelocation && state.SourceFile != "" && filepath.ToSlash(state.SourceFile) != filepath.ToSlash(sourceFile) {
		oldSource := filepath.Join(contentDir, filepath.FromSlash(state.SourceFile))
		if _, statErr := os.Stat(oldSource); errors.Is(statErr, os.ErrNotExist) {
			if statePathToLoad != stateFile {
				_ = os.Remove(statePathToLoad)
			}
			if targetReadPath != targetPath {
				_ = os.Remove(targetReadPath)
			}
		}
	}
	report.Written = true
	return report, nil
}

var errUntrackedTranslation = errors.New("existing text has no translation state")

// readTranslationTarget checks the on-disk target before any provider request.
func (e *Engine) readTranslationTarget(sourceFile, targetPath string, source []byte, state *FileState, options Options) (*Document, error) {
	target, err := os.ReadFile(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		// A sidecar is not a translation. Cached machine text must not hide a
		// deleted target or its lost manual edits from status and missing-only runs.
		state.Segments = map[string]SegmentState{}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	replaceAll := options.Force && options.Scope == "all"
	if replaceAll {
		state.Segments = map[string]SegmentState{}
	} else if err := checkStateForUpdate(state, sourceFile); err != nil {
		return nil, err
	}
	if len(target) > 0 && len(state.Segments) == 0 && !replaceAll {
		return nil, fmt.Errorf("%s: %w; restore its sidecar from Git, or use --scope all --force only to deliberately replace this translation", targetPath, errUntrackedTranslation)
	}
	document, err := extractTargetFile(sourceFile, e.Config.Build.NavFile, source, target)
	if errors.Is(err, errTranslationHeadingStructure) && (len(state.Segments) == 0 || replaceAll) {
		document, err = extractFile(sourceFile, e.Config.Build.NavFile, target)
	}
	if err != nil {
		return nil, fmt.Errorf("parse existing target: %w", err)
	}
	return document, nil
}

func estimateUsage(cfg config.Config, sourceDoc *Document, language string, pending []RequestSegment, styleGuide string, glossary []GlossaryTerm) UsageEstimate {
	if len(pending) == 0 {
		return UsageEstimate{}
	}
	batchSize := 3_000
	if cfg.Translation.Provider == "codex" || cfg.Translation.Provider == "claude" {
		batchSize = 12_000
	}
	batches := segmentBatches(pending, batchSize)
	estimate := UsageEstimate{Requests: len(batches)}
	for _, segment := range pending {
		estimate.SourceCharacters += len([]rune(segment.Text))
	}
	for _, batch := range batches {
		before, after := surroundingContext(sourceDoc.Segments, batch)
		request := TranslationRequest{
			Format: sourceDoc.Format, SourceLanguage: cfg.Translation.SourceLanguage, TargetLanguage: language, LocaleGuidance: localeGuidance(language),
			PageTitle: sourceDoc.Title, Outline: sourceDoc.Outline, StyleGuide: styleGuide,
			Glossary: glossary, PreviousContext: before, FollowingContext: after, Segments: batch,
		}
		payload, _ := json.Marshal(request)
		schema, _ := json.Marshal(translationSchema(batch))
		inputCharacters := len([]rune(translationSystemPrompt(sourceDoc.Format))) + len([]rune(string(payload))) + len([]rune(string(schema)))
		estimate.InputTokens += (inputCharacters + 3) / 4
		outputCharacters := 32
		for _, segment := range batch {
			outputCharacters += len([]rune(segment.Text)) + len([]rune(segment.ID)) + 24
		}
		// Translated CJK output commonly uses fewer characters but more tokens per
		// character than English. Dividing by two is intentionally conservative.
		estimate.OutputTokens += (outputCharacters + 1) / 2
	}
	if cfg.Translation.InputPrice > 0 || cfg.Translation.OutputPrice > 0 {
		estimate.PricingConfigured = true
		estimate.InputCostUSD = float64(estimate.InputTokens) * cfg.Translation.InputPrice / 1_000_000
		estimate.OutputCostUSD = float64(estimate.OutputTokens) * cfg.Translation.OutputPrice / 1_000_000
		estimate.TotalCostUSD = estimate.InputCostUSD + estimate.OutputCostUSD
	}
	return estimate
}

// translateBatch preserves normal provider batching, but isolates a bad segment
// when a model repeatedly contaminates one result with neighbouring text or
// request context. Each smaller request retains the complete page context.
func (e *Engine) translateBatch(ctx context.Context, request TranslationRequest) (map[string]string, error) {
	requestContext, cancel := context.WithTimeout(ctx, 4*time.Minute)
	result, err := e.Provider.Translate(requestContext, request)
	cancel()
	if err == nil || len(request.Segments) <= 1 || ctx.Err() != nil {
		return result, err
	}
	middle := len(request.Segments) / 2
	leftRequest, rightRequest := request, request
	leftRequest.Segments = request.Segments[:middle]
	rightRequest.Segments = request.Segments[middle:]
	left, leftErr := e.translateBatch(ctx, leftRequest)
	if leftErr != nil {
		return nil, leftErr
	}
	right, rightErr := e.translateBatch(ctx, rightRequest)
	if rightErr != nil {
		return nil, rightErr
	}
	for id, value := range right {
		left[id] = value
	}
	return left, nil
}

func rewriteLocalFragments(sourceFile, sourceLanguage, targetLanguage string, source, target []byte) ([]byte, error) {
	if strings.EqualFold(filepath.Ext(sourceFile), ".mpd") {
		return rewriteMPDLocalFragments(sourceFile, source, target)
	}
	return rewriteMarkdownFragments(sourceFile, source, target, false)
}

type nativeHeading struct {
	level int
	id    string
}

// rewriteMPDLocalFragments intentionally stays on the native MPD tree. It
// must not route MPD through the Markdown renderer merely to discover heading
// anchors during translation.
func rewriteMPDLocalFragments(filename string, source, target []byte) ([]byte, error) {
	return rewriteMPDFragments(filename, source, target, false)
}

var errTranslationHeadingStructure = errors.New("heading structure changed; regenerate this page with --scope all --force after preserving any manual corrections")

func rewriteMPDFragments(filename string, source, target []byte, reverse bool) ([]byte, error) {
	if reverse && !bytes.Contains(target, []byte("](#")) {
		return target, nil
	}
	sourceHeadings, err := nativeMPDHeadings(filename, source)
	if err != nil {
		return nil, fmt.Errorf("parse source page anchors: %w", err)
	}
	targetHeadings, err := nativeMPDHeadings(filename, target)
	if err != nil {
		return nil, fmt.Errorf("parse translated page anchors: %w", err)
	}
	if len(sourceHeadings) != len(targetHeadings) {
		if reverse {
			return nil, errTranslationHeadingStructure
		}
		return nil, errors.New("translation changed the rendered heading structure")
	}
	aliases := map[string]string{}
	for index, sourceHeading := range sourceHeadings {
		targetHeading := targetHeadings[index]
		if sourceHeading.level != targetHeading.level {
			if reverse {
				return nil, errTranslationHeadingStructure
			}
			return nil, errors.New("translation changed a rendered heading level")
		}
		if sourceHeading.id == "" || targetHeading.id == "" || sourceHeading.id == targetHeading.id {
			continue
		}
		from, to := sourceHeading.id, targetHeading.id
		if reverse {
			from, to = to, from
		}
		aliases["#"+from] = "#" + to
	}
	return patchMPDFragments(filename, target, aliases), nil
}

func patchMPDFragments(filename string, source []byte, aliases map[string]string) []byte {
	document := mpd.Parse(filename, source)
	var links []mpd.Range
	for _, node := range document.Nodes {
		if node.Kind == mpd.KindLink && node.Flags&mpd.FlagReference == 0 {
			links = append(links, node.Content)
		}
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Start > links[j].Start })
	result := append([]byte(nil), source...)
	for _, link := range links {
		if value, ok := aliases[string(document.Text(link))]; ok {
			result = append(result[:link.Start], append([]byte(value), result[link.End:]...)...)
		}
	}
	return result
}

// Translation state stores source anchor spellings. Normalize only generated
// local anchors when reusing prose; executable examples never enter this map.
func extractTargetFile(file, navFile string, source, target []byte) (*Document, error) {
	if strings.EqualFold(filepath.Ext(file), ".mpd") {
		var err error
		target, err = rewriteMPDFragments(file, source, target, true)
		if err != nil {
			return nil, err
		}
	}
	if !strings.EqualFold(filepath.Ext(file), ".mpd") && filepath.ToSlash(file) != filepath.ToSlash(navFile) {
		var err error
		target, err = rewriteMarkdownFragments(file, source, target, true)
		if err != nil {
			return nil, err
		}
	}
	return extractFile(file, navFile, target)
}

func nativeMPDHeadings(filename string, source []byte) ([]nativeHeading, error) {
	document := mpd.Parse(filename, source)
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			return nil, fmt.Errorf("line %d: %s", diagnostic.Position.Line, diagnostic.Message)
		}
	}
	used := map[string]bool{}
	result := make([]nativeHeading, 0, 16)
	for index := range document.Nodes {
		node := &document.Nodes[index]
		if node.Kind != mpd.KindHeading {
			continue
		}
		text := nativeMPDInlineText(document, uint32(index))
		base := nativeHeadingID(text)
		id := base
		for suffix := 1; used[id]; suffix++ {
			id = base + "-" + strconv.Itoa(suffix)
		}
		used[id] = true
		result = append(result, nativeHeading{level: int(node.Level), id: id})
	}
	return result, nil
}

func nativeMPDInlineText(document *mpd.Document, parent uint32) string {
	var output strings.Builder
	iterator := document.Children(parent)
	for {
		index, node, ok := iterator.Next()
		if !ok {
			break
		}
		switch node.Kind {
		case mpd.KindText, mpd.KindCodeSpan, mpd.KindAutomaticLink, mpd.KindEmoji:
			output.Write(document.Text(node.Content))
		case mpd.KindSoftBreak, mpd.KindHardBreak:
			output.WriteByte(' ')
		default:
			output.WriteString(nativeMPDInlineText(document, index))
		}
	}
	return output.String()
}

// nativeHeadingID matches the ASCII heading-ID rules used by generated sites:
// lowercase letters and digits survive, while spaces, hyphens, and underscores
// become hyphens. Non-ASCII text falls back to "heading".
func nativeHeadingID(value string) string {
	value = strings.TrimSpace(value)
	var output strings.Builder
	for _, character := range value {
		switch {
		case character >= 'A' && character <= 'Z':
			output.WriteRune(character + ('a' - 'A'))
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			output.WriteRune(character)
		case character == ' ' || character == '\t' || character == '\n' || character == '-' || character == '_':
			output.WriteByte('-')
		}
	}
	if output.Len() == 0 {
		return "heading"
	}
	return output.String()
}

func validateGlossary(source, target string, glossary []GlossaryTerm) error {
	for _, term := range glossary {
		if term.Source == "" || term.Translation == "" {
			continue
		}
		if strings.Contains(source, term.Source) && !strings.Contains(target, term.Translation) {
			return fmt.Errorf("required glossary translation %q is missing", term.Translation)
		}
	}
	return nil
}

func segmentBatches(segments []RequestSegment, maxCharacters int) [][]RequestSegment {
	if maxCharacters <= 0 {
		maxCharacters = 18_000
	}
	var batches [][]RequestSegment
	var batch []RequestSegment
	size := 0
	for _, segment := range segments {
		segmentSize := len(segment.Text) + len(segment.Section) + 80
		if len(batch) > 0 && size+segmentSize > maxCharacters {
			batches = append(batches, batch)
			batch = nil
			size = 0
		}
		batch = append(batch, segment)
		size += segmentSize
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}

func surroundingContext(all []Segment, batch []RequestSegment) (string, string) {
	if len(batch) == 0 {
		return "", ""
	}
	first, last := -1, -1
	for index, segment := range all {
		if segment.ID == batch[0].ID {
			first = index
		}
		if segment.ID == batch[len(batch)-1].ID {
			last = index
		}
	}
	var before, after []string
	for index := max(0, first-3); first >= 0 && index < first; index++ {
		before = append(before, all[index].Original)
	}
	for index := last + 1; last >= 0 && index < len(all) && index <= last+3; index++ {
		after = append(after, all[index].Original)
	}
	return strings.Join(before, " "), strings.Join(after, " ")
}

func extractFile(sourceFile, navFile string, data []byte) (*Document, error) {
	if filepath.ToSlash(sourceFile) == filepath.ToSlash(navFile) {
		return ExtractNavigation(data)
	}
	if strings.EqualFold(filepath.Ext(sourceFile), ".mpd") {
		return ExtractMPD(sourceFile, data)
	}
	return ExtractMarkdown(data)
}

func validateFile(sourceFile, navFile string, source *Document, target []byte) error {
	if filepath.ToSlash(sourceFile) == filepath.ToSlash(navFile) {
		targetDoc, err := ExtractNavigation(target)
		if err != nil {
			return err
		}
		if immutableSkeleton(source) != immutableSkeleton(targetDoc) {
			return errors.New("translation changed navigation structure or links")
		}
		return nil
	}
	return Validate(source, target)
}

type targetLayout uint8

const (
	targetAtCurrent targetLayout = 1 << iota
	targetAtOld
)

// Unchanged translations provide anchors for manually edited neighbours.
// Mixed or absent evidence must not silently select a different paragraph.
func inferTargetLayout(matches map[string]string, old map[string]SegmentState, targets map[string]Segment) targetLayout {
	var layout targetLayout
	for currentID, oldID := range matches {
		if currentID == oldID || old[oldID].TargetHash == "" {
			continue
		}
		if target, ok := targets[currentID]; ok && Hash(target.Original) == old[oldID].TargetHash {
			layout |= targetAtCurrent
		}
		if target, ok := targets[oldID]; ok && Hash(target.Original) == old[oldID].TargetHash {
			layout |= targetAtOld
		}
	}
	return layout
}

func reusableTarget(currentID, oldID string, old SegmentState, targets map[string]Segment, layout targetLayout) (Segment, error) {
	if currentID == oldID || oldID == "" || len(targets) == 0 {
		return targets[oldID], nil
	}
	if current, ok := targets[currentID]; ok && Hash(current.Original) == old.TargetHash {
		return current, nil
	}
	if previous, ok := targets[oldID]; ok && Hash(previous.Original) == old.TargetHash {
		return previous, nil
	}
	switch layout {
	case targetAtCurrent:
		return targets[currentID], nil
	case targetAtOld:
		return targets[oldID], nil
	default:
		return Segment{}, fmt.Errorf("segment %s moved and its edited target position is ambiguous; reconcile the translation or use --scope all --force to regenerate it", currentID)
	}
}

func matchSegments(current []Segment, old map[string]SegmentState) map[string]string {
	result, used := matchExactSegments(current, old)
	byHash := indexMovableSegments(old)

	for _, segment := range current {
		if result[segment.ID] != "" || segment.D2Key != "" {
			continue
		}
		for _, id := range byHash[segment.SourceHash] {
			if !used[id] {
				result[segment.ID] = id
				used[id] = true
				break
			}
		}
	}
	for _, segment := range current {
		if result[segment.ID] != "" || used[segment.ID] {
			continue
		}
		if _, ok := old[segment.ID]; ok {
			result[segment.ID] = segment.ID
			used[segment.ID] = true
		}
	}
	return result
}

// Exact identities take precedence over identical text elsewhere. Diagram
// identities encode graph keys and must never move between nodes or edges.
func matchExactSegments(current []Segment, old map[string]SegmentState) (map[string]string, map[string]bool) {
	result := map[string]string{}
	used := map[string]bool{}
	for _, segment := range current {
		if state, ok := old[segment.ID]; ok && (segment.D2Key != "" || state.SourceHash == segment.SourceHash) {
			result[segment.ID] = segment.ID
			used[segment.ID] = true
		}
	}
	return result, used
}

func indexMovableSegments(old map[string]SegmentState) map[string][]string {
	byHash := map[string][]string{}
	for id, state := range old {
		if strings.HasPrefix(id, "d") && strings.Contains(id, "-label-") {
			continue
		}
		byHash[state.SourceHash] = append(byHash[state.SourceHash], id)
	}
	return byHash
}

func segmentStatus(segment Segment, old SegmentState, exists bool, targetText string) string {
	if !exists {
		return "missing"
	}
	sourceChanged := old.SourceHash != segment.SourceHash
	targetChanged := targetText != "" && old.TargetHash != "" && Hash(targetText) != old.TargetHash
	manualTarget := old.Status == "manual" || old.MachineHash != "" && old.TargetHash != old.MachineHash
	switch {
	case sourceChanged && (targetChanged || manualTarget):
		return "conflict"
	case sourceChanged:
		return "stale"
	case old.Status == "migration-review":
		return "migration-review"
	case targetChanged:
		return "manual"
	case old.Status == "manual" || old.Status == "reviewed" || old.Status == "final":
		return old.Status
	default:
		return "machine-translated"
	}
}

func shouldTranslate(status, scope string, force bool) bool {
	if status == "conflict" {
		return force
	}
	if scope == "all" {
		return force || status == "missing" || status == "stale" || status == "machine-translated"
	}
	if scope == "missing" {
		return status == "missing"
	}
	return status == "missing" || status == "stale"
}

type existingPlaceholder struct {
	placeholder string
	position    int
}

func prepareExisting(segment Segment, target string) (string, error) {
	return prepareExistingRanges(segment, target, nil)
}

func prepareExistingRanges(segment Segment, target string, ranges []mpdProtectedRange) (string, error) {
	if len(segment.Placeholders) == 0 {
		return target, nil
	}
	grouped := map[string][]existingPlaceholder{}
	for placeholder, original := range segment.Placeholders {
		grouped[original] = append(grouped[original], existingPlaceholder{placeholder: placeholder, position: strings.Index(segment.Text, placeholder)})
	}
	originals := make([]string, 0, len(grouped))
	for original := range grouped {
		originals = append(originals, original)
	}
	sort.Slice(originals, func(i, j int) bool { return len(originals[i]) > len(originals[j]) })
	patterns := make([]string, 0, len(originals))
	for _, original := range originals {
		replacements := grouped[original]
		sort.Slice(replacements, func(i, j int) bool { return replacements[i].position < replacements[j].position })
		pattern := regexp.QuoteMeta(original)
		// The numeric protector uses a leading word boundary. A version digit in an
		// unprotected name such as v2 must not consume a standalone 2 token.
		if ranges == nil && (original[0] >= '0' && original[0] <= '9' || len(original) > 1 && (original[0] == 'v' || original[0] == 'V') && original[1] >= '0' && original[1] <= '9') {
			pattern = `\b` + pattern
		}
		patterns = append(patterns, pattern)
	}
	// Match the original target once: inserted placeholder indices must never
	// become matches for a later protected number.
	value, counts := replaceExistingPlaceholders(target, regexp.MustCompile(strings.Join(patterns, "|")), grouped, ranges)
	for original, replacements := range grouped {
		if counts[original] != len(replacements) {
			return "", fmt.Errorf("existing translation for %s does not preserve %s", segment.ID, original)
		}
	}
	return value, nil
}

func replaceExistingPlaceholders(target string, pattern *regexp.Regexp, grouped map[string][]existingPlaceholder, ranges []mpdProtectedRange) (string, map[string]int) {
	var output strings.Builder
	counts := map[string]int{}
	cursor := 0
	for _, span := range pattern.FindAllStringIndex(target, -1) {
		original := target[span[0]:span[1]]
		if !withinProtectedRanges(span[0], span[1], ranges) || ranges == nil && partialQuantityMatch(target, span[0], span[1]) {
			continue
		}
		output.WriteString(target[cursor:span[0]])
		cursor = span[1]
		index := counts[original]
		// Translating a sentence can move another @mention to a line start,
		// where the MPD writer must introduce an additional escape.
		if index < len(grouped[original]) || original != `\@` || !lineLeadingEscape(target, span[0]) {
			counts[original]++
		}
		if index < len(grouped[original]) {
			output.WriteString(grouped[original][index].placeholder)
		} else {
			output.WriteString(original)
		}
	}
	output.WriteString(target[cursor:])
	return output.String(), counts
}

func lineLeadingEscape(text string, start int) bool {
	lineStart := strings.LastIndexByte(text[:start], '\n') + 1
	return strings.TrimSpace(text[lineStart:start]) == ""
}

// A unit suffix is allowed, but a fragment of a different decimal, range,
// version, or integer must not satisfy a protected quantity.
func partialQuantityMatch(text string, start, end int) bool {
	digit := func(value byte) bool { return value >= '0' && value <= '9' }
	separator := func(value byte) bool { return strings.ContainsRune(".,:/-", rune(value)) }
	if digit(text[start]) && start >= 2 && separator(text[start-1]) && digit(text[start-2]) {
		return true
	}
	if !digit(text[end-1]) || end == len(text) {
		return false
	}
	return digit(text[end]) || end+1 < len(text) && separator(text[end]) && digit(text[end+1])
}

func Validate(source *Document, target []byte) error {
	// Reused segments may retain equivalent target token spellings. Validation
	// is always grounded in the original source bytes, not those mutable maps.
	if source.Format == markdownTranslationFormat {
		original, err := ExtractMarkdown(source.Source)
		if err != nil {
			return err
		}
		source = original
	}
	if err := validateTranslationControls(string(target)); err != nil {
		return err
	}
	var targetDoc *Document
	var err error
	if source.Format == "mpd" {
		targetDoc, err = extractTargetFile("translated.mpd", "_nav.yaml", source.Source, target)
	} else if source.Format == markdownTranslationFormat {
		targetDoc, err = extractTargetFile("translated.md", "_nav.yaml", source.Source, target)
	} else {
		targetDoc, err = Extract(target)
	}
	if err != nil {
		return fmt.Errorf("parse translated %s: %w", strings.ToUpper(source.Format), err)
	}
	sourceSkeleton := immutableSkeleton(source)
	targetSkeleton := immutableSkeleton(targetDoc)
	if sourceSkeleton != targetSkeleton && source.Format == "mpd" {
		targetDoc, err = extractTargetFile("translated.mpd", "_nav.yaml", source.Source, target)
		if err != nil {
			return err
		}
		targetSkeleton = immutableSkeleton(targetDoc)
	}
	if sourceSkeleton != targetSkeleton {
		return structuralTranslationError(sourceSkeleton, targetSkeleton, source.Format)
	}
	if source.Format == "mpd" || source.Format == markdownTranslationFormat {
		return validateMPDInlineProtection(source, targetDoc)
	}
	return nil
}

func structuralTranslationError(source, target, format string) error {
	label := "Markdown"
	if format == "mpd" {
		label = "MPD"
	}
	sourceLines := strings.Split(source, "\n")
	targetLines := strings.Split(target, "\n")
	limit := min(len(sourceLines), len(targetLines))
	line := 0
	for line < limit && sourceLines[line] == targetLines[line] {
		line++
	}
	if line == limit && len(sourceLines) == len(targetLines) {
		return &structuralTranslationFailure{message: "translation changed " + label + " structure, code, links or component syntax"}
	}
	sourceLine, targetLine := "<end of document>", "<end of document>"
	if line < len(sourceLines) {
		sourceLine = sourceLines[line]
	}
	if line < len(targetLines) {
		targetLine = targetLines[line]
	}
	column := 0
	for column < len(sourceLine) && column < len(targetLine) && sourceLine[column] == targetLine[column] {
		column++
	}
	contextStart := max(0, column-32)
	return &structuralTranslationFailure{message: fmt.Sprintf("translation changed %s structure, code, links or component syntax at line %d, byte %d: source %q, target %q", label, line+1, column+1, preview(sourceLine[contextStart:]), preview(targetLine[contextStart:]))}
}

type structuralTranslationFailure struct{ message string }

func (e *structuralTranslationFailure) Error() string { return e.message }

func immutableSkeleton(document *Document) string {
	values := map[string]string{}
	for _, segment := range document.Segments {
		values[segment.ID] = "⟪TEXT_" + segment.ID + "⟫"
	}
	data, err := applyRaw(document, values)
	if err != nil {
		return ""
	}
	skeleton := string(data)
	if document.Format == "mpd" {
		skeleton = mpdSkeletonText.ReplaceAllString(skeleton, "⟪TEXT⟫")
	}
	return skeleton
}

var mpdSkeletonText = regexp.MustCompile(`(?:⟪TEXT_[^⟫]+⟫)+`)

func applyRaw(document *Document, values map[string]string) ([]byte, error) {
	return applyDocumentValues(document, values, false)
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".mpress-translate-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return err
	}
	return os.Rename(name, path)
}

type glossaryFile struct {
	Terms []struct {
		Source       string            `yaml:"source"`
		Translations map[string]string `yaml:"translations"`
		Note         string            `yaml:"note"`
	} `yaml:"terms"`
}

func loadGlossary(root, name string) (map[string][]GlossaryTerm, error) {
	result := map[string][]GlossaryTerm{}
	if strings.TrimSpace(name) == "" {
		return result, nil
	}
	data, err := readProjectFile(root, name, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("read translation glossary: %w", err)
	}
	var file glossaryFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse translation glossary: %w", err)
	}
	for _, term := range file.Terms {
		for language, translation := range term.Translations {
			result[language] = append(result[language], GlossaryTerm{Source: term.Source, Translation: translation, Note: term.Note})
		}
	}
	return result, nil
}

func readOptionalProjectFile(root, name string, limit int64) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil
	}
	data, err := readProjectFile(root, name, limit)
	return string(data), err
}

func readProjectFile(root, name string, limit int64) ([]byte, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, errors.New("path must stay inside the project")
	}
	data, err := os.ReadFile(filepath.Join(root, clean))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return data, nil
}

func MarshalReport(report Report) string {
	data, _ := json.MarshalIndent(report, "", "  ")
	return string(data)
}
