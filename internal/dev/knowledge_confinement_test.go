package dev

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/knowledge"
)

func TestAuthoringKnowledgeRejectsOversizedManifest(t *testing.T) {
	root := authoringFixture(t)
	server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "knowledge-budget"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	server.cfg.Knowledge.Enabled = true
	output := server.cfg.OutputPath(root)
	fixtureKnowledge(t, output, "Current site")
	file, err := os.OpenFile(filepath.Join(output, knowledge.Directory, knowledge.ManifestFile), os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Truncate((16 << 20) + 1)
	closeErr := file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	host := httptest.NewServer(server.Handler())
	defer host.Close()
	response := requestJSON(t, http.MethodGet, host.URL+"/__mpress/api/knowledge", nil, "knowledge-budget")
	defer response.Body.Close()
	var status struct {
		Ready bool
		Error string
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.Ready || !strings.Contains(status.Error, "knowledge resource limit exceeded") {
		t.Fatalf("oversized manifest reached authoring: %+v", status)
	}
}

func TestAuthoringKnowledgeRejectsExternalBundles(t *testing.T) {
	for _, boundary := range []string{"output", "bundle", "manifest", "artifacts", "traversal", "versions", "version-bundle"} {
		t.Run(boundary, func(t *testing.T) {
			root, outside := authoringFixture(t), t.TempDir()
			server, err := NewServer(root, Options{Host: "0.0.0.0", Authoring: true, Token: "knowledge-boundary"})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = server.Close() })
			server.cfg.Knowledge.Enabled = true
			output := server.cfg.OutputPath(root)
			fixtureKnowledge(t, output, "Current site")
			fixtureKnowledge(t, outside, "external-secret-title")
			redirectKnowledge(t, boundary, output, outside)
			host := httptest.NewServer(server.Handler())
			defer host.Close()
			response := requestJSON(t, http.MethodGet, host.URL+"/__mpress/api/knowledge?q=sentinel", nil, "knowledge-boundary")
			defer response.Body.Close()
			var status struct {
				Ready bool
				Error string
			}
			if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
				t.Fatal(err)
			}
			if status.Ready || status.Error == "" {
				t.Fatalf("authoring read knowledge outside project: %+v", status)
			}
			assertDevFileUnchanged(t, outside, "sentinel", "must survive")
		})
	}
}
func fixtureKnowledge(t *testing.T, output, title string) {
	t.Helper()
	cfg := config.Default()
	cfg.Site.Title = title
	pages := map[string][]*content.Page{"en": {{Title: "Page", PlainText: "sentinel paragraph", HTML: "<p>sentinel paragraph</p>"}}}
	if err := knowledge.Generate(output, cfg, pages); err != nil {
		t.Fatal(err)
	}
	writeDevFixture(t, output, "sentinel", "must survive")
}
func redirectKnowledge(t *testing.T, boundary, output, outside string) {
	t.Helper()
	bundle := filepath.Join(output, knowledge.Directory)
	external := filepath.Join(outside, knowledge.Directory)
	link := func(target, name string) {
		t.Helper()
		if err := os.RemoveAll(name); err != nil {
			t.Fatal(err)
		}
		devSymlink(t, target, name)
	}
	switch boundary {
	case "output":
		link(outside, output)
	case "bundle":
		link(external, bundle)
	case "manifest":
		link(filepath.Join(external, knowledge.ManifestFile), filepath.Join(bundle, knowledge.ManifestFile))
	case "artifacts":
		for _, name := range []string{knowledge.PagesFile, knowledge.ChunksFile, knowledge.IndexFile} {
			link(filepath.Join(external, name), filepath.Join(bundle, name))
		}
	case "traversal":
		path := filepath.Join(bundle, knowledge.ManifestFile)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest knowledge.Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		rel, err := filepath.Rel(bundle, external)
		if err != nil {
			t.Fatal(err)
		}
		manifest.Artifacts = knowledge.Artifacts{Pages: filepath.ToSlash(filepath.Join(rel, knowledge.PagesFile)), Chunks: filepath.ToSlash(filepath.Join(rel, knowledge.ChunksFile)), Index: filepath.ToSlash(filepath.Join(rel, knowledge.IndexFile))}
		data, err = json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	case "versions":
		fixtureKnowledge(t, filepath.Join(outside, "v1"), "External version")
		link(outside, filepath.Join(output, "versions"))
	case "version-bundle":
		version := filepath.Join(output, "versions", "v1")
		if err := os.MkdirAll(version, 0755); err != nil {
			t.Fatal(err)
		}
		devSymlink(t, external, filepath.Join(version, knowledge.Directory))
	}
}
