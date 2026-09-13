package dev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/translate"
)

type translationModel struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Description         string   `json:"description,omitempty"`
	ContextLength       int      `json:"contextLength,omitempty"`
	PromptPrice         float64  `json:"promptPrice"`
	CompletionPrice     float64  `json:"completionPrice"`
	InternalReasonPrice float64  `json:"internalReasonPrice,omitempty"`
	ReasoningMandatory  bool     `json:"reasoningMandatory,omitempty"`
	ReasoningEfforts    []string `json:"reasoningEfforts,omitempty"`
	DefaultReasoning    bool     `json:"defaultReasoning,omitempty"`
	Recommended         bool     `json:"recommended,omitempty"`
	Recommendation      string   `json:"recommendation,omitempty"`
	OneLanguageCost     float64  `json:"oneLanguageCost"`
	OneLanguageBudget   float64  `json:"oneLanguageBudget"`
	AllLanguagesCost    float64  `json:"allLanguagesCost"`
	AllLanguagesBudget  float64  `json:"allLanguagesBudget"`
	RemainingCost       float64  `json:"remainingCost"`
	RemainingBudget     float64  `json:"remainingBudget"`
	SampleCost          float64  `json:"sampleCost"`
}

type translationComparisonState struct {
	A        translate.ComparisonCandidate
	B        translate.ComparisonCandidate
	Language string
	Created  time.Time
}

type translationRecommendation struct {
	Language    string   `json:"language"`
	ModelIDs    []string `json:"modelIds"`
	Basis       string   `json:"basis"`
	ReviewFocus []string `json:"reviewFocus"`
}

var recommendedTranslationModels = map[string]string{
	"openai/gpt-5.4":               "OpenRouter translation-use leader and premium general candidate",
	"openai/gpt-5.4-mini":          "Balanced quality and throughput",
	"google/gemini-3.5-flash-lite": "Low-cost high-throughput candidate",
	"qwen/qwen3.7-plus":            "Value candidate for multilingual text",
	"qwen/qwen3.5-397b-a17b":       "High-capacity multilingual and CJK candidate",
	"mistralai/mistral-small-2603": "Efficient multilingual candidate",
	"mistralai/mistral-saba":       "Middle Eastern and South Asian specialist candidate",
	"anthropic/claude-sonnet-5":    "Premium quality candidate",
}

