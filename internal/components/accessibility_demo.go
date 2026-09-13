package components

// AccessibilityDemo renders an inline, working subset of the reader controls.
// It uses the same data contracts as the navbar menu, so both surfaces share
// one browser-local preference state.
type AccessibilityDemo struct {
	Meta map[string]string
}

func (d *AccessibilityDemo) Parse(string) error { return nil }

func (d *AccessibilityDemo) Render() (string, error) {
	return `<section class="mpress-accessibility-demo" data-a11y-demo aria-label="Accessibility reader controls">
<header><div><span>Reader controls</span><strong>Make this page work for you</strong></div><button type="button" data-open-accessibility>All settings</button></header>
<div class="mpress-accessibility-tabs" role="tablist" aria-label="Reader control categories">
<button type="button" role="tab" aria-selected="true" data-a11y-demo-tab="reading">Reading</button>
<button type="button" role="tab" aria-selected="false" data-a11y-demo-tab="focus">Focus</button>
<button type="button" role="tab" aria-selected="false" data-a11y-demo-tab="vision">Vision</button>
</div>
<div class="mpress-accessibility-section" data-a11y-demo-panel="reading">
<h3>Text size</h3><div class="mpress-a11y-choice" role="radiogroup" aria-label="Demo text size"><button type="button" role="radio" data-a11y-choice="text" data-value="default">Default</button><button type="button" role="radio" data-a11y-choice="text" data-value="large">Large</button><button type="button" role="radio" data-a11y-choice="text" data-value="larger">Larger</button></div>
<h3>Site layout</h3><div class="mpress-a11y-width-mode" role="radiogroup" aria-label="Demo site layout"><button type="button" role="radio" data-a11y-choice="siteWidth" data-value="full">Full width</button><button type="button" role="radio" data-a11y-choice="siteWidth" data-value="fixed">Fixed width</button></div>
<label class="mpress-a11y-toggle"><span><strong>Readable font</strong><small>Use a clear, widely spaced font stack.</small></span><input type="checkbox" data-a11y-toggle="readable"></label>
</div>
<div class="mpress-accessibility-section" data-a11y-demo-panel="focus" hidden>
<label class="mpress-a11y-toggle"><span><strong>Focus mode</strong><small>Dim navigation until you point to it.</small></span><input type="checkbox" data-a11y-toggle="focus"></label>
<label class="mpress-a11y-toggle"><span><strong>Reduce motion</strong><small>Stop non-essential animation.</small></span><input type="checkbox" data-a11y-toggle="motion"></label>
</div>
<div class="mpress-accessibility-section" data-a11y-demo-panel="vision" hidden>
<label class="mpress-a11y-toggle"><span><strong>High contrast</strong><small>Increase text and border contrast.</small></span><input type="checkbox" data-a11y-toggle="contrast"></label>
<label class="mpress-a11y-toggle"><span><strong>Underline links</strong><small>Show links with more than colour.</small></span><input type="checkbox" data-a11y-toggle="links"></label>
</div>
</section>`, nil
}
