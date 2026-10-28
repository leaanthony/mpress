package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

// TestAPIPlaygroundWithoutJavaScript checks two same-origin playgrounds on a
// strict build: with scripts blocked the request reference stays readable and
// no request is sent, while the enhanced Send action still answers native Enter.
func TestAPIPlaygroundWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_PLAYGROUND_BIN")
	if binary == "" {
		t.Skip("set MPRESS_PLAYGROUND_BIN to the candidate CLI")
	}
	project := t.TempDir()
	initialise := exec.Command(binary, "init", ".")
	initialise.Dir = project
	if output, err := initialise.CombinedOutput(); err != nil {
		t.Fatalf("initialise: %v\n%s", err, output)
	}
	content := "---\ntitle: API reference\n---\n\n" +
		":::api-playground{method=GET path=\"/v1/mock\"}\nFetch the mock resource.\n:::\n\n" +
		":::api-playground{method=GET path=\"/v1/other\"}\nFetch the other resource.\n:::\n"
	if err := os.WriteFile(filepath.Join(project, "content", "guide.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command(binary, "build", "--strict")
	build.Dir = project
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}

	var mockRequests atomic.Int64
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/mock", func(w http.ResponseWriter, _ *http.Request) {
		mockRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mock":true}`))
	})
	mux.Handle("/", http.FileServer(http.Dir(filepath.Join(project, "site"))))
	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, stop := newHeaderBrowser(t)
	defer stop()

	paths := []string{"/v1/mock", "/v1/other"}
	for _, width := range []int64{320, 1280} {
		t.Run(fmt.Sprintf("blocked-%d", width), func(t *testing.T) {
			mockRequests.Store(0)
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(width, 900))
			loadWithoutJavaScript(&b, server.URL+"/guide/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			var playgrounds []struct {
				Method, Path, Description, URL, Actions string
				DescriptionHeight, URLHeight            float64
			}
			b.js(`[...document.querySelectorAll('.mpress-api-playground')].map(p => ({
              Method:p.querySelector('.mpress-api-method').textContent,
              Path:p.querySelector('.mpress-api-path').textContent,
              Description:p.querySelector('.mpress-api-description').textContent.trim(),
              DescriptionHeight:p.querySelector('.mpress-api-description').getBoundingClientRect().height,
              URL:p.querySelector('.mpress-api-pg-url-input').value,
              URLHeight:p.querySelector('.mpress-api-pg-url-input').getBoundingClientRect().height,
              Actions:getComputedStyle(p.querySelector('.mpress-api-pg-actions')).display
            }))`, &playgrounds)
			if len(playgrounds) != 2 {
				t.Fatalf("want two playgrounds, got %d", len(playgrounds))
			}
			for index, playground := range playgrounds {
				if playground.Method != "GET" || playground.Path != paths[index] || !strings.HasPrefix(playground.Description, "Fetch the") || playground.DescriptionHeight <= 0 {
					t.Fatalf("playground %d request reference is not readable: %+v", index, playground)
				}
				if playground.URL != paths[index] || playground.URLHeight <= 0 {
					t.Fatalf("playground %d URL is not readable: %+v", index, playground)
				}
				if playground.Actions != "none" {
					t.Fatalf("playground %d still presents its inert Send action: %+v", index, playground)
				}
			}
			if got := mockRequests.Load(); got != 0 {
				t.Fatalf("script-blocked page sent %d API requests", got)
			}
		})

		t.Run(fmt.Sprintf("enhanced-%d", width), func(t *testing.T) {
			mockRequests.Store(0)
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(width, 900), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
			b.poll(`document.documentElement.classList.contains('js')`, "enhanced page is not marked as scripted")
			b.js(`document.querySelectorAll('.mpress-api-playground')[0].querySelector('.mpress-api-pg-send').focus()`, nil)
			b.run(chromedp.KeyEvent(kb.Enter))
			b.poll(`(() => {
              const p=document.querySelectorAll('.mpress-api-playground')[0],r=p.querySelector('.mpress-api-pg-response');
              return !r.hidden && r.querySelector('.mpress-api-pg-response-status').textContent.startsWith('200') && r.querySelector('code').textContent.includes('"mock": true');
            })()`, "native Enter did not send the first playground request")
			if got := mockRequests.Load(); got != 1 {
				t.Fatalf("first playground sent %d requests, want 1", got)
			}
			var secondHidden bool
			b.js(`document.querySelectorAll('.mpress-api-playground')[1].querySelector('.mpress-api-pg-response').hidden`, &secondHidden)
			if !secondHidden {
				t.Fatal("sending the first playground changed the second one")
			}
		})
	}
}
