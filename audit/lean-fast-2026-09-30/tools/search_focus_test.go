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

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const searchProbeTerm = "quasarwidget"

type searchCloseCase struct {
	name   string
	route  string
	title  string
	width  int64
	height int64
	mobile bool
}

// Closing the overlay must return focus to the launcher without changing the
// phone header layout. Both keyboard and pointer dismissal are repeated so a
// stale focus or one-off class cannot pass.
func TestSearchCloseRestoresHeaderControls(t *testing.T) {
	binary := os.Getenv("MPRESS_SEARCH_BIN")
	if binary == "" {
		t.Skip("set MPRESS_SEARCH_BIN to the candidate CLI")
	}
	site := buildSearchCloseFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 4*time.Minute)
	defer timeout()
	cases := []searchCloseCase{
		{"mobile-en", "/guide/", "Search guide", 375, 667, true},
		{"mobile-fr", "/fr/guide/", "Guide de recherche", 375, 667, true},
		{"desktop-en", "/guide/", "Search guide", 1280, 900, false},
		{"desktop-fr", "/fr/guide/", "Guide de recherche", 1280, 900, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			page, closePage := chromedp.NewContext(ctx)
			defer closePage()
			b := browserActions{t: t, ctx: page}
			b.run(chromedp.EmulateViewport(testCase.width, testCase.height), chromedp.Navigate(server.URL+testCase.route), chromedp.WaitReady("body"))
			b.expectSearchHeader(testCase.mobile, false)
			for repeat := 0; repeat < 2; repeat++ {
				b.run(chromedp.KeyEvent("k", chromedp.KeyModifiers(input.ModifierCtrl)))
				b.waitSearchOpen()
				b.checkSearchProbe(testCase.title)
				b.run(chromedp.KeyEvent(kb.Escape))
				b.expectSearchHeader(testCase.mobile, true)
				b.run(chromedp.KeyEvent(kb.Escape), chromedp.KeyEvent(kb.Escape))
				b.expectSearchHeader(testCase.mobile, true)

				b.run(chromedp.Click("#mpress-search-toggle", chromedp.ByID))
				b.waitSearchOpen()
				b.checkSearchProbe(testCase.title)
				b.run(chromedp.Click(".mpress-search-close", chromedp.ByQuery))
				b.expectSearchHeader(testCase.mobile, true)
			}
		})
	}
}

func (b *browserActions) waitSearchOpen() {
	b.t.Helper()
	const predicate = `(() => {const overlay=document.querySelector('.mpress-search-overlay');return Boolean(overlay&&overlay.classList.contains('open')&&document.activeElement===document.querySelector('.mpress-search-query input'))})()`
	if err := chromedp.Run(b.ctx, chromedp.Poll(predicate, nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(2*time.Second))); err != nil {
		b.t.Fatalf("search overlay did not open with its query focused: %s (%v)", b.searchState(), err)
	}
}

func (b *browserActions) checkSearchProbe(title string) {
	b.t.Helper()
	b.run(chromedp.KeyEvent(searchProbeTerm))
	predicate := fmt.Sprintf(`(() => [...document.querySelectorAll('.mpress-search-item-title')].some(item => item.textContent.includes(%q)))()`, title)
	if err := chromedp.Run(b.ctx, chromedp.Poll(predicate, nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(3*time.Second))); err != nil {
		b.t.Fatalf("search did not render the %q result: %s (%v)", title, b.searchState(), err)
	}
}

// expectSearchHeader waits until the overlay is closed, focus is where the
// caller expects it, and the theme, accessibility and language controls can be
// hit-tested. On phones the launcher must keep its compact 44px slot.
func (b *browserActions) expectSearchHeader(mobile, wantFocus bool) {
	b.t.Helper()
	maxWidth := 400
	if mobile {
		maxWidth = 48
	}
	focusCheck := "document.activeElement===q"
	if !wantFocus {
		focusCheck = "document.activeElement!==q"
	}
	predicate := fmt.Sprintf(`(() => {
      const q=document.querySelector('#mpress-search-toggle');
      const overlay=document.querySelector('.mpress-search-overlay');
      if(!q||(overlay&&overlay.classList.contains('open')))return false;
      const usable=e=>{if(!e)return false;const s=getComputedStyle(e),r=e.getBoundingClientRect();if(s.display==='none'||s.visibility!=='visible'||r.width<=0||r.height<=0||r.left<0||r.right>innerWidth||r.top<0||r.bottom>innerHeight)return false;const hit=document.elementFromPoint(r.left+r.width/2,r.top+r.height/2);return Boolean(hit&&(hit===e||e.contains(hit)))};
      const theme=document.querySelector('#mpress-theme-toggle');
      const accessibility=document.querySelector('#mpress-accessibility-toggle');
      const language=document.querySelector('.language-select .utility-menu-trigger');
      return %s&&usable(q)&&q.getBoundingClientRect().width<=%d&&usable(theme)&&usable(accessibility)&&usable(language);
    })()`, focusCheck, maxWidth)
	if err := chromedp.Run(b.ctx, chromedp.Poll(predicate, nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(2*time.Second))); err != nil {
		b.t.Fatalf("search close did not restore a compact header with usable controls: %s (%v)", b.searchState(), err)
	}
}

func (b *browserActions) searchState() string {
	var state string
	b.js(`(() => {
      const overlay=document.querySelector('.mpress-search-overlay');
      const q=document.querySelector('#mpress-search-toggle');
      const links=document.querySelector('.header-links');
      return JSON.stringify({
        focus: document.activeElement?.outerHTML?.slice(0,180),
        overlayOpen: Boolean(overlay&&overlay.classList.contains('open')),
        launcherWidth: q?.getBoundingClientRect().width,
        linksVisibility: links?getComputedStyle(links).visibility:null,
        results: document.querySelectorAll('.mpress-search-item').length
      });
    })()`, &state)
	return state
}

func buildSearchCloseFixture(t *testing.T, binary string) string {
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
	path := filepath.Join(project, "mpress.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := strings.Replace(string(data), "  languages: [en]", "  languages: [en, fr]", 1)
	cfg = strings.Replace(cfg, "    en: English", "    en: English\n    fr: Français", 1)
	if !strings.Contains(cfg, "languages: [en, fr]") || !strings.Contains(cfg, "fr: Français") {
		t.Fatal("fixture did not add the French language")
	}
	content := filepath.Join(project, "content")
	if err := os.RemoveAll(content); err != nil {
		t.Fatal(err)
	}
	write := func(name, value string) {
		t.Helper()
		target := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mpress.yaml", cfg)
	write("content/_nav.yaml", "- label: Home\n  link: /\n- label: Guide\n  link: /guide/\n")
	write("content/index.md", "---\ntitle: Home\n---\n\nSearch fixture home.\n")
	write("content/guide.md", "---\ntitle: Search guide\ndescription: Find the quasarwidget reference.\n---\n\n## Quasarwidget details\n\nThe quasarwidget appears in this guide.\n")
	write("content/fr/_nav.yaml", "- label: Accueil\n  link: /\n- label: Guide\n  link: /guide/\n")
	write("content/fr/index.md", "---\ntitle: Accueil\n---\n\nPage d'accueil.\n")
	write("content/fr/guide.md", "---\ntitle: Guide de recherche\ndescription: Trouvez la référence quasarwidget.\n---\n\n## Détails quasarwidget\n\nLe quasarwidget apparaît dans ce guide.\n")
	run("build", "--strict")
	return filepath.Join(project, "site")
}
