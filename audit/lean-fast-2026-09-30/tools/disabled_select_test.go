package main

import (
	"encoding/json"
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

type selectState struct {
	Value          string `json:"value"`
	Index          int    `json:"index"`
	Expanded       string `json:"expanded"`
	Open           bool   `json:"open"`
	Focus          string `json:"focus"`
	TriggerFocused bool   `json:"triggerFocused"`
}

// TestCustomSelectDisabledOptions checks a strict production build of a raw
// <select> whose disabled states come from both an optgroup and an explicit
// option attribute. The enhanced listbox must follow the browser's effective
// disabled semantics (:disabled), skip both kinds with the keyboard, refuse
// pointer activation, choose the next enabled option and leave an independent
// select alone.
func TestCustomSelectDisabledOptions(t *testing.T) {
	binary := os.Getenv("MPRESS_SELECT_BIN")
	if binary == "" {
		t.Skip("set MPRESS_SELECT_BIN to the candidate CLI")
	}
	ctx, stop := newHeaderBrowser(t)
	defer stop()

	server := httptest.NewServer(http.FileServer(http.Dir(buildSelectFixture(t, binary))))
	defer server.Close()

	for _, size := range [][2]int64{{320, 667}, {1280, 900}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			b := newBrandPage(t, ctx)
			b.run(chromedp.EmulateViewport(size[0], size[1]), chromedp.Navigate(server.URL+"/"), chromedp.WaitReady("body"))
			b.poll(`document.querySelectorAll('.mpress-ddlb').length===2`, "custom selects were not installed")
			b.poll(`document.querySelectorAll('.mpress-ddlb-option').length===7`, "option lists were not rendered")

			var raw string
			b.js(`(() => {
              const select=document.querySelector('#release');
              return JSON.stringify({
                internalDisabled:select.options[1].disabled,
                internalEffective:select.options[1].matches(':disabled'),
                legacyEffective:select.options[2].matches(':disabled'),
                items:[...document.querySelectorAll('.mpress-ddlb')[0].querySelectorAll('.mpress-ddlb-option')].map(option=>({text:option.textContent.trim(),disabled:option.disabled,role:option.getAttribute('role'),selected:option.getAttribute('aria-selected')})),
                channel:(()=>{const w=document.querySelectorAll('.mpress-ddlb')[1];const s=w.querySelector('select');return {value:s.value,gammaDisabled:w.querySelectorAll('.mpress-ddlb-option')[2].disabled};})(),
                aria:(()=>{const w=document.querySelectorAll('.mpress-ddlb')[0],trigger=w.querySelector('.mpress-ddlb-trigger'),menu=w.querySelector('.mpress-ddlb-menu');return {role:trigger.getAttribute('role'),haspopup:trigger.getAttribute('aria-haspopup'),controlsMatches:trigger.getAttribute('aria-controls')===menu.id,menuRole:menu.getAttribute('role'),expanded:trigger.getAttribute('aria-expanded')};})()
              });
            })()`, &raw)
			var before struct {
				InternalDisabled  bool `json:"internalDisabled"`
				InternalEffective bool `json:"internalEffective"`
				LegacyEffective   bool `json:"legacyEffective"`
				Items             []struct {
					Text     string `json:"text"`
					Role     string `json:"role"`
					Selected string `json:"selected"`
					Disabled bool   `json:"disabled"`
				} `json:"items"`
				Channel struct {
					Value         string `json:"value"`
					GammaDisabled bool   `json:"gammaDisabled"`
				} `json:"channel"`
				Aria struct {
					Role            string `json:"role"`
					Haspopup        string `json:"haspopup"`
					MenuRole        string `json:"menuRole"`
					Expanded        string `json:"expanded"`
					ControlsMatches bool   `json:"controlsMatches"`
				} `json:"aria"`
			}
			if err := json.Unmarshal([]byte(raw), &before); err != nil {
				t.Fatalf("read select state: %v (%q)", err, raw)
			}
			if before.InternalDisabled || !before.InternalEffective || !before.LegacyEffective {
				t.Fatalf("native disabled semantics changed: %s", raw)
			}
			for i, want := range []struct {
				text     string
				disabled bool
			}{{"Stable", false}, {"Internal", true}, {"Disabled", true}, {"Next", false}} {
				item := before.Items[i]
				if item.Text != want.text || item.Disabled != want.disabled || item.Role != "option" {
					t.Fatalf("option %d rendered as %+v, want %q disabled=%v", i, item, want.text, want.disabled)
				}
			}
			if before.Items[0].Selected != "true" {
				t.Fatalf("selected option is not marked: %s", raw)
			}
			if before.Channel.Value != "alpha" || !before.Channel.GammaDisabled {
				t.Fatalf("independent select lost its own disabled semantics: %s", raw)
			}
			if before.Aria.Role != "combobox" || before.Aria.Haspopup != "listbox" || before.Aria.MenuRole != "listbox" || !before.Aria.ControlsMatches || before.Aria.Expanded != "false" {
				t.Fatalf("combobox ARIA is wrong: %s", raw)
			}

			events := func() []string {
				var raw string
				b.js(`JSON.stringify(window.__selectEvents||[])`, &raw)
				var got []string
				if err := json.Unmarshal([]byte(raw), &got); err != nil {
					t.Fatalf("read select events: %v (%q)", err, raw)
				}
				return got
			}
			state := func() selectState {
				var raw string
				b.js(`(() => {const w=document.querySelectorAll('.mpress-ddlb')[0],s=w.querySelector('select');return JSON.stringify({value:s.value,index:s.selectedIndex,expanded:w.querySelector('.mpress-ddlb-trigger').getAttribute('aria-expanded'),open:w.querySelector('.mpress-ddlb-menu').matches(':popover-open'),focus:(document.activeElement?.textContent||'').trim(),triggerFocused:document.activeElement===w.querySelector('.mpress-ddlb-trigger')});})()`, &raw)
				var st selectState
				if err := json.Unmarshal([]byte(raw), &st); err != nil {
					t.Fatalf("read listbox state: %v (%q)", err, raw)
				}
				return st
			}
			click := func(index int) {
				var raw string
				b.js(fmt.Sprintf(`(() => {const option=[...document.querySelectorAll('.mpress-ddlb')[0].querySelectorAll('.mpress-ddlb-option')][%d];option.scrollIntoView({block:'nearest'});const r=option.getBoundingClientRect();return JSON.stringify({x:r.left+r.width/2,y:r.top+r.height/2});})()`, index), &raw)
				var point struct{ X, Y float64 }
				if err := json.Unmarshal([]byte(raw), &point); err != nil {
					t.Fatalf("locate option %d: %v (%q)", index, err, raw)
				}
				b.run(chromedp.MouseClickXY(point.X, point.Y))
			}
			b.js(`window.__selectEvents=[];document.querySelectorAll('select').forEach(s=>['input','change'].forEach(type=>s.addEventListener(type,()=>window.__selectEvents.push(s.id+':'+type))));document.querySelectorAll('.mpress-ddlb-trigger')[0].focus();true`, nil)

			// Space opens; ArrowDown skips the group-disabled and the explicitly
			// disabled option; Enter chooses Next and fires each event once.
			b.run(chromedp.KeyEvent(" "))
			b.poll(`document.querySelectorAll('.mpress-ddlb-menu')[0].matches(':popover-open')`, "Space did not open the listbox")
			if st := state(); st.Focus != "Stable" || st.Expanded != "true" {
				t.Fatalf("opening did not focus the selected option: %+v", st)
			}
			b.run(chromedp.KeyEvent(kb.ArrowDown))
			if st := state(); st.Focus != "Next" {
				t.Fatalf("ArrowDown did not skip disabled options: %+v", st)
			}
			b.run(selectEnter())
			b.poll(`document.querySelectorAll('.mpress-ddlb')[0].querySelector('select').value==='next'`, "Enter did not choose Next")
			if st := state(); st.Open || st.Expanded != "false" || !st.TriggerFocused {
				t.Fatalf("choosing did not close the listbox and restore trigger focus: %+v", st)
			}
			if got := events(); len(got) != 2 || got[0] != "release:input" || got[1] != "release:change" {
				t.Fatalf("choice fired the wrong events: %v", got)
			}

			// Home/End traverse only enabled options; Escape closes without a change.
			b.run(selectEnter())
			b.poll(`document.querySelectorAll('.mpress-ddlb-menu')[0].matches(':popover-open')`, "Enter did not reopen the listbox")
			b.run(chromedp.KeyEvent(kb.Home))
			if st := state(); st.Focus != "Stable" {
				t.Fatalf("Home did not reach the first enabled option: %+v", st)
			}
			b.run(chromedp.KeyEvent(kb.End))
			if st := state(); st.Focus != "Next" {
				t.Fatalf("End did not reach the last enabled option: %+v", st)
			}
			b.run(chromedp.KeyEvent(kb.Escape))
			b.poll(`!document.querySelectorAll('.mpress-ddlb-menu')[0].matches(':popover-open')`, "Escape did not close the listbox")
			if st := state(); st.Value != "next" || !st.TriggerFocused {
				t.Fatalf("Escape changed the choice or lost focus: %+v", st)
			}
			if got := events(); len(got) != 2 {
				t.Fatalf("keyboard navigation fired extra events: %v", got)
			}

			// A pointer cannot activate either disabled option, but an enabled
			// option still chooses and reports exactly one input/change pair.
			b.run(chromedp.Click(`.mpress-ddlb-trigger`))
			b.poll(`document.querySelectorAll('.mpress-ddlb-menu')[0].matches(':popover-open')`, "pointer did not open the listbox")
			click(1)
			click(2)
			if st := state(); st.Value != "next" {
				t.Fatalf("pointer activated a disabled option: %+v", st)
			}
			if got := events(); len(got) != 2 {
				t.Fatalf("disabled pointer choice fired events: %v", got)
			}
			click(0)
			b.poll(`document.querySelectorAll('.mpress-ddlb')[0].querySelector('select').value==='stable'`, "pointer did not choose the enabled option")
			if got := events(); len(got) != 4 || got[2] != "release:input" || got[3] != "release:change" {
				t.Fatalf("enabled pointer choice fired the wrong events: %v", got)
			}

			var channel struct {
				Value  string
				Events int
			}
			b.js(`(() => {const s=document.querySelector('#channel');return {Value:s.value,Events:(window.__selectEvents||[]).filter(event=>event.startsWith('channel:')).length};})()`, &channel)
			if channel.Value != "alpha" || channel.Events != 0 {
				t.Fatalf("independent select was affected: %+v", channel)
			}
			// An unavailable current value must not leave keyboard focus on a disabled item.
			b.js(`document.querySelector('#release').selectedIndex=1;document.querySelector('#release').dispatchEvent(new Event('change',{bubbles:true}))`, nil)
			b.run(chromedp.Click(`.mpress-ddlb-trigger`))
			if st := state(); !st.Open || st.Focus != "Stable" {
				t.Fatalf("opening with a disabled selection did not focus an available option: %+v", st)
			}
			b.run(chromedp.KeyEvent(kb.Escape))

		})
	}
}

