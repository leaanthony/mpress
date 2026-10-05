package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type accessibilityViewport struct {
	width, height int64
	scale         int
	theme         string
}
type accessibilityBrowser struct {
	browserActions
	constrained                   bool
	maxPanelScroll, maxBodyScroll float64
}

// Production output, real CDP touch/keyboard input, and explicit 200% text
// simulation. Font sizing does not alter overflow/positioning rules under test.
func TestAccessibilityViewportReachability(t *testing.T) {
	site := os.Getenv("MPRESS_A11Y_SITE")
	if site == "" {
		t.Skip("set MPRESS_A11Y_SITE to an already-built production site")
	}
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 4*time.Minute)
	defer timeout()
	records := []map[string]any{}
	t.Cleanup(func() { saveAccessibilityRecords(t, records) })
	cases := []accessibilityViewport{
		{375, 667, 1, "light"}, {667, 375, 1, "dark"}, {375, 375, 1, "light"}, {320, 320, 2, "dark"},
		{667, 375, 2, "light"}, {1024, 375, 2, "dark"}, {1280, 900, 1, "dark"}, {375, 667, 2, "light"},
		{761, 601, 2, "light"}, {1280, 900, 2, "dark"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%dx%d-text%d-%s", c.width, c.height, c.scale, c.theme), func(t *testing.T) {
			b := &accessibilityBrowser{browserActions: browserActions{t: t, ctx: ctx}, constrained: c.width <= 760 || c.height <= 600}
			record := b.prepare(server.URL, c)
			records = append(records, record)
			b.checkGeometry(record["initial"].(map[string]float64), c)
			b.touchPreferences()
			b.keyboardPreferences()
			b.touch("[data-a11y-close]")
			var closed bool
			b.js(`!document.getElementById('mpress-accessibility-panel').matches(':popover-open')`, &closed)
			if !closed {
				t.Fatal("touch close failed")
			}
			record["maxPanelScroll"] = b.maxPanelScroll
			record["maxBodyScroll"] = b.maxBodyScroll
			record["touch_and_keyboard"] = "pass"
			t.Logf("%dx%d, text %d00%%, %s: preferences/reset/close reachable; panel scroll %.0fpx", c.width, c.height, c.scale, c.theme, b.maxPanelScroll)
		})
	}
}
func saveAccessibilityRecords(t *testing.T, records []map[string]any) {
	t.Helper()
	path := os.Getenv("MPRESS_A11Y_REPORT")
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err == nil {
		err = os.WriteFile(path, append(data, '\n'), 0600)
	}
	if err != nil {
		t.Error(err)
	}
}
func (b *accessibilityBrowser) prepare(url string, c accessibilityViewport) map[string]any {
	b.t.Helper()
	options := []chromedp.EmulateViewportOption{}
	if c.width <= 760 {
		options = append(options, chromedp.EmulateMobile)
	}
	if c.width < c.height {
		options = append(options, chromedp.EmulatePortrait)
	} else {
		options = append(options, chromedp.EmulateLandscape)
	}
	b.run(chromedp.EmulateViewport(c.width, c.height, options...), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(url+"/configuration/"), chromedp.WaitReady("body"))
	b.js(`document.querySelector('[data-a11y-reset]').click();document.documentElement.dataset.theme=`+browserQuote(c.theme), nil)
	if c.scale == 2 {
		b.js(`(() => {const sizes=[...document.querySelectorAll('#mpress-accessibility-panel :is(h2,h3,p,span,label,button,small,output,strong)')].map(e=>[e,parseFloat(getComputedStyle(e).fontSize)]);sizes.forEach(([e,size])=>e.style.fontSize=(size*2)+'px');})()`, nil)
	}
	// Opening is setup; all settings/reset/close below use native input.
	b.js(`document.getElementById('accessibility').click()`, nil)
	b.run(chromedp.Sleep(200 * time.Millisecond))
	var geometry map[string]float64
	b.js(`(() => {const p=document.getElementById('mpress-accessibility-panel'),b=p.querySelector('.mpress-accessibility-body'),r=p.getBoundingClientRect();return {top:r.top,bottom:r.bottom,left:r.left,right:r.right,panelH:p.clientHeight,panelScrollH:p.scrollHeight,bodyH:b.clientHeight,bodyScrollH:b.scrollHeight};})()`, &geometry)
	return map[string]any{"width": c.width, "height": c.height, "textScale": c.scale, "theme": c.theme, "initial": geometry}
}
func (b *accessibilityBrowser) checkGeometry(g map[string]float64, c accessibilityViewport) {
	b.t.Helper()
	if g["top"] < 0 || g["bottom"] > float64(c.height)+1 || g["left"] < 0 || g["right"] > float64(c.width)+1 {
		b.t.Fatalf("panel outside viewport: %v", g)
	}
	if b.constrained && g["bodyScrollH"] > g["bodyH"]+1 {
		b.t.Fatalf("constrained viewport has nested scroll containers: %v", g)
	}
}
func (b *accessibilityBrowser) touch(selector string) {
	b.t.Helper()
	box := b.reachByTouch(selector, func() browserTouchTarget {
		var box struct {
			browserTouchTarget
			PanelScroll, BodyScroll, PageScroll float64
		}
		b.js(`(() => {const e=document.querySelector(`+browserQuote(selector)+`),p=document.getElementById('mpress-accessibility-panel');if(!e||e.disabled)throw Error('missing or disabled control');const r=e.getBoundingClientRect(),pr=p.getBoundingClientRect(),clip=(`+browserClipFunction+`)(e,p),x=r.x+r.width/2,y=r.y+r.height/2;return {X:x,Y:y,Top:r.top,Bottom:r.bottom,ClipTop:clip.top+2,ClipBottom:clip.bottom-2,SwipeX:pr.left+10,PanelScroll:p.scrollTop,BodyScroll:p.querySelector('.mpress-accessibility-body').scrollTop,PageScroll:scrollY,Ready:r.top>=clip.top+2&&r.bottom<=clip.bottom-2&&e.contains(document.elementFromPoint(x,y))};})()`, &box)
		b.maxPanelScroll = max(b.maxPanelScroll, box.PanelScroll)
		b.maxBodyScroll = max(b.maxBodyScroll, box.BodyScroll)
		if box.PageScroll != 0 || (b.constrained && box.BodyScroll != 0) {
			b.t.Fatalf("wrong touch scroll owner: body %.0f, page %.0f", box.BodyScroll, box.PageScroll)
		}
		return box.browserTouchTarget
	})
	b.run(browserTap(box.X, box.Y))
}
func (b *accessibilityBrowser) touchPreferences() {
	b.t.Helper()
	for _, tab := range []string{"reading", "focus", "vision"} {
		b.touch(`[data-a11y-tab="` + tab + `"]`)
		var selectors []string
		b.js(`Array.from(document.querySelectorAll('[data-a11y-tabpanel="`+tab+`"] button,[data-a11y-tabpanel="`+tab+`"] input')).filter(e=>!e.disabled&&!e.closest('[hidden]')).map(e=>e.id?'#'+e.id:e.dataset.a11yChoice?'[data-a11y-choice="'+e.dataset.a11yChoice+'"][data-value="'+e.dataset.value+'"]':e.dataset.a11yWidthMode?'[data-a11y-width-mode="'+e.dataset.a11yWidthMode+'"]':e.dataset.a11yToggle?'[data-a11y-toggle="'+e.dataset.a11yToggle+'"]':'.mpress-a11y-select-trigger')`, &selectors)
		for _, selector := range selectors {
			if selector == ".mpress-a11y-select-trigger" {
				b.touchColourOptions()
			} else {
				b.touch(selector)
			}
		}
		if tab == "reading" {
			b.touch("[data-a11y-width-reset]")
		}
		b.checkTouchPreferences(tab)
	}
	b.touch("[data-a11y-reset]")
	var defaults bool
	b.js(`document.documentElement.dataset.a11yText==='default'&&document.documentElement.dataset.a11yFont==='default'&&document.documentElement.dataset.a11yColour==='default'`, &defaults)
	if !defaults {
		b.t.Fatal("touch reset did not clear preferences")
	}
}
func (b *accessibilityBrowser) touchColourOptions() {
	for _, value := range []string{"default", "red-green", "blue-yellow", "low"} {
		b.touch(".mpress-a11y-select-trigger")
		b.touch(`.mpress-a11y-option[data-value="` + value + `"]`)
	}
}
func (b *accessibilityBrowser) checkTouchPreferences(tab string) {
	b.t.Helper()
	var active bool
	script := map[string]string{
		"reading": `(()=>{const d=document.documentElement.dataset;return d.a11yText==='larger'&&d.a11ySiteWidth==='fixed'&&d.a11yFont==='readable'&&d.a11ySpacing==='relaxed'&&d.a11yBionic==='true'&&document.querySelector('[data-a11y-width-reset]').disabled;})()`,
		"focus":   `(()=>{const d=document.documentElement.dataset;return d.a11yFocus==='true'&&d.a11yGuide==='true'&&d.a11yMotion==='reduced';})()`,
		"vision":  `(()=>{const d=document.documentElement.dataset;return d.a11yColour==='low'&&d.a11yContrast==='high'&&d.a11yLinks==='underlined';})()`,
	}[tab]
	b.js(script, &active)
	if !active {
		b.t.Fatalf("touch failed to apply %s preferences", tab)
	}
}

