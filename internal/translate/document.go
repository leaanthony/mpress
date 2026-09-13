package translate

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Segment is one exact, translatable byte range in a Markdown document.
// Markdown syntax around the range is never regenerated.
type Segment struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"`
	Section      string            `json:"section,omitempty"`
	Original     string            `json:"original"`
	Text         string            `json:"text"`
	Start        int               `json:"-"`
	End          int               `json:"-"`
	SourceHash   string            `json:"sourceHash"`
	Placeholders map[string]string `json:"-"`
	Encoding     string            `json:"-"`
	Protected    bool              `json:"-"`
	D2Key        string            `json:"-"`
	D2Tag        string            `json:"-"`
}

// EditableSegment is the compact subset of Segment required by browser quick
// editing and the parse cache.
type EditableSegment struct {
	ID         string `json:"id" glint:"id"`
	Kind       string `json:"kind" glint:"kind"`
	Original   string `json:"original" glint:"original"`
	Start      int    `json:"start" glint:"start"`
	End        int    `json:"end" glint:"end"`
	SourceHash string `json:"sourceHash" glint:"sourceHash"`
}

type Document struct {
	Source         []byte
	Segments       []Segment
	Title          string
	Outline        []string
	TranslationKey string
	Format         string
}

func ExtractNavigation(source []byte) (*Document, error) {
	var parsed any
	if err := yaml.Unmarshal(source, &parsed); err != nil {
		return nil, fmt.Errorf("parse navigation YAML: %w", err)
	}
	document := &Document{Source: append([]byte(nil), source...), Format: "navigation"}
	position := 0
	labelIndex := 0
	for position < len(source) {
		next := bytes.IndexByte(source[position:], '\n')
		lineEnd := len(source)
		if next >= 0 {
			lineEnd = position + next
		}
		line := source[position:lineEnd]
		if start := navigationLabelStart(line); start >= 0 {
			absoluteStart := position + start
			end := navigationLabelEnd(source, absoluteStart)
			var raw string
			if err := yaml.Unmarshal(source[absoluteStart:end], &raw); err != nil {
				return nil, fmt.Errorf("parse navigation label: %w", err)
			}
			if translatableText(raw) {
				labelIndex++
				protected, placeholders := protect(raw)
				document.Segments = append(document.Segments, Segment{ID: fmt.Sprintf("nav-label-%03d", labelIndex), Kind: "navigation", Original: raw, Text: protected, Start: absoluteStart, End: end, SourceHash: Hash(raw), Placeholders: placeholders, Encoding: "yaml-string"})
			}
			if end > lineEnd {
				lineEnd = end
				if next := bytes.IndexByte(source[end:], '\n'); next >= 0 {
					lineEnd = end + next
				}
			}
		}
		position = lineEnd + 1
	}
	return document, nil
}

func navigationLabelStart(line []byte) int {
	colon := bytes.IndexByte(line, ':')
	if colon <= 0 {
		return -1
	}
	key := strings.TrimSpace(string(line[:colon]))
	if strings.TrimSpace(strings.TrimPrefix(key, "-")) != "label" {
		return -1
	}
	start := colon + 1
	for start < len(line) && (line[start] == ' ' || line[start] == '\t') {
		start++
	}
	return start
}

var protectedPattern = regexp.MustCompile(`&(?:#[0-9]+|#x[0-9A-Fa-f]+|[A-Za-z][A-Za-z0-9]+);|\\[^\r\n]|\{\{[^\n{}]+\}\}|\$\{[^\n{}]+\}|\{[A-Za-z_][A-Za-z0-9_.-]*\}|%[-+#0-9.*]*[bcdeEfFgGopqstvxX]|\b[0-9]+(?:[.,:/-][0-9]+)*`)
var blockPrefixPattern = regexp.MustCompile(`^[ \t]*(?:[-+*][ \t]+|[0-9]+[.)][ \t]+|>+[ \t]*)`)
var parenthesizedMarkerSuffixPattern = regexp.MustCompile(`\([0-9]+\)[ \t]*$`)
var pricingMarkerSuffixPattern = regexp.MustCompile(`[✓✗][ \t]*$`)
var numberedProseSuffixPattern = regexp.MustCompile(`[0-9]+[.)][ \t]*$`)

