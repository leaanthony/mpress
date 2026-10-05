package dev

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	docversion "github.com/leaanthony/mpress/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAuthoringVersionsRejectExternalStore(t *testing.T) {
	for _, action := range []string{"list", "verify", "remove", "current"} {
		t.Run(action, func(t *testing.T) {
			root, outside := authoringFixture(t), t.TempDir()
			server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "version-boundary"})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = server.Close() })
			body := "external secret"
			sum := sha256.Sum256([]byte(body))
			data, err := json.Marshal(docversion.Manifest{SchemaVersion: 1, Version: "v1", Files: map[string]string{"index.html": hex.EncodeToString(sum[:])}})
			if err != nil {
				t.Fatal(err)
			}
			writeDevFixture(t, outside, "v1/index.html", body)
			writeDevFixture(t, outside, "v1/mpress-version.json", string(data))
			if err := os.MkdirAll(filepath.Dir(server.cfg.ArtifactsPath(root)), 0755); err != nil {
				t.Fatal(err)
			}
			devSymlink(t, outside, server.cfg.ArtifactsPath(root))
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			method := http.MethodPost
			if action == "list" {
				method = http.MethodGet
			}
			response := requestJSON(t, method, host.URL+"/__mpress/api/versions", versionActionPayload{Action: action, Label: "v1"}, "version-boundary")
			assertConfinementRejected(t, response)
			if action == "list" {
				client := mcp.NewClient(&mcp.Implementation{Name: "version-boundary", Version: "test"}, nil)
				session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: host.URL + "/__mpress/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: host.Client().Transport, token: "version-boundary"}}, DisableStandaloneSSE: true}, nil)
				if err != nil {
					t.Fatal(err)
				}
				defer session.Close()
				result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "list_versions"})
				if err == nil && !result.IsError {
					t.Error("MCP listed versions through an external store")
				}
			}
			assertDevFileUnchanged(t, outside, "v1/index.html", body)
			assertDevFileUnchanged(t, outside, "v1/mpress-version.json", string(data))
		})
	}
}
