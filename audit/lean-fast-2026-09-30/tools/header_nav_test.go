package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type headerHit struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Width   float64 `json:"width"`
	Height  float64 `json:"height"`
	Hit     bool    `json:"hit"`
	Missing bool    `json:"missing"`
	HitHTML string  `json:"hitHTML"`
}

// TestImportedPhoneHeaderNavigation checks the phone header controls on an
// imported site with real touch dispatch. The menu toggle must receive the tap
// at its own centre instead of the search launcher, and navigation/search must
// keep working in both orders with repeated dismissals.
func TestImportedPhoneHeaderNavigation(t *testing.T) {
	binary := os.Getenv("MPRESS_HEADER_NAV_BIN")
	if binary == "" {
		t.Skip("set MPRESS_HEADER_NAV_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	imported := httptest.NewServer(http.FileServer(http.Dir(buildImportedHeaderFixture(t, binary))))
	defer imported.Close()
	for _, size := range [][2]int64{{390, 844}, {375, 667}} {
		for _, page := range []struct {
			name     string
			route    string
			title    string
			wantPath string
		}{
			{"en", "/guide/", "English guide", "/guide/"},
			{"fr", "/fr/guide/", "Guide français", "/fr/guide/"},
		} {
			t.Run(fmt.Sprintf("%s-%dx%d", page.name, size[0], size[1]), func(t *testing.T) {
				b := newBrandPage(t, ctx)
				b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(imported.URL+page.route), chromedp.WaitReady("body"))
				b.expectHeaderHit("#mpress-menu-toggle", "menu")
				b.expectHeaderHit("#mpress-search-toggle", "search")

				// Menu first, then search.
				b.openMenuByTouch()
				b.followSidebarLink()
				b.openMenuByTouch()
				b.closeMenuByEscape()
				b.openMenuByTouch()
				b.closeMenuByEscape()
				b.openMenuByTouch()
				b.closeMenuByToggle()
				b.openMenuByTouch()
				b.closeMenuByToggle()
				b.searchByTouch(page.title, "escape")
				b.searchByTouch(page.title, "close")
				b.navigateSearchResultByTouch(page.title, page.wantPath)

				// Search first, then menu.
				b.searchByTouch(page.title, "close")
				b.openMenuByTouch()
				b.followSidebarLink()
				b.openMenuByTouch()
				b.closeMenuByEscape()
			})
		}
	}

	t.Run("desktop", func(t *testing.T) {
		for _, route := range []string{"/guide/", "/fr/guide/"} {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(1280, 900), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(imported.URL+route), chromedp.WaitReady("body"))
			var menuHidden bool
			b.js(`(()=>{const m=document.querySelector('#mpress-menu-toggle');return !!m&&getComputedStyle(m).display==='none'})()`, &menuHidden)
			if !menuHidden {
				t.Fatalf("desktop header should not show the phone menu toggle on %s", route)
			}
			title := "English guide"
			if route == "/fr/guide/" {
				title = "Guide français"
			}
			b.openSearchByKeyboard()
			b.checkSearchProbe(title)
			b.run(chromedp.KeyEvent(kb.Escape))
			b.expectSearchClosed(false)
		}
	})

	t.Run("default-theme", func(t *testing.T) {
		defaultSite := httptest.NewServer(http.FileServer(http.Dir(buildDefaultHeaderFixture(t, binary))))
		defer defaultSite.Close()
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(390, 844), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(defaultSite.URL+"/guide/"), chromedp.WaitReady("body"))
		b.expectHeaderHit("#mpress-menu-toggle", "default menu")
		b.expectHeaderHit("#mpress-search-toggle", "default search")
		b.openMenuByTouch()
		b.closeMenuByEscape()
		b.searchByTouch("Search guide", "close")
		b.run(chromedp.EmulateViewport(1280, 900), chromedp.Sleep(100*time.Millisecond))
		b.openSearchByKeyboard()
		b.checkSearchProbe("Search guide")
		b.run(chromedp.KeyEvent(kb.Escape))
		b.expectSearchClosed(false)
	})
}

func newHeaderBrowser(t *testing.T) (context.Context, func()) {
	t.Helper()
	chrome := os.Getenv("MPRESS_CHROME")
	if chrome == "" {
		chrome = "/usr/bin/google-chrome"
	}
	if _, err := os.Stat(chrome); err != nil {
		t.Skipf("Chrome is required for the header navigation regression: %s", chrome)
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(chrome), chromedp.NoSandbox)
	alloc, stopAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	ctx, timeout := context.WithTimeout(ctx, 4*time.Minute)
	return ctx, func() {
		timeout()
		cancel()
		stopAlloc()
	}
}

func (b *browserActions) poll(predicate, message string) {
	b.t.Helper()
	if err := chromedp.Run(b.ctx, chromedp.Poll(predicate, nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(3*time.Second))); err != nil {
		b.t.Fatalf("%s: %v", message, err)
	}
}

