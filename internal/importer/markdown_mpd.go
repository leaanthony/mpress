package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"

	"github.com/leaanthony/mpress/internal/mpd"
)

var mpressDirectiveOpen = regexp.MustCompile(`^([ \t]*):::(\w[\w-]*)(?:\{([^}]*)\})?[ \t]*$`)
var mpressTabLabel = regexp.MustCompile(`^([ \t]*)\[([^]]+)\][ \t]*$`)
var markdownTableSeparator = regexp.MustCompile(`^[ \t]*\|?[ \t]*:?-{3,}:?[ \t]*(?:\|[ \t]*:?-{3,}:?[ \t]*)+\|?[ \t]*$`)
var markdownComponentMetadata = regexp.MustCompile(`([\w-]+)=(?:"([^"]*)"|'([^']*)'|([^\s}]+))`)
var markdownStepItem = regexp.MustCompile(`^([ \t]*)([0-9]+)\.[ \t]+(?:\*\*([^*]+)\*\*|(.+?))[ \t]*$`)
var markdownStepHeading = regexp.MustCompile(`^([ \t]*)#{2,6}[ \t]+(.+?)[ \t]*$`)
var markdownLineRanges = regexp.MustCompile(`^[0-9,\-\s]+$`)
var markdownFenceRanges = regexp.MustCompile(`\b(ins|del)=\{([0-9,\-\s]+)\}`)
var markdownBareHighlight = regexp.MustCompile(`\{([0-9,\-\s]+)\}`)
var markdownBareFenceTitle = regexp.MustCompile(`^("[^"]*"|'[^']*')`)
var markdownTerminalDirective = regexp.MustCompile(`^([ \t]*)@terminal\{([^}]*)\}[ \t]*$`)
var markdownCardLinkTitle = regexp.MustCompile(`^\[([^]]+)\]\(([^)]+)\)$`)
var markdownAtComponent = regexp.MustCompile(`^([ \t]*)@([a-z][a-z0-9-]*)(?:\{([^}]*)\})?[ \t]*$`)
var markdownNativeLeaf = regexp.MustCompile(`^([ \t]*)@([a-z][a-z0-9-]*)(?:[ \t]+.*)?$`)
var markdownListLine = regexp.MustCompile(`^([ \t]*)(?:[-+*]|[0-9]+\.)[ \t]+`)
var markdownCompactNote = regexp.MustCompile(`^([ \t]*)@(info|tip|warning|warn|caution|danger|important|bug|example|note)\[([^]]+)\][ \t]*$`)
var markdownCompactForm = regexp.MustCompile(`^([ \t]*)@form\[([^]]+)\][ \t]*$`)
var markdownCompactButton = regexp.MustCompile(`^([ \t]*)@button\[([^]]+)\]\(([^)]+)\)(?:\{([^}]*)\})?(?:[ \t]+(.*))?$`)
var markdownBlockStart = regexp.MustCompile(`^[ \t]*(?:#{1,6}[ \t]+|>|[-+*][ \t]+|[0-9]+\.[ \t]+)`)
var markdownNonParagraphStart = regexp.MustCompile(`^[ \t]*(?:#{1,6}[ \t]+|>)`)

var markdownMPDLeafComponents = map[string]bool{
	"computed": true, "hr": true, "image": true, "import": true,
	"include": true, "input": true, "link": true, "linkcard": true, "qr": true, "video": true,
}

var markdownMPDContainerComponents = map[string]bool{
	"accessibility-demo": true, "actions": true, "api": true, "api-playground": true, "audience": true,
	"badge": true, "button": true, "calendar": true, "callout": true,
	"capabilities": true, "capability": true, "card": true, "cards": true,
	"changelog": true, "column": true, "columns": true, "comment": true,
	"container": true, "details": true, "diff": true, "docs-preview": true,
	"event": true, "explained": true, "filetree": true, "footnote": true,
	"form": true, "headline": true, "if": true, "lesson": true,
	"matrix": true, "note": true, "plan": true,
	"preview-tabs": true, "file-tabs": true, "pricing": true, "rawHTML": true, "release": true,
	"resource": true, "resources": true, "section": true, "status": true,
	"step": true, "steps": true, "tab": true, "table": true, "tabs": true,
	"terminal": true, "testimonial": true, "testimonials": true,
	"timeline": true, "tutorial": true, "variant": true,
}

var markdownPreserveComponentLines = map[string]bool{
	"comment": true, "diff": true, "explained": true, "filetree": true,
	"headline": true, "pricing": true, "rawHTML": true, "terminal": true,
	"testimonials": true,
}

// markdownToMPD converts the portable Markdown emitted by the Starlight
// normalizer into native MPress Document source. The converter is deliberately
// independent of MDX: framework components have already been reduced to the
// closed MPress component set before this function runs.
func markdownToMPD(source string) (string, error) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	metadata, body, err := markdownFrontmatterToMPD(source)
	if err != nil {
		return "", err
	}
	body = normalizeMPressMarkdownComponents(body)
	converted, err := markdownLinesToMPD(strings.Split(body, "\n"))
	if err != nil {
		return "", err
	}
	return metadata + strings.TrimLeft(converted, "\n"), nil
}

type markdownCodeSpanJoin struct {
	line string
	end  int
}

// markdownConverterOwnsLine reports whether markdownLinesToMPDContext takes a
// dedicated branch for the line, so a code span must not be joined across it.
func markdownConverterOwnsLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if markdownFenceMarker(trimmed) != "" || trimmed == ":::" || trimmed == "@end" ||
		mpressDirectiveOpen.MatchString(line) || markdownTerminalDirective.MatchString(line) {
		return true
	}
	if leaf := markdownNativeLeaf.FindStringSubmatch(line); len(leaf) > 0 && markdownMPDLeafComponents[leaf[2]] {
		return true
	}
	if strings.HasPrefix(trimmed, "<") && !strings.HasPrefix(trimmed, "<http") && !strings.HasPrefix(trimmed, "<mailto:") {
		return true
	}
	return markdownTableSeparator.MatchString(line)
}

