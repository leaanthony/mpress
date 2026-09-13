package translate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type AuditFinding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	File     string `json:"file"`
	Segment  string `json:"segment,omitempty"`
	Message  string `json:"message"`
}

type AuditReport struct {
	Language string         `json:"language"`
	Files    int            `json:"files"`
	Segments int            `json:"segments"`
	Errors   int            `json:"errors"`
	Warnings int            `json:"warnings"`
	Findings []AuditFinding `json:"findings,omitempty"`
}

type AuditPair struct {
	File    string `json:"file"`
	ID      string `json:"id"`
	Section string `json:"section,omitempty"`
	Source  string `json:"source"`
	Target  string `json:"target"`
}

func (report *AuditReport) add(finding AuditFinding) {
	report.Findings = append(report.Findings, finding)
	if finding.Severity == "error" {
		report.Errors++
	} else {
		report.Warnings++
	}
}

func (report *AuditReport) addStructureError(file string, err error) {
	var inlineFailure *inlineProtectionFailure
	if err != nil && !errors.As(err, &inlineFailure) {
		report.add(AuditFinding{Severity: "error", Code: "structure", File: file, Message: err.Error()})
	}
}

func (report *AuditReport) addProtectedContentError(format, file string, source, target Segment) {
	if format != "mpd" || !strings.Contains(source.Text, "⟪MPRESS_INLINE_") {
		return
	}
	if _, err := prepareExistingSegment(source, target); err != nil {
		report.add(AuditFinding{Severity: "error", Code: "protected-content", File: file, Segment: source.ID, Message: err.Error()})
	}
}

func (report AuditReport) JSON() string {
	data, _ := json.MarshalIndent(report, "", "  ")
	return string(data)
}

func (e *Engine) Audit(language, sourceFile string) (AuditReport, error) {
	report, _, err := e.auditPairs(language, sourceFile)
	return report, err
}

func (e *Engine) AuditWithReviewer(ctx context.Context, language, sourceFile string, reviewer AuditReviewer) (AuditReport, error) {
	report, pairs, err := e.auditPairs(language, sourceFile)
	if err != nil || reviewer == nil || len(pairs) == 0 {
		return report, err
	}
	styleGuide, err := readOptionalProjectFile(e.Project, e.Config.Translation.StyleGuide, 128<<10)
	if err != nil {
		return report, fmt.Errorf("read translation style guide: %w", err)
	}
	glossaries, err := loadGlossary(e.Project, e.Config.Translation.Glossary)
	if err != nil {
		return report, err
	}
	for _, batch := range auditPairBatches(pairs, 12_000) {
		findings, reviewErr := reviewer.Review(ctx, AuditRequest{
			SourceLanguage: e.Config.Translation.SourceLanguage,
			TargetLanguage: report.Language,
			LocaleGuidance: localeGuidance(report.Language),
			StyleGuide:     styleGuide,
			Glossary:       glossaries[report.Language],
			Pairs:          batch,
		})
		if reviewErr != nil {
			return report, reviewErr
		}
		for _, finding := range findings {
			finding.Code = "ai-" + strings.TrimPrefix(finding.Code, "ai-")
			report.add(finding)
		}
	}
	return report, nil
}

func auditPairBatches(pairs []AuditPair, maxCharacters int) [][]AuditPair {
	if maxCharacters <= 0 {
		maxCharacters = 12_000
	}
	var batches [][]AuditPair
	var batch []AuditPair
	size := 0
	for _, pair := range pairs {
		pairSize := len(pair.Source) + len(pair.Target) + len(pair.Section) + 100
		if len(batch) > 0 && size+pairSize > maxCharacters {
			batches = append(batches, batch)
			batch = nil
			size = 0
		}
		batch = append(batch, pair)
		size += pairSize
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}
	return batches
}

