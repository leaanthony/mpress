package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func getConfigPayload(t *testing.T, baseURL string) configPayload {
	t.Helper()
	response, err := http.Get(baseURL + "/__mpress/api/config")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("config GET status %d: %s", response.StatusCode, readBody(response))
	}
	var payload configPayload
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

// TestPublishedConfigurationFollowsBuild is the user-visible contract: after a
// configuration change is built, GET /config, the served HTML and the HTTP
// document root all describe the same publication.
func TestPublishedConfigurationFollowsBuild(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.rebuild(false)
	host := httptest.NewServer(server.Handler())
	defer host.Close()

	configPath := filepath.Join(root, config.Filename)
	source, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := []byte(strings.Replace(string(source), "title: Test project", "title: External title", 1))
	if err := os.WriteFile(configPath, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	server.rebuild(true)

	payload := getConfigPayload(t, host.URL)
	if payload.Title != "External title" || payload.Config == nil || payload.Config.Site.Title != "External title" {
		t.Fatalf("GET config did not follow the built configuration: %#v", payload)
	}
	if payload.Revision != revision(edited) {
		t.Fatalf("GET config revision %q does not identify the edited source %q", payload.Revision, revision(edited))
	}
	response, err := http.Get(host.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	_ = response.Body.Close()
	if !strings.Contains(body, "External title") {
		t.Fatalf("served HTML did not use the built title: %s", body)
	}

	// Changing outputDir must move the served document root, not leave the
	// startup file server pointing at the old tree.
	moved := []byte(strings.Replace(string(edited), "outputDir: site", "outputDir: published", 1))
	moved = []byte(strings.Replace(string(moved), "title: External title", "title: Moved output", 1))
	if err := os.WriteFile(configPath, moved, 0o644); err != nil {
		t.Fatal(err)
	}
	server.rebuild(true)
	response, err = http.Get(host.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body = readBody(response)
	_ = response.Body.Close()
	if !strings.Contains(body, "Moved output") {
		t.Fatalf("HTTP root did not move with outputDir: %s", body)
	}
	if payload := getConfigPayload(t, host.URL); payload.Revision != revision(moved) {
		t.Fatalf("GET config revision %q does not identify the moved source", payload.Revision)
	}
}

func TestFailedBuildKeepsLastPublishedPreviewAndRoot(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.rebuild(false)
	host := httptest.NewServer(server.Handler())
	defer host.Close()

	configPath := filepath.Join(root, config.Filename)
	source, _ := os.ReadFile(configPath)
	goodSource := []byte(strings.Replace(string(source), "title: Test project", "title: Last usable", 1))
	if err := os.WriteFile(configPath, goodSource, 0o644); err != nil {
		t.Fatal(err)
	}
	server.rebuild(true)

	// A syntactically broken configuration cannot publish or repoint serving.
	if err := os.WriteFile(configPath, []byte("site: [unterminated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rebuildResult(server, true); err == nil {
		t.Fatal("invalid configuration unexpectedly built")
	}
	if err := os.WriteFile(configPath, []byte(strings.Replace(string(goodSource), "outputDir: site", "outputDir: ../escaped-output", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rebuildResult(server, true); err == nil {
		t.Fatal("unsafe output directory unexpectedly built")
	}
	response, err := http.Get(host.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body := readBody(response)
	_ = response.Body.Close()
	if !strings.Contains(body, "Last usable") || strings.Contains(body, "escaped-output") {
		t.Fatalf("failed builds changed the served preview: %s", body)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escaped-output")); !os.IsNotExist(err) {
		t.Fatalf("unsafe output directory was created: %v", err)
	}
	if payload := getConfigPayload(t, host.URL); payload.Title != "Last usable" {
		t.Fatalf("failed build replaced the published settings: %#v", payload)
	}
	// Route mapping still uses the same published manifest and configuration.
	if route, err := server.sourceForRoute("/"); err != nil || route != "content/index.md" {
		t.Fatalf("route mapping changed after a failed build: %q %v", route, err)
	}
}

// TestOlderBuildCannotRollBackNewerAcceptedSettings pins the sequencing guard:
// a build that started before a settings change must only publish its output,
// never overwrite the newer active configuration.
func TestOlderBuildCannotRollBackNewerAcceptedSettings(t *testing.T) {
	server, err := NewServer(authoringFixture(t), Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })

	generation := server.configGenerationAt()
	older := server.activeConfig()
	older.Site.Title = "Older build"
	newer := older.Clone()
	newer.Site.Title = "Newer settings"
	server.setActiveConfig(newer, "newer-revision")

	server.publishConfig(older, "older-revision", generation)
	if got := server.activeConfig().Site.Title; got != "Newer settings" {
		t.Fatalf("older build rolled back newer active settings to %q", got)
	}
	if got := server.publishedConfig().Site.Title; got != "Older build" {
		t.Fatalf("older build output was not published: %q", got)
	}

	// A build that starts after the accepted change is allowed to adopt the
	// configuration it used, repairing the active snapshot after a failed build.
	server.publishConfig(older, "older-revision", server.configGenerationAt())
	if got := server.activeConfig().Site.Title; got != "Older build" {
		t.Fatalf("current build did not adopt its used configuration: %q", got)
	}
}

func TestConcurrentConfigReadsAndWritesStayConsistent(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "127.0.0.1", Authoring: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.rebuild(false)
	host := httptest.NewServer(server.Handler())
	defer host.Close()

	base := getConfigPayload(t, host.URL)
	if base.Config == nil {
		t.Fatal("config payload did not include the structured configuration")
	}
	type result struct {
		status int
		title  string
	}
	results := make(chan result, 8)
	var writers sync.WaitGroup
	for index := 0; index < 8; index++ {
		writers.Add(1)
		go func(index int) {
			defer writers.Done()
			title := fmt.Sprintf("Concurrent %d", index)
			payload := base
			cfg := base.Config.Clone()
			cfg.Site.Title = title
			payload.Complete = true
			payload.Config = &cfg
			payload.Title = title
			data, _ := json.Marshal(payload)
			request, _ := http.NewRequest(http.MethodPut, host.URL+"/__mpress/api/config", bytes.NewReader(data))
			request.Header.Set("Content-Type", "application/json")
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				results <- result{status: -1}
				return
			}
			defer func() { _ = response.Body.Close() }()
			results <- result{status: response.StatusCode, title: title}
		}(index)
	}
	var readers sync.WaitGroup
	readTitles := make(chan string, 150)
	for index := 0; index < 150; index++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			response, err := http.Get(host.URL + "/__mpress/api/config")
			if err != nil {
				readTitles <- ""
				return
			}
			defer func() { _ = response.Body.Close() }()
			var payload configPayload
			if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&payload) != nil || payload.Config == nil {
				readTitles <- ""
				return
			}
			readTitles <- payload.Config.Site.Title
		}()
	}
	writers.Wait()
	readers.Wait()
	close(results)
	close(readTitles)
	for title := range readTitles {
		if title == "" {
			t.Fatal("concurrent config read failed")
		}
	}
	successes, conflicts := 0, 0
	winner := ""
	for result := range results {
		switch result.status {
		case http.StatusOK:
			successes++
			winner = result.title
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected concurrent config write status %d", result.status)
		}
	}
	if successes != 1 || conflicts != 7 {
		t.Fatalf("expected one winner and seven stale-revision conflicts, got %d/%d", successes, conflicts)
	}
	if payload := getConfigPayload(t, host.URL); payload.Title != winner {
		t.Fatalf("GET config %q disagrees with the winning write %q", payload.Title, winner)
	}
	onDisk, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if onDisk.Site.Title != winner || server.activeConfig().Site.Title != winner {
		t.Fatalf("active config %q, disk config %q and winner %q disagree", server.activeConfig().Site.Title, onDisk.Site.Title, winner)
	}
}

// setTestConfig replaces the active configuration for fixture setup without
// bypassing the snapshot guard.
func setTestConfig(server *Server, mutate func(*config.Config)) {
	cfg := server.activeConfig()
	mutate(&cfg)
	server.setActiveConfig(cfg, "test-fixture")
}

func rebuildResult(server *Server, reload bool) error {
	server.buildMu.Lock()
	defer server.buildMu.Unlock()
	return server.rebuildLocked(reload)
}