// markdownMultilineCodeSpanLines returns the joined logical line for each code
// span that wraps onto later physical lines. Delimiters and content come from
// Goldmark segments, which exclude container prefixes between lines. Spans that
// cross a block owned by the line converter are left alone.
func markdownMultilineCodeSpanLines(lines []string) map[int]markdownCodeSpanJoin {
	source := strings.Join(lines, "\n")
	if !strings.Contains(source, "`") {
		return nil
	}
	raw := []byte(source)
	markdown := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote, extension.Typographer))
	document := markdown.Parser().Parse(text.NewReader(raw))
	lineStarts := []int{0}
	for index, character := range raw {
		if character == '\n' {
			lineStarts = append(lineStarts, index+1)
		}
	}
	lineOf := func(offset int) int {
		return sort.Search(len(lineStarts), func(index int) bool { return lineStarts[index] > offset }) - 1
	}
	type multilineSpan struct {
		start, end         int
		startLine, endLine int
		text               string
	}
	var spans []multilineSpan
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		span, ok := node.(*ast.CodeSpan)
		if !ok {
			return ast.WalkContinue, nil
		}
		first, last := -1, -1
		var content strings.Builder
		endsSpace := false
		for child := span.FirstChild(); child != nil; child = child.NextSibling() {
			segment, ok := child.(*ast.Text)
			if !ok {
				continue
			}
			if first < 0 {
				first = segment.Segment.Start
			}
			last = segment.Segment.Stop
			value := strings.ReplaceAll(string(segment.Segment.Value(raw)), "\n", " ")
			if content.Len() > 0 && !endsSpace && !strings.HasPrefix(value, " ") {
				content.WriteByte(' ')
			}
			content.WriteString(value)
			if value != "" {
				endsSpace = strings.HasSuffix(value, " ")
			}
		}
		if first < 0 || last > len(raw) || first >= last || strings.IndexByte(source[first:last], '\n') < 0 {
			return ast.WalkContinue, nil
		}
		start := first
		for start > 0 && raw[start-1] == ' ' {
			start--
		}
		for start > 0 && raw[start-1] == '`' {
			start--
		}
		if start == first {
			return ast.WalkContinue, nil
		}
		end := last
		for end < len(raw) && raw[end] == ' ' {
			end++
		}
		for end < len(raw) && raw[end] == '`' {
			end++
		}
		if end == last {
			return ast.WalkContinue, nil
		}
		startLine, endLine := lineOf(start), lineOf(end-1)
		if endLine <= startLine {
			return ast.WalkContinue, nil
		}
		for index := startLine; index <= endLine; index++ {
			if markdownConverterOwnsLine(lines[index]) {
				return ast.WalkContinue, nil
			}
		}
		spans = append(spans, multilineSpan{
			start:     start,
			end:       end,
			startLine: startLine,
			endLine:   endLine,
			text:      source[start:first] + content.String() + source[last:end],
		})
		return ast.WalkContinue, nil
	})
	joins := make(map[int]markdownCodeSpanJoin)
	for index := 0; index < len(spans); {
		groupStart, groupEnd := index, spans[index].endLine
		for index++; index < len(spans) && spans[index].startLine <= groupEnd; index++ {
			groupEnd = max(groupEnd, spans[index].endLine)
		}
		startLine := spans[groupStart].startLine
		lineStart := lineStarts[startLine]
		lineEnd := len(source)
		if groupEnd+1 < len(lineStarts) {
			lineEnd = lineStarts[groupEnd+1] - 1
		}
		var line strings.Builder
		cursor := lineStart
		for _, span := range spans[groupStart:index] {
			line.WriteString(source[cursor:span.start])
			line.WriteString(span.text)
			cursor = span.end
		}
		line.WriteString(source[cursor:lineEnd])
		joins[startLine] = markdownCodeSpanJoin{line: line.String(), end: groupEnd}
	}
	return joins
}

// normalizeMPressMarkdownComponents reduces MPress's current Markdown-facing
// @component grammar to the same portable directive stream used by framework
// importers. Code fences are opaque: component-looking examples inside them
// remain literal source.
func normalizeMPressMarkdownComponents(source string) string {
	lines := strings.Split(source, "\n")
	var output strings.Builder
	activeFence := ""
	var containers []string
	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if activeFence != "" {
			output.WriteString(line)
			output.WriteByte('\n')
			if markdownFenceClose(trimmed, activeFence) {
				activeFence = ""
			}
			continue
		}
		if marker := markdownFenceMarker(trimmed); marker != "" {
			activeFence = marker
			output.WriteString(line)
			output.WriteByte('\n')
			continue
		}
		if match := markdownCompactButton.FindStringSubmatch(line); len(match) > 0 {
			if strings.TrimSpace(match[5]) != "" {
				output.WriteString(line)
				output.WriteByte('\n')
				continue
			}
			output.WriteString(match[1] + ":::button{href=" + strconv.Quote(match[3]))
			if attributes := markdownCompactButtonMPDMetadata(match[4]); attributes != "" {
				output.WriteByte(' ')
				output.WriteString(attributes)
			}
			output.WriteString("}\n" + match[2] + "\n:::\n")
			continue
		}
		if match := markdownCompactNote.FindStringSubmatch(line); len(match) > 0 {
			kind := match[2]
			if kind == "warn" {
				kind = "warning"
			}
			output.WriteString(match[1] + ":::note{type=" + strconv.Quote(kind) + " title=" + strconv.Quote(match[3]) + "}\n")
			containers = append(containers, "note")
			continue
		}
		if match := markdownCompactForm.FindStringSubmatch(line); len(match) > 0 {
			output.WriteString(match[1] + ":::form{action=" + strconv.Quote(strings.TrimSpace(match[2])) + "}\n")
			containers = append(containers, "form")
			continue
		}
		if trimmed == "@end" {
			if len(containers) == 0 {
				continue
			}
			containers = containers[:len(containers)-1]
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			output.WriteString(indent + ":::\n")
			continue
		}
		if match := markdownAtComponent.FindStringSubmatch(line); len(match) > 0 {
			name := match[2]
			attributes := markdownMetadataSeparatorsToSpaces(match[3])
			if markdownMPDContainerComponents[name] {
				output.WriteString(match[1] + ":::" + name)
				if attributes != "" {
					output.WriteByte('{')
					output.WriteString(attributes)
					output.WriteByte('}')
				}
				output.WriteByte('\n')
				containers = append(containers, name)
				continue
			}
			if markdownMPDLeafComponents[name] {
				output.WriteString(match[1] + "@" + name)
				if converted := markdownAttributesToMPD(name, attributes); converted != "" {
					output.WriteByte(' ')
					output.WriteString(converted)
				}
				output.WriteByte('\n')
				continue
			}
		}
		output.WriteString(line)
		output.WriteByte('\n')
	}
	return strings.TrimSuffix(output.String(), "\n")
}

