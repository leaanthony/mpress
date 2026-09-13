package mpd

import (
	"bytes"
	"encoding/binary"
	"unicode/utf8"
)

const (
	defaultMaxNesting    = 256
	defaultMaxAttributes = 256
	defaultMaxFileSize   = 256 << 20
)

// Options sets defensive parser limits. A zero value uses the defaults.
type Options struct {
	MaxNesting    int
	MaxAttributes int
	MaxFileSize   int
}

// Parse parses one immutable source buffer with the default limits.
func Parse(filename string, source []byte) *Document {
	return ParseWithOptions(filename, source, Options{})
}

// ParseWithOptions parses one immutable source buffer. It always returns a
// document, including for malformed input.
func ParseWithOptions(filename string, source []byte, options Options) *Document {
	if options.MaxNesting <= 0 {
		options.MaxNesting = defaultMaxNesting
	}
	if options.MaxAttributes <= 0 {
		options.MaxAttributes = defaultMaxAttributes
	}
	if options.MaxFileSize <= 0 {
		options.MaxFileSize = defaultMaxFileSize
	}
	doc := &Document{
		Filename:   filename,
		Source:     source,
		Schema:     1,
		Nodes:      make([]Node, 0, len(source)/14+4),
		Attributes: make([]Attribute, 0, len(source)/64+4),
	}
	ascii := sourceIsASCII(source)
	p := parser{doc: doc, source: source, options: options, line: 1, ascii: ascii, hasCR: bytes.IndexByte(source, '\r') >= 0}
	doc.Root = p.addNode(KindDocument, Range{End: uint32(len(source))}, Position{Line: 1, Column: 1})
	if len(source) > options.MaxFileSize {
		p.diagnostic(doc.Root, "mpd-limit", SeverityError, "document exceeds the configured file-size limit", Range{End: uint32(len(source))}, "split the document or increase MaxFileSize")
		return doc
	}
	if !ascii && !utf8.Valid(source) {
		p.reportInvalidUTF8()
	}
	if len(source) >= 3 && source[0] == 0xef && source[1] == 0xbb && source[2] == 0xbf {
		p.offset = 3
	}
	if !p.parseMetadata() {
		return doc
	}
	p.parseBlocks(doc.Root, false, 0)
	p.validateReferences()
	return doc
}

type parser struct {
	doc               *Document
	source            []byte
	options           Options
	lineStarts        []uint32
	ascii             bool
	hasCR             bool
	offset            int
	line              uint32
	cachedLine        sourceLine
	hasLine           bool
	references        [4]referenceRecord
	referenceCount    int
	referenceOverflow []referenceRecord
}

type sourceLine struct {
	start int
	end   int
	next  int
	line  uint32
}

type quoteStack struct {
	inline   [8]uint32
	overflow []uint32
	count    int
}

func (s *quoteStack) push(node uint32) {
	if s.count < len(s.inline) {
		s.inline[s.count] = node
	} else {
		s.overflow = append(s.overflow, node)
	}
	s.count++
}

func (s *quoteStack) pop() uint32 {
	s.count--
	return s.at(s.count)
}

func (s *quoteStack) at(index int) uint32 {
	if index < len(s.inline) {
		return s.inline[index]
	}
	return s.overflow[index-len(s.inline)]
}

func (s *quoteStack) last() uint32 { return s.at(s.count - 1) }

func (p *parser) currentLine() sourceLine {
	if p.hasLine && p.cachedLine.start == p.offset {
		return p.cachedLine
	}
	p.cachedLine = lineAt(p.source, p.offset, p.line, p.hasCR)
	p.hasLine = true
	return p.cachedLine
}

func lineAt(source []byte, offset int, number uint32, hasCR bool) sourceLine {
	line := sourceLine{start: offset, end: len(source), next: len(source), line: number}
	rest := source[offset:]
	lineFeed := bytes.IndexByte(rest, '\n')
	beforeLineFeed := rest
	if lineFeed >= 0 {
		beforeLineFeed = rest[:lineFeed]
	}
	carriageReturn := -1
	if hasCR {
		carriageReturn = bytes.IndexByte(beforeLineFeed, '\r')
	}
	ending := carriageReturn
	if ending < 0 {
		ending = lineFeed
	}
	if ending >= 0 {
		line.end = offset + ending
		line.next = line.end + 1
		if rest[ending] == '\r' && line.next < len(source) && source[line.next] == '\n' {
			line.next++
		}
	}
	return line
}

func (p *parser) advance(line sourceLine) {
	p.offset = line.next
	p.hasLine = false
	if line.next > line.end {
		p.line++
	}
}

func (p *parser) addNode(kind Kind, source Range, position Position) uint32 {
	index := uint32(len(p.doc.Nodes))
	p.doc.Nodes = append(p.doc.Nodes, Node{
		Kind: kind, Source: source, Position: position,
		FirstChild: noIndex, LastChild: noIndex, NextSibling: noIndex,
	})
	return index
}

func (p *parser) addChild(parent, child uint32) {
	last := p.doc.Nodes[parent].LastChild
	if last == noIndex {
		p.doc.Nodes[parent].FirstChild = child
	} else {
		p.doc.Nodes[last].NextSibling = child
	}
	p.doc.Nodes[parent].LastChild = child
}

