package translate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingTargetIsNotHiddenByCachedMachineText(t *testing.T) {
	root, cfg := translationProject(t)
	provider := &fakeProvider{}
	engine := NewEngine(root, cfg, provider)
	options := Options{Language: "fr", File: "index.md", Workers: 1}
	if _, err := engine.Run(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "content/fr/index.md")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	options.DryRun = true
	report, err := engine.Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Pending == 0 || report.Files[0].States["missing"] == 0 {
		t.Fatalf("deleted target reported current: %#v", report)
	}
	options.DryRun = false
	options.Scope = "missing"
	report, err = engine.Run(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if report.Written != 1 {
		t.Fatalf("deleted target not restored: %#v", report)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal(err)
	}
}

func TestExistingTranslationWithoutStateRequiresExplicitReplacement(t *testing.T) {
	root, cfg := translationProject(t)
	target := filepath.Join(root, "content/fr/index.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	manual := []byte("# Traduction humaine\n\nTexte relu et corrigé.\n")
	if err := os.WriteFile(target, manual, 0o644); err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{}
	engine := NewEngine(root, cfg, provider)
	status, err := engine.Run(context.Background(), Options{Language: "fr", File: "index.md", DryRun: true})
	if err != nil || len(status.Files) != 1 || status.Files[0].States["untracked"] == 0 || status.Pending != 0 {
		t.Fatalf("status must expose untracked translations without scheduling replacement: %v, %#v", err, status)
	}
	for _, options := range []Options{
		{Language: "fr", File: "index.md"},
		{Language: "fr", File: "index.md", Scope: "all"},
		{Language: "fr", File: "index.md", Force: true},
	} {
		_, err := engine.Run(context.Background(), options)
		if err == nil || !strings.Contains(err.Error(), "no translation state") {
			t.Fatalf("untracked translation was not protected: %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil || string(got) != string(manual) {
			t.Fatalf("manual translation changed: %v, %s", err, got)
		}
	}
	if provider.calls != 0 {
		t.Fatalf("called provider %d times", provider.calls)
	}
	report, err := engine.Run(context.Background(), Options{Language: "fr", File: "index.md", Scope: "all", Force: true, Workers: 1})
	if err != nil || report.Written != 1 {
		t.Fatalf("explicit replacement failed: %v, %#v", err, report)
	}
}
