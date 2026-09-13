package translate

import "testing"

func TestModelSelectionTruthTable(t *testing.T) {
	all := ModelCapabilities{CodexSubscription: true, ClaudeCodeSubscription: true, OpenRouterAPI: true, OpenAIAPI: true}
	tests := []struct {
		name         string
		request      ModelSelectionRequest
		wantProvider string
		wantModel    string
	}{
		{"bulk uses subscription", ModelSelectionRequest{Task: TaskDraft, TargetLanguage: "fr", Capabilities: all}, "codex", "gpt-5.6-sol"},
		{"French refinement uses measured winner", ModelSelectionRequest{Task: TaskRefinement, TargetLanguage: "fr", Capabilities: all}, "openrouter", "google/gemini-3.1-pro-preview"},
		{"audit differs from OpenRouter refinement", ModelSelectionRequest{Task: TaskAudit, TargetLanguage: "fr", PreviousProvider: "openrouter", Capabilities: all}, "codex", "gpt-5.6-sol"},
		{"audit differs from Codex draft", ModelSelectionRequest{Task: TaskAudit, TargetLanguage: "fr", PreviousProvider: "codex", Capabilities: all}, "openrouter", "google/gemini-3.1-pro-preview"},
		{"Codex-only refinement", ModelSelectionRequest{Task: TaskRefinement, TargetLanguage: "fr", Capabilities: ModelCapabilities{CodexSubscription: true}}, "codex", "gpt-5.6-sol"},
		{"OpenRouter-only draft", ModelSelectionRequest{Task: TaskDraft, TargetLanguage: "fr", Capabilities: ModelCapabilities{OpenRouterAPI: true}}, "openrouter", "google/gemini-3.1-pro-preview"},
		{"Claude-only draft", ModelSelectionRequest{Task: TaskDraft, TargetLanguage: "fr", Capabilities: ModelCapabilities{ClaudeCodeSubscription: true}}, "claude", "claude-opus-4.6"},
		{"OpenAI-only fallback", ModelSelectionRequest{Task: TaskRefinement, TargetLanguage: "fr", Capabilities: ModelCapabilities{OpenAIAPI: true}}, "openai", "gpt-5.4"},
		{"single provider may audit itself", ModelSelectionRequest{Task: TaskAudit, TargetLanguage: "fr", PreviousProvider: "openrouter", Capabilities: ModelCapabilities{OpenRouterAPI: true}}, "openrouter", "google/gemini-3.1-pro-preview"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := SelectBestTranslationModel(test.request)
			if !ok || got.Provider != test.wantProvider || got.Model != test.wantModel || got.Basis == "" {
				t.Fatalf("selection = %#v, %v; want %s/%s", got, ok, test.wantProvider, test.wantModel)
			}
		})
	}
}

func TestModelSelectionReturnsFalseWithoutAuthenticatedOptions(t *testing.T) {
	if got, ok := SelectBestTranslationModel(ModelSelectionRequest{Task: TaskDraft, TargetLanguage: "fr"}); ok {
		t.Fatalf("selection = %#v; want no selection", got)
	}
}

func TestEveryTaskHasEveryCapabilityFallback(t *testing.T) {
	capabilities := []ModelCapabilities{
		{CodexSubscription: true}, {ClaudeCodeSubscription: true}, {OpenRouterAPI: true}, {OpenAIAPI: true},
	}
	for _, task := range []TranslationTask{TaskDraft, TaskRefinement, TaskAudit} {
		for _, available := range capabilities {
			if selection, ok := SelectBestTranslationModel(ModelSelectionRequest{Task: task, TargetLanguage: "fr", PreviousProvider: "openrouter", Capabilities: available}); !ok || selection.Model == "" {
				t.Errorf("task %s with %#v has no fallback", task, available)
			}
		}
	}
}

func TestTruthTableCoversEveryCapabilityCombination(t *testing.T) {
	for mask := 0; mask < 16; mask++ {
		available := ModelCapabilities{
			CodexSubscription:      mask&1 != 0,
			ClaudeCodeSubscription: mask&2 != 0,
			OpenRouterAPI:          mask&4 != 0,
			OpenAIAPI:              mask&8 != 0,
		}
		for _, task := range []TranslationTask{TaskDraft, TaskRefinement, TaskAudit} {
			selection, ok := SelectBestTranslationModel(ModelSelectionRequest{
				Task: task, TargetLanguage: "fr", PreviousProvider: "openrouter", Capabilities: available,
			})
			if mask == 0 {
				if ok {
					t.Errorf("mask %04b task %s selected %#v without a capability", mask, task, selection)
				}
				continue
			}
			if !ok || selection.Provider == "" || selection.Model == "" {
				t.Errorf("mask %04b task %s has no complete selection: %#v", mask, task, selection)
			}
		}

		draft, ok := SelectBestTranslationModel(ModelSelectionRequest{Task: TaskDraft, TargetLanguage: "fr", Capabilities: available})
		if !ok {
			continue
		}
		wantDraft := "openai"
		switch {
		case available.CodexSubscription:
			wantDraft = "codex"
		case available.ClaudeCodeSubscription:
			wantDraft = "claude"
		case available.OpenRouterAPI:
			wantDraft = "openrouter"
		}
		if draft.Provider != wantDraft {
			t.Errorf("mask %04b draft provider = %s, want %s", mask, draft.Provider, wantDraft)
		}

		refinement, _ := SelectBestTranslationModel(ModelSelectionRequest{Task: TaskRefinement, TargetLanguage: "fr", Capabilities: available})
		wantRefinement := "openai"
		switch {
		case available.OpenRouterAPI:
			wantRefinement = "openrouter"
		case available.CodexSubscription:
			wantRefinement = "codex"
		case available.ClaudeCodeSubscription:
			wantRefinement = "claude"
		}
		if refinement.Provider != wantRefinement {
			t.Errorf("mask %04b refinement provider = %s, want %s", mask, refinement.Provider, wantRefinement)
		}
	}
}
