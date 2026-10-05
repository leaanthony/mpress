package dev

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuthoringExportDoesNotCreateExternalWorkspace(t *testing.T) {
	root, outside := authoringFixture(t), t.TempDir()
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "export-boundary"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	devSymlink(t, outside, filepath.Join(root, ".mpress"))
	writeDevFixture(t, outside, "sentinel", "must survive")
	stamp := time.Unix(946684800, 0)
	if err := os.Chtimes(outside, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(outside)
	if err != nil {
		t.Fatal(err)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	response := requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/export", map[string]bool{"strict": true}, "export-boundary")
	assertConfinementRejected(t, response)
	assertDevFileUnchanged(t, outside, "sentinel", "must survive")
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 1 {
		t.Error("export left an external workspace")
	}
	after, err := os.Stat(outside)
	if err != nil || !after.ModTime().Equal(before.ModTime()) {
		t.Error("export created/removed a workspace outside the project before rejecting it")
	}
}