func (p *parser) diagnostic(node uint32, code string, severity Severity, message string, source Range, suggestion string) {
	p.diagnosticAt(node, code, severity, message, source, suggestion, p.positionAt(int(source.Start)))
}

func (p *parser) diagnosticAt(node uint32, code string, severity Severity, message string, source Range, suggestion string, position Position) {
	diagnostic := Diagnostic{Node: node, Code: code, Severity: severity, Message: message, Suggestion: suggestion, Source: source, Position: position}
	p.doc.Diagnostics = append(p.doc.Diagnostics, diagnostic)
}

func (p *parser) positionAt(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(p.source) {
		offset = len(p.source)
	}
	if len(p.lineStarts) == 0 {
		p.lineStarts = buildLineStarts(p.source)
	}
	low, high := 0, len(p.lineStarts)
	for low+1 < high {
		middle := int(uint(low+high) >> 1)
		if int(p.lineStarts[middle]) <= offset {
			low = middle
		} else {
			high = middle
		}
	}
	start := int(p.lineStarts[low])
	if offset < start {
		return Position{Line: 1, Column: 1}
	}
	column := uint32(utf8.RuneCount(p.source[start:offset]) + 1)
	return Position{Line: uint32(low + 1), Column: column}
}

func buildLineStarts(source []byte) []uint32 {
	start := 0
	if len(source) >= 3 && source[0] == 0xef && source[1] == 0xbb && source[2] == 0xbf {
		start = 3
	}
	lines := make([]uint32, 1, len(source)/28+2)
	lines[0] = uint32(start)
	for index := start; index < len(source); index++ {
		switch source[index] {
		case '\n':
			lines = append(lines, uint32(index+1))
		case '\r':
			if index+1 < len(source) && source[index+1] == '\n' {
				index++
			}
			lines = append(lines, uint32(index+1))
		}
	}
	return lines
}

func sourceIsASCII(source []byte) bool {
	index := 0
	for ; index+8 <= len(source); index += 8 {
		if binary.LittleEndian.Uint64(source[index:])&0x8080808080808080 != 0 {
			return false
		}
	}
	for ; index < len(source); index++ {
		if source[index]&0x80 != 0 {
			return false
		}
	}
	return true
}

func (p *parser) reportInvalidUTF8() {
	offset := 0
	if len(p.source) >= 3 && p.source[0] == 0xef && p.source[1] == 0xbb && p.source[2] == 0xbf {
		offset = 3
	}
	position := Position{Line: 1, Column: 1}
	for offset < len(p.source) {
		if p.source[offset] == '\r' {
			offset++
			if offset < len(p.source) && p.source[offset] == '\n' {
				offset++
			}
			position.Line++
			position.Column = 1
			continue
		}
		if p.source[offset] == '\n' {
			offset++
			position.Line++
			position.Column = 1
			continue
		}
		r, size := utf8.DecodeRune(p.source[offset:])
		if r == utf8.RuneError && size == 1 {
			source := Range{Start: uint32(offset), End: uint32(offset + 1)}
			p.diagnosticAt(p.doc.Root, "mpd-utf8", SeverityError, "invalid UTF-8 byte", source, "save the document as UTF-8", position)
			offset++
			position.Column++
			continue
		}
		offset += size
		position.Column++
	}
}

func (p *parser) parseMetadata() bool {
	if p.offset >= len(p.source) {
		return true
	}
	first := p.currentLine()
	if !bytes.Equal(p.source[first.start:first.end], []byte("---")) {
		return true
	}
	metadata := p.addNode(KindMetadata, Range{Start: uint32(first.start)}, Position{Line: first.line, Column: 1})
	p.doc.Nodes[metadata].FirstAttr = uint32(len(p.doc.Attributes))
	p.addChild(p.doc.Root, metadata)
	p.advance(first)
	firstField := true
	metadataLimitReported := false
	for p.offset < len(p.source) {
		line := p.currentLine()
		body := p.source[line.start:line.end]
		if bytes.Equal(body, []byte("---")) {
			p.advance(line)
			p.doc.Nodes[metadata].Source.End = uint32(line.next)
			if firstField {
				p.diagnostic(metadata, "mpd-metadata", SeverityError, "metadata must start with schema", Range{Start: uint32(first.start), End: uint32(line.next)}, "add schema = 1 as the first field")
			}
			return p.doc.Schema == 1
		}
		nameStart, nameEnd, valueStart, ok := parseMetadataField(body)
		if !ok {
			p.diagnostic(metadata, "mpd-metadata", SeverityError, "invalid metadata field", Range{Start: uint32(line.start), End: uint32(line.end)}, "use key = JSON-value")
			p.advance(line)
			firstField = false
			continue
		}
		absoluteName := Range{Start: uint32(line.start + nameStart), End: uint32(line.start + nameEnd)}
		absoluteValue := Range{Start: uint32(line.start + valueStart), End: uint32(line.end)}
		nameBytes := body[nameStart:nameEnd]
		if p.hasAttributeName(metadata, nameBytes) {
			p.diagnostic(metadata, "mpd-metadata", SeverityError, "duplicate metadata key", absoluteName, "remove the duplicate field")
		}
		if end, valid := scanJSONValue(body, valueStart); !valid || end != len(body) {
			p.diagnostic(metadata, "mpd-metadata", SeverityError, "metadata value is not valid single-line JSON", absoluteValue, "use a JSON string, number, Boolean, null, array, or object")
		}
		if firstField {
			if !bytes.Equal(nameBytes, []byte("schema")) {
				p.diagnostic(metadata, "mpd-metadata", SeverityError, "schema must be the first metadata field", absoluteName, "move schema = 1 to the first field")
			} else if schema, ok := parsePositiveUint(body[valueStart:]); !ok {
				p.diagnostic(metadata, "mpd-version", SeverityError, "schema must be a positive integer", absoluteValue, "use schema = 1")
				return false
			} else {
				p.doc.Schema = schema
				if schema != 1 {
					p.diagnostic(metadata, "mpd-version", SeverityError, "unsupported MPress Document schema", absoluteValue, "use a parser that supports this schema")
					return false
				}
			}
		}
		if int(p.doc.Nodes[metadata].AttrCount) >= p.options.MaxAttributes {
			if !metadataLimitReported {
				p.diagnostic(metadata, "mpd-limit", SeverityError, "metadata exceeds the configured field limit", absoluteName, "remove fields or increase MaxAttributes")
				metadataLimitReported = true
			}
			p.advance(line)
			firstField = false
			continue
		}
		p.doc.Attributes = append(p.doc.Attributes, Attribute{Name: absoluteName, Value: absoluteValue, Position: Position{Line: line.line, Column: uint32(nameStart + 1)}, Metadata: true})
		p.doc.Nodes[metadata].AttrCount++
		p.advance(line)
		firstField = false
	}
	p.doc.Nodes[metadata].Source.End = uint32(len(p.source))
	p.doc.Nodes[metadata].Flags |= FlagUnclosed
	p.diagnostic(metadata, "mpd-metadata", SeverityError, "unterminated metadata block", Range{Start: uint32(first.start), End: uint32(len(p.source))}, "add a closing --- line")
	return false
}