func buildSelectFixture(t *testing.T, binary string) string {
	t.Helper()
	project := t.TempDir()
	files := map[string]string{
		"mpress.yaml": "site:\n  title: Release choices\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n",
		"content/index.md": "---\ntitle: Release choices\n---\n\n# Release choices\n\n" +
			"<section id=\"release-form\">\n" +
			"<label for=\"release\">Release</label>\n" +
			"<select id=\"release\" name=\"release\">\n" +
			"<option value=\"stable\" selected>Stable</option>\n" +
			"<optgroup label=\"Internal\" disabled><option value=\"internal\">Internal</option></optgroup>\n" +
			"<option value=\"legacy\" disabled>Disabled</option>\n" +
			"<option value=\"next\">Next</option>\n" +
			"</select>\n" +
			"<label for=\"channel\">Channel</label>\n" +
			"<select id=\"channel\" name=\"channel\">\n" +
			"<option value=\"alpha\" selected>Alpha</option>\n" +
			"<option value=\"beta\">Beta</option>\n" +
			"<option value=\"gamma\" disabled>Gamma</option>\n" +
			"</select>\n" +
			"</section>\n\nChoose a release.\n",
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

// A separate synthetic char event can activate the newly focused trigger after
// the option's Enter handler has already selected and closed the listbox.
func selectEnter() chromedp.Action {
	return chromedp.Tasks{
		input.DispatchKeyEvent(input.KeyDown).WithKey("Enter").WithCode("Enter").WithWindowsVirtualKeyCode(13),
		input.DispatchKeyEvent(input.KeyUp).WithKey("Enter").WithCode("Enter").WithWindowsVirtualKeyCode(13),
	}
}