var extractionParserPool = sync.Pool{New: func() any {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
	).Parser()
}}

// Extract parses Markdown with Goldmark, then records only literal text-node
// byte ranges. Code, HTML, link destinations, component declarations and all
// other syntax remain outside those ranges.
func Extract(source []byte) (*Document, error) {
	document, _, err := extract(source, false)
	return document, err
}

// ExtractEditable returns only visible body segments and the source ranges
// needed by quick editing. It deliberately skips translation metadata,
// outlines, section labels, protected text, frontmatter segments, and the
// source copy. The returned segments cannot be passed to Apply or Validate.
func ExtractEditable(source []byte) ([]EditableSegment, error) {
	_, segments, err := extract(source, true)
	return segments, err
}

func extract(source []byte, editableOnly bool) (*Document, []EditableSegment, error) {
	doc := &Document{Format: "markdown"}
	var editableSegments []EditableSegment
	if !editableOnly {
		doc.Source = append([]byte(nil), source...)
	}
	bodyStart := frontmatterEnd(source)
	if bodyStart > 0 && !editableOnly {
		var metadata struct {
			TranslationKey string `yaml:"translationKey"`
		}
		frontmatter := source
		firstLine := bytes.IndexByte(frontmatter, '\n')
		lastMarker := bytes.LastIndex(frontmatter[:bodyStart], []byte("---"))
		if firstLine >= 0 && lastMarker > firstLine {
			_ = yaml.Unmarshal(frontmatter[firstLine+1:lastMarker], &metadata)
			doc.TranslationKey = strings.TrimSpace(metadata.TranslationKey)
		}
	}
	body := source[bodyStart:]
	markdownParser := extractionParserPool.Get().(parser.Parser)
	root := markdownParser.Parse(text.NewReader(body))
	extractionParserPool.Put(markdownParser)

	type headingAt struct {
		start int
		level int
		text  string
	}
	var headings []headingAt
	if !editableOnly {
		_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
			if !entering {
				return ast.WalkContinue, nil
			}
			heading, ok := node.(*ast.Heading)
			if !ok {
				return ast.WalkContinue, nil
			}
			var value strings.Builder
			for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
				switch item := child.(type) {
				case *ast.Text:
					value.Write(item.Segment.Value(body))
				case *ast.CodeSpan:
					value.Write(item.Text(body))
				}
			}
			start := bodyStart
			if heading.Lines().Len() > 0 {
				start += heading.Lines().At(0).Start
			}
			plain := strings.TrimSpace(value.String())
			headings = append(headings, headingAt{start: start, level: heading.Level, text: plain})
			if plain != "" {
				doc.Outline = append(doc.Outline, plain)
				if doc.Title == "" && heading.Level == 1 {
					doc.Title = plain
				}
			}
			return ast.WalkContinue, nil
		})
	}

	type blockState struct {
		node ast.Node
		id   int
		text int
	}
	var blockStack []blockState
	var fallbackBlocks map[ast.Node]*blockState
	nextBlock := 0
	sectionAt := func(position int) string {
		stack := make([]string, 6)
		for _, heading := range headings {
			if heading.start >= position {
				break
			}
			if heading.level >= 1 && heading.level <= len(stack) {
				stack[heading.level-1] = heading.text
				for i := heading.level; i < len(stack); i++ {
					stack[i] = ""
				}
			}
		}
		var values []string
		for _, value := range stack {
			if value != "" {
				values = append(values, value)
			}
		}
		return strings.Join(values, " > ")
	}

	err := ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			if isTranslationBlock(node) && len(blockStack) > 0 && blockStack[len(blockStack)-1].node == node {
				blockStack = blockStack[:len(blockStack)-1]
			}
			return ast.WalkContinue, nil
		}
		if isTranslationBlock(node) {
			blockStack = append(blockStack, blockState{node: node})
		}
		item, ok := node.(*ast.Text)
		if !ok || excludedTextNode(node) {
			return ast.WalkContinue, nil
		}
		start, end := bodyStart+item.Segment.Start, bodyStart+item.Segment.Stop
		if start < bodyStart || end > len(source) || start >= end {
			return ast.WalkContinue, nil
		}
		raw := string(source[start:end])
		if lineStart := bytes.LastIndexByte(source[:start], '\n') + 1; strings.TrimSpace(string(source[lineStart:start])) == "" {
			if prefix := blockPrefixPattern.FindString(raw); prefix != "" {
				start += len(prefix)
				raw = raw[len(prefix):]
			}
		}
		leading := len(raw) - len(strings.TrimLeft(raw, " \t"))
		trailing := len(raw) - len(strings.TrimRight(raw, " \t"))
		start += leading
		end -= trailing
		raw = strings.Trim(raw, " \t")
		if !translatableText(raw) || componentSyntaxLine(source, start) {
			return ast.WalkContinue, nil
		}
		block := node.Parent()
		var state *blockState
		if len(blockStack) > 0 {
			state = &blockStack[len(blockStack)-1]
			block = state.node
		} else {
			if fallbackBlocks == nil {
				fallbackBlocks = make(map[ast.Node]*blockState)
			}
			state = fallbackBlocks[block]
			if state == nil {
				state = &blockState{node: block}
				fallbackBlocks[block] = state
			}
		}
		if state.id == 0 {
			nextBlock++
			state.id = nextBlock
		}
		state.text++
		kind := blockKind(block)
		id := blockSegmentID(state.id, kind, state.text)
		if editableOnly {
			editableSegments = append(editableSegments, EditableSegment{
				ID: id, Kind: kind, Original: raw, Start: start, End: end, SourceHash: Hash(raw),
			})
			return ast.WalkContinue, nil
		}
		protected, placeholders := protect(raw)
		doc.Segments = append(doc.Segments, Segment{
			ID: id, Kind: kind, Section: sectionAt(start), Original: raw, Text: protected,
			Start: start, End: end, SourceHash: Hash(raw), Placeholders: placeholders,
		})
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, nil, err
	}
	if editableOnly {
		editableSegments = mergeAdjacentEditableSegments(source, editableSegments)
		return doc, absorbAdjacentEditableProsePunctuation(source, editableSegments), nil
	}
	doc.Segments = mergeAdjacentSegments(source, doc.Segments, true)
	doc.Segments = absorbAdjacentProsePunctuation(source, doc.Segments)

	doc.Segments = append(extractFrontmatter(source, bodyStart), doc.Segments...)
	sort.SliceStable(doc.Segments, func(i, j int) bool { return doc.Segments[i].Start < doc.Segments[j].Start })
	return doc, nil, nil
}