// MarkdownToMPD converts portable Markdown into native MPress Document source.
// It is the public entry point used by the CLI; Starlight import uses the same
// converter after normalizing framework-specific syntax.
func MarkdownToMPD(source string) (string, error) {
	return markdownToMPD(source)
}

func isMPDLeafDirective(name string) bool {
	switch name {
	case "computed", "image", "input", "linkcard", "qr", "video":
		return true
	default:
		return false
	}
}

func markdownFrontmatterToMPD(source string) (string, string, error) {
	if !strings.HasPrefix(source, "---\n") {
		return "", source, nil
	}
	end := strings.Index(source[4:], "\n---\n")
	if end < 0 {
		return "", "", fmt.Errorf("unterminated Markdown frontmatter")
	}
	frontmatter := source[4 : 4+end]
	body := source[4+end+5:]
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return "", "", fmt.Errorf("decode Markdown frontmatter: %w", err)
	}
	var output strings.Builder
	output.WriteString("---\nschema = 1\n")
	if len(document.Content) > 0 && len(document.Content[0].Content)%2 == 0 {
		mapping := document.Content[0]
		for index := 0; index < len(mapping.Content); index += 2 {
			key := mapping.Content[index].Value
			if key == "schema" {
				continue
			}
			value := yamlNodeJSONValue(mapping.Content[index+1])
			encoded, err := json.Marshal(value)
			if err != nil {
				return "", "", fmt.Errorf("encode frontmatter field %s: %w", key, err)
			}
			output.WriteString(key)
			output.WriteString(" = ")
			output.Write(encoded)
			output.WriteByte('\n')
		}
	}
	output.WriteString("---\n\n")
	return output.String(), body, nil
}

func yamlNodeJSONValue(node *yaml.Node) any {
	switch node.Kind {
	case yaml.MappingNode:
		result := make(map[string]any, len(node.Content)/2)
		for index := 0; index+1 < len(node.Content); index += 2 {
			result[node.Content[index].Value] = yamlNodeJSONValue(node.Content[index+1])
		}
		return result
	case yaml.SequenceNode:
		result := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			result = append(result, yamlNodeJSONValue(child))
		}
		return result
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!bool":
			value, _ := strconv.ParseBool(node.Value)
			return value
		case "!!int":
			value, err := strconv.ParseInt(node.Value, 0, 64)
			if err == nil {
				return value
			}
		case "!!float":
			value, err := strconv.ParseFloat(node.Value, 64)
			if err == nil {
				return value
			}
		case "!!null":
			return nil
		}
		return node.Value
	default:
		return nil
	}
}

func markdownLinesToMPD(lines []string) (string, error) {
	return markdownLinesToMPDContext(lines, false)
}

