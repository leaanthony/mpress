package mpd

// componentClass is fixed by schema 1. Switches compile to allocation-free
// length and string dispatch, which is faster than hashing these closed sets.
type componentClass uint8

const (
	componentUnknown componentClass = iota
	componentLeaf
	componentContainer
)

func componentClassFor(name []byte) componentClass {
	switch string(name) {
	case "computed", "hr", "image", "import", "include", "input", "link", "linkcard", "qr", "video":
		return componentLeaf
	case "accessibility-demo", "actions", "api", "api-playground", "audience", "badge", "button",
		"calendar", "callout", "capabilities", "capability", "card", "cards",
		"changelog", "column", "columns", "comment", "container", "details",
		"diff", "docs-preview", "event", "explained", "file-tabs", "filetree", "footnote",
		"form", "headline", "if", "lesson", "matrix", "note", "plan",
		"preview-tabs", "pricing", "rawHTML", "release", "resource", "resources",
		"section", "status", "step", "steps", "tab", "table", "tabs", "terminal",
		"carousel", "testimonial", "testimonials", "timeline", "tutorial", "variant":
		return componentContainer
	default:
		return componentUnknown
	}
}

// IsBooleanAttribute reports whether a component attribute name may use the
// native Boolean shorthand form, where the name alone means true.
func IsBooleanAttribute(name string) bool {
	switch name {
	case "column-separators", "default", "filter", "header", "lineNumbers",
		"open", "paginate", "recommended", "search", "sort":
		return true
	default:
		return false
	}
}

func isBooleanAttribute(name []byte) bool {
	return IsBooleanAttribute(string(name))
}

func isEmojiName(name []byte) bool {
	switch string(name) {
	case "+1", "-1", "checkered_flag", "eyes", "fire", "heart",
		"information_source", "rocket", "sparkles", "tada", "warning",
		"white_check_mark", "x", "zap":
		return true
	default:
		return false
	}
}