func absorbAdjacentProsePunctuation(source []byte, segments []Segment) []Segment {
	for index := range segments {
		segment := &segments[index]
		if segment.Kind == "frontmatter" || segment.Kind == "navigation" || segment.Encoding != "" {
			continue
		}
		start, end := adjacentProseRange(source, segments, index)
		absorbedProse := start != segment.Start || end != segment.End
		if !absorbedProse {
			continue
		}
		raw := string(source[start:end])
		protected, placeholders := protect(raw)
		segment.Start, segment.End = start, end
		segment.Original, segment.Text = raw, protected
		segment.Placeholders, segment.SourceHash = placeholders, Hash(raw)
	}
	return segments
}

func absorbAdjacentEditableProsePunctuation(source []byte, segments []EditableSegment) []EditableSegment {
	for index := range segments {
		segment := &segments[index]
		start, end := adjacentEditableProseRange(source, segments, index)
		if start == segment.Start && end == segment.End {
			continue
		}
		segment.Start, segment.End = start, end
		segment.Original = string(source[start:end])
		segment.SourceHash = Hash(segment.Original)
	}
	return segments
}

func adjacentProseRange(source []byte, segments []Segment, index int) (int, int) {
	lower, upper := 0, len(source)
	if index > 0 {
		lower = segments[index-1].End
	}
	if index+1 < len(segments) {
		upper = segments[index+1].Start
	}
	return expandProseRange(source, segments[index].Start, segments[index].End, lower, upper)
}