func markdownLinesToMPDContext(lines []string, preserveLines bool) (string, error) {
	var joins map[int]markdownCodeSpanJoin
	if !preserveLines {
		joins = markdownMultilineCodeSpanLines(lines)
	}
	var output strings.Builder
	for index := 0; index < len(lines); {
		line := lines[index]
		trimmed := strings.TrimSpace(line)
		if marker := markdownFenceMarker(trimmed); marker != "" {
			end := index + 1
			for end < len(lines) && !markdownFenceClose(strings.TrimSpace(lines[end]), marker) {
				end++
			}
			targetMarker := markdownMPDFenceMarker(marker, lines[index+1:min(end, len(lines))])
			outerIndent := len(line) - len(strings.TrimLeft(line, " \t"))
			normalizedIndent := normalizedMPDIndent(outerIndent)
			removeIndent := outerIndent - normalizedIndent
			for cursor := index; cursor < len(lines) && cursor <= end; cursor++ {
				fenceLine := removeLeadingIndent(lines[cursor], removeIndent)
				if cursor == index {
					fenceLine = markdownFenceOpeningToMPD(fenceLine, marker, targetMarker)
				} else if cursor == end {
					fenceLine = strings.Repeat(" ", normalizedIndent) + targetMarker
				}
				output.WriteString(fenceLine)
				output.WriteByte('\n')
			}
			if end == len(lines) {
				output.WriteString(strings.Repeat(" ", normalizedIndent))
				output.WriteString(targetMarker)
				output.WriteByte('\n')
			}
			index = end + 1
			continue
		}
		if opening := mpressDirectiveOpen.FindStringSubmatch(line); len(opening) > 0 {
			end, err := matchingMarkdownDirective(lines, index)
			if err != nil {
				return "", err
			}
			body := lines[index+1 : end]
			name := opening[2]
			indent := normalizeMPDIndentPrefix(opening[1])
			if name == "tabs" || name == "preview-tabs" || name == "file-tabs" {
				tabs, err := markdownTabsToMPD(indent, name, opening[3], body)
				if err != nil {
					return "", err
				}
				output.WriteString(tabs)
			} else if name == "steps" {
				steps, err := markdownStepsToMPD(indent, opening[3], body)
				if err != nil {
					return "", err
				}
				output.WriteString(steps)
			} else if name == "cards" {
				cards, err := markdownCardsToMPD(indent, opening[3], body)
				if err != nil {
					return "", err
				}
				output.WriteString(cards)
			} else if name == "timeline" {
				converted, err := markdownNamedListToMPD(indent, name, "event", "title", opening[3], body, true)
				if err != nil {
					return "", err
				}
				output.WriteString(converted)
			} else if name == "capabilities" {
				converted, err := markdownNamedListToMPD(indent, name, "capability", "name", opening[3], body, false)
				if err != nil {
					return "", err
				}
				output.WriteString(converted)
			} else if name == "resources" {
				converted, err := markdownResourcesToMPD(indent, opening[3], body)
				if err != nil {
					return "", err
				}
				output.WriteString(converted)
			} else {
				output.WriteString(indent)
				output.WriteByte('@')
				output.WriteString(name)
				if attributes := markdownAttributesToMPD(name, opening[3]); attributes != "" {
					output.WriteByte(' ')
					output.WriteString(attributes)
				}
				output.WriteByte('\n')
				if isMPDLeafDirective(name) {
					index = end + 1
					continue
				}
				dedented := dedentMarkdownLines(body)
				if name == "filetree" {
					dedented = normalizeFiletreeIndent(dedented)
				}
				convertedBody := ""
				if name == "table" {
					convertedBody = markdownTableBodyToMPD(dedented)
				} else {
					convertedBody, err = markdownLinesToMPDContext(dedented, markdownPreserveComponentLines[name])
				}
				if err != nil {
					return "", err
				}
				output.WriteString(convertedBody)
				output.WriteString(indent)
				output.WriteString("@end\n")
			}
			index = end + 1
			continue
		}
		if terminal := markdownTerminalDirective.FindStringSubmatch(line); len(terminal) > 0 {
			output.WriteString(normalizeMPDIndentPrefix(terminal[1]))
			output.WriteString("@terminal")
			attributes := markdownAttributesToMPD("terminal", markdownMetadataSeparatorsToSpaces(terminal[2]))
			if attributes != "" {
				output.WriteByte(' ')
				output.WriteString(attributes)
			}
			output.WriteByte('\n')
			index++
			continue
		}
		if leaf := markdownNativeLeaf.FindStringSubmatch(line); len(leaf) > 0 && markdownMPDLeafComponents[leaf[2]] {
			output.WriteString(normalizeMPDIndentPrefix(leaf[1]))
			output.WriteString(strings.TrimSpace(line))
			output.WriteByte('\n')
			index++
			continue
		}
		if trimmed == "@end" {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			output.WriteString(normalizeMPDIndentPrefix(indent))
			output.WriteString("@end\n")
			index++
			continue
		}
		if index+1 < len(lines) && strings.Contains(line, "|") && markdownTableSeparator.MatchString(lines[index+1]) {
			end := index + 2
			for end < len(lines) && strings.Contains(lines[end], "|") && strings.TrimSpace(lines[end]) != "" {
				end++
			}
			indent := normalizeMPDIndentPrefix(line[:len(line)-len(strings.TrimLeft(line, " \t"))])
			output.WriteString(indent + "@table header=true")
			if align := markdownTableAlignment(lines[index+1]); align != "" {
				output.WriteString(" align=" + align)
			}
			output.WriteByte('\n')
			output.WriteString(markdownTableRowToMPD(line) + "\n")
			for cursor := index + 2; cursor < end; cursor++ {
				output.WriteString(markdownTableRowToMPD(lines[cursor]) + "\n")
			}
			output.WriteString(indent + "@end\n")
			index = end
			continue
		}
		if strings.HasPrefix(trimmed, "<") && !strings.HasPrefix(trimmed, "<http") && !strings.HasPrefix(trimmed, "<mailto:") {
			end := markdownHTMLBlockEnd(lines, index)
			indent := normalizeMPDIndentPrefix(line[:len(line)-len(strings.TrimLeft(line, " \t"))])
			output.WriteString(indent + "@rawHTML\n")
			for cursor := index; cursor < end; cursor++ {
				output.WriteString(lines[cursor])
				output.WriteByte('\n')
			}
			output.WriteString(indent + "@end\n")
			index = end
			continue
		}
		if join, ok := joins[index]; ok {
			line, index = join.line, join.end
		} else if joined, next := markdownJoinedListLine(lines, index); next > index {
			line, index = joined, next
		}
		convertedLine := markdownInlineToMPD(normalizeMPDLineIndent(line))
		if preserveLines {
			convertedLine = line
		}
		if !preserveLines && markdownLineIsSoftWrapped(lines, index) {
			convertedLine += `\`
		}
		output.WriteString(convertedLine)
		output.WriteByte('\n')
		index++
	}
	return output.String(), nil
}

func normalizeFiletreeIndent(lines []string) []string {
	unit := 0
	for _, line := range lines {
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent > 0 && (unit == 0 || indent < unit) {
			unit = indent
		}
	}
	if unit <= 2 {
		return lines
	}
	result := append([]string(nil), lines...)
	for index, line := range result {
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent > 0 {
			result[index] = strings.Repeat(" ", indent/unit*2) + strings.TrimLeft(line, " \t")
		}
	}
	return result
}

func markdownJoinedListLine(lines []string, index int) (string, int) {
	match := markdownListLine.FindStringSubmatch(lines[index])
	if len(match) == 0 {
		return lines[index], index
	}
	baseIndent := len(match[1])
	joined := strings.TrimRight(lines[index], " \t")
	next := index + 1
	for next < len(lines) {
		candidate := lines[next]
		trimmed := strings.TrimSpace(candidate)
		indent := len(candidate) - len(strings.TrimLeft(candidate, " \t"))
		if trimmed == "" || indent <= baseIndent || markdownBlockStart.MatchString(candidate) || markdownFenceMarker(trimmed) != "" || strings.HasPrefix(trimmed, "@") {
			break
		}
		joined += " " + trimmed
		next++
	}
	if next == index+1 {
		return lines[index], index
	}
	return joined, next - 1
}

func markdownLineIsSoftWrapped(lines []string, index int) bool {
	if index+1 >= len(lines) {
		return false
	}
	current := lines[index]
	next := lines[index+1]
	trimmedCurrent := strings.TrimSpace(current)
	trimmedNext := strings.TrimSpace(next)
	if trimmedCurrent == "" || trimmedNext == "" || strings.HasSuffix(current, `\`) || strings.HasSuffix(current, "  ") {
		return false
	}
	if markdownNonParagraphStart.MatchString(current) || trimmedCurrent == "---" || trimmedCurrent == "***" || trimmedCurrent == "___" {
		return false
	}
	if trimmedNext == ":::" || trimmedNext == ":::endtabs" || trimmedNext == "@end" || mpressDirectiveOpen.MatchString(next) || markdownFenceMarker(trimmedNext) != "" {
		return false
	}
	if markdownBlockStart.MatchString(next) || markdownTableSeparator.MatchString(next) || trimmedNext == "---" || trimmedNext == "***" || trimmedNext == "___" {
		return false
	}
	if strings.HasPrefix(trimmedNext, "<") && !strings.HasPrefix(trimmedNext, "<http") && !strings.HasPrefix(trimmedNext, "<mailto:") {
		return false
	}
	return true
}

func markdownNamedListToMPD(indent, outer, child, titleAttribute, attributes string, body []string, ordered bool) (string, error) {
	pattern := regexp.MustCompile(`^[ \t]*(?:[0-9]+\.|-)[ \t]+\*\*([^*]+)\*\*[ \t]*(.*)$`)
	var output strings.Builder
	output.WriteString(indent + "@" + outer)
	if converted := markdownAttributesToMPD(outer, attributes); converted != "" {
		output.WriteByte(' ')
		output.WriteString(converted)
	}
	output.WriteByte('\n')
	count := 0
	for _, line := range body {
		match := pattern.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		if ordered && !regexp.MustCompile(`^[ \t]*[0-9]+\.`).MatchString(line) {
			continue
		}
		if !ordered && !regexp.MustCompile(`^[ \t]*-`).MatchString(line) {
			continue
		}
		count++
		output.WriteString(indent + "@" + child + " " + titleAttribute + "=" + strconv.Quote(strings.TrimSpace(match[1])) + "\n")
		if text := strings.TrimSpace(match[2]); text != "" {
			converted, err := markdownLinesToMPD([]string{text})
			if err != nil {
				return "", err
			}
			output.WriteString(converted)
		}
		output.WriteString(indent + "@end\n")
	}
	if count == 0 {
		return "", fmt.Errorf("%s requires its documented list syntax", outer)
	}
	output.WriteString(indent + "@end\n")
	return output.String(), nil
}

func markdownResourcesToMPD(indent, attributes string, body []string) (string, error) {
	sections := make([][]string, 0, 4)
	start := 0
	for index, line := range body {
		if strings.TrimSpace(line) == "---" {
			sections = append(sections, body[start:index])
			start = index + 1
		}
	}
	sections = append(sections, body[start:])
	var output strings.Builder
	output.WriteString(indent + "@resources")
	if converted := markdownAttributesToMPD("resources", attributes); converted != "" {
		output.WriteByte(' ')
		output.WriteString(converted)
	}
	output.WriteByte('\n')
	count := 0
	for _, section := range sections {
		section = trimBlankMarkdownLines(dedentMarkdownLines(section))
		if len(section) < 2 {
			continue
		}
		link := markdownCardLinkTitle.FindStringSubmatch(strings.TrimSpace(section[0]))
		if len(link) != 3 {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(section[1]), "# "))
		if title == "" {
			continue
		}
		count++
		output.WriteString(indent + "@resource label=" + strconv.Quote(link[1]) + " href=" + strconv.Quote(link[2]) + " title=" + strconv.Quote(title) + "\n")
		converted, err := markdownLinesToMPD(section[2:])
		if err != nil {
			return "", err
		}
		output.WriteString(converted)
		output.WriteString(indent + "@end\n")
	}
	if count == 0 {
		return "", errors.New("resources requires a link, heading, and description")
	}
	output.WriteString(indent + "@end\n")
	return output.String(), nil
}

func markdownCardsToMPD(indent, attributes string, body []string) (string, error) {
	sections := make([][]string, 0, 4)
	start := 0
	for index, line := range body {
		if strings.TrimSpace(line) != "---" {
			continue
		}
		sections = append(sections, body[start:index])
		start = index + 1
	}
	sections = append(sections, body[start:])

	var output strings.Builder
	output.WriteString(indent + "@cards")
	if converted := markdownAttributesToMPD("cards", attributes); converted != "" {
		output.WriteByte(' ')
		output.WriteString(converted)
	}
	output.WriteByte('\n')
	for _, section := range sections {
		section = trimBlankMarkdownLines(dedentMarkdownLines(section))
		if len(section) == 0 {
			continue
		}
		title := strings.TrimSpace(section[0])
		href := ""
		if match := markdownCardLinkTitle.FindStringSubmatch(title); len(match) == 3 {
			title = match[1]
			href = match[2]
		}
		output.WriteString(indent + "@card title=" + strconv.Quote(title))
		if href != "" {
			output.WriteString(" href=" + strconv.Quote(href))
		}
		output.WriteByte('\n')
		converted, err := markdownLinesToMPD(section[1:])
		if err != nil {
			return "", err
		}
		output.WriteString(converted)
		output.WriteString(indent + "@end\n")
	}
	output.WriteString(indent + "@end\n")
	return output.String(), nil
}

func trimBlankMarkdownLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func markdownStepsToMPD(indent, attributes string, body []string) (string, error) {
	type step struct {
		title string
		body  []string
	}
	var steps []step
	baseIndent := -1
	for index := 0; index < len(body); {
		match := markdownStepItem.FindStringSubmatch(body[index])
		heading := markdownStepHeading.FindStringSubmatch(body[index])
		if len(match) == 0 && len(heading) == 0 {
			index++
			continue
		}
		currentIndent := 0
		title := ""
		if len(match) > 0 {
			currentIndent = len(match[1])
			title = match[3]
			if title == "" {
				title = match[4]
			}
		} else {
			currentIndent = len(heading[1])
			title = heading[2]
		}
		if baseIndent < 0 {
			baseIndent = currentIndent
		}
		if currentIndent != baseIndent {
			index++
			continue
		}
		end := index + 1
		for end < len(body) {
			next := markdownStepItem.FindStringSubmatch(body[end])
			nextHeading := markdownStepHeading.FindStringSubmatch(body[end])
			if (len(next) > 0 && len(next[1]) == baseIndent) || (len(nextHeading) > 0 && len(nextHeading[1]) == baseIndent) {
				break
			}
			end++
		}
		stepBody := append([]string(nil), body[index+1:end]...)
		continuationIndent := baseIndent + 3
		for lineIndex, line := range stepBody {
			if strings.TrimSpace(line) == "" {
				continue
			}
			remove := 0
			for remove < len(line) && remove < continuationIndent && line[remove] == ' ' {
				remove++
			}
			stepBody[lineIndex] = line[remove:]
		}
		title = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(title), "# "))
		steps = append(steps, step{title: title, body: stepBody})
		index = end
	}
	if len(steps) == 0 {
		var output strings.Builder
		output.WriteString(indent + "@steps")
		if converted := markdownAttributesToMPD("steps", attributes); converted != "" {
			output.WriteByte(' ')
			output.WriteString(converted)
		}
		output.WriteByte('\n')
		converted, err := markdownLinesToMPD(body)
		if err != nil {
			return "", err
		}
		output.WriteString(converted)
		output.WriteString(indent + "@end\n")
		return output.String(), nil
	}
	var output strings.Builder
	output.WriteString(indent + "@steps")
	if converted := markdownAttributesToMPD("steps", attributes); converted != "" {
		output.WriteByte(' ')
		output.WriteString(converted)
	}
	output.WriteByte('\n')
	for _, step := range steps {
		output.WriteString(indent + "@step title=" + strconv.Quote(step.title) + "\n")
		converted, err := markdownLinesToMPD(step.body)
		if err != nil {
			return "", err
		}
		output.WriteString(converted)
		output.WriteString(indent + "@end\n")
	}
	output.WriteString(indent + "@end\n")
	return output.String(), nil
}

func matchingMarkdownDirective(lines []string, opening int) (int, error) {
	depth := 0
	fence := ""
	for index := opening + 1; index < len(lines); index++ {
		trimmed := strings.TrimSpace(lines[index])
		if fence != "" {
			if markdownFenceClose(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if marker := markdownFenceMarker(trimmed); marker != "" {
			fence = marker
			continue
		}
		if trimmed == ":::" || trimmed == ":::endtabs" {
			if depth == 0 {
				return index, nil
			}
			depth--
			continue
		}
		if mpressDirectiveOpen.MatchString(lines[index]) {
			depth++
		}
	}
	return 0, fmt.Errorf("unclosed MPress Markdown directive on line %d", opening+1)
}

func markdownTabsToMPD(indent, name, attributes string, body []string) (string, error) {
	var output strings.Builder
	output.WriteString(indent + "@" + name)
	if converted := markdownAttributesToMPD(name, attributes); converted != "" {
		output.WriteByte(' ')
		output.WriteString(converted)
	}
	output.WriteByte('\n')
	labelStart := -1
	label := ""
	depth := 0
	flush := func(end int) error {
		if labelStart < 0 {
			return nil
		}
		output.WriteString(indent + "@tab label=" + strconv.Quote(label) + "\n")
		converted, err := markdownLinesToMPD(dedentMarkdownLines(body[labelStart:end]))
		if err != nil {
			return err
		}
		output.WriteString(converted)
		output.WriteString(indent + "@end\n")
		return nil
	}
	fence := ""
	for index, line := range body {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if markdownFenceClose(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if marker := markdownFenceMarker(trimmed); marker != "" {
			fence = marker
			continue
		}
		if (trimmed == ":::" || trimmed == ":::endtabs") && depth > 0 {
			depth--
		} else if mpressDirectiveOpen.MatchString(line) {
			depth++
		}
		match := mpressTabLabel.FindStringSubmatch(line)
		if depth != 0 || len(match) == 0 {
			continue
		}
		if err := flush(index); err != nil {
			return "", err
		}
		label = match[2]
		labelStart = index + 1
	}
	if err := flush(len(body)); err != nil {
		return "", err
	}
	output.WriteString(indent + "@end\n")
	return output.String(), nil
}

// markdownCompactButtonMPDMetadata converts the legacy
// @button[Label](href){...} metadata into canonical :::button attributes. The
// legacy renderer accepts primary/secondary/outline flags (outline wins over
// secondary, which wins over primary) plus action, size, icon and icon-right;
// it ignores other metadata and defaults the variant to secondary. Pipe and
// space separators, quoted values and literal pipes inside quotes keep their
// existing meaning.
func markdownCompactButtonMPDMetadata(source string) string {
	if strings.TrimSpace(source) == "" {
		return ""
	}
	source = markdownMetadataSeparatorsToSpaces(source)
	metadata := make(map[string]string)
	last := 0
	for _, match := range markdownComponentMetadata.FindAllStringSubmatchIndex(source, -1) {
		for _, flag := range strings.Fields(source[last:match[0]]) {
			metadata[flag] = "true"
		}
		value := ""
		for _, group := range [][2]int{{4, 5}, {6, 7}, {8, 9}} {
			if match[group[0]] >= 0 {
				value = source[match[group[0]]:match[group[1]]]
				break
			}
		}
		metadata[source[match[2]:match[3]]] = value
		last = match[1]
	}
	for _, flag := range strings.Fields(source[last:]) {
		metadata[flag] = "true"
	}
	variant := "secondary"
	for _, candidate := range []string{"primary", "secondary", "outline"} {
		if metadata[candidate] == "true" {
			variant = candidate
		}
	}
	attributes := []string{"variant=" + strconv.Quote(variant)}
	for _, key := range []string{"action", "size", "icon", "icon-right"} {
		if value := metadata[key]; value != "" {
			attributes = append(attributes, key+"="+strconv.Quote(value))
		}
	}
	return strings.Join(attributes, " ")
}

func markdownAttributesToMPD(component, source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	var output []string
	last := 0
	for _, match := range markdownComponentMetadata.FindAllStringSubmatchIndex(source, -1) {
		output = appendMarkdownBareFlags(output, component, source[last:match[0]])
		value := ""
		for _, group := range [][2]int{{4, 5}, {6, 7}, {8, 9}} {
			if match[group[0]] >= 0 {
				value = source[match[group[0]]:match[group[1]]]
				break
			}
		}
		output = append(output, source[match[2]:match[3]]+"="+strconv.Quote(value))
		last = match[1]
	}
	output = appendMarkdownBareFlags(output, component, source[last:])
	return strings.Join(output, " ")
}

// appendMarkdownBareFlags keeps shorthand Boolean metadata, such as the
// header, search, and paginate flags on @table, which have no "=" or value.
// The separators follow the same space-or-pipe grammar as explicit attributes.
// Markdown-only flags need explicit values because native shorthand is narrower.
func appendMarkdownBareFlags(output []string, component, source string) []string {
	for _, token := range strings.FieldsFunc(source, func(character rune) bool {
		return character == ' ' || character == '\t' || character == '|'
	}) {
		if mpd.IsBooleanAttribute(token) {
			output = append(output, token)
		} else if component == "image" && token == "expand" || component == "preview-tabs" && token == "source" {
			output = append(output, token+`="true"`)
		}
	}
	return output
}

// markdownMetadataSeparatorsToSpaces normalizes pipe-separated metadata without
// rewriting a literal pipe inside a quoted attribute value.
func markdownMetadataSeparatorsToSpaces(source string) string {
	var output strings.Builder
	var quote byte
	for index := 0; index < len(source); index++ {
		character := source[index]
		if character == '\\' && index+1 < len(source) {
			output.WriteByte(character)
			index++
			output.WriteByte(source[index])
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			}
			output.WriteByte(character)
			continue
		}
		switch character {
		case '"', '\'':
			quote = character
			output.WriteByte(character)
		case '|':
			output.WriteByte(' ')
		default:
			output.WriteByte(character)
		}
	}
	return output.String()
}

func markdownInlineToMPD(line string) string {
	contentStart := len(line) - len(strings.TrimLeft(line, " \t"))
	if contentStart < len(line) && line[contentStart] == '@' {
		line = line[:contentStart] + `\` + line[contentStart:]
	}
	var output strings.Builder
	for index := 0; index < len(line); {
		if line[index] == '\\' && index+1 < len(line) {
			output.WriteString(line[index : index+2])
			index += 2
			continue
		}
		if line[index] == '`' {
			run := 1
			for index+run < len(line) && line[index+run] == '`' {
				run++
			}
			end := markdownCodeSpanCloseIndex(line, index+run, run)
			if end < 0 {
				output.WriteByte('\\')
				output.WriteString(line[index : index+run])
				index += run
				continue
			}
			end += run
			output.WriteString(line[index:end])
			index = end
			continue
		}
		if strings.HasPrefix(line[index:], "**") {
			output.WriteByte('*')
			index += 2
			continue
		}
		if line[index] == '*' {
			if index == contentStart && index+1 < len(line) && (line[index+1] == ' ' || line[index+1] == '\t') {
				output.WriteByte('-')
				index++
				continue
			}
			output.WriteByte('_')
			index++
			continue
		}
		output.WriteByte(line[index])
		index++
	}
	return output.String()
}

