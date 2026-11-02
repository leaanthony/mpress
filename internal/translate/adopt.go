package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/projectfs"
)

// statusAdopted marks a segment whose translation existed before M-Press
// tracked it. It claims no human review: segmentStatus reports it as "adopted"
// while its source is unchanged and "stale" once the source moves on, the
// default scopes leave it alone, and --scope all without --force does not
// replace it. Only --force regenerates adopted text.
const statusAdopted = "adopted"

// adoptedProvider is the Provider recorded on adopted segments.
const adoptedProvider = "adopted"

// adoptWorkingTree is AdoptedFrom for an --accept-drift adoption.
const adoptWorkingTree = "working-tree"

// AdoptOptions selects what Adopt bootstraps.
type AdoptOptions struct {
	Language string
	File     string
	// Since is the Git revision the translations were made from. Source
	// segments that changed after it stay stale rather than being adopted.
	Since string
	// AcceptDrift adopts against the working tree without --since, silently
	// accepting any drift between the source and its translations.
	AcceptDrift bool
	DryRun      bool
}

// Adopt outcomes.
const (
	AdoptAdopted        = "adopted"
	AdoptAlreadyTracked = "already-tracked"
	AdoptNoTarget       = "no-target"
	AdoptNotAtRev       = "source-not-at-rev"
	AdoptRefused        = "refused"
)

// AdoptFileReport is the outcome for one page and language.
type AdoptFileReport struct {
	SourceFile string `json:"sourceFile"`
	TargetFile string `json:"targetFile"`
	Language   string `json:"language"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason,omitempty"`
	Segments   int    `json:"segments"`
	// Adopted segments are current: their source is unchanged since the revision.
	Adopted int `json:"adopted"`
	// Drifted segments changed after the revision; they are recorded against
	// their old source text so status reports them stale.
	Drifted int `json:"drifted"`
	// New segments have no counterpart at the revision; status reports them missing.
	New     int  `json:"new"`
	Written bool `json:"written"`
}

// AdoptReport summarises an adopt run.
type AdoptReport struct {
	Revision string            `json:"revision"`
	DryRun   bool              `json:"dryRun"`
	Files    []AdoptFileReport `json:"files"`
	Adopted  int               `json:"adopted"`
	Drifted  int               `json:"drifted"`
	New      int               `json:"new"`
	Written  int               `json:"written"`
	Refused  int               `json:"refused"`
}

// JSON renders the report.
func (r AdoptReport) JSON() string {
	data, _ := json.MarshalIndent(r, "", "  ")
	return string(data)
}

// Adopt bootstraps tracking state for translations that already exist as
// plain content. It writes sidecars only: translated content is never changed.
//
// The sidecar is built as it would have been had the page been tracked when the
// translation was made: each target segment is paired with the source segment
// it was translated from, using the source at the --since revision. The
// ordinary status machinery then reports unchanged segments as adopted,
// changed ones as stale and inserted ones as missing, and handles moved
// segments exactly as it does for any tracked page.
func (e *Engine) Adopt(options AdoptOptions) (AdoptReport, error) {
	since := strings.TrimSpace(options.Since)
	if since == "" && !options.AcceptDrift {
		return AdoptReport{}, errors.New("adopt needs --since <git rev> so changes made after the translation stay stale; use --accept-drift to adopt against the current source and silently accept any drift")
	}
	if since != "" && options.AcceptDrift {
		return AdoptReport{}, errors.New("--since and --accept-drift are mutually exclusive")
	}
	revision := adoptWorkingTree
	if since != "" {
		resolved, err := gitResolveCommit(e.Project, since)
		if err != nil {
			return AdoptReport{}, err
		}
		revision = resolved
	}
	languages, err := e.targetLanguages(options.Language)
	if err != nil {
		return AdoptReport{}, err
	}
	files, err := projectfs.Open(e.Project)
	if err != nil {
		return AdoptReport{}, err
	}
	defer files.Close()
	sources, err := e.sourceFilesWithRoot(files, options.File)
	if err != nil {
		return AdoptReport{}, err
	}
	report := AdoptReport{Revision: revision, DryRun: options.DryRun}
	for _, language := range languages {
		for _, sourceFile := range sources {
			item, err := e.adoptFile(files, language, sourceFile, revision, options.DryRun)
			if err != nil {
				return report, err
			}
			report.Files = append(report.Files, item)
			report.Adopted += item.Adopted
			report.Drifted += item.Drifted
			report.New += item.New
			if item.Written {
				report.Written++
			}
			if item.Outcome == AdoptRefused {
				report.Refused++
			}
		}
	}
	return report, nil
}

