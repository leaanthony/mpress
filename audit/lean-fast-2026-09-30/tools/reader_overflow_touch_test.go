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
)

// TestReaderOverflowTouch checks that wide reader content can be inspected by
// finger swipes without moving the page, and that the document still scrolls
// vertically afterwards. It covers code, terminal, plain-table and data-table
// overflow on phone viewports.
func TestReaderOverflowTouch(t *testing.T) {
	binary := os.Getenv("MPRESS_HEADER_NAV_BIN")
	if binary == "" {
		t.Skip("set MPRESS_HEADER_NAV_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	site := buildReaderOverflowFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()

	pages := []struct {
		name     string
		route    string
		selector string
	}{
		{"code", "/code/", ".mpress-codeframe pre"},
		{"terminal", "/terminal/", ".mpress-terminal pre"},
		{"plain", "/plain/", "article table"},
		{"data", "/data/", ".mpress-data-table .mpress-table-scroll"},
	}
	for _, size := range [][2]int64{{320, 844}, {390, 844}} {
		for _, page := range pages {
			t.Run(fmt.Sprintf("%s-%dx%d", page.name, size[0], size[1]), func(t *testing.T) {
				b := newBrandPage(t, ctx)
				b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+page.route), chromedp.WaitReady("body"))

				var box struct {
					X, Y, Width, Height         float64
					ScrollWidth, ClientWidth    float64
					ScrollLeft                  float64
					PageX, PageY, BodyScrollTop float64
				}
				b.js(fmt.Sprintf(`(() => {
                  const e=document.querySelector(%s);
                  if(!e)return {};
                  const r=e.getBoundingClientRect();
                  return {X:r.x,Y:r.y,Width:r.width,Height:r.height,
                    ScrollWidth:e.scrollWidth,ClientWidth:e.clientWidth,ScrollLeft:e.scrollLeft,
                    PageX:window.scrollX,PageY:window.scrollY,BodyScrollTop:document.scrollingElement.scrollTop};
                })()`, browserQuote(page.selector)), &box)
				if box.Width <= 0 || box.Height <= 0 || box.Y < 0 || box.Y >= float64(size[1]) {
					t.Fatalf("%s is missing or off-screen at %dx%d: %+v", page.selector, size[0], size[1], box)
				}
				if box.ScrollWidth <= box.ClientWidth {
					t.Fatalf("%s does not overflow horizontally: scrollWidth %g <= clientWidth %g", page.selector, box.ScrollWidth, box.ClientWidth)
				}

				if box.X < 0 || box.X+box.Width > float64(size[0]) || box.Y+box.Height >= 780 {
					t.Fatalf("widget must fit above the separate body swipe: %+v", box)
				}
				y := box.Y + min(box.Height/2, 90)
				b.run(swipe(box.X+box.Width*.8, y, -box.Width*.6, 0))

				var after struct {
					ScrollLeft                  float64
					PageX, PageY, BodyScrollTop float64
				}
				b.js(fmt.Sprintf(`(() => {
                  const e=document.querySelector(%s);
                  return {ScrollLeft:e.scrollLeft,PageX:window.scrollX,PageY:window.scrollY,BodyScrollTop:document.scrollingElement.scrollTop};
                })()`, browserQuote(page.selector)), &after)
				if after.ScrollLeft <= box.ScrollLeft {
					t.Fatalf("horizontal swipe did not reveal off-screen content: scrollLeft %g, want > %g", after.ScrollLeft, box.ScrollLeft)
				}
				if after.PageX != box.PageX || after.PageY != box.PageY || after.BodyScrollTop != box.BodyScrollTop {
					t.Fatalf("horizontal swipe moved the document: before x=%g y=%g top=%g, after x=%g y=%g top=%g",
						box.PageX, box.PageY, box.BodyScrollTop, after.PageX, after.PageY, after.BodyScrollTop)
				}

				b.run(swipe(float64(size[0])/2, 780, 0, -250))
				b.poll(fmt.Sprintf(`document.scrollingElement.scrollTop>%g`, after.BodyScrollTop), "vertical body swipe below the widget did not advance the document")
			})
		}
	}
}

func buildReaderOverflowFixture(t *testing.T, binary string) string {
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
	configPath := filepath.Join(project, "mpress.yaml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config = []byte(strings.Replace(string(config), `title: "My Project"`, `title: "Overflow"`, 1))
	if err := os.WriteFile(configPath, config, 0o644); err != nil {
		t.Fatal(err)
	}
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
	long := strings.Repeat("Long_Unbroken_Identifier_", 10)
	page := func(title, widget string) string {
		var body strings.Builder
		fmt.Fprintf(&body, "---\ntitle: %s\n---\n\n%s\n\n", title, widget)
		for paragraph := 1; paragraph <= 90; paragraph++ {
			fmt.Fprintf(&body, "Paragraph %d keeps the page tall enough to keep reading after the widget.\n\n", paragraph)
		}
		return body.String()
	}
	write("content/index.md", "---\ntitle: Home\nlayout: landing\n---\n\n# Home\n")
	write("content/code.md", page("Code", "```go\nvalue := \""+long+"\"\n```"))
	write("content/terminal.md", page("Terminal", "@terminal{title=\"Inspect\" language=sh}\n$ echo "+long+"\n@end"))
	write("content/plain.md", page("Plain table", "| Name | Value | Detail |\n| --- | --- | --- |\n| "+long+" | "+long+" | "+long+" |\n| second | row | value |\n"))
	write("content/data.md", page("Data table", "@table{header search sort}\n| Name | Value | Detail |\n| --- | --- | --- |\n| "+long+" | "+long+" | "+long+" |\n| second | row | value |\n@end"))
	write("content/_nav.yaml", "- label: Home\n  link: /\n- label: Code\n  link: /code/\n- label: Terminal\n  link: /terminal/\n- label: Plain\n  link: /plain/\n- label: Data\n  link: /data/\n")
	run("build", "--strict")
	return filepath.Join(project, "site")
}
