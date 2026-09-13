package components

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/leaanthony/mpress/internal/highlight"
	"github.com/leaanthony/mpress/internal/icons"
)

// Component is the interface all custom components must implement.
type Component interface {
	Parse(content string) error
	Render() (string, error)
}

// Registry holds registered component factories.
var Registry = map[string]func(metadata map[string]string) Component{
	"tabs":               func(m map[string]string) Component { return &Tabs{Meta: m} },
	"terminal":           func(m map[string]string) Component { return &Terminal{Meta: m} },
	"note":               func(m map[string]string) Component { return &Note{Meta: m} },
	"details":            func(m map[string]string) Component { return &Details{Meta: m} },
	"badge":              func(m map[string]string) Component { return &Badge{Meta: m} },
	"api":                func(m map[string]string) Component { return &API{Meta: m} },
	"steps":              func(m map[string]string) Component { return &Steps{Meta: m} },
	"cards":              func(m map[string]string) Component { return &Cards{Meta: m} },
	"diff":               func(m map[string]string) Component { return &Diff{Meta: m} },
	"filetree":           func(m map[string]string) Component { return &FileTree{Meta: m} },
	"linkcard":           func(m map[string]string) Component { return &LinkCard{Meta: m} },
	"calendar":           func(m map[string]string) Component { return &Calendar{Meta: m} },
	"carousel":           func(m map[string]string) Component { return &Carousel{Meta: m} },
	"changelog":          func(m map[string]string) Component { return &Changelog{Meta: m} },
	"matrix":             func(m map[string]string) Component { return &Matrix{Meta: m} },
	"tutorial":           func(m map[string]string) Component { return &Tutorial{Meta: m} },
	"status":             func(m map[string]string) Component { return &Status{Meta: m} },
	"pricing":            func(m map[string]string) Component { return &Pricing{Meta: m} },
	"release":            func(m map[string]string) Component { return &Release{Meta: m} },
	"testimonials":       func(m map[string]string) Component { return &Testimonials{Meta: m} },
	"api-playground":     func(m map[string]string) Component { return &APIPlayground{Meta: m} },
	"qr":                 func(m map[string]string) Component { return &QR{Meta: m} },
	"audience":           func(m map[string]string) Component { return &Audience{Meta: m} },
	"explained":          func(m map[string]string) Component { return &Explained{Meta: m} },
	"input":              func(m map[string]string) Component { return &Input{Meta: m} },
	"computed":           func(m map[string]string) Component { return &Computed{Meta: m} },
	"button":             func(m map[string]string) Component { return &Button{Meta: m} },
	"image":              func(m map[string]string) Component { return &ThemeImage{Meta: m} },
	"table":              func(m map[string]string) Component { return &Table{Meta: m} },
	"video":              func(m map[string]string) Component { return &Video{Meta: m} },
	"accessibility-demo": func(m map[string]string) Component { return &AccessibilityDemo{Meta: m} },
}

// Block represents a parsed custom component block from Markdown.
type Block struct {
	Type     string
	Metadata map[string]string
	Content  string
}

// componentBlockRegex matches :::type{key="value" ...}\ncontent\n:::
// Supports hyphenated names like api-playground.
//
// The content group matches lines that do NOT start with ::: so that nested
// components (e.g. :::note inside :::tabs) stop the content match rather than
// being swallowed.  Go's RE2 engine has no negative lookahead, so the exclusion
// is expressed as a union of character-class alternatives:
//
//	[ \t]*\n             — blank or whitespace-only line
//	[^:\n][^\n]*\n        — line whose first char is not ':'
//	:[^:\n][^\n]*\n       — line starting with exactly one ':'
//	::[^:\n][^\n]*\n      — line starting with exactly two ':'
//
// Any line starting with ':::' falls into none of these and therefore stops
// the content match.  Combined with running Process in a loop until stable,
// this enables correct inside-out processing of nested components.
//
// The closing ::: must be alone on its line (only trailing whitespace allowed).
var componentBlockRegex = regexp.MustCompile(`(?m)^([ \t]{0,3}):::(\w[\w-]*)(\{[^}]*\})?\s*\n((?:[ \t]*\n|[ \t]{4,}[^\n]*\n|[ \t]{0,3}(?:[^:\s\n]|:[^:\n]|::[^:\n])[^\n]*\n)*)^[ \t]{0,3}:::\s*$`)

// metadataRegex matches key=value pairs with quoted or unquoted values.
var metadataRegex = regexp.MustCompile(`([\w-]+)=(?:"([^"]*)"|([^\s}]+))`)

// protectedCodeBlock is a fenced block held aside while components are
// expanded. prefix records the indentation and blockquote markers the fence
// opened with, so the block can follow its placeholder when a component
// reindents the surrounding body.
type protectedCodeBlock struct {
	text   string
	prefix string
}

// protectFencedCodeBlocks replaces legacy Markdown fences with opaque tokens.
// This scanner deliberately preserves the historical regexp semantics: the
// closing marker may use either fence character, may have trailing text, and
// must not be the line immediately after the opener. Keeping those details
// makes this a byte-for-byte optimization rather than a syntax change.
func protectFencedCodeBlocks(content, tokenPrefix string) (string, []protectedCodeBlock) {
	var blocks []protectedCodeBlock
	var out strings.Builder
	last := 0
	lineStart := 0
	changed := false

	for lineStart < len(content) {
		lineEnd := strings.IndexByte(content[lineStart:], '\n')
		if lineEnd < 0 {
			break
		}
		lineEnd += lineStart
		marker := fenceMarkerAtLine(content, lineStart, lineEnd)
		if marker < 0 {
			lineStart = lineEnd + 1
			continue
		}

		// The old regexp consumes the opener's newline before matching its
		// non-greedy body. Consequently, the next line cannot be the closer.
		candidate := lineEnd + 1
		firstEnd := strings.IndexByte(content[candidate:], '\n')
		if firstEnd < 0 {
			lineStart = lineEnd + 1
			continue
		}
		candidate += firstEnd + 1

		closeEnd := -1
		for candidate < len(content) {
			candidateEnd := strings.IndexByte(content[candidate:], '\n')
			if candidateEnd < 0 {
				candidateEnd = len(content)
			} else {
				candidateEnd += candidate
			}
			closeMarker := fenceMarkerAtLine(content, candidate, candidateEnd)
			if closeMarker >= 0 {
				closeEnd = closeMarker + 3
				break
			}
			if candidateEnd == len(content) {
				break
			}
			candidate = candidateEnd + 1
		}
		if closeEnd < 0 {
			lineStart = lineEnd + 1
			continue
		}

		if !changed {
			out.Grow(len(content))
			changed = true
		}
		out.WriteString(content[last:lineStart])
		prefix := content[lineStart:marker]
		match := content[lineStart:closeEnd]
		blocks = append(blocks, protectedCodeBlock{text: match, prefix: prefix})
		if tokenPrefix != "" {
			out.WriteString(prefix)
			out.WriteByte(0)
			out.WriteString(tokenPrefix)
			var digits [20]byte
			_, _ = out.Write(strconv.AppendInt(digits[:0], int64(len(blocks)-1), 10))
			out.WriteByte(0)
		}
		last = closeEnd

		// A match ends immediately after the closing marker. Resume at the next
		// physical line while retaining any trailing bytes through last.
		nextLine := strings.IndexByte(content[closeEnd:], '\n')
		if nextLine < 0 {
			lineStart = len(content)
		} else {
			lineStart = closeEnd + nextLine + 1
		}
	}
	if !changed {
		return content, nil
	}
	out.WriteString(content[last:])
	return out.String(), blocks
}