// expectHeaderHit returns the centre of a header control after proving that the
// control itself, not an overlapping launcher, receives the hit there.
func (b *browserActions) expectHeaderHit(selector, label string) headerHit {
	b.t.Helper()
	var hit headerHit
	script := fmt.Sprintf(`(() => {
      const e=document.querySelector(%s);
      if(!e)return {missing:true};
      const r=e.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;
      const target=document.elementFromPoint(x,y);
      return {x,y,width:r.width,height:r.height,hit:Boolean(target&&(target===e||e.contains(target))),hitHTML:target?target.outerHTML.slice(0,140):''};
    })()`, browserQuote(selector))
	b.js(script, &hit)
	if hit.Missing {
		b.t.Fatalf("%s %q is missing", label, selector)
	}
	if hit.Width <= 0 || hit.Height <= 0 {
		b.t.Fatalf("%s %q is not visible: %+v", label, selector, hit)
	}
	if !hit.Hit {
		b.t.Fatalf("%s centre is covered by %s", label, hit.HitHTML)
	}
	return hit
}

func (b *browserActions) openMenuByTouch() {
	b.t.Helper()
	menu := b.expectHeaderHit("#mpress-menu-toggle", "menu")
	b.run(browserTap(menu.X, menu.Y), chromedp.Sleep(300*time.Millisecond))
	b.poll(`document.body.classList.contains('nav-open') && document.querySelector('#mpress-menu-toggle').getAttribute('aria-expanded')==='true'`, "touch did not open site navigation")
	b.expectHeaderHit(".sidebar nav a", "sidebar link")
}

func (b *browserActions) closeMenuByEscape() {
	b.t.Helper()
	b.run(chromedp.KeyEvent(kb.Escape))
	b.poll(`!document.body.classList.contains('nav-open') && document.activeElement===document.querySelector('#mpress-menu-toggle')`, "Escape did not close site navigation")
	b.expectHeaderHit("#mpress-menu-toggle", "menu")
	b.expectHeaderHit("#mpress-search-toggle", "search")
}

func (b *browserActions) closeMenuByToggle() {
	b.t.Helper()
	menu := b.expectHeaderHit("#mpress-menu-toggle", "menu")
	b.run(browserTap(menu.X, menu.Y), chromedp.Sleep(300*time.Millisecond))
	b.poll(`!document.body.classList.contains('nav-open') && document.querySelector('#mpress-menu-toggle').getAttribute('aria-expanded')==='false'`, "menu toggle did not close site navigation")
	b.expectHeaderHit("#mpress-menu-toggle", "menu")
	b.expectHeaderHit("#mpress-search-toggle", "search")
}

func (b *browserActions) followSidebarLink() {
	b.t.Helper()
	var link struct {
		X, Y    float64
		Href    string
		Hit     bool
		Missing bool
	}
	b.js(`(() => {
      const link=[...document.querySelectorAll('.sidebar nav a')].find(a=>{
        const r=a.getBoundingClientRect();
        return r.width>0&&r.height>0&&new URL(a.href).pathname!==location.pathname;
      });
      if(!link)return {Missing:true};
      const r=link.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;
      const target=document.elementFromPoint(x,y);
      return {X:x,Y:y,Href:new URL(link.href).pathname,Hit:Boolean(target&&(target===link||link.contains(target)))};
    })()`, &link)
	if link.Missing {
		b.t.Fatal("sidebar has no link to another page")
	}
	if !link.Hit {
		b.t.Fatalf("sidebar link %q does not receive its own centre hit", link.Href)
	}
	response, err := chromedp.RunResponse(b.ctx, browserTap(link.X, link.Y))
	if err != nil {
		b.t.Fatalf("touch on sidebar link %q did not complete a navigation: %v", link.Href, err)
	}
	if response == nil {
		b.t.Fatalf("touch on sidebar link %q returned no document response", link.Href)
	}
	b.run(chromedp.WaitReady("body"))
	b.poll(fmt.Sprintf(`location.pathname===%s && !document.body.classList.contains('nav-open')`, browserQuote(link.Href)), "touch on a sidebar link did not navigate")
	b.expectHeaderHit("#mpress-menu-toggle", "menu")
	b.expectHeaderHit("#mpress-search-toggle", "search")
}

func (b *browserActions) openSearchByTouch() {
	b.t.Helper()
	search := b.expectHeaderHit("#mpress-search-toggle", "search")
	b.run(browserTap(search.X, search.Y))
	b.waitSearchOpen()
}

func (b *browserActions) openSearchByKeyboard() {
	b.t.Helper()
	b.run(chromedp.KeyEvent("k", chromedp.KeyModifiers(input.ModifierCtrl)))
	b.waitSearchOpen()
}

func (b *browserActions) closeSearchByTouch() {
	b.t.Helper()
	close := b.expectHeaderHit(".mpress-search-close", "search close")
	b.run(browserTap(close.X, close.Y))
	b.expectSearchClosed(true)
}

