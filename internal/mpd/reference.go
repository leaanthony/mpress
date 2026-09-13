package mpd

import (
	"bytes"
	"strconv"
)

type referenceKind uint8

const (
	referenceLink referenceKind = iota + 1
	referenceFootnote
)

type referenceRecord struct {
	node       uint32
	name       Range
	hash       uint64
	kind       referenceKind
	definition bool
}

func (p *parser) recordReference(node uint32, kind referenceKind, name Range) {
	p.appendReference(referenceRecord{node: node, name: name, hash: hashReference(p.doc.Text(name), kind), kind: kind})
}

func (p *parser) recordDefinition(node uint32, kind referenceKind) {
	owner := &p.doc.Nodes[node]
	for index := 0; index < int(owner.AttrCount); index++ {
		attribute := &p.doc.Attributes[int(owner.FirstAttr)+index]
		if !bytes.Equal(p.doc.Text(attribute.Name), []byte("id")) {
			continue
		}
		value := p.doc.Text(attribute.Value)
		if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
			p.diagnostic(node, "mpd-reference", SeverityError, "reference id must be a JSON string", attribute.Value, "quote the identifier")
			return
		}
		name := Range{Start: attribute.Value.Start + 1, End: attribute.Value.End - 1}
		value = p.doc.Text(name)
		hash := hashReference(value, kind)
		if bytes.IndexByte(value, '\\') >= 0 {
			// Escaped JSON identifiers need normalization during validation.
			// A zero hash opts into that rare slow path without burdening the
			// ordinary allocation-free record.
			hash = 0
		}
		p.appendReference(referenceRecord{node: node, name: name, hash: hash, kind: kind, definition: true})
		return
	}
	p.diagnostic(node, "mpd-reference", SeverityError, "reference definition requires an id", owner.Source, "add id=\"identifier\"")
}

func (p *parser) appendReference(record referenceRecord) {
	if p.referenceCount < len(p.references) {
		p.references[p.referenceCount] = record
	} else {
		p.referenceOverflow = append(p.referenceOverflow, record)
	}
	p.referenceCount++
}

func (p *parser) referenceAt(index int) referenceRecord {
	if index < len(p.references) {
		return p.references[index]
	}
	return p.referenceOverflow[index-len(p.references)]
}

func (p *parser) validateReferences() {
	if p.referenceCount == 0 {
		return
	}
	if p.referenceCount <= len(p.references) {
		p.validateSmallReferences()
		return
	}
	definitions := make(map[referenceKey]uint32, p.referenceCount)
	for index := 0; index < p.referenceCount; index++ {
		record := p.referenceAt(index)
		if !record.definition {
			continue
		}
		key := referenceKey{kind: record.kind, name: p.referenceName(record)}
		if _, duplicate := definitions[key]; duplicate {
			p.diagnostic(record.node, "mpd-reference", SeverityError, "duplicate reference definition", record.name, "use a unique identifier")
		} else {
			definitions[key] = record.node
		}
	}
	for index := 0; index < p.referenceCount; index++ {
		record := p.referenceAt(index)
		if record.definition {
			continue
		}
		if _, found := definitions[referenceKey{kind: record.kind, name: p.referenceName(record)}]; !found {
			p.diagnostic(record.node, "mpd-reference", SeverityError, "undefined reference", record.name, "add a matching definition")
		}
	}
}

func (p *parser) validateSmallReferences() {
	for index := 0; index < p.referenceCount; index++ {
		record := p.referenceAt(index)
		if !record.definition {
			continue
		}
		for previous := 0; previous < index; previous++ {
			candidate := p.referenceAt(previous)
			if candidate.definition && referencesEqual(p.doc, record, candidate) {
				p.diagnostic(record.node, "mpd-reference", SeverityError, "duplicate reference definition", record.name, "use a unique identifier")
				break
			}
		}
	}
	for index := 0; index < p.referenceCount; index++ {
		record := p.referenceAt(index)
		if record.definition {
			continue
		}
		found := false
		for candidateIndex := 0; candidateIndex < p.referenceCount; candidateIndex++ {
			candidate := p.referenceAt(candidateIndex)
			if candidate.definition && referencesEqual(p.doc, record, candidate) {
				found = true
				break
			}
		}
		if !found {
			p.diagnostic(record.node, "mpd-reference", SeverityError, "undefined reference", record.name, "add a matching definition")
		}
	}
}

func referencesEqual(doc *Document, left, right referenceRecord) bool {
	if left.kind != right.kind {
		return false
	}
	if left.hash != 0 && right.hash != 0 {
		return left.hash == right.hash && bytes.Equal(doc.Text(left.name), doc.Text(right.name))
	}
	return normalizedReferenceName(doc, left) == normalizedReferenceName(doc, right)
}

func (p *parser) referenceName(record referenceRecord) string {
	return normalizedReferenceName(p.doc, record)
}

func normalizedReferenceName(doc *Document, record referenceRecord) string {
	value := doc.Text(record.name)
	if !record.definition || record.hash != 0 {
		return string(value)
	}
	quoted := Range{Start: record.name.Start - 1, End: record.name.End + 1}
	decoded, err := strconv.Unquote(string(doc.Text(quoted)))
	if err != nil {
		return string(value)
	}
	return decoded
}

func hashReference(name []byte, kind referenceKind) uint64 {
	hash := uint64(1469598103934665603) ^ uint64(kind)
	for _, char := range name {
		hash = (hash ^ uint64(char)) * 1099511628211
	}
	return hash
}

type referenceKey struct {
	kind referenceKind
	name string
}