func fenceMarkerAtLine(content string, start, end int) int {
	cursor := start
	for cursor < end {
		switch content[cursor] {
		case ' ', '\t', '>':
			cursor++
		default:
			if cursor+3 <= end && (content[cursor:cursor+3] == "```" || content[cursor:cursor+3] == "~~~") {
				return cursor
			}
			return -1
		}
	}
	return -1
}

// reindentCodeBlock rewrites a held block so every line carries prefix in place
// of the container prefix it was captured with.
func reindentCodeBlock(block protectedCodeBlock, prefix string) string {
	if prefix == block.prefix {
		return block.text
	}
	lines := strings.Split(block.text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if block.prefix != "" && strings.HasPrefix(line, block.prefix) {
			lines[i] = prefix + line[len(block.prefix):]
			continue
		}
		lines[i] = prefix + strings.TrimLeft(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// restoreCodeBlocks restores every token in one traversal. The former caller
// invoked restoreCodeBlock once per fence, copying the complete document on
// every pass even though the tokens are independent and occur in source order.
func restoreCodeBlocks(content, tokenPrefix string, blocks []protectedCodeBlock) string {
	if len(blocks) == 0 {
		return content
	}
	marker := "\x00" + tokenPrefix
	if !strings.Contains(content, marker) {
		return content
	}

	var out strings.Builder
	out.Grow(len(content))
	cursor := 0
	search := 0
	restored := false
	seen := make([]bool, len(blocks))
	for search < len(content) {
		relative := strings.Index(content[search:], marker)
		if relative < 0 {
			break
		}
		at := search + relative
		digits := at + len(marker)
		end := strings.IndexByte(content[digits:], 0)
		if end < 0 {
			break
		}
		end += digits
		index, ok := decimalIndex(content[digits:end], len(blocks))
		if !ok || seen[index] {
			search = end + 1
			continue
		}
		seen[index] = true
		block := blocks[index]
		lineStart := strings.LastIndexByte(content[:at], '\n') + 1
		current := content[lineStart:at]
		if lineStart >= cursor && strings.TrimLeft(current, " \t>") == "" {
			out.WriteString(content[cursor:lineStart])
			out.WriteString(reindentCodeBlock(block, current))
		} else {
			out.WriteString(content[cursor:at])
			out.WriteString(block.text)
		}
		cursor = end + 1
		search = cursor
		restored = true
	}
	if !restored {
		return content
	}
	out.WriteString(content[cursor:])
	return out.String()
}

func decimalIndex(value string, limit int) (int, bool) {
	if value == "" {
		return 0, false
	}
	index := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
		index = index*10 + int(value[i]-'0')
		if index >= limit {
			return 0, false
		}
	}
	return index, true
}

// tabSectionRegex matches [Label] sections in tab content.
var tabSectionRegex = regexp.MustCompile(`(?m)^\[([^\]]+)\]\s*\n`)

// stepHeaderRegex matches ### headings in step content.
var stepHeaderRegex = regexp.MustCompile(`(?m)^###\s+(.+)$`)
var orderedStepHeaderRegex = regexp.MustCompile(`(?m)^([ \t]*)\d+\.[ \t]+(?:(?:#+[ \t]+(.+))|\*\*([^*]+)\*\*)[ \t]*$`)
var orderedStepPrefixRegex = regexp.MustCompile(`^[ \t]*\d+\.[ \t]+`)

// cardSeparatorRegex splits cards by --- separators.
var cardSeparatorRegex = regexp.MustCompile(`(?m)^---$`)

// cardLinkRegex matches [Title](url) format in card titles.
var cardLinkRegex = regexp.MustCompile(`^\[([^\]]+)\]\(([^)]+)\)$`)

var fileTreeEntryRegex = regexp.MustCompile(`^(\S+)\s{2,}(.+)$`)

// Process finds all custom component blocks in Markdown content,
// renders them, and replaces the blocks with HTML.
// Fenced code blocks are protected from component processing.
func Process(content string) (string, []string) {
	return ProcessWithLanguage(content, ActiveLanguage)
}

// ProcessWithLanguage finds all custom component blocks in Markdown content and
// renders labels for the provided language without reading ActiveLanguage.
func ProcessWithLanguage(content string, language string) (string, []string) {
	var warnings []string

	// Step 1: Extract fenced code blocks and replace with placeholders. The
	// placeholder keeps the container prefix of the fence it stands for, so a
	// fence inside a list item or blockquote stays in its container while
	// components are expanded around it.
	protected, codeBlocks := protectFencedCodeBlocks(content, "CODEBLOCK_")
	protected = injectPreviewTabSources(protected, codeBlocks)
	protected = processNativeSyntax(protected)

	// Step 2: Process components on the protected content.
	// Loop until stable so that inner (nested) components are replaced first;
	// subsequent passes then see the rendered HTML instead of raw :::type and
	// correctly match the outer block.
	replaceOnce := func(src string) string {
		if strings.Contains(src, ":::steps") {
			src = replaceStepsBlocks(src, language, &warnings)
		}
		if strings.Contains(src, ":::tabs") {
			src = replaceTabsBlocks(src, language, codeBlocks, &warnings)
		}
		if !strings.Contains(src, ":::") {
			return src
		}
		return componentBlockRegex.ReplaceAllStringFunc(src, func(match string) string {
			block := parseBlock(match)
			if block == nil {
				warnings = append(warnings, fmt.Sprintf("failed to parse component block: %s", truncate(match, 60)))
				return match
			}
			if language != "" {
				block.Metadata["__mpress_lang"] = language
			}

			// Keep fenced-code placeholders protected through every component
			// pass. Otherwise a directive shown inside a fence can become live on
			// the next inside-out pass. Explained is the one component that parses
			// the fence itself, so restore its input only after its outer block has
			// been identified.
			blockContent := block.Content
			if rendersMarkdownContent(block.Type) {
				blockContent = restoreCodeBlocks(blockContent, "CODEBLOCK_", codeBlocks)
			}

			factory, ok := Registry[block.Type]
			if !ok {
				warnings = append(warnings, fmt.Sprintf("unknown component type: %s", block.Type))
				return match
			}

			comp := factory(block.Metadata)
			if err := comp.Parse(blockContent); err != nil {
				warnings = append(warnings, fmt.Sprintf("error parsing %s component: %v", block.Type, err))
				return match
			}

			rendered, err := comp.Render()
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("error rendering %s component: %v", block.Type, err))
				return match
			}
			// A blank line after block HTML returns Goldmark to Markdown parsing.
			// Without it, a following heading can be treated as literal text inside
			// the raw HTML block.
			rendered = strings.TrimRight(rendered, "\r\n") + "\n\n"

			if indent := componentIndent(match); indent != "" {
				return indentRenderedComponent(rendered, indent)
			}
			return rendered
		})
	}

	result := protected
	for {
		next := replaceOnce(result)
		if next == result {
			break
		}
		result = next
	}

	// Step 3: Re-insert protected code blocks
	result = restoreCodeBlocks(result, "CODEBLOCK_", codeBlocks)

	return result, warnings
}

