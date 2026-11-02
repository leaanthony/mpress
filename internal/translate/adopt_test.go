package translate

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

const adoptNav = "- label: Home\n  path: index.md\n- label: Guide\n  path: guide.md\n"

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// untrackedProject builds a Git project whose French translation exists as
// plain content with no sidecar, the state the Wails docs are in. The returned
// revision is the commit that holds the source and its translation.
func untrackedProject(t *testing.T) (root string, cfg config.Config, rev string) {
	t.Helper()
	root, cfg = translationProject(t)
	write(t, root, "content/_nav.yaml", adoptNav)
	write(t, root, "content/index.md", "---\ntitle: Hello\n---\n\n# Welcome\n\nFirst paragraph.\n\nSecond paragraph.\n\n## Details\n\nMore information.\n")
	if _, err := NewEngine(root, cfg, &fakeProvider{}).Run(context.Background(), Options{Language: "fr"}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(root, cfg.Translation.StateDir)); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "init", "-q")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "translated")
	return root, cfg, gitIn(t, root, "rev-parse", "HEAD")
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func adoptFileReport(t *testing.T, report AdoptReport, source string) AdoptFileReport {
	t.Helper()
	for _, item := range report.Files {
		if item.SourceFile == source && item.Language == "fr" {
			return item
		}
	}
	t.Fatalf("no report for %s: %#v", source, report)
	return AdoptFileReport{}
}

