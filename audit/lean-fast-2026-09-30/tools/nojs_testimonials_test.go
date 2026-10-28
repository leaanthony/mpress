package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func TestTestimonialsWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_TESTIMONIAL_BIN")
	if binary == "" {
		t.Skip("set MPRESS_TESTIMONIAL_BIN to the candidate CLI")
	}
	project := t.TempDir()
	initialise := exec.Command(binary, "init", ".")
	initialise.Dir = project
	if output, err := initialise.CombinedOutput(); err != nil {
		t.Fatalf("initialise: %v\n%s", err, output)
	}
	content := "---\ntitle: Reader experiences\n---\n\n"
	for _, quotes := range []string{"First quote\n— Ada, App team\n---\nSecond quote\n— Bea, Docs team\n---\nThird quote\n— Cy, Release team", "Independent quote\n— Dee, Tools team\n---\nOther quote\n— Eli, Desktop team"} {
		content += "\n:::testimonials\n" + quotes + "\n:::\n"
	}
	if err := os.WriteFile(filepath.Join(project, "content", "guide.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command(binary, "build", "--strict")
	build.Dir = project
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	server := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(project, "site"))))
	defer server.Close()
	ctx, stop := newHeaderBrowser(t)
	defer stop()
	for _, width := range []int64{320, 1280} {
		for _, enhanced := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d-js-%v", width, enhanced), func(t *testing.T) {
				b := newBrandPage(t, ctx)
				b.run(chromedp.EmulateViewport(width, 900))
				if enhanced {
					b.run(chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
				} else {
					loadWithoutJavaScript(&b, server.URL+"/guide/")
				}
				var groups []struct {
					Nav   string
					Items []struct {
						Text, Aria, Display string
						Height              float64
					}
				}
				b.js(`[...document.querySelectorAll('.mpress-testimonials')].map(g=>({Nav:getComputedStyle(g.querySelector('nav')).display,Items:[...g.querySelectorAll('blockquote')].map(q=>({Text:q.textContent,Aria:q.getAttribute('aria-hidden')||'',Display:getComputedStyle(q).display,Height:q.getBoundingClientRect().height}))}))`, &groups)
				if len(groups) != 2 {
					t.Fatalf("want two groups: %+v", groups)
				}
				for g, group := range groups {
					if len(group.Items) != 3-g {
						t.Fatalf("wrong quote count: %+v", group)
					}
					if (group.Nav == "none") == enhanced {
						t.Fatalf("wrong navigation availability: %+v", group)
					}
					for i, item := range group.Items {
						visible := !enhanced || i == 0
						if item.Text == "" || (item.Display != "none" && item.Height > 0) != visible {
							t.Fatalf("wrong quote visibility: %+v", item)
						}
						if !enhanced && item.Aria == "true" {
							t.Fatalf("readable quote excluded from accessibility: %+v", item)
						}
						if enhanced && item.Aria != fmt.Sprint(i != 0) {
							t.Fatalf("wrong enhanced accessibility state: %+v", item)
						}
					}
				}
				if enhanced {
					b.run(chromedp.Focus(`.mpress-testimonials-dot:nth-child(2)`), chromedp.KeyEvent(kb.Enter))
					b.poll(`(()=>{const g=document.querySelectorAll('.mpress-testimonials');return g[0].querySelectorAll('blockquote')[1].getAttribute('aria-hidden')==='false'&&g[0].querySelectorAll('button')[1].getAttribute('aria-current')==='true'&&g[1].querySelectorAll('button')[0].getAttribute('aria-current')==='true'})()`, "Enter did not switch only its group")
					b.run(chromedp.Focus(`.mpress-testimonials-dot:nth-child(3)`), chromedp.KeyEvent(" "))
					b.poll(`(()=>{const g=document.querySelectorAll('.mpress-testimonials');return g[0].querySelectorAll('blockquote')[2].getAttribute('aria-hidden')==='false'&&g[0].querySelectorAll('button')[2].getAttribute('aria-current')==='true'&&g[1].querySelectorAll('button')[0].getAttribute('aria-current')==='true'})()`, "Space did not switch only its group")
				}
			})
		}
	}
}
