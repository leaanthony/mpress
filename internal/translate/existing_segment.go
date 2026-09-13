package translate

import (
	"fmt"
	"sort"
	"strings"
)

func validateMPDInlineProtection(source, target *Document) error {
	targets := make(map[string]Segment, len(target.Segments))
	for _, segment := range target.Segments {
		targets[segment.ID] = segment
	}
	for _, segment := range source.Segments {
		if !strings.Contains(segment.Text, "⟪MPRESS_INLINE_") {
			continue
		}
		if _, err := prepareExistingSegment(segment, targets[segment.ID]); err != nil {
			return fmt.Errorf("translation changed protected inline content: %w", err)
		}
	}
	return nil
}

// Reuse the parsed target's inline boundaries. Literal underscores and stars
// in prose must not be counted as formatting delimiters with the same spelling.
func prepareExistingSegment(source, target Segment) (string, error) {
	if source.Original == target.Original {
		return source.Text, nil
	}
	if !strings.Contains(source.Text, "⟪MPRESS_INLINE_") || target.Text == "" {
		return prepareExisting(source, target.Original)
	}
	ranges := make([]mpdProtectedRange, 0, len(target.Placeholders))
	offset := 0
	for _, token := range orderedPlaceholderTokens(target) {
		value := target.Placeholders[token]
		start := strings.Index(target.Text, token) + offset
		ranges = append(ranges, mpdProtectedRange{start, start + len(value)})
		offset += len(value) - len(token)
	}
	return prepareExistingRanges(source, target.Original, ranges)
}

func withinProtectedRanges(start, end int, ranges []mpdProtectedRange) bool {
	if ranges == nil {
		return true
	}
	for _, span := range ranges {
		if start == span.start && end == span.end {
			return true
		}
	}
	return false
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
