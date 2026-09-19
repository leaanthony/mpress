package translate

import "strings"

// Align one added standalone Markdown paragraph only when removing it restores
// BOTH complete baseline documents. Exact surrounding bytes witness structure,
// order and target identity; a duplicate paragraph can make that proof ambiguous.
func insertedParagraphAlignment(baseline, baselineTarget, current, target *Document) map[string]string {
	if baseline.Format != markdownTranslationFormat || current.Format != markdownTranslationFormat ||
		len(current.Segments) != len(baseline.Segments)+1 || len(target.Segments) != len(current.Segments) ||
		len(baselineTarget.Segments) != len(baseline.Segments) {
		return nil
	}
	inserted := -1
	for i, segment := range current.Segments {
		translated := target.Segments[i]
		if segment.ID != translated.ID || !restoresParagraphBaseline(current, segment, baseline) ||
			!restoresParagraphBaseline(target, translated, baselineTarget) {
			continue
		}
		if inserted >= 0 {
			return nil
		}
		inserted = i
	}
	if inserted < 0 {
		return nil
	}
	result := map[string]string{}
	before := 0
	for i, segment := range current.Segments {
		if i == inserted {
			continue
		}
		original, oldTarget := baseline.Segments[before], baselineTarget.Segments[before]
		translated := target.Segments[i]
		if original.ID != oldTarget.ID || segment.ID != translated.ID || segment.Kind != original.Kind ||
			translated.Kind != oldTarget.Kind || segment.SourceHash != original.SourceHash || translated.Original != oldTarget.Original {
			return nil
		}
		result[segment.ID] = original.ID
		before++
	}
	return result
}

func restoresParagraphBaseline(current *Document, segment Segment, baseline *Document) bool {
	if segment.Kind != "text" {
		return false
	}
	source := string(current.Source)
	start, end := segment.Start, segment.End
	// No list, quote, heading, component declaration or other prefix may be
	// removed along with prose. Whitespace handling is intentionally narrow.
	if start != 0 && (start < 2 || source[start-2:start] != "\n\n") {
		return false
	}
	if !strings.HasPrefix(source[end:], "\n\n") && source[end:] != "\n" && end != len(source) {
		return false
	}
	if strings.HasPrefix(source[end:], "\n\n") {
		end += 2
	} else if start >= 2 {
		start -= 2
	} else {
		return false
	}
	restored := source[:start] + source[end:]
	return normalizeMigrationSourcePath(restored) == normalizeMigrationSourcePath(string(baseline.Source))
}
