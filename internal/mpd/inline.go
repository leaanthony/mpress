package mpd

import (
	"bytes"
	"unicode/utf8"
)

type inlineFrame struct {
	node      uint32
	parent    uint32
	delimiter byte
}

var inlineSpecial = [256]bool{
	'\\': true, '\r': true, '\n': true, '_': true, '*': true, '`': true,
	'!': true, '[': true, '<': true, ':': true, '@': true,
}

// parseInline is a single forward scan. Nested role and link labels recurse
// only into disjoint source ranges, so total work remains linear.
func (p *parser) parseInline(parent uint32, start, end int, position Position) {
	p.parseInlineDepth(parent, start, end, position, 0)
}

func (p *parser) parseInlineDepth(parent uint32, start, end int, position Position, depth int) {
	if start >= end {
		return
	}
	if depth > p.options.MaxNesting {
		p.addText(parent, start, end, start, end, position)
		p.diagnostic(parent, "mpd-limit", SeverityError, "inline nesting exceeds the configured limit", Range{Start: uint32(start), End: uint32(end)}, "reduce nesting or increase MaxNesting")
		return
	}
	current := parent
	positions := positionTracker{source: p.source, offset: start, position: position, ascii: p.ascii}
	stack := make([]inlineFrame, 0, 4)
	limitReported := false
	textStart := start
	flush := func(until int) {
		if textStart < until {
			p.addText(current, textStart, until, textStart, until, positions.at(textStart))
		}
	}
	for i := start; i < end; {
		nextSpecial := indexInlineSpecial(p.source[i:end])
		if nextSpecial < 0 {
			i = end
			break
		}
		i += nextSpecial
		char := p.source[i]
		if char == '\\' {
			if i+1 < end && (p.source[i+1] == '\n' || p.source[i+1] == '\r') {
				flush(i)
				next := i + 2
				if p.source[i+1] == '\r' && next < end && p.source[next] == '\n' {
					next++
				}
				node := p.addNode(KindSoftBreak, Range{Start: uint32(i), End: uint32(next)}, positions.at(i))
				p.addChild(current, node)
				i, textStart = next, next
				continue
			}
			if i+1 < end && isASCIIPunctuation(p.source[i+1]) {
				flush(i)
				node := p.addNode(KindText, Range{Start: uint32(i), End: uint32(i + 2)}, positions.at(i))
				p.doc.Nodes[node].Content = Range{Start: uint32(i + 1), End: uint32(i + 2)}
				p.addChild(current, node)
				i += 2
				textStart = i
				continue
			}
		}
		if char == '\r' || char == '\n' {
			flush(i)
			next := i + 1
			if char == '\r' && next < end && p.source[next] == '\n' {
				next++
			}
			node := p.addNode(KindHardBreak, Range{Start: uint32(i), End: uint32(next)}, positions.at(i))
			p.addChild(current, node)
			i, textStart = next, next
			continue
		}
		if char == '_' || char == '*' {
			canClose := i > start && !isInlineWhitespace(p.source[i-1])
			if len(stack) > 0 && stack[len(stack)-1].delimiter == char && canClose {
				flush(i)
				frame := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				p.doc.Nodes[frame.node].Content.End = uint32(i)
				p.doc.Nodes[frame.node].Source.End = uint32(i + 1)
				current = parent
				if len(stack) > 0 {
					current = stack[len(stack)-1].node
				}
				i++
				textStart = i
				continue
			}
			if i+1 < end && !isInlineWhitespace(p.source[i+1]) {
				if len(stack) >= p.options.MaxNesting {
					if !limitReported {
						p.diagnostic(parent, "mpd-limit", SeverityError, "inline delimiter nesting exceeds the configured limit", Range{Start: uint32(i), End: uint32(i + 1)}, "reduce nesting or increase MaxNesting")
						limitReported = true
					}
					i++
					continue
				}
				flush(i)
				kind := KindEmphasis
				if char == '*' {
					kind = KindStrong
				}
				node := p.addNode(kind, Range{Start: uint32(i)}, positions.at(i))
				p.doc.Nodes[node].Content.Start = uint32(i + 1)
				p.addChild(current, node)
				stack = append(stack, inlineFrame{node: node, parent: current, delimiter: char})
				current = node
				i++
				textStart = i
				continue
			}
		}
		if char == '`' {
			run := byteRun(p.source, i, end, '`')
			closeStart := findExactRun(p.source, i+run, end, '`', run)
			flush(i)
			node := p.addNode(KindCodeSpan, Range{Start: uint32(i)}, positions.at(i))
			p.addChild(current, node)
			if closeStart < 0 {
				p.doc.Nodes[node].Content = Range{Start: uint32(i + run), End: uint32(end)}
				p.doc.Nodes[node].Source.End = uint32(end)
				p.doc.Nodes[node].Flags |= FlagUnclosed
				p.diagnostic(node, "mpd-unclosed", SeverityWarning, "unclosed code span", p.doc.Nodes[node].Source, "add a matching backtick run")
				i, textStart = end, end
				continue
			}
			p.doc.Nodes[node].Content = Range{Start: uint32(i + run), End: uint32(closeStart)}
			p.doc.Nodes[node].Source.End = uint32(closeStart + run)
			i, textStart = closeStart+run, closeStart+run
			continue
		}
		if char == '!' && i+1 < end && p.source[i+1] == '[' {
			flush(i)
			if next, ok := p.parseLinkLike(current, i, end, true, positions.at(i), depth); ok {
				i, textStart = next, next
				continue
			}
			textStart = i
		}
		if char == '[' {
			if i+2 < end && p.source[i+1] == '^' {
				if close := indexUnescapedByte(p.source, i+2, end, ']'); close >= 0 {
					flush(i)
					node := p.addNode(KindFootnoteReference, Range{Start: uint32(i), End: uint32(close + 1)}, positions.at(i))
					p.doc.Nodes[node].Name = Range{Start: uint32(i + 2), End: uint32(close)}
					p.addChild(current, node)
					p.recordReference(node, referenceFootnote, p.doc.Nodes[node].Name)
					i, textStart = close+1, close+1
					continue
				}
			}
			flush(i)
			if next, ok := p.parseLinkLike(current, i, end, false, positions.at(i), depth); ok {
				i, textStart = next, next
				continue
			}
			textStart = i
		}
		if char == '<' {
			if close := indexUnescapedByte(p.source, i+1, end, '>'); close > i+1 && isAutomaticDestination(p.source[i+1:close]) {
				flush(i)
				node := p.addNode(KindAutomaticLink, Range{Start: uint32(i), End: uint32(close + 1)}, positions.at(i))
				p.doc.Nodes[node].Content = Range{Start: uint32(i + 1), End: uint32(close)}
				p.addChild(current, node)
				i, textStart = close+1, close+1
				continue
			}
		}
		if char == ':' {
			if close := scanEmoji(p.source, i, end); close > 0 {
				name := p.source[i+1 : close]
				if isEmojiName(name) {
					flush(i)
					node := p.addNode(KindEmoji, Range{Start: uint32(i), End: uint32(close + 1)}, positions.at(i))
					p.doc.Nodes[node].Name = Range{Start: uint32(i + 1), End: uint32(close)}
					p.addChild(current, node)
					i, textStart = close+1, close+1
					continue
				}
			}
		}
		if char == '@' && i+1 < end {
			nameEnd := scanName(p.source, i+1)
			if nameEnd > i+1 && nameEnd < end && p.source[nameEnd] == '[' {
				if close := findBalancedBracket(p.source, nameEnd, end); close >= 0 {
					flush(i)
					kind := KindInlineRole
					name := p.source[i+1 : nameEnd]
					if bytes.Equal(name, []byte("metadata")) {
						kind = KindMetadataReference
					}
					nodePosition := positions.at(i)
					node := p.addNode(kind, Range{Start: uint32(i), End: uint32(close + 1)}, nodePosition)
					p.doc.Nodes[node].Name = Range{Start: uint32(i + 1), End: uint32(nameEnd)}
					p.doc.Nodes[node].Content = Range{Start: uint32(nameEnd + 1), End: uint32(close)}
					p.addChild(current, node)
					if kind == KindMetadataReference {
						if !p.hasMetadata(p.source[nameEnd+1 : close]) {
							p.diagnostic(node, "mpd-metadata-reference", SeverityError, "undefined metadata reference", p.doc.Nodes[node].Source, "define the metadata key or correct the reference")
						}
					} else {
						p.parseInlineDepth(node, nameEnd+1, close, positionAfter(p.source, nodePosition, i, nameEnd+1, p.ascii), depth+1)
					}
					i, textStart = close+1, close+1
					continue
				}
			}
		}
		i++
	}
	flush(end)
	// Unmatched openers are literal bytes, but content parsed after them keeps
	// its meaning. Promote each provisional node's children to its parent from
	// the inside out. This preserves a fully connected arena without rescanning.
	for index := len(stack) - 1; index >= 0; index-- {
		frame := stack[index]
		node := &p.doc.Nodes[frame.node]
		first, last, next := node.FirstChild, node.LastChild, node.NextSibling
		node.Kind = KindText
		node.Content = Range{Start: node.Source.Start, End: node.Source.Start + 1}
		node.Source.End = node.Source.Start + 1
		node.FirstChild, node.LastChild = noIndex, noIndex
		if first == noIndex {
			continue
		}
		node.NextSibling = first
		p.doc.Nodes[last].NextSibling = next
		parentNode := &p.doc.Nodes[frame.parent]
		if parentNode.LastChild == frame.node {
			parentNode.LastChild = last
		}
	}
}