func replaceStepsBlocks(src, language string, warnings *[]string) string {
	var out strings.Builder
	lines := strings.SplitAfter(src, "\n")
	for i := 0; i < len(lines); {
		line := lines[i]
		open := stepsOpenLine(line)
		if open == nil {
			out.WriteString(line)
			i++
			continue
		}

		start := i
		depth := 0
		var body strings.Builder
		i++
		closed := false
		for i < len(lines) {
			current := lines[i]
			trimmed := strings.TrimSpace(current)
			if trimmed == ":::" || strings.HasPrefix(trimmed, ":::endtabs") {
				if depth == 0 {
					closed = true
					i++
					break
				}
				depth--
				body.WriteString(current)
				i++
				continue
			}
			if strings.HasPrefix(trimmed, ":::") && len(trimmed) > 3 {
				depth++
			}
			body.WriteString(current)
			i++
		}
		if !closed {
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}

		if language != "" {
			open["__mpress_lang"] = language
		}
		steps := &Steps{Meta: open}
		if err := steps.Parse(body.String()); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("error parsing steps component: %v", err))
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}
		rendered, err := steps.Render()
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("error rendering steps component: %v", err))
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}
		out.WriteString(rendered)
	}
	return out.String()
}

func stepsOpenLine(line string) map[string]string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, ":::steps") {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, ":::steps"))
	meta := make(map[string]string)
	if rest == "" {
		return meta
	}
	if strings.HasPrefix(rest, "{") && strings.HasSuffix(rest, "}") {
		for _, p := range metadataRegex.FindAllStringSubmatch(rest, -1) {
			value := p[2]
			if value == "" {
				value = p[3]
			}
			meta[p[1]] = value
		}
		return meta
	}
	return nil
}

func rendersMarkdownContent(componentType string) bool {
	switch componentType {
	case "explained", "section", "columns", "column", "actions", "docs-preview", "preview-tabs", "file-tabs", "callout":
		return true
	default:
		return false
	}
}

func replaceTabsBlocks(src, language string, codeBlocks []protectedCodeBlock, warnings *[]string) string {
	var out strings.Builder
	lines := strings.SplitAfter(src, "\n")
	for i := 0; i < len(lines); {
		line := lines[i]
		open := tabsOpenLine(line)
		if open == nil {
			out.WriteString(line)
			i++
			continue
		}

		start := i
		depth := 0
		var body strings.Builder
		i++
		closed := false
		for i < len(lines) {
			current := lines[i]
			trimmed := strings.TrimSpace(current)
			if trimmed == ":::" || strings.HasPrefix(trimmed, ":::endtabs") {
				if depth == 0 {
					closed = true
					i++
					break
				}
				depth--
				body.WriteString(current)
				i++
				continue
			}
			if strings.HasPrefix(trimmed, ":::") && len(trimmed) > 3 {
				depth++
			}
			body.WriteString(current)
			i++
		}
		if !closed {
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}

		// Preserve fenced-code placeholders until all nested component passes
		// have completed. They are restored once, at the end of Process.
		content := body.String()

		if language != "" {
			open["__mpress_lang"] = language
		}
		tabs := &Tabs{Meta: open}
		if err := tabs.Parse(content); err != nil {
			*warnings = append(*warnings, fmt.Sprintf("error parsing tabs component: %v", err))
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}
		rendered, err := tabs.Render()
		if err != nil {
			*warnings = append(*warnings, fmt.Sprintf("error rendering tabs component: %v", err))
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}
		out.WriteString(rendered)
	}
	return out.String()
}

func tabsOpenLine(line string) map[string]string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, ":::tabs") {
		return nil
	}
	rest := strings.TrimSpace(strings.TrimPrefix(trimmed, ":::tabs"))
	meta := make(map[string]string)
	if rest == "" {
		return meta
	}
	if strings.HasPrefix(rest, "{") && strings.HasSuffix(rest, "}") {
		for _, p := range metadataRegex.FindAllStringSubmatch(rest, -1) {
			value := p[2]
			if value == "" {
				value = p[3]
			}
			meta[p[1]] = value
		}
		return meta
	}
	return nil
}

// parseBlock extracts component type, metadata, and content from a matched block.
func parseBlock(match string) *Block {
	m := componentBlockRegex.FindStringSubmatch(match)
	if m == nil {
		return nil
	}

	blockContent := stripComponentIndent(m[4], m[1])
	block := &Block{
		Type:     m[2],
		Metadata: make(map[string]string),
		Content:  stripUniformComponentContentIndent(blockContent),
	}

	if m[3] != "" {
		pairs := metadataRegex.FindAllStringSubmatch(m[3], -1)
		for _, p := range pairs {
			value := p[2]
			if value == "" {
				value = p[3]
			}
			block.Metadata[p[1]] = value
		}
	}

	return block
}

func stripUniformComponentContentIndent(content string) string {
	lines := strings.SplitAfter(content, "\n")
	minimum := -1
	for _, line := range lines {
		body, _ := splitLineEnding(line)
		if strings.TrimSpace(body) == "" {
			continue
		}
		indent := len(body) - len(strings.TrimLeft(body, " \t"))
		if minimum == -1 || indent < minimum {
			minimum = indent
		}
	}
	if minimum < 4 {
		return content
	}
	for i, line := range lines {
		body, newline := splitLineEnding(line)
		if strings.TrimSpace(body) == "" {
			lines[i] = newline
			continue
		}
		remove := min(minimum, len(body)-len(strings.TrimLeft(body, " \t")))
		lines[i] = body[remove:] + newline
	}
	return strings.Join(lines, "")
}

func componentIndent(match string) string {
	m := componentBlockRegex.FindStringSubmatch(match)
	if m == nil {
		return ""
	}
	return m[1]
}

func stripComponentIndent(content, indent string) string {
	if indent == "" {
		return content
	}
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		body, newline := splitLineEnding(line)
		if strings.HasPrefix(body, indent) {
			lines[i] = strings.TrimPrefix(body, indent) + newline
		}
	}
	return strings.Join(lines, "")
}

func indentRenderedComponent(rendered, indent string) string {
	if rendered == "" || indent == "" {
		return rendered
	}
	lines := strings.SplitAfter(rendered, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = indent + line
	}
	return strings.Join(lines, "")
}

func splitLineEnding(line string) (string, string) {
	if strings.HasSuffix(line, "\r\n") {
		return strings.TrimSuffix(line, "\r\n"), "\r\n"
	}
	if strings.HasSuffix(line, "\n") {
		return strings.TrimSuffix(line, "\n"), "\n"
	}
	return line, ""
}

// --- Tabs Component ---

type Tabs struct {
	Meta     map[string]string
	Sections []TabSection
}

type TabSection struct {
	Label   string
	Content string
}

func (t *Tabs) Parse(content string) error {
	// Protect fenced code blocks from [Label] matching.
	// TOML section headers like [params] inside code blocks must not be
	// interpreted as tab labels.
	protected, codeBlocks := protectFencedCodeBlocks(content, "TABCODE_")
	var componentBlocks []string
	protected = protectNestedTabComponents(protected, &componentBlocks)

	// Parse [Label]\ncontent sections on the protected content
	locs := tabSectionRegex.FindAllStringSubmatchIndex(protected, -1)

	if len(locs) == 0 {
		return fmt.Errorf("no tab sections found")
	}

	for i, loc := range locs {
		label := protected[loc[2]:loc[3]]
		contentStart := loc[1]
		var contentEnd int
		if i+1 < len(locs) {
			contentEnd = locs[i+1][0]
		} else {
			contentEnd = len(protected)
		}
		sectionContent := normalizeTabSectionContent(protected[contentStart:contentEnd])
		// Restore protected blocks only after normalizing the section. Their
		// internal indentation is meaningful and must not affect the tab's
		// structural indentation.
		sectionContent = restoreCodeBlocks(sectionContent, "TABCODE_", codeBlocks)
		for j, cb := range componentBlocks {
			placeholder := fmt.Sprintf("\x00TABCOMP_%d\x00", j)
			sectionContent = strings.Replace(sectionContent, placeholder, cb, 1)
		}
		sectionContent = strings.TrimSpace(sectionContent)
		t.Sections = append(t.Sections, TabSection{
			Label:   label,
			Content: sectionContent,
		})
	}

	return nil
}

