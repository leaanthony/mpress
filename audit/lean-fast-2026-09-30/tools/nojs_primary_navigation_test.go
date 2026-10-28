package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

// TestPrimaryNavigationWithoutJavaScript checks the default theme's
// progressive enhancement: a phone reader who blocks JavaScript still gets the
// configured primary links in normal flow, with no inert toggle, and can follow
// them from both a landing page and a documentation page. The produced site is
// the normal minified and purged build.
func TestPrimaryNavigationWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_HEADER_NAV_BIN")
	if binary == "" {
		t.Skip("set MPRESS_HEADER_NAV_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	site := buildLandingNavFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {390, 844}} {
		for _, route := range []string{"/", "/guide/"} {
			t.Run(fmt.Sprintf("%s-%dx%d", route, size[0], size[1]), func(t *testing.T) {
				b := newBrandPage(t, ctx)
				b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true))
				loadWithoutJavaScript(&b, server.URL+route)
				b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

				var layout noScriptNavigationLayout
				b.js(`(() => {
                  const header=document.querySelector('body > header');
                  const nav=document.querySelector('#mpress-primary-navigation');
                  const toggle=document.querySelector('#mpress-primary-toggle');
                  const main=document.querySelector('main');
                  const links=[...nav.querySelectorAll('a')];
                  return {
                    NavDisplay:getComputedStyle(nav).display,
                    ToggleDisplay:toggle?getComputedStyle(toggle).display:'missing',
                    HeaderBottom:header.getBoundingClientRect().bottom,
                    MainTop:main.getBoundingClientRect().top,
                    ScrollWidth:document.documentElement.scrollWidth,
                    InnerWidth:innerWidth,
                    Links:links.map(a=>{
                      const r=a.getBoundingClientRect();
                      const x=Math.min(Math.max(r.x+r.width/2,0),innerWidth-1);
                      const y=Math.min(Math.max(r.y+r.height/2,0),innerHeight-1);
                      const hit=document.elementFromPoint(x,y);
                      return {Text:a.textContent.trim(),Width:r.width,Height:r.height,Top:r.top,Bottom:r.bottom,
                        Hit:Boolean(hit&&(hit===a||a.contains(hit)))};
                    })
                  };
                })()`, &layout)
				if layout.NavDisplay == "none" {
					t.Fatalf("primary navigation is hidden without JavaScript: %+v", layout)
				}
				if layout.ToggleDisplay != "none" {
					t.Fatalf("primary navigation toggle is not inert without JavaScript: %+v", layout)
				}
				if len(layout.Links) != 20 {
					t.Fatalf("primary navigation has %d links, want 20 configured links", len(layout.Links))
				}
				for _, link := range layout.Links {
					if link.Width <= 0 || link.Height < 44 || link.Bottom > float64(size[1]) || !link.Hit {
						t.Fatalf("configured link %q is not reachable in normal flow: %+v", link.Text, link)
					}
				}
				if layout.HeaderBottom > layout.MainTop+.5 {
					t.Fatalf("header covers the page content without JavaScript: header bottom %g, main top %g", layout.HeaderBottom, layout.MainTop)
				}
				if layout.ScrollWidth > layout.InnerWidth {
					t.Fatalf("primary navigation overflows the viewport: scroll width %d, viewport %d", layout.ScrollWidth, layout.InnerWidth)
				}

				if route == "/" {
					tapFirstPrimaryLink(t, &b)
				} else {
					activateFirstPrimaryLinkWithKeyboard(t, &b)
				}
				b.poll(`location.pathname==='/guide/' && document.querySelector('main').textContent.includes('Every configured header link lands here')`, "blocked-JavaScript link did not open its destination")
			})
		}
	}

	t.Run("enhanced-phone", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(320, 667), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/"), chromedp.WaitReady("body"))
		var state struct{ Toggle, Nav string }
		b.js(`(() => ({Toggle:getComputedStyle(document.querySelector('#mpress-primary-toggle')).display,
          Nav:getComputedStyle(document.querySelector('#mpress-primary-navigation')).display}))()`, &state)
		if state.Toggle == "none" || state.Nav != "none" {
			t.Fatalf("enhanced phone header lost its compact navigation: %+v", state)
		}
		b.openPrimaryNavigation()
		b.run(chromedp.KeyEvent(kb.Escape))
		b.poll(`!document.querySelector('#mpress-primary-navigation').classList.contains('primary-open') && document.activeElement===document.querySelector('#mpress-primary-toggle')`, "Escape did not close the enhanced phone navigation")
	})

	t.Run("enhanced-desktop", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(1280, 900), chromedp.Navigate(server.URL+"/"), chromedp.WaitReady("body"))
		var state struct {
			Toggle, Nav string
			Visible     int
			Hit         bool
		}
		b.js(`(() => {
          const nav=document.querySelector('#mpress-primary-navigation');
          const links=[...nav.querySelectorAll('a')],first=links[0],r=first.getBoundingClientRect();
          const hit=document.elementFromPoint(r.x+r.width/2,r.y+r.height/2);
          return {Toggle:getComputedStyle(document.querySelector('#mpress-primary-toggle')).display,
            Nav:getComputedStyle(nav).display,Visible:links.filter(a=>a.getBoundingClientRect().width>0).length,
            Hit:Boolean(hit&&(hit===first||first.contains(hit)))};
        })()`, &state)
		if state.Toggle != "none" || state.Nav == "none" || state.Visible != 20 || !state.Hit {
			t.Fatalf("desktop header lost its direct configured links: %+v", state)
		}
	})
}

