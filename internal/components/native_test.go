package components

import (
	"strings"
	"testing"
)

func TestNativeTerminalAndInlineComponents(t *testing.T) {
	input := `@terminal{title="Build the documentation" language=bash frame=generic}
$ mpress build --strict
Built 12 pages
@end

@button[Get started](/start/){primary|size=m} {badge.warning:Preview}
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{"mpress-terminal-generic", "Build the documentation", "mpress build", `class="mpress-token-keyword">--strict</span>`, "mpress-button-primary", `href="/start/"`, "mpress-badge-warning"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestNativeButtonCanOpenAccessibilityControls(t *testing.T) {
	input := `@button[Try the accessibility controls](#){secondary|action=accessibility}`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`<button class="mpress-button mpress-button-secondary"`, `type="button"`, `data-open-accessibility`, `Try the accessibility controls`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `href=`) {
		t.Fatalf("accessibility action rendered as a link:\n%s", got)
	}
}

func TestLinkCardUsesDescriptionMetadata(t *testing.T) {
	input := `@linkcard{title="Agent control" href="/agent/" description="Inspect a running <application>."}
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`href="/agent/"`, "Agent control", `class="mpress-linkcard-desc">Inspect a running &lt;application&gt;.</span>`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestNativeTerminalWithoutVisiblePrompt(t *testing.T) {
	input := `@terminal{frame=macos|prompt=none}
# Install Wails
wails3 setup
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`mpress-terminal-macos`, `class="mpress-cmd"><span class="mpress-token-function">wails3</span> setup`, `data-commands="wails3 setup"`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `class="mpress-prompt"`) {
		t.Fatalf("prompt=none rendered a visible prompt:\n%s", got)
	}
	if strings.Contains(got, `class="mpress-output"></span>`) {
		t.Fatalf("terminal rendered a trailing empty line:\n%s", got)
	}
}

func TestTerminalCopyExcludesCustomPromptsAndComments(t *testing.T) {
	input := `@terminal{title="Verify" prompt="❯" comment="//"}
// Explain the next command.
❯ // This prompted comment is also display-only.
❯ mpress check
No issues found.
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{
		`data-prompt="❯"`,
		`data-comment="//"`,
		`data-commands="mpress check"`,
		`class="mpress-prompt">❯</span>`,
		`class="mpress-comment">// Explain the next command.</span>`,
		`class="mpress-comment">// This prompted comment is also display-only.</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("terminal output missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{`data-commands="❯`, `data-commands="//`, `data-commands="This prompted comment`} {
		if strings.Contains(got, unwanted) {
			t.Errorf("terminal copy payload contains %q:\n%s", unwanted, got)
		}
	}
}

