package components

import (
	"fmt"
	"html"
	"strings"
)

// Release renders structured release notes with version, date, type, and sections.
// Usage: :::release{version="2.0" date="2026-03-26" type="major"}
// ### Highlights
// - New plugin system
// - 10x faster builds
// ### Breaking Changes
// - Config format changed (see migration guide below)
// ### New Features
// - Dark mode support
// - Blog engine
// ### Bug Fixes
// - Fixed sidebar scroll
// ### Deprecations
// - Old API deprecated (will be removed in v3)
// :::
type Release struct {
	Meta     map[string]string
	Sections []ReleaseSection
	Content  string
}

type ReleaseSection struct {
	Title string
	Items []string
	Raw   string // raw content for non-list sections
}

func (r *Release) Parse(content string) error {
	r.Content = content

	locs := stepHeaderRegex.FindAllStringSubmatchIndex(content, -1)

	if len(locs) == 0 {
		// No sections — treat as single content block
		items := parseReleaseItems(strings.TrimSpace(content))
		if len(items) > 0 {
			r.Sections = append(r.Sections, ReleaseSection{Title: "", Items: items})
		}
		return nil
	}

	for i, loc := range locs {
		title := content[loc[2]:loc[3]]
		contentStart := loc[1]
		var contentEnd int
		if i+1 < len(locs) {
			contentEnd = locs[i+1][0]
		} else {
			contentEnd = len(content)
		}
		sectionContent := strings.TrimSpace(content[contentStart:contentEnd])
		items := parseReleaseItems(sectionContent)

		r.Sections = append(r.Sections, ReleaseSection{
			Title: title,
			Items: items,
			Raw:   sectionContent,
		})
	}

	return nil
}

func parseReleaseItems(content string) []string {
	var items []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			items = append(items, strings.TrimSpace(line[2:]))
		}
	}
	return items
}

func (r *Release) Render() (string, error) {
	version := r.Meta["version"]
	title := strings.TrimSpace(r.Meta["title"])
	date := r.Meta["date"]
	releaseType := strings.ToLower(r.Meta["type"]) // major, minor, patch
	latest := strings.EqualFold(strings.TrimSpace(r.Meta["latest"]), "true")

	versionLabel := "Release"
	if version != "" {
		versionLabel = version
		if !strings.HasPrefix(strings.ToLower(versionLabel), "v") {
			versionLabel = "v" + versionLabel
		}
	}

	typeClass := ""
	badgeLabel := ""
	switch releaseType {
	case "major":
		typeClass = " mpress-release-major"
		badgeLabel = "Major release"
	case "minor":
		typeClass = " mpress-release-minor"
		badgeLabel = "Minor release"
	case "patch":
		typeClass = " mpress-release-patch"
		badgeLabel = "Patch release"
	}
	if latest {
		badgeLabel = "Latest release"
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<article class="mpress-release%s">`, typeClass))
	b.WriteString("\n")

	// Header
	b.WriteString(`  <header class="mpress-release-header">`)
	b.WriteString("\n")
	b.WriteString(`    <div class="mpress-release-title-group">`)
	b.WriteString("\n")
	if badgeLabel != "" {
		b.WriteString(fmt.Sprintf(`      <span class="mpress-release-type">%s</span>`, html.EscapeString(badgeLabel)))
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf(`      <h3 class="mpress-release-version"><span>%s</span>`, html.EscapeString(versionLabel)))
	if title != "" {
		b.WriteString(fmt.Sprintf(`<span class="mpress-release-title">%s</span>`, html.EscapeString(title)))
	}
	b.WriteString("</h3>")
	b.WriteString("\n")
	b.WriteString("    </div>\n")
	if date != "" {
		b.WriteString(fmt.Sprintf(`    <time class="mpress-release-date" datetime="%s">Released %s</time>`, html.EscapeString(date), html.EscapeString(date)))
		b.WriteString("\n")
	}
	b.WriteString("  </header>\n")
	b.WriteString("  <div class=\"mpress-release-notes\">\n")

	// Sections
	for _, section := range r.Sections {
		sectionClass := releaseSectionClass(section.Title)

		b.WriteString(fmt.Sprintf(`    <section class="mpress-release-section %s">`, sectionClass))
		b.WriteString("\n")

		if section.Title != "" {
			b.WriteString(fmt.Sprintf(`      <h4 class="mpress-release-section-title">%s</h4>`, html.EscapeString(section.Title)))
			b.WriteString("\n")
		}

		if len(section.Items) > 0 {
			b.WriteString("      <ul class=\"mpress-release-items\">\n")
			for _, item := range section.Items {
				if sectionClass == "mpress-release-assets" {
					b.WriteString(fmt.Sprintf("        <li><span class=\"mpress-release-asset-icon\">%s</span>%s</li>\n", lucide("download", 14), renderInlineMarkdown(item)))
					continue
				}
				b.WriteString(fmt.Sprintf("        <li>%s</li>\n", renderInlineMarkdown(item)))
			}
			b.WriteString("      </ul>\n")
		}

		b.WriteString("    </section>\n")
	}

	b.WriteString("  </div>\n")
	b.WriteString("</article>\n")

	return b.String(), nil
}

func releaseSectionClass(title string) string {
	lower := strings.ToLower(title)
	switch {
	case strings.Contains(lower, "highlight"):
		return "mpress-release-highlights"
	case strings.Contains(lower, "breaking"):
		return "mpress-release-breaking"
	case strings.Contains(lower, "new") || strings.Contains(lower, "feature") || strings.Contains(lower, "add"):
		return "mpress-release-features"
	case strings.Contains(lower, "fix") || strings.Contains(lower, "bug"):
		return "mpress-release-fixes"
	case strings.Contains(lower, "deprecat"):
		return "mpress-release-deprecations"
	case strings.Contains(lower, "improv") || strings.Contains(lower, "perf"):
		return "mpress-release-improvements"
	case strings.Contains(lower, "asset") || strings.Contains(lower, "download"):
		return "mpress-release-assets"
	default:
		return ""
	}
}
