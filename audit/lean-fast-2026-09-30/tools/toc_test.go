package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func TestResponsiveTOC(t *testing.T) {
	site := os.Getenv("MPRESS_TOC_SITE")
	if site == "" {
		t.Skip("set MPRESS_TOC_SITE to an already-built production site")
	}
	route := os.Getenv("MPRESS_TOC_ROUTE")
	if route == "" {
		route = "/configuration/"
	}
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 2*time.Minute)
	defer timeout()
	for _, theme := range []string{"light", "dark"} {
		for _, size := range [][2]int64{{390, 844}, {375, 375}, {900, 1000}, {1440, 1000}} {
			t.Run(fmt.Sprintf("%dx%d-%s", size[0], size[1], theme), func(t *testing.T) {
				b := browserActions{t: t, ctx: ctx}
				b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+route), chromedp.WaitReady(".toc"))
				b.js(`document.documentElement.dataset.theme=`+browserQuote(theme), nil)
				check := func(script, message string) {
					t.Helper()
					var ok bool
					b.js(script, &ok)
					if !ok {
						t.Fatal(message)
					}
				}
				check(`!!document.querySelector('.toc details > summary')`, "article navigation has no native disclosure")
				if size[0] > 1180 {
					check(`document.querySelector('.toc details').open && [...document.querySelectorAll('.toc a')].every(a=>a.getBoundingClientRect().height>0)`, "desktop sections must remain visible")
					return
				}
				check(`!document.querySelector('.toc details').open`, "compact navigation should start closed")
				var point struct{ X, Y float64 }
				b.js(`(()=>{const r=document.querySelector('.toc summary').getBoundingClientRect();return {X:r.x+r.width/2,Y:r.y+r.height/2}})()`, &point)
				b.run(browserTap(point.X, point.Y))
				check(`document.querySelector('.toc details').open && [...document.querySelectorAll('.toc a')].length>1`, "touch must open article sections")
				captureTOC(t, b, fmt.Sprintf("toc-%dx%d-%s", size[0], size[1], theme))
				b.run(chromedp.KeyEvent("\u001b"))
				check(`!document.querySelector('.toc details').open && document.activeElement===document.querySelector('.toc summary')`, "Escape must close sections and restore focus")
				b.run(chromedp.KeyEvent("\r"))
				check(`document.querySelector('.toc details').open`, "Enter must reopen article sections")
				b.run(chromedp.KeyEvent("\t"), chromedp.KeyEvent("\t"), chromedp.KeyEvent("\r"), chromedp.Sleep(400*time.Millisecond))
				check(`(()=>{const target=document.getElementById(decodeURIComponent(location.hash.slice(1))),bar=document.querySelector('.toc').getBoundingClientRect();return !!target && !document.querySelector('.toc details').open && document.activeElement===target && target.getBoundingClientRect().top>=bar.bottom-1})()`, "section selection must close, focus, and reveal its heading below the bar")
				b.run(chromedp.Click(".toc summary"))
				last := b.reachByTouch("last article section", func() browserTouchTarget {
					var box browserTouchTarget
					b.js(`(()=>{const p=document.querySelector('.toc'),a=p.querySelector('a:last-child'),r=a.getBoundingClientRect(),pr=p.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:pr.top,ClipBottom:pr.bottom,SwipeX:pr.right-10,Ready:r.top>=pr.top&&r.bottom<=pr.bottom&&a.contains(document.elementFromPoint(x,y))}})()`, &box)
					return box
				})
				b.run(browserTap(last.X, last.Y), chromedp.Sleep(400*time.Millisecond))
				check(`!document.querySelector('.toc details').open && location.hash===document.querySelector('.toc a:last-child').hash`, "touch must reach and select the last section in an overflowing chooser")
				if size[0] <= 760 {
					b.run(chromedp.Click("#menu"), chromedp.Sleep(300*time.Millisecond))
					check(`(()=>{const a=document.querySelector('.sidebar nav a'),r=a.getBoundingClientRect();return document.body.classList.contains('nav-open')&&r.top>=document.querySelector('header').getBoundingClientRect().bottom&&a.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2))})()`, "article navigation must not cover the first site navigation link")
					captureTOC(t, b, fmt.Sprintf("menu-%dx%d-%s", size[0], size[1], theme))
					b.run(chromedp.KeyEvent("\u001b"))
					check(`!document.body.classList.contains('nav-open') && document.activeElement===document.querySelector('#menu')`, "Escape must restore site navigation focus")
				}
				b.run(chromedp.EmulateViewport(1440, 1000), chromedp.Sleep(100*time.Millisecond))
				check(`document.querySelector('.toc details').open && [...document.querySelectorAll('.toc a')].every(a=>a.getBoundingClientRect().height>0)`, "resizing to desktop must restore all sections")
			})
		}
	}
	t.Run("without-javascript", func(t *testing.T) {
		b := browserActions{t: t, ctx: ctx}
		b.run(chromedp.EmulateViewport(390, 844), emulation.SetScriptExecutionDisabled(true), chromedp.Navigate(server.URL+route), chromedp.WaitReady(".toc"))
		defer b.run(emulation.SetScriptExecutionDisabled(false))
		var usable bool
		b.js(`(()=>{const d=document.querySelector('.toc details');return !!d && d.open && [...d.querySelectorAll('a')].every(a=>a.getBoundingClientRect().height>0)})()`, &usable)
		if !usable {
			t.Fatal("article sections must remain available without JavaScript")
		}
	})
}

func captureTOC(t *testing.T, b browserActions, name string) {
	t.Helper()
	dir := os.Getenv("MPRESS_TOC_SCREENSHOTS")
	if dir == "" {
		return
	}
	var data []byte
	b.run(chromedp.CaptureScreenshot(&data))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), data, 0644); err != nil {
		t.Fatal(err)
	}
}
