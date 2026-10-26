package dev

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/projectfs"
)

const rootIdentityToken = "root-identity-test"

func newRootIdentityServer(t *testing.T, project string) (*Server, *httptest.Server) {
	t.Helper()
	server, err := NewServer(project, Options{Host: "0.0.0.0", Authoring: true, Token: rootIdentityToken})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	host := httptest.NewServer(server.Handler())
	t.Cleanup(host.Close)
	return server, host
}

func assertRootRejected(t *testing.T, response *http.Response) string {
	t.Helper()
	body := readBody(response)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("root change returned %d, want %d: %s", response.StatusCode, http.StatusConflict, body)
	}
	if !strings.Contains(body, "restart the server or select the project again") {
		t.Fatalf("root change response is not actionable: %s", body)
	}
	return body
}

func snapshotRootFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return snapshot
}

func assertSnapshotUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	for name, want := range before {
		got, ok := after[name]
		if !ok {
			t.Errorf("unrelated project lost %s", name)
			continue
		}
		if got != want {
			t.Errorf("unrelated project changed %s", name)
		}
	}
	for name := range after {
		if _, ok := before[name]; !ok {
			t.Errorf("unrelated project gained %s", name)
		}
	}
}

func assertNoProjectFile(t *testing.T, root, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("unexpected file %s in project (error %v)", name, err)
	}
}