func adjacentEditableProseRange(source []byte, segments []EditableSegment, index int) (int, int) {
	lower, upper := 0, len(source)
	if index > 0 {
		lower = segments[index-1].End
	}
	if index+1 < len(segments) {
		upper = segments[index+1].Start
	}
	return expandProseRange(source, segments[index].Start, segments[index].End, lower, upper)
}

func expandProseRange(source []byte, start, end, lower, upper int) (int, int) {
	originalStart, originalEnd := start, end
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	prefix := source[lineStart:start]
	blockPrefix := blockPrefixPattern.Find(prefix)
	leadingProse := false
	if marker := parenthesizedMarkerSuffixPattern.FindIndex(prefix); marker != nil {
		start = lineStart + marker[0]
		leadingProse = true
	}
	if marker := pricingMarkerSuffixPattern.FindIndex(prefix); marker != nil {
		start = lineStart + marker[0]
		leadingProse = true
	}
	if strings.HasPrefix(strings.TrimLeft(string(prefix), " \t"), "#") {
		if marker := numberedProseSuffixPattern.FindIndex(prefix); marker != nil {
			start = lineStart + marker[0]
			leadingProse = true
		}
	}
	if !leadingProse && len(blockPrefix) != len(prefix) {
		for start > lower && start > lineStart {
			r, size := utf8.DecodeLastRune(source[lower:start])
			allowed := absorbableLeadingProseRune(r)
			// Goldmark inconsistently assigns a closing parenthesis immediately
			// after inline code to the adjacent text node depending on the script
			// of the following prose. Normalize that boundary across languages.
			if !allowed && r == ')' && start-size > lineStart && source[start-size-1] == '`' {
				allowed = true
			}
			if !allowed {
				break
			}
			if r != ' ' && r != '\t' && r != '\u00a0' {
				leadingProse = true
			}
			start -= size
		}
	}
	if !leadingProse {
		start = originalStart
	}
	absorbedProse := false
	for end < upper {
		r, size := utf8.DecodeRune(source[end:upper])
		if !absorbableProseRune(r) {
			break
		}
		if r != ' ' && r != '\t' && r != '\u00a0' {
			absorbedProse = true
		}
		end += size
	}
	if !absorbedProse {
		end = originalEnd
	}
	return start, end
}

func absorbableLeadingProseRune(r rune) bool {
	if r == ' ' || r == '\t' || r == '\u00a0' {
		return true
	}
	if !unicode.IsPunct(r) {
		return false
	}
	switch r {
	case '\\', '`', '*', '_', '[', ']', '(', ')', '<', '>', '{', '}', '@', '|', '#':
		return false
	default:
		return true
	}
}

func absorbableProseRune(r rune) bool {
	if r == '\n' || r == '\r' {
		return false
	}
	if r == ' ' || r == '\t' || r == '\u00a0' {
		return true
	}
	if unicode.IsDigit(r) {
		return true
	}
	if !unicode.IsPunct(r) {
		return false
	}
	switch r {
	case '\\', '`', '*', '_', '[', ']', '(', ')', '<', '>', '{', '}', '@', '|', '#':
		return false
	default:
		return true
	}
}

func mergeAdjacentEditableSegments(source []byte, segments []EditableSegment) []EditableSegment {
	if len(segments) < 2 {
		return segments
	}
	result := make([]EditableSegment, 0, len(segments))
	for _, segment := range segments {
		base := segment.ID[:strings.LastIndex(segment.ID, "-")]
		if len(result) > 0 {
			previous := &result[len(result)-1]
			previousBase := previous.ID[:strings.LastIndex(previous.ID, "-")]
			var gap []byte
			if previous.End <= segment.Start {
				gap = source[previous.End:segment.Start]
			}
			if previousBase == base && len(bytes.Trim(gap, " \t")) == 0 {
				previous.End = segment.End
				previous.Original = joinSegmentText(previous.Original, gap, segment.Original)
				previous.SourceHash = Hash(previous.Original)
				continue
			}
		}
		if len(result) > 0 {
			count := 1
			for i := len(result) - 1; i >= 0; i-- {
				resultBase := result[i].ID[:strings.LastIndex(result[i].ID, "-")]
				if resultBase != base {
					break
				}
				count++
			}
			segment.ID = numberedID(base+"-", count, 2)
		}
		result = append(result, segment)
	}
	return result
}

