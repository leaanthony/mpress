package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAuthoringRESTRejectsExternalSymlinkReadsAndWrites(t *testing.T) {
	for _, kind := range []string{"parent", "final", "dangling"} {
		for _, readOnly := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "/authoring", true: "/read-only"}[readOnly], func(t *testing.T) {
				root, outside := authoringFixture(t), t.TempDir()
				server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: !readOnly, Token: "confinement-test"})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = server.Close() })
				original := "# External secret\n"
				writeDevFixture(t, outside, "external.md", original)
				name, target := "content/linked.md", filepath.Join(outside, "external.md")
				if kind == "parent" {
					name, target = "content/linked/external.md", outside
				}
				if kind == "dangling" {
					target = filepath.Join(outside, "missing.md")
				}
				link := "content/linked.md"
				if kind == "parent" {
					link = "content/linked"
				}
				devSymlink(t, target, filepath.Join(root, link))
				host := httptest.NewServer(server.Handler())
				defer host.Close()
				response, err := http.Get(host.URL + "/__mpress/api/file?path=" + url.QueryEscape(name))
				if err != nil {
					t.Fatal(err)
				}
				assertConfinementRejected(t, response)
				for _, action := range []struct {
					method, endpoint string
					payload          any
				}{
					{http.MethodPut, "file", filePayload{Path: name, Content: "# Changed\n", Revision: revision([]byte(original))}},
					{http.MethodPost, "preview", previewPayload{Path: name, Content: "# Preview\n", Revision: revision([]byte(original))}},
					{http.MethodGet, "blocks?path=" + url.QueryEscape(name), nil},
					{http.MethodPut, "block", blockPayload{Path: name, Start: 0, End: len(original), Markdown: "# Changed\n", Revision: revision([]byte(original))}},
				} {
					response := requestJSON(t, action.method, host.URL+"/__mpress/api/"+action.endpoint, action.payload, "confinement-test")
					assertConfinementRejected(t, response)
				}
				assertDevFileUnchanged(t, outside, "external.md", original)
				if _, err := os.Stat(filepath.Join(outside, "missing.md")); !os.IsNotExist(err) {
					t.Error("dangling external target was created")
				}
				if _, err := os.Lstat(filepath.Join(root, link)); err != nil {
					t.Errorf("rejected link changed: %v", err)
				}
			})
		}
	}
}

func TestAuthoringMCPRejectsExternalSymlinkPaths(t *testing.T) {
	for _, authoring := range []bool{false, true} {
		t.Run(map[bool]string{false: "read-only", true: "authoring"}[authoring], func(t *testing.T) {
			root, outside := authoringFixture(t), t.TempDir()
			server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: authoring, Token: "confinement-test"})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = server.Close() })
			writeDevFixture(t, outside, "external.md", "# External secret\n")
			devSymlink(t, outside, filepath.Join(root, "content/linked"))
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			client := mcp.NewClient(&mcp.Implementation{Name: "confinement-test", Version: "test"}, nil)
			session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: host.URL + "/__mpress/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: host.Client().Transport, token: "confinement-test"}}, DisableStandaloneSSE: true}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			for _, tool := range []string{"read_file", "write_file"} {
				arguments := map[string]any{"path": "content/linked/external.md"}
				if tool == "write_file" {
					arguments["content"], arguments["revision"] = "# Changed\n", revision([]byte("# External secret\n"))
				}
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: arguments})
				if err == nil && !result.IsError {
					t.Errorf("MCP %s accepted external path", tool)
				}
			}
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "write_file", Arguments: map[string]any{"path": "content/linked/new.md", "content": "# New\n"}})
			if err == nil && !result.IsError {
				t.Error("MCP created external file")
			}
			assertDevFileUnchanged(t, outside, "external.md", "# External secret\n")
			if _, err := os.Stat(filepath.Join(outside, "new.md")); !os.IsNotExist(err) {
				t.Error("external file exists")
			}
		})
	}
}

