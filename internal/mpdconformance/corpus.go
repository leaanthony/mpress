// Package mpdconformance provides the executable conformance corpus for the
// M-Press Flavoured Markdown profile. The corpus is data, not a parser implementation: an
// MPD parser can consume the sources and compare its rendered output with the
// checked-in HTML fixtures.
package mpdconformance

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

const (
	LevelTop    = "top"
	LevelNested = "nested"
)

//go:embed testdata
var embedded embed.FS

//go:embed viewer.css
var viewerCSS string

// Manifest describes one version of the corpus.
type Manifest struct {
	Version  int           `json:"version"`
	Fixtures []FixtureMeta `json:"fixtures"`
}

// FixtureMeta describes the syntax and coverage represented by a fixture.
type FixtureMeta struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	Level            string   `json:"level"`
	Features         []string `json:"features"`
	Components       []string `json:"components,omitempty"`
	ParentComponent  string   `json:"parentComponent,omitempty"`
	NestedComponents []string `json:"nestedComponents,omitempty"`
}

// Fixture contains an MPD source, its Markdown interchange reference, and the
// production-renderer HTML expected from both documents.
type Fixture struct {
	FixtureMeta
	MPD      string
	Markdown string
	HTML     string
}

// Corpus is an ordered, validated set of fixtures.
type Corpus struct {
	Version  int
	Fixtures []Fixture
}

// Embedded loads the checked-in corpus bundled with this package.
func Embedded() (*Corpus, error) {
	return LoadFS(embedded, "testdata", true)
}

// LoadFS loads a corpus rooted at base. When requireHTML is false, missing
// expected HTML is allowed so the golden generator can create it.
func LoadFS(source fs.FS, base string, requireHTML bool) (*Corpus, error) {
	manifestBytes, err := fs.ReadFile(source, path.Join(base, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read MPD corpus manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode MPD corpus manifest: %w", err)
	}
	corpus := &Corpus{Version: manifest.Version, Fixtures: make([]Fixture, 0, len(manifest.Fixtures))}
	for _, meta := range manifest.Fixtures {
		fixture := Fixture{FixtureMeta: meta}
		fixture.MPD, err = readFixtureFile(source, base, meta.ID, ".mpd", true)
		if err != nil {
			return nil, err
		}
		fixture.Markdown, err = readFixtureFile(source, base, meta.ID, ".md", true)
		if err != nil {
			return nil, err
		}
		fixture.HTML, err = readFixtureFile(source, base, meta.ID, ".html", requireHTML)
		if err != nil {
			return nil, err
		}
		corpus.Fixtures = append(corpus.Fixtures, fixture)
	}
	if err := corpus.Validate(); err != nil {
		return nil, err
	}
	return corpus, nil
}

func readFixtureFile(source fs.FS, base, id, extension string, required bool) (string, error) {
	name := path.Join(base, "corpus", id+extension)
	data, err := fs.ReadFile(source, name)
	if err != nil {
		if !required && errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read fixture %s: %w", name, err)
	}
	return string(data), nil
}

// Validate checks the corpus structure independently of any MPD parser.
func (c *Corpus) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported MPD corpus version %d", c.Version)
	}
	if len(c.Fixtures) == 0 {
		return fmt.Errorf("MPD corpus has no fixtures")
	}
	seen := make(map[string]bool, len(c.Fixtures))
	for index, fixture := range c.Fixtures {
		if fixture.ID == "" || fixture.ID != path.Clean(fixture.ID) || strings.HasPrefix(fixture.ID, "/") || strings.HasPrefix(fixture.ID, "../") {
			return fmt.Errorf("fixture %d has unsafe id %q", index, fixture.ID)
		}
		if seen[fixture.ID] {
			return fmt.Errorf("duplicate fixture id %q", fixture.ID)
		}
		seen[fixture.ID] = true
		if strings.TrimSpace(fixture.Title) == "" || strings.TrimSpace(fixture.Category) == "" {
			return fmt.Errorf("fixture %q needs a title and category", fixture.ID)
		}
		if fixture.Level != LevelTop && fixture.Level != LevelNested {
			return fmt.Errorf("fixture %q has invalid level %q", fixture.ID, fixture.Level)
		}
		if len(fixture.Features) == 0 {
			return fmt.Errorf("fixture %q has no feature tags", fixture.ID)
		}
		if strings.TrimSpace(fixture.MPD) == "" {
			return fmt.Errorf("fixture %q has an empty MPD source", fixture.ID)
		}
		if strings.Contains(fixture.MPD, "@mpress-document{") || hasLegacyMetadataBlock(fixture.MPD) {
			return fmt.Errorf("fixture %q uses a removed MPD preamble", fixture.ID)
		}
		if line, message := invalidCanonicalDirective(fixture.MPD); line != 0 {
			return fmt.Errorf("fixture %q line %d %s", fixture.ID, line, message)
		}
		lines := strings.Split(fixture.MPD, "\n")
		if lines[0] == "---" {
			if len(lines) < 3 || lines[1] != "schema = 1" {
				return fmt.Errorf("fixture %q metadata does not start with schema = 1", fixture.ID)
			}
			closed := false
			for _, line := range lines[2:] {
				if line == "---" {
					closed = true
					break
				}
			}
			if !closed {
				return fmt.Errorf("fixture %q has unclosed metadata", fixture.ID)
			}
		}
		for lineNumber, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "@end") && trimmed != "@end" && trimmed != `\@end` {
				return fmt.Errorf("fixture %q line %d does not place @end on its own logical line", fixture.ID, lineNumber+1)
			}
		}
		if strings.TrimSpace(fixture.Markdown) == "" {
			return fmt.Errorf("fixture %q has an empty Markdown reference", fixture.ID)
		}
		if strings.Contains(fixture.Markdown, "import {") || strings.Contains(fixture.Markdown, "export default") {
			return fmt.Errorf("fixture %q contains MDX-like module syntax", fixture.ID)
		}
		if fixture.Level == LevelNested && fixture.ParentComponent == "" {
			return fmt.Errorf("nested fixture %q has no parent component", fixture.ID)
		}
	}
	return nil
}

