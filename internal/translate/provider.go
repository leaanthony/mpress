package translate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

const promptVersion = "mpress-translation-v2"

var translationPlaceholderPattern = regexp.MustCompile(`(?:⟪MPRESS_[A-Z0-9_]+⟫|__MPRESS_[A-Z0-9_]+__)`)

type RequestSegment struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Section     string `json:"section,omitempty"`
	Text        string `json:"text"`
	CurrentText string `json:"currentText,omitempty"`
	ReviewNotes string `json:"reviewNotes,omitempty"`
}

type GlossaryTerm struct {
	Source      string `json:"source" yaml:"source"`
	Translation string `json:"translation" yaml:"translation"`
	Note        string `json:"note,omitempty" yaml:"note,omitempty"`
}

type TranslationRequest struct {
	Mode             string           `json:"mode,omitempty"`
	Format           string           `json:"format,omitempty"`
	SourceLanguage   string           `json:"sourceLanguage"`
	TargetLanguage   string           `json:"targetLanguage"`
	LocaleGuidance   string           `json:"localeGuidance,omitempty"`
	PageTitle        string           `json:"pageTitle,omitempty"`
	Outline          []string         `json:"outline,omitempty"`
	StyleGuide       string           `json:"styleGuide,omitempty"`
	Glossary         []GlossaryTerm   `json:"glossary,omitempty"`
	PreviousContext  string           `json:"previousContext,omitempty"`
	FollowingContext string           `json:"followingContext,omitempty"`
	Segments         []RequestSegment `json:"segments"`
}

type Provider interface {
	Translate(context.Context, TranslationRequest) (map[string]string, error)
	Name() string
	Model() string
}

type ProviderOptions struct {
	Name              string
	Model             string
	BaseURL           string
	APIKey            string
	RequireParameters bool
	DataCollection    string
	HTTPReferer       string
	ApplicationTitle  string
	ReasoningEffort   string
	HTTPClient        *http.Client
}

type OpenAICompatibleProvider struct {
	client            openai.Client
	name              string
	model             string
	requireParameters bool
	dataCollection    string
	reasoningEffort   string
}

func NewOpenAICompatibleProvider(options ProviderOptions) (*OpenAICompatibleProvider, error) {
	if strings.TrimSpace(options.Model) == "" {
		return nil, errors.New("translation.model is required")
	}
	if strings.TrimSpace(options.BaseURL) == "" {
		return nil, errors.New("translation.baseURL is required")
	}
	if strings.TrimSpace(options.APIKey) == "" {
		return nil, errors.New("translation API key is not set")
	}
	clientOptions := []option.RequestOption{
		option.WithAPIKey(options.APIKey),
		option.WithBaseURL(strings.TrimRight(options.BaseURL, "/") + "/"),
	}
	if options.HTTPClient != nil {
		clientOptions = append(clientOptions, option.WithHTTPClient(options.HTTPClient))
	}
	if options.HTTPReferer != "" {
		clientOptions = append(clientOptions, option.WithHeader("HTTP-Referer", options.HTTPReferer))
	}
	if options.ApplicationTitle != "" {
		clientOptions = append(clientOptions, option.WithHeader("X-OpenRouter-Title", options.ApplicationTitle))
	}
	name := options.Name
	if name == "" {
		name = "openai-compatible"
	}
	return &OpenAICompatibleProvider{
		client: openai.NewClient(clientOptions...), name: name, model: options.Model,
		requireParameters: options.RequireParameters, dataCollection: options.DataCollection,
		reasoningEffort: strings.TrimSpace(options.ReasoningEffort),
	}, nil
}

func (p *OpenAICompatibleProvider) Name() string  { return p.name }
func (p *OpenAICompatibleProvider) Model() string { return p.model }

