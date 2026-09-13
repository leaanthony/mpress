package components

import (
	"fmt"
	"html"
	"strings"
)

// Matrix renders a feature comparison table with styled indicators.
// Usage: :::matrix{highlight="M-Press"}
// | Feature | M-Press | Hugo | Docusaurus |
// |---------|---------|------|------------|
// | Zero config | ✓ | ~ | ✗ |
// | Fast builds | ✓ | ✓ | ✗ |
// | No JS required | ✓ | ✓ | ✗ |
// :::
//
// Indicators: ✓ or yes or Y → green check, ✗ or no or N → red cross,
// ~ or partial → yellow partial, anything else → text as-is.
type Matrix struct {
	Meta    map[string]string
	Content string
}

func (m *Matrix) Parse(content string) error {
	m.Content = content
	return nil
}

func (m *Matrix) Render() (string, error) {
	content := strings.TrimSpace(m.Content)
	if content == "" {
		return "", nil
	}

	highlight := m.Meta["highlight"] // Column to highlight

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return "<p>Matrix requires at least a header row and separator.</p>", nil
	}

	// Parse header row
	headers := parseTableRow(lines[0])
	if len(headers) < 2 {
		return "<p>Matrix requires at least 2 columns.</p>", nil
	}

	// Skip separator row (line with |---|---|)
	dataStart := 1
	if dataStart < len(lines) && isTableSeparator(lines[dataStart]) {
		dataStart = 2
	}

	var b strings.Builder
	b.WriteString(`<div class="mpress-matrix-wrapper">`)
	b.WriteString("\n")
	b.WriteString(`<table class="mpress-matrix" role="grid">`)
	b.WriteString("\n")

	// Header
	b.WriteString("  <thead>\n    <tr>\n")
	for i, h := range headers {
		h = strings.TrimSpace(h)
		classes := "mpress-matrix-header"
		if highlight != "" && strings.EqualFold(h, highlight) {
			classes += " mpress-matrix-highlight"
		}
		if i == 0 {
			classes += " mpress-matrix-feature"
		}
		b.WriteString(fmt.Sprintf(`      <th scope="col" class="%s">%s</th>`, classes, html.EscapeString(h)))
		b.WriteString("\n")
	}
	b.WriteString("    </tr>\n  </thead>\n")

	// Body rows
	b.WriteString("  <tbody>\n")
	for _, line := range lines[dataStart:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cells := parseTableRow(line)
		if len(cells) == 0 {
			continue
		}

		b.WriteString("    <tr>\n")
		for i, cell := range cells {
			cell = strings.TrimSpace(cell)

			if i == 0 {
				// Feature name column
				b.WriteString(fmt.Sprintf(`      <td class="mpress-matrix-feature">%s</td>`, html.EscapeString(cell)))
			} else {
				// Value column — render indicator
				indicator, class := renderIndicator(cell)
				colClass := "mpress-matrix-cell"
				if highlight != "" && i < len(headers) && strings.EqualFold(strings.TrimSpace(headers[i]), highlight) {
					colClass += " mpress-matrix-highlight"
				}
				b.WriteString(fmt.Sprintf(`      <td class="%s %s">%s</td>`, colClass, class, indicator))
			}
			b.WriteString("\n")
		}
		b.WriteString("    </tr>\n")
	}
	b.WriteString("  </tbody>\n")

	b.WriteString("</table>\n")
	b.WriteString("</div>\n")

	return b.String(), nil
}

func parseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	var cells []string
	for _, p := range parts {
		cells = append(cells, strings.TrimSpace(p))
	}
	return cells
}

func isTableSeparator(line string) bool {
	line = strings.TrimSpace(line)
	return strings.Contains(line, "---") || strings.Contains(line, "===")
}

func renderIndicator(cell string) (string, string) {
	lower := strings.ToLower(strings.TrimSpace(cell))
	switch lower {
	case "✓", "yes", "y", "true", "✔", "✅":
		return `<span class="mpress-matrix-check" aria-label="Yes">` + lucide("check", 15) + `</span>`, "mpress-matrix-yes"
	case "✗", "no", "n", "false", "✘", "❌", "x":
		return `<span class="mpress-matrix-cross" aria-label="No">` + lucide("x", 15) + `</span>`, "mpress-matrix-no"
	case "~", "partial", "some", "limited":
		return `<span class="mpress-matrix-partial" aria-label="Partial">` + lucide("minus", 15) + `</span>`, "mpress-matrix-partial-cell"
	case "":
		return `<span class="mpress-matrix-empty" aria-label="Not applicable">` + lucide("minus", 15) + `</span>`, "mpress-matrix-empty-cell"
	default:
		return html.EscapeString(cell), ""
	}
}
