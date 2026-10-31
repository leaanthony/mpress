package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWailsURLCorrectionPreservesTranslationsAndState(t *testing.T) {
	for _, locale := range []string{"fr", "ko"} {
		for _, page := range []string{"features/notifications/overview.md", "features/windows/options.md", "features/windows/multiple.md"} {
			t.Run(locale+"/"+page, func(t *testing.T) {
				previous, err := os.ReadFile(filepath.Join("../../testdata/wails-links/content", page))
				if err != nil {
					t.Fatal(err)
				}
				target, err := os.ReadFile(filepath.Join("../../testdata/wails-links/content", locale, page))
				if err != nil {
					t.Fatal(err)
				}
				current := bytes.ReplaceAll(previous, []byte("/releases/v3.0.0/"), []byte("/v3.0.0/"))
				current = bytes.ReplaceAll(current, []byte("/v3/examples/multi-window"), []byte("/v3/examples/window"))
				want := bytes.ReplaceAll(target, []byte("/releases/v3.0.0/"), []byte("/v3.0.0/"))
				want = bytes.ReplaceAll(want, []byte("/v3/examples/multi-window"), []byte("/v3/examples/window"))
				root, cfg := translationProject(t)
				cfg.Site.Languages = []string{"en", locale}
				path := filepath.Join(root, "content", page)
				dest := filepath.Join(root, "content", locale, page)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, current, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dest, target, 0600); err != nil {
					t.Fatal(err)
				}
				before, _ := ExtractMarkdown(previous)
				translated, _ := ExtractMarkdown(target)
				state := newFileState(page, "en", locale, "")
				for i, s := range before.Segments {
					state.Segments[s.ID] = SegmentState{SourceHash: s.SourceHash, TargetHash: Hash(translated.Segments[i].Original), Status: "manual", Provider: "human", UpdatedAt: "2026-10-01T00:00:00Z"}
				}
				sidecar := filepath.Join(root, ".mpress/translations", locale, strings.TrimSuffix(page, ".md")+".json")
				if err := os.MkdirAll(filepath.Dir(sidecar), 0700); err != nil {
					t.Fatal(err)
				}
				saved, _ := json.Marshal(state)
				if err := os.WriteFile(sidecar, saved, 0600); err != nil {
					t.Fatal(err)
				}
				exception := filepath.Join(root, "audit-exceptions.json")
				if err := os.WriteFile(exception, []byte(`[{"reason":"human review", "sourceSHA256":"old"}]`), 0600); err != nil {
					t.Fatal(err)
				}
				engine := NewEngine(root, cfg, nil)
				if _, err := engine.Run(context.Background(), Options{Language: locale, File: page, Scope: "missing"}); err == nil || !strings.Contains(err.Error(), "protected") {
					t.Fatalf("expected protected URL mismatch from existing path, got: %v", err)
				}
				for _, write := range []bool{false, true} {
					changed, err := engine.SyncLinks(locale, page, previous, write)
					if err != nil || !changed {
						t.Fatalf("changed=%t: %v", changed, err)
					}
					got, _ := os.ReadFile(dest)
					expected := target
					if write {
						expected = want
					}
					if !bytes.Equal(got, expected) {
						t.Fatalf("changed translated prose or preview wrote data:\n%s", got)
					}
					got, _ = os.ReadFile(sidecar)
					if !bytes.Equal(saved, got) {
						t.Fatal("translation state changed")
					}
					got, _ = os.ReadFile(exception)
					if !bytes.Contains(got, []byte(`"sourceSHA256":"old"`)) {
						t.Fatal("audit exception refreshed")
					}
				}
			})
		}
	}
}

func TestSyncLinksRejectsUnsafeChanges(t *testing.T) {
	old := []byte("# Guide\n\nRead **[the example](https://example.test/old)**.\n")
	target := []byte("# Guide traduit\n\nConsultez **[l’exemple](https://example.test/old)**.\n")
	current := bytes.ReplaceAll(old, []byte("/old"), []byte("/new"))
	tests := []struct {
		name           string
		source, target []byte
	}{
		{"prose", bytes.ReplaceAll(current, []byte("Read"), []byte("Run")), target},
		{"layout", append(current, []byte("\n```go\nprintln(1)\n```\n")...), target},
		{"human URL change", current, bytes.ReplaceAll(target, []byte("/old"), []byte("/custom"))},
		{"removed bold", current, bytes.ReplaceAll(target, []byte("**"), nil)},
		{"missing link", current, []byte("# Guide traduit\n\nConsultez l’exemple.\n")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := syncMarkdownLinks(old, test.source, test.target); err == nil {
				t.Fatal("accepted unsafe update")
			}
		})
	}
	// Code, reference-style, raw HTML and autolink changes are intentionally not inferred.
	for _, source := range []string{"`https://example.test/old`\n", "[example][x]\n\n[x]: https://example.test/old\n", "<https://example.test/old>\n", "<a href=\"https://example.test/old\">example</a>\n"} {
		previous := []byte(source)
		current := bytes.ReplaceAll(previous, []byte("/old"), []byte("/new"))
		if _, err := syncMarkdownLinks(previous, current, previous); err == nil {
			t.Fatalf("accepted unsupported syntax %s", source)
		}
	}
}
