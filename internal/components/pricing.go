package components

import (
	"fmt"
	"html"
	"strings"
)

// Pricing renders a pricing comparison table with plan cards.
// Usage: :::pricing{cols="3"}
// ### Free
// $0/month
// [Get Started](/signup)
// - ✓ 1 project
// - ✓ Basic search
// - ✗ Custom domain
// - ✗ Priority support
// ---
// ### Team
// $29/month
// [Start Trial](/trial)
// recommended
// - ✓ Unlimited projects
// - ✓ Advanced search
// - ✓ Custom domain
// - ✗ Priority support
// ---
// ### Enterprise
// Custom
// [Contact Sales](/contact)
// - ✓ Everything in Team
// - ✓ SSO / SAML
// - ✓ Custom domain
// - ✓ Priority support
// :::
type Pricing struct {
	Meta  map[string]string
	Plans []PricingPlan
}

type PricingPlan struct {
	Name        string
	Price       string
	CTALabel    string
	CTALink     string
	Recommended bool
	Features    []PricingFeature
}

type PricingFeature struct {
	Text     string
	Included bool // true = ✓, false = ✗
	Partial  bool // ~ = partial
}

func (p *Pricing) Parse(content string) error {
	// Split plans by ---
	sections := strings.Split(content, "\n---\n")

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		plan := parsePricingPlan(section)
		if plan.Name != "" {
			p.Plans = append(p.Plans, plan)
		}
	}

	return nil
}

func parsePricingPlan(section string) PricingPlan {
	var plan PricingPlan
	lines := strings.Split(section, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Plan name from ### heading
		if strings.HasPrefix(line, "### ") {
			plan.Name = strings.TrimPrefix(line, "### ")
			continue
		}

		// "recommended" marker
		if strings.ToLower(line) == "recommended" || strings.ToLower(line) == "popular" || strings.ToLower(line) == "best value" {
			plan.Recommended = true
			continue
		}

		// CTA button: [Label](url)
		if strings.HasPrefix(line, "[") && strings.Contains(line, "](") {
			// Parse markdown link
			labelEnd := strings.Index(line, "](")
			if labelEnd > 1 {
				plan.CTALabel = line[1:labelEnd]
				urlStart := labelEnd + 2
				urlEnd := strings.Index(line[urlStart:], ")")
				if urlEnd > 0 {
					plan.CTALink = line[urlStart : urlStart+urlEnd]
				}
			}
			continue
		}

		// Feature items: - ✓ text, - ✗ text, - ~ text
		if strings.HasPrefix(line, "- ") {
			featureText := strings.TrimPrefix(line, "- ")
			feature := PricingFeature{}

			if strings.HasPrefix(featureText, "✓ ") || strings.HasPrefix(featureText, "✔ ") {
				feature.Text = strings.TrimPrefix(strings.TrimPrefix(featureText, "✓ "), "✔ ")
				feature.Included = true
			} else if strings.HasPrefix(featureText, "✗ ") || strings.HasPrefix(featureText, "✘ ") || strings.HasPrefix(featureText, "✕ ") {
				feature.Text = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(featureText, "✗ "), "✘ "), "✕ ")
				feature.Included = false
			} else if strings.HasPrefix(featureText, "~ ") {
				feature.Text = strings.TrimPrefix(featureText, "~ ")
				feature.Partial = true
			} else {
				feature.Text = featureText
				feature.Included = true // default to included
			}

			plan.Features = append(plan.Features, feature)
			continue
		}

		// Price line (anything that starts with $ or contains /month, /year, or is "Custom", "Free")
		if strings.HasPrefix(line, "$") || strings.Contains(strings.ToLower(line), "/month") ||
			strings.Contains(strings.ToLower(line), "/year") || strings.ToLower(line) == "custom" ||
			strings.ToLower(line) == "free" {
			plan.Price = line
			continue
		}

		// If we don't have a price yet, treat as price
		if plan.Price == "" && plan.Name != "" {
			plan.Price = line
		}
	}

	return plan
}

func (p *Pricing) Render() (string, error) {
	if len(p.Plans) == 0 {
		return "", nil
	}

	cols := p.Meta["cols"]
	if cols == "" {
		cols = fmt.Sprintf("%d", len(p.Plans))
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<div class="mpress-pricing" style="--pricing-cols: %s">`, html.EscapeString(cols)))
	b.WriteString("\n")

	for _, plan := range p.Plans {
		recClass := ""
		if plan.Recommended {
			recClass = " mpress-pricing-recommended"
		}

		b.WriteString(fmt.Sprintf(`  <div class="mpress-pricing-card%s">`, recClass))
		b.WriteString("\n")

		// Recommended badge
		if plan.Recommended {
			b.WriteString(`    <div class="mpress-pricing-badge">Recommended</div>`)
			b.WriteString("\n")
		}

		// Plan name
		b.WriteString(fmt.Sprintf(`    <h3 class="mpress-pricing-name">%s</h3>`, html.EscapeString(plan.Name)))
		b.WriteString("\n")

		// Price
		if plan.Price != "" {
			b.WriteString(fmt.Sprintf(`    <div class="mpress-pricing-price">%s</div>`, html.EscapeString(plan.Price)))
			b.WriteString("\n")
		}

		// CTA button
		if plan.CTALink != "" {
			btnClass := "mpress-pricing-cta"
			if plan.Recommended {
				btnClass += " mpress-pricing-cta-primary"
			}
			b.WriteString(fmt.Sprintf(`    <a href="%s" class="%s">%s</a>`,
				html.EscapeString(plan.CTALink), btnClass, html.EscapeString(plan.CTALabel)))
			b.WriteString("\n")
		}

		// Features
		if len(plan.Features) > 0 {
			b.WriteString("    <ul class=\"mpress-pricing-features\">\n")
			for _, f := range plan.Features {
				var icon string
				var featureClass string
				if f.Partial {
					icon = `<span class="mpress-pricing-partial" aria-label="Partial">` + lucide("minus", 15) + `</span>`
					featureClass = "partial"
				} else if f.Included {
					icon = `<span class="mpress-pricing-check" aria-label="Included">` + lucide("check", 15) + `</span>`
					featureClass = "included"
				} else {
					icon = `<span class="mpress-pricing-cross" aria-label="Not included">` + lucide("x", 15) + `</span>`
					featureClass = "excluded"
				}
				b.WriteString(fmt.Sprintf(`      <li class="mpress-pricing-feature %s">%s %s</li>`,
					featureClass, icon, html.EscapeString(f.Text)))
				b.WriteString("\n")
			}
			b.WriteString("    </ul>\n")
		}

		b.WriteString("  </div>\n")
	}

	b.WriteString("</div>\n")
	return b.String(), nil
}
