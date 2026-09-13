package icons

import (
	"fmt"
	"html"
	"strings"
)

// Lucide renders a small, dependency-free subset of the Lucide icon set.
// The generated markup follows Lucide's SVG contract and uses currentColor.
func Lucide(name string, size int) string {
	name = canonicalName(name)
	paths, ok := lucidePaths[name]
	if !ok {
		name = "circle"
		paths = lucidePaths[name]
	}
	if size <= 0 {
		size = 18
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-%s" aria-hidden="true">%s</svg>`, size, size, html.EscapeString(name), paths)
}

// CopyButton renders the shared, language-neutral copy control used by code
// frames and terminals. Both visual states are present in the static HTML so
// the browser only needs to toggle a class after copying.
func CopyButton(className, label string) string {
	className = html.EscapeString(strings.TrimSpace(className))
	label = html.EscapeString(strings.TrimSpace(label))
	return fmt.Sprintf(`<button type="button" class="%s" aria-label="%s" title="%s" data-copy-label="%s" data-copied-label="Copied">%s%s<span class="sr-only" data-copy-status aria-live="polite">%s</span></button>`, className, label, label, label, Lucide("copy", 16), Lucide("check", 16), label)
}

func canonicalName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if alias, ok := lucideAliases[name]; ok {
		return alias
	}
	return name
}

var lucideAliases = map[string]string{
	"→": "arrow-right", "←": "arrow-left", "✓": "check", "✔": "check",
	"✗": "x", "✘": "x", "✕": "x", "~": "minus", "—": "minus",
	"🚀": "rocket", "⚙": "settings", "▣": "package", "♥": "heart",
	"📖": "book-open", "★": "star", "folder-open": "folder",
	"right-arrow": "arrow-right", "left-arrow": "arrow-left", "open-book": "book-open",
}

