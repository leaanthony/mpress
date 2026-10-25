package translate

import (
	"maps"
	"strings"
)

// Repair delimiter parsing only when every original bold placeholder survives
// in source order. Explicit tags retain those exact spans next to CJK text.
func repairMarkdownEmphasis(format string, segment Segment, translated string) Segment {
	if format != markdownTranslationFormat {
		return segment
	}
	tokens := markdownStrongRepairTokens(segment, translated)
	if len(tokens) == 0 {
		return segment
	}
	restored, err := restore(segment, translated, false)
	if err != nil {
		return segment
	}
	target, err := ExtractMarkdown([]byte(restored))
	if err != nil || len(target.Segments) != 1 || markdownEmphasisWeight(target.Segments[0]) >= 2*len(tokens) {
		return segment
	}
	segment.Placeholders = maps.Clone(segment.Placeholders)
	for i := 0; i < len(tokens); i += 2 {
		segment.Placeholders[tokens[i]] = "<strong>"
		segment.Placeholders[tokens[i+1]] = "</strong>"
	}
	return segment
}

func markdownStrongRepairTokens(segment Segment, translated string) []string {
	var tokens []string
	previous := -1
	for _, token := range orderedPlaceholderTokens(segment) {
		marker := segment.Placeholders[token]
		if !markdownEmphasisToken(marker) {
			continue
		}
		if marker != "**" && marker != "__" {
			return nil
		}
		position := strings.Index(translated, token)
		if strings.Count(translated, token) != 1 || position <= previous {
			return nil
		}
		previous = position
		tokens = append(tokens, token)
	}
	if len(tokens) == 0 || len(tokens)%2 != 0 || markdownEmphasisWeight(segment) != 2*len(tokens) {
		return nil
	}
	for i := 0; i < len(tokens); i += 2 {
		start := strings.Index(translated, tokens[i]) + len(tokens[i])
		end := strings.Index(translated, tokens[i+1])
		span := translationPlaceholderPattern.ReplaceAllString(translated[start:end], "")
		if segment.Placeholders[tokens[i]] != segment.Placeholders[tokens[i+1]] || !containsLetterOrNumber(span) {
			return nil
		}
	}
	return tokens
}
