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

// TestDocumentationSidebarWithoutJavaScript checks the default theme's phone
// fallback when scripts are blocked: the documentation sidebar stays in normal
// flow with usable links, its inert launchers are hidden, and a sidebar-only
// destination is reachable by touch and keyboard. The enhanced menu and search
// keep working on the same minified build.
func TestDocumentationSidebarWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_NOJS_SIDEBAR_BIN")
	if binary == "" {
		t.Skip("set MPRESS_NOJS_SIDEBAR_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	site := buildNoScriptSidebarFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {390, 844}} {
		t.Run(fmt.Sprintf("blocked-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true))
			loadWithoutJavaScript(&b, server.URL+"/guide/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			var layout noScriptSidebarLayout
			b.js(`(() => {
              const sidebar=document.querySelector('.sidebar');
              const links=[...sidebar.querySelectorAll('a')];
              const boxes=links.map(a=>{
                a.scrollIntoView({block:'center',behavior:'instant'});
                const r=a.getBoundingClientRect();
                const x=Math.min(Math.max(r.x+r.width/2,0),innerWidth-1);
                const y=Math.min(Math.max(r.y+r.height/2,0),innerHeight-1);
                const hit=document.elementFromPoint(x,y);
                return {Text:a.textContent.trim(),Width:r.width,Height:r.height,
                  Hit:Boolean(hit&&(hit===a||a.contains(hit)))};
              });
              window.scrollTo(0,0);
              return {
                MenuDisplay:getComputedStyle(document.querySelector('#mpress-menu-toggle')).display,
                SearchDisplay:getComputedStyle(document.querySelector('.search')).display,
                SidebarPosition:getComputedStyle(sidebar).position,
                SidebarTransform:getComputedStyle(sidebar).transform,
                SidebarLeft:sidebar.getBoundingClientRect().left,
                SidebarRight:sidebar.getBoundingClientRect().right,
                SidebarWidth:sidebar.getBoundingClientRect().width,
                InnerWidth:innerWidth, InnerHeight:innerHeight,
                ScrollHeight:document.documentElement.scrollHeight,
                Links:boxes
              };
            })()`, &layout)
			if layout.MenuDisplay != "none" {
				t.Fatalf("inert menu launcher is still visible without JavaScript: %+v", layout)
			}
			if layout.SearchDisplay != "none" {
				t.Fatalf("inert search launcher is still visible without JavaScript: %+v", layout)
			}
			if layout.SidebarPosition != "static" || layout.SidebarTransform != "none" {
				t.Fatalf("sidebar is not in normal flow without JavaScript: %+v", layout)
			}
			if layout.SidebarWidth <= 0 || layout.SidebarLeft < -0.5 || layout.SidebarRight > float64(layout.InnerWidth)+0.5 {
				t.Fatalf("sidebar is not readable inside the viewport: %+v", layout)
			}
			if len(layout.Links) == 0 {
				t.Fatalf("sidebar has no links without JavaScript: %+v", layout)
			}
			for _, link := range layout.Links {
				if link.Width <= 0 || link.Height < 44 || !link.Hit {
					t.Fatalf("sidebar link %q is not a usable touch target: %+v", link.Text, link)
				}
			}
			var scrolled int
			b.js(`(() => {window.scrollTo(0,document.documentElement.scrollHeight);return Math.round(window.scrollY)})()`, &scrolled)
			if layout.ScrollHeight <= layout.InnerHeight || scrolled <= 0 {
				t.Fatalf("document body does not scroll with the sidebar in flow: %+v scrolled=%d", layout, scrolled)
			}

			b.js(`window.scrollTo(0,0)`, nil)
			var target browserTouchTarget
			b.js(`(() => {
              const link=[...document.querySelectorAll('.sidebar a')].find(a=>a.textContent.trim()==='Reference');
              if(!link)throw Error('missing sidebar-only reference link');
              link.scrollIntoView({block:'center',behavior:'instant'});
              const r=link.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;
              const hit=document.elementFromPoint(x,y);
              return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:0,ClipBottom:innerHeight,
                Ready:Boolean(hit&&(hit===link||link.contains(hit)))};
            })()`, &target)
			if !target.Ready {
				t.Fatalf("sidebar-only destination does not receive its own centre tap: %+v", target)
			}
			if _, err := chromedp.RunResponse(b.ctx, browserTap(target.X, target.Y)); err != nil {
				t.Fatalf("tap on the sidebar-only destination did not complete a navigation: %v", err)
			}
			b.poll(`location.pathname==='/reference/' && document.querySelector('main').textContent.includes('Reference only reachable from the sidebar')`, "blocked-JavaScript sidebar link did not open its destination")
		})
	}

	t.Run("blocked-keyboard", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(390, 844), emulation.SetTouchEmulationEnabled(true))
		loadWithoutJavaScript(&b, server.URL+"/guide/")
		b.run(chromedp.Evaluate(`document.body.focus()`, nil))
		reached := false
		for range 40 {
			b.run(chromedp.KeyEvent(kb.Tab))
			b.js(`document.activeElement===[...document.querySelectorAll('.sidebar a')].find(a=>a.textContent.trim()==='Reference')`, &reached)
			if reached {
				break
			}
		}
		if !reached {
			t.Fatalf("sidebar is not keyboard reachable without JavaScript")
		}
		if _, err := chromedp.RunResponse(b.ctx, chromedp.KeyEvent(kb.Enter)); err != nil {
			t.Fatalf("keyboard activation of the sidebar did not complete a navigation: %v", err)
		}
		b.poll(`location.pathname==='/reference/' && document.querySelector('main').textContent.includes('Reference only reachable from the sidebar')`, "keyboard activation of the sidebar did not open its reference destination")
	})

	t.Run("enhanced-phone", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(320, 667), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
		var state struct{ Menu, Search string }
		b.js(`(() => ({Menu:getComputedStyle(document.querySelector('#mpress-menu-toggle')).display,
          Search:getComputedStyle(document.querySelector('#mpress-search-toggle')).display}))()`, &state)
		if state.Menu == "none" || state.Search == "none" {
			t.Fatalf("enhanced phone lost its menu or search launcher: %+v", state)
		}
		b.openMenuByTouch()
		b.poll(`document.body.classList.contains('nav-open')`, "touch did not open site navigation")
		b.run(chromedp.KeyEvent(kb.Escape))
		b.poll(`!document.body.classList.contains('nav-open')`, "Escape did not close site navigation")
		b.openSearchByTouch()
	})

	t.Run("enhanced-desktop", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(1280, 900), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
		var state struct {
			Menu, Sidebar, Position string
			Left, Width             float64
		}
		b.js(`(() => {const s=document.querySelector('.sidebar'),r=s.getBoundingClientRect();
          return {Menu:getComputedStyle(document.querySelector('#mpress-menu-toggle')).display,
            Sidebar:getComputedStyle(s).display,Position:getComputedStyle(s).position,Left:r.left,Width:r.width};})()`, &state)
		if state.Menu != "none" || state.Sidebar == "none" || state.Position != "sticky" || state.Width <= 0 || state.Left < 0 {
			t.Fatalf("desktop sidebar or header regressed: %+v", state)
		}
	})
}

