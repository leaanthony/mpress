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
  .docs-page .docs-stage > .toc {
    position: fixed; z-index: 8; top: var(--mpress-header-height, 58px); right: 0;
    left: var(--mpress-toc-left, var(--layout-sidebar-width)); display: block;
    width: auto; height: auto; max-height: calc(100dvh - var(--mpress-header-height, 58px) - var(--mpress-devbar-height, 0px));
    padding: 0 1rem; overflow: auto; border: 0; border-bottom: 1px solid var(--border);
    background: var(--surface-solid); scrollbar-gutter: auto;
  }
  .docs-page .toc > .toc-heading, .docs-page .docs-stage > .toc::after { display: none; }
  .docs-page .toc .toc-disclosure > summary {
    display: flex; min-height: 44px; align-items: center; justify-content: space-between;
    gap: .75rem; color: var(--text); font-size: 14px; font-weight: 600;
  }
  .docs-page .toc .toc-mobile-icon { display: inline-flex; }
  .docs-page .toc .toc-disclosure > nav { padding-bottom: .5rem; }
  .docs-page .toc .toc-disclosure a {
    display: flex; width: 100%%; max-width: none; min-height: 44px; align-items: center;
    margin: 0; padding-block: .5rem; white-space: normal;
  }
  .docs-page .docs-stage > main { padding-top: calc(2rem + 44px); }
  .docs-page article :is(h2, h3, h4, h5, h6) { scroll-margin-top: calc(var(--mpress-header-height, 58px) + 60px); }
  body.nav-open .docs-stage > .toc { visibility: hidden; }
}
@media (max-width: 760px) {
  .docs-stage, .docs-page-wide .docs-stage { width: 100%%; padding: 0; }
  .docs-page .docs-stage > .toc { left: 0; }
}
`, sidebarWidth, contentWidth, wideContentWidth, tocWidth, contentTOCGap,
		contentTOCGap, contentWidth, tocWidth, contentWidth, tocWidth,
		contentTOCGap, wideContentWidth, tocWidth, wideContentWidth, tocWidth,
		responsiveWidth)
	if layout.TOC == "hidden" {
		css += ".docs-stage, .docs-page-wide .docs-stage { grid-template-columns: var(--layout-stage-fill) minmax(0, var(--layout-content-width)) minmax(var(--layout-stage-fill), 1fr); }\n.docs-page .docs-stage > .toc { display: none; }\n@media (max-width: 1180px) { .docs-page .docs-stage > main { padding-top: 2.4rem; } }\n"
	}
	return css
}

func cssClamp(minimum, value, maximum string) string {
	return "clamp(" + minimum + ", " + value + ", " + maximum + ")"
}