func parseMetadataField(line []byte) (nameStart, nameEnd, valueStart int, ok bool) {
	nameStart = skipSpaces(line, 0)
	nameEnd = scanName(line, nameStart)
	if nameEnd == nameStart {
		return 0, 0, 0, false
	}
	i := skipSpaces(line, nameEnd)
	if i >= len(line) || line[i] != '=' {
		return 0, 0, 0, false
	}
	valueStart = skipSpaces(line, i+1)
	if valueStart >= len(line) || trimSpaceEnd(line, len(line)) != len(line) {
		return 0, 0, 0, false
	}
	return nameStart, nameEnd, valueStart, true
}

func (p *parser) hasAttributeName(node uint32, name []byte) bool {
	owner := &p.doc.Nodes[node]
	start := int(owner.FirstAttr)
	for index := 0; index < int(owner.AttrCount); index++ {
		attribute := &p.doc.Attributes[start+index]
		if bytes.Equal(p.doc.Text(attribute.Name), name) {
			return true
		}
	}
	return false
}

func parsePositiveUint(source []byte) (uint32, bool) {
	if len(source) == 0 {
		return 0, false
	}
	var value uint32
	for _, char := range source {
		if char < '0' || char > '9' {
			return 0, false
		}
		digit := uint32(char - '0')
		if value > (^uint32(0)-digit)/10 {
			return 0, false
		}
		value = value*10 + digit
	}
	return value, value != 0
}

func (p *parser) parseBlocks(parent uint32, stopAtEnd bool, depth int) bool {
	if depth > p.options.MaxNesting {
		p.diagnostic(parent, "mpd-limit", SeverityError, "component nesting exceeds the configured limit", Range{Start: uint32(p.offset), End: uint32(p.offset)}, "reduce nesting or increase MaxNesting")
		return false
	}
	for p.offset < len(p.source) {
		line := p.currentLine()
		if invalidListIndent(p.source[line.start:line.end]) {
			p.parseParagraph(parent, stopAtEnd)
			node := p.doc.Nodes[parent].LastChild
			p.diagnostic(node, "mpd-indent", SeverityError, "list indentation must use a multiple of two ASCII spaces", Range{Start: uint32(line.start), End: uint32(line.end)}, "replace tabs and align the list marker to a two-space boundary")
			continue
		}
		body, contentStart := trimIndent(p.source[line.start:line.end], line.start)
		if isBlank(body) {
			p.advance(line)
			continue
		}
		if bytes.Equal(body, []byte("@end")) {
			if stopAtEnd {
				p.advance(line)
				p.doc.Nodes[parent].Source.End = uint32(line.next)
				return true
			}
			paragraph := p.addNode(KindParagraph, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(contentStart - line.start + 1)})
			p.addChild(parent, paragraph)
			p.doc.Nodes[paragraph].Content = Range{Start: uint32(contentStart), End: uint32(line.end)}
			p.parseInline(paragraph, contentStart, line.end, Position{Line: line.line, Column: uint32(contentStart - line.start + 1)})
			p.diagnostic(paragraph, "mpd-end", SeverityError, "unexpected @end", Range{Start: uint32(contentStart), End: uint32(line.end)}, "remove this line or add a matching container")
			p.advance(line)
			continue
		}
		if level, start, ok := heading(body, contentStart); ok {
			node := p.addNode(KindHeading, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(contentStart - line.start + 1)})
			p.doc.Nodes[node].Level = uint8(level)
			p.doc.Nodes[node].Content = Range{Start: uint32(start), End: uint32(line.end)}
			p.addChild(parent, node)
			p.parseInline(node, start, line.end, Position{Line: line.line, Column: uint32(start - line.start + 1)})
			p.advance(line)
			continue
		}
		if width := fenceRun(body); width >= 3 {
			p.parseCode(parent, line, contentStart, width)
			continue
		}
		if marker, ok := parseListMarker(body, contentStart); ok {
			p.parseList(parent, marker, depth)
			continue
		}
		if quoteDepth(body) > 0 {
			p.parseQuote(parent)
			continue
		}
		if len(body) > 1 && body[0] == '@' {
			if p.parseDirective(parent, line, body, contentStart, depth) {
				continue
			}
			if nameEnd := scanName(body, 1); nameEnd > 1 {
				p.parseParagraph(parent, stopAtEnd)
				node := p.doc.Nodes[parent].LastChild
				p.diagnostic(node, "mpd-directive", SeverityError, "unknown or unavailable directive", Range{Start: uint32(contentStart), End: uint32(line.end)}, "use a component declared by this document schema")
				continue
			}
		}
		p.parseParagraph(parent, stopAtEnd)
	}
	if stopAtEnd {
		p.doc.Nodes[parent].Flags |= FlagUnclosed
		p.doc.Nodes[parent].Source.End = uint32(len(p.source))
		p.diagnostic(parent, "mpd-unclosed", SeverityError, "unclosed component", p.doc.Nodes[parent].Source, "add @end")
	}
	return false
}

