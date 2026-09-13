package translate

import "strings"

func localeGuidance(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if separator := strings.IndexAny(language, "-_"); separator >= 0 {
		language = language[:separator]
	}
	switch language {
	case "fr":
		return "Use natural technical French. Reorder complete sentences around placeholders when French grammar requires it. Preserve the force of must, requires, rejects, cannot, and negation. Use French spacing and sentence case. Prefer clear imperatives for procedures."
	case "zh":
		return "Use Simplified Chinese unless the requested locale explicitly identifies Traditional Chinese. Use natural Chinese word order and punctuation. Reorder placeholders with their sentence, and do not insert unnecessary spaces between CJK text and inline code. Preserve requirements and negation exactly."
	case "ja":
		return "Use natural Japanese technical writing and Japanese punctuation. Reorder placeholders with their sentence. Keep product names and code unchanged. Preserve requirements and negation exactly."
	case "ko":
		return "Use natural Korean technical writing. Reorder placeholders with their sentence. Keep product names and code unchanged. Preserve requirements and negation exactly."
	case "de":
		return "Use concise technical German. Reorder placeholders to follow German grammar. Preserve requirements, prohibitions, negation, and technical terminology exactly."
	case "es":
		return "Use neutral international technical Spanish. Reorder placeholders to follow Spanish grammar. Prefer direct imperatives for procedures and preserve requirements and negation exactly."
	case "cy":
		return "Use natural formal Welsh suitable for technical documentation. Reorder placeholders to follow Welsh grammar. Keep product names and code unchanged, and preserve requirements and negation exactly."
	default:
		return "Use natural technical language for the target locale. Reorder placeholders when target-language grammar requires it. Preserve requirements, prohibitions, certainty, and negation exactly."
	}
}
