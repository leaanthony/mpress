package icons

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed vendor/accessibility-icon-traced-monochrome.svg
var accessibilityIconSource string

// Accessibility renders the traced accessibility symbol as a decorative
// inline icon. A luminance mask makes the figure a real transparent cut-out,
// so it works on every navbar surface in light and dark themes.
func Accessibility(size int) string {
	if size <= 0 {
		size = 19
	}
	outerStart := strings.Index(accessibilityIconSource, `<circle`)
	if outerStart < 0 {
		return Lucide("circle-user-round", size)
	}
	outerRelativeEnd := strings.Index(accessibilityIconSource[outerStart:], `/>`)
	if outerRelativeEnd < 0 {
		return Lucide("circle-user-round", size)
	}
	outerEnd := outerStart + outerRelativeEnd + 2
	figureStart := strings.Index(accessibilityIconSource[outerEnd:], `<g`)
	if figureStart < 0 {
		return Lucide("circle-user-round", size)
	}
	figureStart += outerEnd
	figureRelativeEnd := strings.Index(accessibilityIconSource[figureStart:], `</g>`)
	if figureRelativeEnd < 0 {
		return Lucide("circle-user-round", size)
	}
	figureEnd := figureStart + figureRelativeEnd + len(`</g>`)
	figure := strings.ReplaceAll(accessibilityIconSource[figureStart:figureEnd], `fill="#fff"`, `fill="black"`)
	outer := accessibilityIconSource[outerStart:outerEnd]
	outer = strings.Replace(outer, `fill="#111"`, `fill="currentColor" mask="url(#mpress-accessibility-cutout)"`, 1)
	body := `<defs><mask id="mpress-accessibility-cutout" maskUnits="userSpaceOnUse" x="0" y="0" width="512" height="512" mask-type="luminance"><rect width="512" height="512" fill="white"/>` + figure + `</mask></defs>` + outer
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 512 512" class="mpress-accessibility-icon" aria-hidden="true">%s</svg>`, size, size, body)
}
