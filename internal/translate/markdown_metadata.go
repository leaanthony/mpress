package translate

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// YAML node positions retain the original spelling of everything outside a
// translated scalar, including comments and flow-style hero/banner metadata.
func markdownMetadata(source []byte, end int) ([]Segment, error) {
	if end == 0 {
		return nil, nil
	}
	start := bytes.IndexByte(source, '\n') + 1
	finish := bytes.LastIndex(source[:end], []byte("---"))
	if finish < start {
		return nil, fmt.Errorf("invalid Markdown frontmatter")
	}
	var root yaml.Node
	if err := yaml.Unmarshal(source[start:finish], &root); err != nil {
		return nil, fmt.Errorf("parse translation frontmatter: %w", err)
	}
	lines := []int{start}
	for i := start; i < finish; i++ {
		if source[i] == '\n' {
			lines = append(lines, i+1)
		}
	}
	offset := func(node *yaml.Node) int {
		p := lines[node.Line-1]
		for column := 1; column < node.Column; column++ {
			_, size := utf8.DecodeRune(source[p:])
			p += size
		}
		return p
	}
	var result []Segment
	var walk func(*yaml.Node, string, string, int, bool)
	walk = func(node *yaml.Node, path, key string, indent int, flow bool) {
		switch node.Kind {
		case yaml.DocumentNode:
			for _, child := range node.Content {
				walk(child, path, key, indent, flow)
			}
		case yaml.MappingNode:
			for i := 0; i < len(node.Content); i += 2 {
				k, v := node.Content[i], node.Content[i+1]
				walk(v, path+"-"+k.Value, k.Value, k.Column-1, node.Style&yaml.FlowStyle != 0)
			}
		case yaml.SequenceNode:
			for i, child := range node.Content {
				walk(child, fmt.Sprintf("%s-%d", path, i), key, indent, node.Style&yaml.FlowStyle != 0)
			}
		case yaml.ScalarNode:
			if node.Tag != "!!str" || !(mpdTranslatableMetadata[key] || key == "alt" || key == "label") || !translatableText(node.Value) {
				return
			}
			a := offset(node)
			b := markdownYAMLScalarEnd(source, a, finish, indent, flow, node.Style)
			protected, placeholders := protectHTMLMarkup(node.Value)
			result = append(result, Segment{ID: path, Kind: "frontmatter", Original: node.Value, Text: protected, Start: a, End: b, SourceHash: Hash(node.Value), Placeholders: placeholders, Encoding: "yaml-string"})
		}
	}
	walk(&root, "fm", "", 0, false)
	return result, nil
}

func markdownYAMLScalarEnd(source []byte, start, limit, indent int, flow bool, style yaml.Style) int {
	if source[start] == '\'' || source[start] == '"' {
		return navigationLabelEnd(source[:limit], start)
	}
	end := start
	for end < limit && source[end] != '\n' {
		if flow && strings.ContainsRune(",}]", rune(source[end])) {
			break
		}
		if source[end] == '#' && end > start && (source[end-1] == ' ' || source[end-1] == '\t') {
			break
		}
		end++
	}
	if !flow && end < limit && source[end] == '\n' {
		next := end + 1
		for next < limit {
			lineEnd := next
			for lineEnd < limit && source[lineEnd] != '\n' {
				lineEnd++
			}
			line := source[next:lineEnd]
			trimmed := bytes.TrimSpace(line)
			if len(trimmed) == 0 {
				next = lineEnd + 1
				continue
			}
			padding := len(line) - len(bytes.TrimLeft(line, " \t"))
			if padding <= indent || bytes.HasPrefix(trimmed, []byte("#")) && style&(yaml.LiteralStyle|yaml.FoldedStyle) == 0 {
				break
			}
			end = lineEnd
			next = lineEnd + 1
		}
	}
	for end > start && strings.ContainsRune(" \t\r", rune(source[end-1])) {
		end--
	}
	return end
}