var lucidePaths = map[string]string{
	"check-circle":      `<path d="M22 11.1V12a10 10 0 1 1-5.9-9.1"/><path d="m9 11 3 3L22 4"/>`,
	"chevrons-left":     `<path d="m11 17-5-5 5-5"/><path d="m18 17-5-5 5-5"/>`,
	"arrow-left":        `<path d="m12 19-7-7 7-7"/><path d="M19 12H5"/>`,
	"arrow-right":       `<path d="M5 12h14"/><path d="m12 5 7 7-7 7"/>`,
	"arrow-up-down":     `<path d="m21 16-4 4-4-4"/><path d="M17 20V4"/><path d="m3 8 4-4 4 4"/><path d="M7 4v16"/>`,
	"at-sign":           `<circle cx="12" cy="12" r="4"/><path d="M16 8v5a3 3 0 0 0 6 0v-1a10 10 0 1 0-4 8"/>`,
	"book-open":         `<path d="M12 7v14"/><path d="M3 18a1 1 0 0 1-1-1V5a2 2 0 0 1 2-2h5a3 3 0 0 1 3 3v15a3 3 0 0 0-3-3Z"/><path d="M21 18a1 1 0 0 0 1-1V5a2 2 0 0 0-2-2h-5a3 3 0 0 0-3 3v15a3 3 0 0 1 3-3Z"/>`,
	"bold":              `<path d="M14 12a4 4 0 0 0 0-8H6v8"/><path d="M15 20a4 4 0 0 0 0-8H6v8Z"/>`,
	"check":             `<path d="M20 6 9 17l-5-5"/>`,
	"chevron-down":      `<path d="m6 9 6 6 6-6"/>`,
	"chevron-left":      `<path d="m15 18-6-6 6-6"/>`,
	"chevron-right":     `<path d="m9 18 6-6-6-6"/>`,
	"chevron-up":        `<path d="m18 15-6-6-6 6"/>`,
	"circle":            `<circle cx="12" cy="12" r="10"/>`,
	"circle-info":       `<circle cx="12" cy="12" r="10"/><path d="M12 16v-4"/><path d="M12 8h.01"/>`,
	"circle-x":          `<circle cx="12" cy="12" r="10"/><path d="m15 9-6 6"/><path d="m9 9 6 6"/>`,
	"circle-user-round": `<path d="M17.925 20.056a6 6 0 0 0-11.851.001"/><circle cx="12" cy="11" r="4"/><circle cx="12" cy="12" r="10"/>`,
	"code-xml":          `<path d="m18 16 4-4-4-4"/><path d="m6 8-4 4 4 4"/><path d="m14.5 4-5 16"/>`,
	"code":              `<polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/>`,
	"columns-3":         `<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/><path d="M15 3v18"/>`,
	"copy":              `<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>`,
	"download":          `<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>`,
	"file":              `<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/>`,
	"file-text":         `<path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/><polyline points="14 2 14 8 20 8"/><line x1="16" x2="8" y1="13" y2="13"/><line x1="16" x2="8" y1="17" y2="17"/><line x1="10" x2="8" y1="9" y2="9"/>`,
	"eye":               `<path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"/><circle cx="12" cy="12" r="3"/>`,
	"folder":            `<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>`,
	"github":            `<path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3.28-.36 6.72-1.61 6.72-7.25A5.65 5.65 0 0 0 19.22 3.3 5.38 5.38 0 0 0 19.13 1S17.95.65 15 2.48a13.38 13.38 0 0 0-7 0C5.05.65 3.87 1 3.87 1a5.38 5.38 0 0 0-.09 2.3 5.65 5.65 0 0 0-1.5 3.95c0 5.63 3.44 6.88 6.72 7.25A4.8 4.8 0 0 0 9 18v4"/><path d="M9 18c-4.51 2-5-2-7-2"/>`,
	"gauge":             `<path d="m12 14 4-4"/><path d="M3.34 19a10 10 0 1 1 17.32 0"/>`,
	"git-branch":        `<line x1="6" x2="6" y1="3" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>`,
	"git-pull-request":  `<circle cx="18" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><path d="M13 6h3a2 2 0 0 1 2 2v7"/><path d="M6 9v12"/>`,
	"globe-2":           `<path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/><circle cx="12" cy="12" r="10"/>`,
	"heart":             `<path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/>`,
	"image":             `<rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/>`,
	"italic":            `<line x1="19" x2="10" y1="4" y2="4"/><line x1="14" x2="5" y1="20" y2="20"/><line x1="15" x2="9" y1="4" y2="20"/>`,
	"languages":         `<path d="m5 8 6 6"/><path d="m4 14 6-6 2-3"/><path d="M2 5h12"/><path d="M7 2h1"/><path d="m22 22-5-10-5 10"/><path d="M14 18h6"/>`,
	"link":              `<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>`,
	"list-filter":       `<path d="M3 6h18"/><path d="M7 12h10"/><path d="M10 18h4"/>`,
	"menu":              `<line x1="4" x2="20" y1="12" y2="12"/><line x1="4" x2="20" y1="6" y2="6"/><line x1="4" x2="20" y1="18" y2="18"/>`,
	"maximize-2":        `<polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" x2="14" y1="3" y2="10"/><line x1="3" x2="10" y1="21" y2="14"/>`,
	"message-circle":    `<path d="M21 15a4 4 0 0 1-4 4H8l-5 3V7a4 4 0 0 1 4-4h10a4 4 0 0 1 4 4z"/>`,
	"more-vertical":     `<circle cx="12" cy="12" r="1"/><circle cx="12" cy="5" r="1"/><circle cx="12" cy="19" r="1"/>`,
	"messages-square":   `<path d="M14 15a2 2 0 0 1-2 2H6l-4 4V5a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2z"/><path d="M18 9h2a2 2 0 0 1 2 2v9l-4-3h-2"/>`,
	"minus":             `<path d="M5 12h14"/>`,
	"monitor":           `<rect width="20" height="14" x="2" y="3" rx="2"/><line x1="8" x2="16" y1="21" y2="21"/><line x1="12" x2="12" y1="17" y2="21"/>`,
	"smartphone":        `<rect width="14" height="20" x="5" y="2" rx="2" ry="2"/><path d="M12 18h.01"/>`,
	"tablet":            `<rect width="16" height="20" x="4" y="2" rx="2" ry="2"/><path d="M12 18h.01"/>`,
	"moon":              `<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9"/>`,
	"package":           `<path d="m7.5 4.27 9 5.15"/><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/>`,
	"panel-left":        `<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/>`,
	"pencil":            `<path d="M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z"/><path d="m15 5 4 4"/>`,
	"play":              `<polygon points="6 3 20 12 6 21 6 3"/>`,
	"plus":              `<path d="M5 12h14"/><path d="M12 5v14"/>`,
	"rocket":            `<path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/><path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.87 12.87 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/><path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"/><path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"/>`,
	"rss":               `<path d="M4 11a9 9 0 0 1 9 9"/><path d="M4 4a16 16 0 0 1 16 16"/><circle cx="5" cy="19" r="1"/>`,
	"rows-3":            `<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M21 9H3"/><path d="M21 15H3"/>`,
	"search":            `<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>`,
	"save":              `<path d="M15.2 3a2 2 0 0 1 1.4.6l3.8 3.8a2 2 0 0 1 .6 1.4V19a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z"/><path d="M17 21v-8H7v8"/><path d="M7 3v5h8"/>`,
	"settings":          `<path d="M12.22 2h-.44C9.28 2 8 3.28 8 5.78v.44C8 8.72 6.72 10 4.22 10h-.44C1.28 10 0 11.28 0 13.78v.44C0 16.72 1.28 18 3.78 18h.44C6.72 18 8 19.28 8 21.78v.44C8 24.72 9.28 26 11.78 26h.44" transform="scale(.75) translate(4 3)"/><circle cx="12" cy="12" r="3"/>`,
	"star":              `<path d="M11.525 2.295a.53.53 0 0 1 .95 0l2.31 4.679a2.12 2.12 0 0 0 1.595 1.16l5.164.75a.53.53 0 0 1 .294.904l-3.736 3.638a2.12 2.12 0 0 0-.61 1.88l.882 5.14a.53.53 0 0 1-.771.56l-4.618-2.428a2.12 2.12 0 0 0-1.97 0l-4.618 2.428a.53.53 0 0 1-.77-.56l.881-5.139a2.12 2.12 0 0 0-.611-1.879L2.162 9.79a.53.53 0 0 1 .294-.906l5.165-.75a2.12 2.12 0 0 0 1.594-1.16z"/>`,
	"sun":               `<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/>`,
	"terminal":          `<polyline points="4 17 10 11 4 5"/><line x1="12" x2="20" y1="19" y2="19"/>`,
	"triangle-alert":    `<path d="m21.73 18-8-14a2 2 0 0 0-3.46 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>`,
	"upload":            `<path d="M12 3v12"/><path d="m17 8-5-5-5 5"/><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>`,
	"x":                 `<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`,
	"zap":               `<path d="M4 14a1 1 0 0 1-.78-1.63l9-11a.5.5 0 0 1 .87.43l-1.69 6.9A1 1 0 0 0 12.37 10H20a1 1 0 0 1 .78 1.63l-9 11a.5.5 0 0 1-.87-.43l1.69-6.9A1 1 0 0 0 11.63 14Z"/>`,
}