func (s *Server) handleTranslationModels(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Translation.Provider != "openrouter" {
		writeAPIError(w, http.StatusBadRequest, errors.New("model discovery and comparison currently require translation.provider: openrouter"))
		return
	}
	switch r.Method {
	case http.MethodGet:
		models, err := s.translationModels(r.Context(), false)
		if err != nil {
			writeAPIError(w, http.StatusBadGateway, err)
			return
		}
		estimate, err := translate.EstimateProject(s.project, s.cfg)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		priced := priceTranslationModels(models, estimate)
		recommendations := translationRecommendations(s.cfg.Site.Languages, s.cfg.Site.DefaultLanguage, priced)
		writeJSON(w, http.StatusOK, map[string]any{
			"models": priced, "estimate": estimate, "configuredModel": s.cfg.Translation.Model,
			"configuredReasoningEffort": s.cfg.Translation.ReasoningEffort,
			"configuredLanguageModels":  s.cfg.Translation.LanguageModels,
			"recommendations":           recommendations,
			"hasAPIKey":                 translate.ResolveAPIKey(s.cfg) != "",
			"apiKeyEnv":                 s.cfg.Translation.APIKeyEnv,
			"source":                    "OpenRouter live model catalog",
		})
	case http.MethodPost:
		if !s.canWrite(w, r) {
			return
		}
		var input struct {
			Action       string   `json:"action"`
			Models       []string `json:"models"`
			Language     string   `json:"language"`
			ComparisonID string   `json:"comparisonId"`
			Winner       string   `json:"winner"`
			Model        string   `json:"model"`
			File         string   `json:"file"`
		}
		if err := decodeJSON(r, &input); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		switch input.Action {
		case "compare":
			s.compareTranslationModels(w, r, input.Models, input.Language, input.File)
		case "judge":
			s.judgeTranslationModels(w, input.ComparisonID, input.Winner)
		case "set-model":
			s.setTranslationModel(w, r, input.Model, input.Language)
		default:
			writeAPIError(w, http.StatusBadRequest, errors.New("unknown translation model action"))
		}
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) translationModels(ctx context.Context, refresh bool) ([]translationModel, error) {
	s.modelCatalogMu.Lock()
	defer s.modelCatalogMu.Unlock()
	if !refresh && len(s.modelCatalog) > 0 && time.Since(s.modelCatalogAt) < 15*time.Minute {
		return append([]translationModel(nil), s.modelCatalog...), nil
	}
	base, err := url.Parse(strings.TrimRight(s.cfg.Translation.BaseURL, "/") + "/models")
	if err != nil {
		return nil, err
	}
	query := base.Query()
	query.Set("output_modalities", "text")
	query.Set("supported_parameters", "structured_outputs")
	query.Set("sort", "most-popular")
	base.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("load OpenRouter models: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("load OpenRouter models: %s", response.Status)
	}
	var payload struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			Description   string `json:"description"`
			ContextLength int    `json:"context_length"`
			Expiration    string `json:"expiration_date"`
			Pricing       struct {
				Prompt            string `json:"prompt"`
				Completion        string `json:"completion"`
				InternalReasoning string `json:"internal_reasoning"`
			} `json:"pricing"`
			Architecture struct {
				Output []string `json:"output_modalities"`
			} `json:"architecture"`
			Supported []string `json:"supported_parameters"`
			Reasoning struct {
				Mandatory       bool     `json:"mandatory"`
				DefaultEnabled  bool     `json:"default_enabled"`
				SupportedEffort []string `json:"supported_efforts"`
			} `json:"reasoning"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode OpenRouter models: %w", err)
	}
	models := make([]translationModel, 0, len(payload.Data))
	for _, item := range payload.Data {
		if item.ID == "" || strings.Contains(item.ID, ":batch") || strings.Contains(strings.ToLower(item.ID), "image") || !containsString(item.Supported, "structured_outputs") {
			continue
		}
		if len(item.Architecture.Output) > 0 && !containsString(item.Architecture.Output, "text") {
			continue
		}
		prompt, promptErr := strconv.ParseFloat(item.Pricing.Prompt, 64)
		completion, completionErr := strconv.ParseFloat(item.Pricing.Completion, 64)
		if promptErr != nil || completionErr != nil || prompt < 0 || completion < 0 {
			continue
		}
		internal, _ := strconv.ParseFloat(item.Pricing.InternalReasoning, 64)
		recommendation, recommended := recommendedTranslationModels[item.ID]
		models = append(models, translationModel{
			ID: item.ID, Name: item.Name, Description: compactDescription(item.Description), ContextLength: item.ContextLength,
			PromptPrice: prompt, CompletionPrice: completion, InternalReasonPrice: internal,
			ReasoningMandatory: item.Reasoning.Mandatory, ReasoningEfforts: item.Reasoning.SupportedEffort,
			DefaultReasoning: item.Reasoning.DefaultEnabled, Recommended: recommended, Recommendation: recommendation,
		})
	}
	if len(models) == 0 {
		return nil, errors.New("OpenRouter returned no text models with structured output support")
	}
	s.modelCatalog = models
	s.modelCatalogAt = time.Now()
	return append([]translationModel(nil), models...), nil
}

func priceTranslationModels(models []translationModel, estimate translate.ProjectEstimate) []translationModel {
	priced := append([]translationModel(nil), models...)
	for index := range priced {
		model := &priced[index]
		model.OneLanguageCost = estimatedProviderCost(estimate.InputTokens, estimate.OutputTokens, *model)
		model.OneLanguageBudget = roundedCost(model.OneLanguageCost * 3)
		model.AllLanguagesCost = model.OneLanguageCost * float64(estimate.TargetLanguages)
		model.AllLanguagesBudget = roundedCost(model.AllLanguagesCost * 3)
		model.RemainingCost = estimatedProviderCost(estimate.RemainingInputTokens, estimate.RemainingOutputTokens, *model)
		model.RemainingBudget = roundedCost(model.RemainingCost * 3)
		// A comparison uses one compact batch. The upper estimate includes one
		// retry so users see a conservative preflight amount.
		model.SampleCost = estimatedProviderCost(2400, 900, *model) * 2
	}
	sort.SliceStable(priced, func(i, j int) bool {
		if priced[i].Recommended != priced[j].Recommended {
			return priced[i].Recommended
		}
		if priced[i].Recommended && priced[i].Recommendation != priced[j].Recommendation {
			return priced[i].OneLanguageCost < priced[j].OneLanguageCost
		}
		return priced[i].Name < priced[j].Name
	})
	return priced
}

func estimatedProviderCost(inputTokens, outputTokens int, model translationModel) float64 {
	cost := float64(inputTokens)*model.PromptPrice + float64(outputTokens)*model.CompletionPrice
	if model.ReasoningMandatory {
		// Mandatory reasoning varies by provider. Include a visible contingency
		// rather than pretending that output-token use is exact.
		cost *= 1.25
	}
	return roundedCost(cost)
}

func roundedCost(cost float64) float64 {
	return math.Round(cost*1_000_000) / 1_000_000
}

func (s *Server) compareTranslationModels(w http.ResponseWriter, r *http.Request, ids []string, language, sourceFile string) {
	if len(ids) != 2 || ids[0] == ids[1] {
		writeAPIError(w, http.StatusBadRequest, errors.New("choose two different models"))
		return
	}
	models, err := s.translationModels(r.Context(), false)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	byID := map[string]translationModel{}
	for _, model := range models {
		byID[model.ID] = model
	}
	candidates := make([]translate.ComparisonCandidate, 2)
	for index, id := range ids {
		model, ok := byID[id]
		if !ok {
			writeAPIError(w, http.StatusBadRequest, fmt.Errorf("model %q is not available with structured outputs", id))
			return
		}
		candidates[index] = translate.ComparisonCandidate{Model: id, ReasoningEffort: comparisonReasoningEffort(model)}
	}
	comparison, err := translate.CompareProjectModelsForFile(r.Context(), s.project, s.cfg, language, sourceFile, candidates)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	if randomBool() {
		comparison.Outputs[0], comparison.Outputs[1] = comparison.Outputs[1], comparison.Outputs[0]
	}
	id, err := newToken()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	state := translationComparisonState{
		A:        translate.ComparisonCandidate{Model: comparison.Outputs[0].Model, ReasoningEffort: comparisonReasoningEffort(byID[comparison.Outputs[0].Model])},
		B:        translate.ComparisonCandidate{Model: comparison.Outputs[1].Model, ReasoningEffort: comparisonReasoningEffort(byID[comparison.Outputs[1].Model])},
		Language: comparison.TargetLanguage,
		Created:  time.Now(),
	}
	s.translationMu.Lock()
	for key, saved := range s.translationComparisons {
		if time.Since(saved.Created) > 30*time.Minute {
			delete(s.translationComparisons, key)
		}
	}
	s.translationComparisons[id] = state
	s.translationMu.Unlock()
	outputs := []map[string]any{
		{"label": "A", "texts": comparison.Outputs[0].Texts, "durationMs": comparison.Outputs[0].DurationMS},
		{"label": "B", "texts": comparison.Outputs[1].Texts, "durationMs": comparison.Outputs[1].DurationMS},
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"comparisonId": id, "sourceFile": comparison.SourceFile, "targetLanguage": comparison.TargetLanguage,
		"source": comparison.Source, "outputs": outputs, "estimatedInputTokens": comparison.EstimatedInputTokens,
		"estimatedOutputTokens": comparison.EstimatedOutputTokens,
	})
}

func (s *Server) judgeTranslationModels(w http.ResponseWriter, id, winner string) {
	if winner != "A" && winner != "B" {
		writeAPIError(w, http.StatusBadRequest, errors.New("winner must be A or B"))
		return
	}
	s.translationMu.Lock()
	comparison, ok := s.translationComparisons[id]
	if ok {
		delete(s.translationComparisons, id)
	}
	s.translationMu.Unlock()
	if !ok || time.Since(comparison.Created) > 30*time.Minute {
		writeAPIError(w, http.StatusNotFound, errors.New("the comparison expired; run another round"))
		return
	}
	winnerCandidate, loserCandidate := comparison.A, comparison.B
	if winner == "B" {
		winnerCandidate, loserCandidate = comparison.B, comparison.A
	}
	writeJSON(w, http.StatusOK, map[string]any{"winnerModel": winnerCandidate.Model, "winnerReasoningEffort": winnerCandidate.ReasoningEffort, "loserModel": loserCandidate.Model, "language": comparison.Language})
}

func (s *Server) setTranslationModel(w http.ResponseWriter, r *http.Request, id, targetLanguage string) {
	models, err := s.translationModels(r.Context(), false)
	if err != nil {
		writeAPIError(w, http.StatusBadGateway, err)
		return
	}
	var selected *translationModel
	for index := range models {
		if models[index].ID == id {
			selected = &models[index]
			break
		}
	}
	if selected == nil {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("model %q is not available with structured outputs", id))
		return
	}
	updated := s.cfg
	reasoningEffort := comparisonReasoningEffort(*selected)
	if targetLanguage == "" {
		updated.Translation.Model = selected.ID
		updated.Translation.ReasoningEffort = reasoningEffort
	} else {
		validTarget := false
		for _, language := range updated.Site.Languages {
			if language == targetLanguage && language != updated.Site.DefaultLanguage {
				validTarget = true
				break
			}
		}
		if !validTarget {
			writeAPIError(w, http.StatusBadRequest, fmt.Errorf("language %q is not a configured translation target", targetLanguage))
			return
		}
		if updated.Translation.LanguageModels == nil {
			updated.Translation.LanguageModels = map[string]config.TranslationLanguageModel{}
		}
		updated.Translation.LanguageModels[targetLanguage] = config.TranslationLanguageModel{Model: selected.ID, ReasoningEffort: reasoningEffort}
	}
	if err := updated.Validate(); err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	path := filepath.Join(s.project, config.Filename)
	current, err := os.ReadFile(path)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	data := current
	if targetLanguage == "" {
		data, err = setYAMLValue(data, []string{"translation", "model"}, updated.Translation.Model)
		if err == nil {
			data, err = setYAMLValue(data, []string{"translation", "reasoningEffort"}, updated.Translation.ReasoningEffort)
		}
	} else {
		data, err = setYAMLValue(data, []string{"translation", "languageModels"}, updated.Translation.LanguageModels)
	}
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.writeSource(path, data, revision(current)); err != nil {
		writeAPIError(w, http.StatusConflict, err)
		return
	}
	s.cfg = updated
	writeJSON(w, http.StatusOK, map[string]any{"model": selected.ID, "reasoningEffort": reasoningEffort, "language": targetLanguage, "languageModels": updated.Translation.LanguageModels, "state": s.currentState()})
}

func translationRecommendations(languages []string, defaultLanguage string, models []translationModel) map[string]translationRecommendation {
	available := map[string]bool{}
	for _, model := range models {
		available[model.ID] = true
	}
	result := map[string]translationRecommendation{}
	for _, target := range languages {
		if target == defaultLanguage {
			continue
		}
		ids, basis, focus := recommendedModelsForLanguage(target)
		filtered := make([]string, 0, 3)
		for _, id := range ids {
			if available[id] {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) < 2 {
			for _, fallback := range []string{"openai/gpt-5.4-mini", "google/gemini-3.5-flash-lite", "mistralai/mistral-small-2603", "anthropic/claude-sonnet-5"} {
				if available[fallback] && !containsString(filtered, fallback) {
					filtered = append(filtered, fallback)
				}
				if len(filtered) == 3 {
					break
				}
			}
		}
		result[target] = translationRecommendation{Language: target, ModelIDs: filtered, Basis: basis, ReviewFocus: focus}
	}
	return result
}

func recommendedModelsForLanguage(target string) ([]string, string, []string) {
	base := strings.ToLower(strings.SplitN(target, "-", 2)[0])
	switch base {
	case "zh", "ja", "ko":
		focus := []string{"technical terminology", "natural sentence structure", "preserved code and identifiers"}
		if strings.EqualFold(target, "zh-TW") || strings.EqualFold(target, "zh-Hant") {
			focus = append([]string{"Traditional Chinese script and regional terminology"}, focus...)
		}
		return []string{"qwen/qwen3.5-397b-a17b", "openai/gpt-5.4", "google/gemini-3.5-flash-lite"}, "CJK-first candidates informed by multilingual model reports and current OpenRouter translation usage. Run the blind sample because quality changes by language pair.", focus
	case "fr", "de", "es", "pt", "it", "nl", "ca", "ro", "sv", "da", "no", "fi":
		focus := []string{"technical terminology", "idiomatic professional prose", "preserved code and identifiers"}
		if base == "pt" {
			focus = append([]string{"Brazilian or European Portuguese consistency"}, focus...)
		}
		return []string{"mistralai/mistral-small-2603", "openai/gpt-5.4-mini", "google/gemini-3.5-flash-lite"}, "European-language candidate set balancing a multilingual European model family with two strong general models.", focus
	case "id", "ms", "vi", "th", "tl", "jv":
		return []string{"google/gemini-3.5-flash-lite", "qwen/qwen3.7-plus", "openai/gpt-5.4-mini"}, "Southeast Asian languages remain uneven in broad benchmarks. Start with Google and Qwen, then use the blind sample to choose.", []string{"regional terminology", "natural technical register", "preserved code and identifiers"}
	case "ar", "fa", "ur":
		return []string{"mistralai/mistral-saba", "openai/gpt-5.4", "qwen/qwen3.7-plus"}, "Includes a regional Middle Eastern and South Asian specialist alongside premium multilingual candidates.", []string{"right-to-left punctuation", "technical terminology", "preserved code and identifiers"}
	case "hi", "bn", "ta", "te", "ml", "mr", "gu", "kn", "pa", "ne", "si":
		return []string{"google/gemini-3.5-flash-lite", "openai/gpt-5.4", "qwen/qwen3.7-plus"}, "Indic-language benchmarks show large pair-specific variation, so the suggested models are starting candidates rather than a fixed winner.", []string{"native-script consistency", "technical terminology", "preserved code and identifiers"}
	case "ru", "uk", "pl", "cs", "sk", "sl", "hr", "sr", "bg":
		return []string{"openai/gpt-5.4-mini", "google/gemini-3.5-flash-lite", "mistralai/mistral-small-2603"}, "Broad multilingual candidates for Slavic languages. Use project terminology and a blind sample to select the final model.", []string{"case and inflection", "technical terminology", "preserved code and identifiers"}
	default:
		return []string{"openai/gpt-5.4", "anthropic/claude-sonnet-5", "google/gemini-3.5-flash-lite"}, "Premium general candidates are used where current public evidence is too sparse for a language-specific winner.", []string{"meaning preservation", "natural technical register", "preserved code and identifiers"}
	}
}

func comparisonReasoningEffort(model translationModel) string {
	if !model.ReasoningMandatory {
		return "none"
	}
	for index := len(model.ReasoningEfforts) - 1; index >= 0; index-- {
		effort := model.ReasoningEfforts[index]
		if effort == "minimal" || effort == "low" {
			return effort
		}
	}
	if len(model.ReasoningEfforts) > 0 {
		return model.ReasoningEfforts[len(model.ReasoningEfforts)-1]
	}
	return ""
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func compactDescription(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= 220 {
		return value
	}
	return strings.TrimSpace(value[:217]) + "..."
}

func randomBool() bool {
	token, err := newToken()
	return err == nil && len(token) > 0 && token[len(token)-1]%2 == 0
}
