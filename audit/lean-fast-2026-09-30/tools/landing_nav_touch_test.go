package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

// TestLandingPrimaryNavigationTouch checks the compact primary navigation on a
// landing page built from more configured header links than fit on a phone.
// Readers must open it by touch, reach the last configured link with real
// swipes that scroll only the dropdown, follow that link, return, and keep
// scrolling the document after dismissing the dropdown by Escape or outside tap.
func TestLandingPrimaryNavigationTouch(t *testing.T) {
	binary := os.Getenv("MPRESS_HEADER_NAV_BIN")
	if binary == "" {
		t.Skip("set MPRESS_HEADER_NAV_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	site := buildLandingNavFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {390, 844}, {375, 375}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/"), chromedp.WaitReady("body"))
			b.openPrimaryNavigation()

			var before, after struct {
				Nav, Body float64
				Links     int
			}
			b.js(`(() => {
              const nav=document.querySelector('#mpress-primary-navigation');
              return {Nav:nav.scrollTop,Body:document.scrollingElement.scrollTop,Links:nav.querySelectorAll('a').length};
            })()`, &before)
			if before.Links != 20 {
				t.Fatalf("primary navigation has %d links, want 20 configured links", before.Links)
			}
			last := b.reachByTouch("#mpress-primary-navigation a:last-child", func() browserTouchTarget {
				var target browserTouchTarget
				b.js(`(() => {
                  const nav=document.querySelector('#mpress-primary-navigation');
                  const link=nav&&nav.querySelector('a:last-child');
                  if(!link)return {Ready:false};
                  const nr=nav.getBoundingClientRect(),r=link.getBoundingClientRect();
                  const top=nr.top+nav.clientTop,bottom=top+nav.clientHeight;
                  const x=r.x+r.width/2,y=r.y+r.height/2;
                  const hit=document.elementFromPoint(x,y);
                  return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:top,ClipBottom:bottom,SwipeX:nr.x+nr.width/2,
                    Ready:Boolean(hit&&(hit===link||link.contains(hit))&&r.top>=top&&r.bottom<=bottom)};
                })()`, &target)
				return target
			})
			b.js(`(() => {
              const nav=document.querySelector('#mpress-primary-navigation');
              return {Nav:nav.scrollTop,Body:document.scrollingElement.scrollTop};
            })()`, &after)
			if after.Nav <= before.Nav {
				t.Fatalf("swipes did not scroll the dropdown: nav scrollTop %g, want > %g", after.Nav, before.Nav)
			}
			if after.Body != before.Body {
				t.Fatalf("swiping inside the dropdown moved the document: scrollY %g, want %g", after.Body, before.Body)
			}

			response, err := chromedp.RunResponse(b.ctx, browserTap(last.X, last.Y))
			if err != nil {
				t.Fatalf("touch on the last configured link did not complete a navigation: %v", err)
			}
			if response == nil {
				t.Fatal("touch on the last configured link returned no document response")
			}
			b.run(chromedp.WaitReady("body"))
			b.poll(`location.pathname==='/guide/' && document.querySelector('main').textContent.includes('Every configured header link lands here')`, "last configured link did not open the destination page")

			back := b.tappableAnchor("Back home")
			response, err = chromedp.RunResponse(b.ctx, browserTap(back.X, back.Y))
			if err != nil || response == nil {
				t.Fatalf("touch on the back link did not complete a navigation: %v", err)
			}
			b.run(chromedp.WaitReady("body"))
			b.poll(`location.pathname==='/' && document.body.classList.contains('landing-page')`, "back link did not return to the landing page")

			b.openPrimaryNavigation()
			b.run(chromedp.KeyEvent(kb.Escape))
			b.poll(`!document.querySelector('#mpress-primary-navigation').classList.contains('primary-open') && document.activeElement===document.querySelector('#mpress-primary-toggle')`, "Escape did not close the dropdown and focus the visible trigger")
			b.expectHeaderHit("#mpress-primary-toggle", "primary navigation toggle")

			b.openPrimaryNavigation()
			b.run(browserTap(5, float64(size[1])-10))
			b.poll(`!document.querySelector('#mpress-primary-navigation').classList.contains('primary-open') && document.querySelector('#mpress-primary-toggle').getAttribute('aria-expanded')==='false'`, "outside tap did not close the primary navigation")

			var top float64
			b.js(`document.scrollingElement.scrollTop`, &top)
			b.run(swipe(float64(size[0])/2, float64(size[1])*.8, 0, -float64(size[1])*.4))
			b.poll(fmt.Sprintf(`document.scrollingElement.scrollTop>%g`, top), "body swipe after closing the dropdown did not advance the document")
		})
	}

	t.Run("desktop", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(1280, 900), chromedp.Navigate(server.URL+"/"), chromedp.WaitReady("body"))
		var desktop struct {
			Toggle, Nav string
			Links       int
			Visible     int
			Hit         bool
		}
		b.js(`(() => {
          const toggle=document.querySelector('#mpress-primary-toggle');
          const nav=document.querySelector('#mpress-primary-navigation');
          const links=[...nav.querySelectorAll('a')];
          const first=links[0],r=first.getBoundingClientRect();
          const target=document.elementFromPoint(r.x+r.width/2,r.y+r.height/2);
          return {Toggle:getComputedStyle(toggle).display,Nav:getComputedStyle(nav).display,Links:links.length,
            Visible:links.filter(a=>a.getBoundingClientRect().width>0).length,Hit:Boolean(target&&(target===first||first.contains(target)))};
        })()`, &desktop)
		if desktop.Toggle != "none" {
			t.Fatalf("desktop header still shows the primary navigation toggle: %+v", desktop)
		}
		if desktop.Nav == "none" || desktop.Links != 20 || desktop.Visible != 20 || !desktop.Hit {
			t.Fatalf("desktop header lost its direct configured links: %+v", desktop)
		}
	})
}

