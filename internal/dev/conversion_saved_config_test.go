package dev

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestAuthoringConversionRejectsConfigurationSavedByAnotherSession(t *testing.T) {
	root := authoringFixture(t)
	writeDevFixture(t, root, "content/guide.md", "# Guide\n\nOriginal guide.\n")
	indexPath := filepath.Join(root, "content", "index.md")
	guidePath := filepath.Join(root, "content", "guide.md")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	guideBytes, err := os.ReadFile(guidePath)
	if err != nil {
		t.Fatal(err)
	}

	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "conversion-saved-config"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	saved := server.activeConfig()
	saved.Site.Title = "Saved by another session"
	saved.Translation.StyleGuide = "content/guide.md"
	if err := config.Save(root, saved); err != nil {
		t.Fatal(err)
	}
	savedBytes, err := os.ReadFile(filepath.Join(root, config.Filename))
	if err != nil {
		t.Fatal(err)
	}

	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	response := requestJSON(t, http.MethodPost, httpServer.URL+"/__mpress/api/translations", map[string]any{"action": "convert-mpd"}, "conversion-saved-config")
	body := readBody(response)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(body, "reload") {
		t.Fatalf("stale server settings were not reported with an actionable conflict: HTTP %d %s", response.StatusCode, body)
	}
	if got := readTranslationFixture(t, filepath.Join(root, config.Filename)); string(got) != string(savedBytes) {
		t.Fatalf("saved configuration changed: %q", got)
	}
	assertDevFileUnchanged(t, filepath.Dir(indexPath), filepath.Base(indexPath), string(indexBytes))
	assertDevFileUnchanged(t, filepath.Dir(guidePath), filepath.Base(guidePath), string(guideBytes))
	if matches, _ := filepath.Glob(filepath.Join(root, "content", "*.mpd")); len(matches) != 0 {
		t.Fatalf("rejected conversion published targets: %v", matches)
	}

	// Reloading the saved settings into a new server is the documented retry.
	reloaded, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "conversion-saved-config"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })
	retryServer := httptest.NewServer(reloaded.Handler())
	defer retryServer.Close()
	response = requestJSON(t, http.MethodPost, retryServer.URL+"/__mpress/api/translations", map[string]any{"action": "convert-mpd"}, "conversion-saved-config")
	retryBody := readBody(response)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("conversion after reloading saved settings: HTTP %d %s", response.StatusCode, retryBody)
	}
	updated, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Site.Title != saved.Site.Title || updated.Translation.StyleGuide != "content/guide.mpd" {
		t.Fatalf("reloaded conversion lost settings or reference: %#v %q", updated.Site, updated.Translation.StyleGuide)
	}
	if data, err := os.ReadFile(filepath.Join(root, "content", "guide.mpd")); err != nil || len(data) == 0 {
		t.Fatalf("converted style guide is not readable: %v", err)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "content", "*.md")); len(matches) != 0 {
		t.Fatalf("Markdown documents remain after retry: %v", matches)
	}
	if data, err := os.ReadFile(filepath.Join(root, "site", "index.html")); err != nil || !strings.Contains(string(data), "Editable heading") {
		t.Fatalf("rebuilt MPD output is missing: %v", err)
	}
}
