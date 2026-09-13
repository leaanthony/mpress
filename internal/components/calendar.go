package components

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// Calendar renders an interactive month-view calendar for blog posts.
// Usage: :::calendar or :::calendar{month="2026-03" style="compact" today="2026-03-14"}
// Displays a month grid. Blog post dates are highlighted via data attributes.
// No JS framework — pure vanilla JS + CSS.
type Calendar struct {
	Meta    map[string]string
	Content string
}

func (c *Calendar) Parse(content string) error {
	c.Content = content
	return nil
}

func (c *Calendar) Render() (string, error) {
	// Parse month from metadata or use current month
	now := time.Now()
	year := now.Year()
	month := now.Month()
	var today time.Time
	markToday := false

	if m, ok := c.Meta["month"]; ok && m != "" {
		t, err := time.Parse("2006-01", m)
		if err == nil {
			year = t.Year()
			month = t.Month()
		}
	}
	if value := strings.TrimSpace(c.Meta["today"]); value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			today = parsed
			markToday = true
		}
	}

	style := c.Meta["style"] // "compact" or "" (default full)
	isCompact := style == "compact"

	// Generate calendar HTML
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<div class="mpress-calendar%s" data-year="%d" data-month="%d">`, compactClass(isCompact), year, int(month)))
	b.WriteString("\n")

	// Header with month/year and navigation controls
	monthName := month.String()
	b.WriteString(`  <div class="mpress-calendar-header">`)
	b.WriteString(fmt.Sprintf(`<button class="mpress-calendar-nav" data-dir="prev" aria-label="Previous month">%s</button>`, lucide("chevron-left", 16)))
	b.WriteString(fmt.Sprintf(`<span class="mpress-calendar-title">%s %d</span>`, html.EscapeString(monthName), year))
	b.WriteString(fmt.Sprintf(`<button class="mpress-calendar-nav" data-dir="next" aria-label="Next month">%s</button>`, lucide("chevron-right", 16)))
	b.WriteString("</div>\n")

	// Day headers
	b.WriteString(`  <div class="mpress-calendar-grid">`)
	b.WriteString("\n")
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for _, d := range days {
		b.WriteString(fmt.Sprintf(`    <div class="mpress-calendar-day-header">%s</div>`, d))
		b.WriteString("\n")
	}

	// Calculate first day of month and number of days
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)
	daysInMonth := lastDay.Day()

	// Day of week for first day (Monday = 0)
	weekday := int(firstDay.Weekday())
	if weekday == 0 {
		weekday = 6 // Sunday becomes 6 (Monday-based)
	} else {
		weekday-- // Shift to Monday = 0
	}

	// Empty cells before first day
	for i := 0; i < weekday; i++ {
		b.WriteString(`    <div class="mpress-calendar-cell empty"></div>`)
		b.WriteString("\n")
	}

	// Day cells. A highlight is emitted only when the document supplies an
	// explicit date, which keeps generated output reproducible.
	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%d-%02d-%02d", year, int(month), day)
		classes := "mpress-calendar-cell"
		if markToday && year == today.Year() && month == today.Month() && day == today.Day() {
			classes += " today"
		}
		b.WriteString(fmt.Sprintf(`    <div class="%s" data-date="%s"><span class="mpress-calendar-date">%d</span></div>`, classes, dateStr, day))
		b.WriteString("\n")
	}

	b.WriteString("  </div>\n")

	// Optional content (e.g., legend or description)
	if strings.TrimSpace(c.Content) != "" {
		b.WriteString("  <div class=\"mpress-calendar-content\">\n\n")
		b.WriteString(c.Content)
		b.WriteString("\n\n  </div>\n")
	}

	b.WriteString("</div>\n")

	return b.String(), nil
}

func compactClass(isCompact bool) string {
	if isCompact {
		return " mpress-calendar-compact"
	}
	return ""
}
