package components

import (
	"fmt"
	"html"
	"strings"
)

// Testimonials renders a carousel of customer/user testimonials.
// Usage: :::testimonials
// "M-Press is incredibly fast."
// — Jane Doe, CTO at Acme Corp
// ---
// "The best documentation tool we've ever used."
// — John Smith, Lead Engineer at BigCo
// ---
// "Zero config, just works."
// — Alice, Open Source Maintainer
// :::
type Testimonials struct {
	Meta  map[string]string
	Items []Testimonial
}

type Testimonial struct {
	Quote   string
	Author  string
	Company string
}

func (t *Testimonials) Parse(content string) error {
	parts := strings.Split(content, "---")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		item := parseTestimonial(part)
		if item.Quote != "" {
			t.Items = append(t.Items, item)
		}
	}
	return nil
}

func parseTestimonial(raw string) Testimonial {
	lines := strings.Split(raw, "\n")
	var quoteLines []string
	var attribution string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Attribution line starts with — or - or –
		if strings.HasPrefix(line, "—") || strings.HasPrefix(line, "–") || (strings.HasPrefix(line, "- ") && len(quoteLines) > 0) {
			attribution = strings.TrimLeft(line, "—–- ")
			continue
		}
		// Strip surrounding quotes
		if (strings.HasPrefix(line, `"`) && strings.HasSuffix(line, `"`)) ||
			(strings.HasPrefix(line, "\u201c") && strings.HasSuffix(line, "\u201d")) {
			if strings.HasPrefix(line, `"`) {
				line = strings.TrimSuffix(strings.TrimPrefix(line, `"`), `"`)
			} else {
				line = strings.TrimSuffix(strings.TrimPrefix(line, "\u201c"), "\u201d")
			}
		} else if strings.HasPrefix(line, `"`) {
			line = strings.TrimPrefix(line, `"`)
		} else if strings.HasSuffix(line, `"`) {
			line = strings.TrimSuffix(line, `"`)
		}
		quoteLines = append(quoteLines, line)
	}

	quote := strings.Join(quoteLines, " ")
	author, company := splitAttribution(attribution)

	return Testimonial{
		Quote:   quote,
		Author:  author,
		Company: company,
	}
}

// splitAttribution splits "Jane Doe, CTO at Acme" into author and company.
func splitAttribution(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	// Look for ", " separator for company info
	if idx := strings.Index(s, ", "); idx > 0 {
		return s[:idx], s[idx+2:]
	}
	return s, ""
}

func (t *Testimonials) Render() (string, error) {
	if len(t.Items) == 0 {
		return "", nil
	}

	autoplay := t.Meta["autoplay"]
	if autoplay == "" {
		autoplay = "0"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<div class="mpress-testimonials" data-autoplay="%s">`, html.EscapeString(autoplay)))
	b.WriteString("\n")

	// Track / viewport
	b.WriteString("  <div class=\"mpress-testimonials-track\" role=\"region\" aria-label=\"Testimonials\" aria-live=\"off\">\n")
	for i, item := range t.Items {
		activeClass := ""
		if i == 0 {
			activeClass = " active"
		}
		b.WriteString(fmt.Sprintf("    <blockquote class=\"mpress-testimonial%s\" aria-hidden=\"%v\">\n", activeClass, i != 0))
		b.WriteString(fmt.Sprintf("      <p class=\"mpress-testimonial-quote\">\u201c%s\u201d</p>\n", html.EscapeString(item.Quote)))
		if item.Author != "" {
			b.WriteString("      <footer class=\"mpress-testimonial-author\">\n")
			b.WriteString(fmt.Sprintf("        <cite>%s</cite>\n", html.EscapeString(item.Author)))
			if item.Company != "" {
				b.WriteString(fmt.Sprintf("        <span class=\"mpress-testimonial-company\">%s</span>\n", html.EscapeString(item.Company)))
			}
			b.WriteString("      </footer>\n")
		}
		b.WriteString("    </blockquote>\n")
	}
	b.WriteString("  </div>\n")

	// Navigation dots
	if len(t.Items) > 1 {
		b.WriteString("  <nav class=\"mpress-testimonials-nav\" aria-label=\"Testimonial navigation\">\n")
		for i := range t.Items {
			selected := ""
			if i == 0 {
				selected = " aria-current=\"true\""
			}
			b.WriteString(fmt.Sprintf("    <button class=\"mpress-testimonials-dot\" aria-label=\"Show testimonial %d\"%s></button>\n", i+1, selected))
		}
		b.WriteString("  </nav>\n")
	}

	b.WriteString("</div>\n")
	return b.String(), nil
}
