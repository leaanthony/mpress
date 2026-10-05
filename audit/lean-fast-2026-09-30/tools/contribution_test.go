package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type contributionBrowser struct {
	browserActions
	pageScroll, maxDialogScroll float64
}

func TestContributionDialogPhoneReachability(t *testing.T) {
	site := os.Getenv("MPRESS_CONTRIBUTE_SITE")
	if site == "" {
		t.Skip("set MPRESS_CONTRIBUTE_SITE to an already-built production site")
	}
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	checkContributionScripts(t, server.URL)
	options := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), options...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 3*time.Minute)
	defer timeout()
	records := []map[string]any{}
	t.Cleanup(func() {
		if path := os.Getenv("MPRESS_CONTRIBUTE_REPORT"); path != "" {
			data, err := json.MarshalIndent(records, "", "  ")
			if err == nil {
				err = os.WriteFile(path, append(data, '\n'), 0600)
			}
			if err != nil {
				t.Error(err)
			}
		}
	})
	for _, size := range [][2]int64{{320, 568}, {375, 667}, {375, 375}, {667, 375}, {320, 320}, {1024, 375}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			b := contributionBrowser{browserActions: browserActions{t: t, ctx: ctx}}
			b.prepare(server.URL, size)
			b.open()
			b.reach("[data-contribute-translate]")
			b.tap("[data-contribute-computer]")
			b.checkCommands(server.URL, false)
			b.tap("[data-contribute-copy]")
			var copied bool
			b.js(`window.auditClipboard===document.querySelector('[data-contribute-command]').textContent`, &copied)
			if !copied {
				t.Fatal("copy did not receive the page-specific command")
			}
			b.tap(".mpress-contribute-explainer > summary")
			b.reach("[data-contribute-manual]")
			b.reach(`.mpress-contribute-explainer a[href$="contribute.sh"]`)
			b.reach(`.mpress-contribute-explainer a[href$="contribute.ps1"]`)
			b.run(chromedp.KeyEvent(kb.Escape))
			b.checkClosed()
			b.open()
			b.tap("[data-contribute-translate]")
			b.checkCommands(server.URL, true)
			b.tap("[data-contribute-close]")
			b.checkClosed()
			records = append(records, map[string]any{"width": size[0], "height": size[1], "maxDialogScroll": b.maxDialogScroll, "local_setup_explanation_translation_focus": "pass"})
			t.Logf("%dx%d: choices/setup/explanation/copy/close/focus pass; dialog scroll %.0fpx", size[0], size[1], b.maxDialogScroll)
		})
	}
}
func checkContributionScripts(t *testing.T, url string) {
	t.Helper()
	for _, name := range []string{"contribute.sh", "contribute.ps1"} {
		response, err := http.Get(url + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != 200 || !strings.Contains(string(data), "https://github.com/leaanthony/mpress.git") {
			t.Fatalf("missing site-specific %s script: HTTP %d, %v", name, response.StatusCode, err)
		}
	}
}
func (b *contributionBrowser) prepare(url string, size [2]int64) {
	b.t.Helper()
	options := []chromedp.EmulateViewportOption{}
	if size[0] <= 760 {
		options = append(options, chromedp.EmulateMobile)
	}
	if size[0] < size[1] {
		options = append(options, chromedp.EmulatePortrait)
	} else {
		options = append(options, chromedp.EmulateLandscape)
	}
	b.run(chromedp.EmulateViewport(size[0], size[1], options...), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(url+"/configuration/"), chromedp.WaitReady("body"), chromedp.Sleep(100*time.Millisecond))
	// Capture clipboard IO in this test browser; no installer is executed.
	b.js(`Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText:async value=>{window.auditClipboard=value;}}})`, nil)
}
func (b *contributionBrowser) open() {
	b.t.Helper()
	var point struct {
		X, Y, Scroll float64
		Visible      bool
	}
	b.js(`(() => {const button=document.querySelector('.header-contribute');if(!button)throw Error('missing contribution launcher');button.scrollIntoView({block:'center'});const r=button.getBoundingClientRect(),x=r.x+r.width/2,y=r.y+r.height/2;window.auditContributionOpener=button;return {X:x,Y:y,Scroll:scrollY,Visible:r.top>=0&&r.bottom<=innerHeight&&r.left>=0&&r.right<=innerWidth&&button.contains(document.elementFromPoint(x,y))};})()`, &point)
	if !point.Visible {
		b.t.Fatal("contribution launcher is outside viewport")
	}
	b.pageScroll = point.Scroll
	b.run(browserTap(point.X, point.Y), chromedp.Sleep(100*time.Millisecond))
	var open bool
	b.js(`document.getElementById('mpress-contribute-dialog').open`, &open)
	if !open {
		b.t.Fatal("touch launcher did not open dialog")
	}
}
func (b *contributionBrowser) reach(selector string) (float64, float64) {
	b.t.Helper()
	point := b.reachByTouch(selector, func() browserTouchTarget {
		var box struct {
			browserTouchTarget
			Scroll, PageScroll float64
			Open               bool
		}
		b.js(`(() => {const p=document.getElementById('mpress-contribute-dialog'),e=p.querySelector(`+browserQuote(selector)+`);if(!e)throw Error('missing contribution control');const r=e.getBoundingClientRect(),pr=p.getBoundingClientRect(),clip=(`+browserClipFunction+`)(e,p),hit=[...e.getClientRects()].find(rect=>rect.top>=clip.top+2&&rect.bottom<=clip.bottom-2&&e.contains(document.elementFromPoint(rect.x+rect.width/2,rect.y+rect.height/2)))||r,x=hit.x+hit.width/2,y=hit.y+hit.height/2;return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:clip.top+2,ClipBottom:clip.bottom-2,SwipeX:pr.left+10,Scroll:p.scrollTop,PageScroll:scrollY,Open:p.open,Ready:r.height>0&&r.top>=clip.top+2&&r.bottom<=clip.bottom-2&&e.contains(document.elementFromPoint(x,y))};})()`, &box)
		if !box.Open {
			b.t.Fatal("dialog closed while reaching a control")
		}
		if box.PageScroll != b.pageScroll {
			b.t.Fatalf("dialog swipe moved page %.0f -> %.0f", b.pageScroll, box.PageScroll)
		}
		b.maxDialogScroll = max(b.maxDialogScroll, box.Scroll)
		return box.browserTouchTarget
	})
	return point.X, point.Y
}
func (b *contributionBrowser) tap(selector string) {
	b.t.Helper()
	x, y := b.reach(selector)
	b.run(browserTap(x, y))
}
func (b *contributionBrowser) checkCommands(url string, translate bool) {
	b.t.Helper()
	var commands struct{ Source, Command, Manual, Shell, PowerShell string }
	b.js(`(() => {const p=document.getElementById('mpress-contribute-dialog');return {Source:p.dataset.contributionSource,Command:p.querySelector('[data-contribute-command]').textContent,Manual:p.querySelector('[data-contribute-manual]').textContent,Shell:new URL(p.dataset.installerShell,location.href).href,PowerShell:new URL(p.dataset.installerPowershell,location.href).href};})()`, &commands)
	if commands.Source != "docs/configuration.md" || commands.Shell != url+"/contribute.sh" || commands.PowerShell != url+"/contribute.ps1" || !strings.Contains(commands.Command, commands.Shell) || !strings.Contains(commands.Manual, url+"/configuration/") {
		b.t.Fatalf("page-specific launch arguments changed: %+v", commands)
	}
	if translate {
		if !strings.Contains(commands.Manual, " --goal translate") {
			b.t.Fatal("translation goal missing")
		}
	} else if !strings.Contains(commands.Command, commands.Source) {
		b.t.Fatal("local command lost exact source path")
	}
}
func (b *contributionBrowser) checkClosed() {
	b.t.Helper()
	var restored bool
	b.js(`!document.getElementById('mpress-contribute-dialog').open&&document.activeElement===window.auditContributionOpener`, &restored)
	if !restored {
		b.t.Fatal("close/Escape did not restore contribution launcher focus")
	}
}
