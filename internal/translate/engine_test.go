package translate

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

type fakeProvider struct{ calls int }

type fakeRefinementProvider struct{ requests []TranslationRequest }

func (p *fakeRefinementProvider) Name() string  { return "refiner" }
func (p *fakeRefinementProvider) Model() string { return "quality-model" }
func (p *fakeRefinementProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.requests = append(p.requests, request)
	result := map[string]string{}
	for _, segment := range request.Segments {
		result[segment.ID] = "CORRECTED " + segment.CurrentText
	}
	return result, nil
}

func (p *fakeProvider) Name() string  { return "fake" }
func (p *fakeProvider) Model() string { return "test-model" }
func (p *fakeProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.calls++
	result := map[string]string{}
	for _, segment := range request.Segments {
		result[segment.ID] = "FR " + segment.Text
	}
	return result, nil
}

type structurallyFlakyProvider struct{ calls int }

func (p *structurallyFlakyProvider) Name() string  { return "flaky" }
func (p *structurallyFlakyProvider) Model() string { return "test-model" }
func (p *structurallyFlakyProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.calls++
	result := map[string]string{}
	for _, segment := range request.Segments {
		result[segment.ID] = "FR " + segment.Text
		if p.calls == 1 && strings.Contains(segment.Text, "Edit the file") {
			result[segment.ID] = "@note translated text."
		}
	}
	return result, nil
}

type singletonOnlyProvider struct{ calls int }

type fakeAuditReviewer struct {
	pairs int
	calls int
}

func (r *fakeAuditReviewer) Review(_ context.Context, request AuditRequest) ([]AuditFinding, error) {
	r.calls++
	r.pairs = len(request.Pairs)
	if len(request.Pairs) == 0 {
		return nil, nil
	}
	return []AuditFinding{{Severity: "warning", Code: "grammar", File: request.Pairs[0].File, Segment: request.Pairs[0].ID, Message: "check grammar"}}, nil
}

func TestAuditPairBatchesKeepEveryPairInOrder(t *testing.T) {
	pairs := []AuditPair{{ID: "one", Source: strings.Repeat("a", 10)}, {ID: "two", Source: strings.Repeat("b", 10)}, {ID: "three", Source: strings.Repeat("c", 10)}}
	batches := auditPairBatches(pairs, 115)
	if len(batches) != 3 {
		t.Fatalf("got %d batches, want 3: %#v", len(batches), batches)
	}
	for index, batch := range batches {
		if len(batch) != 1 || batch[0].ID != pairs[index].ID {
			t.Fatalf("batch %d = %#v", index, batch)
		}
	}
}

func (p *singletonOnlyProvider) Name() string  { return "singleton-test" }
func (p *singletonOnlyProvider) Model() string { return "test-model" }
func (p *singletonOnlyProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.calls++
	if len(request.Segments) > 1 {
		return nil, errors.New("batch contamination")
	}
	return map[string]string{request.Segments[0].ID: "ZH " + request.Segments[0].Text}, nil
}

func translationProject(t *testing.T) (string, config.Config) {
	t.Helper()
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Title = "Test"
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "index.md"), []byte("---\ntitle: Hello\n---\n\n# Welcome\n\nEdit the file.\n\n@button[Jump](#details){primary}\n\n## Details\n\nMore information.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, cfg
}

func TestEngineTranslatesAndTracksState(t *testing.T) {
	root, cfg := translationProject(t)
	provider := &fakeProvider{}
	engine := NewEngine(root, cfg, provider)
	report, err := engine.Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Written != 1 || report.Pending == 0 || provider.calls != 1 {
		t.Fatalf("unexpected report %#v, calls %d", report, provider.calls)
	}
	targetPath := filepath.Join(root, "content", "fr", "index.md")
	translated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(translated), "FR Welcome") || !strings.Contains(string(translated), `title: "FR Hello"`) {
		t.Fatalf("translation not written:\n%s", translated)
	}
	if !strings.Contains(string(translated), "(#fr-details)") {
		t.Fatalf("local heading fragment was not updated:\n%s", translated)
	}
	second, err := engine.Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if second.Pending != 0 || provider.calls != 1 {
		t.Fatalf("up-to-date run used provider: %#v calls=%d", second, provider.calls)
	}
	stateName := filepath.Join(root, ".mpress", "translations", "fr", "index.json")
	if _, err := os.Stat(stateName); err != nil {
		t.Fatalf("state not written: %v", err)
	}
}

