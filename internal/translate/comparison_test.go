package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEstimateProjectMeasuresSourceAndRemainingWork(t *testing.T) {
	root, cfg := translationProject(t)
	estimate, err := EstimateProject(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if estimate.Files == 0 || estimate.Segments == 0 || estimate.Requests == 0 || estimate.InputTokens == 0 || estimate.OutputTokens == 0 {
		t.Fatalf("incomplete estimate: %#v", estimate)
	}
	if estimate.TargetLanguages != 1 || estimate.RemainingSegments == 0 || estimate.RemainingInputTokens == 0 || estimate.RemainingOutputTokens == 0 {
		t.Fatalf("remaining translation estimate is missing: %#v", estimate)
	}
}

func TestCompareProjectModelsUsesSameRandomSampleWithoutWriting(t *testing.T) {
	root, cfg := translationProject(t)
	page := "---\ntitle: Comparison\n---\n\n# Translation quality\n\nThis is a substantial technical paragraph about desktop application services and event handling.\n\nThe second paragraph explains how developers validate generated bindings before release.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "index.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		var request TranslationRequest
		if err := json.Unmarshal([]byte(body.Messages[len(body.Messages)-1].Content), &request); err != nil {
			t.Fatal(err)
		}
		translations := make([]map[string]string, 0, len(request.Segments))
		for _, segment := range request.Segments {
			translations = append(translations, map[string]string{"id": segment.ID, "text": "Traduction technique validée par " + body.Model + "."})
		}
		content, _ := json.Marshal(map[string]any{"translations": translations})
		encoded, _ := json.Marshal(string(content))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":` + string(encoded) + `},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	t.Setenv("OPENROUTER_API_KEY", "secret")
	cfg.Translation.BaseURL = server.URL
	cfg.Translation.Provider = "openrouter"
	cfg.Translation.APIKeyEnv = "OPENROUTER_API_KEY"
	comparison, err := CompareProjectModels(context.Background(), root, cfg, "fr", []ComparisonCandidate{{Model: "example/alpha", ReasoningEffort: "none"}, {Model: "example/beta", ReasoningEffort: "none"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(comparison.Source) == 0 || len(comparison.Outputs) != 2 || len(comparison.Outputs[0].Texts) != len(comparison.Source) || len(comparison.Outputs[1].Texts) != len(comparison.Source) {
		t.Fatalf("unexpected comparison: %#v", comparison)
	}
	if comparison.Outputs[0].Texts[0].Text == comparison.Outputs[1].Texts[0].Text || !strings.Contains(comparison.Outputs[0].Texts[0].Text, "example/alpha") {
		t.Fatalf("model outputs were not kept distinct: %#v", comparison.Outputs)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "fr", "index.md")); !os.IsNotExist(err) {
		t.Fatalf("comparison wrote translated content: %v", err)
	}
}

func TestRandomComparisonSampleCanTargetOnePage(t *testing.T) {
	root, cfg := translationProject(t)
	page := "---\ntitle: Candidate\n---\n\n# Candidate\n\nThis candidate page contains enough technical prose to produce a useful translation comparison sample.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "candidate.md"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	_, sourceFile, sample, err := randomComparisonSample(root, cfg, "candidate.md")
	if err != nil {
		t.Fatal(err)
	}
	if sourceFile != "candidate.md" || len(sample) == 0 {
		t.Fatalf("sourceFile=%q sample=%#v", sourceFile, sample)
	}
}