func protectNestedTabComponents(content string, blocks *[]string) string {
	var out strings.Builder
	lines := strings.SplitAfter(content, "\n")
	for i := 0; i < len(lines); {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, ":::") || trimmed == ":::" || strings.HasPrefix(trimmed, ":::endtabs") {
			out.WriteString(line)
			i++
			continue
		}
		start := i
		depth := 0
		i++
		closed := false
		for i < len(lines) {
			current := lines[i]
			currentTrimmed := strings.TrimSpace(current)
			if currentTrimmed == ":::" || strings.HasPrefix(currentTrimmed, ":::endtabs") {
				if depth == 0 {
					i++
					closed = true
					break
				}
				depth--
				i++
				continue
			}
			if strings.HasPrefix(currentTrimmed, ":::") && len(currentTrimmed) > 3 {
				depth++
			}
			i++
		}
		if !closed {
			for ; start < i; start++ {
				out.WriteString(lines[start])
			}
			continue
		}
		block := strings.Join(lines[start:i], "")
		body, _ := splitLineEnding(line)
		indent := body[:len(body)-len(strings.TrimLeft(body, " \t"))]
		placeholder := fmt.Sprintf("\x00TABCOMP_%d\x00", len(*blocks))
		*blocks = append(*blocks, stripComponentIndent(block, indent))
		out.WriteString(indent)
		out.WriteString(placeholder)
		if !strings.HasSuffix(block, "\n") {
			out.WriteString("\n")
		}
	}
	return out.String()
}

func normalizeTabSectionContent(content string) string {
	content = strings.Trim(content, "\r\n")
	lines := strings.Split(content, "\n")
	minimum := -1
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minimum == -1 || indent < minimum {
			minimum = indent
		}
	}
	if minimum <= 0 {
		return strings.TrimSpace(content)
	}
	for i, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
			continue
		}
		remove := min(minimum, len(line)-len(strings.TrimLeft(line, " \t")))
		lines[i] = line[remove:]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (t *Tabs) Render() (string, error) {
	return t.renderWithPanelContent(func(content string) string { return content }), nil
}

func (t *Tabs) renderWithPanelContent(render func(string) string) string {
	id := strings.TrimSpace(t.Meta["id"])
	if id == "" {
		var identity strings.Builder
		for _, section := range t.Sections {
			identity.WriteString(section.Label)
			identity.WriteByte('\n')
			identity.WriteString(section.Content)
			identity.WriteByte('\n')
		}
		sum := sha256.Sum256([]byte(identity.String()))
		id = "mpress-tabs-" + hex.EncodeToString(sum[:4])
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<ot-tabs class="mpress-tabs" id="%s"`, html.EscapeString(id)))
	for key, value := range t.Meta {
		if key == "id" || strings.HasPrefix(key, "__") {
			continue
		}
		b.WriteString(fmt.Sprintf(` data-%s="%s"`, html.EscapeString(key), html.EscapeString(value)))
	}
	b.WriteString(`>`)
	b.WriteString("\n")

	// Tab buttons in a tablist
	b.WriteString(`<div role="tablist">`)
	b.WriteString("\n")
	for i, s := range t.Sections {
		selected := "false"
		tabIndex := ` tabindex="-1"`
		if i == 0 {
			selected = "true"
			tabIndex = ""
		}
		classAttr := tabLabelClassAttr(s.Label)
		tabID := fmt.Sprintf("%s-tab-%d", id, i+1)
		panelID := fmt.Sprintf("%s-panel-%d", id, i+1)
		b.WriteString(fmt.Sprintf(`<button id="%s" role="tab" type="button"%s aria-selected="%s" aria-controls="%s"%s>%s</button>`, html.EscapeString(tabID), classAttr, selected, html.EscapeString(panelID), tabIndex, tabLabelContent(s.Label)))
		b.WriteString("\n")
	}
	b.WriteString("</div>\n")

	// Tab panels. Most tabs leave their source for the page renderer. Specialized
	// tab groups can supply a renderer when their panels must be self-contained.
	for i, s := range t.Sections {
		hidden := ""
		if i != 0 {
			hidden = " hidden"
		}
		tabID := fmt.Sprintf("%s-tab-%d", id, i+1)
		panelID := fmt.Sprintf("%s-panel-%d", id, i+1)
		b.WriteString(fmt.Sprintf("<div id=\"%s\" role=\"tabpanel\" aria-labelledby=\"%s\"%s>\n\n", html.EscapeString(panelID), html.EscapeString(tabID), hidden))
		b.WriteString(render(s.Content))
		b.WriteString("\n\n</div>\n")
	}

	b.WriteString("</ot-tabs>\n")
	return b.String()
}

func tabLabelClassAttr(label string) string {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "windows":
		return ` class="mpress-tab-os mpress-tab-os-windows"`
	case "macos":
		return ` class="mpress-tab-os mpress-tab-os-macos"`
	case "linux":
		return ` class="mpress-tab-os mpress-tab-os-linux"`
	default:
		return ""
	}
}

func tabLabelContent(label string) string {
	escaped := html.EscapeString(label)
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "windows":
		return starlightIcon("windows", 16) + " " + escaped
	case "macos":
		return starlightIcon("apple", 16) + " " + escaped
	case "linux":
		return starlightIcon("linux", 16) + " " + escaped
	default:
		return escaped
	}
}

// --- Terminal Component ---

type Terminal struct {
	Meta  map[string]string
	Lines []string
	Title string
}

func (t *Terminal) Parse(content string) error {
	t.Title = t.Meta["title"]
	t.Lines = strings.Split(strings.TrimRight(content, "\n"), "\n")
	return nil
}

func (t *Terminal) Render() (string, error) {
	var b strings.Builder
	language := strings.ToLower(t.Meta["language"])
	if language == "" {
		language = strings.ToLower(t.Title)
	}
	if language == "" || language == "terminal" {
		language = "shell"
	}
	prompt := t.Meta["prompt"]
	if prompt == "none" {
		prompt = ""
	} else if prompt == "" {
		prompt = "$"
	}
	comment := t.Meta["comment"]
	if comment == "none" {
		comment = ""
	} else if comment == "" {
		comment = "#"
	}
	frame := t.Meta["frame"]
	if frame == "" {
		frame = "macos"
	}
	switch frame {
	case "macos", "windows", "linux", "generic", "plain":
	default:
		frame = "macos"
	}

	isComment := func(value string) bool {
		return comment != "" && strings.HasPrefix(strings.TrimSpace(value), comment)
	}
	commandText := func(line string) (string, bool) {
		if prompt == "" {
			trimmed := strings.TrimSpace(line)
			return line, trimmed != "" && !isComment(trimmed)
		}
		if line == prompt {
			return "", false
		}
		prefix := prompt + " "
		if !strings.HasPrefix(line, prefix) {
			return "", false
		}
		command := strings.TrimPrefix(line, prefix)
		return command, strings.TrimSpace(command) != "" && !isComment(command)
	}

	// Collect only executable commands. Prompt prefixes, output, and complete
	// comment lines remain visible but never enter the clipboard payload.
	var commands []string
	for _, line := range t.Lines {
		if command, ok := commandText(line); ok {
			commands = append(commands, command)
		}
	}

	b.WriteString(fmt.Sprintf(`<div class="mpress-terminal mpress-terminal-%s" data-prompt="%s" data-comment="%s">`, frame, html.EscapeString(prompt), html.EscapeString(comment)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <div class="mpress-terminal-title"><span>%s</span>%s</div>`, html.EscapeString(t.Title), icons.CopyButton("mpress-copy", "Copy commands")))
	b.WriteString("\n")
	// Store commands as data attribute for smart copy
	b.WriteString(fmt.Sprintf("  <pre data-commands=\"%s\"><code>", html.EscapeString(strings.Join(commands, "\n"))))
	for _, line := range t.Lines {
		escaped := html.EscapeString(line)
		if prompt == "" {
			if isComment(line) {
				b.WriteString(fmt.Sprintf(`<span class="mpress-comment">%s</span>`, escaped))
			} else if trimmed := strings.TrimSpace(line); trimmed != "" {
				b.WriteString(`<span class="mpress-cmd">`)
				highlight.WriteLine(&b, language, line)
				b.WriteString(`</span>`)
			} else {
				b.WriteString(fmt.Sprintf(`<span class="mpress-output">%s</span>`, escaped))
			}
			b.WriteString("\n")
			continue
		}
		prefix := prompt + " "
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimPrefix(line, prefix)
			className := "mpress-cmd"
			if isComment(value) {
				className = "mpress-comment"
			}
			b.WriteString(fmt.Sprintf(`<span class="mpress-prompt">%s</span> <span class="%s">`, html.EscapeString(prompt), className))
			if className == "mpress-cmd" {
				highlight.WriteLine(&b, language, value)
			} else {
				b.WriteString(html.EscapeString(value))
			}
			b.WriteString(`</span>`)
		} else if isComment(line) {
			b.WriteString(fmt.Sprintf(`<span class="mpress-comment">%s</span>`, escaped))
		} else {
			b.WriteString(fmt.Sprintf(`<span class="mpress-output">%s</span>`, escaped))
		}
		b.WriteString("\n")
	}
	b.WriteString("</code></pre>\n</div>\n")
	return b.String(), nil
}