func TestAuditFindsUntranslatedMPDProseAndWeakenedFrenchRequirement(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content", "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "# Install\n\nM-Press requires one executable.\n\nThis sentence was not translated.\n"
	target := "# Installer\n\nM-Press fournit un exécutable.\n\nThis sentence was not translated.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "install.mpd"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "fr", "install.mpd"), []byte(target), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := NewEngine(root, cfg, nil).Audit("fr", "install.mpd")
	if err != nil {
		t.Fatal(err)
	}
	if report.Errors != 1 || report.Warnings != 1 {
		t.Fatalf("unexpected audit report: %#v", report)
	}
	codes := map[string]bool{}
	for _, finding := range report.Findings {
		codes[finding.Code] = true
	}
	if !codes["untranslated"] || !codes["requirement-language"] {
		t.Fatalf("missing expected findings: %#v", report.Findings)
	}
}

func TestAuditWithReviewerMergesIndependentFindings(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content", "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "page.mpd"), []byte("# Hello\n\nBuild the site.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "content", "fr", "page.mpd"), []byte("# Bonjour\n\nGénérez le site.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reviewer := &fakeAuditReviewer{}
	report, err := NewEngine(root, cfg, nil).AuditWithReviewer(context.Background(), "fr", "page.mpd", reviewer)
	if err != nil {
		t.Fatal(err)
	}
	if reviewer.pairs != 2 || report.Warnings != 1 || len(report.Findings) != 1 || report.Findings[0].Code != "ai-grammar" {
		t.Fatalf("unexpected reviewed audit: pairs=%d report=%#v", reviewer.pairs, report)
	}
}