func TestNativeContainerAndAdmonition(t *testing.T) {
	input := `@container{display=grid|columns=2|gap=1rem|class=features}
@tip[Portable]
Markdown remains ordinary text.
@end
@end
`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{"mpress-container features", "grid-template-columns", "mpress-admonition-tip", "Portable", "Markdown remains ordinary text"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestNativeNoteTypeAttributeIsPreserved(t *testing.T) {
	tests := []struct {
		noteType string
		label    string
		variant  string
		icon     string
	}{
		{noteType: "info", label: "Info", icon: "circle-info"},
		{noteType: "tip", label: "Tip", variant: "tip", icon: "rocket"},
		{noteType: "warning", label: "Warning", variant: "warning", icon: "triangle-alert"},
		{noteType: "caution", label: "Warning", variant: "warning", icon: "triangle-alert"},
		{noteType: "danger", label: "Danger", variant: "danger", icon: "circle-x"},
		{noteType: "important", label: "Important", variant: "warning", icon: "star"},
	}
	for _, test := range tests {
		t.Run(test.noteType, func(t *testing.T) {
			input := "@note{type=\"" + test.noteType + "\"}\nBody\n@end\n"
			got, warnings := ProcessWithLanguage(input, "en")
			if len(warnings) != 0 {
				t.Fatalf("warnings: %#v", warnings)
			}
			classType := test.noteType
			if classType == "caution" {
				classType = "warning"
			}
			for _, want := range []string{
				"mpress-admonition-" + classType,
				`aria-label="` + test.label + `"`,
				`class="lucide lucide-` + test.icon + `"`,
			} {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
			if test.variant == "" {
				if strings.Contains(got, "data-variant=") {
					t.Errorf("default info note received a variant:\n%s", got)
				}
			} else if want := `data-variant="` + test.variant + `"`; !strings.Contains(got, want) {
				t.Errorf("output missing %q:\n%s", want, got)
			}
		})
	}
}

func TestNativeThemeImage(t *testing.T) {
	input := `@image{light="/images/diagram-light.png" dark="/images/diagram-dark.png" alt="System diagram"}`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{"mpress-theme-image-light", "diagram-light.png", "mpress-theme-image-dark", "diagram-dark.png", `alt="System diagram"`} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
}

func TestNativeThemeImageAcceptsEscapedDescriptionQuotes(t *testing.T) {
	input := `@image{light="/images/diagram-light.png" dark="/images/diagram-dark.png" alt="The \"ready\" state"}`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	if !strings.Contains(got, `alt="The &#34;ready&#34; state"`) {
		t.Fatalf("escaped description quotes were not preserved:\n%s", got)
	}
}

func TestCalendarWithExplicitMonthIsBuildDateIndependent(t *testing.T) {
	input := "@calendar{month=\"2026-08\"|style=compact}\n@end\n"
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{`data-year="2026"`, `data-month="8"`, `data-date="2026-08-24"`} {
		if !strings.Contains(got, want) {
			t.Errorf("calendar output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `class="mpress-calendar-cell today"`) {
		t.Fatalf("calendar baked the build date into static output:\n%s", got)
	}
}

func TestNativeImageCanOpenAccessibleLightbox(t *testing.T) {
	input := `@image{src="/images/diagram.png" alt="System diagram" expand}`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{
		`class="mpress-image-expand"`,
		`aria-haspopup="dialog"`,
		`aria-label="Expand image: System diagram"`,
		`src="/images/diagram.png"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "mpress-image-expand-hint") || strings.Contains(got, "lucide-maximize-2") {
		t.Fatalf("expandable image contains obsolete visible expansion chrome:\n%s", got)
	}
}

func TestCalendarWithExplicitTodayIsDeterministic(t *testing.T) {
	input := "@calendar{month=\"2026-08\" today=\"2026-08-11\"}\n@end\n"
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	if !strings.Contains(got, `class="mpress-calendar-cell today" data-date="2026-08-11"`) {
		t.Fatalf("calendar did not mark the explicit date:\n%s", got)
	}
}

func TestNativeImageDoesNotExpandByDefault(t *testing.T) {
	got, warnings := ProcessWithLanguage(`@image{src="/images/diagram.png" alt="System diagram"}`, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	if strings.Contains(got, "mpress-image-expand") {
		t.Fatalf("ordinary image unexpectedly became expandable:\n%s", got)
	}
}

func TestNativeVideoLeafRendersWithoutEndMarker(t *testing.T) {
	input := `@video{base="/media" src="setup.webm,setup.mp4" title="Setup"}`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{
		`<figure class="mpress-video">`,
		`<source src="/media/setup.webm" type="video/webm">`,
		`<source src="/media/setup.mp4" type="video/mp4">`,
		`<figcaption>Setup</figcaption>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@video") {
		t.Fatalf("native video declaration leaked into output:\n%s", got)
	}
}

func TestLeafComponentDoesNotConsumeContainingLayout(t *testing.T) {
	input := `@section{variant=hero}
@column{variant=site-screenshot}
@image{light="/images/light.png" dark="/images/dark.png" alt="Preview"}
@end
@end`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %v", warnings)
	}
	for _, want := range []string{`mpress-section-hero`, `mpress-column-site-screenshot`, `src="/images/light.png"`, `src="/images/dark.png"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "@section") || strings.Contains(got, "@image") || strings.Contains(got, "@end") {
		t.Fatalf("leaf component left raw directives:\n%s", got)
	}
}

func TestNestedNativeComponentsConsumeEveryEndMarker(t *testing.T) {
	input := `@tutorial{title="Publish safely"}
### Validate
@note{type="warning"}
Fix every broken internal link.
@end
@end`
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %v", warnings)
	}
	if strings.Contains(got, "@end") {
		t.Fatalf("nested components left raw end markers:\n%s", got)
	}
}

func TestPlatformTabsUseStarlightPlatformIcons(t *testing.T) {
	input := ":::tabs\n[Windows]\n\nWindows setup.\n\n[macOS]\n\nmacOS setup.\n\n[Linux]\n\nLinux setup.\n:::\n"
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, want := range []string{"starlight-icon-seti-windows", "starlight-icon-apple", "starlight-icon-linux"} {
		if !strings.Contains(got, want) {
			t.Errorf("platform tabs missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "lucide-monitor") {
		t.Fatalf("platform tabs still use the generic monitor icon:\n%s", got)
	}
}

func TestAdvancedRegistryParity(t *testing.T) {
	want := []string{"tabs", "terminal", "note", "details", "badge", "api", "steps", "cards", "diff", "filetree", "linkcard", "calendar", "changelog", "matrix", "tutorial", "status", "pricing", "release", "testimonials", "api-playground", "qr", "audience", "explained", "input", "computed", "button", "image", "video", "if", "variant", "container", "section", "columns", "column", "actions", "headline", "docs-preview", "preview-tabs", "file-tabs", "callout", "timeline", "capabilities", "resources"}
	for _, name := range want {
		if Registry[name] == nil {
			t.Errorf("component %q is not registered", name)
		}
	}
}