func (b *browserActions) openPrimaryNavigation() {
	b.t.Helper()
	toggle := b.expectHeaderHit("#mpress-primary-toggle", "primary navigation toggle")
	if toggle.Width < 44 || toggle.Height < 44 {
		b.t.Fatalf("primary navigation toggle is %gx%g, want a 44px touch target", toggle.Width, toggle.Height)
	}
	b.run(browserTap(toggle.X, toggle.Y))
	b.poll(`(() => {
      const nav=document.querySelector('#mpress-primary-navigation');
      return nav.classList.contains('primary-open') && document.activeElement===nav.querySelector('a');
    })()`, "touch did not open the primary navigation and focus its first link")
}

// tappableAnchor finds a visible link by its exact text and proves that the
// link itself receives a touch at its centre.
func (b *browserActions) tappableAnchor(text string) browserTouchTarget {
	b.t.Helper()
	var target browserTouchTarget
	b.js(fmt.Sprintf(`(() => {
      const link=[...document.querySelectorAll('main a')].find(a=>a.textContent.trim()===%s);
      if(!link)return {Ready:false};
      const r=link.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;
      const hit=document.elementFromPoint(x,y);
      return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:0,ClipBottom:innerHeight,
        Ready:Boolean(hit&&(hit===link||link.contains(hit))&&r.width>0&&r.height>0)};
    })()`, browserQuote(text)), &target)
	if !target.Ready {
		b.t.Fatalf("link %q is missing or does not receive its own centre touch", text)
	}
	return target
}

func buildLandingNavFixture(t *testing.T, binary string) string {
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
	configPath := filepath.Join(project, "mpress.yaml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var links strings.Builder
	for index := 1; index <= 20; index++ {
		fmt.Fprintf(&links, "    - label: Link%d\n      url: /guide/\n", index)
	}
	updated := strings.Replace(string(config), `title: "My Project"`, `title: "Menu"`, 1)
	updated = strings.Replace(updated, "  headerLinks: []", "  headerLinks:\n"+links.String(), 1)
	if err := os.WriteFile(configPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
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
	var home strings.Builder
	home.WriteString("---\ntitle: Product home\nlayout: landing\n---\n\n")
	for paragraph := 1; paragraph <= 100; paragraph++ {
		fmt.Fprintf(&home, "Paragraph %d explains how readers scan the configured header links, dismiss the menu, and continue reading the document.\n\n", paragraph)
	}
	write("content/index.md", home.String())
	write("content/guide.md", "---\ntitle: Guide\n---\n\n# Guide\n\nEvery configured header link lands here.\n\n[Back home](/)\n")
	write("content/_nav.yaml", "- label: Home\n  link: /\n- label: Guide\n  link: /guide/\n")
	run("build", "--strict")
	return filepath.Join(project, "site")
}