func (p *parser) parseParagraph(parent uint32, stopAtEnd bool) {
	first := p.currentLine()
	_, contentStart := trimIndent(p.source[first.start:first.end], first.start)
	lastEnd, lastNext := first.end, first.next
	p.advance(first)
	for p.offset < len(p.source) {
		line := p.currentLine()
		body, contentStart := trimIndent(p.source[line.start:line.end], line.start)
		if isBlank(body) || (stopAtEnd && bytes.Equal(body, []byte("@end"))) {
			break
		}
		if _, _, ok := heading(body, contentStart); ok {
			break
		}
		if fenceRun(body) >= 3 {
			break
		}
		if _, ok := parseListMarker(body, contentStart); ok {
			break
		}
		if quoteDepth(body) > 0 {
			break
		}
		if len(body) > 1 && body[0] == '@' && componentClassFor(body[1:scanName(body, 1)]) != componentUnknown {
			break
		}
		lastEnd, lastNext = line.end, line.next
		p.advance(line)
	}
	node := p.addNode(KindParagraph, Range{Start: uint32(first.start), End: uint32(lastNext)}, Position{Line: first.line, Column: uint32(contentStart - first.start + 1)})
	p.doc.Nodes[node].Content = Range{Start: uint32(contentStart), End: uint32(lastEnd)}
	p.addChild(parent, node)
	p.parseInline(node, contentStart, lastEnd, Position{Line: first.line, Column: uint32(contentStart - first.start + 1)})
}

func (p *parser) parseCode(parent uint32, opening sourceLine, contentStart, width int) {
	node := p.addNode(KindCode, Range{Start: uint32(opening.start)}, Position{Line: opening.line, Column: uint32(contentStart - opening.start + 1)})
	p.addChild(parent, node)
	opener := p.source[contentStart:opening.end]
	infoStart := width
	for infoStart < len(opener) && opener[infoStart] == ' ' {
		infoStart++
	}
	if infoStart < len(opener) {
		languageEnd := infoStart
		for languageEnd < len(opener) && opener[languageEnd] != ' ' && opener[languageEnd] != '{' {
			languageEnd++
		}
		p.doc.Nodes[node].Name = Range{Start: uint32(contentStart + infoStart), End: uint32(contentStart + languageEnd)}
		attributeStart := skipSpaces(opener, languageEnd)
		if attributeStart < len(opener) {
			if opener[attributeStart] != '{' || opener[len(opener)-1] != '}' {
				p.diagnostic(node, "mpd-attribute", SeverityError, "code-fence attributes must be enclosed in braces", Range{Start: uint32(contentStart + attributeStart), End: uint32(opening.end)}, "use {name=JSON-value}")
			} else {
				p.parseAttributes(node, opener[:len(opener)-1], contentStart, attributeStart+1)
			}
		}
	}
	p.advance(opening)
	bodyStart := p.offset
	for p.offset < len(p.source) {
		line := p.currentLine()
		body := bytes.TrimSpace(p.source[line.start:line.end])
		run := fenceRun(body)
		if run >= width && len(bytes.TrimSpace(body[run:])) == 0 {
			p.doc.Nodes[node].Content = Range{Start: uint32(bodyStart), End: uint32(line.start)}
			p.doc.Nodes[node].Source.End = uint32(line.next)
			p.advance(line)
			return
		}
		p.advance(line)
	}
	p.doc.Nodes[node].Content = Range{Start: uint32(bodyStart), End: uint32(len(p.source))}
	p.doc.Nodes[node].Source.End = uint32(len(p.source))
	p.doc.Nodes[node].Flags |= FlagUnclosed
	p.diagnostic(node, "mpd-unclosed", SeverityError, "unclosed code fence", p.doc.Nodes[node].Source, "add a closing fence at least as long as the opening fence")
}