func indexInlineSpecial(source []byte) int {
	index := 0
	for ; index+8 <= len(source); index += 8 {
		if inlineSpecial[source[index]] {
			return index
		}
		if inlineSpecial[source[index+1]] {
			return index + 1
		}
		if inlineSpecial[source[index+2]] {
			return index + 2
		}
		if inlineSpecial[source[index+3]] {
			return index + 3
		}
		if inlineSpecial[source[index+4]] {
			return index + 4
		}
		if inlineSpecial[source[index+5]] {
			return index + 5
		}
		if inlineSpecial[source[index+6]] {
			return index + 6
		}
		if inlineSpecial[source[index+7]] {
			return index + 7
		}
	}
	for ; index < len(source); index++ {
		if inlineSpecial[source[index]] {
			return index
		}
	}
	return -1
}

func (p *parser) addText(parent uint32, sourceStart, sourceEnd, contentStart, contentEnd int, position Position) {
	if sourceStart >= sourceEnd {
		return
	}
	last := p.doc.Nodes[parent].LastChild
	if last != noIndex {
		node := &p.doc.Nodes[last]
		if node.Kind == KindText && node.Source.End == uint32(sourceStart) && node.Content.End == uint32(contentStart) {
			node.Source.End = uint32(sourceEnd)
			node.Content.End = uint32(contentEnd)
			return
		}
	}
	node := p.addNode(KindText, Range{Start: uint32(sourceStart), End: uint32(sourceEnd)}, position)
	p.doc.Nodes[node].Content = Range{Start: uint32(contentStart), End: uint32(contentEnd)}
	p.addChild(parent, node)
}

