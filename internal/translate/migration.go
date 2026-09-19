package translate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/mpd"
)

// MigrationReport describes a sidecar-only operation. Content files are never
// written; ambiguous evidence requires human review.
type MigrationReport struct {
	Files   []MigrationFile `json:"files"`
	Written int             `json:"written"`
}
type MigrationFile struct {
	File          string         `json:"file"`
	Language      string         `json:"language"`
	Segments      int            `json:"segments"`
	Conflicts     int            `json:"requiresReview"`
	Unchanged     bool           `json:"unchanged"`
	ReviewReasons map[string]int `json:"reviewReasons,omitempty"`
}
type migrationWrite struct {
	path  string
	state *FileState
}

// MigrateState reads original source/target/state triplets from a project
// snapshot. All files are planned before the first sidecar is written.
func (e *Engine) MigrateState(from, language, file string, write bool) (MigrationReport, error) {
	oldConfig, err := config.Load(from)
	if err != nil {
		return MigrationReport{}, err
	}
	languages, err := e.targetLanguages(language)
	if err != nil {
		return MigrationReport{}, err
	}
	files, err := e.sourceFiles(file)
	if err != nil {
		return MigrationReport{}, err
	}
	report := MigrationReport{}
	var writes []migrationWrite
	for _, lang := range languages {
		for _, name := range files {
			path, err := statePath(e.Project, e.Config.Translation.StateDir, lang, name)
			if err != nil {
				return report, err
			}
			oldPath, err := statePath(from, oldConfig.Translation.StateDir, lang, name)
			if err != nil {
				return report, err
			}
			if _, err = os.Stat(oldPath); os.IsNotExist(err) {
				continue
			} else if err != nil {
				return report, err
			}
			old, err := loadState(oldPath, name, oldConfig.Translation.SourceLanguage, lang, "")
			if err != nil {
				return report, err
			}
			destination, err := loadState(path, name, e.Config.Translation.SourceLanguage, lang, "")
			if err != nil {
				return report, err
			}
			item := MigrationFile{File: name, Language: lang}
			if destination.SchemaVersion == stateSchemaVersion && destination.Extractor == currentExtractor(name) && len(destination.Segments) > 0 {
				item.Unchanged = true
				report.Files = append(report.Files, item)
				continue
			}
			state, conflicts, err := e.migrateTriplet(from, oldConfig, name, lang, old)
			if err != nil {
				return report, fmt.Errorf("migrate %s/%s: %w", lang, name, err)
			}
			item.Segments, item.Conflicts = len(state.Segments), conflicts
			item.ReviewReasons = map[string]int{}
			for _, reason := range state.Migration.ReviewReasons {
				item.ReviewReasons[reason]++
			}
			report.Files = append(report.Files, item)
			writes = append(writes, migrationWrite{path, state})
		}
	}
	if write {
		for _, item := range writes {
			if err := saveState(item.path, item.state); err != nil {
				return report, err
			}
			report.Written++
		}
	}
	return report, nil
}

func extractHistorical(file, nav string, source []byte, extractor string) (*Document, error) {
	if extractor == "markdown-text-v1" {
		return Extract(source)
	}
	if extractor != currentExtractor(file) {
		return nil, fmt.Errorf("unsupported historical extractor %q for %s", extractor, file)
	}
	return extractFile(file, nav, source)
}

func convertMigrationDocument(from, to string, source []byte, markdownToMPD func(string) (string, error)) ([]byte, error) {
	fromMPD, toMPD := strings.EqualFold(filepath.Ext(from), ".mpd"), strings.EqualFold(filepath.Ext(to), ".mpd")
	if fromMPD == toMPD {
		return source, nil
	}
	if fromMPD {
		return mpd.Markdown(mpd.Parse(from, source))
	}
	if markdownToMPD == nil {
		return nil, fmt.Errorf("Markdown to MPD state migration requires the project converter")
	}
	result, err := markdownToMPD(string(source))
	return []byte(result), err
}

// Use markers through the real converter to discover many-to-one mappings.
// This avoids guessing that an ordinal ID or similar text means the same thing.
var migrationMarker = regexp.MustCompile(`MPRESSMIGRATION([0-9a-f]{12})END`)

