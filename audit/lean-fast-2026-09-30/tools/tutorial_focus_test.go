package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

type tutorialFocusState struct {
	Missing      bool    `json:"missing"`
	Count        string  `json:"count"`
	Checked      int     `json:"checked"`
	Tutorials    int     `json:"tutorials"`
	Focused      bool    `json:"focused"`
	FocusVisible bool    `json:"focusVisible"`
	InputOpacity string  `json:"inputOpacity"`
	OutlineStyle string  `json:"outlineStyle"`
	OutlineWidth float64 `json:"outlineWidth"`
	OutlineColor string  `json:"outlineColor"`
}

// TestTutorialCheckboxShowsKeyboardFocus checks the visible checkmark, not the
// opacity-0 native checkbox, receives the theme focus ring when a keyboard
// reader tabs to a step. Space still records progress, the record survives a
// reload, a second tutorial stays independent, and Enter resets progress.
func TestTutorialCheckboxShowsKeyboardFocus(t *testing.T) {
	binary := os.Getenv("MPRESS_TUTORIAL_BIN")
	if binary == "" {
		t.Skip("set MPRESS_TUTORIAL_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()
	site := buildTutorialFocusFixture(t, binary)
	server := httptest.NewServer(http.FileServer(http.Dir(site)))
	defer server.Close()

	for _, viewport := range []struct {
		name   string
		width  int64
		height int64
	}{
		{"phone", 320, 720},
		{"desktop", 1280, 900},
	} {
		t.Run(viewport.name, func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(viewport.width, viewport.height), chromedp.Navigate(server.URL+"/tutorial/"), chromedp.WaitReady("body"))

			first := b.expectTutorialFocusState(0, "0/2 completed", 0)
			b.expectTutorialFocusState(1, "0/2 completed", 0)
			if first.Tutorials != 2 {
				t.Fatalf("fixture should render two independent tutorials, got %d", first.Tutorials)
			}

			b.focusTutorialCheckboxByKeyboard()
			focused := b.expectTutorialFocusState(0, "0/2 completed", 0)
			if !focused.Focused || !focused.FocusVisible {
				t.Fatalf("keyboard focus did not reach the first tutorial checkbox: %+v", focused)
			}
			if focused.InputOpacity != "0" {
				t.Fatalf("native checkbox should stay invisible, got opacity %q", focused.InputOpacity)
			}
			if focused.OutlineStyle != "solid" || focused.OutlineWidth < 2 || focused.OutlineColor == "rgba(0, 0, 0, 0)" {
				t.Fatalf("visible checkmark has no focus ring: %+v", focused)
			}

			b.run(chromedp.KeyEvent(" "))
			b.pollTutorialState(0, "1/2 completed", 1)
			b.expectTutorialFocusState(1, "0/2 completed", 0)

			b.run(chromedp.Navigate(server.URL+"/tutorial/"), chromedp.WaitReady("body"))
			b.pollTutorialState(0, "1/2 completed", 1)
			b.expectTutorialFocusState(1, "0/2 completed", 0)

			b.focusTutorialReset()
			b.run(chromedp.KeyEvent(kb.Enter))
			b.pollTutorialState(0, "0/2 completed", 0)
			b.expectTutorialFocusState(1, "0/2 completed", 0)
		})
	}
}

func tutorialFocusScript(index int) string {
	return fmt.Sprintf(`(() => {
      const tutorials = [...document.querySelectorAll('.mpress-tutorial')];
      const tutorial = tutorials[%d];
      if (!tutorial) return {missing: true};
      const box = tutorial.querySelector('input[type="checkbox"]');
      const mark = box && box.nextElementSibling;
      const style = mark && getComputedStyle(mark);
      return {
        count: tutorial.querySelector('.mpress-tutorial-count')?.textContent || '',
        checked: [...tutorial.querySelectorAll('input[type="checkbox"]')].filter(input => input.checked).length,
        tutorials: tutorials.length,
        focused: document.activeElement === box,
        focusVisible: Boolean(box && box.matches(':focus-visible')),
        inputOpacity: box ? getComputedStyle(box).opacity : '',
        outlineStyle: style ? style.outlineStyle : '',
        outlineWidth: style ? parseFloat(style.outlineWidth) : 0,
        outlineColor: style ? style.outlineColor : ''
      };
    })()`, index)
}

func (b *browserActions) tutorialState(index int) tutorialFocusState {
	b.t.Helper()
	var state tutorialFocusState
	b.js(tutorialFocusScript(index), &state)
	if state.Missing {
		b.t.Fatalf("tutorial %d is missing", index)
	}
	return state
}

func (b *browserActions) expectTutorialFocusState(index int, count string, checked int) tutorialFocusState {
	b.t.Helper()
	state := b.tutorialState(index)
	if state.Count != count || state.Checked != checked {
		b.t.Fatalf("tutorial %d has %q with %d checked, want %q with %d: %+v", index, state.Count, state.Checked, count, checked, state)
	}
	return state
}

func (b *browserActions) pollTutorialState(index int, count string, checked int) {
	b.t.Helper()
	predicate := fmt.Sprintf(`(() => {
      const tutorial = document.querySelectorAll('.mpress-tutorial')[%d];
      if (!tutorial) return false;
      const inputs = [...tutorial.querySelectorAll('input[type="checkbox"]')];
      return tutorial.querySelector('.mpress-tutorial-count')?.textContent === %s && inputs.filter(input => input.checked).length === %d;
    })()`, index, browserQuote(count), checked)
	b.poll(predicate, "tutorial progress did not update")
}

// focusTutorialCheckboxByKeyboard leaves the first checkbox focused after real
// Tab and Shift+Tab key presses so :focus-visible applies as it does for a
// keyboard reader.
func (b *browserActions) focusTutorialCheckboxByKeyboard() {
	b.t.Helper()
	var found bool
	b.js(`(() => {
      const box = document.querySelector('.mpress-tutorial input[type="checkbox"]');
      if (!box) return false;
      box.focus({preventScroll: true});
      return document.activeElement === box;
    })()`, &found)
	if !found {
		b.t.Fatal("first tutorial checkbox is not focusable")
	}
	b.run(chromedp.KeyEvent(kb.Tab), chromedp.KeyEvent(kb.Tab, chromedp.KeyModifiers(input.ModifierShift)))
}

func (b *browserActions) focusTutorialReset() {
	b.t.Helper()
	var found bool
	b.js(`(() => {
      const reset = document.querySelector('.mpress-tutorial .mpress-tutorial-reset');
      if (!reset) return false;
      reset.focus({preventScroll: true});
      return document.activeElement === reset;
    })()`, &found)
	if !found {
		b.t.Fatal("first tutorial reset button is not focusable")
	}
}

func buildTutorialFocusFixture(t *testing.T, binary string) string {
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
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("content/tutorial.md", `---
title: Tutorial focus
---

@tutorial{title="Focus first"}
### First step
Read the first step.

### Second step
Read the second step.
@end

@tutorial{title="Focus second"}
### Alpha
Read the alpha step.

### Beta
Read the beta step.
@end
`)
	run("build", "--strict")
	return filepath.Join(project, "site")
}