func (p *parser) parseLinkLike(parent uint32, start, end int, image bool, position Position, depth int) (int, bool) {
	labelOpen := start
	if image {
		labelOpen++
	}
	labelClose := findBalancedBracket(p.source, labelOpen, end)
	if labelClose < 0 || labelClose+1 >= end {
		return start, false
	}
	next := labelClose + 1
	kind := KindLink
	if image {
		kind = KindImage
	}
	if p.source[next] == '(' {
		close := findBalancedParenthesis(p.source, next, end)
		if close < 0 {
			return start, false
		}
		node := p.addNode(kind, Range{Start: uint32(start), End: uint32(close + 1)}, position)
		p.doc.Nodes[node].Content = Range{Start: uint32(next + 1), End: uint32(close)}
		p.addChild(parent, node)
		p.parseInlineDepth(node, labelOpen+1, labelClose, positionAfter(p.source, position, start, labelOpen+1, p.ascii), depth+1)
		return close + 1, true
	}
	if p.source[next] == '[' {
		refClose := indexUnescapedByte(p.source, next+1, end, ']')
		if refClose < 0 {
			return start, false
		}
		node := p.addNode(kind, Range{Start: uint32(start), End: uint32(refClose + 1)}, position)
		p.doc.Nodes[node].Name = Range{Start: uint32(next + 1), End: uint32(refClose)}
		p.doc.Nodes[node].Flags |= FlagReference
		p.addChild(parent, node)
		p.recordReference(node, referenceLink, p.doc.Nodes[node].Name)
		p.parseInlineDepth(node, labelOpen+1, labelClose, positionAfter(p.source, position, start, labelOpen+1, p.ascii), depth+1)
		return refClose + 1, true
	}
	return start, false
}