func migrationContributors(old *Document, from, to, nav string, markdownToMPD func(string) (string, error)) (map[string][]string, error) {
	values := map[string]string{}
	ids := map[string]string{}
	for _, s := range old.Segments {
		key := Hash(s.ID)[:12]
		ids[key] = s.ID
		// Insert into prose while retaining inline syntax and component field
		// prefixes. Replacing a whole link would change the converted block shape.
		if marked, ok := insertMigrationMarker(s, "MPRESSMIGRATION"+key+"END"); ok {
			values[s.ID] = marked
		}
	}
	marked, err := Apply(old, values)
	if err != nil {
		return nil, err
	}
	baseline, err := convertMigrationDocument(from, to, old.Source, markdownToMPD)
	if err != nil {
		return nil, err
	}
	original, err := extractFile(to, nav, baseline)
	if err != nil {
		return nil, err
	}
	marked, err = convertMigrationDocument(from, to, marked, markdownToMPD)
	if err != nil {
		return nil, err
	}
	doc, err := extractFile(to, nav, marked)
	if err != nil {
		return nil, err
	}
	if migrationSkeleton(doc) != migrationSkeleton(original) {
		return map[string][]string{}, nil
	}
	result := map[string][]string{}
	for _, s := range doc.Segments {
		for _, m := range migrationMarker.FindAllStringSubmatch(s.Original, -1) {
			result[s.ID] = append(result[s.ID], ids[m[1]])
		}
	}
	return result, nil
}

func insertMigrationMarker(segment Segment, marker string) (string, bool) {
	placeholders := translationPlaceholderPattern.FindAllStringIndex(segment.Text, -1)
	for offset, r := range segment.Text {
		protected := false
		for _, span := range placeholders {
			if offset >= span[0] && offset < span[1] {
				protected = true
				break
			}
		}
		if !protected && unicode.IsLetter(r) {
			return segment.Text[:offset] + marker + segment.Text[offset:], true
		}
	}
	return "", false
}

func migrationSegments(doc *Document) map[string]Segment {
	result := map[string]Segment{}
	for _, s := range doc.Segments {
		result[s.ID] = s
	}
	return result
}

func (e *Engine) migrateTriplet(from string, oldConfig config.Config, file, language string, old *FileState) (*FileState, int, error) {
	oldFile := old.SourceFile
	if _, err := statePath(from, oldConfig.Translation.StateDir, language, oldFile); err != nil {
		return nil, 0, err
	}
	if old.TargetLanguage != language || old.SourceLanguage != e.Config.Translation.SourceLanguage {
		return nil, 0, fmt.Errorf("sidecar languages do not match the requested migration")
	}
	read := func(root, name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	}
	oldSource, err := read(oldConfig.ContentPath(from), oldFile)
	if err != nil {
		return nil, 0, err
	}
	oldTarget, err := read(oldConfig.ContentPath(from), filepath.ToSlash(filepath.Join(language, oldFile)))
	if err != nil {
		return nil, 0, err
	}
	source, err := read(e.Config.ContentPath(e.Project), file)
	if err != nil {
		return nil, 0, err
	}
	target, err := read(e.Config.ContentPath(e.Project), filepath.ToSlash(filepath.Join(language, file)))
	if err != nil {
		return nil, 0, err
	}
	return migrateDocuments(migrationPair{oldFile, oldConfig.Build.NavFile, oldSource, oldTarget}, migrationPair{file, e.Config.Build.NavFile, source, target}, old, e.Config.Translation.SourceLanguage, e.MarkdownToMPD)
}

type migrationPair struct {
	file, nav      string
	source, target []byte
}

