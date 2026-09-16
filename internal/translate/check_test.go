package translate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

func checkFixture(t *testing.T) *Engine {
	t.Helper()
	root := t.TempDir()
	cfg := config.Default()
	cfg.Build.ContentDir = "content"
	cfg.Site.Languages = []string{"en", "fr"}
	cfg.Site.DefaultLanguage = "en"
	cfg.Translation.SourceLanguage = "en"
	writeCheckFile(t, root, "content/guide.md", "# Installation\n\nInstall the application now.\n")
	writeCheckFile(t, root, "content/_nav.yaml", "- label: Installation\n  link: /guide/\n")
	return NewEngine(root, cfg, nil)
}

func writeCheckFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckTranslationCoverage(t *testing.T) {
	e := checkFixture(t)
	report, err := e.Check(CheckOptions{})
	if err != nil {
		t.Fatal(err)
	}
	problems := strings.Join(report.Errors, "\n")
	for _, expected := range []string{"fr/guide.md: missing translation", "fr/_nav.yaml: missing translation"} {
		if !strings.Contains(problems, expected) {
			t.Fatalf("missing %q in %s", expected, problems)
		}
	}
	writeCheckFile(t, e.Project, "content/fr/guide.md", " \n")
	writeCheckFile(t, e.Project, "content/fr/old.mpd", "# Ancien")
	writeCheckFile(t, e.Project, "content/fr/_nav.yaml", "- label: Installation\n  link: /guide/\n")
	report, err = e.Check(CheckOptions{})
	if err != nil {
		t.Fatal(err)
	}
	problems = strings.Join(report.Errors, "\n")
	for _, expected := range []string{"fr/guide.md: empty translation", "fr/old.mpd: no matching source"} {
		if !strings.Contains(problems, expected) {
			t.Fatalf("missing %q in %s", expected, problems)
		}
	}
}

func TestCheckAuditsLocallyAndKeepsExclusionCoverage(t *testing.T) {
	e := checkFixture(t)
	writeCheckFile(t, e.Project, "content/fr/guide.md", "# Installation\n\nInstall the application now.\n")
	writeCheckFile(t, e.Project, "content/fr/_nav.yaml", "- label: Installation\n  link: /guide/\n")
	report, err := e.Check(CheckOptions{})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "untranslated") {
		t.Fatalf("%+v %v", report, err)
	}
	report, err = e.Check(CheckOptions{ExcludeAudit: []string{"guide.md"}})
	if err != nil || len(report.Errors) != 0 || len(report.ExcludedAudit) != 1 {
		t.Fatalf("%+v %v", report, err)
	}
	os.Remove(filepath.Join(e.Project, "content/fr/guide.md"))
	report, err = e.Check(CheckOptions{ExcludeAudit: []string{"guide.md"}})
	if err != nil || len(report.Errors) != 1 || !strings.Contains(report.Errors[0], "missing translation") {
		t.Fatalf("%+v %v", report, err)
	}
	if _, err := e.Check(CheckOptions{ExcludeAudit: []string{"typo.md"}}); err == nil {
		t.Fatal("unknown exclusion accepted")
	}
	if _, err := e.Check(CheckOptions{Language: "de"}); err == nil {
		t.Fatal("unconfigured language accepted")
	}
}

func TestCheckHashPinnedExceptions(t *testing.T) {
	e := checkFixture(t)
	target := "# Installation\n\nInstall the application now.\n"
	writeCheckFile(t, e.Project, "content/fr/guide.md", target)
	finding := AuditFinding{File: "guide.md", Segment: "body", Code: "untranslated"}
	source, _ := os.ReadFile(filepath.Join(e.Project, "content/guide.md"))
	entry := AuditException{Language: "fr", File: finding.File, Segment: finding.Segment, Code: finding.Code, Reason: "Reviewed terminology", SourceSHA256: fmt.Sprintf("%x", sha256.Sum256(source)), TargetSHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(target)))}
	if !e.acceptedFinding("fr", finding, []AuditException{entry}) {
		t.Fatal("current exception rejected")
	}
	for _, code := range []string{"structure", "missing-file", "missing-segment", "protected-content", "glossary"} {
		changed := finding
		changed.Code = code
		waived := entry
		waived.Code = code
		if e.acceptedFinding("fr", changed, []AuditException{waived}) {
			t.Fatalf("waived %s", code)
		}
	}
	invalid := entry
	invalid.Reason = " "
	if e.acceptedFinding("fr", finding, []AuditException{invalid}) {
		t.Fatal("blank reason accepted")
	}
	invalid = entry
	invalid.SourceSHA256 = "stale"
	if e.acceptedFinding("fr", finding, []AuditException{invalid}) {
		t.Fatal("stale source accepted")
	}
	invalid = entry
	invalid.Language = "de"
	if e.acceptedFinding("fr", finding, []AuditException{invalid}) {
		t.Fatal("wrong language accepted")
	}
	writeCheckFile(t, e.Project, "content/fr/guide.md", target+"\n")
	if e.acceptedFinding("fr", finding, []AuditException{entry}) {
		t.Fatal("stale translation accepted")
	}
	for _, path := range []string{"../guide.md", "nested/../guide.md", "/tmp/guide.md", `..\guide.md`} {
		changed := finding
		changed.File = path
		if e.acceptedFinding("fr", changed, []AuditException{entry}) {
			t.Fatalf("unsafe path accepted: %s", path)
		}
	}
}

func TestCheckReportsAcceptedFindingsAndWarnings(t *testing.T) {
	e := checkFixture(t)
	writeCheckFile(t, e.Project, "content/fr/guide.md", "# Installation\n\nInstall the application now.\n")
	writeCheckFile(t, e.Project, "content/fr/_nav.yaml", "- label: Installation\n  link: /guide/\n")
	audit, err := e.Audit("fr", "")
	if err != nil {
		t.Fatal(err)
	}
	var exceptions []AuditException
	for _, finding := range audit.Findings {
		source, _ := os.ReadFile(filepath.Join(e.Project, "content", finding.File))
		target, _ := os.ReadFile(filepath.Join(e.Project, "content/fr", finding.File))
		exceptions = append(exceptions, AuditException{Language: "fr", File: finding.File, Segment: finding.Segment, Code: finding.Code, Reason: "Reviewed", SourceSHA256: fmt.Sprintf("%x", sha256.Sum256(source)), TargetSHA256: fmt.Sprintf("%x", sha256.Sum256(target))})
	}
	report, err := e.Check(CheckOptions{Exceptions: exceptions})
	if err != nil || len(report.Errors) > 0 || report.Accepted == 0 {
		t.Fatalf("%+v %v", report, err)
	}
	writeCheckFile(t, e.Project, "content/guide.md", "# Setup guide\n")
	writeCheckFile(t, e.Project, "content/fr/guide.md", "# Setup guide\n")
	report, err = e.Check(CheckOptions{})
	if err != nil || !strings.Contains(strings.Join(report.Errors, "\n"), "untranslated") {
		t.Fatalf("warning was not enforced: %+v %v", report, err)
	}
}