// --- Note Component ---

type Note struct {
	Meta    map[string]string
	Content string
}

func (n *Note) Parse(content string) error {
	n.Content = content
	return nil
}

// noteIconNames maps note types to the bundled Lucide icon set.
var noteIconNames = map[string]string{
	"info":      "circle-info",
	"caution":   "triangle-alert",
	"warning":   "triangle-alert",
	"tip":       "rocket",
	"danger":    "circle-x",
	"important": "star",
}

func (n *Note) Render() (string, error) {
	noteType := n.Meta["type"]
	if noteType == "" {
		noteType = "info"
	}
	// Caution is accepted as an authoring alias, but it renders through the
	// canonical warning contract so the two names cannot drift visually or in
	// translated labels.
	if noteType == "caution" {
		noteType = "warning"
	}

	// Map to Oat data-variant
	variant := ""
	switch noteType {
	case "warning":
		variant = ` data-variant="warning"`
	case "tip":
		variant = ` data-variant="tip"`
	case "danger":
		variant = ` data-variant="danger"`
	case "info":
		// default
	case "important":
		variant = ` data-variant="warning"`
	default:
		variant = fmt.Sprintf(` data-variant="%s"`, html.EscapeString(noteType))
	}

	iconName := noteIconNames[noteType]
	if iconName == "" {
		iconName = noteIconNames["info"]
	}
	icon := lucide(iconName, 16)

	// Label — use custom title if provided, otherwise use translated label
	label := n.Meta["title"]
	if label == "" {
		label = TForLanguage(n.Meta["__mpress_lang"], "note."+noteType)
	}

	// Output opening/closing HTML on separate lines so Goldmark processes the body as Markdown.
	// Goldmark's unsafe HTML mode passes through raw HTML blocks but processes text between them.
	var b strings.Builder
	b.WriteString(fmt.Sprintf("<aside aria-label=\"%s\" class=\"mpress-admonition mpress-admonition-%s\"%s>\n", html.EscapeString(label), html.EscapeString(noteType), variant))
	b.WriteString(fmt.Sprintf("  <p class=\"mpress-admonition-titlebar\" aria-hidden=\"true\">%s <span>%s</span></p>\n", icon, html.EscapeString(label)))
	b.WriteString("<div class=\"mpress-admonition-body\">\n\n")
	b.WriteString(n.Content)
	b.WriteString("\n\n</div>\n")
	b.WriteString("</aside>\n")
	return b.String(), nil
}

// --- Details Component ---

type Details struct {
	Meta    map[string]string
	Content string
}

func (d *Details) Parse(content string) error {
	d.Content = content
	return nil
}

func (d *Details) Render() (string, error) {
	title := d.Meta["title"]
	if title == "" {
		title = TForLanguage(d.Meta["__mpress_lang"], "details.title")
	}
	open := ""
	if d.Meta["open"] == "true" {
		open = " open"
	}
	// Output content between separate HTML tags so Goldmark processes it as Markdown
	var b strings.Builder
	disclosureIcon := `<span class="mpress-disclosure-icon">` + lucide("chevron-right", 16) + lucide("chevron-down", 16) + `</span>`
	b.WriteString(fmt.Sprintf("<details class=\"mpress-details\"%s>\n  <summary>%s<span>%s</span></summary>\n", open, disclosureIcon, html.EscapeString(title)))
	b.WriteString("<div class=\"mpress-details-content\">\n\n")
	b.WriteString(d.Content)
	b.WriteString("\n\n</div>\n")
	b.WriteString("</details>\n")
	return b.String(), nil
}

// --- Badge Component ---

type Badge struct {
	Meta    map[string]string
	Content string
}

func (b *Badge) Parse(content string) error {
	b.Content = strings.TrimSpace(content)
	return nil
}

func (b *Badge) Render() (string, error) {
	badgeType := b.Meta["type"]
	if badgeType == "" {
		badgeType = "default"
	}
	text := b.Content
	if text == "" {
		text = badgeType
	}
	return fmt.Sprintf("<span class=\"mpress-badge mpress-badge-%s\">%s</span>", html.EscapeString(badgeType), html.EscapeString(text)), nil
}

// --- API Endpoint Component ---

type API struct {
	Meta    map[string]string
	Content string
}

func (a *API) Parse(content string) error {
	a.Content = strings.TrimSpace(content)
	return nil
}

// methodColors maps HTTP methods to badge colors.
var methodColors = map[string]string{
	"GET":     "#0d9488",
	"POST":    "#3b82f6",
	"PUT":     "#d97706",
	"PATCH":   "#d97706",
	"DELETE":  "#dc2626",
	"HEAD":    "#8b5cf6",
	"OPTIONS": "#6b7280",
}