func migrateDocuments(previous, current migrationPair, old *FileState, sourceLanguage string, markdownToMPD func(string) (string, error)) (*FileState, int, error) {
	oldFile, file, language := previous.file, current.file, old.TargetLanguage
	oldSource, oldTarget, source, target := previous.source, previous.target, current.source, current.target
	oldSourceDoc, err := extractHistorical(oldFile, previous.nav, oldSource, old.Extractor)
	if err != nil {
		return nil, 0, err
	}
	oldTargetDoc, err := extractHistorical(oldFile, previous.nav, oldTarget, old.Extractor)
	if err != nil {
		return nil, 0, err
	}
	if oldSourceDoc.Format == "mpd" {
		oldTargetDoc, err = migrationTarget(oldFile, previous.nav, oldSource, oldTarget)
		if err != nil {
			return nil, 0, err
		}
	}
	baselineSource, err := convertMigrationDocument(oldFile, file, oldSource, markdownToMPD)
	if err != nil {
		return nil, 0, err
	}
	baselineTarget, err := convertMigrationDocument(oldFile, file, oldTarget, markdownToMPD)
	if err != nil {
		return nil, 0, err
	}
	baselineDoc, err := extractFile(file, current.nav, baselineSource)
	if err != nil {
		return nil, 0, err
	}
	baselineTargetDoc, err := migrationTarget(file, current.nav, baselineSource, baselineTarget)
	if err != nil {
		return nil, 0, err
	}
	sourceDoc, err := extractFile(file, current.nav, source)
	if err != nil {
		return nil, 0, err
	}
	targetDoc, err := migrationTarget(file, current.nav, source, target)
	if err != nil {
		return nil, 0, err
	}
	contributors, err := migrationContributors(oldSourceDoc, oldFile, file, current.nav, markdownToMPD)
	if err != nil {
		return nil, 0, err
	}
	oldSources, oldTargets := migrationSegments(oldSourceDoc), migrationSegments(oldTargetDoc)
	// Code-only prose has nowhere to insert a marker. Exact, unique source AND
	// target content is an independent witness; ambiguous duplicates stay pending.
	exactOrigins := map[string][]string{}
	for id, segment := range oldSources {
		if target, ok := oldTargets[id]; ok {
			key := segment.SourceHash + "\x00" + Hash(target.Original)
			exactOrigins[key] = append(exactOrigins[key], id)
		}
	}
	baselines, baselineTargets, targets := migrationSegments(baselineDoc), migrationSegments(baselineTargetDoc), migrationSegments(targetDoc)
	state := newFileState(file, sourceLanguage, language, sourceDoc.TranslationKey)
	state.PageKey = old.PageKey
	state.Migration = &StateMigration{FromExtractor: old.Extractor, FromSourceFile: old.SourceFile, SourceHash: Hash(string(oldSource)), TargetHash: Hash(string(oldTarget)), PreviousSegments: old.Segments, ReviewReasons: map[string]string{}}
	conflicts := 0
	// An incompatible layout cannot safely establish target segment identity.
	layoutOK := migrationSkeleton(baselineDoc) == migrationSkeleton(sourceDoc) && migrationSkeleton(baselineTargetDoc) == migrationSkeleton(targetDoc) && migrationSkeleton(baselineDoc) == migrationSkeleton(baselineTargetDoc)
	var alignment map[string]string
	if !layoutOK && migrationSkeleton(baselineDoc) == migrationSkeleton(baselineTargetDoc) {
		alignment = insertedParagraphAlignment(baselineDoc, baselineTargetDoc, sourceDoc, targetDoc)
		layoutOK = alignment != nil
	}
	for _, s := range sourceDoc.Segments {
		baselineID := s.ID
		if alignment != nil {
			baselineID = alignment[s.ID]
		}
		origins := contributors[baselineID]
		if len(origins) == 0 {
			if exact := exactOrigins[s.SourceHash+"\x00"+Hash(targets[s.ID].Original)]; len(exact) == 1 {
				origins = exact
			}
		}
		entry, proven := migrateSegment(origins, old.Segments, oldSources, oldTargets)
		proven = proven && layoutOK && baselines[baselineID].SourceHash == s.SourceHash && targets[s.ID].Original != ""
		targetText := targets[s.ID].Original
		if !proven && s.Protected && targetText == s.Original {
			state.Segments[s.ID] = SegmentState{SourceHash: s.SourceHash, TargetHash: Hash(targetText), Status: "protected", SourcePreview: preview(s.Original)}
			continue
		}
		if !proven {
			entry = SegmentState{Status: "migration-review", SourceHash: s.SourceHash, TargetHash: Hash(targetText)}
			conflicts++
			reason := "unverified-mapping"
			switch {
			case migrationSkeleton(baselineDoc) != migrationSkeleton(baselineTargetDoc):
				reason = "preexisting-layout-mismatch"
			case !layoutOK:
				reason = "current-layout-mismatch"
			case alignment != nil && baselineID == "":
				reason = "inserted-content"
			case baselines[baselineID].SourceHash != s.SourceHash:
				reason = "source-changed-since-snapshot"
			case len(origins) == 0:
				reason = "unmapped-content"
			case targetText == "":
				reason = "missing-target"
			default:
				for _, id := range origins {
					entry, exists := old.Segments[id]
					if !exists {
						reason = "untracked-original"
						break
					}
					if entry.SourceHash != oldSources[id].SourceHash {
						reason = "stale-original"
						break
					}
					if oldTargets[id].Original == "" {
						reason = "missing-original-target"
						break
					}
				}
			}
			state.Migration.ReviewReasons[s.ID] = reason
		} else {
			entry.SourceHash = s.SourceHash
			entry.TargetHash = Hash(targetText)
			if targetText != baselineTargets[baselineID].Original {
				entry.Status = "manual"
				entry.MachineHash = ""
				entry.MachineText = ""
			}
			if entry.MachineHash != "" {
				entry.MachineHash = Hash(targetText)
				entry.MachineText = targetText
			}
		}
		entry.SourcePreview = preview(s.Original)
		state.Segments[s.ID] = entry
	}
	return state, conflicts, nil
}