// A stable root keeps authoring available through ordinary source changes,
// blog writes and rebuilds.
func TestAuthoringKeepsStableProjectRoot(t *testing.T) {
	root := authoringFixture(t)
	server, host := newRootIdentityServer(t, root)

	file := getFile(t, host.URL, "content/index.md")
	response := requestJSON(t, http.MethodPut, host.URL+"/__mpress/api/file", filePayload{Path: file.Path, Content: file.Content + "\nStable edit.\n", Revision: file.Revision}, rootIdentityToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("stable source update: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()
	if err := server.files.CheckRoot(); err != nil {
		t.Fatalf("stable project lost its identity after a source change: %v", err)
	}

	response = requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Stable post", Slug: "stable-post", Body: "# Stable\n"}, rootIdentityToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("stable blog create: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()
	post := getFile(t, host.URL, "content/blog/stable-post.md")
	response = requestJSON(t, http.MethodPut, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Path: post.Path, Title: "Stable post", Slug: "stable-post", Body: "# Updated\n", Revision: post.Revision}, rootIdentityToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("stable blog update: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()

	server.buildMu.Lock()
	err := server.rebuildLocked(true)
	server.buildMu.Unlock()
	if err != nil {
		t.Fatalf("stable rebuild: %v", err)
	}
	if state := server.currentState(); state.Status != "ready" {
		t.Fatalf("stable build status %q: %s", state.Status, state.Error)
	}
	saved, err := os.ReadFile(filepath.Join(root, "content/blog/stable-post.md"))
	if err != nil || !strings.Contains(string(saved), "# Updated\n") {
		t.Fatalf("stable blog post was not updated on disk (%v): %s", err, saved)
	}
}

// Starting through a stable symbolic link remains supported.
func TestAuthoringKeepsStableProjectAlias(t *testing.T) {
	root := authoringFixture(t)
	alias := filepath.Join(t.TempDir(), "project")
	devSymlink(t, root, alias)
	server, host := newRootIdentityServer(t, alias)

	response := requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Alias post", Slug: "alias-post", Body: "# Alias\n"}, rootIdentityToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("stable alias blog create: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()
	if err := server.files.CheckRoot(); err != nil {
		t.Fatalf("stable alias lost its identity: %v", err)
	}
	if state := server.currentState(); state.Status != "ready" {
		t.Fatalf("stable alias build status %q: %s", state.Status, state.Error)
	}
}

// Replacing the project directory must reject every delegated operation and
// leave both the pinned original and the unrelated replacement untouched.
func TestAuthoringRejectsReplacedProjectRoot(t *testing.T) {
	root := authoringFixture(t)
	server, host := newRootIdentityServer(t, root)

	preserved := root + "-preserved"
	if err := os.Rename(root, preserved); err != nil {
		t.Fatal(err)
	}
	writeDevFixture(t, root, "mpress.yaml", "site:\n  title: Other project\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n")
	writeDevFixture(t, root, "site/keep.txt", "unrelated project output\n")
	writeDevFixture(t, root, "content/other.md", "# Other project\n")
	replacementBefore := snapshotRootFiles(t, root)
	source, err := os.ReadFile(filepath.Join(preserved, "content/index.md"))
	if err != nil {
		t.Fatal(err)
	}
	configSource, err := os.ReadFile(filepath.Join(preserved, "mpress.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	response := requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Injected", Slug: "injected", Body: "# Injected\n"}, rootIdentityToken)
	assertRootRejected(t, response)
	assertSnapshotUnchanged(t, replacementBefore, snapshotRootFiles(t, root))
	assertDevFileUnchanged(t, preserved, "content/index.md", string(source))
	assertDevFileUnchanged(t, preserved, "mpress.yaml", string(configSource))
	assertNoProjectFile(t, preserved, "content/blog/injected.md")

	server.buildMu.Lock()
	err = server.rebuildLocked(true)
	server.buildMu.Unlock()
	if !errors.Is(err, projectfs.ErrRootChanged) {
		t.Fatalf("direct rebuild of replaced root: %v", err)
	}
	if state := server.currentState(); state.Status != "error" || !strings.Contains(state.Error, "restart the server or select the project again") {
		t.Fatalf("rejected direct rebuild state %+v", state)
	}
	server.rebuild(true)
	if state := server.currentState(); state.Status != "error" {
		t.Fatalf("rejected background rebuild state %+v", state)
	}

	response = requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/preview", previewPayload{Path: "content/index.md", Content: string(source) + "\nPreview\n", Revision: revision(source)}, rootIdentityToken)
	assertRootRejected(t, response)

	request, err := http.NewRequest(http.MethodPost, host.URL+"/__mpress/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"root-identity-test","version":"1"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+rootIdentityToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	_ = response.Body.Close()
	var rpcError struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &rpcError); err != nil || rpcError.Error == nil || !strings.Contains(rpcError.Error.Message, "restart the server") {
		t.Fatalf("missing actionable MCP protocol error: %s (%v)", body, err)
	}
	assertSnapshotUnchanged(t, replacementBefore, snapshotRootFiles(t, root))
	assertNoProjectFile(t, root, "site/index.html")

	// Restoring the original directory permits a retry without a restart.
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(preserved, root); err != nil {
		t.Fatal(err)
	}
	response = requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Recovered", Slug: "recovered", Body: "# Recovered\n"}, rootIdentityToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("restored project blog create: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()
	if state := server.currentState(); state.Status != "ready" {
		t.Fatalf("restored project build status %q: %s", state.Status, state.Error)
	}
}

// A moved project with no replacement at its old path is rejected as well.
func TestAuthoringRejectsMissingProjectRoot(t *testing.T) {
	root := authoringFixture(t)
	_, host := newRootIdentityServer(t, root)
	preserved := root + "-preserved"
	if err := os.Rename(root, preserved); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(preserved, "content/index.md"))
	if err != nil {
		t.Fatal(err)
	}
	response := requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Injected", Slug: "injected", Body: "# Injected\n"}, rootIdentityToken)
	assertRootRejected(t, response)
	assertDevFileUnchanged(t, preserved, "content/index.md", string(source))
	assertNoProjectFile(t, preserved, "content/blog/injected.md")
}

// Retargeting the symbolic link used to select the project is detected before
// dispatch, and restoring the link permits a normal write.
func TestAuthoringRejectsRetargetedProjectSymlink(t *testing.T) {
	first := authoringFixture(t)
	second := authoringFixture(t)
	writeDevFixture(t, second, "site/keep.txt", "second project output\n")
	alias := filepath.Join(t.TempDir(), "project")
	devSymlink(t, first, alias)
	server, host := newRootIdentityServer(t, alias)

	response := requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Before", Slug: "before", Body: "# Before\n"}, rootIdentityToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("initial alias blog create: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()

	secondBefore := snapshotRootFiles(t, second)
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, second, alias)

	response = requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Retargeted", Slug: "retargeted", Body: "# Retargeted\n"}, rootIdentityToken)
	assertRootRejected(t, response)
	if err := server.files.CheckRoot(); !errors.Is(err, projectfs.ErrRootChanged) {
		t.Fatalf("retargeted alias CheckRoot: %v", err)
	}
	assertSnapshotUnchanged(t, secondBefore, snapshotRootFiles(t, second))
	assertNoProjectFile(t, second, "content/blog/retargeted.md")

	// Restoring the link makes the original project usable again.
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	devSymlink(t, first, alias)
	response = requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/blog-posts", blogPostPayload{Title: "Restored", Slug: "restored", Body: "# Restored\n"}, rootIdentityToken)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("restored alias blog create: %d %s", response.StatusCode, readBody(response))
	}
	_ = response.Body.Close()
	if state := server.currentState(); state.Status != "ready" {
		t.Fatalf("restored alias build status %q: %s", state.Status, state.Error)
	}
	if _, err := os.Stat(filepath.Join(first, "content/blog/restored.md")); err != nil {
		t.Fatalf("restored project did not receive the post: %v", err)
	}
}
