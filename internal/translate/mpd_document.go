package translate

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/mpd"
)

var mpdTranslatableMetadata = map[string]bool{
	"title": true, "description": true, "tagline": true, "content": true, "text": true,
}

var mpdTranslatableAttributes = map[string]bool{
	"alt": true, "aria-label": true, "breadcrumb": true, "description": true,
	"eyebrow": true, "label": true, "name": true, "output": true,
	"service": true, "title": true,
}

var mpdNonProseComponents = map[string]bool{
	"diff": true, "rawHTML": true, "terminal": true,
}

// ExtractMPD uses the native MPD syntax tree. Component declarations, code,
// destinations and identifiers never become provider input, while visible
// component attributes such as step titles and tab labels do.
func ExtractMPD(filename string, source []byte) (*Document, error) {
	parsed := mpd.Parse(filename, source)
	for _, diagnostic := range parsed.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			return nil, fmt.Errorf("parse MPD %s:%d:%d: %s", filename, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Message)
		}
	}
	document := &Document{Source: append([]byte(nil), source...), Format: "mpd"}
	type heading struct {
		level int
		text  string
	}
	var headings []heading
	blockNumber := 0
	attributeNumber := 0
	htmlNumber := 0
	diagramNumber := 0

	var visit func(uint32, string) error
	visit = func(parent uint32, section string) error {
		iterator := parsed.Children(parent)
		for {
			index, node, ok := iterator.Next()
			if !ok {
				break
			}
			switch node.Kind {
			case mpd.KindMetadata:
				appendMPDMetadata(document, parsed, node)
			case mpd.KindComponent:
				attributeNumber++
				name := string(parsed.Text(node.Name))
				document.Segments = append(document.Segments, mpdAttributeSegments(parsed, node, "a", attributeNumber, mpdTranslatableAttributes)...)
				if name == "filetree" {
					blockNumber++
					document.Segments = append(document.Segments, mpdFiletreeSegments(parsed, node, blockNumber, section)...)
					continue
				}
				if name == "explained" {
					document.Segments = append(document.Segments, mpdExplainedSegments(parsed, node, attributeNumber, section)...)
					continue
				}
				if !mpdNonProseComponents[name] {
					if err := visit(index, section); err != nil {
						return err
					}
				}
			case mpd.KindHeading, mpd.KindParagraph, mpd.KindTableCell:
				blockNumber++
				kind := "text"
				if node.Kind == mpd.KindHeading {
					kind = "heading"
				}
				if node.Kind == mpd.KindTableCell {
					kind = "cell"
				}
				segments := mpdInlineSegments(parsed, index, blockNumber, kind, section)
				document.Segments = append(document.Segments, segments...)
				if node.Kind == mpd.KindHeading {
					value := strings.TrimSpace(mpdPlainInlineText(parsed, index))
					if value != "" {
						level := int(node.Level)
						for len(headings) >= level {
							headings = headings[:len(headings)-1]
						}
						headings = append(headings, heading{level: level, text: value})
						document.Outline = append(document.Outline, value)
						if level == 1 && document.Title == "" {
							document.Title = value
						}
						parts := make([]string, 0, len(headings))
						for _, item := range headings {
							parts = append(parts, item.text)
						}
						section = strings.Join(parts, " > ")
					}
				}
			case mpd.KindRawHTML:
				htmlNumber++
				document.Segments = append(document.Segments, htmlSegments(parsed.Text(node.Content), int(node.Content.Start), fmt.Sprintf("h%04d", htmlNumber), section)...)
			case mpd.KindCode:
				if info := strings.Fields(string(parsed.Text(node.Name))); len(info) > 0 && info[0] == "d2" {
					diagramNumber++
					segments, err := d2Segments(parsed.Text(node.Content), int(node.Content.Start), int(node.Content.End), diagramNumber, section)
					if err != nil {
						return err
					}
					document.Segments = append(document.Segments, segments...)
				}
			case mpd.KindComment:
				continue
			default:
				if err := visit(index, section); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(parsed.Root, ""); err != nil {
		return nil, err
	}
	sort.SliceStable(document.Segments, func(i, j int) bool { return document.Segments[i].Start < document.Segments[j].Start })
	return document, nil
}

func appendMPDMetadata(document *Document, parsed *mpd.Document, node *mpd.Node) {
	document.Segments = append(document.Segments, mpdAttributeSegments(parsed, node, "fm", 0, mpdTranslatableMetadata)...)
	for offset := 0; offset < int(node.AttrCount); offset++ {
		attribute := parsed.Attributes[int(node.FirstAttr)+offset]
		if string(parsed.Text(attribute.Name)) != "translationKey" {
			continue
		}
		var value string
		if json.Unmarshal(parsed.Text(attribute.Value), &value) == nil {
			document.TranslationKey = strings.TrimSpace(value)
		}
	}
}

func mpdPlainInlineText(document *mpd.Document, parent uint32) string {
	var output strings.Builder
	var walk func(uint32)
	walk = func(index uint32) {
		iterator := document.Children(index)
		for {
			childIndex, node, ok := iterator.Next()
			if !ok {
				return
			}
			switch node.Kind {
			case mpd.KindText, mpd.KindCodeSpan, mpd.KindAutomaticLink:
				output.Write(document.Text(node.Content))
			case mpd.KindSoftBreak, mpd.KindHardBreak:
				output.WriteByte(' ')
			case mpd.KindEmoji:
				output.WriteByte(':')
				output.Write(document.Text(node.Name))
				output.WriteByte(':')
			case mpd.KindFootnoteReference, mpd.KindMetadataReference:
				output.Write(document.Text(node.Source))
			default:
				walk(childIndex)
			}
		}
	}
	walk(parent)
	return output.String()
}

func mpdInlineSegments(document *mpd.Document, parent uint32, block int, kind, section string) []Segment {
	start, end, ok := mpdInlineBounds(document, parent)
	if !ok {
		return nil
	}
	raw := string(document.Source[start:end])
	if !translatableText(raw) {
		return nil
	}
	protected, placeholders := protectMPDInline(document, parent, start, end)
	return []Segment{{
		ID: blockSegmentID(block, kind, 1), Kind: kind, Section: section,
		Original: raw, Text: protected, Start: start, End: end,
		SourceHash: Hash(raw), Placeholders: placeholders,
	}}
}

type mpdProtectedRange struct{ start, end int }

func mpdInlineBounds(document *mpd.Document, parent uint32) (int, int, bool) {
	iterator := document.Children(parent)
	start, end := len(document.Source), 0
	for {
		_, node, ok := iterator.Next()
		if !ok {
			break
		}
		start = min(start, int(node.Source.Start))
		end = max(end, int(node.Source.End))
	}
	return start, end, start < end
}

func protectMPDInline(document *mpd.Document, parent uint32, start, end int) (string, map[string]string) {
	var ranges []mpdProtectedRange
	add := func(rangeStart, rangeEnd int) {
		rangeStart, rangeEnd = max(start, rangeStart), min(end, rangeEnd)
		if rangeStart < rangeEnd {
			ranges = append(ranges, mpdProtectedRange{rangeStart, rangeEnd})
		}
	}
	var walk func(uint32)
	walk = func(index uint32) {
		iterator := document.Children(index)
		for {
			childIndex, node, ok := iterator.Next()
			if !ok {
				return
			}
			switch node.Kind {
			case mpd.KindCodeSpan, mpd.KindAutomaticLink, mpd.KindEmoji, mpd.KindMetadataReference, mpd.KindFootnoteReference:
				add(int(node.Source.Start), int(node.Source.End))
			case mpd.KindEmphasis, mpd.KindStrong:
				add(int(node.Source.Start), int(node.Content.Start))
				add(int(node.Content.End), int(node.Source.End))
				walk(childIndex)
			case mpd.KindLink, mpd.KindImage:
				labelStart, labelEnd, hasLabel := mpdInlineBounds(document, childIndex)
				if !hasLabel {
					add(int(node.Source.Start), int(node.Source.End))
					continue
				}
				add(int(node.Source.Start), labelStart)
				add(labelEnd, int(node.Source.End))
				walk(childIndex)
			default:
				walk(childIndex)
			}
		}
	}
	walk(parent)
	for _, match := range protectedPattern.FindAllIndex(document.Source[start:end], -1) {
		add(start+match[0], start+match[1])
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].start == ranges[j].start {
			return ranges[i].end < ranges[j].end
		}
		return ranges[i].start < ranges[j].start
	})
	merged := ranges[:0]
	for _, item := range ranges {
		if len(merged) > 0 && item.start <= merged[len(merged)-1].end {
			merged[len(merged)-1].end = max(merged[len(merged)-1].end, item.end)
			continue
		}
		merged = append(merged, item)
	}
	if len(merged) == 0 {
		return string(document.Source[start:end]), nil
	}
	values := make(map[string]string, len(merged))
	var output strings.Builder
	cursor := start
	for index, item := range merged {
		output.Write(document.Source[cursor:item.start])
		key := fmt.Sprintf("⟪MPRESS_INLINE_%d⟫", index)
		values[key] = string(document.Source[item.start:item.end])
		output.WriteString(key)
		cursor = item.end
	}
	output.Write(document.Source[cursor:end])
	return output.String(), values
}

