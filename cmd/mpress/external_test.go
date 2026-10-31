package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

// The same test file can run at the baseline revision: all fixture requests
// are intercepted, and no new checker types are required to prove the gap.
func TestWailsExternalCommandRegression(t *testing.T) {
	root := t.TempDir()
	fixtures, err := filepath.Abs("../../testdata/wails-links/content")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(root, "content"), os.DirFS(fixtures)); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Site.Title = "Wails"
	cfg.Site.BaseURL = "https://docs.example.test"
	cfg.Site.Languages = []string{"en", "fr", "ko"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français", "ko": "한국어"}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if err := build([]string{"--strict"}); err != nil {
		t.Fatal(err)
	}
	if _, err := captureCheckOutput(t, "check"); err != nil {
		t.Fatal(err)
	}
	// Standard transport replacement models GitHub paths without changing the
	// historical documents or replacing their URLs with localhost URLs.
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	requests := map[string]int{}
	http.DefaultTransport = fixtureHTTP(func(r *http.Request) (*http.Response, error) {
		requests[r.URL.String()]++
		status := 200
		if strings.Contains(r.URL.Path, "/releases/v3.0.0/") {
			status = 404
		}
		return &http.Response{StatusCode: status, Status: fmt.Sprint(status), Header: http.Header{}, Body: http.NoBody, Request: r}, nil
	})
	out, err := captureCheckOutput(t, "check", "external", "--host", "github.com", "--retries", "0", "--json")
	var report struct {
		Links []struct {
			URL, Status string
			Pages       []string
		} `json:"links"`
	}
	if parseErr := json.Unmarshal([]byte(out), &report); parseErr != nil {
		t.Fatalf("external audit did not report the historical broken links: err=%v output=%q", err, out)
	}
	broken, valid := 0, 0
	for _, link := range report.Links {
		if requests[link.URL] != 1 {
			t.Fatalf("duplicate requests: %v", requests)
		}
		switch link.Status {
		case "broken":
			broken++
			if len(link.Pages) != 3 {
				t.Fatalf("lost a locale: %+v", link)
			}
		case "ok":
			valid++
		}
	}
	if err == nil || broken != 3 || valid != 2 {
		t.Fatalf("broken=%d valid=%d error=%v output=%s", broken, valid, err, out)
	}
}

type fixtureHTTP func(*http.Request) (*http.Response, error)

func (f fixtureHTTP) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestExternalExitProcess(t *testing.T) {
	if os.Getenv("MPRESS_EXTERNAL_EXIT_CHILD") == "1" {
		os.Args = []string{"mpress", "check", "external", "--json", "--retries", "0"}
		main()
		return
	}
	for _, status := range []int{200, 404, 410, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			root := t.TempDir()
			cfg := config.Default()
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(cfg.OutputPath(root), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(cfg.OutputPath(root), "index.html"), []byte(`<a href="`+server.URL+`">target</a>`), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestExternalExitProcess$")
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "MPRESS_EXTERNAL_EXIT_CHILD=1")
			out, err := cmd.CombinedOutput()
			code := 0
			if err != nil {
				exit, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			want := 0
			if status == 404 || status == 410 {
				want = 1
			} else if status != 200 {
				want = 2
			}
			if code != want {
				t.Fatalf("exit=%d want=%d: %s", code, want, out)
			}
		})
	}
}
