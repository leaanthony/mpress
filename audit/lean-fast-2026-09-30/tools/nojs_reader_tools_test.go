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

// TestReaderToolsWithoutJavaScript checks a strict production build: with
// scripts blocked the accessibility, theme and contribution launchers are not
// presented while the document and native utility links stay usable; with
// scripts enabled all three still change state.
func TestReaderToolsWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_NOJS_READER_BIN")
	if binary == "" {
		t.Skip("set MPRESS_NOJS_READER_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildReaderToolsFixture(t, binary))))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {1280, 900}} {
		t.Run(fmt.Sprintf("blocked-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]))
			loadWithoutJavaScript(&b, server.URL+"/guide/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			var state struct {
				Accessibility, Theme, Contribute string
				Content                          bool
				LanguageTrigger                  string
				LanguageLinks                    int
			}
			b.js(`(() => {
              const shown=s=>{const e=document.querySelector(s);return e?getComputedStyle(e).display:'missing'};
              const language=document.querySelector('.language-select');
              return {
                Accessibility:shown('.accessibility-select'),Theme:shown('#mpress-theme-toggle'),
                Contribute:shown('[data-mpress-contribute]'),
                Content:Boolean(document.querySelector('main')&&document.querySelector('main').textContent.includes('Reader tools stay reachable without scripts')),
                LanguageTrigger:language?shown('.language-select .utility-menu-trigger'):'missing',
                LanguageLinks:language?[...language.querySelectorAll('a')].filter(a=>a.getAttribute('href')).length:0};
            })()`, &state)
			for name, display := range map[string]string{
				"accessibility": state.Accessibility,
				"theme":         state.Theme,
				"contribution":  state.Contribute,
			} {
				if display != "none" {
					t.Fatalf("inert %s launcher is still presented without JavaScript: %+v", name, state)
				}
			}
			if !state.Content {
				t.Fatalf("document content is not readable without JavaScript: %+v", state)
			}
			if state.LanguageTrigger == "missing" || state.LanguageTrigger == "none" {
				t.Fatalf("native language launcher is hidden without JavaScript: %+v", state)
			}
			if state.LanguageLinks != 2 {
				t.Fatalf("native language links are not rendered: %+v", state)
			}
		})
	}

	t.Run("enhanced", func(t *testing.T) {
		b := newBrandPage(t, ctx)
		b.run(chromedp.EmulateViewport(390, 844), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))

		b.run(chromedp.Click("#mpress-theme-toggle"))
		b.poll(`document.documentElement.getAttribute('data-theme')==='dark'`, "theme toggle did not switch the document theme")

		b.run(chromedp.Click("#mpress-accessibility-toggle"))
		b.poll(`document.querySelector('#mpress-accessibility-panel').matches(':popover-open')`, "accessibility launcher did not open its panel")
		b.run(chromedp.Click(`[data-a11y-choice="text"][data-value="larger"]`))
		b.poll(`document.documentElement.dataset.a11yText==='larger'`, "accessibility preference did not change")

		b.run(chromedp.KeyEvent(kb.Escape))
		b.poll(`!document.querySelector('#mpress-accessibility-panel').matches(':popover-open')`, "accessibility panel did not close")
		b.run(chromedp.Click("[data-mpress-contribute]"))
		b.poll(`document.querySelector('#mpress-contribute-dialog').open`, "contribution launcher did not open its dialog")
		b.run(chromedp.Click("[data-contribute-close]"))
		b.poll(`!document.querySelector('#mpress-contribute-dialog').open`, "contribution dialog did not close")
	})
}

func buildReaderToolsFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	files := map[string]string{
		"mpress.yaml": `site:
  title: Reader tools
  languages: [en, fr]
build:
  contentDir: content
  staticDir: static
  outputDir: site
search:
  enabled: true
accessibility:
  enabled: true
contribution:
  enabled: true
  repository: https://github.com/leaanthony/mpress.git
  branch: main
`,
		"content/index.md":    "---\ntitle: Home\nlayout: landing\n---\n\n# Home\n\n[Guide](/guide/)\n",
		"content/guide.md":    "---\ntitle: Guide\n---\n\n# Guide\n\nReader tools stay reachable without scripts.\n",
		"content/fr/index.md": "---\ntitle: Home\n---\n\nHome.\n",
		"content/fr/guide.md": "---\ntitle: Guide\n---\n\nReader tools stay reachable without scripts.\n",
		"content/_nav.yaml":   "- label: Guide\n  link: /guide/\n",
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