func normalizeMPDLineIndent(line string) string {
	spaces := 0
	for spaces < len(line) && line[spaces] == ' ' {
		spaces++
	}
	normalized := normalizedMPDIndent(spaces)
	if normalized == spaces {
		return line
	}
	return strings.Repeat(" ", normalized) + line[spaces:]
}

func dedentMarkdownLines(lines []string) []string {
	indent := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		current := 0
		for current < len(line) && (line[current] == ' ' || line[current] == '\t') {
			current++
		}
		indent = current
		break
	}
	if indent <= 0 {
		return append([]string(nil), lines...)
	}
	result := make([]string, len(lines))
	for index, line := range lines {
		remove := 0
		for remove < len(line) && remove < indent && (line[remove] == ' ' || line[remove] == '\t') {
			remove++
		}
		result[index] = line[remove:]
	}
	return result
}

func normalizedMPDIndent(spaces int) int {
	if spaces <= 1 {
		return 0
	}
	return ((spaces + 3) / 4) * 2
}

func normalizeMPDIndentPrefix(prefix string) string {
	return strings.Repeat(" ", normalizedMPDIndent(len(prefix)))
}

func removeLeadingIndent(line string, count int) string {
	removed := 0
	for removed < len(line) && removed < count && (line[removed] == ' ' || line[removed] == '\t') {
		removed++
	}
	return line[removed:]
}

