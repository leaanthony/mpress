package contribute

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoverReadsGeneratedContributionMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><meta name="mpress:repository" content="https://github.com/example/docs.git"><meta name="mpress:branch" content="next"><meta name="mpress:source" content="docs/guide.md"><meta name="mpress:route" content="/guide/">`))
	}))
	defer server.Close()
	metadata, err := Discover(context.Background(), server.Client(), server.URL+"/guide/#install")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Repository != "https://github.com/example/docs.git" || metadata.Branch != "next" || metadata.Source != "docs/guide.md" || metadata.Route != "/guide/" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestDiscoverRejectsOrdinaryPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<h1>Documentation</h1>")) }))
	defer server.Close()
	if _, err := Discover(context.Background(), server.Client(), server.URL); err == nil || !strings.Contains(err.Error(), "does not advertise") {
		t.Fatalf("expected missing metadata error, got %v", err)
	}
}

func TestResolveAcceptsRepositoryWithoutOpeningItAsAPage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("repository target should not make an HTTP request")
		return nil, nil
	})}
	metadata, err := Resolve(context.Background(), client, "https://github.com/example/docs.git", "next")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Repository != "https://github.com/example/docs.git" || metadata.Branch != "next" || metadata.Route != "/" || metadata.SiteURL != "" {
		t.Fatalf("unexpected repository metadata: %#v", metadata)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestPrepareClonesAndReusesAContributionCheckout(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "mpress.yaml"), []byte("site:\n  title: Docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-b", "main", source}, {"-C", source, "config", "user.email", "mpress@example.test"}, {"-C", source, "config", "user.name", "M-Press Test"}, {"-C", source, "add", "."}, {"-C", source, "commit", "-m", "Initial"}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	destination := filepath.Join(t.TempDir(), "checkout")
	metadata := Metadata{Repository: source, Branch: "main"}
	root, branch, reused, err := Prepare(context.Background(), metadata, destination, time.Date(2026, 8, 6, 15, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if reused || branch != "contribute/20260806-153000" || root != destination {
		t.Fatalf("unexpected prepared checkout: root=%q branch=%q reused=%v", root, branch, reused)
	}
	root, branch, reused, err = Prepare(context.Background(), metadata, destination, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !reused || branch != "contribute/20260806-153000" || root != destination {
		t.Fatalf("existing contribution checkout was not reused: root=%q branch=%q reused=%v", root, branch, reused)
	}
}

func TestDetectGuideUsesConfiguredThenConventionalPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github", "CONTRIBUTING.md"), []byte("guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DetectGuide(root, ""); got != ".github/CONTRIBUTING.md" {
		t.Fatalf("detected guide = %q", got)
	}
	if err := os.WriteFile(filepath.Join(root, "PROJECT.md"), []byte("project guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DetectGuide(root, "PROJECT.md"); got != "PROJECT.md" {
		t.Fatalf("configured guide = %q", got)
	}
}
