package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var introducedNumberedMarkerPattern = regexp.MustCompile(`(?:^|\s)\([0-9]+\)(?:\s|$)`)

// LocalProvider sends the same structured translation request to a locally
// installed coding agent. The command receives the prompt on stdin and must
// write either the translation object or a JSON response containing it.
// Keeping this adapter command-based lets users choose their authenticated
// Codex or Claude Code installation without putting a provider token in YAML.
type LocalProvider struct {
	name    string
	model   string
	command string
	args    []string
}

func NewLocalProvider(name, model, command string) (*LocalProvider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name != "codex" && name != "claude" {
		return nil, fmt.Errorf("unsupported local translation provider %q", name)
	}
	if strings.TrimSpace(command) == "" {
		command = name
	}
	args := []string{}
	if name == "codex" {
		args = []string{"exec", "--json", "-"}
	} else {
		args = []string{"--print", "--output-format", "json"}
	}
	return &LocalProvider{name: name, model: model, command: command, args: args}, nil
}

func (p *LocalProvider) Name() string  { return p.name }
func (p *LocalProvider) Model() string { return p.model }

func (p *LocalProvider) Translate(ctx context.Context, request TranslationRequest) (map[string]string, error) {
	if len(request.Segments) == 0 {
		return map[string]string{}, nil
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	prompt := translationPrompt(request) + "\n\nTranslate this JSON request and return the required structured result:\n" + string(payload)
	args := append([]string(nil), p.args...)
	var outputPath, schemaPath string
	if p.name == "codex" {
		outputFile, fileErr := os.CreateTemp("", "mpress-codex-output-*.json")
		if fileErr != nil {
			return nil, fileErr
		}
		outputPath = outputFile.Name()
		_ = outputFile.Close()
		defer os.Remove(outputPath)
		schemaFile, fileErr := os.CreateTemp("", "mpress-codex-schema-*.json")
		if fileErr != nil {
			return nil, fileErr
		}
		schemaPath = schemaFile.Name()
		schema, _ := json.Marshal(translationSchema(request.Segments))
		if _, fileErr = schemaFile.Write(schema); fileErr == nil {
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
		return nil, fmt.Errorf("local %s translation failed: %s", p.name, message)
	}
	providerOutput := stdout.String()
	if outputPath != "" {
		if data, readErr := os.ReadFile(outputPath); readErr == nil && len(data) > 0 {
			providerOutput = string(data)
		}
	}
	result, err := parseLocalTranslation(providerOutput)
	if err != nil {
		return nil, fmt.Errorf("local %s translation returned invalid output: %w; output: %s", p.name, err, preview(providerOutput))
	}
	return validateTranslations(request, result)
}

func parseLocalTranslation(output string) (map[string]string, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil, errors.New("empty output")
	}
	// Agents may wrap the final answer in a JSON event or emit JSONL events.
	candidates := []string{output}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			candidates = append(candidates, line)
		}
	}
	for _, candidate := range candidates {
		var direct map[string]any
		if json.Unmarshal([]byte(candidate), &direct) != nil {
			continue
		}
		if translations, ok := direct["translations"]; ok {
			return translationsMap(translations)
		}
		for _, key := range []string{"result", "text", "output"} {
			if nested, ok := direct[key].(string); ok {
				if parsed, parseErr := parseLocalTranslation(nested); parseErr == nil {
					return parsed, nil
				}
			}
		}
		if item, ok := direct["item"]; ok {
			if encoded, encodeErr := json.Marshal(item); encodeErr == nil {
				if parsed, parseErr := parseLocalTranslation(string(encoded)); parseErr == nil {
					return parsed, nil
				}
			}
		}
	}
	return nil, errors.New("expected a JSON object with translations")
}

func translationsMap(raw any) (map[string]string, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, errors.New("translations is not an array")
	}
	result := make(map[string]string, len(items))
	for _, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return nil, errors.New("translation item is not an object")
		}
		id, idOK := item["id"].(string)
		text, textOK := item["text"].(string)
		if !idOK || !textOK || id == "" {
			return nil, errors.New("translation item requires id and text")
		}
		if _, exists := result[id]; exists {
			return nil, fmt.Errorf("duplicate segment %q", id)
		}
		result[id] = text
	}
	return result, nil
}

func validateTranslations(request TranslationRequest, result map[string]string) (map[string]string, error) {
	for _, segment := range request.Segments {
		translated, ok := result[segment.ID]
		if !ok {
			return nil, fmt.Errorf("provider omitted segment %q", segment.ID)
		}
		if strings.TrimSpace(translated) == "" {
			return nil, fmt.Errorf("provider returned an empty translation for segment %q", segment.ID)
		}
		if request.Format != "mpd" && strings.Count(translated, "\n") != strings.Count(segment.Text, "\n") {
			return nil, fmt.Errorf("provider changed line count for segment %q", segment.ID)
		}
		plainSource := translationPlaceholderPattern.ReplaceAllString(segment.Text, "")
		plainTarget := translationPlaceholderPattern.ReplaceAllString(translated, "")
		if introducedNumberedMarkerPattern.MatchString(plainTarget) && !introducedNumberedMarkerPattern.MatchString(plainSource) {
			return nil, fmt.Errorf("provider introduced a numbered marker in segment %q", segment.ID)
		}
		sourceRunes := utf8.RuneCountInString(strings.TrimSpace(plainSource))
		targetRunes := utf8.RuneCountInString(strings.TrimSpace(plainTarget))
		if request.Format != "mpd" && sourceRunes >= 12 && !containsLetterOrNumber(plainTarget) {
			return nil, fmt.Errorf("provider returned punctuation only for segment %q", segment.ID)
		}
		if sourceRunes >= 20 && targetRunes < max(2, sourceRunes/10) {
			return nil, fmt.Errorf("provider collapsed segment %q from %d to %d characters", segment.ID, sourceRunes, targetRunes)
		}
		if request.Format != "mpd" {
			for _, delimiter := range "()[]{}<>" {
				if strings.Count(translated, string(delimiter)) != strings.Count(segment.Text, string(delimiter)) {
					return nil, fmt.Errorf("provider changed structural delimiter %q in segment %q", delimiter, segment.ID)
				}
			}
		}
		for _, placeholder := range translationPlaceholderPattern.FindAllString(segment.Text, -1) {
			if strings.Count(translated, placeholder) != strings.Count(segment.Text, placeholder) {
				return nil, fmt.Errorf("provider did not preserve placeholder %q in segment %q", placeholder, segment.ID)
			}
		}
	}
	for id := range result {
		found := false
		for _, segment := range request.Segments {
			if segment.ID == id {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("provider returned unknown segment %q", id)
		}
	}
	return result, nil
}

func containsLetterOrNumber(value string) bool {
	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			return true
		}
	}
	return false
}
