package components

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Changelog renders release entries as a vertical timeline.
// Usage: :::changelog
// ### v1.2.0 (2026-03-20)
// #### Features
// - Added dark mode support
// #### Fixes
// - Fixed sidebar scroll
// :::
type Changelog struct {
	Meta    map[string]string
	Content string
}

func (c *Changelog) Parse(content string) error {
	c.Content = content
	return nil
}

// changelogVersionRegex matches version headings like ### v1.2.0 (2026-03-20) or ### 2026-03-20 — Title
var changelogVersionRegex = regexp.MustCompile(`(?m)^###\s+(.+)$`)

// changelogCategoryRegex matches category headings like #### Features or #### Fixes
var changelogCategoryRegex = regexp.MustCompile(`(?m)^####\s+(.+)$`)

// changelog title parsing regexes (hoisted from parseVersionTitle)
var versionDateRe = regexp.MustCompile(`(v?\d+\.\d+(?:\.\d+)?)\s*\((\d{4}-\d{2}-\d{2})\)`)
var dateTitleRe = regexp.MustCompile(`(\d{4}-\d{2}-\d{2})\s*[—–-]\s*(.+)`)
var justVersionRe = regexp.MustCompile(`^(v?\d+\.\d+(?:\.\d+)?)$`)

func (c *Changelog) Render() (string, error) {
	content := strings.TrimSpace(c.Content)
	if content == "" {
		return "", nil
	}

	// Parse entries by splitting on ### headings
	entries := parseChangelogEntries(content)

	var b strings.Builder
	b.WriteString(`<div class="mpress-changelog" role="list">`)
	b.WriteString("\n")

	for _, entry := range entries {
		b.WriteString(renderChangelogEntry(entry))
	}

	b.WriteString("</div>\n")
	return b.String(), nil
}

type changelogEntry struct {
	Title      string
	Version    string
	Date       string
	Type       string // "major", "minor", "patch", "breaking", ""
	Categories []changelogCategory
}

type changelogCategory struct {
	Name  string
	Items []string
}

func parseChangelogEntries(content string) []changelogEntry {
	// Split by ### headings
	sections := changelogVersionRegex.Split(content, -1)
	titles := changelogVersionRegex.FindAllStringSubmatch(content, -1)

	var entries []changelogEntry

	for i, title := range titles {
		body := ""
		if i+1 < len(sections) {
			body = strings.TrimSpace(sections[i+1])
		}

		entry := changelogEntry{
			Title: strings.TrimSpace(title[1]),
		}

		// Try to extract version and date from title
		// Patterns: "v1.2.0 (2026-03-20)", "2026-03-20 — Title", "v2.0"
		entry.Version, entry.Date, entry.Type = parseVersionTitle(entry.Title)

		// Parse categories within this entry
		entry.Categories = parseCategories(body)

		entries = append(entries, entry)
	}

	return entries
}

func parseVersionTitle(title string) (version, date, entryType string) {
	// Try: v1.2.0 (2026-03-20)
	if m := versionDateRe.FindStringSubmatch(title); len(m) >= 3 {
		version = m[1]
		date = m[2]
		entryType = detectVersionType(version)
		return
	}

	// Try: 2026-03-20 — Title
	if m := dateTitleRe.FindStringSubmatch(title); len(m) >= 3 {
		date = m[1]
		version = strings.TrimSpace(m[2])
		return
	}

	// Try: just a version
	if m := justVersionRe.FindStringSubmatch(title); len(m) >= 2 {
		version = m[1]
		entryType = detectVersionType(version)
		return
	}

	// Fallback: title as-is
	version = title
	return
}

func detectVersionType(version string) string {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) >= 1 {
		if parts[0] != "0" && (len(parts) < 2 || parts[1] == "0") {
			return "major"
		}
	}
	return ""
}

func parseCategories(body string) []changelogCategory {
	if body == "" {
		return nil
	}

	// Split by #### headings
	sections := changelogCategoryRegex.Split(body, -1)
	titles := changelogCategoryRegex.FindAllStringSubmatch(body, -1)

	var categories []changelogCategory

	// Content before first category heading (uncategorized items)
	if len(sections) > 0 {
		items := parseListItems(strings.TrimSpace(sections[0]))
		if len(items) > 0 {
			categories = append(categories, changelogCategory{
				Name:  "",
				Items: items,
			})
		}
	}

	for i, title := range titles {
		body := ""
		if i+1 < len(sections) {
			body = strings.TrimSpace(sections[i+1])
		}
		items := parseListItems(body)
		if len(items) > 0 {
			categories = append(categories, changelogCategory{
				Name:  strings.TrimSpace(title[1]),
				Items: items,
			})
		}
	}

	return categories
}

func parseListItems(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			items = append(items, strings.TrimSpace(line[2:]))
		}
	}
	return items
}

func renderChangelogEntry(entry changelogEntry) string {
	var b strings.Builder

	typeClass := ""
	if entry.Type == "major" {
		typeClass = " mpress-changelog-major"
	}

	b.WriteString(fmt.Sprintf(`<article class="mpress-changelog-entry%s" role="listitem">`, typeClass))
	b.WriteString("\n")

	// Header
	b.WriteString(`  <header class="mpress-changelog-header">`)
	if entry.Version != "" {
		b.WriteString(fmt.Sprintf(`<span class="mpress-changelog-version">%s</span>`, html.EscapeString(entry.Version)))
	}
	if entry.Date != "" {
		b.WriteString(fmt.Sprintf(`<time class="mpress-changelog-date" datetime="%s">%s</time>`, html.EscapeString(entry.Date), html.EscapeString(entry.Date)))
	}
	b.WriteString("</header>\n")
	b.WriteString(`  <div class="mpress-changelog-body">`)
	b.WriteString("\n")
	b.WriteString(`    <div class="mpress-changelog-content">`)
	b.WriteString("\n")

	// Categories
	for _, cat := range entry.Categories {
		badge := categoryBadgeClass(cat.Name)
		b.WriteString(fmt.Sprintf(`      <section class="mpress-changelog-section %s">`, badge))
		b.WriteString("\n")
		if cat.Name != "" {
			b.WriteString(fmt.Sprintf(`        <h4 class="mpress-changelog-category"><span aria-hidden="true"></span>%s</h4>`, html.EscapeString(cat.Name)))
			b.WriteString("\n")
		} else {
			b.WriteString(`        <h4 class="mpress-changelog-category mpress-changelog-category-visually-hidden">Changes</h4>`)
			b.WriteString("\n")
		}
		b.WriteString("        <ul class=\"mpress-changelog-items\">\n")
		for _, item := range cat.Items {
			b.WriteString(fmt.Sprintf("          <li>%s</li>\n", html.EscapeString(item)))
		}
		b.WriteString("        </ul>\n")
		b.WriteString("      </section>\n")
	}

	b.WriteString("    </div>\n")
	b.WriteString("  </div>\n")
	b.WriteString("</article>\n")

	return b.String()
}

func categoryBadgeClass(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "feature") || strings.Contains(lower, "new") || strings.Contains(lower, "add"):
		return "mpress-badge-feature"
	case strings.Contains(lower, "fix") || strings.Contains(lower, "bug"):
		return "mpress-badge-fix"
	case strings.Contains(lower, "break") || strings.Contains(lower, "deprecat") || strings.Contains(lower, "remov"):
		return "mpress-badge-breaking"
	case strings.Contains(lower, "perf") || strings.Contains(lower, "improve"):
		return "mpress-badge-improvement"
	case strings.Contains(lower, "doc"):
		return "mpress-badge-docs"
	default:
		return "mpress-badge-other"
	}
}
