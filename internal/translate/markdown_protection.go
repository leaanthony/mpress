package translate

import (
	"fmt"
	"regexp"
)

// The MPD Markdown exporter uses HTML emphasis next to CJK characters where
// delimiter spelling would change parsing. Compare the parsed syntax roles,
// then retain the target spelling when reusing that translation.
func markdownInlineRole(value string) string {
	switch value {
	case "**", "__", "<strong>", "</strong>":
		return "emphasis:strong"
	case "*", "_", "<em>", "</em>":
		return "emphasis:normal"
	default:
		return "literal:" + value
	}
}

func reconcileMarkdownPlaceholders(source, target Segment) (Segment, error) {
	if err := validateMarkdownProtection(source, target); err != nil {
		return source, err
	}
	adjusted, err := reconcileMatchingMarkdownPlaceholders(source, target)
	if err == nil {
		return adjusted, nil
	}
	// Equivalent nested formatting may have a different number of tokens. Reuse
	// the parsed target's own tokens rather than flattening its valid markup.
	source.Text = target.Text
	source.Placeholders = target.Placeholders
	return source, nil
}

func reconcileMatchingMarkdownPlaceholders(source, target Segment) (Segment, error) {
	available := map[string][]string{}
	for _, key := range orderedPlaceholderTokens(target) {
		value := target.Placeholders[key]
		role := markdownInlineRole(value)
		available[role] = append(available[role], value)
	}
	values := map[string]string{}
	for _, key := range orderedPlaceholderTokens(source) {
		original := source.Placeholders[key]
		role := markdownInlineRole(original)
		matches := available[role]
		if len(matches) == 0 {
			return source, fmt.Errorf("segment %s does not preserve protected content %q", source.ID, original)
		}
		values[key] = matches[0]
		available[role] = matches[1:]
	}
	for _, extra := range available {
		for _, value := range extra {
			if markdownExtraQuantity.MatchString(value) {
				continue
			}
			if len(extra) > 0 {
				return source, fmt.Errorf("segment %s adds protected content %q", source.ID, value)
			}
		}
	}
	source.Placeholders = values
	return source, nil
}

func prepareExistingForFormat(format string, source *Segment, target Segment) (string, error) {
	if format == markdownTranslationFormat {
		originalText := source.Text
		adjusted, err := reconcileMarkdownPlaceholders(*source, target)
		if err != nil {
			return "", err
		}
		*source = adjusted
		if source.Text != originalText {
			return target.Text, nil
		}
	}
	return prepareExistingSegment(*source, target)
}

var markdownExtraQuantity = regexp.MustCompile(`^[vV]?[0-9]+(?:[.,:/-][0-9]+)*$`)