func (b *browserActions) expectSearchClosed(mobile bool) {
	b.t.Helper()
	b.poll(`(() => {const o=document.querySelector('.mpress-search-overlay');return !(o&&o.classList.contains('open'))&&document.activeElement===document.querySelector('#mpress-search-toggle')})()`, "search did not close and restore launcher focus")
	b.expectHeaderHit("#mpress-search-toggle", "search")
	if mobile {
		b.expectHeaderHit("#mpress-menu-toggle", "menu")
	}
}

func (b *browserActions) searchByTouch(title, dismiss string) {
	b.t.Helper()
	b.openSearchByTouch()
	b.checkSearchProbe(title)
	if dismiss == "escape" {
		b.run(chromedp.KeyEvent(kb.Escape))
		b.expectSearchClosed(true)
		b.run(chromedp.KeyEvent(kb.Escape))
		b.expectSearchClosed(true)
		return
	}
	b.closeSearchByTouch()
}

// navigateSearchResultByTouch opens search, taps the locale result with real
// touch input, and waits for the destination document before checking its
// header controls.
func (b *browserActions) navigateSearchResultByTouch(title, wantPath string) {
	b.t.Helper()
	b.openSearchByTouch()
	b.checkSearchProbe(title)
	var result struct {
		X, Y    float64
		Href    string
		Hit     bool
		Missing bool
	}
	b.js(fmt.Sprintf(`(() => {
      const title=%s;
      const item=[...document.querySelectorAll('a.mpress-search-item')].find(a=>a.querySelector('.mpress-search-item-title')?.textContent.includes(title));
      if(!item)return {Missing:true};
      const r=item.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;
      const target=document.elementFromPoint(x,y);
      return {X:x,Y:y,Href:new URL(item.href).pathname,Hit:Boolean(target&&(target===item||item.contains(target)))};
    })()`, browserQuote(title)), &result)
	if result.Missing {
		b.t.Fatalf("search has no result titled %q", title)
	}
	if result.Href != wantPath {
		b.t.Fatalf("search result %q links to %q, want the locale route %q", title, result.Href, wantPath)
	}
	if !result.Hit {
		b.t.Fatalf("search result %q does not receive its own centre hit", title)
	}
	response, err := chromedp.RunResponse(b.ctx, browserTap(result.X, result.Y))
	if err != nil {
		b.t.Fatalf("touch on search result %q did not complete a navigation: %v", title, err)
	}
	if response == nil {
		b.t.Fatalf("touch on search result %q returned no document response", title)
	}
	b.run(chromedp.WaitReady("body"))
	b.poll(fmt.Sprintf(`location.pathname===%s && document.title.includes(%s)`, browserQuote(wantPath), browserQuote(title)), "search result did not open the locale page")
	b.expectHeaderHit("#mpress-menu-toggle", "menu")
	b.expectHeaderHit("#mpress-search-toggle", "search")
}

func buildImportedHeaderFixture(t *testing.T, binary string) string {
	t.Helper()
	source := t.TempDir()
	project := t.TempDir()
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
	write(source, "astro.config.mjs", `starlight({
  title: 'Header navigation',
  locales: { root: { label: 'English', lang: 'en' }, fr: { label: 'Français', lang: 'fr' } },
  sidebar: [{ label: 'Guide', link: '/guide/' }, { label: 'Reference', link: '/reference/' }]
})
`)
	write(source, "src/content/docs/index.md", "---\ntitle: Home\n---\n\n# Home\n")
	write(source, "src/content/docs/guide.md", "---\ntitle: English guide\n---\n\n# English guide\n\nThe quasarwidget reference.\n")
	write(source, "src/content/docs/reference.md", "---\ntitle: Reference\n---\n\n# Reference\n")
	write(source, "src/content/docs/fr/index.md", "---\ntitle: Accueil\n---\n\n# Accueil\n")
	write(source, "src/content/docs/fr/guide.md", "---\ntitle: Guide français\n---\n\n# Guide français\n\nLa référence quasarwidget.\n")
	write(source, "src/content/docs/fr/reference.md", "---\ntitle: Référence\n---\n\n# Référence\n")

	run := func(args ...string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = project
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
		}
	}
	run("import", "--from", "starlight", source, "--output", project)
	run("build", "--strict")
	return filepath.Join(project, "site")
}

func buildDefaultHeaderFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = project
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
		}
	}
	run("init", ".")
	if err := os.RemoveAll(filepath.Join(project, "content")); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		target := filepath.Join(project, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("content/_nav.yaml", "- label: Home\n  link: /\n- label: Guide\n  link: /guide/\n")
	write("content/index.md", "---\ntitle: Home\n---\n\nDefault header fixture.\n")
	write("content/guide.md", "---\ntitle: Search guide\n---\n\n## Quasarwidget details\n\nThe quasarwidget appears in this guide.\n")
	run("build", "--strict")
	return filepath.Join(project, "site")
}