func (a *API) Render() (string, error) {
	method := strings.ToUpper(a.Meta["method"])
	if method == "" {
		method = "GET"
	}
	path := a.Meta["path"]
	if path == "" {
		path = "/"
	}

	color := methodColors[method]
	if color == "" {
		color = "#6b7280"
	}

	var b strings.Builder

	b.WriteString("<div class=\"mpress-api-endpoint\">\n")

	// Method badge + path
	b.WriteString("  <div class=\"mpress-api-header\">\n")
	b.WriteString(fmt.Sprintf("    <span class=\"mpress-api-method\" style=\"background:%s\">%s</span>\n", color, html.EscapeString(method)))
	b.WriteString(fmt.Sprintf("    <code class=\"mpress-api-path\">%s</code>\n", html.EscapeString(path)))
	b.WriteString("  </div>\n")

	// Description content — use blank lines so Goldmark processes Markdown (tables, etc.)
	if a.Content != "" {
		b.WriteString("  <div class=\"mpress-api-description\">\n\n")
		b.WriteString(a.Content)
		b.WriteString("\n\n  </div>\n")
	}

	b.WriteString("</div>\n")
	return b.String(), nil
}

// --- API Playground Component ---

// APIPlayground renders an interactive API endpoint with editable request body,
// headers, Send button, and formatted response. All client-side via fetch().
type APIPlayground struct {
	Meta    map[string]string
	Content string
}

func (a *APIPlayground) Parse(content string) error {
	a.Content = strings.TrimSpace(content)
	return nil
}

func (a *APIPlayground) Render() (string, error) {
	method := strings.ToUpper(a.Meta["method"])
	if method == "" {
		method = "GET"
	}
	path := a.Meta["path"]
	if path == "" {
		path = "/"
	}
	baseURL := a.Meta["baseUrl"]
	if baseURL == "" {
		baseURL = a.Meta["baseurl"] // case fallback
	}

	color := methodColors[method]
	if color == "" {
		color = "#6b7280"
	}

	// Generate a stable ID for this playground
	id := fmt.Sprintf("api-pg-%x", hashString(method + path + baseURL)[:8])

	var b strings.Builder

	b.WriteString(fmt.Sprintf("<div class=\"mpress-api-playground\" id=\"%s\" data-method=\"%s\" data-path=\"%s\" data-base-url=\"%s\">\n",
		id, html.EscapeString(method), html.EscapeString(path), html.EscapeString(baseURL)))

	// Header with method + path
	b.WriteString("  <div class=\"mpress-api-header\">\n")
	b.WriteString(fmt.Sprintf("    <span class=\"mpress-api-method\" style=\"background:%s\">%s</span>\n", color, html.EscapeString(method)))
	b.WriteString(fmt.Sprintf("    <code class=\"mpress-api-path\">%s</code>\n", html.EscapeString(path)))
	b.WriteString("  </div>\n")

	// Description
	if a.Content != "" {
		b.WriteString("  <div class=\"mpress-api-description\">\n\n")
		b.WriteString(a.Content)
		b.WriteString("\n\n  </div>\n")
	}

	// Request section
	b.WriteString("  <div class=\"mpress-api-pg-body\">\n")

	// Server/environment selector (if multiple servers configured)
	servers := a.Meta["servers"]
	if servers != "" {
		serverList := strings.Split(servers, ",")
		if len(serverList) > 1 {
			b.WriteString("    <div class=\"mpress-api-pg-server\">\n")
			b.WriteString("      <label>Server</label>\n")
			b.WriteString("      <select class=\"mpress-api-pg-server-select\">\n")
			for _, s := range serverList {
				s = strings.TrimSpace(s)
				label := s
				// Show friendly labels for common patterns
				if strings.Contains(s, "localhost") || strings.Contains(s, "127.0.0.1") {
					label = "Local (" + s + ")"
				} else if strings.Contains(s, "staging") {
					label = "Staging (" + s + ")"
				} else if strings.Contains(s, "sandbox") {
					label = "Sandbox (" + s + ")"
				}
				selected := ""
				if s == baseURL {
					selected = " selected"
				}
				b.WriteString(fmt.Sprintf("        <option value=\"%s\"%s>%s</option>\n", html.EscapeString(s), selected, html.EscapeString(label)))
			}
			b.WriteString("      </select>\n")
			b.WriteString("    </div>\n")
		}
	}

	// URL input
	fullURL := baseURL + path
	b.WriteString("    <div class=\"mpress-api-pg-url\">\n")
	b.WriteString("      <label>URL</label>\n")
	b.WriteString(fmt.Sprintf("      <input type=\"text\" class=\"mpress-api-pg-url-input\" value=\"%s\" />\n", html.EscapeString(fullURL)))
	b.WriteString("    </div>\n")

	// Headers section
	b.WriteString("    <details class=\"mpress-api-pg-section\">\n")
	b.WriteString(fmt.Sprintf("      <summary><span class=\"mpress-disclosure-icon\">%s%s</span><span>Headers</span></summary>\n", lucide("chevron-right", 16), lucide("chevron-down", 16)))
	b.WriteString("      <div class=\"mpress-api-pg-headers\">\n")
	b.WriteString("        <div class=\"mpress-api-pg-header-row\">\n")
	b.WriteString("          <input type=\"text\" placeholder=\"Content-Type\" value=\"Content-Type\" />\n")
	b.WriteString("          <input type=\"text\" placeholder=\"application/json\" value=\"application/json\" />\n")
	b.WriteString("        </div>\n")
	b.WriteString("        <div class=\"mpress-api-pg-header-row\">\n")
	b.WriteString("          <input type=\"text\" placeholder=\"Header name\" />\n")
	b.WriteString("          <input type=\"text\" placeholder=\"Header value\" />\n")
	b.WriteString("        </div>\n")
	b.WriteString("      </div>\n")
	b.WriteString("    </details>\n")

	// Request body (for POST, PUT, PATCH)
	if method == "POST" || method == "PUT" || method == "PATCH" {
		b.WriteString("    <details class=\"mpress-api-pg-section\" open>\n")
		b.WriteString(fmt.Sprintf("      <summary><span class=\"mpress-disclosure-icon\">%s%s</span><span>Request Body</span></summary>\n", lucide("chevron-right", 16), lucide("chevron-down", 16)))
		b.WriteString("      <textarea class=\"mpress-api-pg-request\" rows=\"6\" placeholder='{\"key\": \"value\"}'>{}</textarea>\n")
		b.WriteString("    </details>\n")
	}

	// Send button
	b.WriteString("    <div class=\"mpress-api-pg-actions\">\n")
	b.WriteString(fmt.Sprintf("      <button class=\"mpress-api-pg-send\" style=\"background:%s\">Send Request</button>\n", color))
	b.WriteString("      <span class=\"mpress-api-pg-status\"></span>\n")
	b.WriteString("    </div>\n")

	// Response section
	b.WriteString("    <div class=\"mpress-api-pg-response\" hidden>\n")
	b.WriteString("      <div class=\"mpress-api-pg-response-header\">\n")
	b.WriteString("        <span class=\"mpress-api-pg-response-status\"></span>\n")
	b.WriteString("        <span class=\"mpress-api-pg-response-time\"></span>\n")
	b.WriteString("      </div>\n")
	b.WriteString("      <pre class=\"mpress-api-pg-response-body\"><code></code></pre>\n")
	b.WriteString("    </div>\n")

	b.WriteString("  </div>\n")
	b.WriteString("</div>\n")

	return b.String(), nil
}

// hashString returns a hex hash for generating stable IDs.
func hashString(s string) string {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return fmt.Sprintf("%08x", h)
}

// --- Steps Component ---

type Steps struct {
	Meta  map[string]string
	Items []StepItem
}

type StepItem struct {
	Title   string
	Content string
}