func migrateSegment(ids []string, states map[string]SegmentState, sources, targets map[string]Segment) (SegmentState, bool) {
	if len(ids) == 0 {
		return SegmentState{}, false
	}
	var result SegmentState
	for i, id := range ids {
		entry, ok := states[id]
		source, target := sources[id], targets[id]
		if !ok || source.SourceHash != entry.SourceHash || target.Original == "" {
			return SegmentState{}, false
		}
		if entry.TargetHash != Hash(target.Original) {
			entry.Status = "manual"
			entry.MachineHash = ""
			entry.MachineText = ""
		}
		if entry.MachineHash != Hash(target.Original) {
			entry.MachineHash = ""
			entry.MachineText = ""
		}
		if i == 0 {
			result = entry
			continue
		}
		if result.Status != entry.Status {
			if (result.Status == "final" || result.Status == "reviewed") && (entry.Status == "final" || entry.Status == "reviewed") {
				result.Status = "reviewed"
			} else {
				result.Status = "manual"
			}
		}
		if result.Provider != entry.Provider || result.Model != entry.Model || result.PromptVersion != entry.PromptVersion {
			result.Provider = ""
			result.Model = ""
			result.PromptVersion = ""
		}
		if entry.MachineHash == "" {
			result.MachineHash = ""
			result.MachineText = ""
		}
		if entry.UpdatedAt > result.UpdatedAt {
			result.UpdatedAt = entry.UpdatedAt
		}
	}
	return result, true
}

func (report MigrationReport) JSON() string {
	data, _ := json.MarshalIndent(report, "", "  ")
	return string(data)
}

func migrationTarget(file, nav string, source, target []byte) (*Document, error) {
	doc, err := extractTargetFile(file, nav, source, target)
	if errors.Is(err, errTranslationHeadingStructure) {
		return extractFile(file, nav, target)
	}
	return doc, err
}

var migrationSourcePath = regexp.MustCompile(`(?m)^(sourcePath:.*)\.(?:mpd|markdown|md)(["']?\r?)$`)

func migrationSkeleton(doc *Document) string {
	return normalizeMigrationSourcePath(immutableSkeleton(doc))
}

func normalizeMigrationSourcePath(source string) string {
	end := frontmatterEnd([]byte(source))
	if end == 0 {
		return source
	}
	return migrationSourcePath.ReplaceAllString(source[:end], "${1}.document${2}") + source[end:]
}
