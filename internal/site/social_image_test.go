package site

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

func TestPageSocialImageMetadata(t *testing.T) {
	for _, tc := range []struct{ name, base, pageImage, siteImage, want string }{
		{"article image", "https://example.com", "/film.webp", "", "https://example.com/film.webp"},
		{"article overrides fallback", "https://example.com", "/film.webp", "/default.png", "https://example.com/film.webp"},
		{"site fallback", "https://example.com", "", "/default.png", "https://example.com/default.png"},
		{"blank article uses fallback", "https://example.com", "  ", "/default.png", "https://example.com/default.png"},
		{"absolute article", "https://example.com", "https://cdn.example.com/film.webp", "", "https://cdn.example.com/film.webp"},
		{"absolute without base", "", "https://cdn.example.com/film.webp", "", "https://cdn.example.com/film.webp"},
		{"absolute fallback without base", "", "", "https://cdn.example.com/default.png", "https://cdn.example.com/default.png"},
		{"base path", "https://example.com/docs/", "assets/film.webp", "", "https://example.com/docs/assets/film.webp"},
		{"no image", "https://example.com", "", "", ""},
		{"relative image without base", "", "/film.webp", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Site.BaseURL = tc.base
			cfg.Site.SocialImage = tc.siteImage
			page := &content.Page{Title: "Release", Language: "en", URLPath: "blog/release"}
			page.Meta.Image = tc.pageImage
			markup, err := renderPage(templateData{Config: cfg, Page: page, Root: "../../"})
			if err != nil {
				t.Fatal(err)
			}
			got := string(markup)
			if tc.want == "" {
				if strings.Contains(got, `property="og:image"`) || strings.Contains(got, `name="twitter:image"`) || !strings.Contains(got, `name="twitter:card" content="summary"`) {
					t.Fatal("image-free page has inconsistent social metadata")
				}
				return
			}
			for _, want := range []string{`property="og:image" content="` + tc.want + `"`, `name="twitter:image" content="` + tc.want + `"`, `name="twitter:card" content="summary_large_image"`} {
				if !strings.Contains(got, want) {
					t.Errorf("missing %s", want)
				}
			}
		})
	}
}
