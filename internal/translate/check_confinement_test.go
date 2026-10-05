package translate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func linkedCheckFixture(t *testing.T, boundary string, external bool) (*Engine, AuditFinding, AuditException) {
	t.Helper()
	e := checkFixture(t)
	probe := filepath.Join(e.Project, "symlink-probe")
	if err := os.Symlink("missing", probe); err != nil {
		t.Skipf("native symlinks unavailable: %v", err)
	}
	if err := os.Remove(probe); err != nil {
		t.Fatal(err)
	}
	data := "# Installation\n\nInstall the application now.\n"
	writeCheckFile(t, e.Project, "content/fr/guide.md", data)
	path := filepath.Join(e.Project, "content", "guide.md")
	if boundary == "target" {
		path = filepath.Join(e.Project, "content", "fr", "guide.md")
	}
	saved := path + ".saved"
	if external {
		saved = filepath.Join(t.TempDir(), "guide.md")
	}
	if err := os.Rename(path, saved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(saved, path); err != nil {
		t.Fatal(err)
	}
	finding := AuditFinding{File: "guide.md", Segment: "body", Code: "untranslated"}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
	entry := AuditException{Language: "fr", File: finding.File, Segment: finding.Segment, Code: finding.Code, Reason: "Reviewed", SourceSHA256: digest, TargetSHA256: digest}
	return e, finding, entry
}

func TestCheckExceptionsRejectExternalFiles(t *testing.T) {
	for _, boundary := range []string{"source", "target"} {
		t.Run(boundary, func(t *testing.T) {
			e, finding, entry := linkedCheckFixture(t, boundary, true)
			if e.acceptedFinding("fr", finding, []AuditException{entry}) {
				t.Fatal("check accepted hashes read from outside the project")
			}
		})
	}
}

func TestCheckCoverageRejectsExternalTarget(t *testing.T) {
	e, _, _ := linkedCheckFixture(t, "target", true)
	if problems := e.checkCoverage("fr", []string{"guide.md"}, []string{"guide.md"}); len(problems) == 0 {
		t.Fatal("coverage read an external target without reporting its boundary")
	}
}

func TestCheckRetainsInternalAliases(t *testing.T) {
	for _, boundary := range []string{"source", "target"} {
		t.Run(boundary, func(t *testing.T) {
			e, finding, entry := linkedCheckFixture(t, boundary, false)
			if !e.acceptedFinding("fr", finding, []AuditException{entry}) {
				t.Fatal("safe internal alias lost a valid hash-pinned exception")
			}
			if problems := e.checkCoverage("fr", []string{"guide.md"}, []string{"guide.md"}); len(problems) != 0 {
				t.Fatalf("safe coverage alias failed: %v", problems)
			}
		})
	}
}
