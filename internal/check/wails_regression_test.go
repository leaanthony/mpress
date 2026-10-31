package check_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/site"
)

func buildWailsFixture(t *testing.T) (string, config.Config) {
	t.Helper()
	root := t.TempDir()
	content, err := filepath.Abs("../../testdata/wails-links/content")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(root, "content"), os.DirFS(content)); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Site.Title = "Wails link regression"
	cfg.Site.BaseURL = "https://docs.example.test"
	cfg.Site.Languages = []string{"en", "fr", "ko"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français", "ko": "한국어"}
	cfg.Site.DefaultAtRoot = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := site.Build(root, site.BuildOptions{}); err != nil {
		t.Fatal(err)
	}
	return root, cfg
}

type githubFixtureTransport struct{ requests map[string]int }

func (tr *githubFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != "github.com" && req.URL.Host != "discord.gg" {
		return nil, fmt.Errorf("unexpected host %s", req.URL.Host)
	}
	tr.requests[req.URL.String()]++
	code := 200
	if strings.Contains(req.URL.Path, "/releases/v3.0.0/") || strings.Contains(req.URL.Path, "/v3/examples/multi-window") {
		code = 404
	}
	return &http.Response{StatusCode: code, Status: fmt.Sprintf("%d", code), Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
}

func TestWailsHistoricalExternalRegression(t *testing.T) {
	root, cfg := buildWailsFixture(t)
	tr := &githubFixtureTransport{requests: map[string]int{}}
	options := check.ExternalOptions{BaseURL: cfg.Site.BaseURL, Timeout: time.Second, MaxRedirects: 10, Client: &http.Client{Transport: tr}, Hosts: []string{"github.com"}}
	report, err := check.External(context.Background(), cfg.OutputPath(root), options)
	if err != nil {
		t.Fatal(err)
	}
	broken, controls := 0, 0
	for _, link := range report.Links {
		if tr.requests[link.URL] != 1 {
			t.Fatalf("not deduplicated: %+v", tr.requests)
		}
		switch link.Status {
		case "broken":
			broken++
			if len(link.Pages) != 3 {
				t.Fatalf("lost locale references: %+v", link)
			}
			for _, locale := range []string{"features/", "fr/features/", "ko/features/"} {
				found := false
				for _, p := range link.Pages {
					if strings.HasPrefix(p, locale) {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing %s: %+v", locale, link)
				}
			}
		case "ok":
			controls++
		default:
			t.Fatalf("unexpected result: %+v", link)
		}
	}
	if broken != 3 || controls != 2 || report.ExitCode() != 1 {
		t.Fatalf("broken=%d controls=%d report=%+v", broken, controls, report)
	}
	// A branch substitution alone cannot repair the nonexistent example. Even
	// after every release reference becomes a tag URL the audit must still fail.
	if err := filepath.WalkDir(cfg.OutputPath(root), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(path, bytes.ReplaceAll(b, []byte("/releases/v3.0.0/"), []byte("/v3.0.0/")), 0600)
	}); err != nil {
		t.Fatal(err)
	}
	report, err = check.External(context.Background(), cfg.OutputPath(root), options)
	if err != nil {
		t.Fatal(err)
	}
	broken = 0
	for _, link := range report.Links {
		if link.Status == "broken" {
			broken++
			if !strings.HasSuffix(link.URL, "/multi-window") {
				t.Fatalf("%+v", link)
			}
		}
	}
	if broken != 1 {
		t.Fatalf("missed nonexistent directory: %+v", report)
	}
}