func TestRefinementRepairsOnlyFlaggedSegmentsAndKeepsStateCurrent(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "# Hello\n\nBuild the site.\n\nKeep this sentence.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "index.mpd"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	initial := &fakeProvider{}
	engine := NewEngine(root, cfg, initial)
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	document, err := extractFile("index.mpd", cfg.Build.NavFile, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	flagged := document.Segments[1]
	refiner := &fakeRefinementProvider{}
	report, err := NewEngine(root, cfg, nil).RefineWithProvider(context.Background(), "fr", "index.mpd", []AuditFinding{{Severity: "warning", Code: "grammar", File: "index.mpd", Segment: flagged.ID, Message: "improve grammar"}}, refiner)
	if err != nil {
		t.Fatal(err)
	}
	if report.Segments != 1 || report.Written != 1 || len(refiner.requests) != 1 || refiner.requests[0].Mode != "refine" || refiner.requests[0].Segments[0].ReviewNotes != "improve grammar" {
		t.Fatalf("unexpected refinement: report=%#v requests=%#v", report, refiner.requests)
	}
	target, err := os.ReadFile(filepath.Join(root, "content", "fr", "index.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(target), "CORRECTED FR Build the site.") || strings.Contains(string(target), "CORRECTED FR Keep this sentence.") {
		t.Fatalf("refinement changed the wrong prose:\n%s", target)
	}
	status, err := NewEngine(root, cfg, nil).Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.Pending != 0 || status.Files[0].States["manual"] != 0 {
		t.Fatalf("refined state is not current: %#v", status)
	}
}

func TestAuditAndRefinementRepairDamagedProtectedContent(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content", "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "# Intro\n\nWait 20 seconds.\n\nCall `Do()` now.\n\nKeep this sentence.\n"
	target := "# Introduction\n\nAttendre 200 secondes.\n\nAppeler `Other()` maintenant.\n\nGarder cette phrase.\n"
	targetPath := filepath.Join(root, "content", "fr", "page.mpd")
	for path, text := range map[string]string{filepath.Join(root, "content", "page.mpd"): source, targetPath: target} {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	engine := NewEngine(root, cfg, nil)
	audit, err := engine.Audit("fr", "page.mpd")
	if err != nil || audit.Errors != 2 || len(audit.Findings) != 2 {
		t.Fatalf("expected both damaged segments: %#v, %v", audit, err)
	}
	for _, finding := range audit.Findings {
		if finding.Code != "protected-content" || finding.Segment == "" {
			t.Fatalf("finding cannot target a repair: %#v", finding)
		}
	}
	bad := &fakeRefinementProvider{}
	if _, err := engine.RefineWithProvider(context.Background(), "fr", "page.mpd", audit.Findings, bad); err == nil {
		t.Fatal("accepted a refinement that retained damaged protected content")
	}
	if len(bad.requests) != 1 || !strings.Contains(bad.requests[0].Segments[0].ReviewNotes, "Rebuild this segment") {
		t.Fatalf("repair guidance was not sent: %#v", bad.requests)
	}
	unchanged, err := os.ReadFile(targetPath)
	if err != nil || string(unchanged) != target {
		t.Fatalf("failed refinement changed the file: %q, %v", unchanged, err)
	}
	report, err := engine.RefineWithProvider(context.Background(), "fr", "page.mpd", audit.Findings, &fakeProvider{})
	if err != nil || report.Segments != 2 || report.Written != 1 {
		t.Fatalf("targeted repair failed: %#v, %v", report, err)
	}
	output, err := os.ReadFile(targetPath)
	if err != nil || !strings.Contains(string(output), "Wait 20 seconds.") || !strings.Contains(string(output), "`Do()`") || !strings.Contains(string(output), "Garder cette phrase.") {
		t.Fatalf("incorrect repair: %s, %v", output, err)
	}
	audit, err = engine.Audit("fr", "page.mpd")
	if err != nil || audit.Errors != 0 {
		t.Fatalf("repaired content still fails: %#v, %v", audit, err)
	}
	reportState, err := engine.Run(context.Background(), Options{Language: "fr", File: "page.mpd", DryRun: true})
	if err != nil || reportState.Files[0].States["migration-review"] != 2 {
		t.Fatalf("untracked untouched prose was silently adopted: %#v, %v", reportState, err)
	}
}

func TestLocaleGuidanceCoversFrenchAndCJK(t *testing.T) {
	for _, language := range []string{"fr-FR", "zh-Hans", "ja", "ko"} {
		if guidance := localeGuidance(language); strings.TrimSpace(guidance) == "" {
			t.Fatalf("localeGuidance(%q) is empty", language)
		}
	}
	if !strings.Contains(localeGuidance("fr"), "Reorder complete sentences") {
		t.Fatalf("French guidance does not address placeholder reordering: %q", localeGuidance("fr"))
	}
}

func TestUnchangedProseDistinguishesSentencesLabelsAndCode(t *testing.T) {
	for _, test := range []struct {
		kind, text, severity string
		found                bool
	}{
		{kind: "text", text: "This sentence remains untranslated.", severity: "error", found: true},
		{kind: "heading", text: "Open source", severity: "warning", found: true},
		{kind: "cell", text: "`mpress build --strict`"},
		{kind: "heading", text: "Architecture"},
		{kind: "filetree", text: "Home page", severity: "error", found: true},
	} {
		severity, found := unchangedProse(test.kind, test.text, test.text)
		if found != test.found || severity != test.severity {
			t.Fatalf("unchangedProse(%q, %q) = %q, %v; want %q, %v", test.kind, test.text, severity, found, test.severity, test.found)
		}
	}
}

func TestPrepareExistingRestoresRepeatedFormattingDelimiters(t *testing.T) {
	document, err := ExtractMPD("page.mpd", []byte("Use *Translations* now.\n"))
	if err != nil {
		t.Fatal(err)
	}
	segment := document.Segments[0]
	prepared, err := prepareExisting(segment, "Utilisez *Traductions* maintenant.")
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restore(segment, prepared, false)
	if err != nil {
		t.Fatal(err)
	}
	if restored != "Utilisez *Traductions* maintenant." {
		t.Fatalf("restored existing translation = %q", restored)
	}
}

func TestTranslateBatchBisectsAContaminatedProviderBatch(t *testing.T) {
	provider := &singletonOnlyProvider{}
	engine := &Engine{Provider: provider}
	request := TranslationRequest{Segments: []RequestSegment{
		{ID: "one", Text: "First segment"},
		{ID: "two", Text: "Second segment"},
		{ID: "three", Text: "Third segment"},
	}}
	result, err := engine.translateBatch(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	for _, segment := range request.Segments {
		if result[segment.ID] != "ZH "+segment.Text {
			t.Fatalf("segment %q = %q", segment.ID, result[segment.ID])
		}
	}
	if provider.calls <= len(request.Segments) {
		t.Fatalf("provider calls = %d; expected failed batch calls plus isolated segments", provider.calls)
	}
}

func TestEngineRetriesStructurallyInvalidProviderOutput(t *testing.T) {
	root, cfg := translationProject(t)
	provider := &structurallyFlakyProvider{}
	report, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 || report.Written != 1 {
		t.Fatalf("structural response was not retried: calls=%d report=%#v", provider.calls, report)
	}
	target, err := os.ReadFile(filepath.Join(root, "content", "fr", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(target), "@note translated") || !strings.Contains(string(target), "FR Edit the file.") {
		t.Fatalf("invalid first response reached disk:\n%s", target)
	}
}

func TestEnginePreservesManualEditsAndRetranslatesStaleText(t *testing.T) {
	root, cfg := translationProject(t)
	provider := &fakeProvider{}
	engine := NewEngine(root, cfg, provider)
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(root, "content", "fr", "index.md")
	target, _ := os.ReadFile(targetPath)
	target = []byte(strings.Replace(string(target), "FR Welcome", "Bienvenue manuellement", 1))
	if err := os.WriteFile(targetPath, target, 0o644); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "content", "index.md")
	source, _ := os.ReadFile(sourcePath)
	source = []byte(strings.Replace(string(source), "Edit the file.", "Edit and save the file.", 1))
	if err := os.WriteFile(sourcePath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := engine.Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].States["manual"] != 1 || report.Files[0].States["stale"] != 1 {
		t.Fatalf("manual and stale states not detected: %#v", report.Files[0].States)
	}
	updated, _ := os.ReadFile(targetPath)
	if !strings.Contains(string(updated), "Bienvenue manuellement") || !strings.Contains(string(updated), "FR Edit and save the file.") {
		t.Fatalf("manual edit or stale translation was not handled:\n%s", updated)
	}
}

func TestMarkReviewedTracksLaterEdits(t *testing.T) {
	root, cfg := translationProject(t)
	engine := NewEngine(root, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	marked, err := engine.Mark("fr", "index.md", "reviewed")
	if err != nil {
		t.Fatal(err)
	}
	if marked.States["reviewed"] == 0 {
		t.Fatalf("review state not recorded: %#v", marked)
	}
	status, err := engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.Files[0].States["reviewed"] != status.Files[0].Segments {
		t.Fatalf("review status not reported: %#v", status.Files[0])
	}
	targetPath := filepath.Join(root, "content", "fr", "index.md")
	target, _ := os.ReadFile(targetPath)
	target = []byte(strings.Replace(string(target), "FR Welcome", "Bienvenue", 1))
	if err := os.WriteFile(targetPath, target, 0o644); err != nil {
		t.Fatal(err)
	}
	status, err = engine.Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.Files[0].States["manual"] != 1 {
		t.Fatalf("edit after review was not detected: %#v", status.Files[0].States)
	}
}

func TestEngineDryRunDoesNotNeedProvider(t *testing.T) {
	root, cfg := translationProject(t)
	cfg.Translation.InputPrice = 1.25
	cfg.Translation.OutputPrice = 5
	report, err := NewEngine(root, cfg, nil).Run(context.Background(), Options{Language: "fr", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Pending == 0 || report.Written != 0 {
		t.Fatalf("unexpected dry run report %#v", report)
	}
	if report.Estimate.Requests == 0 || report.Estimate.SourceCharacters == 0 || report.Estimate.InputTokens == 0 || report.Estimate.OutputTokens == 0 {
		t.Fatalf("dry run did not estimate provider usage: %#v", report.Estimate)
	}
	if !report.Estimate.PricingConfigured || report.Estimate.TotalCostUSD <= 0 {
		t.Fatalf("dry run did not apply configured pricing: %#v", report.Estimate)
	}
	wantCost := float64(report.Estimate.InputTokens)*cfg.Translation.InputPrice/1_000_000 + float64(report.Estimate.OutputTokens)*cfg.Translation.OutputPrice/1_000_000
	if report.Estimate.TotalCostUSD != wantCost {
		t.Fatalf("estimated cost = %f, want %f", report.Estimate.TotalCostUSD, wantCost)
	}
}

func TestTranslationKeyFollowsRenamedPage(t *testing.T) {
	root, cfg := translationProject(t)
	sourcePath := filepath.Join(root, "content", "index.md")
	source, _ := os.ReadFile(sourcePath)
	source = []byte(strings.Replace(string(source), "title: Hello", "title: Hello\ntranslationKey: getting-started", 1))
	if err := os.WriteFile(sourcePath, source, 0o644); err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{}
	engine := NewEngine(root, cfg, provider)
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	newSource := filepath.Join(root, "content", "guide.md")
	if err := os.Rename(sourcePath, newSource); err != nil {
		t.Fatal(err)
	}
	report, err := engine.Run(context.Background(), Options{Language: "fr", File: "guide.md"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 || report.Written != 1 || report.Pending != 0 {
		t.Fatalf("rename should reuse translation: report=%#v calls=%d", report, provider.calls)
	}
	translated, err := os.ReadFile(filepath.Join(root, "content", "fr", "guide.md"))
	if err != nil || !strings.Contains(string(translated), "FR Welcome") {
		t.Fatalf("renamed target missing: %v, %s", err, translated)
	}
	if _, err := os.Stat(filepath.Join(root, "content", "fr", "index.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old translated page was not removed: %v", err)
	}
}

func TestEngineTranslatesNativeMPDAndOutputRenders(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Title = "Native translation"
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := `---
schema = 1
title = "Install M-Press"
translationKey = "install"
---

# Installation

@note type="info" title="Before you start"
Install the binary.
@end

@steps
@step title="Download"
Get the release.
@end
@step title="Build"
Run ` + "`mpress build`" + `.
@end
@end

@tabs
@tab label="Linux"
Use the package.
@end
@tab label="Windows"
Use the archive.
@end
@end
`
	if err := os.WriteFile(filepath.Join(root, "content", "install.mpd"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	provider := &fakeProvider{}
	report, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr", File: "install.mpd"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Written != 1 || report.Pending == 0 || provider.calls != 1 {
		t.Fatalf("unexpected report %#v, calls=%d", report, provider.calls)
	}
	target, err := os.ReadFile(filepath.Join(root, "content", "fr", "install.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(target)
	for _, want := range []string{`title = "FR Install M-Press"`, `title="FR Before you start"`, `title="FR Download"`, `label="FR Linux"`, "`mpress build`"} {
		if !strings.Contains(text, want) {
			t.Errorf("translated MPD missing %q:\n%s", want, text)
		}
	}
	markdown, err := translationRenderable("install.mpd", target)
	if err != nil {
		t.Fatal(err)
	}
	renderer := content.NewRenderer()
	page, diagnostics, err := renderer.ParseBytes("install.mpd", "fr", markdown)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == "error" {
			t.Fatalf("translated MPD did not render: %#v", diagnostics)
		}
	}
	if page == nil || !strings.Contains(page.HTML, "FR Before you start") || !strings.Contains(page.HTML, "FR Linux") {
		t.Fatalf("translated rendered HTML is incomplete: %#v", page)
	}
}

func TestTranslationReusesLocalizedMPDLinksWithoutChangingCode(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "content", "index.mpd")
	source := "# Hello\n\nRead [the details](#details).\n\n## Details\n\nBuild the site.\n\n```go\nfmt.Println(\"(#details)\")\n```\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(root, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	source += "\nA newly added paragraph.\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	report, err := engine.Run(context.Background(), Options{Language: "fr"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Files[0].Translated != 1 || report.Files[0].States["manual"] != 0 {
		t.Fatalf("unexpected state: %#v", report)
	}
	doc, err := ExtractMPD("index.mpd", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	finding := AuditFinding{File: "index.mpd", Segment: doc.Segments[1].ID, Severity: "warning", Message: "Clarify the link instruction"}
	if _, err := engine.RefineWithProvider(context.Background(), "fr", "index.mpd", []AuditFinding{finding}, &fakeRefinementProvider{}); err != nil {
		t.Fatal(err)
	}
	target, err := os.ReadFile(filepath.Join(root, "content", "fr", "index.mpd"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(target), "](#fr-details)") || !strings.Contains(string(target), `fmt.Println("(#details)")`) {
		t.Fatalf("lost link/code contract: %s", target)
	}
	if err := Validate(doc, target); err != nil {
		t.Fatal(err)
	}
}

func TestChangedMPDHeadingStructureRequiresExplicitRetranslation(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Build.ContentDir = "content"
	cfg.Translation.SourceLanguage = "en"
	if err := os.MkdirAll(filepath.Join(root, "content"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "content", "index.mpd")
	source := "# Hello\n\nRead [details](#details).\n\n## Details\n\nBuild this.\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	engine := NewEngine(root, cfg, &fakeProvider{})
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(root, "content", "fr", "index.mpd")
	before, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	source += "\n## Added heading\n\nMore instructions.\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Run(context.Background(), Options{Language: "fr"}); err == nil || !strings.Contains(err.Error(), "--scope all --force") {
		t.Fatalf("expected explicit retry instruction, got %v", err)
	}
	after, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("overwrote existing work")
	}
	if _, err := engine.Run(context.Background(), Options{Language: "fr", Scope: "all", Force: true}); err != nil {
		t.Fatal(err)
	}
}