func mergeAdjacentSegments(source []byte, segments []Segment, protectText bool) []Segment {
	if len(segments) < 2 {
		return segments
	}
	result := make([]Segment, 0, len(segments))
	for _, segment := range segments {
		base := segment.ID[:strings.LastIndex(segment.ID, "-")]
		if len(result) > 0 {
			previous := &result[len(result)-1]
			previousBase := previous.ID[:strings.LastIndex(previous.ID, "-")]
			var gap []byte
			if previous.End <= segment.Start {
				gap = source[previous.End:segment.Start]
			}
			if previousBase == base && len(bytes.Trim(gap, " \t")) == 0 {
				previous.End = segment.End
				previous.Original = joinSegmentText(previous.Original, gap, segment.Original)
				previous.Text = previous.Original
				previous.Placeholders = nil
				if protectText {
					previous.Text, previous.Placeholders = protect(previous.Original)
				}
				previous.SourceHash = Hash(previous.Original)
				continue
			}
		}
		if len(result) > 0 {
			count := 1
			for i := len(result) - 1; i >= 0; i-- {
				resultBase := result[i].ID[:strings.LastIndex(result[i].ID, "-")]
				if resultBase != base {
					break
				}
				count++
			}
			segment.ID = numberedID(base+"-", count, 2)
		}
		result = append(result, segment)
	}
	return result
}

func joinSegmentText(left string, gap []byte, right string) string {
	var result strings.Builder
	result.Grow(len(left) + len(gap) + len(right))
	result.WriteString(left)
	_, _ = result.Write(gap)
	result.WriteString(right)
	return result.String()
}

func excludedTextNode(node ast.Node) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.(type) {
		case *ast.CodeSpan, *ast.FencedCodeBlock, *ast.CodeBlock, *ast.RawHTML, *ast.HTMLBlock, *ast.AutoLink:
			return true
		}
	}
	return false
}

func isTranslationBlock(node ast.Node) bool {
	switch node.(type) {
	case *ast.Heading, *ast.Paragraph, *ast.TextBlock:
		return true
	default:
		return false
	}
}

func blockKind(node ast.Node) string {
	switch node.(type) {
	case *ast.Heading:
		return "heading"
	case *ast.TextBlock:
		return "cell"
	default:
		return "text"
	}
}