func dryRun(t *testing.T, root string, cfg config.Config, scope string) FileReport {
	t.Helper()
	report, err := NewEngine(root, cfg, nil).Run(context.Background(), Options{Language: "fr", File: "index.md", Scope: scope, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	return report.Files[0]
}

func TestAdoptTracksAlignedTranslationWithoutChangingContent(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	if got := dryRun(t, root, cfg, "stale"); got.States["untracked"] == 0 {
		t.Fatalf("fixture should start untracked: %#v", got)
	}
	before := readFile(t, root, "content/fr/index.md")
	report, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Language: "fr", File: "index.md", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	item := adoptFileReport(t, report, "index.md")
	if item.Outcome != AdoptAdopted || !item.Written || item.Adopted != item.Segments || item.Drifted != 0 || item.New != 0 {
		t.Fatalf("unexpected report %#v", item)
	}
	if readFile(t, root, "content/fr/index.md") != before {
		t.Fatal("adopt changed translated content")
	}
	got := dryRun(t, root, cfg, "stale")
	if got.NeedsTranslation != 0 || got.States["adopted"] == 0 || got.States["untracked"] != 0 {
		t.Fatalf("adopted page should be current: %#v", got)
	}
	state, err := loadState(filepath.Join(root, cfg.Translation.StateDir, "fr", "index.json"), "index.md", "en", "fr", "")
	if err != nil {
		t.Fatal(err)
	}
	for id, entry := range state.Segments {
		if entry.Provider != "adopted" || entry.AdoptedFrom != rev || entry.Status != "adopted" || entry.MachineText != "" {
			t.Fatalf("segment %s has wrong provenance: %#v", id, entry)
		}
	}
}

func TestAdoptKeepsDriftedSegmentsStale(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	write(t, root, "content/index.md", "---\ntitle: Hello\n---\n\n# Welcome\n\nFirst paragraph, changed.\n\nSecond paragraph.\n\n## Details\n\nMore information.\n")
	// A hand-polished paragraph is the proof that untouched text survives.
	polished := strings.Replace(readFile(t, root, "content/fr/index.md"), "FR Second paragraph.", "Deuxième paragraphe soigné.", 1)
	write(t, root, "content/fr/index.md", polished)
	before := polished
	report, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Language: "fr", File: "index.md", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	item := adoptFileReport(t, report, "index.md")
	if item.Drifted != 1 || item.Adopted != item.Segments-1 {
		t.Fatalf("expected exactly one drifted segment: %#v", item)
	}
	got := dryRun(t, root, cfg, "stale")
	if got.States["stale"] != 1 || got.NeedsTranslation != 1 {
		t.Fatalf("drifted segment must be stale and the only work: %#v", got)
	}
	if readFile(t, root, "content/fr/index.md") != before {
		t.Fatal("adopt changed translated content")
	}
	provider := &fakeProvider{}
	if _, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr", File: "index.md"}); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("translate should contact the provider once, got %d", provider.calls)
	}
	after := readFile(t, root, "content/fr/index.md")
	if !strings.Contains(after, "FR First paragraph, changed.") || strings.Contains(after, "FR First paragraph.\n") || !strings.Contains(after, "Deuxième paragraphe soigné.") {
		t.Fatalf("only the stale paragraph should be retranslated:\n%s", after)
	}
}

func TestAdoptHandlesInsertedParagraphs(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	write(t, root, "content/index.md", "---\ntitle: Hello\n---\n\n# Welcome\n\nBrand new paragraph.\n\nFirst paragraph.\n\nSecond paragraph.\n\n## Details\n\nMore information.\n")
	report, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Language: "fr", File: "index.md", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	item := adoptFileReport(t, report, "index.md")
	if item.New != 1 || item.Drifted != 0 {
		t.Fatalf("expected one new segment and no drift: %#v", item)
	}
	if got := dryRun(t, root, cfg, "stale"); got.States["missing"] != 1 || got.NeedsTranslation != 1 {
		t.Fatalf("only the inserted paragraph needs work: %#v", got)
	}
}

func TestAdoptRefusesMisalignedStructure(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	target := readFile(t, root, "content/fr/index.md")
	lines := strings.Split(target, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if !strings.HasPrefix(line, "## ") {
			kept = append(kept, line)
		}
	}
	write(t, root, "content/fr/index.md", strings.Join(kept, "\n"))
	report, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Language: "fr", File: "index.md", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	item := adoptFileReport(t, report, "index.md")
	if item.Outcome != AdoptRefused || item.Reason == "" || item.Written || report.Refused != 1 || len(report.RefusedPages()) != 1 {
		t.Fatalf("misaligned page must be refused: %#v", item)
	}
	if _, err := os.Stat(filepath.Join(root, cfg.Translation.StateDir, "fr", "index.json")); !os.IsNotExist(err) {
		t.Fatalf("refused page must not get a sidecar: %v", err)
	}
}

func TestAdoptNavigationYAML(t *testing.T) {
	root, cfg := translationProject(t)
	write(t, root, "content/_nav.yaml", adoptNav)
	write(t, root, "content/fr/_nav.yaml", "- label: Accueil\n  path: index.md\n- label: Guide\n  path: guide.md\n")
	gitIn(t, root, "init", "-q")
	gitIn(t, root, "add", "-A")
	gitIn(t, root, "commit", "-q", "-m", "translated navigation")
	rev := gitIn(t, root, "rev-parse", "HEAD")
	write(t, root, "content/_nav.yaml", adoptNav+"- label: Extras\n  path: extras.md\n")
	before := readFile(t, root, "content/fr/_nav.yaml")
	// The extra entry means the translated navigation no longer matches the
	// current source, but it does match the source at the revision.
	report, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Language: "fr", File: "_nav.yaml", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	item := adoptFileReport(t, report, "_nav.yaml")
	if item.Outcome != AdoptAdopted || item.Segments != 3 || item.Adopted != 2 || item.New != 1 || !item.Written {
		t.Fatalf("navigation labels should be adopted, the new one missing: %#v", item)
	}
	if readFile(t, root, "content/fr/_nav.yaml") != before {
		t.Fatal("adopt changed the translated navigation")
	}
	state, err := loadState(filepath.Join(root, cfg.Translation.StateDir, "fr", "_nav.json"), "_nav.yaml", "en", "fr", "")
	if err != nil || state.Extractor != "navigation-v1" || len(state.Segments) != 2 {
		t.Fatalf("unexpected navigation state: %#v %v", state, err)
	}
	// A navigation file whose structure differs from the source is refused.
	write(t, root, "content/fr/_nav.yaml", "- label: Accueil\n  path: index.md\n")
	if err := os.RemoveAll(filepath.Join(root, cfg.Translation.StateDir)); err != nil {
		t.Fatal(err)
	}
	report, err = NewEngine(root, cfg, nil).Adopt(AdoptOptions{Language: "fr", File: "_nav.yaml", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	if item := adoptFileReport(t, report, "_nav.yaml"); item.Outcome != AdoptRefused {
		t.Fatalf("misaligned navigation must be refused: %#v", item)
	}
}

func TestAdoptIsIdempotentAndNeverOverwritesState(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	engine := NewEngine(root, cfg, nil)
	if _, err := engine.Adopt(AdoptOptions{Language: "fr", File: "index.md", Since: rev}); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(root, cfg.Translation.StateDir, "fr", "index.json")
	first, _ := os.ReadFile(statePath)
	report, err := engine.Adopt(AdoptOptions{Language: "fr", File: "index.md", Since: rev})
	if err != nil {
		t.Fatal(err)
	}
	if item := adoptFileReport(t, report, "index.md"); item.Outcome != AdoptAlreadyTracked || item.Written {
		t.Fatalf("second run must be a no-op: %#v", item)
	}
	if second, _ := os.ReadFile(statePath); !bytes.Equal(first, second) {
		t.Fatal("second adopt rewrote the sidecar")
	}
}

func TestAdoptDryRunWritesNothing(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	report, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Since: rev, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if item := adoptFileReport(t, report, "index.md"); item.Outcome != AdoptAdopted || item.Written || item.Adopted == 0 {
		t.Fatalf("dry run should report but not write: %#v", item)
	}
	if report.Written != 0 {
		t.Fatalf("dry run wrote %d files", report.Written)
	}
	if _, err := os.Stat(filepath.Join(root, cfg.Translation.StateDir)); !os.IsNotExist(err) {
		t.Fatalf("dry run created the state directory: %v", err)
	}
}

func TestAdoptRequiresSinceOrAcceptDrift(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	engine := NewEngine(root, cfg, nil)
	if _, err := engine.Adopt(AdoptOptions{}); err == nil || !strings.Contains(err.Error(), "--since") {
		t.Fatalf("adopt without --since must be refused, got %v", err)
	}
	if _, err := engine.Adopt(AdoptOptions{Since: rev, AcceptDrift: true}); err == nil {
		t.Fatal("--since with --accept-drift must be refused")
	}
	if _, err := engine.Adopt(AdoptOptions{Since: "no-such-rev"}); err == nil {
		t.Fatal("unknown revision must be refused")
	}
	if _, err := engine.Adopt(AdoptOptions{Since: "--output=x"}); err == nil {
		t.Fatal("option-like revision must be refused")
	}
	write(t, root, "content/index.md", "---\ntitle: Hello\n---\n\n# Welcome\n\nFirst paragraph, changed.\n\nSecond paragraph.\n\n## Details\n\nMore information.\n")
	report, err := engine.Adopt(AdoptOptions{File: "index.md", AcceptDrift: true})
	if err != nil {
		t.Fatal(err)
	}
	if item := adoptFileReport(t, report, "index.md"); item.Drifted != 0 || item.Adopted != item.Segments {
		t.Fatalf("--accept-drift adopts everything as current: %#v", item)
	}
	if report.Revision != "working-tree" {
		t.Fatalf("revision = %q", report.Revision)
	}
}

func TestAdoptedSegmentsAreNotClobberedByScopeAll(t *testing.T) {
	root, cfg, rev := untrackedProject(t)
	if _, err := NewEngine(root, cfg, nil).Adopt(AdoptOptions{Since: rev}); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, root, "content/fr/index.md")
	provider := &fakeProvider{}
	if _, err := NewEngine(root, cfg, provider).Run(context.Background(), Options{Language: "fr", File: "index.md", Scope: "all"}); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 0 || readFile(t, root, "content/fr/index.md") != before {
		t.Fatalf("--scope all without --force replaced adopted text (calls=%d)", provider.calls)
	}
	if got := dryRun(t, root, cfg, "all"); got.NeedsTranslation != 0 {
		t.Fatalf("adopted segments must not be pending under --scope all: %#v", got)
	}
	forced, err := NewEngine(root, cfg, nil).Run(context.Background(), Options{Language: "fr", File: "index.md", Scope: "all", Force: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if forced.Files[0].NeedsTranslation == 0 {
		t.Fatal("--force must still be able to regenerate adopted text")
	}
}

func TestStateDirIgnored(t *testing.T) {
	root := t.TempDir()
	gitIn(t, root, "init", "-q")
	if StateDirIgnored(root, ".mpress/translations") {
		t.Fatal("nothing is ignored yet")
	}
	write(t, root, ".gitignore", ".mpress/\n")
	if !StateDirIgnored(root, ".mpress/translations") {
		t.Fatal("ignored state directory was not detected")
	}
	write(t, root, ".gitignore", ".mpress/*\n!.mpress/translations/\n")
	if StateDirIgnored(root, ".mpress/translations") {
		t.Fatal("re-included state directory reported as ignored")
	}
	if StateDirIgnored(t.TempDir(), ".mpress/translations") {
		t.Fatal("a non-Git directory must not warn")
	}
}