func hasLegacyMetadataBlock(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		if strings.TrimSpace(line) == "@meta" {
			return true
		}
	}
	return false
}

func invalidCanonicalDirective(source string) (int, string) {
	lines := strings.Split(source, "\n")
	fenceWidth := 0
	literalBlock := false
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceWidth > 0 {
			if run := leadingBackticks(trimmed); run >= fenceWidth && strings.TrimSpace(trimmed[run:]) == "" {
				fenceWidth = 0
			}
			continue
		}
		if literalBlock {
			if trimmed == "@end" {
				literalBlock = false
			}
			continue
		}
		if run := leadingBackticks(trimmed); run >= 3 {
			fenceWidth = run
			continue
		}
		name, directive := displayDirectiveName(trimmed)
		if !directive {
			continue
		}
		rest := trimmed[len(name)+1:]
		if name == "raw" {
			return index + 1, "uses removed @raw syntax; use @rawHTML"
		}
		if name == "rawHTML" && strings.TrimSpace(rest) != "" {
			return index + 1, "adds attributes to @rawHTML"
		}
		if strings.HasPrefix(rest, "[") {
			continue
		}
		if strings.HasPrefix(rest, "{") {
			return index + 1, "uses braces around MPD component attributes"
		}
		if strings.HasSuffix(rest, "/") {
			return index + 1, "uses removed / leaf marker"
		}
		if name == "rawHTML" || name == "comment" {
			literalBlock = true
		}
	}
	return 0, ""
}

// Categories returns the category names in lexical order.
func (c *Corpus) Categories() []string {
	set := make(map[string]bool)
	for _, fixture := range c.Fixtures {
		set[fixture.Category] = true
	}
	result := make([]string, 0, len(set))
	for category := range set {
		result = append(result, category)
	}
	sort.Strings(result)
	return result
}

// CountLevel returns the number of top-level or nested fixtures.
func (c *Corpus) CountLevel(level string) int {
	count := 0
	for _, fixture := range c.Fixtures {
		if fixture.Level == level {
			count++
		}
	}
	return count
}
