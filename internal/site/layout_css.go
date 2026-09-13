package site

import (
	"fmt"

	"github.com/leaanthony/mpress/internal/config"
)

// documentationLayoutCSS turns validated layout settings into a small set of
// CSS overrides. Keeping this separate from the user stylesheet prevents a
// configured width from becoming arbitrary CSS.
func documentationLayoutCSS(layout config.LayoutConfig) string {
	sidebarWidth := cssClamp("12rem", layout.SidebarWidth, "30rem")
	contentWidth := cssClamp("30rem", layout.ContentWidth, "100rem")
	wideContentWidth := cssClamp("30rem", layout.WideContentWidth, "120rem")
	tocWidth := cssClamp("12rem", layout.TOCWidth, "25rem")
	contentTOCGap := cssClamp(".75rem", layout.ContentTOCGap, "6rem")
	responsiveWidth := contentWidth
	if len(layout.ContentWidth) > 0 && layout.ContentWidth[len(layout.ContentWidth)-1] == '%' {
		responsiveWidth = "100%"
	}
	css := fmt.Sprintf(`
:root {
  --layout-sidebar-width: %s;
  --layout-content-width: %s;
  --layout-wide-content-width: %s;
  --layout-toc-width: %s;
  --layout-content-toc-gap: %s;
}
.docs-stage {
  --layout-stage-fill: max(%s, calc((100%% - %s - %s) / 2));
  grid-template-columns: var(--layout-stage-fill) minmax(0, %s) var(--layout-stage-fill) minmax(0, %s);
  width: 100%%;
  margin-inline: 0;
}
.docs-page-wide .docs-stage {
  --layout-stage-fill: max(%s, calc((100%% - %s - %s) / 2));
  grid-template-columns: var(--layout-stage-fill) minmax(0, %s) var(--layout-stage-fill) minmax(0, %s);
  width: 100%%;
}
@media (max-width: 1180px) {
  .docs-stage, .docs-page-wide .docs-stage {
    display: block;
    width: min(100%%, %s);
    margin-inline: auto;
    padding-inline: 1.5rem;
  }
  .toc { display: none; }
}
@media (max-width: 760px) {
  .docs-stage, .docs-page-wide .docs-stage { width: 100%%; padding: 0; }
}
`, sidebarWidth, contentWidth, wideContentWidth, tocWidth, contentTOCGap,
		contentTOCGap, contentWidth, tocWidth, contentWidth, tocWidth,
		contentTOCGap, wideContentWidth, tocWidth, wideContentWidth, tocWidth,
		responsiveWidth)
	if layout.TOC == "hidden" {
		css += ".docs-stage, .docs-page-wide .docs-stage { grid-template-columns: var(--layout-stage-fill) minmax(0, var(--layout-content-width)) minmax(var(--layout-stage-fill), 1fr); }\n.docs-stage .toc { display: none; }\n"
	}
	return css
}

func cssClamp(minimum, value, maximum string) string {
	return "clamp(" + minimum + ", " + value + ", " + maximum + ")"
}