type noScriptSidebarLayout struct {
	MenuDisplay, SearchDisplay              string
	SidebarPosition, SidebarTransform       string
	SidebarLeft, SidebarRight, SidebarWidth float64
	InnerWidth, InnerHeight, ScrollHeight   int
	Links                                   []struct {
		Text          string
		Width, Height float64
		Hit           bool
	}
}

// buildNoScriptSidebarFixture creates a documentation site whose reference page
// is listed only in the sidebar navigation.
func buildNoScriptSidebarFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	files := map[string]string{
		"mpress.yaml":          "site:\n  title: Sidebar fallback\n",
		"content/index.md":     "---\ntitle: Home\nlayout: landing\n---\n\n# Home\n",
		"content/guide.md":     "---\ntitle: Guide\n---\n\n# Guide\n\n" + strings.Repeat("Read the guide and follow the sidebar to the reference.\n\n", 40),
		"content/reference.md": "---\ntitle: Reference\n---\n\n# Reference\n\nReference only reachable from the sidebar.\n",
		"content/_nav.yaml":    "- label: Guide\n  link: /guide/\n- label: Reference\n  link: /reference/\n",
	}
	for name, body := range files {
		target := filepath.Join(project, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command(binary, "build", "--strict")
	command.Dir = project
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build sidebar fixture: %v\n%s", err, output)
	}
	return filepath.Join(project, "site")
}