var mpdFiletreeDescriptionPattern = regexp.MustCompile(`^(?:[ \t]*)(?:-[ \t]+)?\S.*?(?:[ \t]{2,}|[ \t]+#[ \t]*)(\S.*?)[ \t]*$`)
var mpdExplainedAnnotationPattern = regexp.MustCompile(`^[ \t]*\([0-9]+\)[ \t]+(\S.*?)[ \t]*$`)

func mpdFiletreeSegments(document *mpd.Document, node *mpd.Node, block int, section string) []Segment {
	start, end := int(node.Content.Start), int(node.Content.End)
	var result []Segment
	lineStart := start
	for lineStart < end {
		lineEnd := lineStart
		for lineEnd < end && document.Source[lineEnd] != '\n' && document.Source[lineEnd] != '\r' {
			lineEnd++
		}
		line := document.Source[lineStart:lineEnd]
		if match := mpdFiletreeDescriptionPattern.FindSubmatchIndex(line); len(match) >= 4 {
			descriptionStart, descriptionEnd := lineStart+match[2], lineStart+match[3]
			if document.Source[descriptionStart] == '#' {
				descriptionStart++
				for descriptionStart < descriptionEnd && (document.Source[descriptionStart] == ' ' || document.Source[descriptionStart] == '\t') {
					descriptionStart++
				}
			}
			raw := string(document.Source[descriptionStart:descriptionEnd])
			if translatableText(raw) {
				protected, placeholders := protect(raw)
				result = append(result, Segment{
					ID: blockSegmentID(block, "filetree", len(result)+1), Kind: "filetree", Section: section,
					Original: raw, Text: protected, Start: descriptionStart, End: descriptionEnd,
					SourceHash: Hash(raw), Placeholders: placeholders,
				})
			}
		}
		lineStart = lineEnd + 1
		if lineEnd < end && document.Source[lineEnd] == '\r' && lineStart < end && document.Source[lineStart] == '\n' {
			lineStart++
		}
	}
	return result
}

