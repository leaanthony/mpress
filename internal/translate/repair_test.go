package translate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// flakyProvider breaks protected content in one segment until it is asked
// again about that segment alone with the validation failure attached.
type flakyProvider struct {
	calls   int
	retried []string
}

func (p *flakyProvider) Name() string  { return "flaky" }
func (p *flakyProvider) Model() string { return "test" }
func (p *flakyProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.calls++
	result := map[string]string{}
	for _, segment := range request.Segments {
		if segment.ReviewNotes != "" {
			p.retried = append(p.retried, segment.ID)
			result[segment.ID] = "FR " + segment.Text
			continue
		}
		if strings.Contains(segment.Text, "MPRESS_INLINE") {
			result[segment.ID] = "FR dropped the protected code"
			continue
		}
		result[segment.ID] = "FR " + segment.Text
	}
	return result, nil
}

func TestRetranslatesOnlyTheSegmentThatBrokeProtectedContent(t *testing.T) {
	root, cfg := translationProject(t)
	if err := os.WriteFile(filepath.Join(root, "content", "index.md"), []byte("# Title\n\nPlain paragraph.\n\nUse `wails3 build` here.\n\nAnother plain paragraph.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	provider := &flakyProvider{}
	report, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatalf("a single bad segment must not fail the page: %v", err)
	}
	if report.Written != 1 || len(provider.retried) != 1 {
		t.Fatalf("expected one targeted retry, got %v (%#v)", provider.retried, report)
	}
	out, _ := os.ReadFile(filepath.Join(root, "content", "fr", "index.md"))
	if !strings.Contains(string(out), "`wails3 build`") || strings.Contains(string(out), "dropped") {
		t.Fatalf("repaired page is wrong:\n%s", out)
	}
}

func TestStillFailsWhenTheRetryAlsoBreaksProtectedContent(t *testing.T) {
	root, cfg := translationProject(t)
	if err := os.WriteFile(filepath.Join(root, "content", "index.md"), []byte("# Title\n\nUse `wails3 build` here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	provider := &alwaysBrokenProvider{}
	if _, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr"}); err == nil {
		t.Fatal("a persistently broken segment must still fail the page")
	}
	if _, err := os.Stat(filepath.Join(root, "content", "fr", "index.md")); !os.IsNotExist(err) {
		t.Fatal("nothing may be written for a failed page")
	}
}

type alwaysBrokenProvider struct{}

func (alwaysBrokenProvider) Name() string  { return "broken" }
func (alwaysBrokenProvider) Model() string { return "test" }
func (alwaysBrokenProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	result := map[string]string{}
	for _, segment := range request.Segments {
		result[segment.ID] = "FR no protected content"
	}
	return result, nil
}
