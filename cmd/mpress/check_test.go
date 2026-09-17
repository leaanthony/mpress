package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/translate"
)

func captureCheckOutput(t *testing.T, args ...string) (string, error) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = original; file.Close() }()
	runErr := run(args)
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(data), runErr
}

func TestCheckCommandsOnBuiltProject(t *testing.T) {
	root := t.TempDir()
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Site.BaseURL = "https://docs.example.test"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	if err := build(nil); err != nil {
		t.Fatal(err)
	}
	output, err := captureCheckOutput(t, "check", "site", "--json")
	var report check.RenderedReport
	if decodeErr := json.Unmarshal([]byte(output), &report); decodeErr != nil {
		t.Fatalf("%v: %s", decodeErr, output)
	}
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	if report.Files == 0 || report.HTMLPages == 0 || len(report.Errors) > 0 {
		t.Fatalf("%+v", report)
	}
	if err := run([]string{"check"}); err != nil {
		t.Fatal(err)
	}
	// Standalone validation must inspect the existing build, not silently repair it.
	if err := os.WriteFile(filepath.Join(cfg.OutputPath(root), "index.html"), []byte(`<html lang="en"><a href="https://docs.example.test/missing/">Broken</a>`), 0644); err != nil {
		t.Fatal(err)
	}
	output, err = captureCheckOutput(t, "check", "site", "--json", "--cloudflare-pages")
	if err == nil || !strings.Contains(output, "missing link") {
		t.Fatalf("%v: %s", err, output)
	}
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatal(err)
	}
	cfg.Site.Languages = append(cfg.Site.Languages, "fr")
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	output, err = captureCheckOutput(t, "translate", "check", "--json")
	var translations translate.CheckReport
	if decodeErr := json.Unmarshal([]byte(output), &translations); decodeErr != nil {
		t.Fatalf("%v: %s", decodeErr, output)
	}
	if err == nil || !strings.Contains(output, "missing translation") {
		t.Fatalf("%v: %s", err, output)
	}
	for _, args := range [][]string{{"check", "site", "--bogus"}, {"check", "unexpected"}, {"check", "site", "unexpected"}, {"translate", "check", "--bogus"}, {"translate", "check", "unexpected"}, {"translate", "check", "--exceptions", "missing.json"}} {
		if _, err := captureCheckOutput(t, args...); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestCheckSiteSupportsLanguagePrefixAndOutputOverride(t *testing.T) {
	root := t.TempDir()
	if err := initProject([]string{root}); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	// Use a minimal site so starter examples with intentionally root-relative
	// links do not obscure validation of language-prefixed routes.
	if err := os.RemoveAll(cfg.ContentPath(root)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.ContentPath(root), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.ContentPath(root), "index.md"), []byte("# Home\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg.Site.BaseURL = "https://docs.example.test"
	cfg.Site.DefaultAtRoot = false
	cfg.Build.OutputDir = "release"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	if err := build(nil); err != nil {
		t.Fatal(err)
	}
	cfg.Build.OutputDir = "missing"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cfg.ContentPath(root))
	output, err := captureCheckOutput(t, "check", "site", "--output", "release", "--json")
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
}