func translatableText(value string) bool {
	for _, r := range value {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func componentSyntaxLine(source []byte, position int) bool {
	lineStart := bytes.LastIndexByte(source[:position], '\n') + 1
	lineEnd := bytes.IndexByte(source[position:], '\n')
	if lineEnd < 0 {
		lineEnd = len(source)
	} else {
		lineEnd += position
	}
	line := bytes.TrimSpace(source[lineStart:lineEnd])
	return bytes.HasPrefix(line, []byte(":::")) || bytes.HasPrefix(line, []byte("@"))
}

func frontmatterEnd(source []byte) int {
	if !bytes.HasPrefix(source, []byte("---\n")) && !bytes.HasPrefix(source, []byte("---\r\n")) {
		return 0
	}
	position := bytes.IndexByte(source, '\n') + 1
	for position < len(source) {
		next := bytes.IndexByte(source[position:], '\n')
		end := len(source)
		if next >= 0 {
			end = position + next + 1
		}
		line := strings.TrimSpace(string(source[position:end]))
		if line == "---" {
			return end
		}
		position = end
	}
	return 0
}

func extractFrontmatter(source []byte, end int) []Segment {
	if end == 0 {
		return nil
	}
	allowed := map[string]bool{"title": true, "description": true, "tagline": true, "content": true, "text": true}
	var result []Segment
	counts := map[string]int{}
	blockKey := ""
	blockIndent := -1
	position := 0
	for position < end {
		next := bytes.IndexByte(source[position:end], '\n')
		lineEnd := end
		if next >= 0 {
			lineEnd = position + next
		}
		line := source[position:lineEnd]
		indent := len(line) - len(bytes.TrimLeft(line, " \t"))
		if blockKey != "" {
			if len(bytes.TrimSpace(line)) == 0 {
				position = lineEnd + 1
				continue
			}
			if indent > blockIndent {
				valueStart := indent
				raw := string(line[valueStart:])
				if translatableText(raw) {
					counts[blockKey]++
					protected, placeholders := protect(raw)
					result = append(result, Segment{ID: fmt.Sprintf("fm-%s-%02d", blockKey, counts[blockKey]), Kind: "frontmatter", Original: raw, Text: protected, Start: position + valueStart, End: lineEnd, SourceHash: Hash(raw), Placeholders: placeholders})
				}
				position = lineEnd + 1
				continue
			}
			blockKey = ""
			blockIndent = -1
		}
		colon := bytes.IndexByte(line, ':')
		if colon > 0 {
			key := strings.TrimSpace(string(line[:colon]))
			key = strings.TrimSpace(strings.TrimPrefix(key, "-"))
			if allowed[key] {
				valueStart := colon + 1
				for valueStart < len(line) && (line[valueStart] == ' ' || line[valueStart] == '\t') {
					valueStart++
				}
				valueEnd := len(line)
				if valueStart < valueEnd && (line[valueStart] == '\'' || line[valueStart] == '"') && line[valueEnd-1] == line[valueStart] {
					valueStart++
					valueEnd--
				}
				raw := string(line[valueStart:valueEnd])
				if strings.Trim(raw, "0123456789+- ") == "|" || strings.Trim(raw, "0123456789+- ") == ">" {
					blockKey = key
					blockIndent = indent
					position = lineEnd + 1
					continue
				}
				if translatableText(raw) {
					counts[key]++
					protected, placeholders := protect(raw)
					result = append(result, Segment{ID: fmt.Sprintf("fm-%s-%02d", key, counts[key]), Kind: "frontmatter", Original: raw, Text: protected, Start: position + valueStart, End: position + valueEnd, SourceHash: Hash(raw), Placeholders: placeholders})
				}
			}
		}
		position = lineEnd + 1
	}
	return result
}

func protect(value string) (string, map[string]string) {
	// Placeholder syntax always starts with one of these three bytes. Most
	// documentation prose contains none of them, so avoid starting the regexp
	// engine for the overwhelmingly common case.
	if !strings.ContainsAny(value, "&\\{$%0123456789") {
		return value, nil
	}
	values := map[string]string{}
	index := 0
	protected := protectedPattern.ReplaceAllStringFunc(value, func(match string) string {
		key := fmt.Sprintf("⟪MPRESS_%d⟫", index)
		index++
		values[key] = match
		return key
	})
	return protected, values
}

func restore(segment Segment, translated string, preserveLineStructure bool) (string, error) {
	if err := validateTranslationControls(translated); err != nil {
		return "", fmt.Errorf("segment %s: %w", segment.ID, err)
	}
	for placeholder, original := range segment.Placeholders {
		if strings.Count(translated, placeholder) != 1 {
			return "", fmt.Errorf("segment %s did not preserve placeholder %s", segment.ID, placeholder)
		}
		translated = strings.ReplaceAll(translated, placeholder, original)
	}
	if !preserveLineStructure {
		translated = preserveOuterWhitespace(segment.Original, translated)
	}
	if preserveLineStructure && strings.Count(translated, "\n") != strings.Count(segment.Original, "\n") {
		return "", fmt.Errorf("segment %s changed its line structure", segment.ID)
	}
	if strings.TrimSpace(translated) == "" {
		return "", fmt.Errorf("segment %s returned an empty translation", segment.ID)
	}
	if preserveLineStructure && syntaxSignature(translated) != syntaxSignature(segment.Original) {
		return "", fmt.Errorf("segment %s changed protected Markdown punctuation", segment.ID)
	}
	return translated, nil
}

func validateTranslationControls(value string) error {
	for _, r := range value {
		if r < 0x20 && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("translation contains forbidden control character U+%04X", r)
		}
	}
	return nil
}

func preserveOuterWhitespace(original, translated string) string {
	leading := original[:len(original)-len(strings.TrimLeft(original, " \t\r\n"))]
	trailing := original[len(strings.TrimRight(original, " \t\r\n")):]
	return leading + strings.Trim(translated, " \t\r\n") + trailing
}

func syntaxSignature(value string) string {
	var signature strings.Builder
	for _, r := range value {
		switch r {
		case '\\', '`', '*', '_', '[', ']', '<', '>', '{', '}':
			signature.WriteRune(r)
		}
	}
	return signature.String()
}

// Apply patches translated text into the original bytes from the end towards
// the beginning. Untouched bytes remain identical, except translated D2 blocks
// which are formatted by D2's editing API to preserve their graph structure.
func Apply(document *Document, translated map[string]string) ([]byte, error) {
	return applyDocumentValues(document, translated, true)
}

func applyDocumentValues(document *Document, translated map[string]string, restoreText bool) ([]byte, error) {
	result := append([]byte(nil), document.Source...)
	segments := append([]Segment(nil), document.Segments...)
	sort.Slice(segments, func(i, j int) bool { return segments[i].Start > segments[j].Start })
	diagrams := map[int]bool{}
	for _, segment := range segments {
		value, ok := translated[segment.ID]
		if !ok {
			continue
		}
		var err error
		restored := value
		if segment.D2Key != "" {
			if diagrams[segment.Start] {
				continue
			}
			diagrams[segment.Start] = true
			restored, err = applyD2Labels(document, segment, translated, restoreText)
		} else {
			restored, err = encodeTranslationSegment(document.Format, segment, value, restoreText)
		}
		if err != nil {
			return nil, err
		}
		result = append(result[:segment.Start], append([]byte(restored), result[segment.End:]...)...)
	}
	return result, nil
}

func encodeTranslationSegment(format string, segment Segment, value string, restoreText bool) (string, error) {
	if !restoreText {
		return encodeSegment(segment, value)
	}
	restored, err := restore(segment, value, preservesMarkdownSyntax(format))
	if err != nil {
		return "", err
	}
	if format == "mpd" && segment.Kind == "text" {
		restored = escapeMPDDirectiveText(restored)
	}
	return encodeSegment(segment, restored)
}

func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	const digits = "0123456789abcdef"
	var encoded [sha256.Size * 2]byte
	for i, value := range sum {
		encoded[i*2] = digits[value>>4]
		encoded[i*2+1] = digits[value&0x0f]
	}
	return string(encoded[:])
}

