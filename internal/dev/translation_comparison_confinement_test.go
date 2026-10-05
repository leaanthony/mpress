package dev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leaanthony/mpress/internal/translate"
)

func TestBorrowedComparisonEngineRetainsRootOwnership(t *testing.T) {
	server, calls := comparisonBoundaryFixture(t)
	engine := translate.NewEngine(server.project, server.cfg, nil)
	borrowed, err := engine.BorrowRoot(server.files)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []translate.ComparisonCandidate{{Model: "boundary/first"}, {Model: "boundary/second"}}
	if _, err := borrowed.Estimate(); err != nil {
		t.Fatal(err)
	}
	if _, err := borrowed.CompareModels(context.Background(), "fr", "index.md", candidates); err != nil {
		t.Fatal(err)
	}
	if _, err := server.files.Stat("content/index.md"); err != nil {
		t.Fatalf("borrowed operations closed the caller's root: %v", err)
	}
	if err := server.files.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Estimate(); err != nil {
		t.Fatalf("estimate retained the borrowed closed root: %v", err)
	}
	if _, err := engine.CompareModels(context.Background(), "fr", "index.md", candidates); err != nil || calls.Load() != 4 {
		t.Fatalf("comparison retained the borrowed root: %v calls=%d", err, calls.Load())
	}
}

func comparisonBoundaryFixture(t *testing.T) (*Server, *atomic.Int32) {
	t.Helper()
	server := translationReviewFixture(t)
	// Sampling requires a body segment with at least 24 visible characters.
	source := translationReviewPath(server, "source")
	writeDevFixture(t, server.project, "content/index.md", string(readTranslationFixture(t, source))+"\nBuild the compiler and validate all generated application bindings before publishing a release.\n")
	calls := &atomic.Int32{}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var input struct {
			Messages []struct{ Content string }
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.Messages) == 0 {
			http.Error(w, "invalid fixture request", http.StatusBadRequest)
			return
		}
		var request translate.TranslationRequest
		if err := json.Unmarshal([]byte(input.Messages[len(input.Messages)-1].Content), &request); err != nil {
			http.Error(w, "invalid fixture payload", http.StatusBadRequest)
			return
		}
		var values []map[string]string
		for _, segment := range request.Segments {
			values = append(values, map[string]string{"id": segment.ID, "text": "fr " + segment.Text})
		}
		content, _ := json.Marshal(map[string]any{"translations": values})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": string(content)}}}})
	}))
	t.Cleanup(provider.Close)
	t.Setenv("MPRESS_COMPARISON_TEST_KEY", "local-fixture-only")
	server.cfg.Translation.Provider = "openrouter"
	server.cfg.Translation.BaseURL = provider.URL
	server.cfg.Translation.APIKeyEnv = "MPRESS_COMPARISON_TEST_KEY"
	server.modelCatalog = []translationModel{{ID: "boundary/first"}, {ID: "boundary/second"}}
	server.modelCatalogAt = time.Now()
	return server, calls
}

func comparisonBoundaryRequest(t *testing.T, server *Server) *http.Response {
	t.Helper()
	return translationBoundaryRequest(t, server, http.MethodPost, "/__mpress/api/translation-models", map[string]any{"action": "compare", "models": []string{"boundary/first", "boundary/second"}, "language": "fr", "file": "index.md"})
}

func TestAuthoringComparisonRejectsExternalFilesBeforeProvider(t *testing.T) {
	for _, boundary := range []string{"source", "style-guide", "glossary"} {
		t.Run(boundary, func(t *testing.T) {
			server, calls := comparisonBoundaryFixture(t)
			sentinel, original := installTranslationInputBoundary(t, server, boundary)
			response := comparisonBoundaryRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode < 400 || calls.Load() != 0 {
				t.Errorf("comparison exposed external %s: HTTP %d provider calls=%d", boundary, response.StatusCode, calls.Load())
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
		})
	}
}

func TestAuthoringEstimateRejectsExternalFilesBeforeTargetsExist(t *testing.T) {
	for _, boundary := range []string{"source", "style-guide"} {
		t.Run(boundary, func(t *testing.T) {
			server, calls := comparisonBoundaryFixture(t)
			server.cfg.Site.Languages = []string{"en"}
			if err := os.RemoveAll(filepath.Join(server.cfg.ContentPath(server.project), "fr")); err != nil {
				t.Fatal(err)
			}
			sentinel, original := installTranslationInputBoundary(t, server, boundary)
			response := translationBoundaryRequest(t, server, http.MethodGet, "/__mpress/api/translation-models", nil)
			defer response.Body.Close()
			if response.StatusCode < 400 || calls.Load() != 0 {
				t.Errorf("estimate read external %s: HTTP %d provider calls=%d", boundary, response.StatusCode, calls.Load())
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
		})
	}
}

func TestAuthoringComparisonRetainsInternalSourceAliasesWithoutWrites(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		t.Run(map[bool]string{false: "relative", true: "absolute"}[absolute], func(t *testing.T) {
			server, calls := comparisonBoundaryFixture(t)
			source := translationReviewPath(server, "source")
			target := translationReviewPath(server, "target")
			state := translationReviewPath(server, "state-file")
			before := map[string]string{source: string(readTranslationFixture(t, source)), target: string(readTranslationFixture(t, target)), state: string(readTranslationFixture(t, state))}
			saved := source + ".saved"
			if err := os.Rename(source, saved); err != nil {
				t.Fatal(err)
			}
			link := saved
			if !absolute {
				link = filepath.Base(saved)
			}
			devSymlink(t, link, source)
			response := comparisonBoundaryRequest(t, server)
			defer response.Body.Close()
			var result struct {
				ComparisonID string            `json:"comparisonId"`
				SourceFile   string            `json:"sourceFile"`
				Outputs      []json.RawMessage `json:"outputs"`
				Error        string            `json:"error"`
			}
			if err := json.NewDecoder(response.Body).Decode(&result); err != nil || response.StatusCode != http.StatusOK || result.ComparisonID == "" || result.SourceFile != "index.md" || len(result.Outputs) != 2 || calls.Load() != 2 {
				t.Fatalf("safe comparison failed: HTTP %d error=%v result=%+v calls=%d", response.StatusCode, err, result, calls.Load())
			}
			for path, data := range before {
				assertDevFileUnchanged(t, filepath.Dir(path), filepath.Base(path), data)
			}
			assertDevFileUnchanged(t, filepath.Dir(saved), filepath.Base(saved), before[source])
		})
	}
}
