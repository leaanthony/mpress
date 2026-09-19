package dev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/translate"
)

type pageUpdateProvider struct{}

func (pageUpdateProvider) Name() string  { return "test" }
func (pageUpdateProvider) Model() string { return "test" }
func (pageUpdateProvider) Translate(_ context.Context, request translate.TranslationRequest) (map[string]string, error) {
	result := map[string]string{}
	for _, segment := range request.Segments {
		result[segment.ID] = request.TargetLanguage + " " + segment.Text
	}
	return result, nil
}

func TestPageTranslationPlanAfterOneLineEdit(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.cfg.Site.Languages = []string{"en", "fr", "de"}
	engine := translate.NewEngine(root, server.cfg, pageUpdateProvider{})
	if _, err := engine.Run(context.Background(), translate.Options{File: "index.md"}); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "content/index.md")
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	writeDevFixture(t, root, "content/index.md", strings.Replace(string(before), "Original **paragraph** with emphasis.", "Updated **paragraph** with emphasis.", 1))
	for _, tc := range []struct {
		query   string
		pending int
	}{
		{"file=content/index.md", 2},
		{"file=content/index.md&lang=fr", 1},
		{"file=content/index.md&scope=missing", 0},
		{"file=content/index.md&scope=all", 6},
	} {
		t.Run(tc.query, func(t *testing.T) {
			response := httptest.NewRecorder()
			server.handleTranslations(response, httptest.NewRequest(http.MethodGet, "/__mpress/api/translations?"+tc.query, nil))
			if response.Code != http.StatusOK {
				t.Fatal(response.Body.String())
			}
			var result struct {
				Report translate.Report `json:"report"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Report.Pending != tc.pending {
				t.Fatalf("pending = %d, want %d: %+v", result.Report.Pending, tc.pending, result.Report)
			}
			for _, file := range result.Report.Files {
				if file.SourceFile != "index.md" || file.States["stale"] != 1 {
					t.Fatalf("plan lost page or single changed passage: %+v", file)
				}
			}
		})
	}
	// Status and estimates must not update either translated file.
	for _, language := range []string{"fr", "de"} {
		target, err := os.ReadFile(filepath.Join(root, "content", language, "index.md"))
		if err != nil || !strings.Contains(string(target), "Original **paragraph**") {
			t.Fatalf("status changed %s: %s (%v)", language, target, err)
		}
	}
}

func TestTranslationStatusKeepsUntrackedTargetsAccessible(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/fr/index.md", "# Traduction humaine\n\nTexte relu.\n")
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	server.cfg.Site.Languages = []string{"en", "fr"}
	request := httptest.NewRequest(http.MethodGet, "/__mpress/api/translations?lang=fr&file=index.md", nil)
	response := httptest.NewRecorder()
	server.handleTranslations(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("translation tools cannot open: %d %s", response.Code, response.Body.String())
	}
	var result struct {
		Report translate.Report `json:"report"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Report.Files) != 1 || result.Report.Files[0].States["untracked"] == 0 || result.Report.Pending != 0 {
		t.Fatalf("untracked text should require review, not automatic replacement: %#v", result.Report)
	}
}
