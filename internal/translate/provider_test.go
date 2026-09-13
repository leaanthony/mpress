package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestOpenAICompatibleProviderUsesStructuredOutputAndOpenRouterPolicy(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("X-OpenRouter-Title"); got != "M-Press" {
			t.Errorf("title header = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"translations\":[{\"id\":\"s1\",\"text\":\"Bonjour\"}]}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`))
	}))
	defer server.Close()

	provider, err := NewOpenAICompatibleProvider(ProviderOptions{
		Name: "openrouter", Model: "example/model", BaseURL: server.URL, APIKey: "secret",
		RequireParameters: true, DataCollection: "deny", ApplicationTitle: "M-Press",
		ReasoningEffort: "none", HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Translate(context.Background(), TranslationRequest{SourceLanguage: "en", TargetLanguage: "fr", Segments: []RequestSegment{{ID: "s1", Text: "Hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	if result["s1"] != "Bonjour" {
		t.Fatalf("result = %#v", result)
	}
	providerPolicy, ok := requestBody["provider"].(map[string]any)
	if !ok || providerPolicy["require_parameters"] != true || providerPolicy["data_collection"] != "deny" {
		t.Fatalf("OpenRouter policy missing: %#v", requestBody["provider"])
	}
	reasoning, ok := requestBody["reasoning"].(map[string]any)
	if !ok || reasoning["enabled"] != false {
		t.Fatalf("OpenRouter reasoning policy missing: %#v", requestBody["reasoning"])
	}
	format, ok := requestBody["response_format"].(map[string]any)
	if !ok || format["type"] != "json_schema" {
		t.Fatalf("structured output missing: %#v", requestBody["response_format"])
	}
}

func TestConfiguredProviderUsesLanguageSpecificModel(t *testing.T) {
	models := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		models <- body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"translations\":[{\"id\":\"s1\",\"text\":\"translated\"}]}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	t.Setenv("MPRESS_TEST_TRANSLATION_KEY", "secret")
	cfg := config.Default()
	cfg.Translation.Provider = "openrouter"
	cfg.Translation.Model = "default/model"
	cfg.Translation.BaseURL = server.URL
	cfg.Translation.APIKeyEnv = "MPRESS_TEST_TRANSLATION_KEY"
	cfg.Translation.LanguageModels = map[string]config.TranslationLanguageModel{
		"ja": {Model: "japanese/model", ReasoningEffort: "none"},
	}
	provider := &configuredProvider{cfg: cfg}
	for _, language := range []string{"fr", "ja"} {
		_, err := provider.Translate(context.Background(), TranslationRequest{
			TargetLanguage: language,
			Segments:       []RequestSegment{{ID: "s1", Text: "source"}},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := <-models; got != "default/model" {
		t.Fatalf("default model = %q", got)
	}
	if got := <-models; got != "japanese/model" {
		t.Fatalf("Japanese model = %q", got)
	}
}

func TestOpenAICompatibleProviderReviewsKnownSegmentsWithStructuredOutput(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"audit","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"findings\":[{\"severity\":\"error\",\"code\":\"missing-negation\",\"file\":\"guide.mpd\",\"segment\":\"p1\",\"message\":\"The target omits the negation.\"}]}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	provider, err := NewOpenAICompatibleProvider(ProviderOptions{Name: "openrouter", Model: "test", BaseURL: server.URL, APIKey: "secret", RequireParameters: true, HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	findings, err := provider.Review(context.Background(), AuditRequest{SourceLanguage: "en", TargetLanguage: "fr", Pairs: []AuditPair{{File: "guide.mpd", ID: "p1", Source: "Do not stop.", Target: "Arrêtez."}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "missing-negation" {
		t.Fatalf("findings = %#v", findings)
	}
	format, ok := requestBody["response_format"].(map[string]any)
	if !ok || format["type"] != "json_schema" {
		t.Fatalf("structured audit output missing: %#v", requestBody["response_format"])
	}
}

func TestProviderRejectsIncompleteResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"translations\":[]}"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	provider, err := NewOpenAICompatibleProvider(ProviderOptions{Name: "openai-compatible", Model: "test", BaseURL: server.URL, APIKey: "secret", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.Translate(context.Background(), TranslationRequest{Segments: []RequestSegment{{ID: "required", Text: "Hello"}}})
	if err == nil {
		t.Fatal("expected omitted segment error")
	}
}

func TestProviderRetriesInvalidSegmentIDs(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		content := `{"translations":[{"id":"invented","text":"Bonjour"}]}`
		if attempt > 1 {
			content = `{"translations":[{"id":"required","text":"Bonjour"}]}`
		}
		response := map[string]any{"id": "test", "object": "chat.completion", "created": 1, "model": "test", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}}}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	provider, err := NewOpenAICompatibleProvider(ProviderOptions{Name: "openai-compatible", Model: "test", BaseURL: server.URL, APIKey: "secret", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Translate(context.Background(), TranslationRequest{Segments: []RequestSegment{{ID: "required", Text: "Hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || result["required"] != "Bonjour" {
		t.Fatalf("invalid structured output was not retried: calls=%d result=%#v", calls.Load(), result)
	}
}

func TestProviderRetriesEmptyTranslation(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempt := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		text := ""
		if attempt > 1 {
			text = "Bonjour"
		}
		content, _ := json.Marshal(map[string]any{"translations": []any{map[string]any{"id": "required", "text": text}}})
		response := map[string]any{"id": "test", "object": "chat.completion", "created": 1, "model": "test", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": string(content)}, "finish_reason": "stop"}}}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	provider, err := NewOpenAICompatibleProvider(ProviderOptions{Name: "openai-compatible", Model: "test", BaseURL: server.URL, APIKey: "secret", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Translate(context.Background(), TranslationRequest{Segments: []RequestSegment{{ID: "required", Text: "Hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || result["required"] != "Bonjour" {
		t.Fatalf("empty translation was not retried: calls=%d result=%#v", calls.Load(), result)
	}
}

func TestProviderRetriesCollapsedTranslationAndIntroducedMarker(t *testing.T) {
	for name, invalid := range map[string]string{
		"collapsed":       "：",
		"numbered marker": "(2) Un appel transforme l'arbre de contenu.",
		"extra delimiter": ") ) comme métadonnées.",
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			valid := "Une traduction technique complète et correcte."
			if name == "extra delimiter" {
				valid = ") comme métadonnées."
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				text := invalid
				if calls.Add(1) > 1 {
					text = valid
				}
				content, _ := json.Marshal(map[string]any{"translations": []any{map[string]any{"id": "required", "text": text}}})
				response := map[string]any{"id": "test", "object": "chat.completion", "created": 1, "model": "test", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": string(content)}, "finish_reason": "stop"}}}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			provider, err := NewOpenAICompatibleProvider(ProviderOptions{Name: "openai-compatible", Model: "test", BaseURL: server.URL, APIKey: "secret", HTTPClient: server.Client()})
			if err != nil {
				t.Fatal(err)
			}
			source := "One call turns the complete content tree into the static site."
			if name == "extra delimiter" {
				source = ") as metadata."
			}
			result, err := provider.Translate(context.Background(), TranslationRequest{Segments: []RequestSegment{{ID: "required", Text: source}}})
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 || result["required"] != valid {
				t.Fatalf("invalid translation was not retried: calls=%d result=%#v", calls.Load(), result)
			}
		})
	}
}

func TestLocalProviderUsesConfiguredAgentCommand(t *testing.T) {
	for _, harness := range []string{"codex", "claude"} {
		t.Run(harness, func(t *testing.T) {
			dir := t.TempDir()
			command := filepath.Join(dir, "agent")
			arguments := filepath.Join(dir, "arguments")
			script := "#!/bin/sh\nprintf '%s\\n' \"$@\" >\"$MPRESS_HARNESS_ARGUMENTS\"\ncat >/dev/null\nprintf '%s' '{\"translations\":[{\"id\":\"s1\",\"text\":\"Bonjour\"}]}'\n"
			if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("MPRESS_HARNESS_ARGUMENTS", arguments)
			provider, err := NewLocalProvider(harness, "selected-model", command)
			if err != nil {
				t.Fatal(err)
			}
			result, err := provider.Translate(context.Background(), TranslationRequest{Segments: []RequestSegment{{ID: "s1", Text: "Hello"}}})
			if err != nil {
				t.Fatal(err)
			}
			if result["s1"] != "Bonjour" {
				t.Fatalf("result = %#v", result)
			}
			argumentData, err := os.ReadFile(arguments)
			if err != nil {
				t.Fatal(err)
			}
			if harness == "codex" && !strings.Contains(string(argumentData), "exec") {
				t.Fatalf("Codex invocation arguments = %q", argumentData)
			}
			if harness == "claude" && (!strings.Contains(string(argumentData), "--print") || !strings.Contains(string(argumentData), "--output-format")) {
				t.Fatalf("Claude Code invocation arguments = %q", argumentData)
			}
			if !strings.Contains(string(argumentData), "--model\nselected-model") {
				t.Fatalf("selected model was not passed to %s: %q", harness, argumentData)
			}
		})
	}
}

func TestMPDTranslationMayReflowParagraphLines(t *testing.T) {
	request := TranslationRequest{Format: "mpd", Segments: []RequestSegment{{ID: "p1", Text: "First source line (with context).\nSecond source line."}}}
	result, err := validateTranslations(request, map[string]string{"p1": "翻译后的段落（包含上下文）可以重新排版为一行。"})
	if err != nil {
		t.Fatal(err)
	}
	if result["p1"] == "" {
		t.Fatal("MPD translation was not retained")
	}
}

func TestTranslationPromptUsesQualityHierarchy(t *testing.T) {
	prompt := translationSystemPrompt("mpd")
	for _, required := range []string{
		"1. Fidelity:",
		"2. Integrity:",
		"3. Terminology:",
		"4. Naturalness:",
		"5. Style:",
		"A Git checkout or checked-out project is a working copy or working tree, never an extraction",
		"The source format is MPress Document (MPD), not Markdown",
		"__MPRESS_INLINE_n__ tokens, exactly once and unchanged",
	} {
		if !strings.Contains(prompt, required) {
			t.Errorf("translation prompt does not contain %q", required)
		}
	}
}

func TestTranslationPromptVersionChangesWithEditorialPolicy(t *testing.T) {
	if promptVersion != "mpress-translation-v2" {
		t.Fatalf("promptVersion = %q, want mpress-translation-v2", promptVersion)
	}
}

func TestParseLocalTranslationReadsCodexAgentMessageEvent(t *testing.T) {
	result, err := parseLocalTranslation(`{"type":"item.completed","item":{"type":"agent_message","text":"{\"translations\":[{\"id\":\"s1\",\"text\":\"Bonjour\"}]}"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if result["s1"] != "Bonjour" {
		t.Fatalf("result = %#v", result)
	}
}

func TestParseAuditFindingsAcceptsDirectAndNestedJSON(t *testing.T) {
	for _, input := range []string{
		`{"findings":[{"severity":"warning","code":"grammar","file":"page.mpd","segment":"b0001-text-01","message":"Check grammar"}]}`,
		`{"result":"{\"findings\":[]}"}`,
	} {
		findings, err := parseAuditFindings(input)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(input, "grammar") && (len(findings) != 1 || findings[0].Code != "grammar") {
			t.Fatalf("unexpected findings: %#v", findings)
		}
	}
}
