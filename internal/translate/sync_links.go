package translate

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// SyncLinks applies only an explicitly supplied, URL-only source revision to
// one existing Markdown translation. It preserves target prose byte for byte.
// State and exceptions are deliberately not changed: this is not a review.
func (e *Engine) SyncLinks(language, file string, previous []byte, write bool) (bool, error) {
	languages, err := e.targetLanguages(language)
	if err != nil {
		return false, err
	}
	if len(languages) != 1 {
		return false, fmt.Errorf("sync-links requires one --lang")
	}
	files, closeFiles, err := e.projectFiles()
	if err != nil {
		return false, err
	}
	defer closeFiles()
	sources, err := e.sourceFilesWithRoot(files, file)
	if err != nil {
		return false, err
	}
	if file == "" || len(sources) != 1 || filepath.Ext(sources[0]) != ".md" {
		return false, fmt.Errorf("sync-links requires one Markdown --file")
	}
	source := filepath.Join(e.Config.ContentPath(e.Project), filepath.FromSlash(sources[0]))
	target := filepath.Join(e.Config.ContentPath(e.Project), language, filepath.FromSlash(sources[0]))
	current, err := files.ReadFile(source)
	if err != nil {
		return false, err
	}
	translated, err := files.ReadFile(target)
	if err != nil {
		return false, err
	}
	output, err := syncMarkdownLinks(previous, current, translated)
	if err != nil {
		return false, err
	}
	changed := !bytes.Equal(output, translated)
	if changed && write {
		err = files.WriteAtomic(target, output)
	}
	return changed, err
}

type linkEdit struct {
	start, end int
	value      string
}

// Only simple inline Markdown HTTP(S) destinations are supported. Requiring an
// exact reconstruction of the source revision rules out prose, code, layout,
// metadata and ambiguous syntax changes rather than guessing their intention.
func syncMarkdownLinks(previous, current, target []byte) ([]byte, error) {
	before, err := ExtractMarkdown(previous)
	if err != nil {
		return nil, err
	}
	after, err := ExtractMarkdown(current)
	if err != nil {
		return nil, err
	}
	if len(before.Segments) != len(after.Segments) {
		return nil, fmt.Errorf("source revision changes more than inline link destinations")
	}
	replacements, err := sourceLinkCorrections(before, after, previous, current)
	if err != nil {
		return nil, err
	}
	if len(replacements) == 0 {
		return target, nil
	}
	if err := Validate(before, target); err != nil {
		return nil, fmt.Errorf("translation does not match previous English structure; review manually: %w", err)
	}
	translated, err := ExtractMarkdown(target)
	if err != nil {
		return nil, err
	}
	var edits []linkEdit
	for _, segment := range translated.Segments {
		for _, link := range inlineLinkEdits(segment) {
			if replacement, ok := replacements[link.value]; ok {
				edits = append(edits, linkEdit{link.start, link.end, replacement})
			}
		}
	}
	output := applyLinkEdits(target, edits)
	if err := Validate(after, output); err != nil {
		return nil, fmt.Errorf("corrected translation fails validation; no file written: %w", err)
	}
	return output, nil
}

func inlineLinkEdits(segment Segment) []linkEdit {
	var links []linkEdit
	offset := 0
	for _, token := range orderedPlaceholderTokens(segment) {
		original := segment.Placeholders[token]
		start := segment.Start + strings.Index(segment.Text, token) + offset
		offset += len(original) - len(token)
		if !strings.HasPrefix(original, "](") || !strings.HasSuffix(original, ")") {
			continue
		}
		destination := original[2 : len(original)-1]
		parsed, err := url.Parse(destination)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil || strings.ContainsAny(destination, " ()<>\t\n\\") {
			continue
		}
		links = append(links, linkEdit{start + 2, start + len(original) - 1, destination})
	}
	return links
}

func applyLinkEdits(source []byte, edits []linkEdit) []byte {
	var result bytes.Buffer
	cursor := 0
	for _, edit := range edits {
		result.Write(source[cursor:edit.start])
		result.WriteString(edit.value)
		cursor = edit.end
	}
	result.Write(source[cursor:])
	return result.Bytes()
}

func sourceLinkCorrections(before, after *Document, previous, current []byte) (map[string]string, error) {
	replacements := map[string]string{}
	var edits []linkEdit
	for i, old := range before.Segments {
		next := after.Segments[i]
		oldLinks := inlineLinkEdits(old)
		newLinks := inlineLinkEdits(next)
		if old.ID != next.ID || len(oldLinks) != len(newLinks) {
			return nil, fmt.Errorf("source revision changes link structure")
		}
		for j, link := range oldLinks {
			replacement := newLinks[j].value
			if link.value == replacement {
				continue
			}
			if other, ok := replacements[link.value]; ok && other != replacement {
				return nil, fmt.Errorf("ambiguous correction for %s", link.value)
			}
			replacements[link.value] = replacement
			edits = append(edits, linkEdit{link.start, link.end, replacement})
		}
	}
	if !bytes.Equal(applyLinkEdits(previous, edits), current) {
		return nil, fmt.Errorf("source revision must change only simple inline HTTP(S) link destinations")
	}
	return replacements, nil
}
