package translate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/content"
)

// AuditException accepts a reviewed linguistic finding only while both files
// retain the exact content the reviewer saw. Structural findings are never waived.
type AuditException struct {
	Language     string `json:"language"`
	File         string `json:"file"`
	Segment      string `json:"segment"`
	Code         string `json:"code"`
	Reason       string `json:"reason"`
	SourceSHA256 string `json:"sourceSHA256"`
	TargetSHA256 string `json:"targetSHA256"`
}

type CheckOptions struct {
	Language   string
	Exceptions []AuditException
	// ExcludeAudit skips auditing explicitly named legacy files, but never skips coverage.
	ExcludeAudit []string
}

type CheckReport struct {
	Languages     []string `json:"languages"`
	Errors        []string `json:"errors"`
	Accepted      int      `json:"accepted"`
	ExcludedAudit []string `json:"excluded_audit"`
}

// Check enforces complete translations and a clean local audit, including warnings.
// It does not invoke providers, subprocesses, or modify translation state.
func (e *Engine) Check(options CheckOptions) (CheckReport, error) {
	report := CheckReport{Errors: []string{}, ExcludedAudit: []string{}}
	languages, err := e.targetLanguages(options.Language)
	if err != nil {
		return report, err
	}
	report.Languages = languages
	sources, err := e.sourceFiles("")
	if err != nil {
		return report, err
	}
	if len(sources) == 0 {
		return report, fmt.Errorf("no source files to check")
	}
	excluded, err := auditExclusions(sources, options.ExcludeAudit)
	if err != nil {
		return report, err
	}
	for file := range excluded {
		report.ExcludedAudit = append(report.ExcludedAudit, file)
	}
	sort.Strings(report.ExcludedAudit)
	discovered, err := content.Discover(e.Config.ContentPath(e.Project), e.Config.Site.Languages, e.Config.Site.DefaultLanguage)
	if err != nil {
		return report, err
	}
	for _, language := range languages {
		targets := discovered[language]
		nav := filepath.ToSlash(e.Config.Build.NavFile)
		if _, err := os.Stat(filepath.Join(e.Config.ContentPath(e.Project), language, filepath.FromSlash(nav))); err == nil {
			targets = append(targets, nav)
		}
		report.Errors = append(report.Errors, e.checkCoverage(language, sources, targets)...)
		e.checkLanguageAudit(language, options, excluded, &report)
	}
	sort.Strings(report.Errors)
	return report, nil
}

func (e *Engine) checkLanguageAudit(language string, options CheckOptions, excluded map[string]bool, report *CheckReport) {
	audit, auditErr := e.Audit(language, "")
	if auditErr != nil {
		report.Errors = append(report.Errors, language+": translation audit failed: "+auditErr.Error())
		return
	}
	for _, finding := range audit.Findings {
		if excluded[finding.File] {
			continue
		}
		if e.acceptedFinding(language, finding, options.Exceptions) {
			report.Accepted++
			continue
		}
		location := language + "/" + finding.File
		if finding.Segment != "" {
			location += " " + finding.Segment
		}
		report.Errors = append(report.Errors, location+": "+finding.Code+": "+finding.Message)
	}
}

func auditExclusions(sources, exclusions []string) (map[string]bool, error) {
	known := map[string]bool{}
	for _, file := range sources {
		known[file] = true
	}
	excluded := map[string]bool{}
	for _, file := range exclusions {
		if !known[file] {
			return nil, fmt.Errorf("audit exclusion %q does not name a source file", file)
		}
		excluded[file] = true
	}
	return excluded, nil
}

func (e *Engine) checkCoverage(language string, sources, targets []string) []string {
	var problems []string
	sourceSet := map[string]bool{}
	targetSet := map[string]bool{}
	for _, file := range targets {
		targetSet[file] = true
	}
	for _, file := range sources {
		sourceSet[file] = true
		location := language + "/" + file
		if !targetSet[file] {
			problems = append(problems, location+": missing translation")
			continue
		}
		data, err := os.ReadFile(filepath.Join(e.Config.ContentPath(e.Project), language, filepath.FromSlash(file)))
		if err != nil {
			problems = append(problems, location+": "+err.Error())
			continue
		}
		if strings.TrimSpace(string(data)) == "" {
			problems = append(problems, location+": empty translation")
		}
	}
	for _, file := range targets {
		if !sourceSet[file] {
			problems = append(problems, language+"/"+file+": no matching source")
		}
	}
	return problems
}

func (e *Engine) acceptedFinding(language string, finding AuditFinding, exceptions []AuditException) bool {
	if finding.Code != "untranslated" && finding.Code != "requirement-language" {
		return false
	}
	if !safeAuditPath(finding.File) || !safeAuditPath(language) {
		return false
	}
	for _, entry := range exceptions {
		if entry.Language != language || entry.File != finding.File || entry.Segment != finding.Segment || entry.Code != finding.Code || strings.TrimSpace(entry.Reason) == "" {
			continue
		}
		root := e.Config.ContentPath(e.Project)
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(finding.File)))
		if err != nil {
			return false
		}
		target, err := os.ReadFile(filepath.Join(root, language, filepath.FromSlash(finding.File)))
		if err != nil {
			return false
		}
		if fmt.Sprintf("%x", sha256.Sum256(source)) == entry.SourceSHA256 && fmt.Sprintf("%x", sha256.Sum256(target)) == entry.TargetSHA256 {
			return true
		}
	}
	return false
}

func safeAuditPath(path string) bool {
	if strings.Contains(path, "\\") || !filepath.IsLocal(path) {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}
