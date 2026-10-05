package dev

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAuthoringMCPCapturePreservesExternalStore(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "capture-boundary"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	writeDevFixture(t, outside, "v1/index.html", "must survive")
	writeDevFixture(t, outside, "v1/mpress-version.json", "foreign manifest")
	if err := os.MkdirAll(filepath.Dir(server.cfg.ArtifactsPath(root)), 0755); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, outside, server.cfg.ArtifactsPath(root))
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "capture-boundary", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: host.URL + "/__mpress/mcp", HTTPClient: &http.Client{Transport: bearerTransport{base: host.Client().Transport, token: "capture-boundary"}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "capture_version", Arguments: map[string]any{"label": "v1", "force": true}})
	if err == nil && !result.IsError {
		t.Error("MCP capture accepted an external store")
	}
	assertDevFileUnchanged(t, outside, "v1/index.html", "must survive")
	assertDevFileUnchanged(t, outside, "v1/mpress-version.json", "foreign manifest")
}
