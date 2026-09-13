package dev

import "testing"

func TestRecommendedModelsForLanguageUsesLanguageSpecificCandidates(t *testing.T) {
	tests := map[string]string{
		"zh-Hant": "qwen/qwen3.5-397b-a17b",
		"fr":      "mistralai/mistral-small-2603",
		"id":      "google/gemini-3.5-flash-lite",
		"ar":      "mistralai/mistral-saba",
		"cy":      "openai/gpt-5.4",
	}
	for language, wanted := range tests {
		models, basis, focus := recommendedModelsForLanguage(language)
		if len(models) < 2 || models[0] != wanted || basis == "" || len(focus) == 0 {
			t.Errorf("%s recommendation = %#v, %q, %#v", language, models, basis, focus)
		}
	}
}

func TestTranslationRecommendationsExcludeUnavailableModels(t *testing.T) {
	models := []translationModel{{ID: "openai/gpt-5.4-mini"}, {ID: "google/gemini-3.5-flash-lite"}}
	recommendations := translationRecommendations([]string{"en", "ja"}, "en", models)
	got := recommendations["ja"].ModelIDs
	available := map[string]bool{"openai/gpt-5.4-mini": true, "google/gemini-3.5-flash-lite": true}
	if len(got) != 2 || !available[got[0]] || !available[got[1]] || got[0] == got[1] {
		t.Fatalf("recommendation contains unavailable models: %#v", got)
	}
}