func mpdExplainedSegments(document *mpd.Document, node *mpd.Node, component int, section string) []Segment {
	start, end := int(node.Content.Start), int(node.Content.End)
	var result []Segment
	lineStart := start
	fenceMarker := byte(0)
	fenceLength := 0
	for lineStart < end {
		lineEnd := lineStart
		for lineEnd < end && document.Source[lineEnd] != '\n' && document.Source[lineEnd] != '\r' {
			lineEnd++
		}
		line := document.Source[lineStart:lineEnd]
		trimmed := strings.TrimLeft(string(line), " \t")
		if marker, length := mpdFenceRun(trimmed); length >= 3 {
			if fenceMarker == 0 {
				fenceMarker, fenceLength = marker, length
			} else if marker == fenceMarker && length >= fenceLength {
				fenceMarker, fenceLength = 0, 0
			}
		} else if fenceMarker == 0 {
			if match := mpdExplainedAnnotationPattern.FindSubmatchIndex(line); len(match) >= 4 {
				descriptionStart, descriptionEnd := lineStart+match[2], lineStart+match[3]
				nextStart := mpdNextLineStart(document.Source, lineEnd, end)
				for nextStart < end {
					nextEnd := nextStart
					for nextEnd < end && document.Source[nextEnd] != '\n' && document.Source[nextEnd] != '\r' {
						nextEnd++
					}
					nextLine := document.Source[nextStart:nextEnd]
					if strings.TrimSpace(string(nextLine)) == "" || mpdExplainedAnnotationPattern.Match(nextLine) {
						break
					}
					descriptionEnd = nextEnd
					nextStart = mpdNextLineStart(document.Source, nextEnd, end)
				}
				raw := string(document.Source[descriptionStart:descriptionEnd])
				if translatableText(raw) {
					protected, placeholders := protect(raw)
					result = append(result, Segment{
						ID: fmt.Sprintf("e%04d-note-%02d", component, len(result)+1), Kind: "explained", Section: section,
						Original: raw, Text: protected, Start: descriptionStart, End: descriptionEnd,
						SourceHash: Hash(raw), Placeholders: placeholders,
					})
				}
				lineStart = nextStart
				continue
			}
		}
		lineStart = lineEnd + 1
		if lineEnd < end && document.Source[lineEnd] == '\r' && lineStart < end && document.Source[lineStart] == '\n' {
			lineStart++
		}
	}
	return result
}