func (p *parser) parseDirective(parent uint32, line sourceLine, body []byte, absoluteStart, depth int) bool {
	nameEnd := scanName(body, 1)
	if nameEnd == 1 {
		return false
	}
	nameBytes := body[1:nameEnd]
	class := componentClassFor(nameBytes)
	if class == componentUnknown {
		return false
	}
	kind := KindComponent
	switch string(nameBytes) {
	case "hr":
		kind = KindRule
	case "rawHTML":
		kind = KindRawHTML
	case "comment":
		kind = KindComment
	case "table":
		kind = KindTable
	case "footnote":
		kind = KindFootnote
	case "import":
		kind = KindImport
	case "include":
		kind = KindInclude
	}
	node := p.addNode(kind, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(absoluteStart - line.start + 1)})
	p.doc.Nodes[node].Name = Range{Start: uint32(absoluteStart + 1), End: uint32(absoluteStart + nameEnd)}
	p.addChild(parent, node)
	p.parseAttributes(node, body, absoluteStart, nameEnd)
	if kind == KindRawHTML && p.doc.Nodes[node].AttrCount != 0 {
		p.diagnostic(node, "mpd-attribute", SeverityError, "raw HTML does not accept attributes", p.doc.Nodes[node].Source, "remove the attributes")
	}
	if kind == KindFootnote {
		p.recordDefinition(node, referenceFootnote)
	}
	if string(nameBytes) == "link" {
		p.recordDefinition(node, referenceLink)
	}
	p.advance(line)
	if class == componentLeaf {
		return true
	}
	if kind == KindRawHTML || kind == KindComment || opaqueComponentBody(string(nameBytes)) {
		p.parseOpaque(node)
		return true
	}
	if kind == KindTable {
		p.parseTable(node, depth+1)
		return true
	}
	p.parseBlocks(node, true, depth+1)
	return true
}

func opaqueComponentBody(name string) bool {
	switch name {
	case "diff", "explained", "filetree", "terminal":
		return true
	default:
		return false
	}
}

func (p *parser) parseOpaque(node uint32) {
	bodyStart := p.offset
	for p.offset < len(p.source) {
		line := p.currentLine()
		body := bytes.TrimSpace(p.source[line.start:line.end])
		if bytes.Equal(body, []byte("@end")) {
			p.doc.Nodes[node].Content = Range{Start: uint32(bodyStart), End: uint32(line.start)}
			p.doc.Nodes[node].Source.End = uint32(line.next)
			p.advance(line)
			return
		}
		p.advance(line)
	}
	p.doc.Nodes[node].Content = Range{Start: uint32(bodyStart), End: uint32(len(p.source))}
	p.doc.Nodes[node].Source.End = uint32(len(p.source))
	p.doc.Nodes[node].Flags |= FlagUnclosed
	p.diagnostic(node, "mpd-unclosed", SeverityError, "unclosed opaque block", p.doc.Nodes[node].Source, "add @end")
}

func (p *parser) parseAttributes(node uint32, line []byte, absoluteStart, offset int) {
	offset = skipSpaces(line, offset)
	p.doc.Nodes[node].FirstAttr = uint32(len(p.doc.Attributes))
	for offset < len(line) {
		if int(p.doc.Nodes[node].AttrCount) >= p.options.MaxAttributes {
			p.diagnostic(node, "mpd-limit", SeverityError, "component exceeds the configured attribute limit", Range{Start: uint32(absoluteStart + offset), End: uint32(absoluteStart + len(line))}, "remove attributes or increase MaxAttributes")
			return
		}
		nameStart := offset
		nameEnd := scanName(line, nameStart)
		if nameEnd == nameStart {
			p.diagnostic(node, "mpd-attribute", SeverityError, "invalid attribute name", Range{Start: uint32(absoluteStart + offset), End: uint32(absoluteStart + len(line))}, "use name=JSON-value")
			return
		}
		nameBytes := line[nameStart:nameEnd]
		if p.hasAttributeName(node, nameBytes) {
			p.diagnostic(node, "mpd-attribute", SeverityError, "duplicate attribute", Range{Start: uint32(absoluteStart + nameStart), End: uint32(absoluteStart + nameEnd)}, "remove the duplicate attribute")
		}
		attribute := Attribute{Name: Range{Start: uint32(absoluteStart + nameStart), End: uint32(absoluteStart + nameEnd)}, Position: Position{Line: p.doc.Nodes[node].Position.Line, Column: uint32(absoluteStart + nameStart - int(p.doc.Nodes[node].Source.Start) + 1)}}
		offset = nameEnd
		if offset >= len(line) || line[offset] != '=' {
			attribute.Flag = true
			if !isBooleanAttribute(nameBytes) {
				p.diagnostic(node, "mpd-attribute", SeverityError, "shorthand is only valid for a declared Boolean attribute", attribute.Name, "write an explicit JSON value")
			}
		} else {
			valueStart := offset + 1
			valueEnd, ok := scanJSONValue(line, valueStart)
			if !ok {
				p.diagnostic(node, "mpd-attribute", SeverityError, "invalid JSON attribute value", Range{Start: uint32(absoluteStart + valueStart), End: uint32(absoluteStart + len(line))}, "use a single-line JSON value")
				return
			}
			attribute.Value = Range{Start: uint32(absoluteStart + valueStart), End: uint32(absoluteStart + valueEnd)}
			offset = valueEnd
		}
		p.doc.Attributes = append(p.doc.Attributes, attribute)
		p.doc.Nodes[node].AttrCount++
		if offset < len(line) && line[offset] != ' ' {
			p.diagnostic(node, "mpd-attribute", SeverityError, "attributes must be separated by ASCII spaces", Range{Start: uint32(absoluteStart + offset), End: uint32(absoluteStart + len(line))}, "insert a space")
			return
		}
		offset = skipSpaces(line, offset)
	}
}

