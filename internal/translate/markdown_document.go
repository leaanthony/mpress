package translate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/mpd"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const markdownTranslationFormat = "markdown-blocks-v2"

// ExtractMarkdown records complete inline blocks for translation, retaining
// source byte ranges. The legacy literal-text extractor remains available for
// browser quick edits and migration of its existing sidecars.
func ExtractMarkdown(source []byte) (*Document, error) {
	legacy, err := Extract(source)
	if err != nil {
		return nil, err
	}
	document := &Document{Source: source, Format: markdownTranslationFormat, Title: legacy.Title, Outline: legacy.Outline, TranslationKey: legacy.TranslationKey}
	start := frontmatterEnd(source)
	document.Segments, err = markdownMetadata(source, start)
	if err != nil {
		return nil, err
	}
	view, attributes := markdownDirectiveView(source, start)
	document.Segments = append(document.Segments, attributes...)
	markdownParser := extractionParserPool.Get().(parser.Parser)
	root := markdownParser.Parse(text.NewReader(view[start:]))
	extractionParserPool.Put(markdownParser)
	block, html, diagram := 0, 0, 0
	err = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch item := node.(type) {
		case *ast.Paragraph, *ast.Heading, *ast.TextBlock, *extast.TableCell:
			if node.Lines().Len() == 0 {
				return ast.WalkContinue, nil
			}
			first := node.Lines().At(0).Start + start
			last := node.Lines().At(node.Lines().Len()-1).Stop + start
			for last > first && strings.ContainsRune(" \t\r\n", rune(source[last-1])) {
				last--
			}
			if first >= last {
				return ast.WalkSkipChildren, nil
			}
			block++
			kind := "text"
			if _, ok := node.(*ast.Heading); ok {
				kind = "heading"
			}
			if _, ok := node.(*extast.TableCell); ok {
				kind = "cell"
			}
			raw := string(source[first:last])
			protected, placeholders := protectMarkdownInline(node, view[start:], first-start, last-start)
			if translatableText(raw) {
				document.Segments = append(document.Segments, Segment{ID: blockSegmentID(block, kind, 1), Kind: kind, Original: raw, Text: protected, Start: first, End: last, SourceHash: Hash(raw), Placeholders: placeholders, Protected: !translatableText(translationPlaceholderPattern.ReplaceAllString(protected, ""))})
			}
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock:
			html++
			first := item.Lines().At(0).Start + start
			last := item.Lines().At(item.Lines().Len()-1).Stop + start
			if item.HasClosure() {
				last = item.ClosureLine.Stop + start
			}
			document.Segments = append(document.Segments, htmlSegments(source[first:last], first, fmt.Sprintf("h%04d", html), "")...)
			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock:
			if item.Info == nil || string(item.Language(view[start:])) != "d2" || item.Lines().Len() == 0 {
				return ast.WalkSkipChildren, nil
			}
			diagram++
			first := item.Lines().At(0).Start + start
			last := item.Lines().At(item.Lines().Len()-1).Stop + start
			segments, err := d2Segments(source[first:last], first, last, diagram, "")
			if err != nil {
				return ast.WalkStop, err
			}
			document.Segments = append(document.Segments, segments...)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(document.Segments, func(i, j int) bool { return document.Segments[i].Start < document.Segments[j].Start })
	return document, nil
}

// The gaps between Goldmark text nodes contain code and inline syntax. Their
// outer whitespace is prose, so it must not bind an inline token to word order.
func protectMarkdownInline(node ast.Node, source []byte, start, end int) (string, map[string]string) {
	var textRanges []mpdProtectedRange
	opaqueHTML := 0
	_ = ast.Walk(node, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if item, ok := child.(*ast.RawHTML); ok {
			for i := 0; i < item.Segments.Len(); i++ {
				span := item.Segments.At(i)
				raw := string(span.Value(source))
				tag := strings.ToLower(strings.Trim(strings.Fields(raw)[0], "<>/"))
				if nonProseHTML(tag) {
					if strings.HasPrefix(raw, "</") {
						opaqueHTML = max(0, opaqueHTML-1)
					} else if !strings.HasSuffix(raw, "/>") {
						opaqueHTML++
					}
				}
			}
		}
		if item, ok := child.(*ast.Text); ok && opaqueHTML == 0 && !excludedTextNode(item) {
			textRanges = append(textRanges, mpdProtectedRange{max(start, item.Segment.Start), min(end, item.Segment.Stop)})
		}
		return ast.WalkContinue, nil
	})
	var ranges []mpdProtectedRange
	cursor := start
	for _, span := range textRanges {
		if span.start > cursor {
			ranges = append(ranges, markdownSyntaxRanges(source, cursor, span.start)...)
		}
		cursor = max(cursor, span.end)
	}
	if cursor < end {
		ranges = append(ranges, markdownSyntaxRanges(source, cursor, end)...)
	}
	for _, match := range protectedPattern.FindAllIndex(source[start:end], -1) {
		ranges = append(ranges, mpdProtectedRange{start + match[0], start + match[1]})
	}
	// GFM autolink recognition depends on surrounding punctuation. Protect bare
	// URLs also when a target uses full-width punctuation around the same URL.
	for _, match := range markdownBareURL.FindAllIndex(source[start:end], -1) {
		a, b := start+match[0], start+match[1]
		depth := 0
		for i := a; i < b; i++ {
			if source[i] == '(' {
				depth++
			}
			if source[i] == ')' {
				if depth == 0 {
					b = i
					break
				}
				depth--
			}
			if source[i] == '`' || source[i] == ']' || source[i] == '}' {
				b = i
				break
			}
		}
		for b > a && strings.ContainsRune(".,;:!?", rune(source[b-1])) {
			b--
		}
		for b > a && source[b-1] == ')' && bytes.Count(source[a:b], []byte(")")) > bytes.Count(source[a:b], []byte("(")) {
			b--
		}
		ranges = append(ranges, mpdProtectedRange{a, b})
	}
	return protectInlineRanges(source, start, end, ranges)
}

