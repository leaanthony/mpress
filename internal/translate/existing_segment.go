package translate

import (
	"fmt"
	"sort"
	"strings"
)

// Reuse the parsed target's inline boundaries. Literal underscores and stars
// in prose must not be counted as formatting delimiters with the same spelling.
func prepareExistingSegment(source, target Segment) (string, error) {
	if source.Original == target.Original {
		return source.Text, nil
	}
	if !strings.Contains(source.Text, "⟪MPRESS_INLINE_") || target.Text == "" {
		return prepareExisting(source, target.Original)
	}
	available := map[string][]string{}
	for _, token := range orderedPlaceholderTokens(source) {
		value := source.Placeholders[token]
		available[value] = append(available[value], token)
	}
	counts := map[string]int{}
	var replacements []string
	for _, token := range orderedPlaceholderTokens(target) {
		value := target.Placeholders[token]
		candidates, exists := available[value]
		if !exists {
			replacements = append(replacements, token, value)
			continue
		}
		index := counts[value]
		counts[value]++
		if index >= len(candidates) {
			return "", fmt.Errorf("existing translation for %s does not preserve %s", source.ID, value)
		}
		replacements = append(replacements, token, candidates[index])
	}
	for value, tokens := range available {
		if counts[value] != len(tokens) {
			return "", fmt.Errorf("existing translation for %s does not preserve %s", source.ID, value)
		}
	}
	return strings.NewReplacer(replacements...).Replace(target.Text), nil
}

func orderedPlaceholderTokens(segment Segment) []string {
	tokens := make([]string, 0, len(segment.Placeholders))
	for token := range segment.Placeholders {
		tokens = append(tokens, token)
	}
	sort.Slice(tokens, func(i, j int) bool {
		return strings.Index(segment.Text, tokens[i]) < strings.Index(segment.Text, tokens[j])
	})
	return tokens
}
