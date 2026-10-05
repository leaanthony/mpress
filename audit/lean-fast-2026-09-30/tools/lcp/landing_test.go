package main

import (
	"context"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func TestLandingVisualContracts(t *testing.T) {
	v3, v2 := os.Getenv("MPRESS_V3_SITE"), os.Getenv("MPRESS_V2_SITE")
	if v3 == "" || v2 == "" {
		t.Skip("set MPRESS_V3_SITE and MPRESS_V2_SITE to generated Wails sites")
	}
	s3, s2 := httptest.NewServer(serveSite(v3)), httptest.NewServer(serveSite(v2))
	defer s3.Close()
	defer s2.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 90*time.Second)
	defer timeout()
	run := func(actions ...chromedp.Action) {
		t.Helper()
		if err := chromedp.Run(ctx, actions...); err != nil {
			t.Fatal(err)
		}
	}
	for _, width := range []int64{1365, 390} {
		for _, mode := range []string{"light", "dark"} {
			checkLandingHero(t, run, s3.URL, width, mode)
		}
		checkLandingCarousel(t, run, s2.URL, width)
	}
}

func checkLandingHero(t *testing.T, run func(...chromedp.Action), baseURL string, width int64, mode string) {
	t.Helper()
	run(chromedp.EmulateViewport(width, 844), chromedp.Navigate(baseURL), chromedp.Evaluate(`localStorage.setItem('mpress-theme',`+strconv.Quote(mode)+`);`, nil), chromedp.Navigate(baseURL), chromedp.WaitReady(".mpress-frontmatter-hero-title"), chromedp.Sleep(200*time.Millisecond))
	var good bool
	run(chromedp.Evaluate(`(() => {const images=Array.from(document.querySelectorAll('.mpress-frontmatter-hero-image img')).filter(e=>getComputedStyle(e).display!=='none'),line=document.querySelector('.morph-title-line'),word=document.querySelector('.morph-word-title span');return images.length===1 && images[0].complete && images[0].naturalWidth>0 && document.documentElement.dataset.theme===`+strconv.Quote(mode)+` && getComputedStyle(line).display==='inline-flex' && getComputedStyle(word).transitionDuration.includes('0.65s') && document.documentElement.scrollWidth<=innerWidth;})()`, &good))
	if !good {
		t.Fatalf("v3 logo, animation styles or layout failed at %dpx in %s", width, mode)
	}
	current := mode
	for n := 0; n < 3; n++ {
		next := map[string]string{"system": "dark", "dark": "light", "light": "system"}[current]
		run(chromedp.Click("#theme", chromedp.ByQuery), chromedp.Evaluate(`(() => {const theme=document.documentElement.dataset.theme,dark=theme==='dark'||(theme==='system'&&matchMedia('(prefers-color-scheme: dark)').matches),images=Array.from(document.querySelectorAll('.mpress-frontmatter-hero-image img')).filter(e=>getComputedStyle(e).display!=='none');return theme===`+strconv.Quote(next)+`&&localStorage.getItem('mpress-theme')===theme&&images.length===1&&images[0].complete&&images[0].naturalWidth>0&&images[0].classList.contains(dark?'hero-logo-dark':'hero-logo-light');})()`, &good))
		if !good {
			t.Fatalf("theme toggle failed at %dpx: %s to %s", width, current, next)
		}
		current = next
	}
	// The static hero remains visible with scripting disabled in either OS theme.
	run(emulation.SetScriptExecutionDisabled(true), emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: mode}}), chromedp.Navigate(baseURL+"/ja/"), chromedp.Evaluate(`(() => {const title=document.querySelector('.mpress-frontmatter-hero-title'),images=Array.from(document.querySelectorAll('.mpress-frontmatter-hero-image img')).filter(e=>getComputedStyle(e).display!=='none');return !!title?.textContent.trim() && images.length===1 && images[0].naturalWidth>0;})()`, &good), emulation.SetScriptExecutionDisabled(false))
	if !good {
		t.Fatalf("translated v3 hero is missing without JavaScript at %dpx in %s", width, mode)
	}
}

func checkLandingCarousel(t *testing.T, run func(...chromedp.Action), baseURL string, width int64) {
	t.Helper()
	run(chromedp.Navigate(baseURL), chromedp.WaitReady(".carousel .slide.selected"))
	var initial struct {
		Height float64
		Source string
		Good   bool
	}
	run(chromedp.Evaluate(`(() => {const slider=document.querySelector('.carousel .slider-wrapper'),first=document.querySelector('.carousel img[fetchpriority="high"]'),images=Array.from(document.querySelectorAll('.carousel img'));return {Height:slider.getBoundingClientRect().height,Source:first.currentSrc,Good:first.complete && first.loading==='eager' && images.filter(e=>e.fetchPriority==='low' && e.loading==='lazy').length===8};})()`, &initial))
	if !initial.Good {
		t.Fatalf("carousel image loading contract failed at %dpx", width)
	}
	run(chromedp.Sleep(5500 * time.Millisecond))
	var rotated bool
	run(chromedp.Evaluate(`(() => {const slider=document.querySelector('.carousel .slider-wrapper'),selected=Array.from(document.querySelectorAll('.carousel .slide.selected img')).find(e=>{const r=e.getBoundingClientRect();return r.right>0 && r.left<innerWidth;});return !!selected && selected.complete && selected.naturalWidth>0 && !selected.currentSrc.endsWith('mac-app.webp') && Math.abs(slider.getBoundingClientRect().height-`+strconv.FormatFloat(initial.Height, 'f', -1, 64)+`) < 1;})()`, &rotated))
	if !rotated {
		t.Fatalf("carousel autoplay, lazy slide or stable height failed at %dpx", width)
	}
	t.Logf("%dpx: theme/no-JS hero checks pass; carousel rotates with stable %.1fpx height", width, initial.Height)
}
