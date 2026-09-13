package translate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranslationReusesTargetsAfterStructuralEdits(t *testing.T) {
	for _, moveTarget := range []bool{false, true} {
		t.Run(map[bool]string{false: "source only", true: "source and target"}[moveTarget], func(t *testing.T) {
			root, cfg := translationProject(t)
			source := filepath.Join(root, "content", "page.mpd")
			target := filepath.Join(root, "content", "fr", "page.mpd")
			write := func(path, text string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			original := "First `one`.\n\nSecond `two`.\n"
			write(source, original)
			engine := NewEngine(root, cfg, &fakeProvider{})
			options := Options{Language: "fr", File: "page.mpd", Workers: 1}
			if _, err := engine.Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			translated, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			write(source, "New paragraph.\n\n"+original)
			if moveTarget {
				write(target, "FR New paragraph.\n\n"+string(translated))
			}
			if _, err := engine.Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			want := "FR New paragraph.\n\n" + string(translated)
			if string(got) != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}

func TestMarkReorderedTranslationsKeepsEveryState(t *testing.T) {
	root, cfg := translationProject(t)
	source := filepath.Join(root, "content", "page.mpd")
	target := filepath.Join(root, "content", "fr", "page.mpd")
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "First `one`.\n\nSecond `two`.\n")
	engine := NewEngine(root, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr", File: "page.mpd", Workers: 1}); err != nil {
		t.Fatal(err)
	}
	write(source, "Second `two`.\n\nFirst `one`.\n")
	write(target, "FR Second `two`.\n\nFR First `one`.\n")
	for _, status := range []string{"reviewed", "final"} {
		report, err := engine.Mark("fr", "page.mpd", status)
		if err != nil {
			t.Fatal(err)
		}
		if report.States[status] != 2 {
			t.Fatalf("lost state: %#v", report)
		}
	}
	stateFile, err := statePath(root, cfg.Translation.StateDir, "fr", "page.mpd")
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadState(stateFile, "page.mpd", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range state.Segments {
		if s.Status != "final" || !strings.Contains(string(data), s.MachineText) {
			t.Fatalf("wrong final state: %#v", s)
		}
	}
}
