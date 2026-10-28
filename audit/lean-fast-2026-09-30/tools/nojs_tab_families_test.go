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

// TestFileAndPreviewTabsWithoutJavaScript checks the shared fallback on a
// strict production build: with scripts blocked every file-tab and
// preview-tab panel stays readable under its own label and the tab buttons are
// not presented. The enhanced groups keep one visible panel, clicks and
// Home/End, and stay independent of each other.
func TestFileAndPreviewTabsWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_NOJS_TAB_FAMILIES_BIN")
	if binary == "" {
		t.Skip("set MPRESS_NOJS_TAB_FAMILIES_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildTabFamiliesFixture(t, binary))))
	defer server.Close()

	const familyProbe = `(() => [...document.querySelectorAll(%s)].map(g => ({
      List:getComputedStyle(g.querySelector('[role="tablist"]')).display,
      Panels:[...g.querySelectorAll('[role="tabpanel"]')].map(p => {
        const label=getComputedStyle(p,'::before');
        return {Text:p.textContent,Label:p.getAttribute('data-tab-label'),
          LabelVisible:label.display==='block'&&label.content===JSON.stringify(p.getAttribute('data-tab-label')),
          Visible:getComputedStyle(p).display!=='none'&&p.getBoundingClientRect().height>0};
      })
    })))()`

	for _, size := range [][2]int64{{320, 667}, {390, 844}, {1280, 900}} {
		t.Run(fmt.Sprintf("blocked-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]))
			loadWithoutJavaScript(&b, server.URL+"/families/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			for _, family := range []struct {
				Selector string
				Labels   []string
				Bodies   []string
			}{
				{"ot-tabs.mpress-file-tabs", []string{"main.go", "app.go", "requiredMethod"}, []string{"package main", "package app", "func Run() error"}},
				{"ot-tabs.mpress-preview-tabs", []string{"Example", "Result"}, []string{"Example body", "Result body"}},
			} {
				var groups []struct {
					List   string
					Panels []struct {
						Text, Label  string
						LabelVisible bool
						Visible      bool
					}
				}
				b.js(fmt.Sprintf(familyProbe, browserQuote(family.Selector)), &groups)
				if len(groups) != 1 {
					t.Fatalf("%s: found %d groups, want 1", family.Selector, len(groups))
				}
				group := groups[0]
				if group.List != "none" {
					t.Fatalf("%s: inert tab buttons are still presented: %+v", family.Selector, group)
				}
				if len(group.Panels) != len(family.Labels) {
					t.Fatalf("%s: has %d panels, want %d: %+v", family.Selector, len(group.Panels), len(family.Labels), group)
				}
				for index, want := range family.Labels {
					panel := group.Panels[index]
					if panel.Label != want || !panel.Visible || !panel.LabelVisible || !strings.Contains(panel.Text, family.Bodies[index]) {
						t.Fatalf("%s: panel %q is not readable under its label: %+v", family.Selector, want, panel)
					}
				}
			}
		})
	}

	t.Run("enhanced", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(390, 844), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/families/"), chromedp.WaitReady("body"))
		state := func() (int, int, int, int) {
			var counts [4]int
			b.js(`(() => {
              const file=document.querySelector('ot-tabs.mpress-file-tabs');
              const preview=document.querySelector('ot-tabs.mpress-preview-tabs');
              const visible=g=>[...g.querySelectorAll('[role="tabpanel"]')].filter(p=>getComputedStyle(p).display!=='none').length;
              return [visible(file),visible(preview),
                file.querySelector('[role="tablist"]').querySelectorAll('[role="tab"][aria-selected="true"]').length,
                preview.querySelector('[role="tablist"]').querySelectorAll('[role="tab"][aria-selected="true"]').length];})()`, &counts)
			return counts[0], counts[1], counts[2], counts[3]
		}
		if file, preview, fileSelected, previewSelected := state(); file != 1 || preview != 1 || fileSelected != 1 || previewSelected != 1 {
			t.Fatalf("enhanced groups must show one selected panel each: file=%d preview=%d selected=%d/%d", file, preview, fileSelected, previewSelected)
		}

		b.run(chromedp.Click(`ot-tabs.mpress-file-tabs [role="tab"]:nth-child(3)`))
		b.poll(`(() => {const f=document.querySelector('ot-tabs.mpress-file-tabs'),p=document.querySelector('ot-tabs.mpress-preview-tabs');
          return f.querySelector('[role="tabpanel"]:not([hidden])')===f.querySelectorAll('[role="tabpanel"]')[2] &&
            p.querySelector('[role="tabpanel"]:not([hidden])')===p.querySelectorAll('[role="tabpanel"]')[0];})()`,
			"clicking a file tab changed the preview group or did not select its panel")

		b.run(chromedp.Click(`ot-tabs.mpress-preview-tabs [role="tab"]:nth-child(2)`))
		b.poll(`document.querySelector('ot-tabs.mpress-preview-tabs [role="tabpanel"]:not([hidden])').textContent.includes('Result body')`, "clicking a preview tab did not select its panel")

		b.run(chromedp.Focus(`ot-tabs.mpress-preview-tabs [role="tab"]`), chromedp.KeyEvent(kb.End))
		b.poll(`(() => {const g=document.querySelector('ot-tabs.mpress-preview-tabs'),tabs=[...g.querySelectorAll('[role="tab"]')];
          return document.activeElement===tabs[1] && tabs[1].getAttribute('aria-selected')==='true' &&
            g.querySelector('[role="tabpanel"]:not([hidden])')===g.querySelectorAll('[role="tabpanel"]')[1];})()`, "End did not select the last preview tab")
		b.run(chromedp.KeyEvent(kb.Home))
		b.poll(`(() => {const g=document.querySelector('ot-tabs.mpress-preview-tabs'),tabs=[...g.querySelectorAll('[role="tab"]')];
          return document.activeElement===tabs[0] && tabs[0].getAttribute('aria-selected')==='true' &&
            g.querySelector('[role="tabpanel"]:not([hidden])')===g.querySelectorAll('[role="tabpanel"]')[0];})()`, "Home did not select the first preview tab")
	})
}

func buildTabFamiliesFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	files := map[string]string{
		"mpress.yaml":      "site:\n  title: Tab families\n",
		"content/index.md": "---\ntitle: Home\nlayout: landing\n---\n\n# Home\n",
		"content/families.md": `---
title: Families
---

@file-tabs
[main.go]

~~~go
package main
~~~

[app.go]

~~~go
package app
~~~

[requiredMethod]

~~~go
func Run() error
~~~
@end

@preview-tabs
[Example]

Example body

[Result]

Result body
@end
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
		t.Fatalf("CLI build: %v\n%s", err, output)
	}
	return filepath.Join(project, "site")
}
