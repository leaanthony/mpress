package check

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func externalOptions() ExternalOptions {
	return ExternalOptions{Timeout: time.Second, MaxRedirects: 10}
}

func TestExternalHTTPOutcomes(t *testing.T) {
	tests := []struct {
		path, status   string
		code, attempts int
	}{
		{"/ok", "ok", 0, 1}, {"/missing", "broken", 1, 1}, {"/gone", "broken", 1, 1},
		{"/redirect", "ok", 0, 1}, {"/redirect-missing", "broken", 1, 1},
		{"/loop", "inconclusive", 2, 2}, {"/rate", "inconclusive", 2, 2},
		{"/retry-after", "inconclusive", 2, 1}, {"/unavailable", "inconclusive", 2, 2},
		{"/forbidden", "inconclusive", 2, 1}, {"/timeout", "inconclusive", 2, 2},
		{"/recover", "ok", 0, 2},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := requests.Add(1)
				switch r.URL.Path {
				case "/missing":
					w.WriteHeader(404)
				case "/gone":
					w.WriteHeader(410)
				case "/redirect":
					http.Redirect(w, r, "/ok", http.StatusFound)
				case "/redirect-missing":
					http.Redirect(w, r, "/missing", http.StatusMovedPermanently)
				case "/loop":
					http.Redirect(w, r, "/loop", http.StatusFound)
				case "/rate":
					w.WriteHeader(429)
				case "/retry-after":
					w.Header().Set("Retry-After", "120")
					w.WriteHeader(429)
				case "/unavailable":
					w.WriteHeader(503)
				case "/forbidden":
					w.WriteHeader(403)
				case "/timeout":
					<-r.Context().Done()
				case "/recover":
					if n == 1 {
						w.WriteHeader(503)
					}
				}
			}))
			defer server.Close()
			root := t.TempDir()
			// Distinct fragments and repeats share one HTTP probe sequence.
			html := fmt.Sprintf(`<a href="%s%s#one">a</a><img src="%s%s#two"><a href="%s%s#one">b</a>`, server.URL, test.path, server.URL, test.path, server.URL, test.path)
			if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(html), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(root, "ko"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "ko/index.html"), []byte(html), 0600); err != nil {
				t.Fatal(err)
			}
			options := externalOptions()
			options.Retries = 1
			options.Timeout = 100 * time.Millisecond
			options.MaxRedirects = 2
			report, err := External(context.Background(), root, options)
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Links) != 1 {
				t.Fatalf("%+v", report)
			}
			link := report.Links[0]
			if link.Status != test.status || report.ExitCode() != test.code || link.Attempts != test.attempts || strings.Join(link.Pages, ",") != "index.html,ko/index.html" {
				t.Fatalf("%+v exit=%d", link, report.ExitCode())
			}
			expected := int32(test.attempts)
			if strings.HasPrefix(test.path, "/redirect") {
				expected = 2
			}
			if test.path == "/loop" {
				expected = 6
			}
			if requests.Load() != expected {
				t.Fatalf("requests=%d want=%d", requests.Load(), expected)
			}
		})
	}
}

func TestExternalSelectionAndErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(`<a href="/local">local</a><a href="https://docs.example.test/page">own</a><a href="mailto:a@b.test">mail</a><a href="https://other.test/">other</a>`), 0600); err != nil {
		t.Fatal(err)
	}
	options := externalOptions()
	options.BaseURL = "https://docs.example.test"
	options.Hosts = []string{"github.com"}
	report, err := External(context.Background(), root, options)
	if err != nil || len(report.Links) != 0 {
		t.Fatalf("%+v %v", report, err)
	}
	options.Retries = -1
	if _, err := External(context.Background(), root, options); err == nil {
		t.Fatal("accepted invalid retries")
	}
	options = externalOptions()
	if _, err := External(context.Background(), filepath.Join(root, "missing"), options); err == nil {
		t.Fatal("accepted missing output")
	}
	for _, links := range [][]ExternalLink{{{Status: "inconclusive"}, {Status: "broken"}}, {{Status: "broken"}, {Status: "inconclusive"}}} {
		if (ExternalReport{Links: links}).ExitCode() != 1 {
			t.Fatal("broken must take precedence")
		}
	}
}

func TestExternalURLPrefixes(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer server.Close()
	root := t.TempDir()
	body := fmt.Sprintf(`<a href="%s/tree/release/example">directory</a><a href="%s/blob/tag/file.go">file</a><a href="%s/pull/1">changelog</a><a href="%s/profile">credit</a>`, server.URL, server.URL, server.URL, server.URL)
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	options := externalOptions()
	options.URLPrefixes = []string{server.URL + "/tree/", server.URL + "/blob/"}
	report, err := External(context.Background(), root, options)
	if err != nil || len(report.Links) != 2 || calls.Load() != 2 || report.ExitCode() != 0 {
		t.Fatalf("%+v calls=%v err=%v", report, calls.Load(), err)
	}
	options.URLPrefixes = []string{"/relative/"}
	if _, err := External(context.Background(), root, options); err == nil {
		t.Fatal("accepted relative URL prefix")
	}
}