const accessibilityFocusStateJS = `(() => {const p=document.getElementById('mpress-accessibility-panel'),e=document.activeElement,r=e.getBoundingClientRect(),clip=(` + browserClipFunction + `)(e,p);return {Inside:p.contains(e),Visible:r.top>=clip.top&&r.bottom<=clip.bottom&&r.left>=clip.left&&r.right<=clip.right&&e.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2)),Footer:e.matches('[data-a11y-reset]'),HTML:e.outerHTML.slice(0,180),Top:r.top,Bottom:r.bottom,PanelTop:clip.top,PanelBottom:clip.bottom,Scroll:p.scrollTop};})()`

func (b *accessibilityBrowser) keyboardPreferences() {
	b.t.Helper()
	for index := 0; index < 3; index++ {
		b.js(`document.querySelector('[data-a11y-tab="reading"]').focus()`, nil)
		for arrow := 0; arrow < index; arrow++ {
			b.run(chromedp.KeyEvent(kb.ArrowRight))
		}
		b.tabToReset()
	}
}
func (b *accessibilityBrowser) tabToReset() {
	b.t.Helper()
	for step := 0; step < 32; step++ {
		b.run(chromedp.KeyEvent(kb.Tab))
		// Wait for focus scrolling to settle; persistent clipping remains a failure.
		if err := chromedp.Run(b.ctx, chromedp.Poll(accessibilityFocusStateJS+".Visible", nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(time.Second))); err != nil {
			b.t.Logf("focus did not settle within visible panel: %v", err)
		}
		var state struct {
			Inside, Visible, Footer                    bool
			HTML                                       string
			Top, Bottom, PanelTop, PanelBottom, Scroll float64
		}
		b.js(accessibilityFocusStateJS, &state)
		if !state.Inside || !state.Visible {
			b.t.Fatalf("keyboard control unreachable: %+v", state)
		}
		if state.Footer {
			return
		}
	}
	b.t.Fatal("keyboard could not reach reset")
}
