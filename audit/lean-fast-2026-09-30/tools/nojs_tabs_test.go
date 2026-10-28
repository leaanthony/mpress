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

// TestOrdinaryTabsWithoutJavaScript checks the default theme's tab fallback on
// a strict production build: with scripts blocked every panel of every
// ordinary tab group stays readable under its own visible label and the tab
// buttons are not presented, while the enhanced tablist keeps click, Home and
// End behaviour.
func TestOrdinaryTabsWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_NOJS_TABS_BIN")
	if binary == "" {
		t.Skip("set MPRESS_NOJS_TABS_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildNoScriptTabsFixture(t, binary))))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {390, 844}} {
		t.Run(fmt.Sprintf("blocked-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true))
			loadWithoutJavaScript(&b, server.URL+"/platforms/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			var groups []struct {
				TablistDisplay string
				Panels         []struct {
					Display, Text string
					Height        float64
					Label         string
					LabelHeight   float64
				}
			}
			b.js(`[...document.querySelectorAll('ot-tabs.mpress-tabs')].map(g => ({
              TablistDisplay:getComputedStyle(g.querySelector('[role="tablist"]')).display,
              Panels:[...g.querySelectorAll('[role="tabpanel"]')].map(p => {
                const label=getComputedStyle(p,'::before');
                return {Display:getComputedStyle(p).display,Height:p.getBoundingClientRect().height,
                  Text:p.textContent,Label:p.getAttribute('data-tab-label'),
                  LabelHeight:label.display==='block'&&label.content===JSON.stringify(p.getAttribute('data-tab-label'))?1:0};
              })
            }))`, &groups)
			if len(groups) != 2 {
				t.Fatalf("fixture has %d ordinary tab groups, want 2", len(groups))
			}
			wantLabels := [][]string{{"Windows", "Linux"}, {"macOS"}}
			for index, group := range groups {
				if group.TablistDisplay != "none" {
					t.Fatalf("inert tab buttons are still presented in group %d: %+v", index, group)
				}
				if len(group.Panels) != len(wantLabels[index]) {
					t.Fatalf("group %d has %d panels, want %d: %+v", index, len(group.Panels), len(wantLabels[index]), group)
				}
				for panelIndex, panel := range group.Panels {
					if panel.Display == "none" || panel.Height <= 0 {
						t.Fatalf("panel %d of group %d is hidden without JavaScript: %+v", panelIndex, index, panel)
					}
					if panel.Label != wantLabels[index][panelIndex] || panel.LabelHeight <= 0 {
						t.Fatalf("panel %d of group %d lacks its visible label: %+v", panelIndex, index, panel)
					}
					if !strings.Contains(panel.Text, "Steps "+wantLabels[index][panelIndex]) {
						t.Fatalf("panel %d of group %d is not readable: %+v", panelIndex, index, panel)
					}
				}
			}
		})
	}

	t.Run("enhanced", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(390, 844), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/platforms/"), chromedp.WaitReady("body"))
		var state struct {
			Tablist, Labels string
			Visible         int
		}
		b.js(`(() => {const g=document.querySelector('ot-tabs.mpress-tabs');
          return {Tablist:getComputedStyle(g.querySelector('[role="tablist"]')).display,
            Labels:getComputedStyle(g.querySelector('[role=tabpanel]'),'::before').content,
            Visible:[...g.querySelectorAll('[role="tabpanel"]')].filter(p=>getComputedStyle(p).display!=='none').length};})()`, &state)
		if state.Tablist == "none" || (state.Labels != "none" && state.Labels != "normal") || state.Visible != 1 {
			t.Fatalf("enhanced tabs changed their initial state: %+v", state)
		}

		b.run(chromedp.Click(`ot-tabs.mpress-tabs [role="tab"]:nth-child(2)`))
		b.poll(`(() => {const g=document.querySelector('ot-tabs.mpress-tabs');
          const panels=[...g.querySelectorAll('[role="tabpanel"]')];
          return g.querySelectorAll('[role="tab"][aria-selected="true"]').length===1 &&
            panels.filter(p=>getComputedStyle(p).display!=='none').length===1 &&
            panels[1].getBoundingClientRect().height>0;})()`, "clicking a tab did not leave exactly its panel visible")

		b.run(chromedp.Focus(`ot-tabs.mpress-tabs [role="tab"]`), chromedp.KeyEvent(kb.End))
		b.poll(`(() => {const g=document.querySelector('ot-tabs.mpress-tabs');
          const tabs=[...g.querySelectorAll('[role="tab"]')];
          return document.activeElement===tabs[tabs.length-1] &&
            tabs[tabs.length-1].getAttribute('aria-selected')==='true' &&
            g.querySelector('[role="tabpanel"]:not([hidden])')===g.querySelectorAll('[role="tabpanel"]')[1];})()`, "End did not select the last tab")
		b.run(chromedp.KeyEvent(kb.Home))
		b.poll(`(() => {const g=document.querySelector('ot-tabs.mpress-tabs');
          const tabs=[...g.querySelectorAll('[role="tab"]')];
          return document.activeElement===tabs[0] && tabs[0].getAttribute('aria-selected')==='true' &&
            g.querySelector('[role="tabpanel"]:not([hidden])')===g.querySelectorAll('[role="tabpanel"]')[0];})()`, "Home did not select the first tab")
	})
}

func buildNoScriptTabsFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	files := map[string]string{
		"mpress.yaml":      "site:\n  title: Tab fallback\n",
		"content/index.md": "---\ntitle: Home\nlayout: landing\n---\n\n# Home\n",
		"content/platforms.md": `---
title: Platforms
---

:::tabs
[Windows]
Steps Windows

Install on Windows.

[Linux]
Steps Linux

Install on Linux.
:::

:::tabs
[macOS]
Steps macOS

Install on macOS.
:::
`,
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
		t.Fatalf("build tab fixture: %v\n%s", err, output)
	}
	return filepath.Join(project, "site")
}
