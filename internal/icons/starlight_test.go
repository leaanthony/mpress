package icons

import (
	"strings"
	"testing"
)

func TestCompleteStarlightIconCataloguesAreEmbedded(t *testing.T) {
	entries := len(starlightIconEntry.FindAllStringSubmatch(starlightIconBundle, -1)) + len(starlightIconEntry.FindAllStringSubmatch(starlightFileIconBundle, -1))
	if got, want := entries, 704; got != want {
		t.Fatalf("embedded Starlight bundles have %d entries, want %d", got, want)
	}
	if got, want := len(starlightPaths), 684; got != want {
		t.Fatalf("embedded Starlight catalogue has %d unique icons, want %d", got, want)
	}
	for _, name := range []string{"github", "discord", "reddit", "x.com", "rss", "apple", "linux", "seti:windows", "seti:go", "astro", "cloudflare"} {
		if starlightPaths[name] == "" {
			t.Errorf("embedded Starlight catalogue is missing %q", name)
		}
	}
}

func TestStarlightAliasesWindowsAndX(t *testing.T) {
	for name, className := range map[string]string{
		"windows": "starlight-icon-seti-windows",
		"x":       "starlight-icon-x-com",
	} {
		if got := Starlight(name, 16); !strings.Contains(got, className) {
			t.Errorf("Starlight(%q) = %q, want class %q", name, got, className)
		}
	}
}
