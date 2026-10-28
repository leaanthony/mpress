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

type tabsState struct {
	ScrollLeft, ScrollWidth, ClientWidth          float64
	Left, Right, Top, Height, LastLeft, LastRight float64
	Selected, Panel, Active, PageX, PageY         float64
}

// TestPhoneTabsTouchScroll reproduces the wide default-theme tab strip: real
// horizontal swipes on a phone must reveal the last of 12 tabs, tapping it must
// select instructions 12 without moving the document, and End/Home keyboard
// focus must bring the selected tab into view. Short desktop groups stay put.
func TestPhoneTabsTouchScroll(t *testing.T) {
	binary := os.Getenv("MPRESS_HEADER_NAV_BIN")
	if binary == "" {
		t.Skip("set MPRESS_HEADER_NAV_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildTabsTouchFixture(t, binary))))
	defer server.Close()

	probe := func(b browserActions) tabsState {
		t.Helper()
		var s tabsState
		b.js(`(() => {
          const list=document.querySelector('.mpress-tabs [role="tablist"]');
          const tabs=[...list.querySelectorAll('[role="tab"]')];
          const sel=list.querySelector('[role="tab"][aria-selected="true"]');
          const panel=list.parentElement.querySelector('[role="tabpanel"]:not([hidden])');
          const r=list.getBoundingClientRect(), last=tabs[tabs.length-1].getBoundingClientRect();
          return {ScrollLeft:list.scrollLeft,ScrollWidth:list.scrollWidth,ClientWidth:list.clientWidth,
            Left:r.left,Right:r.right,Top:r.top,Height:r.height,
            LastLeft:last.left,LastRight:last.right,
            Selected:sel?Number(sel.textContent.replace(/\D/g,'')):0,
            Panel:panel?Number(panel.textContent.replace(/\D/g,'')):0,
            Active:Number((document.activeElement?.textContent||'').replace(/\D/g,'')),
            PageX:window.scrollX,PageY:window.scrollY};
        })()`, &s)
		return s
	}
	visible := func(s tabsState) bool { return s.LastLeft >= s.Left-1 && s.LastRight <= s.Right+1 }

	for _, size := range [][2]int64{{320, 844}, {390, 844}} {
		t.Run(fmt.Sprintf("phone-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/platforms/"), chromedp.WaitReady("body"))
			start := probe(b)
			if start.ScrollWidth <= start.ClientWidth || start.LastRight <= start.Right {
				t.Fatalf("12 tabs must overflow horizontally at %d: %+v", size[0], start)
			}
			y := start.Top + start.Height/2
			for attempt := 0; attempt < 12 && !visible(probe(b)); attempt++ {
				b.run(swipe(start.Right-.2*start.ClientWidth, y, -.7*start.ClientWidth, 0))
			}
			reached := probe(b)
			if !visible(reached) || reached.ScrollLeft <= 0 {
				t.Fatalf("horizontal swipes did not reach the last tab: %+v", reached)
			}
			x := reached.LastLeft + (reached.LastRight-reached.LastLeft)/2
			b.run(browserTap(x, y))
			b.poll(`(() => {const p=document.querySelector('.mpress-tabs [role="tabpanel"]:not([hidden])');return p&&p.textContent.includes('Instructions 12');})()`, "tapping the last platform tab did not select its instructions")
			after := probe(b)
			if after.PageX != start.PageX || after.PageY != start.PageY {
				t.Fatalf("touching tabs moved the document: before %g,%g after %g,%g", start.PageX, start.PageY, after.PageX, after.PageY)
			}

			b.run(chromedp.Focus(`.mpress-tabs [role="tab"]`), chromedp.KeyEvent(kb.End))
			b.poll(`document.activeElement===document.querySelector('.mpress-tabs [role="tablist"]').lastElementChild`, "End did not focus the last tab")
			end := probe(b)
			if end.Selected != 12 || end.Panel != 12 || end.Active != 12 || end.LastLeft < end.Left-1 || end.LastRight > end.Right+1 {
				t.Fatalf("End-focused tab is not scrolled into view: %+v", end)
			}
			b.run(chromedp.KeyEvent(kb.Home))
			b.poll(`document.activeElement===document.querySelector('.mpress-tabs [role="tab"]')`, "Home did not focus the first tab")
			home := probe(b)
			if home.Selected != 1 || home.Panel != 1 || home.Active != 1 || home.ScrollLeft != 0 {
				t.Fatalf("Home did not return to the first tab: %+v", home)
			}
		})
	}

	t.Run("desktop-short-group", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(1280, 900), chromedp.Navigate(server.URL+"/short/"), chromedp.WaitReady("body"))
		s := probe(b)
		if s.ScrollWidth > s.ClientWidth+1 || s.LastRight > s.Right+1 {
			t.Fatalf("short desktop tab group must not scroll: %+v", s)
		}
	})
}

func buildTabsTouchFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Dir = project
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("CLI %v: %v\n%s", args, err, output)
		}
	}
	run("init", ".")
	if err := os.RemoveAll(filepath.Join(project, "content")); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		target := filepath.Join(project, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	page := func(title string, tabs int) string {
		var body strings.Builder
		fmt.Fprintf(&body, "---\ntitle: %s\n---\n\n:::tabs\n", title)
		for n := 1; n <= tabs; n++ {
			fmt.Fprintf(&body, "[Platform %d]\nInstructions %d\n", n, n)
		}
		body.WriteString(":::endtabs\n\n")
		for paragraph := 1; paragraph <= 40; paragraph++ {
			fmt.Fprintf(&body, "Paragraph %d keeps the page tall enough to verify the document does not move.\n\n", paragraph)
		}
		return body.String()
	}
	write("content/index.md", "---\ntitle: Home\nlayout: landing\n---\n\n# Home\n")
	write("content/platforms.md", page("Platforms", 12))
	write("content/short.md", page("Short", 3))
	run("build", "--strict")
	return filepath.Join(project, "site")
}
