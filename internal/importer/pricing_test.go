package importer

import (
	"strings"
	"testing"
)

// Legacy flat pricing/testimonials bodies are ordinary Markdown. Conversion
// must keep the authored plans, CTAs, features, quotes and attributions that
// the production renderer turns into card and carousel markup.
func TestMarkdownToMPDKeepsLegacyFlatPricingAndTestimonialsContent(t *testing.T) {
	source := `# Plans and proof

@pricing{cols="2"}
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

@testimonials{autoplay="0"}
"Fast and clear."
— Writer, Example team
---
"Reliable every day."
— Reader, Another team
@end
`
	native, markdown, html := renderImportedTable(t, source)
	_, _, htmlAgain := renderImportedTable(t, markdown)

	if !strings.Contains(native, `@pricing cols="2"`) {
		t.Fatalf("converted MPD lost pricing attributes:\n%s", native)
	}
	if !strings.Contains(native, `@testimonials autoplay="0"`) {
		t.Fatalf("converted MPD lost testimonials attributes:\n%s", native)
	}

	for stage, rendered := range map[string]string{"first conversion": html, "second conversion": htmlAgain} {
		if got := strings.Count(rendered, `class="mpress-pricing-card`); got != 2 {
			t.Fatalf("%s rendered %d pricing cards, want 2:\n%s", stage, got, rendered)
		}
		for _, want := range []string{
			`<h3 class="mpress-pricing-name">Personal</h3>`,
			`<div class="mpress-pricing-price">$0</div>`,
			`<a href="/guide/" class="mpress-pricing-cta">Start</a>`,
			`<h3 class="mpress-pricing-name">Team</h3>`,
			`<div class="mpress-pricing-price">$15/month</div>`,
			`<a href="/guide/" class="mpress-pricing-cta mpress-pricing-cta-primary">More</a>`,
			`<div class="mpress-pricing-badge">Recommended</div>`,
			"First feature",
			"Shared work",
		} {
			if !strings.Contains(rendered, want) {
				t.Errorf("%s lost %q:\n%s", stage, want, rendered)
			}
		}
		if got := strings.Count(rendered, `<blockquote class="mpress-testimonial`); got != 2 {
			t.Fatalf("%s rendered %d testimonial quotes, want 2:\n%s", stage, got, rendered)
		}
		for _, want := range []string{
			`data-autoplay="0"`,
			`“Fast and clear.”`,
			`<cite>Writer</cite>`,
			`<span class="mpress-testimonial-company">Example team</span>`,
			`“Reliable every day.”`,
			`<cite>Reader</cite>`,
			`<span class="mpress-testimonial-company">Another team</span>`,
		} {
			if !strings.Contains(rendered, want) {
				t.Errorf("%s lost %q:\n%s", stage, want, rendered)
			}
		}
		if got := strings.Count(rendered, `class="mpress-testimonials-dot"`); got != 2 {
			t.Errorf("%s rendered %d testimonial controls, want 2:\n%s", stage, got, rendered)
		}
	}
}

// Fenced examples stay literal source even though the directive grammar inside
// them matches the live pricing and testimonials components.
func TestMarkdownToMPDKeepsFencedPricingAndTestimonialsLiteral(t *testing.T) {
	source := "# Examples\n\n```md\n" +
		"@pricing{cols=\"2\"}\n### Personal\n$0\n- ✓ First feature\n@end\n\n" +
		"@testimonials{autoplay=\"0\"}\n\"Not a quote.\"\n— Nobody, Nowhere\n@end\n" +
		"```\n"
	native, _, html := renderImportedTable(t, source)

	if strings.Contains(html, `class="mpress-pricing-card`) {
		t.Fatalf("fenced pricing example rendered live cards:\n%s", html)
	}
	if strings.Contains(html, `class="mpress-testimonial`) {
		t.Fatalf("fenced testimonials example rendered a live carousel:\n%s", html)
	}
	for _, want := range []string{"@pricing", "@testimonials", "Not a quote."} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML lost the literal fenced text %q:\n%s", want, html)
		}
	}
	if !strings.Contains(native, "@pricing") || !strings.Contains(native, "@testimonials") {
		t.Errorf("converted MPD lost the fenced example text:\n%s", native)
	}
}