func (s *Steps) Parse(content string) error {
	// Split content by lines starting with "### " to define steps
	locs := stepHeaderRegex.FindAllStringSubmatchIndex(content, -1)
	if len(locs) == 0 {
		orderedLocs := orderedStepHeaderRegex.FindAllStringSubmatchIndex(content, -1)
		for i, loc := range orderedLocs {
			title := ""
			if loc[4] >= 0 {
				title = content[loc[4]:loc[5]]
			} else if loc[6] >= 0 {
				title = content[loc[6]:loc[7]]
			}
			contentStart := loc[1]
			var contentEnd int
			if i+1 < len(orderedLocs) {
				contentEnd = orderedLocs[i+1][0]
			} else {
				contentEnd = len(content)
			}
			stepContent := strings.TrimSpace(content[contentStart:contentEnd])
			prefix := orderedStepPrefixRegex.FindString(content[loc[0]:loc[1]])
			stepContent = stripStepContinuationIndent(stepContent, len(prefix))
			s.Items = append(s.Items, StepItem{Title: strings.TrimSpace(title), Content: stepContent})
		}
		if len(s.Items) > 0 {
			return nil
		}
	}

	if len(locs) == 0 {
		// Treat entire content as a single step
		s.Items = append(s.Items, StepItem{Title: "Step 1", Content: strings.TrimSpace(content)})
		return nil
	}

	for i, loc := range locs {
		title := content[loc[2]:loc[3]]
		contentStart := loc[1]
		var contentEnd int
		if i+1 < len(locs) {
			contentEnd = locs[i+1][0]
		} else {
			contentEnd = len(content)
		}
		stepContent := strings.TrimSpace(content[contentStart:contentEnd])
		s.Items = append(s.Items, StepItem{Title: title, Content: stepContent})
	}
	return nil
}

func stripStepContinuationIndent(content string, width int) string {
	if width <= 0 {
		return content
	}
	indent := strings.Repeat(" ", width)
	lines := strings.SplitAfter(content, "\n")
	for i, line := range lines {
		body, newline := splitLineEnding(line)
		if strings.HasPrefix(body, indent) {
			lines[i] = strings.TrimPrefix(body, indent) + newline
		}
	}
	return strings.Join(lines, "")
}

func (s *Steps) Render() (string, error) {
	var b strings.Builder
	b.WriteString("<div class=\"mpress-timeline mpress-steps\">\n")
	for i, item := range s.Items {
		b.WriteString("  <div class=\"mpress-timeline-entry mpress-step\">\n")
		b.WriteString(fmt.Sprintf("    <div class=\"mpress-timeline-marker mpress-step-number\">%d</div>\n", i+1))
		b.WriteString("    <div class=\"mpress-timeline-body mpress-step-body\">\n")
		b.WriteString(fmt.Sprintf("      <h3 class=\"mpress-step-title\">%s</h3>\n", html.EscapeString(item.Title)))
		if item.Content != "" {
			// No indentation on closing tags — Goldmark treats 4+ spaces as code blocks
			b.WriteString("<div class=\"mpress-step-content\">\n\n")
			b.WriteString(item.Content)
			b.WriteString("\n\n</div>\n")
		}
		b.WriteString("    </div>\n")
		b.WriteString("  </div>\n")
	}
	b.WriteString("</div>\n")
	return b.String(), nil
}

// --- Cards Component ---

type Cards struct {
	Meta  map[string]string
	Items []CardItem
}

type CardItem struct {
	Icon        string
	Title       string
	Description string
	Link        string
}

func (c *Cards) Parse(content string) error {
	// Split by --- separators
	sections := cardSeparatorRegex.Split(content, -1)
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		lines := strings.SplitN(section, "\n", 2)
		titleLine := strings.TrimSpace(lines[0])
		desc := ""
		if len(lines) > 1 {
			desc = strings.TrimSpace(lines[1])
		}

		item := CardItem{}

		// Check for link: [Title](url) format
		if m := cardLinkRegex.FindStringSubmatch(titleLine); m != nil {
			titleLine = m[1]
			item.Link = m[2]
		}

		// Check for leading HTML icon span (from ProcessIconShortcodes)
		if strings.HasPrefix(titleLine, `<span class="mpress-icon"`) {
			// Extract the icon span and the remaining title
			endIdx := strings.Index(titleLine, "</span>")
			if endIdx > 0 {
				endIdx += len("</span>")
				item.Icon = titleLine[:endIdx]
				item.Title = strings.TrimSpace(titleLine[endIdx:])
			} else {
				item.Title = titleLine
			}
		} else {
			// Check for leading emoji/icon (first rune > 127 or first word is single char)
			runes := []rune(titleLine)
			if len(runes) > 0 && runes[0] > 127 {
				// Emoji prefix — find where the text starts
				idx := strings.IndexByte(titleLine, ' ')
				if idx > 0 {
					item.Icon = titleLine[:idx]
					item.Title = titleLine[idx+1:]
				} else {
					item.Title = titleLine
				}
			} else {
				item.Title = titleLine
			}
		}

		item.Description = desc
		c.Items = append(c.Items, item)
	}
	return nil
}

func (c *Cards) Render() (string, error) {
	cols := c.Meta["cols"]
	if cols == "" {
		cols = "3"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<div class=\"mpress-card-grid mpress-cards mpress-cards-%s\">\n", html.EscapeString(cols)))

	for _, item := range c.Items {
		if item.Link != "" {
			b.WriteString(fmt.Sprintf("  <a href=\"%s\" class=\"mpress-card\">\n", html.EscapeString(item.Link)))
		} else {
			b.WriteString("  <div class=\"mpress-card\">\n")
		}

		if item.Icon != "" {
			b.WriteString(fmt.Sprintf("    <div class=\"mpress-card-icon\">%s</div>\n", cardIconHTML(item.Icon)))
		}
		b.WriteString(fmt.Sprintf("    <div class=\"mpress-card-title\">%s</div>\n", html.EscapeString(item.Title)))
		if item.Description != "" {
			// No indentation on closing tags — Goldmark treats 4+ spaces as code blocks
			b.WriteString("<div class=\"mpress-card-desc\">\n\n")
			b.WriteString(item.Description)
			b.WriteString("\n\n</div>\n")
		}

		if item.Link != "" {
			b.WriteString("  </a>\n")
		} else {
			b.WriteString("  </div>\n")
		}
	}

	b.WriteString("</div>\n")
	return b.String(), nil
}

func cardIconHTML(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return ""
	}
	return lucide(icon, 20)
}

// --- Diff Component ---

type Diff struct {
	Meta   map[string]string
	Before string
	After  string
}

func (d *Diff) Parse(content string) error {
	parts := strings.SplitN(content, "---", 2)
	d.Before = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		d.After = strings.TrimSpace(parts[1])
	}
	return nil
}