func (p *parser) hasMetadata(name []byte) bool {
	if len(p.doc.Nodes) < 2 || p.doc.Nodes[1].Kind != KindMetadata {
		return false
	}
	metadata := &p.doc.Nodes[1]
	start := int(metadata.FirstAttr)
	for i := 0; i < int(metadata.AttrCount); i++ {
		attribute := &p.doc.Attributes[start+i]
		if bytes.Equal(p.doc.Text(attribute.Name), name) {
			return true
		}
	}
	return false
}

func findBalancedBracket(source []byte, open, end int) int {
	if open >= end || source[open] != '[' {
		return -1
	}
	depth := 1
	for i := open + 1; i < end; i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		switch source[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func findBalancedParenthesis(source []byte, open, end int) int {
	if open >= end || source[open] != '(' {
		return -1
	}
	depth, quoted := 1, false
	for i := open + 1; i < end; i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == '"' {
			quoted = !quoted
			continue
		}
		if quoted {
			continue
		}
		switch source[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		case '\r', '\n':
			return -1
		}
	}
	return -1
}

func findExactRun(source []byte, start, end int, char byte, width int) int {
	for i := start; i < end; {
		if source[i] != char {
			i++
			continue
		}
		run := byteRun(source, i, end, char)
		if run == width {
			return i
		}
		i += run
	}
	return -1
}

func byteRun(source []byte, start, end int, char byte) int {
	i := start
	for i < end && source[i] == char {
		i++
	}
	return i - start
}

func indexUnescapedByte(source []byte, start, end int, wanted byte) int {
	for i := start; i < end; i++ {
		if source[i] == '\\' {
			i++
			continue
		}
		if source[i] == wanted {
			return i
		}
	}
	return -1
}

func scanEmoji(source []byte, start, end int) int {
	if start+2 >= end {
		return -1
	}
	for i := start + 1; i < end; i++ {
		char := source[i]
		if char == ':' {
			if i == start+1 {
				return -1
			}
			return i
		}
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_' || char == '-' || char == '+') {
			return -1
		}
	}
	return -1
}

func isAutomaticDestination(value []byte) bool {
	return bytes.HasPrefix(value, []byte("http://")) || bytes.HasPrefix(value, []byte("https://")) || bytes.HasPrefix(value, []byte("mailto:")) || bytes.Contains(value, []byte("@"))
}

func isInlineWhitespace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

func isASCIIPunctuation(char byte) bool {
	return char >= '!' && char <= '/' || char >= ':' && char <= '@' || char >= '[' && char <= '`' || char >= '{' && char <= '~'
}

type positionTracker struct {
	source   []byte
	offset   int
	position Position
	ascii    bool
}

func (t *positionTracker) at(target int) Position {
	if target < t.offset {
		return t.position
	}
	if t.ascii {
		for t.offset < target {
			segment := t.source[t.offset:target]
			lineFeed := bytes.IndexByte(segment, '\n')
			carriageReturn := bytes.IndexByte(segment, '\r')
			newline := lineFeed
			if newline < 0 || carriageReturn >= 0 && carriageReturn < newline {
				newline = carriageReturn
			}
			if newline < 0 {
				t.position.Column += uint32(target - t.offset)
				t.offset = target
				break
			}
			t.offset += newline + 1
			if segment[newline] == '\r' && t.offset < target && t.source[t.offset] == '\n' {
				t.offset++
			}
			t.position.Line++
			t.position.Column = 1
		}
		return t.position
	}
	for t.offset < target {
		char := t.source[t.offset]
		switch char {
		case '\r':
			t.offset++
			if t.offset < target && t.source[t.offset] == '\n' {
				t.offset++
			}
			t.position.Line++
			t.position.Column = 1
		case '\n':
			t.offset++
			t.position.Line++
			t.position.Column = 1
		default:
			size := 1
			if char >= utf8.RuneSelf {
				_, size = utf8.DecodeRune(t.source[t.offset:target])
				if size < 1 {
					size = 1
				}
			}
			t.offset += size
			t.position.Column++
		}
	}
	return t.position
}

func positionAfter(source []byte, position Position, start, end int, ascii bool) Position {
	tracker := positionTracker{source: source, offset: start, position: position, ascii: ascii}
	return tracker.at(end)
}
