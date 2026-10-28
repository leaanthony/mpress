package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type carouselState struct {
	Visible []int
	Status  string
	Current int
}

const carouselStateProbe = `[...document.querySelectorAll('[data-carousel]')].map(c => ({
  Visible:[...c.querySelectorAll('[data-carousel-slide]')]
    .map((s,i) => getComputedStyle(s).display !== 'none' && s.getBoundingClientRect().height > 0 ? i : -1)
    .filter(i => i >= 0),
  Status:(c.querySelector('[data-carousel-status]') || {}).textContent,
  Current:[...c.querySelectorAll('[data-carousel-dot]')].findIndex(d => d.getAttribute('aria-current') === 'true')
}))`

// TestCarouselWithoutJavaScript checks that a blocked-JavaScript reader gets
// every authored slide with its links, while the enhanced carousel keeps one
// slide at a time and native Enter/Space navigation per document-scoped
// instance.
func TestCarouselWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_CAROUSEL_BIN")
	if binary == "" {
		t.Skip("set MPRESS_CAROUSEL_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildCarouselFixture(t, binary))))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {1280, 900}} {
		t.Run(fmt.Sprintf("blocked-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(size[0] < 600))
			loadWithoutJavaScript(&b, server.URL+"/guide/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			var state []struct {
				Slides []struct {
					Display, Text, Href string
					Width, Height       float64
					Hit                 bool
				}
				Controls string
			}
			b.js(`[...document.querySelectorAll('[data-carousel]')].map(c => ({
              Slides:[...c.querySelectorAll('[data-carousel-slide]')].map(s => {
                const link=s.querySelector('a');
                link.scrollIntoView({block:'center'});
                const r=link.getBoundingClientRect();
                const hit=document.elementFromPoint(r.x+r.width/2, r.y+r.height/2);
                return {Display:getComputedStyle(s).display, Text:s.textContent.trim(), Href:link.getAttribute('href'),
                  Width:r.width, Height:r.height, Hit:Boolean(hit && (hit === link || link.contains(hit)))};
              }),
              Controls:getComputedStyle(c.querySelector('.mpress-carousel-controls')).display
            }))`, &state)
			if len(state) != 2 {
				t.Fatalf("fixture has %d carousels, want 2", len(state))
			}
			for index, carousel := range state {
				if carousel.Controls != "none" {
					t.Fatalf("carousel %d shows inert controls without JavaScript: %+v", index, carousel.Controls)
				}
				if len(carousel.Slides) != 3-index {
					t.Fatalf("carousel %d has %d slides, want %d", index, len(carousel.Slides), 3-index)
				}
				for slideIndex, slide := range carousel.Slides {
					if slide.Display == "none" || slide.Width <= 0 || slide.Height <= 0 || slide.Href == "" || !slide.Hit {
						t.Fatalf("slide %d of carousel %d is not readable without JavaScript: %+v", slideIndex, index, slide)
					}
					if slide.Text == "" {
						t.Fatalf("slide %d of carousel %d lost its content: %+v", slideIndex, index, slide)
					}
				}
			}
		})

		t.Run(fmt.Sprintf("enhanced-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(size[0] < 600), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
			var initial []carouselState
			b.js(carouselStateProbe, &initial)
			if len(initial) != 2 || len(initial[0].Visible) != 1 || initial[0].Visible[0] != 0 || initial[0].Status != "1 / 3" || initial[0].Current != 0 ||
				len(initial[1].Visible) != 1 || initial[1].Visible[0] != 0 || initial[1].Status != "1 / 2" || initial[1].Current != 0 {
				t.Fatalf("enhanced carousels did not start on their first slide: %+v", initial)
			}

			first := `[data-carousel][aria-label="Reading"]`
			b.run(chromedp.Focus(first+` [data-carousel-next]`), chromedp.KeyEvent(kb.Enter))
			b.poll(`(() => {const c=document.querySelectorAll('[data-carousel]');
              return c[0].querySelector('[data-carousel-status]').textContent==='2 / 3' &&
                c[0].querySelectorAll('[data-carousel-slide]')[1].hidden===false &&
                c[0].querySelectorAll('[data-carousel-slide]')[0].hidden===true &&
                c[0].querySelectorAll('[data-carousel-dot]')[1].getAttribute('aria-current')==='true' &&
                c[1].querySelector('[data-carousel-status]').textContent==='1 / 2' &&
                c[1].querySelectorAll('[data-carousel-slide]')[0].hidden===false;})()`, "Enter on the next button did not advance only its own carousel")

			b.run(chromedp.Focus(first+` [data-carousel-dot="2"]`), chromedp.KeyEvent(" "))
			b.poll(`(() => {const c=document.querySelectorAll('[data-carousel]');
              return c[0].querySelector('[data-carousel-status]').textContent==='3 / 3' &&
                c[0].querySelectorAll('[data-carousel-slide]')[2].hidden===false &&
                c[0].querySelectorAll('[data-carousel-slide]')[1].hidden===true &&
                c[0].querySelectorAll('[data-carousel-dot]')[2].getAttribute('aria-current')==='true' &&
                c[1].querySelector('[data-carousel-status]').textContent==='1 / 2';})()`, "Space on a dot did not select only its own slide")
		})
	}
}

func buildCarouselFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	files := map[string]string{
		"mpress.yaml": "site:\n  title: Carousel fallback\n",
		"content/guide.md": `---
title: Guide
---

:::carousel{label="Reading"}
## One

[Open one](/one/)

---

## Two

[Open two](/two/)

---

## Three

[Open three](/three/)
:::

:::carousel{label="Navigation"}
## Alpha

[Open alpha](/alpha/)

---

## Beta

[Open beta](/beta/)
:::
`,
	}
	for _, name := range []string{"one", "two", "three", "alpha", "beta"} {
		files["content/"+name+".md"] = "---\ntitle: " + name + "\n---\n\n# " + name + "\n"
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
		t.Fatalf("build carousel fixture: %v\n%s", err, output)
	}
	return filepath.Join(project, "site")
}
