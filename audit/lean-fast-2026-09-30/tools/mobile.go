package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

func swipe(x, y, dx, dy float64) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: x, Y: y, ID: 1}}).Do(ctx); err != nil {
			return err
		}
		for i := 1; i <= 16; i++ {
			if err := input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{{X: x + dx*float64(i)/16, Y: y + dy*float64(i)/16, ID: 1}}).Do(ctx); err != nil {
				return err
			}
			time.Sleep(16 * time.Millisecond)
		}
		if err := input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx); err != nil {
			return err
		}
		time.Sleep(400 * time.Millisecond)
		return nil
	}
}

func main() {
	server := httptest.NewServer(http.FileServer(http.Dir(os.Args[1])))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelTimeout()
	records := []map[string]any{}
	run := func(actions ...chromedp.Action) {
		if err := chromedp.Run(ctx, actions...); err != nil {
			panic(err)
		}
	}
	for _, height := range []int64{667, 375} {
		for _, path := range []string{"/configuration/", "/code-components/", "/data-components/"} {
			r := map[string]any{"path": path, "width": 375, "height": height}
			run(chromedp.EmulateViewport(375, height), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+path), chromedp.WaitReady("body"))
			var initial map[string]any
			run(chromedp.Evaluate(`(() => ({hit:document.elementFromPoint(185,300)?.outerHTML.slice(0,180),panels:[...document.querySelectorAll('[popover]')].map(p=>{let s=getComputedStyle(p),r=p.getBoundingClientRect();return{id:p.id,open:p.matches(':popover-open'),display:s.display,opacity:s.opacity,pointer:s.pointerEvents,x:r.x,y:r.y,w:r.width,h:r.height}}),scrollHeight:document.scrollingElement.scrollHeight,viewport:innerHeight}))()`, &initial))
			r["initial"] = initial
			run(swipe(185, 300, 0, -180))
			var scroll float64
			run(chromedp.Evaluate("scrollY", &scroll))
			r["bodyScrollAfterSwipe"] = scroll
			run(chromedp.Evaluate(`window.scrollTo(0,0);document.querySelector('#accessibility')?.click()`, nil), chromedp.Sleep(200*time.Millisecond))
			var geometry map[string]any
			run(chromedp.Evaluate(`(() => {let p=document.querySelector('#mpress-accessibility-panel'),b=p.querySelector('.mpress-accessibility-body'),r=p.getBoundingClientRect();return {panelY:r.y,panelBottom:r.bottom,viewport:innerHeight,bodyH:b.clientHeight,bodyScrollH:b.scrollHeight,panelH:p.clientHeight,panelScrollH:p.scrollHeight,hit:document.elementFromPoint(185,300)?.className}})()`, &geometry))
			r["accessibilityGeometry"] = geometry
			run(swipe(185, 300, 0, -180))
			var state map[string]any
			run(chromedp.Evaluate(`(() => {let p=document.querySelector('#mpress-accessibility-panel'),b=p.querySelector('.mpress-accessibility-body');return{body:b.scrollTop,panel:p.scrollTop,page:scrollY}})()`, &state))
			r["accessibilityScrollAfterSwipe"] = state
			run(chromedp.Evaluate(`document.querySelector('#mpress-accessibility-panel').hidePopover();window.scrollTo(0,0)`, nil), chromedp.Sleep(200*time.Millisecond), swipe(185, 300, 0, -180), chromedp.Evaluate("scrollY", &scroll))
			r["bodyScrollAfterClose"] = scroll
			run(chromedp.Evaluate(`(()=>{let s=document.createElement('style');s.textContent='.utility-menu-panel[popover]{visibility:hidden;pointer-events:none}.utility-menu-panel[popover]:popover-open{visibility:visible;pointer-events:auto}';document.head.append(s);window.scrollTo(0,0)})()`, nil), chromedp.Sleep(200*time.Millisecond), swipe(185, 300, 0, -180), chromedp.Evaluate("scrollY", &scroll))
			r["bodyScrollWithInertCSS"] = scroll
			records = append(records, r)
		}
	}
	data, _ := json.MarshalIndent(records, "", "  ")
	fmt.Println(string(data))
}
