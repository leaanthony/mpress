package translate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var markdownComponentLink = regexp.MustCompile(`^[ \t]*\[([^]]+)\]\([^)]+\)[ \t]*$`)
var markdownComponentItem = regexp.MustCompile(`^[ \t]*(?:[-+*]|[0-9]+[.)])[ \t]+\*\*([^*]+)\*\*(?:[ \t]+(.+))?$`)
var markdownPricingFeature = regexp.MustCompile(`^[ \t]*-[ \t]+[✓✗][ \t]+(.+)$`)
var markdownPricingRate = regexp.MustCompile(`^[ \t]*[$€£¥][0-9.,]+(?:/([^\r\n]+))?[ \t]*$`)

// These components interpret line prefixes as fields rather than ordinary
// Markdown. Keep those boundaries outside the translated ranges.
func markdownComponentLine(component string, line []byte, base, ordinal, index int) ([]Segment, bool) {
	var spans [][]int
	switch component {
	case "cards", "resources", "pricing":
		if match := markdownComponentLink.FindSubmatchIndex(line); match != nil {
			spans = append(spans, match[2:4])
		}
		if component == "pricing" {
			if strings.TrimSpace(string(line)) == "recommended" {
				return nil, true
			}
			if match := markdownPricingFeature.FindSubmatchIndex(line); match != nil {
				spans = append(spans, match[2:4])
			}
			if match := markdownPricingRate.FindSubmatchIndex(line); match != nil {
				if match[2] >= 0 {
					spans = append(spans, match[2:4])
				} else {
					return nil, true
				}
			}
		}
	case "timeline", "capabilities":
		if match := markdownComponentItem.FindSubmatchIndex(line); match != nil {
			spans = append(spans, match[2:4])
			if len(match) > 4 && match[4] >= 0 {
				spans = append(spans, match[4:6])
			}
		}
	case "headline", "testimonials":
		trimmed := strings.TrimSpace(string(line))
		if trimmed != "" && trimmed != "---" && !strings.HasPrefix(trimmed, "@") {
			spans = append(spans, []int{0, len(line)})
		}
	}
	var segments []Segment
	for number, span := range spans {
		a, b := span[0], span[1]
		raw := string(line[a:b])
		p := extractionParserPool.Get().(parser.Parser)
		root := p.Parse(text.NewReader(line[a:b]))
		extractionParserPool.Put(p)
		protected, placeholders := protectMarkdownInline(root, line[a:b], 0, b-a)
		segments = append(segments, Segment{ID: fmt.Sprintf("c%04d-line-%03d-%02d", ordinal, index, number+1), Kind: "component", Original: raw, Text: protected, Start: base + a, End: base + b, SourceHash: Hash(raw), Placeholders: placeholders})
	}
	return segments, len(spans) > 0
}

func componentTranslationAttributes(name string) map[string]bool {
	if name != "input" && name != "variant" {
		return mpdTranslatableAttributes
	}
	attributes := make(map[string]bool, len(mpdTranslatableAttributes))
	for key, value := range mpdTranslatableAttributes {
		if key != "name" {
			attributes[key] = value
		}
	}
	return attributes
}
