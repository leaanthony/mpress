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

func TestRelocationPreservesManualTargetEdits(t *testing.T) {
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
			write(source, "First sentence.\n\nSecond sentence.\n")
			engine := NewEngine(root, cfg, &fakeProvider{})
			options := Options{Language: "fr", File: "page.mpd", Workers: 1}
			if _, err := engine.Run(context.Background(), options); err != nil {
				t.Fatal(err)
			}
			write(source, "Second sentence.\n\nFirst sentence.\n\nNew paragraph.\n")
			edited := "FR First sentence, manually revised."
			if moveTarget {
				write(target, "FR Second sentence.\n\n"+edited+"\n")
			} else {
				write(target, edited+"\n\nFR Second sentence.\n")
			}
			report, err := engine.Run(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			want := "FR Second sentence.\n\n" + edited + "\n\nFR New paragraph.\n"
			if string(got) != want || report.Files[0].States["manual"] != 1 {
				t.Fatalf("manual translation moved incorrectly: %q, %#v", got, report)
			}
		})
	}
}

func TestMarkRelocatedManualTranslation(t *testing.T) {
	root, cfg := translationProject(t)
	source := filepath.Join(root, "content", "page.mpd")
	target := filepath.Join(root, "content", "fr", "page.mpd")
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "First sentence.\n\nSecond sentence.\n")
	engine := NewEngine(root, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr", File: "page.mpd", Workers: 1}); err != nil {
		t.Fatal(err)
	}
	write(source, "Second sentence.\n\nFirst sentence.\n")
	edited := "FR First sentence, manually revised."
	write(target, "FR Second sentence.\n\n"+edited+"\n")
	if _, err := engine.Mark("fr", "page.mpd", "final"); err != nil {
		t.Fatal(err)
	}
	p, err := statePath(root, cfg.Translation.StateDir, "fr", "page.mpd")
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadState(p, "page.mpd", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	if state.Segments["b0002-text-01"].TargetHash != Hash(edited) {
		t.Fatal("review state describes the wrong paragraph")
	}
}

func TestAmbiguousRelocationDoesNotOverwriteManualWork(t *testing.T) {
	root, cfg := translationProject(t)
	source := filepath.Join(root, "content", "page.mpd")
	target := filepath.Join(root, "content", "fr", "page.mpd")
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "First sentence.\n\nSecond sentence.\n")
	engine := NewEngine(root, cfg, &fakeProvider{})
	options := Options{Language: "fr", File: "page.mpd", Workers: 1}
	if _, err := engine.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	write(source, "Second sentence.\n\nFirst sentence.\n")
	edited := "Second manually rewritten.\n\nFirst manually rewritten.\n"
	write(target, edited)
	stateFile, err := statePath(root, cfg.Translation.StateDir, "fr", "page.mpd")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Run(context.Background(), options); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous relocation, got %v", err)
	}
	if _, err := engine.Mark("fr", "page.mpd", "final"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous review, got %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != edited || string(after) != string(before) {
		t.Fatal("ambiguous relocation changed target or state")
	}
	options.Force = true
	options.Scope = "all"
	if _, err := engine.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "FR Second sentence.\n\nFR First sentence.\n" {
		t.Fatalf("explicit regeneration failed: %q", got)
	}
}

func TestReviewedMPDLinksRemainFinalUntilEdited(t *testing.T) {
	root, cfg := translationProject(t)
	source := filepath.Join(root, "content", "links.mpd")
	target := filepath.Join(root, "content", "fr", "links.mpd")
	if err := os.WriteFile(source, []byte("# Welcome\n\n[See details](#details).\n\n## Details\n\nMore information.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(root, cfg, &fakeProvider{})
	options := Options{Language: "fr", File: "links.mpd", Workers: 1}
	if _, err := engine.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	translated, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(translated), "(#fr-details)") {
		t.Fatalf("missing translated anchor: %s", translated)
	}
	if _, err := engine.Mark("fr", "links.mpd", "final"); err != nil {
		t.Fatal(err)
	}
	options.DryRun = true
	report, err := engine.Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["final"] != report.Segments || report.Files[0].States["manual"] != 0 {
		t.Fatalf("reviewed links appear modified: %#v", report)
	}
	edited := strings.Replace(string(translated), "See details", "Voir les détails", 1)
	if err := os.WriteFile(target, []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	report, err = engine.Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["manual"] != 1 {
		t.Fatalf("real manual edit was missed: %#v", report)
	}
}
