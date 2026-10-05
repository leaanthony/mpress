package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

// TestUtilityPopoverInteraction checks production output, without injecting CSS.
// Run it against both accessibility-enabled and disabled site directories.
func TestUtilityPopoverInteraction(t *testing.T) {
	site := os.Getenv("MPRESS_TOUCH_SITE")
	if site == "" {
		t.Skip("set MPRESS_TOUCH_SITE to an already-built site directory")
	}
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 60*time.Second)
	defer timeout()
	run := func(actions ...chromedp.Action) {
		t.Helper()
		if err := chromedp.Run(ctx, actions...); err != nil {
			t.Fatal(err)
		}
	}
	for _, height := range []int64{667, 375} {
		run(chromedp.EmulateViewport(375, height), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/configuration/"), chromedp.WaitReady("body"))
		var ids []string
		run(chromedp.Evaluate(`Array.from(document.querySelectorAll('.utility-menu-panel[popover]'), p => p.id)`, &ids))
		if len(ids) == 0 {
			t.Fatal("fixture must contain at least one utility popover")
		}
		for _, id := range ids {
			encodedID, _ := json.Marshal(id)
			panelID := string(encodedID)
			checkClosed := func() {
				t.Helper()
				var inert bool
				// Focusing a hidden descendant must fail, including during exit transitions.
				run(chromedp.Evaluate(`(() => {const p=document.getElementById(`+panelID+`),s=getComputedStyle(p); p.querySelector('button,a,input,select,[tabindex]')?.focus(); const r=p.getBoundingClientRect(),hit=document.elementFromPoint(Math.max(0,r.x+10),Math.max(0,r.y+10)); return !p.matches(':popover-open') && s.visibility==='hidden' && s.pointerEvents==='none' && !p.contains(document.activeElement) && !p.contains(hit);})()`, &inert))
				if !inert {
					t.Fatalf("closed panel %q fails visibility/pointer/focus/hit checks at height %d", id, height)
				}
			}
			checkClosed()
			var interactive bool
			run(chromedp.Evaluate(`document.getElementById(`+panelID+`).showPopover()`, nil), chromedp.Sleep(200*time.Millisecond), chromedp.Evaluate(`(() => {const p=document.getElementById(`+panelID+`),s=getComputedStyle(p),f=p.querySelector('button,a,input,select,[tabindex]'); f?.focus(); const r=p.getBoundingClientRect(),hit=document.elementFromPoint(r.x+r.width/2,Math.min(innerHeight-1,r.y+Math.min(r.height/2,30))); return p.matches(':popover-open') && s.visibility==='visible' && s.pointerEvents==='auto' && p.contains(hit) && !!f && document.activeElement===f;})()`, &interactive))
			if !interactive {
				t.Fatalf("open panel %q is not interactive at height %d", id, height)
			}
			run(chromedp.Evaluate(`document.getElementById(`+panelID+`).hidePopover(); document.activeElement?.blur()`, nil))
			checkClosed()
			run(chromedp.Sleep(200 * time.Millisecond))
			checkClosed()
		}
		var scroll float64
		run(chromedp.Evaluate("window.scrollTo(0,0)", nil), swipe(185, 300, 0, -180), chromedp.Evaluate("scrollY", &scroll))
		if scroll <= 0 {
			t.Fatalf("page cannot scroll after closing popovers at height %d", height)
		}
		t.Logf("height=%d: %d popovers pass focus/hit/transition checks; touch scroll %.0fpx", height, len(ids), scroll)
	}
}
