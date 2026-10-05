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

func TestAuthoringTranslationReviewRejectsExternalFiles(t *testing.T) {
	for _, boundary := range []string{"source", "target", "state-file", "state-parent", "state-discovery"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			sentinel, original := installTranslationReviewBoundary(t, server, boundary)
			response := translationReviewRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode < 400 {
				t.Errorf("translation review read/wrote external %s: %d %s", boundary, response.StatusCode, readBody(response))
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
		})
	}
}

func TestAuthoringTranslationReviewRetainsInternalStateAliases(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		t.Run(map[bool]string{false: "relative", true: "absolute"}[absolute], func(t *testing.T) {
			server := translationReviewFixture(t)
			path := translationReviewPath(server, "state-parent")
			parent := filepath.Dir(path)
			target := filepath.Join(server.project, "safe-state")
			if err := os.Rename(parent, target); err != nil {
				t.Fatal(err)
			}
			link := target
			if !absolute {
				var err error
				link, err = filepath.Rel(filepath.Dir(parent), target)
				if err != nil {
					t.Fatal(err)
				}
			}
			devSymlink(t, link, parent)
			response := translationReviewRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("safe state alias rejected: %d %s", response.StatusCode, readBody(response))
			}
			assertReviewedTranslationState(t, path)
		})
	}
}

func translationReviewFixture(t *testing.T) *Server {
	t.Helper()
	root := authoringFixture(t)
	source := string(readTranslationFixture(t, filepath.Join(root, "content/index.md")))
	writeDevFixture(t, root, "content/index.md", strings.Replace(source, "title: Home", "title: Home\ntranslationKey: authoring-review-key", 1))
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "translation-boundary"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.cfg.Site.Languages = []string{"en", "fr"}
	if _, err := translate.NewEngine(root, server.cfg, pageUpdateProvider{}).Run(context.Background(), translate.Options{Language: "fr", File: "index.md"}); err != nil {
		t.Fatal(err)
	}
	return server
}

func translationReviewPath(server *Server, boundary string) string {
	switch boundary {
	case "source":
		return filepath.Join(server.cfg.ContentPath(server.project), "index.md")
	case "target":
		return filepath.Join(server.cfg.ContentPath(server.project), "fr", "index.md")
	default:
		return filepath.Join(server.project, server.cfg.Translation.StateDir, "fr", "index.json")
	}
}

func translationReviewRequest(t *testing.T, server *Server) *http.Response {
	t.Helper()
	return translationBoundaryRequest(t, server, http.MethodPost, "/__mpress/api/translations", map[string]any{"action": "mark", "language": "fr", "file": "index.md", "status": "reviewed"})
}

func translationBoundaryRequest(t *testing.T, server *Server, method, endpoint string, payload any) *http.Response {
	t.Helper()
	host := httptest.NewServer(server.Handler())
	t.Cleanup(host.Close)
	return requestJSON(t, method, host.URL+endpoint, payload, "translation-boundary")
}

func readTranslationFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAuthoringTranslationReviewRetainsInternalFileAliases(t *testing.T) {
	for _, boundary := range []string{"source", "target", "state-file"} {
		for _, absolute := range []bool{false, true} {
			t.Run(boundary+map[bool]string{false: "/relative", true: "/absolute"}[absolute], func(t *testing.T) {
				server := translationReviewFixture(t)
				path := translationReviewPath(server, boundary)
				original := readTranslationFixture(t, path)
				saved := path + ".saved"
				if err := os.Rename(path, saved); err != nil {
					t.Fatal(err)
				}
				link := saved
				if !absolute {
					link = filepath.Base(saved)
				}
				devSymlink(t, link, path)
				response := translationReviewRequest(t, server)
				defer response.Body.Close()
				if response.StatusCode != http.StatusOK {
					t.Fatalf("safe %s alias rejected: %d %s", boundary, response.StatusCode, readBody(response))
				}
				assertReviewedTranslationState(t, translationReviewPath(server, "state-file"))
				if string(readTranslationFixture(t, saved)) != string(original) {
					t.Fatal("review changed alias target")
				}
			})
		}
	}
}

func assertReviewedTranslationState(t *testing.T, path string) {
	t.Helper()
	var state translate.FileState
	if err := json.Unmarshal(readTranslationFixture(t, path), &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Segments) == 0 || state.SourceFile != "index.md" || state.PageKey != "authoring-review-key" {
		t.Fatalf("review lost identity or segments: %+v", state)
	}
	for _, segment := range state.Segments {
		if segment.Status != "reviewed" {
			t.Fatalf("review status was not persisted: %+v", segment)
		}
	}
}

func installTranslationReviewBoundary(t *testing.T, server *Server, boundary string) (string, []byte) {
	t.Helper()
	outside := t.TempDir()
	path := translationReviewPath(server, boundary)
	original := readTranslationFixture(t, path)
	sentinel := filepath.Join(outside, filepath.Base(path))
	switch boundary {
	case "state-discovery":
		var state translate.FileState
		if err := json.Unmarshal(original, &state); err != nil {
			t.Fatal(err)
		}
		state.PageKey = "unrelated-private-state"
		var err error
		original, err = json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		sentinel = filepath.Join(outside, "private.json")
		if err := os.WriteFile(sentinel, original, 0600); err != nil {
			t.Fatal(err)
		}
		devSymlink(t, sentinel, filepath.Join(filepath.Dir(path), "a-private.json"))
	case "state-parent":
		parent := filepath.Dir(path)
		if err := os.Rename(parent, filepath.Join(outside, "state")); err != nil {
			t.Fatal(err)
		}
		sentinel = filepath.Join(outside, "state", filepath.Base(path))
		devSymlink(t, filepath.Join(outside, "state"), parent)
	default:
		if err := os.Rename(path, sentinel); err != nil {
			t.Fatal(err)
		}
		devSymlink(t, sentinel, path)
	}
	return sentinel, original
}

func installTranslationInputBoundary(t *testing.T, server *Server, boundary string) (string, []byte) {
	t.Helper()
	if boundary != "style-guide" && boundary != "glossary" {
		return installTranslationReviewBoundary(t, server, boundary)
	}
	name, data := "style.txt", "Follow the project's writing style.\n"
	if boundary == "glossary" {
		name, data = "glossary.yaml", "terms:\n  - source: compiler\n    translations:\n      fr: compilateur\n"
		server.cfg.Translation.Glossary = name
	} else {
		server.cfg.Translation.StyleGuide = name
	}
	sentinel := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(sentinel, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, sentinel, filepath.Join(server.project, name))
	return sentinel, []byte(data)
}
