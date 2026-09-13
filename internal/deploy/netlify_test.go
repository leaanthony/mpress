package deploy

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestNetlifyPagesCreatesDraftZipDeploy(t *testing.T) {
	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(output, "index.html"), []byte("<h1>Hello</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(output, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "assets", "app.css"), []byte("body{color:#123}"), 0o644); err != nil {
		t.Fatal(err)
	}
	created := false
	deployed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer netlify-token" {
			t.Errorf("unexpected authorization header %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sites/docs.example.com":
			http.Error(w, "not found", http.StatusNotFound)
		case r.Method == http.MethodPost && r.URL.Path == "/team/sites/":
			created = true
			io.WriteString(w, `{"id":"site-1","name":"docs","url":"https://docs.netlify.app","deploy_url":"https://draft.netlify.app"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/sites/site-1/deploys":
			if r.URL.Query().Get("draft") != "true" {
				t.Errorf("preview deploy was not marked draft: %s", r.URL.RawQuery)
			}
			if r.Header.Get("Content-Type") != "application/zip" {
				t.Errorf("deploy content type = %q", r.Header.Get("Content-Type"))
			}
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				t.Errorf("deploy body was not a zip: %v", err)
			} else if len(archive.File) != 2 {
				t.Errorf("zip contained %d files, want 2", len(archive.File))
			}
			deployed = true
			io.WriteString(w, `{"id":"deploy-1","state":"ready","deploy_url":"https://draft.netlify.app"}`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	result, err := NetlifyPages(context.Background(), output, "netlify", "preview", config.DeployTarget{Provider: "netlify", AccountID: "team", Project: "docs.example.com"}, NetlifyOptions{Token: "netlify-token", APIURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if !created || !deployed {
		t.Fatalf("site creation and deploy were not both completed: created=%t deployed=%t", created, deployed)
	}
	if result.URL != "https://draft.netlify.app" || result.Deployment != "deploy-1" || result.Uploaded != 2 {
		t.Fatalf("unexpected deployment result: %#v", result)
	}
}

func TestNetlifyPagesRequiresTokenBeforeNetworkAccess(t *testing.T) {
	_, err := NetlifyPages(context.Background(), t.TempDir(), "netlify", "preview", config.DeployTarget{Provider: "netlify", Project: "docs"}, NetlifyOptions{})
	if err == nil || !strings.Contains(err.Error(), "NETLIFY_AUTH_TOKEN") {
		t.Fatalf("expected token guidance, got %v", err)
	}
}
