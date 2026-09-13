package deploy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func TestCloudflarePagesUploadsMissingAssetsAndCreatesPreview(t *testing.T) {
	output := t.TempDir()
	if err := os.WriteFile(filepath.Join(output, "index.html"), []byte("<h1>Hello</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "app.css"), []byte("body{color:#123}"), 0o644); err != nil {
		t.Fatal(err)
	}
	assets, err := collectAssets(output)
	if err != nil {
		t.Fatal(err)
	}
	missingHash := assets[0].Hash
	uploaded := false
	deployed := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" && got != "Bearer upload-jwt" {
			t.Errorf("unexpected authorization header %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs":
			io.WriteString(w, `{"success":true,"result":{"name":"docs"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs/upload-token":
			io.WriteString(w, `{"success":true,"result":{"jwt":"upload-jwt"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/check-missing":
			io.WriteString(w, `{"success":true,"result":["`+missingHash+`"]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upload":
			var payload []struct {
				Key string `json:"key"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode upload: %v", err)
			}
			if len(payload) != 1 || payload[0].Key != missingHash {
				t.Errorf("unexpected uploaded assets: %#v", payload)
			}
			uploaded = true
			io.WriteString(w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upsert-hashes":
			io.WriteString(w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/account/pages/projects/docs/deployments":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("parse deployment: %v", err)
			}
			if branch := r.FormValue("branch"); branch != "mpress-preview" {
				t.Errorf("preview branch = %q", branch)
			}
			if manifest := r.FormValue("manifest"); !strings.Contains(manifest, "/index.html") || !strings.Contains(manifest, "/app.css") {
				t.Errorf("deployment manifest is incomplete: %s", manifest)
			}
			deployed = true
			io.WriteString(w, `{"success":true,"result":{"id":"deployment-1","url":"https://preview.example.pages.dev","environment":"preview"}}`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	result, err := CloudflarePages(context.Background(), output, "cloudflare", "preview", config.DeployTarget{
		Provider: "cloudflare-pages", AccountID: "account", Project: "docs", ProductionBranch: "main",
	}, Options{Token: "test-token", APIURL: server.URL, AssetsURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if !uploaded || !deployed {
		t.Fatalf("upload and deployment were not both completed: upload=%t deploy=%t", uploaded, deployed)
	}
	if result.Uploaded != 1 || result.Reused != 1 || result.URL != "https://preview.example.pages.dev" {
		t.Fatalf("unexpected deployment result: %#v", result)
	}
}

func TestCloudflarePagesRequiresTokenBeforeNetworkAccess(t *testing.T) {
	_, err := CloudflarePages(context.Background(), t.TempDir(), "cloudflare", "preview", config.DeployTarget{Provider: "cloudflare-pages"}, Options{})
	if err == nil || !strings.Contains(err.Error(), "CLOUDFLARE_API_TOKEN") {
		t.Fatalf("expected token guidance, got %v", err)
	}
}