func TestAuthoringUploadRejectsExternalStaticAndContentParents(t *testing.T) {
	for _, kind := range []string{"brand", "import"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := authoringFixture(t), t.TempDir()
			server, err := NewServer(root, Options{Authoring: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = server.Close() })
			endpoint, name, data := "brand-asset", "logo.svg", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`
			if kind == "brand" {
				devSymlink(t, outside, filepath.Join(root, "static/brand"))
			} else {
				if err := os.Rename(filepath.Join(root, "content"), filepath.Join(root, "original-content")); err != nil {
					t.Fatal(err)
				}
				devSymlink(t, outside, filepath.Join(root, "content"))
				endpoint, name, data = "import", "uploaded.md", "# Uploaded\n"
			}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if err := writer.WriteField("slot", "logoLight"); err != nil {
				t.Fatal(err)
			}
			part, err := writer.CreateFormFile("file", name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(part, data); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			request, err := http.NewRequest(http.MethodPost, host.URL+"/__mpress/api/"+endpoint, &body)
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", writer.FormDataContentType())
			response, err := host.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			assertConfinementRejected(t, response)
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Error("upload wrote outside project")
			}
		})
	}
}

func TestAuthoringBackupsRejectExternalParentWithoutChangingSource(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if err := os.MkdirAll(filepath.Join(root, ".mpress"), 0755); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, outside, filepath.Join(root, ".mpress/backups"))
	path := filepath.Join(root, "content/index.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := server.writeSource(path, []byte("# Changed\n"), revision(original)); err == nil {
		t.Error("backup escaped project")
	}
	assertDevFileUnchanged(t, root, "content/index.md", string(original))
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Error("external backup created")
	}
}

func TestNewServerPreservesUnsafePreviewPools(t *testing.T) {
	for _, kind := range []string{"external-parent", "external-final", "content", "nested-content", "static", "translation-state", "version-store", "published-output", "navigation", "glossary", "style-guide", "contributor-guide"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := authoringFixture(t), t.TempDir()
			cfg, err := config.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			pool := ".mpress/live-preview"
			if kind == "external-parent" {
				writeDevFixture(t, outside, "live-preview/sentinel", "must survive")
				devSymlink(t, outside, filepath.Join(root, ".mpress"))
			} else if kind == "external-final" {
				writeDevFixture(t, outside, "sentinel", "must survive")
				if err := os.MkdirAll(filepath.Join(root, ".mpress"), 0755); err != nil {
					t.Fatal(err)
				}
				devSymlink(t, outside, filepath.Join(root, pool))
			} else {
				switch kind {
				case "content":
					cfg.Build.ContentDir = pool
				case "nested-content":
					cfg.Build.ContentDir = pool + "/docs"
				case "static":
					cfg.Build.StaticDir = pool + "/assets"
				case "translation-state":
					cfg.Translation.StateDir = pool + "/translations"
				case "version-store":
					cfg.Version.Artifacts = pool + "/versions"
				case "published-output":
					cfg.Build.OutputDir = pool + "/site"
				case "navigation":
					cfg.Build.NavFile = "../" + pool + "/navigation.yaml"
				case "glossary":
					cfg.Translation.Glossary = pool + "/glossary.yaml"
				case "style-guide":
					cfg.Translation.StyleGuide = pool + "/writing.md"
				case "contributor-guide":
					cfg.Contribution.Guide = pool + "/guide.md"
				}
				if err := config.Save(root, cfg); err != nil {
					t.Fatal(err)
				}
				writeDevFixture(t, root, pool+"/sentinel", "must survive")
			}
			if _, err := NewServer(root, Options{Authoring: true}); err == nil {
				t.Error("unsafe preview pool accepted")
			}
			if kind == "external-parent" {
				assertDevFileUnchanged(t, outside, "live-preview/sentinel", "must survive")
			} else if kind == "external-final" {
				assertDevFileUnchanged(t, outside, "sentinel", "must survive")
				if _, err := os.Lstat(filepath.Join(root, pool)); err != nil {
					t.Error("external pool link removed")
				}
			} else {
				assertDevFileUnchanged(t, root, pool+"/sentinel", "must survive")
			}
		})
	}
}

func TestAuthoringConfigurationAndTreeDoNotFollowExternalLinks(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	writeDevFixture(t, outside, "external-secret.md", "# Secret\n")
	devSymlink(t, outside, filepath.Join(root, "content/linked"))
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	response, err := http.Get(host.URL + "/__mpress/api/tree")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	response.Body.Close()
	if strings.Contains(body, "external-secret.md") {
		t.Error("tree exposed external filenames")
	}
	data, err := os.ReadFile(filepath.Join(root, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	writeDevFixture(t, outside, config.Filename, string(data))
	if err := os.Remove(filepath.Join(root, config.Filename)); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, filepath.Join(outside, config.Filename), filepath.Join(root, config.Filename))
	response, err = http.Get(host.URL + "/__mpress/api/config")
	if err != nil {
		t.Fatal(err)
	}
	assertConfinementRejected(t, response)
	if _, err := NewServer(root, Options{Authoring: true}); err == nil {
		t.Error("startup read external configuration")
	}
}

func devSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		if probeErr := os.Symlink(t.TempDir(), filepath.Join(t.TempDir(), "probe")); probeErr != nil {
			t.Skipf("symlinks unavailable: %v", probeErr)
		}
		t.Fatal(err)
	}
}

func assertConfinementRejected(t *testing.T, response *http.Response) {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		t.Errorf("external path accepted with HTTP %d", response.StatusCode)
	}
}

func assertDevFileUnchanged(t *testing.T, root, name, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil || string(data) != want {
		t.Errorf("sentinel %s changed (%d bytes, error %v)", name, len(data), err)
	}
}

func TestAuthoringRetainsSafeAliasesAndCreatesMissingFiles(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		t.Run(map[bool]string{false: "relative", true: "absolute"}[absolute], func(t *testing.T) {
			root := authoringFixture(t)
			writeDevFixture(t, root, "shared/original.md", "# Original\n")
			target := "../shared"
			if absolute {
				target = filepath.Join(root, "shared")
			}
			devSymlink(t, target, filepath.Join(root, "content/alias"))
			devSymlink(t, "../content", filepath.Join(root, "shared/loop"))
			server, err := NewServer(root, Options{Authoring: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = server.Close() })

			host := httptest.NewServer(server.Handler())
			defer host.Close()
			file := getFile(t, host.URL, "content/alias/original.md")
			if file.Path != "content/alias/original.md" || file.Content != "# Original\n" {
				t.Error("safe alias read lost logical path or content")
			}
			response := requestJSON(t, http.MethodPut, host.URL+"/__mpress/api/file", filePayload{Path: file.Path, Content: "# Updated\n", Revision: file.Revision}, "")
			if response.StatusCode != http.StatusOK {
				t.Fatalf("safe alias update: %d", response.StatusCode)
			}
			response.Body.Close()
			assertDevFileUnchanged(t, root, "shared/original.md", "# Updated\n")
			response = requestJSON(t, http.MethodPut, host.URL+"/__mpress/api/file", filePayload{Path: "content/alias/nested/new.md", Content: "# New\n"}, "")
			if response.StatusCode != http.StatusOK {
				t.Fatalf("safe creation: %d", response.StatusCode)
			}
			response.Body.Close()
			assertDevFileUnchanged(t, root, "shared/nested/new.md", "# New\n")
			response, err = http.Get(host.URL + "/__mpress/api/tree")
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatalf("safe cyclic tree: %d", response.StatusCode)
			}
			body := readBody(response)
			response.Body.Close()
			if !strings.Contains(body, `"name":"alias"`) || !strings.Contains(body, "content/alias/nested/new.md") {
				t.Error("tree lost safe alias names")
			}
		})
	}
}

func TestAuthoringPreviewCleanupRechecksRedirectedPool(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	writeDevFixture(t, outside, "live-preview/build/sentinel", "must survive")
	devSymlink(t, outside, filepath.Join(root, ".mpress"))
	if err := server.removePreview(filepath.Join(server.previewDir, "build")); err == nil {
		t.Error("cleanup followed redirected pool")
	}
	assertDevFileUnchanged(t, outside, "live-preview/build/sentinel", "must survive")
}

func TestAuthoringServingRejectsExternalPreviewAndSiteFiles(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	writeDevFixture(t, outside, "secret.html", "<h1>External secret</h1>")
	writeDevFixture(t, root, "site/index.html", "<h1>Home</h1>")
	if err := os.MkdirAll(filepath.Join(root, ".mpress/live-preview/build"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{"site/linked.html", ".mpress/live-preview/build/linked.html"} {
		devSymlink(t, filepath.Join(outside, "secret.html"), filepath.Join(root, link))
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	for _, route := range []string{"/linked.html", "/__mpress/preview/build/linked.html"} {
		response, err := http.Get(host.URL + route)
		if err != nil {
			t.Fatal(err)
		}
		assertConfinementRejected(t, response)
	}
}

func TestAuthoringRebuildDoesNotWriteCacheThroughExternalParent(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if err := os.MkdirAll(filepath.Join(root, ".mpress"), 0755); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, outside, filepath.Join(root, ".mpress/cache"))
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	file := getFile(t, host.URL, "content/index.md")
	response := requestJSON(t, http.MethodPut, host.URL+"/__mpress/api/file", filePayload{Path: file.Path, Content: "# Updated\n", Revision: file.Revision}, "")
	response.Body.Close()
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Error("authoring rebuild wrote cache outside project")
	}
}

func TestAuthoringPreviewRejectsOtherExternalInputs(t *testing.T) {
	for _, kind := range []string{"content", "static", "navigation", "custom-css"} {
		t.Run(kind, func(t *testing.T) {
			root, outside := authoringFixture(t), t.TempDir()
			if kind == "custom-css" {
				cfg, err := config.Load(root)
				if err != nil {
					t.Fatal(err)
				}
				cfg.Build.CustomCSS = "custom.css"
				if err := config.Save(root, cfg); err != nil {
					t.Fatal(err)
				}
			}
			server, err := NewServer(root, Options{Authoring: true})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = server.Close() })

			name, link, source := "other.md", "content/other.md", "# External secret\n"
			switch kind {
			case "static":
				name, link = "other.txt", "static/other.txt"
			case "navigation":
				name, link, source = "_nav.yaml", "content/_nav.yaml", "- label: External secret\n  link: /\n"
				if err := os.Remove(filepath.Join(root, link)); err != nil {
					t.Fatal(err)
				}
			case "custom-css":
				name, link, source = "custom.css", "custom.css", "/* External secret */\n"
			}
			writeDevFixture(t, outside, name, source)
			devSymlink(t, filepath.Join(outside, name), filepath.Join(root, link))
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			file := getFile(t, host.URL, "content/index.md")
			response := requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/preview", previewPayload{Path: file.Path, Content: "# Preview\n", Revision: file.Revision}, "")
			defer response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				var result struct {
					OK bool `json:"ok"`
				}
				if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if result.OK {
					t.Error("preview published an external input")
				}
			}
			assertDevFileUnchanged(t, outside, name, source)
		})
	}
}
