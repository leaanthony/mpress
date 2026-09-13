package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
)

type AuditRequest struct {
	SourceLanguage string         `json:"sourceLanguage"`
	TargetLanguage string         `json:"targetLanguage"`
	LocaleGuidance string         `json:"localeGuidance,omitempty"`
	StyleGuide     string         `json:"styleGuide,omitempty"`
	Glossary       []GlossaryTerm `json:"glossary,omitempty"`
	Pairs          []AuditPair    `json:"segments"`
}

type AuditReviewer interface {
	Review(context.Context, AuditRequest) ([]AuditFinding, error)
}

// NewConfiguredAuditReviewer creates an independent reviewer from the selected
// local harness or OpenAI-compatible provider configuration.
func NewConfiguredAuditReviewer(cfg config.Config) (AuditReviewer, error) {
	if cfg.Translation.Provider == "codex" || cfg.Translation.Provider == "claude" {
		return NewLocalProvider(cfg.Translation.Provider, cfg.Translation.Model, cfg.Translation.Command)
	}
	return NewOpenAICompatibleProvider(ProviderOptions{
		Name: cfg.Translation.Provider, Model: cfg.Translation.Model, BaseURL: cfg.Translation.BaseURL,
		APIKey: ResolveAPIKey(cfg), RequireParameters: cfg.Translation.RequireParameters,
		DataCollection: cfg.Translation.DataCollection, HTTPReferer: cfg.Site.BaseURL,
		ApplicationTitle: "M-Press", ReasoningEffort: cfg.Translation.ReasoningEffort,
	})
}

func (p *LocalProvider) Review(ctx context.Context, request AuditRequest) ([]AuditFinding, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	prompt := `You are an independent technical-translation reviewer. Do not rewrite the translation.

Compare every source and target segment. Report only concrete defects: omitted or added meaning, incorrect terminology, changed requirements or negation, untranslated prose, broken grammar caused by inline placeholders, or a target-language error that changes clarity. Do not report subjective style preferences. Use severity "error" for wrong or missing meaning and "warning" for credible ambiguity. Use short kebab-case category codes. Return the required JSON object and no commentary.

Review this request:
` + string(payload)
	args := append([]string(nil), p.args...)
	var outputPath, schemaPath string
	if p.name == "codex" {
		outputFile, fileErr := os.CreateTemp("", "mpress-codex-audit-output-*.json")
		if fileErr != nil {
			return nil, fileErr
		}
		outputPath = outputFile.Name()
		_ = outputFile.Close()
		defer os.Remove(outputPath)
		schemaFile, fileErr := os.CreateTemp("", "mpress-codex-audit-schema-*.json")
		if fileErr != nil {
			return nil, fileErr
		}
		schemaPath = schemaFile.Name()
		encodedSchema, _ := json.Marshal(auditSchema())
		if _, fileErr = schemaFile.Write(encodedSchema); fileErr == nil {
			fileErr = schemaFile.Close()
		}
		if fileErr != nil {
			return nil, fileErr
		}
		defer os.Remove(schemaPath)
		args = []string{"exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules", "--output-schema", schemaPath, "--output-last-message", outputPath}
		if selectedModel := strings.TrimSpace(p.model); selectedModel != "" && selectedModel != "local" {
			args = append(args, "--model", selectedModel)
		}
		args = append(args, "-")
	} else if selectedModel := strings.TrimSpace(p.model); selectedModel != "" && selectedModel != "local" {
		args = append(args, "--model", selectedModel)
	}
	cmd := exec.CommandContext(ctx, p.command, args...)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("local %s translation audit failed: %s", p.name, message)
	}
	output := stdout.String()
	if outputPath != "" {
		if data, readErr := os.ReadFile(outputPath); readErr == nil && len(data) > 0 {
			output = string(data)
		}
	}
	findings, err := parseAuditFindings(output)
	if err != nil {
		return nil, fmt.Errorf("local %s translation audit returned invalid output: %w", p.name, err)
	}
	if err := validateAuditFindings(request, findings); err != nil {
		return nil, err
	}
	return findings, nil
}

func validateAuditFindings(request AuditRequest, findings []AuditFinding) error {
	valid := make(map[string]AuditPair, len(request.Pairs))
	for _, pair := range request.Pairs {
		valid[pair.File+"\x00"+pair.ID] = pair
	}
	for index := range findings {
		finding := &findings[index]
		if finding.Severity != "error" && finding.Severity != "warning" {
			return fmt.Errorf("audit finding has invalid severity %q", finding.Severity)
		}
		if _, ok := valid[finding.File+"\x00"+finding.Segment]; !ok {
			return fmt.Errorf("audit finding refers to unknown segment %s:%s", finding.File, finding.Segment)
		}
		if strings.TrimSpace(finding.Code) == "" || strings.TrimSpace(finding.Message) == "" {
			return errors.New("audit finding requires code and message")
		}
	}
	return nil
}

func auditSchema() map[string]any {
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"findings": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object", "additionalProperties": false,
					"properties": map[string]any{
						"severity": map[string]any{"type": "string", "enum": []string{"error", "warning"}},
						"code":     map[string]any{"type": "string"}, "file": map[string]any{"type": "string"},
						"segment": map[string]any{"type": "string"}, "message": map[string]any{"type": "string"},
					},
					"required": []string{"severity", "code", "file", "segment", "message"},
				},
			},
		},
		"required": []string{"findings"},
	}
}

func parseAuditFindings(output string) ([]AuditFinding, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, errors.New("empty output")
	}
	candidates := []string{output}
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			candidates = append(candidates, line)
		}
	}
	for _, candidate := range candidates {
		var value map[string]json.RawMessage
		if json.Unmarshal([]byte(candidate), &value) != nil {
			continue
		}
		if raw, ok := value["findings"]; ok {
			var findings []AuditFinding
			if err := json.Unmarshal(raw, &findings); err == nil {
				return findings, nil
			}
		}
		for _, key := range []string{"result", "text", "output"} {
			var nested string
			if raw, ok := value[key]; ok && json.Unmarshal(raw, &nested) == nil {
				if findings, err := parseAuditFindings(nested); err == nil {
					return findings, nil
				}
			}
		}
	}
	return nil, errors.New("expected a JSON object with findings")
}
