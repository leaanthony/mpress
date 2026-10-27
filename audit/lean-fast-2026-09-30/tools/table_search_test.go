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
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

const tableSearchTerm = "relay"

type tableToolbarMetrics struct {
	Title            string  `json:"title"`
	Lang             string  `json:"lang"`
	ToolbarDirection string  `json:"toolbarDirection"`
	ToolbarHeight    float64 `json:"toolbarHeight"`
	ToolbarWidth     float64 `json:"toolbarWidth"`
	SearchHeight     float64 `json:"searchHeight"`
	SearchWidth      float64 `json:"searchWidth"`
	InputHeight      float64 `json:"inputHeight"`
	SearchInside     bool    `json:"searchInside"`
	InputHit         bool    `json:"inputHit"`
	TableTop         float64 `json:"tableTop"`
	ToolbarBottom    float64 `json:"toolbarBottom"`
	DocumentWidth    float64 `json:"documentWidth"`
	ViewportWidth    float64 `json:"viewportWidth"`
}

const tableToolbarMetricsScript = `(() => {
  const root=document.querySelector('.mpress-data-table');
  const toolbar=root.querySelector('.mpress-table-toolbar');
  const search=root.querySelector('.mpress-table-search');
  const input=root.querySelector('[data-table-query]');
  const t=toolbar.getBoundingClientRect(),s=search.getBoundingClientRect(),i=input.getBoundingClientRect();
  const hit=document.elementFromPoint(i.left+i.width/2,i.top+i.height/2);
  return {
    title:document.title,
    lang:document.documentElement.lang,
    toolbarDirection:getComputedStyle(toolbar).flexDirection,
    toolbarHeight:t.height,
    toolbarWidth:t.width,
    searchHeight:s.height,
    searchWidth:s.width,
    inputHeight:i.height,
    searchInside:s.top>=t.top-.5&&s.bottom<=t.bottom+.5,
    inputHit:Boolean(hit&&(hit===input||input.contains(hit))),
    tableTop:root.querySelector('.mpress-table-scroll').getBoundingClientRect().top,
    toolbarBottom:t.bottom,
    documentWidth:document.documentElement.scrollWidth,
    viewportWidth:innerWidth
  };
})()`

type tableState struct {
	VisibleRows  int    `json:"visibleRows"`
	Status       string `json:"status"`
	PageStatus   string `json:"pageStatus"`
	FirstVisible string `json:"firstVisible"`
	Sort         string `json:"sort"`
}

const tableStateScript = `(() => {
  const root=document.querySelector('.mpress-data-table');
  const rows=[...root.querySelectorAll('[data-table-row]')];
  const visible=rows.filter(row=>!row.hidden);
  return {
    visibleRows:visible.length,
    status:root.querySelector('.mpress-table-status')?.textContent.trim()||'',
    pageStatus:root.querySelector('[data-table-page-status]')?.textContent.trim()||'',
    firstVisible:visible[0]?.textContent.replace(/\s+/g,' ').trim()||'',
    sort:root.querySelector('th[aria-sort]')?.getAttribute('aria-sort')||'none'
  };
})()`

// TestCompactMobileTableSearch serves a CLI-built fixture and checks real
// browser geometry. The phone toolbar changes to a column, so a surviving
// desktop flex basis would become empty vertical space; the desktop row must
// keep its original field-height toolbar. Controls are exercised afterwards.
func TestCompactMobileTableSearch(t *testing.T) {
	binary := os.Getenv("MPRESS_TABLE_SEARCH_BIN")
	if binary == "" {
		t.Skip("set MPRESS_TABLE_SEARCH_BIN to the candidate CLI")
	}
	site := buildTableSearchFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath("/usr/bin/google-chrome"), chromedp.NoSandbox)
	alloc, stop := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stop()
	ctx, cancel := chromedp.NewContext(alloc, chromedp.WithLogf(func(string, ...any) {}))
	defer cancel()
	ctx, timeout := context.WithTimeout(ctx, 3*time.Minute)
	defer timeout()
	cases := []struct {
		name   string
		route  string
		title  string
		lang   string
		width  int64
		height int64
		mobile bool
	}{
		{"phone-en", "/tables/", "Table toolkit", "en", 390, 844, true},
		{"phone-fr", "/fr/tables/", "Boîte à outils", "fr", 390, 844, true},
		{"desktop-en", "/tables/", "Table toolkit", "en", 1280, 900, false},
		{"desktop-fr", "/fr/tables/", "Boîte à outils", "fr", 1280, 900, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			page, closePage := chromedp.NewContext(ctx)
			defer closePage()
			b := browserActions{t: t, ctx: page}
			b.run(chromedp.EmulateViewport(testCase.width, testCase.height), emulation.SetTouchEmulationEnabled(true), chromedp.Navigate(server.URL+testCase.route), chromedp.WaitReady(".mpress-data-table"))
			var metrics tableToolbarMetrics
			b.js(tableToolbarMetricsScript, &metrics)
			if !strings.Contains(metrics.Title, testCase.title) || metrics.Lang != testCase.lang {
				t.Fatalf("wrong localized fixture: title=%q lang=%q", metrics.Title, metrics.Lang)
			}
			checkTableToolbarLayout(t, metrics, testCase.mobile)
			captureTableSearch(t, b, testCase.name)
			exerciseTableSearchControls(t, b, testCase.lang)
		})
	}
}