func mpdNextLineStart(source []byte, lineEnd, end int) int {
	next := lineEnd
	if next < end && source[next] == '\r' {
		next++
		if next < end && source[next] == '\n' {
			next++
		}
	} else if next < end && source[next] == '\n' {
		next++
	}
	return next
}

func mpdFenceRun(line string) (byte, int) {
	if line == "" || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	marker := line[0]
	length := 0
	for length < len(line) && line[length] == marker {
		length++
	}
	return marker, length
}

func mpdAttributeSegments(document *mpd.Document, node *mpd.Node, prefix string, ordinal int, allowed map[string]bool) []Segment {
	var result []Segment
	counts := map[string]int{}
	for offset := 0; offset < int(node.AttrCount); offset++ {
		attribute := document.Attributes[int(node.FirstAttr)+offset]
		name := string(document.Text(attribute.Name))
		if attribute.Flag {
			continue
		}
		var value string
		if json.Unmarshal(document.Text(attribute.Value), &value) != nil {
			if prefix == "fm" && (name == "hero" || name == "banner") {
				result = append(result, nestedMetadataSegments(document.Text(attribute.Value), int(attribute.Value.Start), fmt.Sprintf("%s%04d-%s", prefix, ordinal, name))...)
			}
			continue
		}
		if !allowed[name] || !translatableText(value) {
			continue
		}
		counts[name]++
		id := fmt.Sprintf("%s-%s-%02d", prefix, name, counts[name])
		if ordinal > 0 {
			id = fmt.Sprintf("%s%04d-%s-%02d", prefix, ordinal, name, counts[name])
		}
		protected, placeholders := protect(value)
		result = append(result, Segment{ID: id, Kind: "attribute", Original: value, Text: protected, Start: int(attribute.Value.Start) + 1, End: int(attribute.Value.End) - 1, SourceHash: Hash(value), Placeholders: placeholders, Encoding: "json-string"})
	}
	return result
}

func encodeSegment(segment Segment, value string) (string, error) {
	if segment.Encoding == "html-unquoted" {
		return `"` + escapeHTMLTranslation(value) + `"`, nil
	}
	if segment.Encoding == "html-text" {
		return escapeHTMLTranslation(value), nil
	}
	if segment.Encoding == "yaml-string" {
		encoded, err := json.Marshal(value)
		return string(encoded), err
	}
	if segment.Encoding != "json-string" {
		return value, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode segment %s: %w", segment.ID, err)
	}
	return string(encoded[1 : len(encoded)-1]), nil
}

func translationRenderable(filename string, source []byte) ([]byte, error) {
	if !strings.EqualFold(filepath.Ext(filename), ".mpd") {
		return source, nil
	}
	document := mpd.Parse(filename, source)
	return mpd.Markdown(document)
}
