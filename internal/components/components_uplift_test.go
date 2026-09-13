package components

import (
	"strings"
	"testing"
)

func TestStepsParseOrderedBoldMarkdownSteps(t *testing.T) {
	var steps Steps
	err := steps.Parse(`1. **Generate the project**

   Run the init command.

2. **Run the app**

   Start the dev server.`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(steps.Items) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps.Items))
	}
	if steps.Items[0].Title != "Generate the project" || steps.Items[1].Title != "Run the app" {
		t.Fatalf("unexpected step titles: %#v", steps.Items)
	}
	if !strings.Contains(steps.Items[0].Content, "Run the init command.") {
		t.Fatalf("unexpected first step content: %q", steps.Items[0].Content)
	}
}

func TestStepsRenderDoubleDigitMarkers(t *testing.T) {
	steps := Steps{Items: make([]StepItem, 12)}
	for index := range steps.Items {
		steps.Items[index] = StepItem{Title: "Step", Content: "Content"}
	}
	got, err := steps.Render()
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(got, `class="mpress-timeline-marker mpress-step-number"`); count != 12 {
		t.Fatalf("rendered %d step markers, want 12:\n%s", count, got)
	}
	for _, want := range []string{
		`class="mpress-timeline mpress-steps"`,
		`class="mpress-timeline-entry mpress-step"`,
		`class="mpress-timeline-body mpress-step-body"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered steps do not contain shared timeline class %q:\n%s", want, got)
		}
	}
	for _, number := range []string{">9</div>", ">10</div>", ">11</div>", ">12</div>"} {
		if !strings.Contains(got, number) {
			t.Errorf("rendered steps do not contain %q:\n%s", number, got)
		}
	}
}

func TestTestimonialsPreserveUnicodeQuotes(t *testing.T) {
	component := &Testimonials{Meta: map[string]string{}}
	if err := component.Parse("“The content stays ordinary Markdown.”\n— Documentation author"); err != nil {
		t.Fatal(err)
	}
	got, err := component.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "The content stays ordinary Markdown.") || strings.Contains(got, "�") {
		t.Fatalf("unicode quote parsing corrupted the testimonial: %s", got)
	}
}

func TestProcessTabsWithUnquotedMetadata(t *testing.T) {
	got, warnings := Process(":::tabs{sync-key=os}\n[macOS]\n\n```bash\nwails3 dev\n```\n\n[Windows]\n\n```powershell\nwails3 dev\n```\n:::\n")
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if !strings.Contains(got, "<ot-tabs") || !strings.Contains(got, `data-sync-key="os"`) || strings.Contains(got, ":::tabs") {
		t.Fatalf("tabs block was not expanded:\n%s", got)
	}
}

func TestExplainedSupportsTildeFencesAndPreservesIndentation(t *testing.T) {
	got, warnings := Process(`:::explained
~~~go
func main() { // (1)
    site.Build() // (2)
}
~~~

(1) Start the program.

(2) Build the site.
:::
`)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{`class="language-go"`, `data-ref="1" data-explanation="Start the program." tabindex="0"`, `data-ref="2" data-explanation="Build the site." tabindex="0"`, `class="mpress-explained-copy"`, "    site.Build()", "Start the program.", "Build the site."} {
		if !strings.Contains(got, want) {
			t.Errorf("explained code output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "~~~") {
		t.Fatalf("explained code leaked its tilde fence:\n%s", got)
	}
}

func TestExplainedNativeSyntaxKeepsKeyboardAndTooltipMarkup(t *testing.T) {
	got, warnings := Process(`@explained
~~~yaml
site: # (1)
~~~

(1) Site settings.
@end
`)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{`data-ref="1" data-explanation="Site settings." tabindex="0"`, `class="mpress-explained-copy"`, "Site settings."} {
		if !strings.Contains(got, want) {
			t.Errorf("native explained output missing %q:\n%s", want, got)
		}
	}
}

func TestExplainedSplitsAdjacentNumberedExplanations(t *testing.T) {
	got, warnings := Process(`@explained
~~~go
func main() { // (1)
    site.Build() // (2)
}
~~~

(1) Start with the entry point.
(2) Build the static site.
@end
`)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{
		`data-ref="1" data-explanation="Start with the entry point." tabindex="0"`,
		`data-ref="2" data-explanation="Build the static site." tabindex="0"`,
		`class="mpress-explained-para" data-ref="1"`,
		`class="mpress-explained-para" data-ref="2"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("explained code output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Start with the entry point. (2)") {
		t.Fatalf("adjacent explanation 2 was merged into explanation 1:\n%s", got)
	}
}

func TestTableInteractiveFeaturesAreIndependent(t *testing.T) {
	tests := []struct {
		name    string
		feature string
		want    string
	}{
		{name: "search", feature: "search", want: `data-table-query`},
		{name: "filter", feature: "filter", want: `data-table-filter-column="0"`},
		{name: "sort", feature: "sort", want: `data-table-sort-column="0"`},
		{name: "paginate", feature: "paginate page-size=1", want: `data-table-page-next`},
		{name: "column separators", feature: "column-separators", want: `class="mpress-data-table mpress-table-column-separators"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, warnings := Process("@table{header=true " + test.feature + "}\n| Package | Downloads |\n| Core | 1840 |\n| Studio | 920 |\n@end\n")
			if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %#v", warnings)
			}
			if !strings.Contains(got, test.want) {
				t.Fatalf("table output missing %q:\n%s", test.want, got)
			}
		})
	}
}

func TestTableCombinesSearchFilterAndSortAccessibly(t *testing.T) {
	got, warnings := Process(`@table{header search filter sort paginate column-separators page-size=2}
| Package | Platform | Downloads |
| Core | Linux | 1840 |
| Studio | macOS | 920 |
| Core | Windows | 1260 |
@end
`)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{
		`class="mpress-data-table mpress-table-column-separators"`,
		`aria-controls="mpress-table-`,
		`aria-sort="none"`,
		`role="status" aria-live="polite"`,
		`role="menuitem" data-table-filter-value="linux">Linux</button>`,
		`data-table-value="1840"`,
		`data-table-page-status>Page 1 of 2</span>`,
		`data-table-column-separators="true"`,
		`class="lucide lucide-search"`,
		`class="lucide lucide-arrow-up-down"`,
		`class="lucide lucide-list-filter"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("interactive table output missing %q:\n%s", want, got)
		}
	}
}

func TestTableRejectsRowsWithDifferentWidths(t *testing.T) {
	table := &Table{Meta: map[string]string{"header": "true"}}
	if err := table.Parse("| A | B |\n| one |"); err == nil {
		t.Fatal("expected mismatched table row to fail")
	}
}

func TestTableRendersEscapedPipesInlineMarkdownAndAlignment(t *testing.T) {
	table := &Table{Meta: map[string]string{
		"header": "true",
		"align":  `["left","center","right"]`,
	}}
	if err := table.Parse("| Capability | Support | Time |\n| Escaped \\| pipe | *Complete* | `4 ms` |"); err != nil {
		t.Fatal(err)
	}
	got, err := table.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`style="text-align:left"`,
		`style="text-align:center"`,
		`style="text-align:right"`,
		`Escaped | pipe`,
		`<em>Complete</em>`,
		`<code>4 ms</code>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("table output missing %q:\n%s", want, got)
		}
	}
}

func TestInlineDiffDoesNotInsertBlankPreformattedLines(t *testing.T) {
	got, warnings := Process(":::diff{title=\"settings.yaml\" mode=\"inline\"}\nsearch: false\n---\nsearch: true\n:::\n")
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{"mpress-diff-title", "settings.yaml", "mpress-diff-removed", "mpress-diff-added"} {
		if !strings.Contains(got, want) {
			t.Errorf("diff output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "mpress-diff-header") || strings.Contains(got, "Before") || strings.Contains(got, "After") {
		t.Fatalf("diff renderer included redundant before/after labels:\n%s", got)
	}
	if strings.Contains(got, "</span>\n<span class=\"mpress-diff-line") {
		t.Fatalf("diff renderer inserted whitespace-only preformatted lines:\n%s", got)
	}
}

func TestDetailsUsesLucideDisclosureIcon(t *testing.T) {
	got, warnings := Process(":::details{title=\"Open this disclosure\"}\nContent.\n:::\n")
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{`class="mpress-details"`, `class="mpress-disclosure-icon"`, `class="lucide lucide-chevron-right"`, `class="lucide lucide-chevron-down"`, `<span>Open this disclosure</span>`} {
		if !strings.Contains(got, want) {
			t.Errorf("details output missing %q:\n%s", want, got)
		}
	}
}

func TestVisualIndicatorsUseLucideIcons(t *testing.T) {
	yes, _ := renderIndicator("yes")
	no, _ := renderIndicator("no")
	for name, got := range map[string]string{"matrix yes": yes, "matrix no": no} {
		if !strings.Contains(got, `class="lucide `) {
			t.Errorf("%s did not render a Lucide icon: %s", name, got)
		}
	}
}

func TestAPIPlaygroundDisclosureUsesLucideIcons(t *testing.T) {
	got, warnings := Process(":::api-playground{method=POST path=/v1/builds}\nBuild.\n:::\n")
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if count := strings.Count(got, `class="lucide lucide-chevron-right"`); count != 2 {
		t.Fatalf("expected two Lucide disclosure icons, got %d:\n%s", count, got)
	}
}

func TestProcessTabsWithNestedAdmonition(t *testing.T) {
	got, warnings := Process(`:::tabs
[Go]

    Install Go.

[npm]

    Install npm.

:::note{type="info"}
      Use any package manager.
:::

:::
`)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	for _, want := range []string{"<ot-tabs", `role="tabpanel"`, "mpress-admonition", "Use any package manager."} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rendered tabs to contain %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{":::tabs", "<p>:::</p>", "```"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("rendered tabs leaked %q:\n%s", unwanted, got)
		}
	}
}

func TestProcessTabsKeepsNestedTabsInsidePanel(t *testing.T) {
	got, warnings := Process(`:::tabs{sync-key=os}
[Linux]

Choose your distro:

:::tabs{sync-key=distro}
[Ubuntu/Debian]

sudo apt install libwebkitgtk-6.0-dev

[Fedora]

sudo dnf install webkitgtk6.0-devel
:::

[macOS]

xcode-select --install

:::
`)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if strings.Count(got, "<ot-tabs") != 2 {
		t.Fatalf("expected outer and nested tabs, got:\n%s", got)
	}
	for _, want := range []string{`data-sync-key="os"`, `data-sync-key="distro"`, "Ubuntu/Debian", "Fedora", "macOS"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rendered tabs to contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, `<button role="tab" type="button" aria-selected="false" tabindex="-1">Ubuntu/Debian</button>`) {
		t.Fatalf("nested tab label was promoted to the outer tablist:\n%s", got)
	}
}

func TestTabsParseNormalizesImportedStarlightIndentation(t *testing.T) {
	var tabs Tabs
	err := tabs.Parse(`[Bugs]

    Report the issue.

    - Include ` + "`wails3 doctor`" + ` output.

    ` + "```d2" + `
    app -> webview
    ` + "```" + `

[Fixes]

  Open a pull request.
`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(tabs.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(tabs.Sections))
	}

	first := tabs.Sections[0].Content
	for _, want := range []string{"Report the issue.", "- Include `wails3 doctor` output.", "```d2\napp -> webview\n```"} {
		if !strings.Contains(first, want) {
			t.Fatalf("expected normalized tab content to contain %q:\n%s", want, first)
		}
	}
	for _, unwanted := range []string{"    Report the issue.", "    - Include", "    ```d2"} {
		if strings.Contains(first, unwanted) {
			t.Fatalf("tab content kept imported indentation %q:\n%s", unwanted, first)
		}
	}
	if got := tabs.Sections[1].Content; got != "Open a pull request." {
		t.Fatalf("expected two-space imported indentation to normalize, got %q", got)
	}
}

func TestProgressiveComponentsRemainReadableWithoutJavaScript(t *testing.T) {
	audience, warnings := Process(":::audience{role=writer}\nWriter guidance.\n:::\n")
	if len(warnings) != 0 || strings.Contains(audience, `style="display:none"`) || !strings.Contains(audience, "Writer guidance") {
		t.Fatalf("audience content is not static-first: warnings=%#v output=%s", warnings, audience)
	}
	conditional, warnings := Process(":::if{param=framework value=go default=true}\nGo guidance.\n:::\n")
	if len(warnings) != 0 || strings.Contains(conditional, `style="display:none"`) || !strings.Contains(conditional, "Go guidance") {
		t.Fatalf("conditional content is not static-first: warnings=%#v output=%s", warnings, conditional)
	}
	testimonials, warnings := Process(":::testimonials\nA useful quote.\n— Lea\n---\nAnother quote.\n— Ada\n:::\n")
	if len(warnings) != 0 || !strings.Contains(testimonials, `data-autoplay="0"`) || !strings.Contains(testimonials, `aria-live="off"`) {
		t.Fatalf("testimonials should be reader-controlled by default: warnings=%#v output=%s", warnings, testimonials)
	}
}

func TestFileTreeRendersImportedStarlightListTree(t *testing.T) {
	var tree FileTree
	err := tree.Parse(`- build/           Contains files used by the build process
    - appicon.png  The application icon
    - darwin/      macOS specific build files
        - Info.plist    Production configuration
- frontend/        Frontend application files
    - Inter Font License.txt Font license`)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	got, err := tree.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, want := range []string{
		`mpress-filetree-dir`,
		`<span class="mpress-filetree-name">build</span>`,
		`<span class="mpress-filetree-desc">Contains files used by the build process</span>`,
		`<span class="mpress-filetree-name">appicon.png</span>`,
		`<span class="mpress-filetree-name">darwin</span>`,
		`<span class="mpress-filetree-name">Info.plist</span>`,
		`<span class="mpress-filetree-name">frontend</span>`,
		`Inter Font License.txt Font license`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered file tree missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{
		`<span class="mpress-filetree-name">- build/`,
		`<span class="mpress-filetree-name">- appicon.png`,
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("rendered file tree kept Markdown list marker %q:\n%s", unwanted, got)
		}
	}
}

func TestNoteRendersCautionAsLabelledAside(t *testing.T) {
	note := Note{
		Meta: map[string]string{
			"type":  "caution",
			"title": "Minimum distro versions",
		},
		Content: "Use supported packages.",
	}

	got, err := note.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	for _, want := range []string{
		`<aside aria-label="Minimum distro versions" class="mpress-admonition mpress-admonition-warning" data-variant="warning">`,
		`<p class="mpress-admonition-titlebar" aria-hidden="true">`,
		`<div class="mpress-admonition-body">`,
		"Use supported packages.",
		`</aside>`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered caution note missing %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{`role="alert"`, `<div role=`} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("rendered caution note kept old ARIA/container markup %q:\n%s", unwanted, got)
		}
	}
}