func checkTableToolbarLayout(t *testing.T, metrics tableToolbarMetrics, mobile bool) {
	t.Helper()
	if !metrics.SearchInside || !metrics.InputHit {
		t.Fatalf("table search must sit inside the toolbar and remain hit-testable: %+v", metrics)
	}
	if metrics.SearchHeight > metrics.InputHeight+8 {
		t.Fatalf("search wrapper reserves %.2fpx around a %.2fpx field: %+v", metrics.SearchHeight, metrics.InputHeight, metrics)
	}
	if metrics.TableTop-metrics.ToolbarBottom > 2 {
		t.Fatalf("table data starts %.2fpx below the toolbar: %+v", metrics.TableTop-metrics.ToolbarBottom, metrics)
	}
	if metrics.DocumentWidth > metrics.ViewportWidth+1 {
		t.Fatalf("page overflows horizontally: %.2f > %.2f", metrics.DocumentWidth, metrics.ViewportWidth)
	}
	if mobile {
		if metrics.ToolbarDirection != "column" {
			t.Fatalf("phone toolbar must keep its column layout, got %q", metrics.ToolbarDirection)
		}
		if metrics.ToolbarHeight >= 90 {
			t.Fatalf("phone toolbar reserves %.2fpx instead of tracking the search field", metrics.ToolbarHeight)
		}
		if metrics.ToolbarHeight-metrics.InputHeight > 28 {
			t.Fatalf("phone toolbar has %.2fpx of space beyond its search field", metrics.ToolbarHeight-metrics.InputHeight)
		}
		if metrics.SearchWidth < metrics.ToolbarWidth*0.8 {
			t.Fatalf("phone search field is squeezed to %.2fpx in a %.2fpx toolbar", metrics.SearchWidth, metrics.ToolbarWidth)
		}
		t.Logf("phone toolbar %.2fx%.2fpx, search %.2fx%.2fpx", metrics.ToolbarWidth, metrics.ToolbarHeight, metrics.SearchWidth, metrics.SearchHeight)
		return
	}
	if metrics.ToolbarDirection != "row" {
		t.Fatalf("desktop toolbar layout changed to %q", metrics.ToolbarDirection)
	}
	if metrics.ToolbarHeight < 50 || metrics.ToolbarHeight > 70 {
		t.Fatalf("desktop toolbar height changed to %.2fpx", metrics.ToolbarHeight)
	}
	if metrics.SearchWidth < metrics.ToolbarWidth*0.8 {
		t.Fatalf("desktop search field is squeezed to %.2fpx in a %.2fpx toolbar", metrics.SearchWidth, metrics.ToolbarWidth)
	}
	t.Logf("desktop toolbar %.2fx%.2fpx, search %.2fx%.2fpx", metrics.ToolbarWidth, metrics.ToolbarHeight, metrics.SearchWidth, metrics.SearchHeight)
}