func (p *parser) parseTable(node uint32, depth int) {
	expectedCells := -1
	for p.offset < len(p.source) {
		line := p.currentLine()
		body, contentStart := trimIndent(p.source[line.start:line.end], line.start)
		if bytes.Equal(body, []byte("@end")) {
			p.doc.Nodes[node].Source.End = uint32(line.next)
			p.advance(line)
			return
		}
		if len(body) >= 2 && body[0] == '|' && body[len(body)-1] == '|' {
			row := p.addNode(KindTableRow, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(contentStart - line.start + 1)})
			p.addChild(node, row)
			cells := p.parseTableCells(row, body, contentStart)
			if expectedCells < 0 {
				expectedCells = cells
			} else if cells != expectedCells {
				p.diagnostic(row, "mpd-table", SeverityError, "table rows must contain the same number of cells", p.doc.Nodes[row].Source, "add or remove cells to match the first row")
			}
			p.advance(line)
			continue
		}
		if isBlank(body) {
			p.advance(line)
			continue
		}
		// Keep malformed table content as a paragraph child and recover at the
		// next line, so no bytes disappear.
		paragraph := p.addNode(KindParagraph, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(contentStart - line.start + 1)})
		p.doc.Nodes[paragraph].Content = Range{Start: uint32(contentStart), End: uint32(line.end)}
		p.addChild(node, paragraph)
		p.parseInline(paragraph, contentStart, line.end, Position{Line: line.line, Column: uint32(contentStart - line.start + 1)})
		p.diagnostic(paragraph, "mpd-table", SeverityError, "table row must begin and end with |", p.doc.Nodes[paragraph].Content, "add leading and trailing pipes")
		p.advance(line)
	}
	p.doc.Nodes[node].Source.End = uint32(len(p.source))
	p.doc.Nodes[node].Flags |= FlagUnclosed
	p.diagnostic(node, "mpd-unclosed", SeverityError, "unclosed table", p.doc.Nodes[node].Source, "add @end")
	_ = depth
}

func (p *parser) parseTableCells(row uint32, body []byte, absoluteStart int) int {
	cellStart := 1
	count := 0
	for i := 1; i < len(body); i++ {
		if body[i] == '\\' {
			i++
			continue
		}
		if body[i] != '|' {
			continue
		}
		start, end := cellStart, i
		for start < end && (body[start] == ' ' || body[start] == '\t') {
			start++
		}
		for end > start && (body[end-1] == ' ' || body[end-1] == '\t') {
			end--
		}
		position := positionAfter(p.source, p.doc.Nodes[row].Position, int(p.doc.Nodes[row].Source.Start), absoluteStart+start, p.ascii)
		cell := p.addNode(KindTableCell, Range{Start: uint32(absoluteStart + cellStart), End: uint32(absoluteStart + i + 1)}, position)
		p.doc.Nodes[cell].Content = Range{Start: uint32(absoluteStart + start), End: uint32(absoluteStart + end)}
		p.addChild(row, cell)
		p.parseInline(cell, absoluteStart+start, absoluteStart+end, position)
		count++
		cellStart = i + 1
	}
	return count
}

type listMarker struct {
	indent, markerEnd, contentStart int
	ordered                         bool
	number                          uint32
	checked                         int8
}

func parseListMarker(line []byte, absoluteStart int) (listMarker, bool) {
	var marker listMarker
	for marker.indent < len(line) && line[marker.indent] == ' ' {
		marker.indent++
	}
	if marker.indent%2 != 0 || marker.indent >= len(line) {
		return marker, false
	}
	i := marker.indent
	if line[i] == '-' && i+1 < len(line) && line[i+1] == ' ' {
		marker.markerEnd, marker.contentStart = i+1, i+2
	} else if line[i] >= '0' && line[i] <= '9' {
		start := i
		for i < len(line) && line[i] >= '0' && line[i] <= '9' {
			i++
		}
		if i+1 >= len(line) || line[i] != '.' || line[i+1] != ' ' {
			return marker, false
		}
		value, ok := parsePositiveUint(line[start:i])
		if !ok {
			return marker, false
		}
		marker.ordered, marker.number, marker.markerEnd, marker.contentStart = true, value, i+1, i+2
	} else {
		return marker, false
	}
	if marker.contentStart+3 < len(line) && line[marker.contentStart] == '[' && line[marker.contentStart+2] == ']' && line[marker.contentStart+3] == ' ' {
		switch line[marker.contentStart+1] {
		case ' ':
			marker.checked = -1
		case 'x', 'X':
			marker.checked = 1
		default:
			return marker, false
		}
		marker.contentStart += 4
	}
	marker.markerEnd += absoluteStart
	marker.contentStart += absoluteStart
	return marker, true
}