var markdownBareURL = regexp.MustCompile(`(?i)(?:https?://|www\.)[^\s<>"'\x{3000}-\x{303f}\x{ff00}-\x{ffef}]+`)

func markdownSyntaxRanges(source []byte, start, end int) []mpdProtectedRange {
	var spans []mpdProtectedRange
	for i := start; i < end; {
		if strings.ContainsRune(" \t\r\n", rune(source[i])) {
			i++
			continue
		}
		next := i + 1
		switch source[i] {
		case '`':
			for next < end && source[next] == '`' {
				next++
			}
			marker := source[i:next]
			if close := bytes.Index(source[next:end], marker); close >= 0 {
				next += close + len(marker)
			}
		case '<':
			if close := bytes.IndexByte(source[next:end], '>'); close >= 0 {
				next += close + 1
				tag := strings.ToLower(strings.Trim(strings.Fields(string(source[i:next]))[0], "<>/"))
				if nonProseHTML(tag) && source[i+1] != '/' {
					if finish := bytes.Index(bytes.ToLower(source[next:end]), []byte("</"+tag+">")); finish >= 0 {
						next += finish + len(tag) + 3
					}
				}
			}
		case ']':
			if next < end && source[next] == '(' {
				depth := 1
				next++
				for next < end && depth > 0 {
					if source[next] == '\\' {
						next += 2
						continue
					}
					if source[next] == '(' {
						depth++
					}
					if source[next] == ')' {
						depth--
					}
					next++
				}
			}
		case '*', '_', '~':
			for next < end && source[next] == source[i] {
				next++
			}
		case '!':
			if next < end && source[next] == '[' {
				next++
			}
		}
		next = min(next, end)
		spans = append(spans, mpdProtectedRange{i, next})
		i = next
	}
	return spans
}

func protectInlineRanges(source []byte, start, end int, ranges []mpdProtectedRange) (string, map[string]string) {
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	var merged []mpdProtectedRange
	for _, span := range ranges {
		if span.start >= span.end {
			continue
		}
		if len(merged) > 0 && span.start < merged[len(merged)-1].end {
			merged[len(merged)-1].end = max(merged[len(merged)-1].end, span.end)
		} else {
			merged = append(merged, span)
		}
	}
	values := map[string]string{}
	var output strings.Builder
	cursor := start
	for index, span := range merged {
		output.Write(source[cursor:span.start])
		key := fmt.Sprintf("⟪MPRESS_INLINE_%d⟫", index)
		values[key] = string(source[span.start:span.end])
		output.WriteString(key)
		cursor = span.end
	}
	output.Write(source[cursor:end])
	return output.String(), values
}

var markdownDirective = regexp.MustCompile(`^[ \t]*(?:@|:::)([A-Za-z][A-Za-z0-9-]*)(?:[ \t{\[].*)?$`)
var markdownAttribute = regexp.MustCompile(`([\w-]+)=("(?:[^"\\]|\\.)*"|'[^']*'|[^\s}|]+)`)
var markdownCompactLabel = regexp.MustCompile(`^\s*@(note|info|tip|warning|warn|caution|danger|important|bug|example|button)\[([^]]*)\]`)