func exerciseTableSearchControls(t *testing.T, b browserActions, lang string) {
	t.Helper()
	separator := "of"
	if lang == "fr" {
		separator = "sur"
	}
	pageStatus := func(page, total int) string {
		return fmt.Sprintf("Page %d %s %d", page, separator, total)
	}
	b.run(chromedp.Click(`[data-table-query]`, chromedp.ByQuery), chromedp.SendKeys(`[data-table-query]`, tableSearchTerm, chromedp.ByQuery))
	var focused bool
	b.js(`document.activeElement===document.querySelector('[data-table-query]')`, &focused)
	if !focused {
		t.Fatal("keyboard input did not focus the table search field")
	}
	state := b.waitTableState("one search result", func(state tableState) bool {
		return state.VisibleRows == 1 && strings.Contains(strings.ToLower(state.FirstVisible), tableSearchTerm)
	})
	if !strings.Contains(state.Status, "1") {
		t.Fatalf("search status does not report the live result count: %q", state.Status)
	}

	b.run(chromedp.KeyEvent("a", chromedp.KeyModifiers(input.ModifierCtrl)), chromedp.KeyEvent(kb.Backspace))
	state = b.waitTableState("first page restored", func(state tableState) bool {
		return state.VisibleRows == 2 && strings.Contains(state.PageStatus, pageStatus(1, 3))
	})

	b.run(chromedp.Click(`[data-table-page-next]`, chromedp.ByQuery))
	state = b.waitTableState("second page", func(state tableState) bool {
		return state.VisibleRows == 2 && strings.Contains(state.PageStatus, pageStatus(2, 3)) && strings.Contains(state.FirstVisible, "Windows")
	})

	b.run(chromedp.Click(`[data-table-sort-column="0"]`, chromedp.ByQuery))
	b.waitTableState("ascending sort", func(state tableState) bool {
		return state.Sort == "ascending" && strings.Contains(state.PageStatus, pageStatus(1, 3))
	})
	b.run(chromedp.Click(`[data-table-sort-column="0"]`, chromedp.ByQuery))
	state = b.waitTableState("descending sort", func(state tableState) bool {
		return state.Sort == "descending" && strings.Contains(state.FirstVisible, "Studio")
	})

	b.tap(`.mpress-table-filter-trigger`)
	b.waitPopoverOpen(`[data-table-filter-column="0"]`)
	b.run(chromedp.Click(`[data-table-filter-column="0"] button[data-table-filter-value="studio"]`, chromedp.ByQuery))
	state = b.waitTableState("filtered row", func(state tableState) bool {
		return state.VisibleRows == 1 && strings.Contains(state.FirstVisible, "Studio")
	})
	if !strings.Contains(state.PageStatus, pageStatus(1, 1)) {
		t.Fatalf("filter did not reset pagination: %q", state.PageStatus)
	}
	b.tap(`.mpress-table-filter-trigger`)
	b.waitPopoverOpen(`[data-table-filter-column="0"]`)
	b.run(chromedp.Click(`[data-table-filter-column="0"] button[data-table-filter-value=""]`, chromedp.ByQuery))
	b.waitTableState("filter cleared", func(state tableState) bool {
		return state.VisibleRows == 2 && strings.Contains(state.FirstVisible, "Studio")
	})
}

func (b *browserActions) tap(selector string) {
	b.t.Helper()
	var point struct{ X, Y float64 }
	b.js(fmt.Sprintf(`(() => {const e=document.querySelector(%s);e.scrollIntoView({block:'center',behavior:'instant'});const r=e.getBoundingClientRect();return {X:r.left+r.width/2,Y:r.top+r.height/2}})()`, browserQuote(selector)), &point)
	b.run(browserTap(point.X, point.Y))
}

func (b *browserActions) waitPopoverOpen(selector string) {
	b.t.Helper()
	predicate := fmt.Sprintf(`(() => {const p=document.querySelector(%s);return Boolean(p&&p.matches(':popover-open')&&!p.getAnimations().some(a=>a.playState==='running'))})()`, browserQuote(selector))
	if err := chromedp.Run(b.ctx, chromedp.Poll(predicate, nil, chromedp.WithPollingInterval(16*time.Millisecond), chromedp.WithPollingTimeout(2*time.Second))); err != nil {
		b.t.Fatalf("popover %s did not open: %v", selector, err)
	}
}

func (b *browserActions) waitTableState(goal string, match func(tableState) bool) tableState {
	b.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var state tableState
		b.js(tableStateScript, &state)
		if match(state) {
			return state
		}
		if time.Now().After(deadline) {
			b.t.Fatalf("table did not reach %s: %+v", goal, state)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func captureTableSearch(t *testing.T, b browserActions, name string) {
	t.Helper()
	dir := os.Getenv("MPRESS_TABLE_SEARCH_SCREENSHOTS")
	if dir == "" {
		return
	}
	var data []byte
	b.run(chromedp.CaptureScreenshot(&data))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".png"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

const tableSearchFixturePage = `---
title: %s
description: Searchable table fixture.
---

## Searchable packages

@table{header search filter sort paginate column-separators page-size=2}
| Package | Platform | Downloads |
| Core | Linux | 1840 |
| Studio | macOS | 920 |
| Core | Windows | 1260 |
| Edge | Linux | 640 |
| Relay | macOS | 310 |
@end
`

func buildTableSearchFixture(t *testing.T, binary string) string {
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
	write("content/_nav.yaml", "- label: Home\n  link: /\n- label: Tables\n  link: /tables/\n")
	write("content/index.md", "---\ntitle: Home\n---\n\nTable fixture home.\n")
	write("content/tables.md", fmt.Sprintf(tableSearchFixturePage, "Table toolkit"))
	write("content/fr/_nav.yaml", "- label: Accueil\n  link: /\n- label: Tableaux\n  link: /tables/\n")
	write("content/fr/index.md", "---\ntitle: Accueil\n---\n\nPage d'accueil.\n")
	write("content/fr/tables.md", fmt.Sprintf(tableSearchFixturePage, "Boîte à outils"))
	run("build", "--strict")
	return filepath.Join(project, "site")
}