func (p *parser) parseList(parent uint32, first listMarker, depth int) {
	opening := p.currentLine()
	physicalBaseIndent := leadingSpaces(p.source[opening.start:opening.end])
	indentPrefix := physicalBaseIndent - first.indent
	if indentPrefix < 0 {
		indentPrefix = 0
	}
	if depth > p.options.MaxNesting {
		paragraph := p.addNode(KindParagraph, Range{Start: uint32(opening.start), End: uint32(opening.next)}, Position{Line: opening.line, Column: uint32(indentPrefix + first.indent + 1)})
		p.doc.Nodes[paragraph].Content = Range{Start: uint32(first.contentStart), End: uint32(opening.end)}
		p.addChild(parent, paragraph)
		p.parseInline(paragraph, first.contentStart, opening.end, Position{Line: opening.line, Column: uint32(first.contentStart - opening.start + 1)})
		p.diagnostic(paragraph, "mpd-limit", SeverityError, "list nesting exceeds the configured limit", p.doc.Nodes[paragraph].Source, "reduce nesting or increase MaxNesting")
		p.advance(opening)
		return
	}
	list := p.addNode(KindList, Range{Start: uint32(opening.start)}, Position{Line: opening.line, Column: uint32(indentPrefix + first.indent + 1)})
	if first.ordered {
		p.doc.Nodes[list].Flags |= FlagOrdered
	}
	p.doc.Nodes[list].Level = uint8(first.indent / 2)
	p.addChild(parent, list)
	baseIndent := first.indent
	for p.offset < len(p.source) {
		line := p.currentLine()
		body := p.source[line.start:line.end]
		marker, ok := parseListMarker(body, line.start)
		if ok && marker.indent >= indentPrefix {
			marker.indent -= indentPrefix
		} else if ok {
			ok = false
		}
		if !ok || marker.indent != baseIndent || marker.ordered != first.ordered {
			break
		}
		item := p.addNode(KindItem, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(indentPrefix + baseIndent + 1)})
		p.doc.Nodes[item].Level = uint8(baseIndent / 2)
		if marker.ordered {
			numberEnd := marker.markerEnd - 1
			numberStart := line.start + indentPrefix + baseIndent
			p.doc.Nodes[item].Name = Range{Start: uint32(numberStart), End: uint32(numberEnd)}
		}
		p.doc.Nodes[item].Content = Range{Start: uint32(marker.contentStart), End: uint32(line.end)}
		if marker.checked < 0 {
			p.doc.Nodes[item].Flags |= FlagUnchecked
		}
		if marker.checked > 0 {
			p.doc.Nodes[item].Flags |= FlagChecked
		}
		p.addChild(list, item)
		if marker.contentStart < line.end {
			position := Position{Line: line.line, Column: uint32(marker.contentStart - line.start + 1)}
			paragraph := p.addNode(KindParagraph, Range{Start: uint32(marker.contentStart), End: uint32(line.next)}, position)
			p.doc.Nodes[paragraph].Content = Range{Start: uint32(marker.contentStart), End: uint32(line.end)}
			p.addChild(item, paragraph)
			p.parseInline(paragraph, marker.contentStart, line.end, position)
		}
		p.advance(line)
		for p.offset < len(p.source) {
			next := p.currentLine()
			nextBody := p.source[next.start:next.end]
			if nextMarker, nested := parseListMarker(nextBody, next.start); nested {
				if nextMarker.indent >= indentPrefix {
					nextMarker.indent -= indentPrefix
				} else {
					nested = false
				}
				if !nested {
					break
				}
				if nextMarker.indent > baseIndent {
					p.parseList(item, nextMarker, depth+1)
					if nextMarker.indent != baseIndent+2 {
						nestedList := p.doc.Nodes[item].LastChild
						p.diagnostic(nestedList, "mpd-indent", SeverityError, "a nested list must be indented exactly two spaces", p.doc.Nodes[nestedList].Source, "align the nested marker two spaces beyond its parent")
					}
					continue
				}
				break
			}
			if isBlank(nextBody) {
				// A blank line remains within the item only when followed by an
				// indented continuation.
				after := lineAt(p.source, next.next, next.line+1, p.hasCR)
				if after.start < len(p.source) && leadingSpaces(p.source[after.start:after.end]) >= indentPrefix+baseIndent+2 {
					p.advance(next)
					continue
				}
				break
			}
			physicalIndent := leadingSpaces(nextBody)
			if physicalIndent < indentPrefix {
				break
			}
			indent := physicalIndent - indentPrefix
			if indent < baseIndent+2 {
				break
			}
			if indent != baseIndent+2 {
				p.diagnostic(item, "mpd-indent", SeverityError, "list continuation content must be indented exactly two spaces", Range{Start: uint32(next.start), End: uint32(next.end)}, "align the content two spaces beyond its item")
			}
			trimmed := nextBody[physicalIndent:]
			if width := fenceRun(trimmed); width >= 3 {
				p.parseCode(item, next, next.start+physicalIndent, width)
				continue
			}
			if len(trimmed) > 1 && trimmed[0] == '@' {
				if p.parseDirective(item, next, trimmed, next.start+physicalIndent, depth+1) {
					continue
				}
			}
			paragraph := p.addNode(KindParagraph, Range{Start: uint32(next.start), End: uint32(next.next)}, Position{Line: next.line, Column: uint32(physicalIndent + 1)})
			p.doc.Nodes[paragraph].Content = Range{Start: uint32(next.start + physicalIndent), End: uint32(next.end)}
			p.addChild(item, paragraph)
			p.parseInline(paragraph, next.start+physicalIndent, next.end, Position{Line: next.line, Column: uint32(physicalIndent + 1)})
			p.advance(next)
		}
		p.doc.Nodes[item].Source.End = uint32(p.offset)
	}
	p.doc.Nodes[list].Source.End = uint32(p.offset)
}