type noScriptNavigationLayout struct {
	NavDisplay, ToggleDisplay string
	HeaderBottom, MainTop     float64
	ScrollWidth, InnerWidth   int
	Links                     []struct {
		Text          string
		Width, Height float64
		Top, Bottom   float64
		Hit           bool
	}
}

// loadWithoutJavaScript reloads a document while script execution is blocked,
// then re-enables the runtime only so the test can inspect the resulting page.
func loadWithoutJavaScript(b *browserActions, url string) {
	b.t.Helper()
	b.run(emulation.SetScriptExecutionDisabled(true), chromedp.Navigate(url), emulation.SetScriptExecutionDisabled(false))
	b.poll(`document.readyState==='complete'`, "page did not finish loading with JavaScript blocked")
}

// activateFirstPrimaryLinkWithKeyboard tabs from the document body until the
// first configured link has focus, then follows it with Enter.
func activateFirstPrimaryLinkWithKeyboard(t *testing.T, b *browserActions) {
	t.Helper()
	b.run(chromedp.Evaluate(`document.body.focus()`, nil))
	reached := false
	for range 24 {
		b.run(chromedp.KeyEvent(kb.Tab))
		b.js(`document.activeElement===document.querySelector('#mpress-primary-navigation a')`, &reached)
		if reached {
			break
		}
	}
	if !reached {
		t.Fatalf("first configured link is not keyboard reachable without JavaScript")
	}
	if _, err := chromedp.RunResponse(b.ctx, chromedp.KeyEvent(kb.Enter)); err != nil {
		t.Fatalf("keyboard activation of the first configured link did not complete a navigation: %v", err)
	}
}

func tapFirstPrimaryLink(t *testing.T, b *browserActions) {
	t.Helper()
	var target browserTouchTarget
	b.js(`(() => {
      const link=document.querySelector('#mpress-primary-navigation a');
      const r=link.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;
      const hit=document.elementFromPoint(x,y);
      return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:0,ClipBottom:innerHeight,
        Ready:Boolean(hit&&(hit===link||link.contains(hit)))};
    })()`, &target)
	if !target.Ready {
		t.Fatalf("first configured link does not receive its own centre tap: %+v", target)
	}
	if _, err := chromedp.RunResponse(b.ctx, browserTap(target.X, target.Y)); err != nil {
		t.Fatalf("tap on the first configured link did not complete a navigation: %v", err)
	}
}
