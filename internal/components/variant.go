package components

// ActiveVariant is the currently active content variant (e.g., "react", "vue").
// Set from config (build.variant) before content processing.
// When empty, all variant blocks are shown (no filtering).
var ActiveVariant string

// Variant is a conditional content block. Content is only rendered when
// the block's name matches the ActiveVariant. When ActiveVariant is empty,
// all variant blocks render their content (passthrough mode).
//
// Usage in Markdown:
//
//	:::variant{name="react"}
//	React-specific content here.
//	:::
//
//	:::variant{name="vue"}
//	Vue-specific content here.
//	:::
type Variant struct {
	Meta    map[string]string
	Content string
}

func (v *Variant) Parse(content string) error {
	v.Content = content
	return nil
}

func (v *Variant) Render() (string, error) {
	name := v.Meta["name"]

	// No active variant set — show all variant blocks (passthrough)
	if ActiveVariant == "" {
		return v.Content, nil
	}

	// Active variant matches — render content
	if name == ActiveVariant {
		return v.Content, nil
	}

	// Non-matching variant — strip content entirely
	return "", nil
}

func init() {
	Registry["variant"] = func(m map[string]string) Component { return &Variant{Meta: m} }
}
