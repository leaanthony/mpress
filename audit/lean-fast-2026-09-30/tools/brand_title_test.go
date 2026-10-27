package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const brandTitleProbe = `(() => {
  const header = document.querySelector('body > header');
  const brand = header && header.querySelector('.brand');
  if (!header || !brand) return '';
  const box = element => {
    const rect = element.getBoundingClientRect();
    return {top: rect.top, bottom: rect.bottom, left: rect.left, right: rect.right, width: rect.width, height: rect.height};
  };
  const text = brand.querySelector(':scope > span:not(.brand-mark):not(.theme-logo)');
  const logo = brand.querySelector('.theme-logo, .brand-logo');
  const image = logo && (logo.tagName === 'IMG' ? logo : logo.querySelector('img'));
  return JSON.stringify({
    header: box(header),
    brand: box(brand),
    text: text ? box(text) : null,
    textContent: text ? text.textContent : '',
    scrollWidth: text ? text.scrollWidth : 0,
    clientWidth: text ? text.clientWidth : 0,
    logo: logo ? box(logo) : null,
    logoLoaded: Boolean(image && image.complete && image.naturalWidth > 0)
  });
})()`

type brandElementBox struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type brandLayout struct {
	Header      brandElementBox  `json:"header"`
	Brand       brandElementBox  `json:"brand"`
	Text        *brandElementBox `json:"text"`
	TextContent string           `json:"textContent"`
	ScrollWidth int              `json:"scrollWidth"`
	ClientWidth int              `json:"clientWidth"`
	Logo        *brandElementBox `json:"logo"`
	LogoLoaded  bool             `json:"logoLoaded"`
}

// Imported text-only branding must keep the site title inside the header slot
// instead of letting it wrap out of the 40px brand box over page chrome. The
// link keeps the complete title text, so its accessible name stays whole.
func TestImportedBrandTitleStaysInsideHeader(t *testing.T) {
	binary := os.Getenv("MPRESS_BRAND_BIN")
	if binary == "" {
		t.Skip("set MPRESS_BRAND_BIN to the candidate CLI")
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 4*time.Minute)
	defer timeout()

	const longTitle = "Migration reader qualification"
	longSite := buildBrandFixture(t, binary, longTitle, false)
	longServer := httptest.NewServer(http.FileServer(http.Dir(longSite)))
	defer longServer.Close()
	for _, page := range []struct {
		name   string
		route  string
		width  int64
		height int64
		hidden bool
	}{
		{"desktop-en", "/guide/", 1280, 900, false},
		{"desktop-en-narrow", "/guide/", 1024, 900, false},
		{"desktop-fr-narrow", "/fr/guide/", 1024, 900, false},
		{"mobile-en", "/guide/", 375, 667, true},
	} {
		t.Run(page.name, func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(page.width, page.height), chromedp.Navigate(longServer.URL+page.route), chromedp.WaitReady("body"))
			layout := b.brandLayout()
			if page.hidden {
				if layout.Brand.Width != 0 {
					t.Fatalf("phone header kept a visible brand: %+v", layout)
				}
				return
			}
			expectBrandInsideHeader(t, layout)
			if layout.TextContent != longTitle {
				t.Fatalf("brand lost its full title text: %q, want %q", layout.TextContent, longTitle)
			}
			if layout.ScrollWidth <= layout.ClientWidth {
				t.Fatalf("long brand title was not clipped with an ellipsis: %+v", layout)
			}
		})
	}

	t.Run("short-title", func(t *testing.T) {
		site := buildBrandFixture(t, binary, "Docs", false)
		server := httptest.NewServer(http.FileServer(http.Dir(site)))
		defer server.Close()
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(1280, 900), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
		layout := b.brandLayout()
		expectBrandInsideHeader(t, layout)
		if layout.TextContent != "Docs" {
			t.Fatalf("short brand title %q changed, want %q", layout.TextContent, "Docs")
		}
		if layout.ScrollWidth > layout.ClientWidth {
			t.Fatalf("short brand title was clipped: %+v", layout)
		}
	})

	t.Run("image-logo", func(t *testing.T) {
		site := buildBrandFixture(t, binary, longTitle, true)
		server := httptest.NewServer(http.FileServer(http.Dir(site)))
		defer server.Close()
		for _, width := range []int64{1280, 1024} {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(width, 900), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
			layout := b.brandLayout()
			if layout.Text != nil {
				t.Fatalf("logo brand rendered a text title: %+v", layout)
			}
			if layout.Logo == nil || !layout.LogoLoaded {
				t.Fatalf("brand logo image did not load: %+v", layout)
			}
			if math.Abs(layout.Logo.Width-188.45) > .5 || math.Abs(layout.Logo.Height-40) > .5 {
				t.Fatalf("brand logo size changed to %gx%g, want 188.45x40", layout.Logo.Width, layout.Logo.Height)
			}
			if layout.Logo.Top < layout.Header.Top || layout.Logo.Bottom > layout.Header.Bottom {
				t.Fatalf("brand logo left the header: %+v", layout)
			}
		}
	})
}

