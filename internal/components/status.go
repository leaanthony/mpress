package components

import (
	"fmt"
	"html"
	"strings"
	"unicode"
)

// Status renders an inline service status badge.
// Usage: :::status{service="API" state="operational"}
// or: :::status{service="API" state="degraded" url="https://status.example.com"}
//
// States: operational (green), degraded (yellow), outage (red), maintenance (blue)
// Optional url links the badge to an external status page.
type Status struct {
	Meta    map[string]string
	Content string
}

func (s *Status) Parse(content string) error {
	s.Content = strings.TrimSpace(content)
	return nil
}

func (s *Status) Render() (string, error) {
	service := s.Meta["service"]
	if service == "" {
		service = "Service"
	}

	state := strings.ToLower(s.Meta["state"])
	if state == "" {
		state = "operational"
	}

	statusURL := s.Meta["url"]

	// Map state to display label and CSS class
	label, cssClass := statusInfo(state)

	var b strings.Builder

	// Wrap in link if URL provided
	if statusURL != "" {
		b.WriteString(fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener" class="mpress-status-link">`, html.EscapeString(statusURL)))
	}

	b.WriteString(fmt.Sprintf(`<span class="mpress-status mpress-status-%s" role="status" aria-label="%s: %s">`,
		html.EscapeString(cssClass),
		html.EscapeString(service),
		html.EscapeString(label)))

	// Status dot
	b.WriteString(`<span class="mpress-status-dot"></span>`)

	// Service name
	b.WriteString(fmt.Sprintf(`<span class="mpress-status-service">%s</span>`, html.EscapeString(service)))

	// Status label
	b.WriteString(fmt.Sprintf(`<span class="mpress-status-label">%s</span>`, html.EscapeString(label)))

	b.WriteString("</span>")

	if statusURL != "" {
		b.WriteString("</a>")
	}

	// Optional description content
	if s.Content != "" {
		b.WriteString(fmt.Sprintf(`<span class="mpress-status-desc">%s</span>`, html.EscapeString(s.Content)))
	}

	return b.String(), nil
}

func statusInfo(state string) (label string, cssClass string) {
	switch state {
	case "operational", "up", "ok", "healthy":
		return "Operational", "operational"
	case "degraded", "slow", "partial":
		return "Degraded", "degraded"
	case "outage", "down", "error":
		return "Outage", "outage"
	case "maintenance", "updating":
		return "Maintenance", "maintenance"
	default:
		if state == "" {
			return "Unknown", "unknown"
		}
		runes := []rune(state)
		runes[0] = unicode.ToUpper(runes[0])
		return string(runes), "unknown"
	}
}