func (e *Engine) auditPairs(language, sourceFile string) (AuditReport, []AuditPair, error) {
	languages, err := e.targetLanguages(language)
	if err != nil {
		return AuditReport{}, nil, err
	}
	if len(languages) != 1 {
		return AuditReport{}, nil, errors.New("translation audit requires one target language")
	}
	language = languages[0]
	files, err := e.sourceFiles(sourceFile)
	if err != nil {
		return AuditReport{}, nil, err
	}
	glossaries, err := loadGlossary(e.Project, e.Config.Translation.Glossary)
	if err != nil {
		return AuditReport{}, nil, err
	}
	report := AuditReport{Language: language}
	var pairs []AuditPair
	contentDir := e.Config.ContentPath(e.Project)
	for _, file := range files {
		report.Files++
		source, readErr := os.ReadFile(filepath.Join(contentDir, filepath.FromSlash(file)))
		if readErr != nil {
			return report, pairs, readErr
		}
		sourceDoc, extractErr := extractFile(file, e.Config.Build.NavFile, source)
		if extractErr != nil {
			return report, pairs, extractErr
		}
		targetPath := filepath.Join(contentDir, filepath.FromSlash(language), filepath.FromSlash(file))
		target, readErr := os.ReadFile(targetPath)
		if errors.Is(readErr, os.ErrNotExist) {
			report.add(AuditFinding{Severity: "error", Code: "missing-file", File: file, Message: "translated file does not exist"})
			continue
		}
		if readErr != nil {
			return report, pairs, readErr
		}
		targetDoc, extractErr := extractTargetFile(file, e.Config.Build.NavFile, source, target)
		if extractErr != nil {
			report.add(AuditFinding{Severity: "error", Code: "invalid-target", File: file, Message: extractErr.Error()})
			continue
		}
		report.addStructureError(file, validateFile(file, e.Config.Build.NavFile, sourceDoc, target))
		targets := make(map[string]Segment, len(targetDoc.Segments))
		for _, segment := range targetDoc.Segments {
			targets[segment.ID] = segment
		}
		for _, segment := range sourceDoc.Segments {
			if segment.Protected {
				continue
			}
			report.Segments++
			targetSegment, ok := targets[segment.ID]
			if !ok {
				report.add(AuditFinding{Severity: "error", Code: "missing-segment", File: file, Segment: segment.ID, Message: "translated segment is missing"})
				continue
			}
			pairs = append(pairs, AuditPair{File: file, ID: segment.ID, Section: segment.Section, Source: segment.Original, Target: targetSegment.Original})
			report.addProtectedContentError(sourceDoc.Format, file, segment, targetSegment)
			if severity, unchanged := unchangedSegmentProse(segment, targetSegment.Original, e.Config.Site.Title); unchanged {
				report.add(AuditFinding{Severity: severity, Code: "untranslated", File: file, Segment: segment.ID, Message: "prose is unchanged from the source"})
			}
			if glossaryErr := validateGlossary(segment.Original, targetSegment.Original, glossaries[language]); glossaryErr != nil {
				report.add(AuditFinding{Severity: "error", Code: "glossary", File: file, Segment: segment.ID, Message: glossaryErr.Error()})
			}
			if message := requirementWarning(language, segment.Original, targetSegment.Original); message != "" {
				report.add(AuditFinding{Severity: "warning", Code: "requirement-language", File: file, Segment: segment.ID, Message: message})
			}
		}
	}
	sort.SliceStable(report.Findings, func(i, j int) bool {
		if report.Findings[i].File == report.Findings[j].File {
			return report.Findings[i].Segment < report.Findings[j].Segment
		}
		return report.Findings[i].File < report.Findings[j].File
	})
	return report, pairs, nil
}

func unchangedSegmentProse(segment Segment, target, siteTitle string) (string, bool) {
	terms := []string{siteTitle}
	for _, original := range segment.Placeholders {
		terms = append(terms, original)
	}
	return unchangedProse(segment.Kind, segment.Original, target, terms...)
}

var auditCodeSpanPattern = regexp.MustCompile("`[^`]*`")
var auditWordPattern = regexp.MustCompile(`[\pL]+(?:['’][\pL]+)?`)

func unchangedProse(kind, source, target string, protectedTerms ...string) (string, bool) {
	source = strings.Join(strings.Fields(source), " ")
	target = strings.Join(strings.Fields(target), " ")
	if source != target || len([]rune(source)) < 8 {
		return "", false
	}
	if !strings.ContainsAny(source, " \t") && strings.ContainsAny(source, "./\\") {
		return "", false
	}
	plain := auditCodeSpanPattern.ReplaceAllString(source, "")
	for _, term := range protectedTerms {
		if term = strings.TrimSpace(term); term != "" {
			plain = strings.ReplaceAll(plain, term, "")
		}
	}
	words := auditWordPattern.FindAllString(plain, -1)
	if kind == "filetree" && len(words) > 0 {
		return "error", true
	}
	if len(words) >= 3 {
		return "error", true
	}
	if len(words) == 2 {
		return "warning", true
	}
	return "", false
}

var frenchRequirementRules = []struct {
	source *regexp.Regexp
	target *regexp.Regexp
	name   string
}{
	{regexp.MustCompile(`(?i)\brequires?\b`), regexp.MustCompile(`(?i)\b(?:exig|nécessit|requier|requis)`), "requires"},
	{regexp.MustCompile(`(?i)\brejects?\b`), regexp.MustCompile(`(?i)\b(?:rejet|refus)`), "rejects"},
	{regexp.MustCompile(`(?i)\bmust\b`), regexp.MustCompile(`(?i)\b(?:must|doi(?:t|vent)|dev(?:ez|ra|ront)|oblig|requis)`), "must"},
	{regexp.MustCompile(`(?i)\bcannot\b`), regexp.MustCompile(`(?i)(?:ne |n’).*(?:peut|peuv|puiss|possible)|impossible`), "cannot"},
}

func requirementWarning(language, source, target string) string {
	if !strings.HasPrefix(strings.ToLower(language), "fr") {
		return ""
	}
	for _, rule := range frenchRequirementRules {
		if rule.source.MatchString(source) && !rule.target.MatchString(target) {
			return fmt.Sprintf("check that the source requirement %q keeps the same force", rule.name)
		}
	}
	return ""
}
