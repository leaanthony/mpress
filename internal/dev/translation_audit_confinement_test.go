package dev

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/translate"
)

func TestAuthoringTranslationAuditRejectsExternalFiles(t *testing.T) {
	// Disable provider detection so the HTTP regression never calls a service.
	isolated := t.TempDir()
	t.Setenv("PATH", isolated)
	t.Setenv("XDG_CONFIG_HOME", isolated)
	t.Setenv("APPDATA", isolated)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	for _, boundary := range []string{"source", "target", "glossary"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			if translate.DetectModelCapabilities(server.cfg) != (translate.ModelCapabilities{}) {
				t.Fatal("fixture could not isolate provider detection")
			}
			sentinel, original := installTranslationInputBoundary(t, server, boundary)
			response := translationBoundaryRequest(t, server, http.MethodPost, "/__mpress/api/translations", map[string]any{"action": "audit", "language": "fr", "file": "index.md"})
			defer response.Body.Close()
			if response.StatusCode < 400 {
				t.Errorf("translation audit read external %s: HTTP %d", boundary, response.StatusCode)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
		})
	}
}

func TestTranslationAuditRejectsExternalFilesBeforeReviewer(t *testing.T) {
	testTranslationInputBoundaries(t, []string{"source", "target", "style-guide", "glossary"}, "audit")
}

func TestTranslationRefinementRejectsExternalFilesBeforeProvider(t *testing.T) {
	testTranslationInputBoundaries(t, []string{"source", "target", "state-file", "state-parent", "style-guide", "glossary"}, "refine")
}

func TestTranslationRefinementRejectsParentsRearrangedDuringProvider(t *testing.T) {
	for _, boundary := range []string{"target", "state-file"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			findings := refinementBoundaryFinding(t, server)
			provider, sentinel := translationParentSwap(t, server, boundary)
			engine, err := translate.NewEngine(server.project, server.cfg, nil).BorrowRoot(server.files)
			if err != nil {
				t.Fatal(err)
			}
			_, err = engine.RefineWithProvider(context.Background(), "fr", "index.md", findings, provider)
			if err == nil || provider.calls != 1 {
				t.Errorf("refinement parent swap escaped boundary: calls=%d error=%v", provider.calls, err)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), "outside must survive")
		})
	}
}

func TestTranslationAuditRefinementRetainInternalAliases(t *testing.T) {
	for _, boundary := range []string{"source", "target", "state-parent"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			findings := refinementBoundaryFinding(t, server)
			path := translationReviewPath(server, boundary)
			if boundary == "state-parent" {
				path = filepath.Dir(path)
			}
			if err := os.Rename(path, path+".saved"); err != nil {
				t.Fatal(err)
			}
			devSymlink(t, filepath.Base(path)+".saved", path)
			engine, err := translate.NewEngine(server.project, server.cfg, nil).BorrowRoot(server.files)
			if err != nil {
				t.Fatal(err)
			}
			reviewer := &boundaryAuditReviewer{}
			if _, err := engine.AuditWithReviewer(context.Background(), "fr", "index.md", reviewer); err != nil || reviewer.calls == 0 {
				t.Fatalf("safe audit alias failed: %v calls=%d", err, reviewer.calls)
			}
			provider := &countedBoundaryProvider{}
			report, err := engine.RefineWithProvider(context.Background(), "fr", "index.md", findings, provider)
			if err != nil || report.Written != 1 || provider.calls == 0 {
				t.Fatalf("safe refinement alias failed: %v %+v calls=%d", err, report, provider.calls)
			}
		})
	}
}
