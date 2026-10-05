package dev

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/translate"
)

func TestAuthoringTranslationPlanRejectsExternalFiles(t *testing.T) {
	for _, boundary := range []string{"source", "target", "state-file", "state-parent", "state-discovery", "style-guide", "glossary"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			sentinel, original := installTranslationInputBoundary(t, server, boundary)
			response := translationBoundaryRequest(t, server, http.MethodGet, "/__mpress/api/translations?lang=fr&file=index.md", nil)
			defer response.Body.Close()
			if response.StatusCode < 400 {
				t.Errorf("translation plan read external %s with HTTP %d", boundary, response.StatusCode)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
		})
	}
}

type countedBoundaryProvider struct{ calls int }

func (p *countedBoundaryProvider) Name() string  { return "test" }
func (p *countedBoundaryProvider) Model() string { return "test" }
func (p *countedBoundaryProvider) Translate(ctx context.Context, request translate.TranslationRequest) (map[string]string, error) {
	p.calls++
	return pageUpdateProvider{}.Translate(ctx, request)
}

type rearrangingBoundaryProvider struct {
	hook  func() error
	calls int
}

func (p *rearrangingBoundaryProvider) Name() string  { return "test" }
func (p *rearrangingBoundaryProvider) Model() string { return "test" }
func (p *rearrangingBoundaryProvider) Translate(ctx context.Context, request translate.TranslationRequest) (map[string]string, error) {
	if p.calls == 0 {
		if err := p.hook(); err != nil {
			return nil, err
		}
	}
	p.calls++
	return pageUpdateProvider{}.Translate(ctx, request)
}

func TestTranslationRunRejectsParentsRearrangedDuringProvider(t *testing.T) {
	for _, boundary := range []string{"target", "state-file"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			provider, sentinel := translationParentSwap(t, server, boundary)
			_, err := translate.NewEngine(server.project, server.cfg, provider).RunRoot(context.Background(), server.files, translate.Options{Language: "fr", File: "index.md", Scope: "all", Force: true})
			if err == nil || provider.calls != 1 {
				t.Errorf("provider rearrangement escaped boundary: calls=%d error=%v", provider.calls, err)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), "outside must survive")
		})
	}
}

func TestTranslationRunRejectsExternalFilesBeforeProvider(t *testing.T) {
	testTranslationInputBoundaries(t, []string{"source", "target", "state-file", "state-parent", "state-discovery"}, "run")
}

func testTranslationInputBoundaries(t *testing.T, boundaries []string, operation string) {
	t.Helper()
	for _, boundary := range boundaries {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			var findings []translate.AuditFinding
			if operation == "refine" {
				findings = refinementBoundaryFinding(t, server)
			}
			sentinel, original := installTranslationInputBoundary(t, server, boundary)
			provider := &countedBoundaryProvider{}
			reviewer := &boundaryAuditReviewer{}
			engine := translate.NewEngine(server.project, server.cfg, provider)
			var err error
			var calls int
			switch operation {
			case "run":
				_, err = engine.RunRoot(context.Background(), server.files, translate.Options{Language: "fr", File: "index.md", Scope: "all", Force: true})
				calls = provider.calls
			case "audit":
				_, err = engine.AuditWithReviewer(context.Background(), "fr", "index.md", reviewer)
				calls = reviewer.calls
			case "refine":
				_, err = engine.RefineWithProvider(context.Background(), "fr", "index.md", findings, provider)
				calls = provider.calls
			default:
				t.Fatalf("unknown boundary operation %q", operation)
			}
			if err == nil || calls != 0 {
				t.Errorf("unsafe %s reached provider: calls=%d error=%v", operation, calls, err)
			}
			assertDevFileUnchanged(t, filepath.Dir(sentinel), filepath.Base(sentinel), string(original))
		})
	}
}

func TestTranslationRunRetainsInternalAliases(t *testing.T) {
	for _, boundary := range []string{"source", "target", "state-file", "state-parent"} {
		t.Run(boundary, func(t *testing.T) {
			server := translationReviewFixture(t)
			path := translationReviewPath(server, boundary)
			if boundary == "state-parent" {
				path = filepath.Dir(path)
			}
			saved := path + ".saved"
			if err := os.Rename(path, saved); err != nil {
				t.Fatal(err)
			}
			devSymlink(t, filepath.Base(saved), path)
			provider := &countedBoundaryProvider{}
			report, err := translate.NewEngine(server.project, server.cfg, provider).RunRoot(context.Background(), server.files, translate.Options{Language: "fr", File: "index.md", Scope: "all", Force: true})
			if err != nil || report.Written != 1 || provider.calls == 0 {
				t.Fatalf("safe run alias failed: %v %+v calls=%d", err, report, provider.calls)
			}
			response := translationReviewRequest(t, server)
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("written translation cannot be reviewed: %d", response.StatusCode)
			}
			assertReviewedTranslationState(t, translationReviewPath(server, "state-file"))
		})
	}
}

func translationParentSwap(t *testing.T, server *Server, boundary string) (*rearrangingBoundaryProvider, string) {
	t.Helper()
	path := translationReviewPath(server, boundary)
	parent := filepath.Dir(path)
	outside := t.TempDir()
	sentinel := filepath.Join(outside, filepath.Base(path))
	if err := os.WriteFile(sentinel, []byte("outside must survive"), 0600); err != nil {
		t.Fatal(err)
	}
	// Probe native support before the worker callback; later fixture
	// creation failures are returned and must fail this case.
	probe := filepath.Join(outside, "probe")
	devSymlink(t, "missing", probe)
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	provider := &rearrangingBoundaryProvider{hook: func() error {
		if err := os.Rename(parent, parent+".owned"); err != nil {
			return err
		}
		return os.Symlink(outside, parent)
	}}
	return provider, sentinel
}

type boundaryAuditReviewer struct{ calls int }

func (r *boundaryAuditReviewer) Review(_ context.Context, request translate.AuditRequest) ([]translate.AuditFinding, error) {
	r.calls++
	return nil, nil
}

func refinementBoundaryFinding(t *testing.T, server *Server) []translate.AuditFinding {
	t.Helper()
	document, err := translate.ExtractMarkdown(readTranslationFixture(t, translationReviewPath(server, "source")))
	if err != nil {
		t.Fatal(err)
	}
	for _, segment := range document.Segments {
		if !segment.Protected && segment.Kind != "frontmatter" {
			return []translate.AuditFinding{{Severity: "warning", File: "index.md", Segment: segment.ID, Message: "Improve grammar"}}
		}
	}
	t.Fatal("fixture has no refinable segment")
	return nil
}
