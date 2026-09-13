package site

import (
	"bytes"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/navigation"
)

// TestCompiledNavMatchesRenderNav asserts the compiled navigation replays to
// exactly the bytes renderNav produces, across languages, roots, and current
// pages. The build relies on this equivalence to precompile the sidebar once
// per language.
func TestCompiledNavMatchesRenderNav(t *testing.T) {
	routes := map[string]*content.Page{
		"":            {},
		"guide":       {},
		"guide/setup": {},
		"reference":   {},
	}
	items := []navigation.Item{
		{Label: "Home", Link: "/"},
		{Label: "Guide <script>", Link: "guide"},
		{
			Label:     "Deep & Dark",
			Collapsed: true,
			Items: []navigation.Item{
				{Label: "Setup", Link: "guide/setup/"},
				{Label: "External", Link: "https://example.com/a?b=1&c=2"},
			},
		},
		{
			Label: "Open Section",
			Items: []navigation.Item{
				{Label: "Reference", Link: "reference"},
			},
		},
		{Label: "Just a label"},
		{Label: "Sneaky", Link: "javascript:alert(1)"},
	}
	cases := []struct {
		name                      string
		language, defaultLanguage string
		defaultAtRoot             bool
		root, current             string
	}{
		{"default-root-home", "en", "en", true, "./", ""},
		{"default-nested", "en", "en", true, "../../", "guide/setup"},
		{"default-active", "en", "en", true, "../", "guide"},
		{"translated", "fr", "en", true, "../../", "guide"},
		{"default-not-at-root", "en", "en", false, "../", "reference"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var direct bytes.Buffer
			renderer := pageRenderer{
				data: &templateData{
					Config: configWithLanguages(tc.language, tc.defaultLanguage, tc.defaultAtRoot),
					Page:   &content.Page{Language: tc.language, URLPath: tc.current},
					Root:   tc.root, CurrentRoutes: routes,
				},
				out: &direct,
			}
			renderer.renderNav(items, tc.current)

			var replayed bytes.Buffer
			compiled := compileNav(items, tc.language, tc.defaultLanguage, tc.defaultAtRoot, routes)
			compiled.render(&replayed, tc.root, tc.current)

			if direct.String() != replayed.String() {
				t.Fatalf("compiled nav diverged from renderNav\nrenderNav: %s\ncompiled:  %s", direct.String(), replayed.String())
			}
		})
	}
}

func configWithLanguages(language, defaultLanguage string, defaultAtRoot bool) (cfg config.Config) {
	cfg.Site.DefaultLanguage = defaultLanguage
	cfg.Site.Languages = []string{defaultLanguage, language}
	cfg.Site.DefaultAtRoot = defaultAtRoot
	return cfg
}