func (p *parser) parseQuote(parent uint32) {
	first := p.currentLine()
	_, firstContentStart := trimIndent(p.source[first.start:first.end], first.start)
	quote := p.addNode(KindQuote, Range{Start: uint32(first.start)}, Position{Line: first.line, Column: uint32(firstContentStart - first.start + 1)})
	p.addChild(parent, quote)
	var levels quoteStack
	levels.push(quote)
	limitReported := false
	for p.offset < len(p.source) {
		line := p.currentLine()
		body, absoluteStart := trimIndent(p.source[line.start:line.end], line.start)
		depth := quoteDepth(body)
		if depth == 0 {
			break
		}
		if depth > p.options.MaxNesting {
			if !limitReported {
				p.diagnostic(quote, "mpd-limit", SeverityError, "quote nesting exceeds the configured limit", Range{Start: uint32(line.start), End: uint32(line.end)}, "reduce nesting or increase MaxNesting")
				limitReported = true
			}
			depth = p.options.MaxNesting
		}
		for levels.count > depth {
			p.doc.Nodes[levels.pop()].Source.End = uint32(line.start)
		}
		for levels.count < depth {
			level := levels.count + 1
			column := absoluteStart - line.start + quotePrefixEnd(body, level-1) + 1
			nested := p.addNode(KindQuote, Range{Start: uint32(line.start)}, Position{Line: line.line, Column: uint32(column)})
			p.addChild(levels.last(), nested)
			levels.push(nested)
		}
		content := 0
		for consumed := 0; consumed < depth; consumed++ {
			content++
			if content < len(body) && body[content] == ' ' {
				content++
			}
		}
		if content < len(body) {
			column := absoluteStart - line.start + content + 1
			paragraph := p.addNode(KindParagraph, Range{Start: uint32(line.start), End: uint32(line.next)}, Position{Line: line.line, Column: uint32(column)})
			p.doc.Nodes[paragraph].Level = uint8(depth)
			p.doc.Nodes[paragraph].Content = Range{Start: uint32(absoluteStart + content), End: uint32(line.end)}
			p.addChild(levels.last(), paragraph)
			p.parseInline(paragraph, absoluteStart+content, line.end, Position{Line: line.line, Column: uint32(column)})
		}
		p.advance(line)
	}
	for levels.count > 1 {
		p.doc.Nodes[levels.pop()].Source.End = uint32(p.offset)
	}
	p.doc.Nodes[quote].Source.End = uint32(p.offset)
}

func quotePrefixEnd(line []byte, depth int) int {
	offset := 0
	for consumed := 0; consumed < depth && offset < len(line) && line[offset] == '>'; consumed++ {
		offset++
		if offset < len(line) && line[offset] == ' ' {
			offset++
		}
	}
	return offset
}

func heading(line []byte, absoluteStart int) (level, contentStart int, ok bool) {
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, 0, false
	}
	return level, absoluteStart + level + 1, true
}

func fenceRun(line []byte) int {
	count := 0
	for count < len(line) && line[count] == '`' {
		count++
	}
	return count
}

func quoteDepth(line []byte) int {
	depth, offset := 0, 0
	for offset < len(line) && line[offset] == '>' {
		depth++
		offset++
		if offset < len(line) && line[offset] == ' ' {
			offset++
		}
	}
	return depth
}

func trimIndent(line []byte, absoluteStart int) ([]byte, int) {
	offset := leadingSpaces(line)
	return line[offset:], absoluteStart + offset
}

func leadingSpaces(line []byte) int {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	return i
}

func invalidListIndent(line []byte) bool {
	offset, hasTab := 0, false
	for offset < len(line) && (line[offset] == ' ' || line[offset] == '\t') {
		if line[offset] == '\t' {
			hasTab = true
		}
		offset++
	}
	if offset == len(line) || !looksLikeListMarker(line[offset:]) {
		return false
	}
	return hasTab || offset%2 != 0
}

func looksLikeListMarker(line []byte) bool {
	if len(line) >= 2 && line[0] == '-' && line[1] == ' ' {
		return true
	}
	offset := 0
	for offset < len(line) && line[offset] >= '0' && line[offset] <= '9' {
		offset++
	}
	return offset > 0 && offset+1 < len(line) && line[offset] == '.' && line[offset+1] == ' '
}

func skipSpaces(source []byte, offset int) int {
	for offset < len(source) && source[offset] == ' ' {
		offset++
	}
	return offset
}

func trimSpaceEnd(source []byte, end int) int {
	for end > 0 && (source[end-1] == ' ' || source[end-1] == '\t') {
		end--
	}
	return end
}

func scanName(source []byte, offset int) int {
	if offset >= len(source) || !isNameStart(source[offset]) {
		return offset
	}
	offset++
	for offset < len(source) && isNameContinue(source[offset]) {
		offset++
	}
	return offset
}

func isNameStart(char byte) bool {
	return char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z'
}

func isNameContinue(char byte) bool {
	return isNameStart(char) || char >= '0' && char <= '9' || char == '-' || char == '.'
}

func isBlank(line []byte) bool {
	for _, char := range line {
		if char != ' ' && char != '\t' {
			return false
		}
	}
	return true
}
