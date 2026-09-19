package translate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func markdownEmphasisToken(value string) bool {
	switch value {
	case "<strong>", "</strong>", "<em>", "</em>":
		return true
	}
	return value != "" && (strings.Trim(value, "*") == "" || strings.Trim(value, "_") == "")
}

// Delimiter runs are not individual formatting nodes: **** can open two
// nested strong nodes, and *** can close a strong node and an emphasis node.
// Count parsed emphasis weight, retaining the existing freedom to move emphasis
// between translated clauses without treating literal stars or code as markup.
func markdownEmphasisWeight(segment Segment) int {
	hasMarkup := false
	for _, value := range segment.Placeholders {
		hasMarkup = hasMarkup || markdownEmphasisToken(value)
	}
	if !hasMarkup {
		return 0
	}
	source := []byte(segment.Original)
	p := extractionParserPool.Get().(parser.Parser)
	root := p.Parse(text.NewReader(source))
	extractionParserPool.Put(p)
	weight := 0
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch item := node.(type) {
		case *ast.Emphasis:
			weight += 2 * item.Level
		case *ast.RawHTML:
			for i := 0; i < item.Segments.Len(); i++ {
				span := item.Segments.At(i)
				switch string(span.Value(source)) {
				case "<strong>", "</strong>":
					weight += 2
				case "<em>", "</em>":
					weight++
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return weight
}

func validateMarkdownProtection(source, target Segment) error {
	sourceWeight, targetWeight := markdownEmphasisWeight(source), markdownEmphasisWeight(target)
	if sourceWeight != targetWeight {
		return fmt.Errorf("segment %s changes parsed emphasis weight from %d to %d", source.ID, sourceWeight, targetWeight)
	}
	available := map[string]int{}
	for _, value := range target.Placeholders {
		if targetWeight > 0 && markdownEmphasisToken(value) {
			continue
		}
		available[value]++
	}
	for _, original := range source.Placeholders {
		if sourceWeight > 0 && markdownEmphasisToken(original) {
			continue
		}
		if available[original] == 0 {
			return fmt.Errorf("segment %s does not preserve protected content %q", source.ID, original)
		}
		available[original]--
	}
	plain := translationPlaceholderPattern.ReplaceAllString(source.Text, " ")
	for value, count := range available {
		if count == 0 || markdownExtraQuantity.MatchString(value) || supportedLiteralFormatting(value, count, plain) {
			continue
		}
		return fmt.Errorf("segment %s adds protected content %q", source.ID, value)
	}
	return nil
}

var markdownLiteralWords = map[string]*regexp.Regexp{
	"true":  regexp.MustCompile(`\btrue\b`),
	"false": regexp.MustCompile(`\bfalse\b`),
	"nil":   regexp.MustCompile(`\bnil\b`),
	"null":  regexp.MustCompile(`\bnull\b`),
}

// A target may put an existing literal keyword in code font. This does not
// license new commands, changed values, or extra occurrences of that keyword.
func supportedLiteralFormatting(value string, count int, plainSource string) bool {
	if len(value) < 3 || value[0] != '`' || value[len(value)-1] != '`' {
		return false
	}
	words := markdownLiteralWords[value[1:len(value)-1]]
	return words != nil && len(words.FindAllStringIndex(plainSource, -1)) >= count
}
