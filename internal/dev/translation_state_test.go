package dev

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leaanthony/mpress/internal/translate"
)

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