func markdownFenceOpeningToMPD(line, sourceMarker, targetMarker string) string {
	indentLength := len(line) - len(strings.TrimLeft(line, " \t"))
	indent := line[:indentLength]
	opening := strings.TrimSpace(line[indentLength:])
	if !strings.HasPrefix(opening, sourceMarker) {
		return line
	}
	info := strings.TrimSpace(strings.TrimPrefix(opening, sourceMarker))
	if info == "" {
		return indent + targetMarker
	}
	languageEnd := strings.IndexAny(info, " \t{")
	if languageEnd < 0 {
		return indent + targetMarker + info
	}
	language := info[:languageEnd]
	metadata := strings.TrimSpace(info[languageEnd:])
	if metadata == "" {
		return indent + targetMarker + language
	}
	if strings.HasPrefix(metadata, "{") && strings.HasSuffix(metadata, "}") && balancedBraces(metadata) && !strings.Contains(metadata[1:len(metadata)-1], "} {") {
		inner := strings.TrimSpace(metadata[1 : len(metadata)-1])
		if markdownLineRanges.MatchString(inner) {
			return indent + targetMarker + language + ` {highlight=` + strconv.Quote(inner) + `}`
		}
		return indent + targetMarker + language + " " + metadata
	}
	metadata = markdownBareFenceTitle.ReplaceAllString(metadata, `title=$1`)
	metadata = markdownFenceRanges.ReplaceAllString(metadata, `${1}="${2}"`)
	metadata = markdownBareHighlight.ReplaceAllString(metadata, `highlight="$1"`)
	return indent + targetMarker + language + " {" + strings.TrimSpace(metadata) + "}"
}

