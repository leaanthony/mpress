package icons

import (
	_ "embed"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// The complete upstream Starlight catalogues are kept as vendored source
// bundles. Keeping the original bundles makes upgrades mechanical and avoids
// maintaining a second, handpicked icon list in Go.
//
//go:embed vendor/starlight-icons.ts
var starlightIconBundle string

//go:embed vendor/starlight-file-icons.ts
var starlightFileIconBundle string

var starlightIconEntry = regexp.MustCompile(`(?m)^\s*(?:'([^']+)'|([A-Za-z0-9_.]+)):\s*(?:\n\s*)?'([^']*)',?\s*$`)

var starlightPaths = loadStarlightIcons(starlightIconBundle, starlightFileIconBundle)

// Starlight renders any glyph from the complete Astro Starlight icon and file
// icon catalogues. The vendored sources and their license are in vendor/.
func Starlight(name string, size int) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "x" {
		name = "x.com"
	}
	if name == "windows" {
		name = "seti:windows"
	}
	path, ok := starlightPaths[name]
	if !ok {
		return ""
	}
	if size <= 0 {
		size = 16
	}
	className := strings.NewReplacer(".", "-", ":", "-").Replace(name)
	return fmt.Sprintf(`<svg aria-hidden="true" class="starlight-icon starlight-icon-%s" width="%d" height="%d" viewBox="0 0 24 24" fill="currentColor">%s</svg>`, html.EscapeString(className), size, size, path)
}

func loadStarlightIcons(bundles ...string) map[string]string {
	icons := make(map[string]string)
	for _, bundle := range bundles {
		for _, match := range starlightIconEntry.FindAllStringSubmatch(bundle, -1) {
			name := match[1]
			if name == "" {
				name = match[2]
			}
			icons[strings.ToLower(name)] = match[3]
		}
	}
	return icons
}
