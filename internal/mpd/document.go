// Package mpd parses MPress Document source into a source-preserving syntax
// tree. Parsing never resolves imports, opens assets, or renders HTML.
package mpd

import "fmt"

// Kind identifies a syntax-tree node.
type Kind uint8

const (
	KindInvalid Kind = iota
	KindDocument
	KindMetadata
	KindParagraph
	KindHeading
	KindRule
	KindList
	KindItem
	KindQuote
	KindCode
	KindRawHTML
	KindComment
	KindTable
	KindTableRow
	KindTableCell
	KindFootnote
	KindComponent
	KindImport
	KindInclude
	KindText
	KindSoftBreak
	KindHardBreak
	KindEmphasis
	KindStrong
	KindCodeSpan
	KindLink
	KindImage
	KindAutomaticLink
	KindEmoji
	KindFootnoteReference
	KindMetadataReference
	KindInlineRole
)

var kindNames = [...]string{
	KindInvalid:           "invalid",
	KindDocument:          "document",
	KindMetadata:          "metadata",
	KindParagraph:         "paragraph",
	KindHeading:           "heading",
	KindRule:              "rule",
	KindList:              "list",
	KindItem:              "item",
	KindQuote:             "quote",
	KindCode:              "code",
	KindRawHTML:           "raw-html",
	KindComment:           "comment",
	KindTable:             "table",
	KindTableRow:          "table-row",
	KindTableCell:         "table-cell",
	KindFootnote:          "footnote",
	KindComponent:         "component",
	KindImport:            "import",
	KindInclude:           "include",
	KindText:              "text",
	KindSoftBreak:         "soft-break",
	KindHardBreak:         "hard-break",
	KindEmphasis:          "emphasis",
	KindStrong:            "strong",
	KindCodeSpan:          "code-span",
	KindLink:              "link",
	KindImage:             "image",
	KindAutomaticLink:     "automatic-link",
	KindEmoji:             "emoji",
	KindFootnoteReference: "footnote-reference",
	KindMetadataReference: "metadata-reference",
	KindInlineRole:        "inline-role",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return fmt.Sprintf("kind(%d)", k)
}

// Range is a half-open byte range in Document.Source.
type Range struct {
	Start uint32
	End   uint32
}

func (r Range) Len() int { return int(r.End - r.Start) }

// Position records the source location at the start of a range. Line and
// Column are one-based. Byte ranges remain authoritative.
type Position struct {
	Line   uint32
	Column uint32
}

// Node is one entry in Document.Nodes. Children are linked in source order.
// Name refers to a component name, role name, reference identifier, or emoji
// shortcode. Content excludes structural delimiters where the kind has them.
type Node struct {
	Kind        Kind
	Level       uint8
	Flags       uint16
	Source      Range
	Content     Range
	Name        Range
	Position    Position
	FirstChild  uint32
	LastChild   uint32
	NextSibling uint32
	FirstAttr   uint32
	AttrCount   uint16
}

const noIndex = ^uint32(0)

// Attribute is an ordered metadata field or component attribute. Value is the
// exact JSON source range. A shorthand Boolean attribute has an empty Value
// range and Flag set.
type Attribute struct {
	Name     Range
	Value    Range
	Position Position
	Flag     bool
	Metadata bool
}

// Severity identifies the impact of a diagnostic.
type Severity uint8

const (
	SeverityWarning Severity = iota + 1
	SeverityError
)

// Diagnostic reports malformed input without discarding source bytes.
type Diagnostic struct {
	Node       uint32
	Code       string
	Severity   Severity
	Message    string
	Suggestion string
	Source     Range
	Position   Position
}

// DiagnosticsFor returns an allocation-free iterator over diagnostics attached
// to node. Diagnostics are expected to be rare, so a compact document-level
// arena is preferable to two fields on every syntax node.
func (d *Document) DiagnosticsFor(node uint32) DiagnosticIterator {
	return DiagnosticIterator{doc: d, node: node}
}

type DiagnosticIterator struct {
	doc   *Document
	node  uint32
	index int
}

func (it *DiagnosticIterator) Next() (*Diagnostic, bool) {
	for it.index < len(it.doc.Diagnostics) {
		diagnostic := &it.doc.Diagnostics[it.index]
		it.index++
		if diagnostic.Node == it.node {
			return diagnostic, true
		}
	}
	return nil, false
}

// Document owns the syntax arenas and refers to the caller-owned immutable
// Source. The caller must not modify Source while using the document.
type Document struct {
	Filename    string
	Source      []byte
	Schema      uint32
	Root        uint32
	Nodes       []Node
	Attributes  []Attribute
	Diagnostics []Diagnostic
}

// Text returns the source bytes in a range without allocating.
func (d *Document) Text(r Range) []byte {
	if r.End < r.Start || int(r.End) > len(d.Source) {
		return nil
	}
	return d.Source[r.Start:r.End]
}

// Children returns an iterator over direct children of node.
func (d *Document) Children(node uint32) ChildIterator {
	if int(node) >= len(d.Nodes) {
		return ChildIterator{doc: d, next: noIndex}
	}
	return ChildIterator{doc: d, next: d.Nodes[node].FirstChild}
}

// ChildIterator walks linked children without allocating a temporary slice.
type ChildIterator struct {
	doc  *Document
	next uint32
}

func (it *ChildIterator) Next() (uint32, *Node, bool) {
	if it.next == noIndex || int(it.next) >= len(it.doc.Nodes) {
		return 0, nil, false
	}
	index := it.next
	node := &it.doc.Nodes[index]
	it.next = node.NextSibling
	return index, node, true
}

// NodeFlags describe syntax details without widening the common node shape.
const (
	FlagOrdered uint16 = 1 << iota
	FlagChecked
	FlagUnchecked
	FlagUnclosed
	FlagReference
)
