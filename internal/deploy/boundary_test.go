package deploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
)

const deploySentinel = "EXTERNAL-DEPLOY-SENTINEL"

func writeDeployFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	if _, err := io.WriteString(w, body); err != nil {
		t.Errorf("write response: %v", err)
	}
}

func closeTestFiles(t *testing.T, files *projectfs.FS) {
	t.Helper()
	t.Cleanup(func() {
		if err := files.Close(); err != nil {
			t.Errorf("close test root: %v", err)
		}
	})
}

func writeDeploymentResponse(t *testing.T, w http.ResponseWriter) {
	writeJSON(t, w, `{"success":true,"result":{"id":"deployment-1","url":"https://preview.example.pages.dev"}}`)
}

func TestBoundaryCloudflarePagesRejectsExternalAssetSwap(t *testing.T) {
	output, outside := t.TempDir(), t.TempDir()
	writeDeployFile(t, output, "index.html", "<h1>safe</h1>")
	writeDeployFile(t, output, "app.css", "body{color:#123}")
	writeDeployFile(t, outside, "secret.txt", deploySentinel)
	probe := filepath.Join(output, "symlink-probe")
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), probe); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	hash := assetHash("app.css", []byte("body{color:#123}"))
	link := filepath.Join(output, "app.css")

	sentinelSent := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs":
			writeJSON(t, w, `{"success":true,"result":{"name":"docs"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs/upload-token":
			writeJSON(t, w, `{"success":true,"result":{"jwt":"upload-jwt"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/check-missing":
			// uploadAssets re-reads the advertised asset after this callback.
			if err := os.Remove(link); err != nil {
				t.Errorf("remove asset: %v", err)
			}
			if err := os.Symlink(filepath.Join(outside, "secret.txt"), link); err != nil {
				t.Errorf("swap asset: %v", err)
			}
			writeJSON(t, w, `{"success":true,"result":["`+hash+`"]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upload":
			var payload []struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode upload: %v", err)
			}
			for _, item := range payload {
				data, err := base64.StdEncoding.DecodeString(item.Value)
				if err == nil && bytes.Contains(data, []byte(deploySentinel)) {
					sentinelSent = true
				}
			}
			writeJSON(t, w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upsert-hashes":
			writeJSON(t, w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/account/pages/projects/docs/deployments":
			writeDeploymentResponse(t, w)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err := CloudflarePages(context.Background(), output, "cloudflare", "preview", config.DeployTarget{
		Provider: "cloudflare-pages", AccountID: "account", Project: "docs",
	}, Options{Token: "test-token", APIURL: server.URL, AssetsURL: server.URL, Client: server.Client()})
	if sentinelSent {
		t.Fatal("uploaded content read through an external symlink")
	}
	if err == nil {
		t.Fatal("expected a swapped asset to be rejected")
	}
	if !errors.Is(err, projectfs.ErrOutside) {
		t.Fatalf("expected a rooted read rejection, got %v", err)
	}
}

func TestBoundaryCloudflarePagesRejectsAssetChangedAfterCollection(t *testing.T) {
	output := t.TempDir()
	writeDeployFile(t, output, "index.html", "<h1>safe</h1>")
	writeDeployFile(t, output, "app.css", "body{color:#123}")
	path := filepath.Join(output, "app.css")
	hash := assetHash("app.css", []byte("body{color:#123}"))
	changedSent := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs":
			writeJSON(t, w, `{"success":true,"result":{"name":"docs"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs/upload-token":
			writeJSON(t, w, `{"success":true,"result":{"jwt":"upload-jwt"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/check-missing":
			if err := os.WriteFile(path, []byte("body{color:#456}"), 0o644); err != nil {
				t.Errorf("replace asset: %v", err)
			}
			writeJSON(t, w, `{"success":true,"result":["`+hash+`"]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upload":
			var payload []struct {
				Value string `json:"value"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode upload: %v", err)
			}
			for _, item := range payload {
				data, err := base64.StdEncoding.DecodeString(item.Value)
				if err == nil && string(data) != "body{color:#123}" {
					changedSent = true
				}
			}
			writeJSON(t, w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upsert-hashes":
			writeJSON(t, w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/account/pages/projects/docs/deployments":
			writeDeploymentResponse(t, w)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err := CloudflarePages(context.Background(), output, "cloudflare", "preview", config.DeployTarget{
		Provider: "cloudflare-pages", AccountID: "account", Project: "docs",
	}, Options{Token: "test-token", APIURL: server.URL, AssetsURL: server.URL, Client: server.Client()})
	if changedSent {
		t.Fatal("uploaded content that no longer matched its advertised hash")
	}
	if err == nil {
		t.Fatal("expected changed asset data to be rejected")
	}
}

func TestBoundaryCloudflarePagesOwnsSelectedAliasRoot(t *testing.T) {
	base := t.TempDir()
	writeDeployFile(t, filepath.Join(base, "site"), "index.html", "<h1>safe</h1>")
	writeDeployFile(t, filepath.Join(base, "site"), "assets/app.css", "body{color:#123}")
	alias := filepath.Join(base, "alias")
	if err := os.Symlink("site", alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs":
			writeJSON(t, w, `{"success":true,"result":{"name":"docs"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs/upload-token":
			writeJSON(t, w, `{"success":true,"result":{"jwt":"upload-jwt"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/check-missing":
			writeJSON(t, w, `{"success":true,"result":[]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upsert-hashes":
			writeJSON(t, w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/account/pages/projects/docs/deployments":
			writeDeploymentResponse(t, w)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	result, err := CloudflarePages(context.Background(), alias, "cloudflare", "preview", config.DeployTarget{
		Provider: "cloudflare-pages", AccountID: "account", Project: "docs",
	}, Options{Token: "test-token", APIURL: server.URL, AssetsURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reused != 2 {
		t.Fatalf("selected output alias contributed %d assets, want 2", result.Reused)
	}
}

func TestBoundaryNetlifyPagesOwnsSelectedAliasRoot(t *testing.T) {
	base := t.TempDir()
	writeDeployFile(t, filepath.Join(base, "site"), "index.html", "<h1>safe</h1>")
	writeDeployFile(t, filepath.Join(base, "site"), "assets/app.css", "body{color:#123}")
	alias := filepath.Join(base, "alias")
	if err := os.Symlink("site", alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sites/site-1":
			writeJSON(t, w, `{"id":"site-1","name":"docs","deploy_url":"https://draft.netlify.app"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/sites/site-1/deploys":
			writeJSON(t, w, `{"id":"deploy-1","state":"ready","deploy_url":"https://draft.netlify.app"}`)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	result, err := NetlifyPages(context.Background(), alias, "netlify", "preview", config.DeployTarget{
		Provider: "netlify", Project: "site-1",
	}, NetlifyOptions{Token: "netlify-token", APIURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Uploaded != 2 {
		t.Fatalf("selected output alias contributed %d files, want 2", result.Uploaded)
	}
}

func TestBoundaryNetlifyPagesRejectsExternalFileSymlink(t *testing.T) {
	output, outside := t.TempDir(), t.TempDir()
	writeDeployFile(t, output, "index.html", "<h1>safe</h1>")
	writeDeployFile(t, outside, "secret.txt", deploySentinel)
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(output, "leak.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	requested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = true
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/sites/site-1/deploys" {
			writeJSON(t, w, `{"id":"deploy-1","state":"ready","deploy_url":"https://draft.netlify.app"}`)
			return
		}
		writeJSON(t, w, `{"id":"site-1","name":"docs","deploy_url":"https://draft.netlify.app"}`)
	}))
	defer server.Close()

	_, err := NetlifyPages(context.Background(), output, "netlify", "preview", config.DeployTarget{
		Provider: "netlify", Project: "site-1",
	}, NetlifyOptions{Token: "netlify-token", APIURL: server.URL, Client: server.Client()})
	if err == nil {
		t.Fatal("expected a non-regular output entry to be rejected")
	}
	if requested {
		t.Fatal("provider was contacted for an unreadable output")
	}
}

func TestBoundaryCloudflarePagesBorrowsPinnedOutput(t *testing.T) {
	base := t.TempDir()
	site := filepath.Join(base, "site")
	writeDeployFile(t, site, "index.html", "<h1>safe</h1>")
	project, err := projectfs.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	closeTestFiles(t, project)
	output, err := project.Sub("site")
	if err != nil {
		t.Fatal(err)
	}
	closeTestFiles(t, output)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs":
			writeJSON(t, w, `{"success":true,"result":{"name":"docs"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/accounts/account/pages/projects/docs/upload-token":
			writeJSON(t, w, `{"success":true,"result":{"jwt":"upload-jwt"}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/check-missing":
			writeJSON(t, w, `{"success":true,"result":[]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/pages/assets/upsert-hashes":
			writeJSON(t, w, `{"success":true,"result":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/accounts/account/pages/projects/docs/deployments":
			writeDeploymentResponse(t, w)
		default:
			http.Error(w, "unexpected request", http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err = CloudflarePages(context.Background(), site, "cloudflare", "preview", config.DeployTarget{
		Provider: "cloudflare-pages", AccountID: "account", Project: "docs",
	}, Options{Token: "test-token", APIURL: server.URL, AssetsURL: server.URL, Client: server.Client(), Files: output})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := output.ReadFile("index.html"); err != nil {
		t.Fatalf("borrowed output was closed by the provider: %v", err)
	}
}

func TestBoundaryNetlifyPagesBorrowsOutputRoot(t *testing.T) {
	base := t.TempDir()
	writeDeployFile(t, filepath.Join(base, "site"), "index.html", "<h1>safe</h1>")
	writeDeployFile(t, filepath.Join(base, "site"), "assets/app.css", "body{color:#123}")
	project, err := projectfs.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	closeTestFiles(t, project)
	output, err := project.Sub("site")
	if err != nil {
		t.Fatal(err)
	}
	closeTestFiles(t, output)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/sites/site-1" {
			writeJSON(t, w, `{"id":"site-1","name":"docs","deploy_url":"https://draft.netlify.app"}`)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/sites/site-1/deploys" {
			writeJSON(t, w, `{"id":"deploy-1","state":"ready","deploy_url":"https://draft.netlify.app"}`)
			return
		}
		http.Error(w, "unexpected request", http.StatusNotFound)
	}))
	defer server.Close()

	result, err := NetlifyPages(context.Background(), filepath.Join(base, "site"), "netlify", "preview", config.DeployTarget{
		Provider: "netlify", Project: "site-1",
	}, NetlifyOptions{Token: "netlify-token", APIURL: server.URL, Client: server.Client(), Files: output})
	if err != nil {
		t.Fatal(err)
	}
	if result.Uploaded != 2 {
		t.Fatalf("borrowed output contributed %d files, want 2", result.Uploaded)
	}
	if _, err := output.ReadFile("index.html"); err != nil {
		t.Fatalf("borrowed output was closed by the provider: %v", err)
	}
}
