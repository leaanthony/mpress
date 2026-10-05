package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type focusDialog struct{ name, trigger, dialog, close string }

func TestDialogOpenerFocus(t *testing.T) {
	binary := os.Getenv("MPRESS_DIALOG_BIN")
	if binary == "" {
		t.Skip("set MPRESS_DIALOG_BIN to the candidate CLI")
	}
	site := buildDialogFocusFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc)
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 3*time.Minute)
	defer timeout()
	dialogs := []focusDialog{
		{"image-1", ".mpress-image-expand[data-audit-image='0']", "#mpress-image-lightbox", "[data-image-lightbox-close]"},
		{"image-2", ".mpress-image-expand[data-audit-image='1']", "#mpress-image-lightbox", "[data-image-lightbox-close]"},
		{"contribution", ".header-contribute", "#mpress-contribute-dialog", "[data-contribute-close]"},
	}
	for _, dialog := range dialogs {
		for _, activation := range []string{"enter", "space", "mouse", "touch", "script"} {
			for _, closeMode := range []string{"escape", "button", "backdrop", "script"} {
				t.Run(fmt.Sprintf("%s/%s/%s", dialog.name, activation, closeMode), func(t *testing.T) {
					checkDialogFocus(t, ctx, server.URL, dialog, activation, closeMode)
				})
			}
		}
	}
}
func checkDialogFocus(t *testing.T, ctx context.Context, url string, dialog focusDialog, activation, closeMode string) {
	t.Helper()
	b := browserActions{t: t, ctx: ctx}
	b.run(chromedp.EmulateViewport(375, 667, chromedp.EmulateMobile, chromedp.EmulatePortrait), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(url+"/focus/"), chromedp.WaitReady("body"))
	// Label the fixture's two existing triggers without changing layout or focus.
	b.js(`document.querySelectorAll('.mpress-image-expand').forEach((e,i)=>e.dataset.auditImage=i)`, nil)
	for repeat := 0; repeat < 2; repeat++ {
		b.activateDialog(dialog, activation)
		b.closeDialog(dialog, closeMode)
		expected := "window.auditDialogOpener"
		if activation == "script" {
			expected = "window.auditDialogPriorFocus"
		}
		predicate := `!document.querySelector(` + browserQuote(dialog.dialog) + `).open&&document.activeElement===` + expected
		if strings.HasPrefix(dialog.name, "image-") {
			predicate += `&&document.querySelector('[data-image-lightbox-content]').childElementCount===0`
		}
		if err := chromedp.Run(ctx, chromedp.Poll(predicate, nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(500*time.Millisecond))); err != nil {
			var state string
			b.js(`document.activeElement.outerHTML.slice(0,200)`, &state)
			t.Fatalf("close did not restore expected %s focus on repeat %d: %s (%v)", dialog.name, repeat, state, err)
		}
	}
}

func buildDialogFocusFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		c := exec.Command(binary, args...)
		c.Dir = project
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, out)
		}
	}
	run("init", ".")
	path := filepath.Join(project, "mpress.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := strings.Replace(string(data), "contribution:\n  enabled: false", "contribution:\n  enabled: true", 1)
	cfg = strings.Replace(cfg, `repository: ""`, `repository: https://github.com/leaanthony/mpress`, 1)
	if !strings.Contains(cfg, "contribution:\n  enabled: true") {
		t.Fatal("fixture did not enable contributions")
	}
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("mpress.yaml", cfg)
	write("static/diagram.svg", `<svg xmlns="http://www.w3.org/2000/svg" width="96" height="48"><rect width="96" height="48" fill="teal"/></svg>`)
	write("content/focus.md", `---
title: Dialog focus
---

@image{src="/diagram.svg" alt="First diagram" expand}

@image{src="/diagram.svg" alt="Second diagram" expand}
`)
	run("build", "--strict")
	return filepath.Join(project, "site")
}
func (b *browserActions) activateDialog(d focusDialog, activation string) {
	b.t.Helper()
	b.js(`(() => {const e=document.querySelector(`+browserQuote(d.trigger)+`);if(!e)throw Error('missing opener');e.scrollIntoView({block:'center'});window.auditDialogPriorFocus=document.querySelector('#theme');window.auditDialogPriorFocus.focus({preventScroll:true});if(document.activeElement!==window.auditDialogPriorFocus)throw Error('prior focus was not established');window.auditDialogOpener=e;})()`, nil)
	b.run(chromedp.Poll(`(() => {const e=window.auditDialogOpener,r=e.getBoundingClientRect();return r.width>0&&r.height>0&&e.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2));})()`, nil, chromedp.WithPollingTimeout(time.Second)))
	var point struct{ X, Y float64 }
	b.js(`(() => {const r=window.auditDialogOpener.getBoundingClientRect();return {X:r.x+r.width/2,Y:r.y+r.height/2};})()`, &point)

	switch activation {
	case "enter", "space":
		b.js(`window.auditDialogOpener.focus({preventScroll:true})`, nil)
		key := kb.Enter
		if activation == "space" {
			key = " "
		}
		b.run(chromedp.KeyEvent(key))
	case "mouse":
		b.run(chromedp.MouseClickXY(point.X, point.Y))
	case "touch":
		b.run(browserTap(point.X, point.Y))
	case "script":
		b.js(`window.auditDialogOpener.click()`, nil)
	}
	b.run(chromedp.Poll(`document.querySelector(`+browserQuote(d.dialog)+`).open`, nil, chromedp.WithPollingTimeout(time.Second)))
}
func (b *browserActions) closeDialog(d focusDialog, mode string) {
	b.t.Helper()
	switch mode {
	case "escape":
		b.run(chromedp.KeyEvent(kb.Escape))
	case "button":
		b.run(chromedp.Click(d.dialog+" "+d.close, chromedp.ByQuery))
	case "backdrop":
		var hit bool
		b.js(`document.elementFromPoint(1,1)===document.querySelector(`+browserQuote(d.dialog)+`)`, &hit)
		if !hit {
			b.t.Fatal("backdrop input location did not hit dialog")
		}
		b.run(chromedp.MouseClickXY(1, 1))
	case "script":
		b.js(`document.querySelector(`+browserQuote(d.dialog)+`).close()`, nil)
	}
}