func (d *Diff) Render() (string, error) {
	title := strings.TrimSpace(d.Meta["title"])
	mode := d.Meta["mode"]
	if mode == "" {
		mode = "side-by-side"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<div class=\"mpress-diff mpress-diff-%s\">\n", html.EscapeString(mode)))
	if title != "" {
		b.WriteString(fmt.Sprintf("  <div class=\"mpress-diff-title\">%s</div>\n", html.EscapeString(title)))
	}

	if mode == "inline" {
		// Unified diff: interleave removed/added lines
		b.WriteString("  <div class=\"mpress-diff-body\">\n    <pre><code>")
		beforeLines := strings.Split(d.Before, "\n")
		afterLines := strings.Split(d.After, "\n")

		// Show removed lines
		for _, line := range beforeLines {
			b.WriteString(fmt.Sprintf("<span class=\"mpress-diff-line mpress-diff-removed\">− %s</span>", html.EscapeString(line)))
		}
		// Show added lines
		for _, line := range afterLines {
			b.WriteString(fmt.Sprintf("<span class=\"mpress-diff-line mpress-diff-added\">+ %s</span>", html.EscapeString(line)))
		}
		b.WriteString("</code></pre>\n  </div>\n")
	} else {
		// Side-by-side
		b.WriteString("  <div class=\"mpress-diff-body\">\n")
		b.WriteString("    <div class=\"mpress-diff-pane mpress-diff-pane-before\">\n      <pre><code>")
		for _, line := range strings.Split(d.Before, "\n") {
			b.WriteString(fmt.Sprintf("<span class=\"mpress-diff-line\">%s</span>", html.EscapeString(line)))
		}
		b.WriteString("</code></pre>\n    </div>\n")
		b.WriteString("    <div class=\"mpress-diff-pane mpress-diff-pane-after\">\n      <pre><code>")
		for _, line := range strings.Split(d.After, "\n") {
			b.WriteString(fmt.Sprintf("<span class=\"mpress-diff-line\">%s</span>", html.EscapeString(line)))
		}
		b.WriteString("</code></pre>\n    </div>\n")
		b.WriteString("  </div>\n")
	}

	b.WriteString("</div>\n")
	return b.String(), nil
}

// --- LinkCard Component ---

type LinkCard struct {
	Meta    map[string]string
	Content string
}

func (l *LinkCard) Parse(content string) error {
	l.Content = strings.TrimSpace(content)
	return nil
}

func (l *LinkCard) Render() (string, error) {
	href := l.Meta["href"]
	if href == "" {
		href = "#"
	}
	title := l.Meta["title"]
	if title == "" {
		title = l.Content
		l.Content = ""
	}
	description := l.Content
	if description == "" {
		description = html.EscapeString(l.Meta["description"])
	}
	icon := l.Meta["icon"]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("<a href=\"%s\" class=\"mpress-linkcard\">\n", html.EscapeString(href)))
	if icon != "" {
		b.WriteString(fmt.Sprintf("  <span class=\"mpress-linkcard-icon\">%s</span>\n", lucide(icon, 20)))
	}
	b.WriteString("  <div class=\"mpress-linkcard-body\">\n")
	b.WriteString(fmt.Sprintf("    <span class=\"mpress-linkcard-title\">%s</span>\n", html.EscapeString(title)))
	if description != "" {
		b.WriteString(fmt.Sprintf("    <span class=\"mpress-linkcard-desc\">%s</span>\n", description))
	}
	b.WriteString("  </div>\n")
	b.WriteString("  <span class=\"mpress-linkcard-arrow\">" + lucide("arrow-right", 16) + "</span>\n")
	b.WriteString("</a>\n")
	return b.String(), nil
}

// --- FileTree Component ---

type FileTree struct {
	Meta  map[string]string
	Lines []string
}

type fileTreeEntry struct {
	Depth       int
	Name        string
	Description string
	IsDir       bool
}

func (f *FileTree) Parse(content string) error {
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) != "" {
			f.Lines = append(f.Lines, line)
		}
	}
	return nil
}

func (f *FileTree) Render() (string, error) {
	var b strings.Builder
	b.WriteString("<div class=\"mpress-filetree\">\n")
	b.WriteString("  <ul>\n")

	entries := parseFileTreeEntries(f.Lines)

	// Parse indentation to build tree
	type stackItem struct {
		depth int
	}
	var stack []stackItem

	for _, entry := range entries {
		// Close items at deeper or equal levels
		for len(stack) > 0 && stack[len(stack)-1].depth >= entry.Depth {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			_ = top
			b.WriteString("      </ul>\n    </li>\n")
		}

		// File icon based on extension or directory
		icon := fileIcon(entry.Name, entry.IsDir)
		description := ""
		if entry.Description != "" {
			description = fmt.Sprintf(`<span class="mpress-filetree-desc">%s</span>`, html.EscapeString(entry.Description))
		}

		if entry.IsDir {
			b.WriteString(fmt.Sprintf("    <li class=\"mpress-filetree-dir\"><span class=\"mpress-filetree-row\"><span class=\"mpress-filetree-icon\">%s</span><span class=\"mpress-filetree-name\">%s</span>%s</span>\n      <ul>\n",
				icon, html.EscapeString(entry.Name), description))
			stack = append(stack, stackItem{entry.Depth})
		} else {
			b.WriteString(fmt.Sprintf("    <li class=\"mpress-filetree-file\"><span class=\"mpress-filetree-row\"><span class=\"mpress-filetree-icon\">%s</span><span class=\"mpress-filetree-name\">%s</span>%s</span>",
				icon, html.EscapeString(entry.Name), description))
			b.WriteString("</li>\n")
		}
	}

	// Close remaining open items
	for range stack {
		b.WriteString("      </ul>\n    </li>\n")
	}

	b.WriteString("  </ul>\n</div>\n")
	return b.String(), nil
}

func parseFileTreeEntries(lines []string) []fileTreeEntry {
	entries := make([]fileTreeEntry, 0, len(lines))
	for _, line := range lines {
		entry := parseFileTreeLine(line)
		if entry.Name != "" {
			entries = append(entries, entry)
		}
	}
	for i := range entries {
		entries[i].IsDir = strings.HasSuffix(entries[i].Name, "/") || (i+1 < len(entries) && entries[i+1].Depth > entries[i].Depth)
		entries[i].Name = strings.TrimSuffix(entries[i].Name, "/")
	}
	return entries
}

func parseFileTreeLine(line string) fileTreeEntry {
	trimmedLeft := strings.TrimLeft(line, " ")
	indent := len(line) - len(trimmedLeft)
	body := strings.TrimSpace(trimmedLeft)
	if strings.HasPrefix(body, "- ") {
		body = strings.TrimSpace(body[2:])
	}

	name := body
	description := ""
	if fields := fileTreeEntryRegex.FindStringSubmatch(body); len(fields) == 3 {
		name = fields[1]
		description = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(fields[2]), "#"))
	}

	return fileTreeEntry{
		Depth:       indent / 2,
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	}
}

func fileIcon(name string, isDir bool) string {
	if isDir {
		return lucide("folder", 14)
	}
	ext := ""
	if idx := strings.LastIndex(name, "."); idx != -1 {
		ext = name[idx:]
	}
	switch ext {
	case ".md", ".markdown":
		return lucide("file-text", 14)
	case ".yaml", ".yml", ".json", ".toml":
		return lucide("file", 14)
	case ".go", ".js", ".ts", ".py", ".rs":
		return lucide("code-xml", 14)
	case ".css", ".html":
		return lucide("code-xml", 14)
	case ".png", ".jpg", ".svg", ".gif", ".ico":
		return lucide("image", 14)
	default:
		return lucide("file", 14)
	}
}

// CountExpanded counts component blocks by type in the given markdown content.
// Used by debug mode to report which components were expanded.
func CountExpanded(content string) map[string]int {
	// Protect code blocks first
	protected, _ := protectFencedCodeBlocks(content, "")

	matches := componentBlockRegex.FindAllStringSubmatch(protected, -1)
	counts := make(map[string]int)
	for _, m := range matches {
		if len(m) > 1 {
			counts[m[1]]++
		}
	}
	return counts
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
