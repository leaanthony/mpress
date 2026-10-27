package mpd_test

import (
	"strings"
	"testing"
)

// Native @plan/@testimonial children remain the structured representation and
// keep their plan body, recommended flag, CTA and attribution metadata.
func TestMarkdownRendersNativeStructuredPricingAndTestimonials(t *testing.T) {
	_, html := renderNativeMPD(t, `@pricing cols="2"
@plan
## Starter
$0
[Start](/start/)
- One project
@end
@plan recommended
## Team
$15/month
[Upgrade](/team/)
- Shared projects
@end
@end

@testimonials autoplay="0"
@testimonial author="Jane Doe" company="Acme"
"Native structured quote."
@end
@end
`)

	if got := strings.Count(html, `class="mpress-pricing-card`); got != 2 {
		t.Fatalf("rendered %d pricing cards, want 2:\n%s", got, html)
	}
	for _, want := range []string{
		`<h3 class="mpress-pricing-name">Starter</h3>`,
		`<div class="mpress-pricing-price">$0</div>`,
		`<a href="/start/" class="mpress-pricing-cta">Start</a>`,
		"One project",
		`<h3 class="mpress-pricing-name">Team</h3>`,
		`<div class="mpress-pricing-price">$15/month</div>`,
		`<a href="/team/" class="mpress-pricing-cta mpress-pricing-cta-primary">Upgrade</a>`,
		`<div class="mpress-pricing-badge">Recommended</div>`,
		"Shared projects",
		`data-autoplay="0"`,
		`“Native structured quote.”`,
		`<cite>Jane Doe</cite>`,
		`<span class="mpress-testimonial-company">Acme</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("native structured rendering lost %q:\n%s", want, html)
		}
	}
}

// A flat legacy body has no @plan/@testimonial children; the adapter must emit
// the body lines instead of dropping the container contents.
func TestMarkdownKeepsFlatPricingAndTestimonialsBodies(t *testing.T) {
	markdown, _ := renderNativeMPD(t, `@pricing cols="2"
### Personal
$0
[Start](/guide/)
- ✓ First feature
---
### Team
$15/month
[More](/guide/)
recommended
- ✓ Shared work
@end

@testimonials autoplay="0"
"Fast and clear."
— Writer, Example team
---
"Reliable every day."
— Reader, Another team
@end
`)
	for _, want := range []string{
		"### Personal",
		"$0",
		"[Start](/guide/)",
		"First feature",
		"### Team",
		"$15/month",
		"recommended",
		"Shared work",
		`"Fast and clear."`,
		"— Writer, Example team",
		`"Reliable every day."`,
		"— Reader, Another team",
	} {
		if !strings.Contains(markdown, want) {
			t.Errorf("flat body lost %q:\n%s", want, markdown)
		}
	}
}
