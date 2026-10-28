package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type calendarState struct {
	Title string
	Days  int
}

const calendarStateProbe = `[...document.querySelectorAll('.mpress-calendar')].map(c => ({
  Title:c.querySelector('.mpress-calendar-title').textContent,
  Days:c.querySelectorAll('.mpress-calendar-cell[data-date]').length
}))`

// TestCalendarWithoutJavaScript checks that a blocked-JavaScript reader sees
// the authored month grids, titles and content without inert month controls,
// while enhanced calendars still navigate by native keyboard per instance.
func TestCalendarWithoutJavaScript(t *testing.T) {
	binary := os.Getenv("MPRESS_CALENDAR_BIN")
	if binary == "" {
		t.Skip("set MPRESS_CALENDAR_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildCalendarFixture(t, binary))))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {1280, 900}} {
		t.Run(fmt.Sprintf("blocked-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(size[0] < 600))
			loadWithoutJavaScript(&b, server.URL+"/guide/")
			b.poll(`!document.documentElement.classList.contains('js')`, "blocked JavaScript still marked the document as enhanced")

			var state []struct {
				Title, Grid, Content string
				Nav                  []string
				Cells                []struct {
					Text, Display string
					Width, Height float64
				}
			}
			b.js(`[...document.querySelectorAll('.mpress-calendar')].map(c => ({
              Title:c.querySelector('.mpress-calendar-title').textContent,
              Grid:getComputedStyle(c.querySelector('.mpress-calendar-grid')).display,
              Content:(c.querySelector('.mpress-calendar-content')||{}).textContent,
              Nav:[...c.querySelectorAll('.mpress-calendar-nav')].map(b => getComputedStyle(b).display),
              Cells:[...c.querySelectorAll('.mpress-calendar-cell[data-date]')].map(cell => {
                const r=cell.getBoundingClientRect();
                return {Text:cell.textContent.trim(), Display:getComputedStyle(cell).display,
                  Width:r.width, Height:r.height};
              })
            }))`, &state)
			if len(state) != 2 {
				t.Fatalf("fixture has %d calendars, want 2", len(state))
			}
			for index, calendar := range state {
				if calendar.Title == "" || calendar.Grid == "none" || calendar.Content == "" {
					t.Fatalf("calendar %d is not readable without JavaScript: %+v", index, calendar)
				}
				if len(calendar.Nav) != 2 || calendar.Nav[0] != "none" || calendar.Nav[1] != "none" {
					t.Fatalf("calendar %d shows inert month navigation without JavaScript: %v", index, calendar.Nav)
				}
				if want := []int{31, 30}[index]; len(calendar.Cells) != want {
					t.Fatalf("calendar %d has %d dated cells, want %d", index, len(calendar.Cells), want)
				}
				for _, cell := range calendar.Cells {
					if cell.Text == "" || cell.Display == "none" || cell.Width <= 0 || cell.Height <= 0 {
						t.Fatalf("calendar %d has an unreadable date cell without JavaScript: %+v", index, cell)
					}
				}
			}
		})

		t.Run(fmt.Sprintf("enhanced-%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), emulation.SetTouchEmulationEnabled(size[0] < 600), chromedp.Navigate(server.URL+"/guide/"), chromedp.WaitReady("body"))
			expectCalendarState(t, &b, []calendarState{{"December 2026", 31}, {"June 2025", 30}}, "enhanced calendars did not start on their authored months")

			b.run(chromedp.Focus(`.mpress-calendar [data-dir="next"]`), chromedp.KeyEvent(" "))
			expectCalendarState(t, &b, []calendarState{{"January 2027", 31}, {"June 2025", 30}}, "Space on next did not advance only its own calendar")
			b.poll(`document.activeElement === document.querySelector('.mpress-calendar [data-dir="next"]')`, "next lost focus after Space")

			b.run(chromedp.Focus(`.mpress-calendar [data-dir="prev"]`), chromedp.KeyEvent(kb.Enter))
			expectCalendarState(t, &b, []calendarState{{"December 2026", 31}, {"June 2025", 30}}, "Enter on previous did not return its own calendar")
			b.poll(`document.activeElement === document.querySelector('.mpress-calendar [data-dir="prev"]')`, "previous lost focus after Enter")

			b.run(chromedp.KeyEvent(" "))
			expectCalendarState(t, &b, []calendarState{{"November 2026", 30}, {"June 2025", 30}}, "Space on previous did not rewind only its own calendar")
			b.poll(`document.activeElement === document.querySelector('.mpress-calendar [data-dir="prev"]')`, "previous lost focus after Space")
		})
	}
}

func expectCalendarState(t *testing.T, b *browserActions, want []calendarState, message string) {
	t.Helper()
	var got []calendarState
	b.js(calendarStateProbe, &got)
	if !slices.Equal(got, want) {
		t.Fatalf("%s: got %+v, want %+v", message, got, want)
	}
}

func buildCalendarFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	initialise := exec.Command(binary, "init", ".")
	initialise.Dir = project
	if output, err := initialise.CombinedOutput(); err != nil {
		t.Fatalf("initialise calendar fixture: %v\n%s", err, output)
	}
	if err := os.WriteFile(filepath.Join(project, "content", "guide.md"), []byte(`---
title: Guide
---

:::calendar{month="2026-12"}
**December planning.** Registrations are open.
:::

:::calendar{month="2025-06"}
**June archive.** Past sessions are listed.
:::
`), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "build", "--strict")
	command.Dir = project
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build calendar fixture: %v\n%s", err, output)
	}
	return filepath.Join(project, "site")
}