// Mask declarations without changing offsets, so Goldmark sees only their
// body prose. Special component bodies share the native extraction rules.
func markdownDirectiveView(source []byte, start int) ([]byte, []Segment) {
	view := append([]byte(nil), source...)
	mask := func(start, end int) {
		for i := start; i < end; i++ {
			if view[i] != '\n' && view[i] != '\r' {
				view[i] = ' '
			}
		}
	}
	var segments []Segment
	var stack []string
	ordinal := 0
	structuredLine := 0
	opaqueStart, opaqueOrdinal := 0, 0
	fence, fenceLength := byte(0), 0
	for position := start; position < len(source); {
		end := position
		for end < len(source) && source[end] != '\n' {
			end++
		}
		line := source[position:end]
		trimmed := strings.TrimSpace(string(line))
		opaque := ""
		if len(stack) > 0 {
			opaque = stack[len(stack)-1]
		}
		isOpaque := opaque == "terminal" || opaque == "diff" || opaque == "comment" || opaque == "filetree" || opaque == "explained"
		marker, length := mpdFenceRun(trimmed)
		if fence != 0 {
			if marker == fence && length >= fenceLength && strings.TrimSpace(trimmed[length:]) == "" {
				fence = 0
			}
			if isOpaque {
				mask(position, end)
			}
			position = end + 1
			continue
		}
		if length >= 3 {
			fence, fenceLength = marker, length
			if isOpaque {
				mask(position, end)
			}
			position = end + 1
			continue
		}
		if trimmed == "@end" || trimmed == ":::" {
			if isOpaque && (opaque == "filetree" || opaque == "explained") {
				doc := &mpd.Document{Source: source}
				node := &mpd.Node{}
				node.Content.Start, node.Content.End = uint32(opaqueStart), uint32(position)
				if opaque == "filetree" {
					segments = append(segments, mpdFiletreeSegments(doc, node, opaqueOrdinal, "")...)
				} else {
					segments = append(segments, mpdExplainedSegments(doc, node, opaqueOrdinal, "")...)
				}
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			mask(position, end)
			position = end + 1
			continue
		}
		if isOpaque {
			mask(position, end)
			position = end + 1
			continue
		}
		if (opaque == "tabs" || opaque == "preview-tabs" || opaque == "file-tabs") && strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			a := position + bytes.IndexByte(line, '[') + 1
			b := position + bytes.LastIndexByte(line, ']')
			raw := string(source[a:b])
			protected, placeholders := protect(raw)
			segments = append(segments, Segment{ID: fmt.Sprintf("tab%04d-label-%03d", ordinal, len(segments)+1), Kind: "attribute", Original: raw, Text: protected, Start: a, End: b, SourceHash: Hash(raw), Placeholders: placeholders})
			mask(position, end)
			position = end + 1
			continue
		}
		if extracted, handled := markdownComponentLine(opaque, line, position, ordinal, structuredLine+1); handled {
			structuredLine++
			segments = append(segments, extracted...)
			mask(position, end)
			position = end + 1
			continue
		}
		match := markdownDirective.FindSubmatchIndex(line)
		if match != nil {
			name := string(line[match[2]:match[3]])
			// A bare @mention is prose. Known components or declarations with
			// explicit attributes/labels are directives.
			known := strings.Contains(" actions api api-playground audience badge button calendar callout capabilities capability card cards changelog column columns comment container details diff docs-preview event explained filetree footnote form headline if lesson matrix note plan preview-tabs file-tabs pricing rawHTML release resource resources section status step steps tab table tabs terminal testimonial testimonials timeline tutorial variant computed hr image import include input link linkcard qr video ", " "+name+" ")
			if known || bytes.ContainsAny(line, "{[") || strings.HasPrefix(trimmed, ":::") {
				ordinal++
				segments = append(segments, markdownComponentAttributes(line, position, ordinal, name)...)
				leaf := strings.Contains(" computed hr image import include input link linkcard qr video ", " "+name+" ") || name == "button" && bytes.Contains(line, []byte("]("))
				if !leaf {
					stack = append(stack, name)
					opaqueStart, opaqueOrdinal = end+1, ordinal
				}
				mask(position, end)
			}
		}
		position = end + 1
	}
	return view, segments
}

func markdownComponentAttributes(line []byte, base, ordinal int, name string) []Segment {
	var segments []Segment
	allowed := componentTranslationAttributes(name)
	for _, attr := range markdownAttribute.FindAllSubmatchIndex(line, -1) {
		key := string(line[attr[2]:attr[3]])
		if !allowed[key] {
			continue
		}
		a, b := attr[4], attr[5]
		raw := string(line[a:b])
		encoding := ""
		if strings.HasPrefix(raw, "\"") {
			var decoded string
			if json.Unmarshal([]byte(raw), &decoded) != nil {
				continue
			}
			raw = decoded
			a++
			b--
			encoding = "json-string"
		} else if strings.HasPrefix(raw, "'") {
			raw = raw[1 : len(raw)-1]
			encoding = "yaml-string"
		} else {
			encoding = "yaml-string"
		}
		if !translatableText(raw) {
			continue
		}
		protected, placeholders := protect(raw)
		segments = append(segments, Segment{ID: fmt.Sprintf("a%04d-%s-01", ordinal, key), Kind: "attribute", Original: raw, Text: protected, Start: base + a, End: base + b, SourceHash: Hash(raw), Placeholders: placeholders, Encoding: encoding})
	}
	if label := markdownCompactLabel.FindSubmatchIndex(line); label != nil {
		a, b := base+label[4], base+label[5]
		raw := string(line[a-base : b-base])
		protected, placeholders := protect(raw)
		segments = append(segments, Segment{ID: fmt.Sprintf("a%04d-label-01", ordinal), Kind: "attribute", Original: raw, Text: protected, Start: a, End: b, SourceHash: Hash(raw), Placeholders: placeholders})
	}
	return segments
}
