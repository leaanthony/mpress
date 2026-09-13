package translate

import (
	"strings"

	"github.com/leaanthony/mpress/internal/config"
)

// TranslationTask identifies the quality and cost profile needed from a model.
type TranslationTask string

const (
	TaskDraft      TranslationTask = "draft"
	TaskRefinement TranslationTask = "refinement"
	TaskAudit      TranslationTask = "audit"
)

// ModelCapabilities are the authenticated services available on this machine.
// Detection is kept outside the policy so this truth table is deterministic and
// can be exhaustively tested without starting a harness or contacting a service.
type ModelCapabilities struct {
	CodexSubscription      bool
	ClaudeCodeSubscription bool
	OpenRouterAPI          bool
	OpenAIAPI              bool
}

// ModelSelection is one complete provider configuration chosen by the policy.
type ModelSelection struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Command  string `json:"command,omitempty"`
	Basis    string `json:"basis"`
}

// ModelSelectionRequest contains every input to the selection truth table.
type ModelSelectionRequest struct {
	Task             TranslationTask
	TargetLanguage   string
	PreviousProvider string
	Capabilities     ModelCapabilities
}

type modelSelectionRule struct {
	task              TranslationTask
	capability        string
	provider          string
	model             string
	command           string
	languageFamily    string
	avoidSameProvider bool
	basis             string
}

// modelSelectionTruthTable is ordered from best to fallback. These choices use
// subscriptions for the large draft pass, the measured French refinement
// winner for difficult repairs, and a different provider for independent audit.
var modelSelectionTruthTable = []modelSelectionRule{
	{task: TaskDraft, capability: "codex", provider: "codex", model: "gpt-5.6-sol", command: "codex", basis: "best available subscription model for the bulk pass"},
	{task: TaskDraft, capability: "claude", provider: "claude", model: "claude-opus-4.6", command: "claude", basis: "authenticated Claude Code subscription fallback"},
	{task: TaskDraft, capability: "openrouter", provider: "openrouter", model: "google/gemini-3.1-pro-preview", basis: "best measured paid model when no subscription harness is available"},
	{task: TaskDraft, capability: "openai", provider: "openai", model: "gpt-5.4", basis: "direct OpenAI API fallback"},

	{task: TaskRefinement, capability: "openrouter", provider: "openrouter", model: "google/gemini-3.1-pro-preview", languageFamily: "european", basis: "best result on the fixed French refinement corpus"},
	{task: TaskRefinement, capability: "codex", provider: "codex", model: "gpt-5.6-sol", command: "codex", basis: "strongest measured subscription refinement fallback"},
	{task: TaskRefinement, capability: "openrouter", provider: "openrouter", model: "google/gemini-3.1-pro-preview", basis: "strongest measured OpenRouter refinement candidate"},
	{task: TaskRefinement, capability: "claude", provider: "claude", model: "claude-opus-4.6", command: "claude", basis: "Claude Code refinement fallback; 4.6 beat 5 in the fixed corpus"},
	{task: TaskRefinement, capability: "openai", provider: "openai", model: "gpt-5.4", basis: "direct OpenAI API fallback"},

	{task: TaskAudit, capability: "codex", provider: "codex", model: "gpt-5.6-sol", command: "codex", avoidSameProvider: true, basis: "independent blind reviewer through the Codex subscription"},
	{task: TaskAudit, capability: "openrouter", provider: "openrouter", model: "google/gemini-3.1-pro-preview", avoidSameProvider: true, basis: "independent OpenRouter reviewer"},
	{task: TaskAudit, capability: "claude", provider: "claude", model: "claude-opus-4.6", command: "claude", avoidSameProvider: true, basis: "independent Claude Code reviewer"},
	{task: TaskAudit, capability: "openai", provider: "openai", model: "gpt-5.4", avoidSameProvider: true, basis: "independent direct OpenAI reviewer"},
	// A same-provider audit is still better than silently skipping AI review when
	// the user has only one authenticated option.
	{task: TaskAudit, capability: "codex", provider: "codex", model: "gpt-5.6-sol", command: "codex", basis: "only available authenticated reviewer"},
	{task: TaskAudit, capability: "openrouter", provider: "openrouter", model: "google/gemini-3.1-pro-preview", basis: "only available authenticated reviewer"},
	{task: TaskAudit, capability: "claude", provider: "claude", model: "claude-opus-4.6", command: "claude", basis: "only available authenticated reviewer"},
	{task: TaskAudit, capability: "openai", provider: "openai", model: "gpt-5.4", basis: "only available authenticated reviewer"},
}

// SelectBestTranslationModel applies the ordered truth table. The boolean is
// false when no authenticated provider or harness is available.
func SelectBestTranslationModel(request ModelSelectionRequest) (ModelSelection, bool) {
	task := request.Task
	if task == "" {
		task = TaskDraft
	}
	previous := strings.ToLower(strings.TrimSpace(request.PreviousProvider))
	family := translationLanguageFamily(request.TargetLanguage)
	for _, rule := range modelSelectionTruthTable {
		if rule.task != task || !hasModelCapability(request.Capabilities, rule.capability) {
			continue
		}
		if rule.languageFamily != "" && rule.languageFamily != family {
			continue
		}
		if rule.avoidSameProvider && previous == rule.provider {
			continue
		}
		return ModelSelection{Provider: rule.provider, Model: rule.model, Command: rule.command, Basis: rule.basis}, true
	}
	return ModelSelection{}, false
}

// ApplyModelSelection configures one in-memory run without persisting provider
// details or credentials to the project file.
func ApplyModelSelection(cfg *config.Config, selection ModelSelection) {
	cfg.Translation.Provider = selection.Provider
	cfg.Translation.Model = selection.Model
	if selection.Command != "" {
		cfg.Translation.Command = selection.Command
	}
	switch selection.Provider {
	case "openrouter":
		cfg.Translation.BaseURL = "https://openrouter.ai/api/v1"
		cfg.Translation.APIKeyEnv = "OPENROUTER_API_KEY"
	case "openai":
		cfg.Translation.BaseURL = "https://api.openai.com/v1"
		cfg.Translation.APIKeyEnv = "OPENAI_API_KEY"
	}
}

func hasModelCapability(capabilities ModelCapabilities, capability string) bool {
	switch capability {
	case "codex":
		return capabilities.CodexSubscription
	case "claude":
		return capabilities.ClaudeCodeSubscription
	case "openrouter":
		return capabilities.OpenRouterAPI
	case "openai":
		return capabilities.OpenAIAPI
	default:
		return false
	}
}

func translationLanguageFamily(language string) string {
	base := strings.ToLower(strings.SplitN(strings.TrimSpace(language), "-", 2)[0])
	switch base {
	case "fr", "de", "es", "pt", "it", "nl", "ca", "ro", "sv", "da", "no", "fi":
		return "european"
	case "zh", "ja", "ko":
		return "cjk"
	case "ar", "fa", "ur":
		return "arabic"
	default:
		return "general"
	}
}