func markdownMPDFenceMarker(sourceMarker string, body []string) string {
	if sourceMarker[0] == '`' {
		return sourceMarker
	}
	width := 3
	for _, line := range body {
		for index := 0; index < len(line); {
			if line[index] != '`' {
				index++
				continue
			}
			end := index
			for end < len(line) && line[end] == '`' {
				end++
			}
			if end-index >= width {
				width = end - index + 1
			}
			index = end
		}
	}
	return strings.Repeat("`", width)
}

func balancedBraces(value string) bool {
	depth := 0
	for _, char := range value {
		switch char {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func markdownFenceMarker(line string) string {
	if len(line) < 3 || line[0] != '`' && line[0] != '~' {
		return ""
	}
	end := 0
	for end < len(line) && line[end] == line[0] {
		end++
	}
	if end < 3 {
		return ""
	}
	return line[:end]
}

func markdownFenceClose(line, marker string) bool {
	if len(line) < len(marker) || line[0] != marker[0] {
		return false
	}
	run := 0
	for run < len(line) && line[run] == marker[0] {
		run++
	}
	return run >= len(marker) && strings.TrimSpace(line[run:]) == ""
}

func markdownTableRowToMPD(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") {
		line = "| " + line
	}
	if !strings.HasSuffix(line, "|") {
		line += " |"
	}
	line = escapeMarkdownTableCodePipes(line)
	line = escapeMarkdownTableBracketPipes(line)
	return markdownInlineToMPD(line)
}

// markdownTableBodyToMPD converts the authored rows of a rich @table component
// to native table rows. Rows are already explicit boundaries, so they must not
// inherit paragraph soft-wrap backslashes or component line normalization.
func markdownTableBodyToMPD(lines []string) string {
	var output strings.Builder
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		output.WriteString(markdownTableRowToMPD(line))
		output.WriteByte('\n')
	}
	return output.String()
}

func markdownTableAlignment(separator string) string {
	separator = strings.TrimSpace(separator)
	separator = strings.TrimPrefix(separator, "|")
	separator = strings.TrimSuffix(separator, "|")
	parts := strings.Split(separator, "|")
	align := make([]string, 0, len(parts))
	meaningful := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		left, right := strings.HasPrefix(part, ":"), strings.HasSuffix(part, ":")
		value := "default"
		switch {
		case left && right:
			value = "center"
		case left:
			value = "left"
		case right:
			value = "right"
		}
		meaningful = meaningful || value != "default"
		align = append(align, strconv.Quote(value))
	}
	if !meaningful {
		return ""
	}
	return "[" + strings.Join(align, ",") + "]"
}

