package site

import (
	stdhtml "html"
	"strings"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/icons"
	"github.com/leaanthony/mpress/internal/navigation"
)

// The sidebar navigation is rendered on every documentation page, but between
// pages of one language it differs only in three places: the relative root
// prefix on internal URLs, which link is the current page, and whether a
// collapsed section containing the current page is forced open. Everything
// else - escaped labels, icons, and the language-routing suffix of every URL -
// is identical. compileNav does that shared work once per language and records
// it as static HTML segments with small dynamic ops between them, so the
// per-page render replays the segments instead of re-deriving them.

type navSegKind uint8

const (
	navSegRaw    navSegKind = iota // write text verbatim
	navSegRoot                     // write the page's relative root prefix
	navSegActive                   // close the class attr, marking the current page active
	navSegOpen                     // write " open" when the current page is in the subtree
)

type navSegment struct {
	kind navSegKind
	text string              // navSegRaw: literal HTML; navSegActive: the route to match
	yes  map[string]struct{} // navSegOpen: routes inside the collapsed subtree
}

type compiledNav struct{ segments []navSegment }

// render replays the compiled navigation for one page. current must be the
// page's URL path exactly as renderNav received it, so active and open
// decisions match the uncompiled renderer byte for byte.
func (nav *compiledNav) render(out interface{ WriteString(string) (int, error) }, root, current string) {
	trimmed := strings.Trim(current, "/")
	for _, segment := range nav.segments {
		switch segment.kind {
		case navSegRaw:
			out.WriteString(segment.text)
		case navSegRoot:
			out.WriteString(root)
		case navSegActive:
			if segment.text == current {
				out.WriteString(`active" aria-current="page"`)
			} else {
				out.WriteString(`"`)
			}
		case navSegOpen:
			if trimmed != "" {
				if _, ok := segment.yes[trimmed]; ok {
					out.WriteString(` open`)
				}
			}
		}
	}
}

type navCompiler struct {
	segments []navSegment
	pending  strings.Builder
}

func (c *navCompiler) raw(text string) { c.pending.WriteString(text) }

func (c *navCompiler) text(value string) { c.raw(stdhtml.EscapeString(value)) }

func (c *navCompiler) op(segment navSegment) {
	if c.pending.Len() > 0 {
		c.segments = append(c.segments, navSegment{kind: navSegRaw, text: c.pending.String()})
		c.pending.Reset()
	}
	c.segments = append(c.segments, segment)
}

func compileNav(items []navigation.Item, language, defaultLanguage string, defaultAtRoot bool, routes map[string]*content.Page) *compiledNav {
	compiler := &navCompiler{}
	compiler.compile(items, language, defaultLanguage, defaultAtRoot, routes)
	compiler.op(navSegment{kind: navSegRaw})
	// The trailing sentinel op flushed any pending raw text; drop it.
	return &compiledNav{segments: compiler.segments[:len(compiler.segments)-1]}
}

func (c *navCompiler) compile(items []navigation.Item, language, defaultLanguage string, defaultAtRoot bool, routes map[string]*content.Page) {
	for _, item := range items {
		label, translated := item.LabelFor(language)
		if !translated && language != defaultLanguage && item.Link != "" {
			if page := routes[strings.Trim(item.Link, "/")]; page != nil && strings.TrimSpace(page.Title) != "" {
				label, translated = page.Title, true
			}
		}
		if len(item.Items) > 0 {
			c.raw(`<details`)
			if !item.Collapsed {
				c.raw(` open`)
			} else {
				c.op(navSegment{kind: navSegOpen, yes: collectNavRoutes(item.Items)})
			}
			c.raw(`><summary><span`)
			c.fallbackLabelAttributes(label, language, defaultLanguage, translated)
			c.raw(`>`)
			c.text(label)
			c.raw(`</span>`)
			c.raw(icons.Lucide("chevron-right", 14))
			c.raw(`</summary><div>`)
			c.compile(item.Items, language, defaultLanguage, defaultAtRoot, routes)
			c.raw(`</div></details>`)
			continue
		}
		if item.Link != "" {
			c.raw(`<a class="`)
			c.op(navSegment{kind: navSegActive, text: strings.Trim(item.Link, "/")})
			c.raw(` href="`)
			if strings.HasPrefix(item.Link, "http://") || strings.HasPrefix(item.Link, "https://") {
				// External URLs never receive the root prefix, so the full
				// escaped safeURL result is known at compile time.
				c.text(safeURL(item.Link))
			} else {
				// Internal URLs are root + suffix. The root prefix always
				// starts with "." and contains only dots and slashes, so
				// safeURL's scheme check can never fire on the combined value
				// and HTML escaping never alters the root. Emitting the root
				// dynamically and the escaped suffix statically therefore
				// reproduces escape(safeURL(root+suffix)) exactly.
				c.op(navSegment{kind: navSegRoot})
				c.text(navURLSuffix(item.Link, language, defaultLanguage, defaultAtRoot, routes))
			}
			c.raw(`">`)
			c.text(label)
			c.raw(`</a>`)
			continue
		}
		c.raw(`<span class="nav-label"`)
		c.fallbackLabelAttributes(label, language, defaultLanguage, translated)
		c.raw(`>`)
		c.text(label)
		c.raw(`</span>`)
	}
}

func (c *navCompiler) fallbackLabelAttributes(label, language, defaultLanguage string, translated bool) {
	if translated || language == "" || language == defaultLanguage {
		return
	}
	c.raw(` lang="`)
	c.text(defaultLanguage)
	c.raw(`" title="`)
	c.text(label + " is shown in " + strings.ToUpper(defaultLanguage) + " because a translated navigation label is not configured")
	c.raw(`"`)
}

// navURLSuffix mirrors pageNavURL for internal links with the root prefix
// removed; pageNavURL returns root + this suffix for every non-external link.
func navURLSuffix(link, language, defaultLanguage string, defaultAtRoot bool, routes map[string]*content.Page) string {
	clean := strings.Trim(link, "/")
	if language != "" && (clean == language || strings.HasPrefix(clean, language+"/")) {
		return clean + "/"
	}
	if language != "" && (language != defaultLanguage || !defaultAtRoot) && routes[clean] != nil {
		if clean == "" {
			clean = language
		} else {
			clean = language + "/" + clean
		}
	}
	if clean == "" {
		return ""
	}
	return clean + "/"
}

// collectNavRoutes gathers every non-empty trimmed link in a subtree, matching
// the set of pages navContains would report as contained.
func collectNavRoutes(items []navigation.Item) map[string]struct{} {
	routes := make(map[string]struct{})
	var walk func([]navigation.Item)
	walk = func(items []navigation.Item) {
		for _, item := range items {
			if route := strings.Trim(item.Link, "/"); route != "" {
				routes[route] = struct{}{}
			}
			walk(item.Items)
		}
	}
	walk(items)
	return routes
}