func blockSegmentID(block int, kind string, item int) string {
	var result strings.Builder
	result.Grow(len(kind) + 9)
	result.WriteByte('b')
	writePaddedInt(&result, block, 4)
	result.WriteByte('-')
	result.WriteString(kind)
	result.WriteByte('-')
	writePaddedInt(&result, item, 2)
	return result.String()
}

func numberedID(prefix string, value, width int) string {
	var result strings.Builder
	result.Grow(len(prefix) + width)
	result.WriteString(prefix)
	writePaddedInt(&result, value, width)
	return result.String()
}

func writePaddedInt(result *strings.Builder, value, width int) {
	var storage [20]byte
	digits := strconv.AppendInt(storage[:0], int64(value), 10)
	for padding := len(digits); padding < width; padding++ {
		result.WriteByte('0')
	}
	_, _ = result.Write(digits)
}

func preservesMarkdownSyntax(format string) bool { return format != "mpd" && format != "navigation" }

func navigationLabelEnd(line []byte, start int) int {
	if start < len(line) && (line[start] == '\'' || line[start] == '"') {
		quote := line[start]
		for i := start + 1; i < len(line); i++ {
			if quote == '"' && line[i] == '\\' {
				i++
				continue
			}
			if line[i] != quote {
				continue
			}
			if quote == '\'' && i+1 < len(line) && line[i+1] == quote {
				i++
				continue
			}
			return i + 1
		}
	}
	end := len(line)
	if next := bytes.IndexByte(line[start:], '\n'); next >= 0 {
		end = start + next
	}
	for i := start + 1; i < end; i++ {
		if line[i] == '#' && (line[i-1] == ' ' || line[i-1] == '\t') {
			end = i
			break
		}
	}
	for end > start && (line[end-1] == ' ' || line[end-1] == '\t') {
		end--
	}
	return end
}