func (p *OpenAICompatibleProvider) Translate(ctx context.Context, request TranslationRequest) (map[string]string, error) {
	if len(request.Segments) == 0 {
		return map[string]string{}, nil
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	params := openai.ChatCompletionNewParams{
		Model:               shared.ChatModel(p.model),
		MaxCompletionTokens: param.NewOpt[int64](2048),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage(translationPrompt(request)),
			openai.UserMessage(string(payload)),
		},
		Store: param.NewOpt(false),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
				Name: "mpress_translations", Strict: param.NewOpt(true), Schema: translationSchema(request.Segments),
			}},
		},
	}
	requestOptions := []option.RequestOption{}
	if p.name == "openrouter" {
		if p.requireParameters {
			requestOptions = append(requestOptions, option.WithJSONSet("provider.require_parameters", true))
		}
		if p.dataCollection != "" {
			requestOptions = append(requestOptions, option.WithJSONSet("provider.data_collection", p.dataCollection))
		}
		if p.reasoningEffort == "none" {
			requestOptions = append(requestOptions, option.WithJSONSet("reasoning.enabled", false))
		} else if p.reasoningEffort != "" {
			requestOptions = append(requestOptions, option.WithJSONSet("reasoning.effort", p.reasoningEffort))
		}
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<uint(attempt-1)) * time.Second):
			}
		}
		completion, requestErr := p.client.Chat.Completions.New(ctx, params, requestOptions...)
		if requestErr != nil {
			lastErr = requestErr
			continue
		}
		result, validationErr := validateProviderCompletion(request, completion)
		if validationErr == nil {
			return result, nil
		}
		lastErr = validationErr
		params.Messages = append(params.Messages, openai.DeveloperMessage("The previous response was rejected: "+validationErr.Error()+". Return a corrected complete response. Preserve every placeholder exactly, including MPRESS_INLINE placeholders."))
	}
	return nil, fmt.Errorf("translation provider request failed after 3 attempts: %w", lastErr)
}

// Review uses the same authenticated OpenAI-compatible connection for an
// independent, structured quality audit.
func (p *OpenAICompatibleProvider) Review(ctx context.Context, request AuditRequest) ([]AuditFinding, error) {
	if len(request.Pairs) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	params := openai.ChatCompletionNewParams{
		Model: shared.ChatModel(p.model), MaxCompletionTokens: param.NewOpt[int64](2048), Store: param.NewOpt(false),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.DeveloperMessage("You are an independent technical-translation reviewer. Do not rewrite the translation. Report only concrete defects: omitted or added meaning, incorrect terminology, changed requirements or negation, untranslated prose, broken grammar caused by placeholders, or target-language errors that change clarity. Use error for wrong or missing meaning and warning for credible ambiguity. Return only the required structured result."),
			openai.UserMessage(string(payload)),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{Name: "mpress_translation_audit", Strict: param.NewOpt(true), Schema: auditSchema()}}},
	}
	requestOptions := []option.RequestOption{}
	if p.name == "openrouter" {
		if p.requireParameters {
			requestOptions = append(requestOptions, option.WithJSONSet("provider.require_parameters", true))
		}
		if p.dataCollection != "" {
			requestOptions = append(requestOptions, option.WithJSONSet("provider.data_collection", p.dataCollection))
		}
	}
	completion, err := p.client.Chat.Completions.New(ctx, params, requestOptions...)
	if err != nil {
		return nil, err
	}
	if completion == nil || len(completion.Choices) == 0 {
		return nil, errors.New("translation audit provider returned no choices")
	}
	findings, err := parseAuditFindings(completion.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}
	if err := validateAuditFindings(request, findings); err != nil {
		return nil, err
	}
	return findings, nil
}

