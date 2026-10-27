package dev

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const queuedExportPrivateMarker = "private replacement marker"

// Include directory entries so an unexpected workspace is visible to the check.
func queuedExportTreeDigest(t *testing.T, root string) string {
	t.Helper()
	digest := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(digest, filepath.ToSlash(rel))
		_, _ = digest.Write([]byte{0})
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, _ = digest.Write(data)
		_, _ = digest.Write([]byte{0})
		return nil
	})
	if err != nil {
		t.Fatalf("hash %s: %v", root, err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

// waitForQueuedExport observes the owned HTTP goroutine blocked on buildMu
// after it passed the dispatch guard. There is no test-only hook in the server.
func waitForQueuedExport(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	buffer := make([]byte, 1<<20)
	for time.Now().Before(deadline) {
		n := runtime.Stack(buffer, true)
		for _, goroutine := range strings.Split(string(buffer[:n]), "\n\n") {
			if strings.Contains(goroutine, "(*Server).handleExport") && strings.Contains(goroutine, "sync.(*Mutex).Lock") {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("export handler did not queue on buildMu")
}

// An export that waits for another build must not export a project installed at
// the selected path while it waited. The original and the replacement trees are
// preserved, and restoring the original directory keeps the server usable.
func TestQueuedExportRejectsReplacedProjectRoot(t *testing.T) {
	root := authoringFixture(t)
	server, host := newRootIdentityServer(t, root)
	server.rebuild(false)

	preserved := root + "-preserved"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	type exportResult struct {
		response *http.Response
		err      error
	}
	results := make(chan exportResult, 1)
	locked, reaped, restored := true, false, false
	t.Cleanup(func() {
		if locked {
			server.buildMu.Unlock()
		}
		if !reaped {
			select {
			case result := <-results:
				if result.response != nil {
					_ = result.response.Body.Close()
				}
			case <-time.After(5 * time.Second):
			}
		}
		if !restored {
			if _, err := os.Stat(preserved); err == nil {
				_ = os.RemoveAll(root)
				_ = os.Rename(preserved, root)
			}
		}
	})

	// Stand in for a running build: the handler's dispatch guard has already
	// passed when it queues here.
	server.buildMu.Lock()
	go func() {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, host.URL+"/__mpress/api/export", strings.NewReader(`{"strict":true}`))
		if err != nil {
			results <- exportResult{err: err}
			return
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-MPress-Token", rootIdentityToken)
		response, err := http.DefaultClient.Do(request)
		results <- exportResult{response: response, err: err}
	}()
	waitForQueuedExport(t)

	if err := os.Rename(root, preserved); err != nil {
		t.Fatal(err)
	}
	writeDevFixture(t, root, "mpress.yaml", "site:\n  title: Replacement project\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n")
	writeDevFixture(t, root, "content/index.md", "---\ntitle: Replacement\n---\n\n"+queuedExportPrivateMarker+"\n")
	writeDevFixture(t, root, "content/_nav.yaml", "- label: Home\n  link: /\n")
	if err := os.MkdirAll(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	replacementBefore := queuedExportTreeDigest(t, root)
	originalBefore := queuedExportTreeDigest(t, preserved)

	server.buildMu.Unlock()
	locked = false

	result := <-results
	reaped = true
	if result.err != nil {
		t.Fatal(result.err)
	}
	response := result.response
	if strings.Contains(response.Header.Get("Content-Type"), "zip") || response.Header.Get("Content-Disposition") != "" {
		t.Fatalf("queued export returned an archive: %#v", response.Header)
	}
	body := assertRootRejected(t, response)
	if strings.Contains(body, queuedExportPrivateMarker) || bytes.Contains([]byte(body), []byte("PK\x03\x04")) {
		t.Fatalf("queued export disclosed replacement content: %q", body)
	}
	if got := queuedExportTreeDigest(t, root); got != replacementBefore {
		t.Error("replacement project changed during the rejected export")
	}
	if got := queuedExportTreeDigest(t, preserved); got != originalBefore {
		t.Error("original project changed during the rejected export")
	}
	assertNoProjectFile(t, root, "site/index.html")

	// Restoring the selected directory permits an ordinary export from the same
	// server, and that export contains the original project's pages.
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(preserved, root); err != nil {
		t.Fatal(err)
	}
	restored = true

	if err := server.files.CheckRoot(); err != nil {
		t.Fatalf("restored project lost its identity: %v", err)
	}
	response = requestJSON(t, http.MethodPost, host.URL+"/__mpress/api/export", map[string]bool{"strict": true, "drafts": false}, rootIdentityToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("restored export status %d: %s", response.StatusCode, readBody(response))
	}
	data, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	foundIndex := false
	for _, item := range archive.File {
		if item.Name != "index.html" {
			continue
		}
		foundIndex = true
		reader, err := item.Open()
		if err != nil {
			t.Fatal(err)
		}
		markup, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(markup), "Editable heading") {
			t.Fatalf("restored export is missing the original page: %s", markup)
		}
		if strings.Contains(string(markup), queuedExportPrivateMarker) {
			t.Fatal("restored export contains replacement content")
		}
	}
	if !foundIndex {
		t.Fatal("restored export did not contain index.html")
	}
}