func escapeMarkdownTableBracketPipes(line string) string {
	var output strings.Builder
	bracketDepth := 0
	for index := 0; index < len(line); index++ {
		if line[index] == '\\' && index+1 < len(line) {
			output.WriteString(line[index : index+2])
			index++
			continue
		}
		switch line[index] {
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '|':
			if bracketDepth > 0 {
				output.WriteByte('\\')
			}
		}
		output.WriteByte(line[index])
	}
	return output.String()
}

func escapeMarkdownTableCodePipes(line string) string {
	var output strings.Builder
	codeFenceWidth := 0
	for index := 0; index < len(line); {
		if line[index] == '\\' && index+1 < len(line) {
			output.WriteString(line[index : index+2])
			index += 2
			continue
		}
		if line[index] == '`' {
			run := markdownBacktickRun(line, index)
			if codeFenceWidth == 0 {
				// An unmatched backtick belongs to literal cell text, not to a
				// code span: it must not hide the row's remaining separators.
				if markdownCodeSpanCloses(line, index+run, run) {
					codeFenceWidth = run
				}
			} else if run == codeFenceWidth {
				codeFenceWidth = 0
			}
			output.WriteString(line[index : index+run])
			index += run
			continue
		}
		if line[index] == '|' && codeFenceWidth > 0 {
			output.WriteByte('\\')
		}
		output.WriteByte(line[index])
		index++
	}
	return output.String()
}

// markdownBacktickRun returns the length of the backtick run starting at index.
func markdownBacktickRun(line string, index int) int {
	run := 1
	for index+run < len(line) && line[index+run] == '`' {
		run++
	}
	return run
}

// markdownCodeSpanCloseIndex returns the index of the backtick run that closes
// a span opened with width backticks, or -1 when the span is unclosed. Runs of
// any other width are span content, not a closing delimiter.
func markdownCodeSpanCloseIndex(line string, index, width int) int {
	for index < len(line) {
		if line[index] != '`' {
			index++
			continue
		}
		run := markdownBacktickRun(line, index)
		if run == width {
			return index
		}
		index += run
	}
	return -1
}

// markdownCodeSpanCloses reports whether a run of exactly width backticks
// closes a code span opened before index, matching CommonMark's rule that the
// closing delimiter must have the same length as the opening one.
func markdownCodeSpanCloses(line string, index, width int) bool {
	for index < len(line) {
		if line[index] == '\\' && index+1 < len(line) {
			index += 2
			continue
		}
		if line[index] == '`' {
			run := markdownBacktickRun(line, index)
			if run == width {
				return true
			}
			index += run
			continue
		}
		index++
	}
	return false
}

func markdownHTMLBlockEnd(lines []string, start int) int {
	trimmed := strings.TrimSpace(lines[start])
	if strings.Contains(trimmed, "></") || strings.HasSuffix(trimmed, "/>") || strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
		return start + 1
	}
	nameEnd := strings.IndexAny(strings.TrimPrefix(trimmed, "<"), " >")
	if nameEnd < 0 {
		return start + 1
	}
	name := strings.TrimPrefix(trimmed, "<")[:nameEnd]
	closing := "</" + name + ">"
	for index := start; index < len(lines); index++ {
		if strings.Contains(lines[index], closing) {
			return index + 1
		}
	}
	return start + 1
}