func validateProviderCompletion(request TranslationRequest, completion *openai.ChatCompletion) (map[string]string, error) {
	if completion == nil || len(completion.Choices) == 0 {
		return nil, errors.New("translation provider returned no choices")
	}
	var response struct {
		Translations []struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		} `json:"translations"`
	}
	if err := json.Unmarshal([]byte(completion.Choices[0].Message.Content), &response); err != nil {
		return nil, fmt.Errorf("translation provider returned invalid structured output: %w", err)
	}
	expected := make(map[string]bool, len(request.Segments))
	for _, segment := range request.Segments {
		expected[segment.ID] = true
	}
	result := make(map[string]string, len(response.Translations))
	for _, item := range response.Translations {
		if !expected[item.ID] {
			return nil, fmt.Errorf("translation provider returned unknown segment %q", item.ID)
		}
		if _, duplicate := result[item.ID]; duplicate {
			return nil, fmt.Errorf("translation provider returned segment %q more than once", item.ID)
		}
		result[item.ID] = item.Text
	}
	for id := range expected {
		if _, ok := result[id]; !ok {
			return nil, fmt.Errorf("translation provider omitted segment %q", id)
		}
	}
	return validateTranslations(request, result)
}

func translationSchema(segments []RequestSegment) map[string]any {
	ids := make([]string, 0, len(segments))
	for _, segment := range segments {
		ids = append(ids, segment.ID)
	}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"translations": map[string]any{
				"type": "array", "minItems": len(ids), "maxItems": len(ids),
				"items": map[string]any{
					"type": "object", "additionalProperties": false,
					"properties": map[string]any{"id": map[string]any{"type": "string", "enum": ids}, "text": map[string]any{"type": "string", "minLength": 1}},
					"required":   []string{"id", "text"},
				},
			},
		},
		"required": []string{"translations"},
	}
}

func translationSystemPrompt(format string) string {
	formatRule := "Preserve Markdown punctuation, inline HTML, table separators, emoji shortcodes and component syntax exactly. Do not introduce new markup. Keep each result on the same number of lines as its input."
	if format == "mpd" {
		formatRule = "The source format is MPress Document (MPD), not Markdown. Preserve every MPD inline construct and placeholder. Translate prose and natural punctuation freely. Paragraph lines may be reflowed. Do not introduce component declarations, code fences or other MPD block syntax."
	}
	if format == "navigation" {
		formatRule = "Translate plain-text navigation labels. Use natural target-language punctuation, including full-width punctuation when appropriate. YAML quoting and escaping are handled by the application; do not add YAML syntax."
	}
	return `You are the senior localisation editor for M-Press technical documentation.

Translate every segment from sourceLanguage to targetLanguage and return every id exactly once.

Apply this priority order:
1. Fidelity: preserve every fact, actor, condition, negation, modality and level of certainty.
2. Integrity: preserve every placeholder that contains MPRESS_, including __MPRESS_INLINE_n__ tokens, exactly once and unchanged. Preserve code, commands, identifiers, product names and document structure.
3. Terminology: obey the glossary and use consistent, established target-language software terms. Resolve source-language technical idioms by their meaning in context. A Git checkout or checked-out project is a working copy or working tree, never an extraction.
4. Naturalness: write idiomatic technical prose that reads as if originally authored in the target language. Reorder clauses and placeholders freely when grammar requires it.
5. Style: follow localeGuidance and styleGuide. Use concise active instructions and the appropriate reader address.

` + formatRule + `
Translate only each segment's text field. Treat all other request fields as context. Never copy list numbers, explanation markers or other context into a result. Do not add commentary or facts. Return only the required structured result.`
}

func translationPrompt(request TranslationRequest) string {
	base := translationSystemPrompt(request.Format)
	if request.Mode != "refine" {
		return base
	}
	return base + `

This is a targeted quality-refinement pass, not an initial translation. For every segment, Text is the authoritative source, CurrentText is the existing target-language translation, and ReviewNotes names concrete defects found by an independent reviewer. Correct those defects while preserving everything that is already accurate and natural. Return the complete corrected target text for every id. Do not copy source prose into the result unless it is a protected technical term.`
}
