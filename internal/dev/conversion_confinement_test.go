package dev

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/translate"
)

func TestAuthoringConversionRetainsInternalAliases(t *testing.T) {
	for _, boundary := range []string{"source", "state-parent"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			path := translationReviewPath(server, boundary)
			if boundary == "state-parent" {
				path = filepath.Dir(path)
			}
			original := ""
			if boundary == "source" {
				original = string(readTranslationFixture(t, path))
			}
			saved := path + ".saved"
			if err := os.Rename(path, saved); err != nil {
				t.Fatal(err)
			}
			devSymlink(t, saved, path)
			response := conversionBoundaryRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("safe conversion alias failed: HTTP %d", response.StatusCode)
			}
			var state translate.FileState
			if err := json.Unmarshal(readTranslationFixture(t, translationReviewPath(server, "state-file")), &state); err != nil || state.SourceFile != "index.mpd" || state.SchemaVersion != 2 || len(state.Segments) == 0 {
				t.Fatalf("conversion lost translation identity: %v %+v", err, state)
			}
			if boundary == "source" {
				assertDevFileUnchanged(t, filepath.Dir(saved), filepath.Base(saved), original)
			}
		})
	}
}

func conversionBoundaryRequest(t *testing.T, server *Server) *http.Response {
	t.Helper()
	return translationBoundaryRequest(t, server, http.MethodPost, "/__mpress/api/translations", map[string]any{"action": "convert-mpd"})
}

func TestAuthoringConversionRejectsExternalInputs(t *testing.T) {
	for _, boundary := range []string{"source", "state-file", "state-parent"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			sentinel, original := installTranslationReviewBoundary(t, server, boundary)
			response := conversionBoundaryRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode < 400 {
				t.Errorf("conversion accepted external %s: HTTP %d", boundary, response.StatusCode)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
			if _, err := os.Stat(filepath.Join(server.cfg.ContentPath(server.project), "fr/index.mpd")); !os.IsNotExist(err) {
				t.Errorf("unsafe conversion published a target: %v", err)
			}
		})
	}
}

func TestAuthoringConversionPreservesTemporarySentinels(t *testing.T) {
	for _, linked := range []bool{false, true} {
		t.Run(map[bool]string{false: "owned-file", true: "external-link"}[linked], func(t *testing.T) {
			server := translationReviewFixture(t)
			temporary := filepath.Join(server.cfg.ContentPath(server.project), "index.mpd.tmp")
			sentinel := temporary
			if linked {
				sentinel = filepath.Join(t.TempDir(), "must-survive")
			}
			if err := os.WriteFile(sentinel, []byte("unrelated temporary data"), 0600); err != nil {
				t.Fatal(err)
			}
			if linked {
				devSymlink(t, sentinel, temporary)
			}
			response := conversionBoundaryRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Errorf("ordinary conversion failed: HTTP %d", response.StatusCode)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), "unrelated temporary data")
			if _, err := os.Lstat(temporary); err != nil {
				t.Errorf("conversion consumed unrelated temporary entry: %v", err)
			}
		})
	}
}