func (e *Engine) adoptFile(files *projectfs.FS, language, sourceFile, revision string, dryRun bool) (AdoptFileReport, error) {
	contentDir := e.Config.ContentPath(e.Project)
	item := AdoptFileReport{SourceFile: filepath.ToSlash(sourceFile), TargetFile: filepath.ToSlash(filepath.Join(language, sourceFile)), Language: language}
	sourceBytes, err := files.ReadFile(filepath.Join(contentDir, filepath.FromSlash(sourceFile)))
	if err != nil {
		return item, err
	}
	sourceDoc, err := extractFile(sourceFile, e.Config.Build.NavFile, sourceBytes)
	if err != nil {
		return item, fmt.Errorf("%s: %w", sourceFile, err)
	}
	item.Segments = len(sourceDoc.Segments)

	targetBytes, err := files.ReadFile(filepath.Join(contentDir, filepath.FromSlash(language), filepath.FromSlash(sourceFile)))
	if errors.Is(err, os.ErrNotExist) {
		item.Outcome = AdoptNoTarget
		item.Reason = "no translated page exists"
		return item, nil
	}
	if err != nil {
		return item, fmt.Errorf("read translation target: %w", err)
	}

	state, stateFile, _, err := e.readReviewState(files, language, sourceFile, sourceDoc)
	if err != nil {
		return item, err
	}
	if len(state.Segments) > 0 {
		item.Outcome = AdoptAlreadyTracked
		item.Reason = "a translation sidecar already exists; adopt never overwrites tracking state"
		return item, nil
	}

	// baseBytes is the source the translation is assumed to be made from.
	baseBytes := sourceBytes
	if revision != adoptWorkingTree {
		rel, relErr := filepath.Rel(e.Project, filepath.Join(contentDir, filepath.FromSlash(sourceFile)))
		if relErr != nil {
			return item, relErr
		}
		baseBytes, err = gitShowFile(e.Project, revision, filepath.ToSlash(rel))
		if errors.Is(err, errNotAtRevision) {
			item.Outcome = AdoptNotAtRev
			item.Reason = "the source page did not exist at the revision"
			return item, nil
		}
		if err != nil {
			return item, err
		}
	}
	baseDoc, err := extractFile(sourceFile, e.Config.Build.NavFile, baseBytes)
	if err != nil {
		item.Outcome, item.Reason = AdoptRefused, fmt.Sprintf("source at revision does not parse: %v", err)
		return item, nil
	}
	// The same alignment checks as `translate review`, against the source the
	// translation was made from. Nothing is guessed when they fail.
	targetDoc, err := extractTargetFile(sourceFile, e.Config.Build.NavFile, baseBytes, targetBytes)
	if err == nil {
		err = validateFile(sourceFile, e.Config.Build.NavFile, baseDoc, targetBytes)
	}
	if err != nil {
		item.Outcome, item.Reason = AdoptRefused, err.Error()
		return item, nil
	}

	targetByID := make(map[string]Segment, len(targetDoc.Segments))
	for _, segment := range targetDoc.Segments {
		targetByID[segment.ID] = segment
	}
	adopted := newFileState(sourceFile, e.Config.Translation.SourceLanguage, language, sourceDoc.TranslationKey)
	now := nowString()
	for _, base := range baseDoc.Segments {
		target, ok := targetByID[base.ID]
		if !ok || target.Original == "" {
			continue
		}
		adopted.Segments[base.ID] = SegmentState{
			SourceHash: base.SourceHash, TargetHash: Hash(target.Original), Status: statusAdopted,
			SourcePreview: preview(base.Original), Provider: adoptedProvider, UpdatedAt: now, AdoptedFrom: revision,
		}
	}
	matches := matchSegments(sourceDoc.Segments, adopted.Segments)
	for _, segment := range sourceDoc.Segments {
		old, ok := adopted.Segments[matches[segment.ID]]
		switch {
		case !ok:
			item.New++
		case old.SourceHash == segment.SourceHash:
			item.Adopted++
		default:
			item.Drifted++
		}
	}
	item.Outcome = AdoptAdopted
	if dryRun {
		return item, nil
	}
	if err := saveStateRoot(files, stateFile, adopted); err != nil {
		return item, err
	}
	item.Written = true
	return item, nil
}

// RefusedPages lists the pages adopt would not guess about, sorted.
func (r AdoptReport) RefusedPages() []AdoptFileReport {
	var refused []AdoptFileReport
	for _, item := range r.Files {
		if item.Outcome == AdoptRefused {
			refused = append(refused, item)
		}
	}
	sort.Slice(refused, func(i, j int) bool {
		if refused[i].Language != refused[j].Language {
			return refused[i].Language < refused[j].Language
		}
		return refused[i].SourceFile < refused[j].SourceFile
	})
	return refused
}

var errNotAtRevision = errors.New("path does not exist at revision")

func gitCommand(project string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", project}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), strings.TrimSpace(stderr.String()), err
}

func gitResolveCommit(project, rev string) (string, error) {
	if strings.HasPrefix(rev, "-") {
		return "", fmt.Errorf("invalid Git revision %q", rev)
	}
	out, stderr, err := gitCommand(project, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve --since %q: not a commit in this repository (%s)", rev, strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(out), nil
}

// gitShowFile returns path (relative to the project directory) at commit.
func gitShowFile(project, commit, path string) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "git", "-C", project, "show", commit+":./"+path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		message := stderr.String()
		if strings.Contains(message, "does not exist") || strings.Contains(message, "exists on disk, but not in") {
			return nil, errNotAtRevision
		}
		return nil, fmt.Errorf("git show %s:%s: %s", commit, path, strings.TrimSpace(message))
	}
	return stdout.Bytes(), nil
}

// StateDirIgnored reports whether Git ignores the translation state directory,
// which silently loses tracking state on the next clone. It is false when the
// project is not in a Git work tree or Git is unavailable.
func StateDirIgnored(project, stateDir string) bool {
	probe := filepath.ToSlash(filepath.Join(stateDir, "xx", "probe.json"))
	cmd := exec.CommandContext(context.Background(), "git", "-C", project, "check-ignore", "-q", "--no-index", "--", probe)
	return cmd.Run() == nil
}
