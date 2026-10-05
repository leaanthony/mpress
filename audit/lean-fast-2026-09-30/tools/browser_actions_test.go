package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

type browserActions struct {
	t   *testing.T
	ctx context.Context
}

func (b *browserActions) run(actions ...chromedp.Action) {
	b.t.Helper()
	if err := chromedp.Run(b.ctx, actions...); err != nil {
		b.t.Fatal(err)
	}
}
func (b *browserActions) js(script string, result any) {
	b.t.Helper()
	b.run(chromedp.Evaluate(script, result))
}
func browserQuote(s string) string { data, _ := json.Marshal(s); return string(data) }

const browserClipFunction = `(e,p) => {const pr=p.getBoundingClientRect(),clip={top:pr.top+p.clientTop,bottom:pr.top+p.clientTop+p.clientHeight,left:pr.left+p.clientLeft,right:pr.left+p.clientLeft+p.clientWidth};for(let a=e.parentElement;a&&a!==p;a=a.parentElement){const s=getComputedStyle(a),r=a.getBoundingClientRect();if(/^(auto|scroll|hidden|clip)$/.test(s.overflowY)){clip.top=Math.max(clip.top,r.top+a.clientTop);clip.bottom=Math.min(clip.bottom,r.top+a.clientTop+a.clientHeight);}if(/^(auto|scroll|hidden|clip)$/.test(s.overflowX)){clip.left=Math.max(clip.left,r.left+a.clientLeft);clip.right=Math.min(clip.right,r.left+a.clientLeft+a.clientWidth);}}return clip;}`

func browserTap(x, y float64) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{{X: x, Y: y, ID: 1}}).Do(ctx); err != nil {
			return err
		}
		return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
	}
}

// Reach controls using native swipes; callers enforce their own scroll owner.
type browserTouchTarget struct {
	X, Y, Top, Bottom, ClipTop, ClipBottom, SwipeX float64
	Ready                                          bool
}

func (b *browserActions) reachByTouch(selector string, probe func() browserTouchTarget) browserTouchTarget {
	b.t.Helper()
	for attempt := 0; attempt < 24; attempt++ {
		box := probe()
		if box.Ready {
			return box
		}
		span := box.ClipBottom - box.ClipTop
		distance, y := span*.5, box.ClipTop+span*.75
		if box.Top < box.ClipTop {
			distance, y = -distance, box.ClipTop+span*.25
		}
		b.run(swipe(box.SwipeX, y, 0, -distance))
	}
	b.t.Fatalf("touch cannot reach %s", selector)
	return browserTouchTarget{}
}