func newBrandPage(t *testing.T, ctx context.Context) browserActions {
	t.Helper()
	page, closePage := chromedp.NewContext(ctx)
	t.Cleanup(closePage)
	return browserActions{t: t, ctx: page}
}

func (b *browserActions) brandLayout() brandLayout {
	b.t.Helper()
	var raw string
	b.js(brandTitleProbe, &raw)
	var layout brandLayout
	if err := json.Unmarshal([]byte(raw), &layout); err != nil {
		b.t.Fatalf("read brand layout: %v (%q)", err, raw)
	}
	return layout
}

// expectBrandInsideHeader checks that both the link and its visible title stay
// within the header and within the link's allotted brand box.
func expectBrandInsideHeader(t *testing.T, layout brandLayout) {
	t.Helper()
	if layout.Text == nil {
		t.Fatalf("text-only brand has no title element: %+v", layout)
	}
	if layout.Brand.Top < layout.Header.Top || layout.Brand.Bottom > layout.Header.Bottom || layout.Brand.Left < layout.Header.Left || layout.Brand.Right > layout.Header.Right {
		t.Fatalf("brand left the header: %+v", layout)
	}
	text := *layout.Text
	if text.Top < layout.Header.Top || text.Bottom > layout.Header.Bottom {
		t.Fatalf("title overflows the header vertically: %+v", layout)
	}
	if text.Left < layout.Brand.Left-.5 || text.Right > layout.Brand.Right+.5 {
		t.Fatalf("title overflows the brand box horizontally: %+v", layout)
	}
}

func buildBrandFixture(t *testing.T, binary, title string, withLogo bool) string {
	t.Helper()
	source := t.TempDir()
	project := t.TempDir()
	logoConfig := ""
	if withLogo {
		logoConfig = ",\n  logo: { light: 'public/logo-light.svg', dark: 'public/logo-dark.svg' }"
	}
	write := func(root, name, body string) {
		t.Helper()
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "astro.config.mjs", fmt.Sprintf(`starlight({
  title: %q,
  locales: { root: { label: 'English', lang: 'en' }, fr: { label: 'Français', lang: 'fr' } },
  sidebar: [{ label: 'Guide', link: '/guide/' }]%s
})
`, title, logoConfig))
	write(source, "src/content/docs/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	write(source, "src/content/docs/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n\nBrand fixture page.\n")
	write(source, "src/content/docs/fr/index.md", "---\ntitle: Accueil\n---\n\n# Accueil\n")
	write(source, "src/content/docs/fr/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n")
	write(source, "public/logo-light.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="160" height="40" viewBox="0 0 160 40"><rect width="160" height="40" rx="6" fill="#4263eb"/><circle cx="20" cy="20" r="12" fill="#ffffff"/></svg>`)
	write(source, "public/logo-dark.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="160" height="40" viewBox="0 0 160 40"><rect width="160" height="40" rx="6" fill="#17181c"/><circle cx="20" cy="20" r="12" fill="#ffffff"/></svg>`)

	run := func(dir string, args ...string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = dir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
		}
	}
	run(project, "import", "--from", "starlight", source, "--output", project)
	run(project, "build", "--strict")
	site := filepath.Join(project, "site")
	if _, err := os.Stat(filepath.Join(site, "guide", "index.html")); err != nil {
		t.Fatalf("imported fixture was not published: %v", err)
	}
	return site
}
