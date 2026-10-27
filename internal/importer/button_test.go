package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

// mpdButtonAttributes returns the first button component's attributes in a
// parsed document. Flag attributes are reported as "true"; explicit values keep
// their JSON quoting, matching the native document representation.
func mpdButtonAttributes(t *testing.T, document *mpd.Document) map[string]string {
	t.Helper()
	var button *mpd.Node
	var walk func(uint32) bool
	walk = func(index uint32) bool {
		node := &document.Nodes[index]
		if node.Kind == mpd.KindComponent && string(document.Text(node.Name)) == "button" {
			button = node
			return true
		}
		children := document.Children(index)
		for {
			child, _, ok := children.Next()
			if !ok {
				return false
			}
			if walk(child) {
				return true
			}
		}
	}
	walk(document.Root)
	if button == nil {
		t.Fatal("converted MPD has no button component")
	}
	attributes := make(map[string]string)
	for offset := 0; offset < int(button.AttrCount); offset++ {
		attribute := document.Attributes[int(button.FirstAttr)+offset]
		name := string(document.Text(attribute.Name))
		if attribute.Flag {
			attributes[name] = "true"
			continue
		}
		attributes[name] = string(document.Text(attribute.Value))
	}
	return attributes
}

// A legacy compact button with an accessibility action must keep its action,
// variant, href and label through Markdown -> MPD -> Markdown rendering.
func TestMarkdownToMPDCompactButtonKeepsAccessibilityAction(t *testing.T) {
	native, markdown, html := renderImportedTable(t, "# Settings\n\n@button[Open reading settings](#){primary|action=accessibility}\n")

	document := mpd.Parse("imported.mpd", []byte(native))
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("converted MPD is invalid: %s\n%s", diagnostic.Message, native)
		}
	}
	attributes := mpdButtonAttributes(t, document)
	for name, want := range map[string]string{
		"href":    `"#"`,
		"variant": `"primary"`,
		"action":  `"accessibility"`,
	} {
		if got := attributes[name]; got != want {
			t.Errorf("button attribute %s = %q, want %q\n%s", name, got, want, native)
		}
	}
	if strings.Contains(native, "primaryactionaccessibility") {
		t.Errorf("variant and action metadata were merged:\n%s", native)
	}
	if !strings.Contains(markdown, `@button{href="#" variant="primary" action="accessibility"}`) {
		t.Errorf("MPD-to-Markdown lost the button metadata:\n%s", markdown)
	}
	for _, want := range []string{
		`<button class="mpress-button mpress-button-primary" type="button" data-open-accessibility>`,
		`Open reading settings`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, `href=`) {
		t.Errorf("accessibility action rendered as an inert link:\n%s", html)
	}
}

// Ordinary compact buttons keep their link href, variant, size and icon side.
func TestMarkdownToMPDCompactButtonKeepsLinkMetadata(t *testing.T) {
	native, _, html := renderImportedTable(t, "@button[Read the guide](/guide/?from=nav){secondary|size=m|icon=arrow-right|icon-right}\n")

	document := mpd.Parse("imported.mpd", []byte(native))
	attributes := mpdButtonAttributes(t, document)
	for name, want := range map[string]string{
		"href":       `"/guide/?from=nav"`,
		"variant":    `"secondary"`,
		"size":       `"m"`,
		"icon":       `"arrow-right"`,
		"icon-right": `"true"`,
	} {
		if got := attributes[name]; got != want {
			t.Errorf("button attribute %s = %q, want %q\n%s", name, got, want, native)
		}
	}
	for _, want := range []string{
		`<a class="mpress-button mpress-button-secondary mpress-button-m" href="/guide/?from=nav">`,
		`Read the guide<span class="mpress-button-icon">`,
		`lucide-arrow-right`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "data-open-accessibility") {
		t.Errorf("ordinary link button gained an accessibility action:\n%s", html)
	}
}

// Quoted values keep spaces and literal pipes; mixed metadata keeps the
// legacy variant precedence and the accessibility action.
func TestMarkdownToMPDCompactButtonKeepsQuotedAndMixedMetadata(t *testing.T) {
	native, _, html := renderImportedTable(t, `@button[Mixed settings](#){outline|size="extra large"|icon="arrow|left"|action="accessibility"}`+"\n")

	document := mpd.Parse("imported.mpd", []byte(native))
	attributes := mpdButtonAttributes(t, document)
	for name, want := range map[string]string{
		"variant": `"outline"`,
		"size":    `"extra large"`,
		"icon":    `"arrow|left"`,
		"action":  `"accessibility"`,
	} {
		if got := attributes[name]; got != want {
			t.Errorf("button attribute %s = %q, want %q\n%s", name, got, want, native)
		}
	}
	for _, want := range []string{
		`<button class="mpress-button mpress-button-outline mpress-button-extralarge" type="button" data-open-accessibility>`,
		`Mixed settings`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered HTML missing %q:\n%s", want, html)
		}
	}
}

// Legacy flag precedence and the secondary default survive conversion.
func TestMarkdownToMPDCompactButtonVariantPrecedence(t *testing.T) {
	for _, test := range []struct {
		name     string
		metadata string
		want     string
	}{
		{name: "default", metadata: "action=accessibility", want: `"secondary"`},
		{name: "primary", metadata: "primary", want: `"primary"`},
		{name: "secondary over primary", metadata: "primary|secondary", want: `"secondary"`},
		{name: "outline over secondary", metadata: "primary|outline|secondary", want: `"outline"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			native, err := MarkdownToMPD("@button[Action](#){" + test.metadata + "}\n")
			if err != nil {
				t.Fatal(err)
			}
			document := mpd.Parse("imported.mpd", []byte(native))
			if got := mpdButtonAttributes(t, document)["variant"]; got != test.want {
				t.Fatalf("variant = %q, want %q\n%s", got, test.want, native)
			}
		})
	}
}

// Trailing text after a compact declaration and fenced examples stay outside
// the converted button form.
func TestMarkdownToMPDCompactButtonKeepsInlineTextAndFencedExamplesOpaque(t *testing.T) {
	native, _, html := renderImportedTable(t, "@button[Inline](/inline/){primary} plus trailing text\n\n```md\n@button[Fenced](#){primary|action=accessibility}\n```\n")

	document := mpd.Parse("imported.mpd", []byte(native))
	if got := countMPDComponents(document)["button"]; got != 0 {
		t.Fatalf("converted %d buttons from trailing text or a fenced example, want 0:\n%s", got, native)
	}
	if !strings.Contains(mpdCodeText(document), "@button[Fenced](#){primary|action=accessibility}") {
		t.Errorf("fenced button example did not stay literal:\n%s", native)
	}
	if strings.Contains(html, "data-open-accessibility") {
		t.Errorf("fenced example rendered a live accessibility button:\n%s", html)
	}
}

func TestMarkdownToMPDInlineCompactButtonSurvivesRoundTrip(t *testing.T) {
	source := "# Start\n\nUse @button[Read the guide](/guide/){secondary} inside prose with {badge.info:Beta} and [ordinary link](/docs/).\n"
	native, markdown, html := renderImportedTable(t, source)

	if !strings.Contains(native, "@button[Read the guide](/guide/){secondary}") {
		t.Errorf("converted MPD dropped the inline compact button:\n%s", native)
	}
	if !strings.Contains(markdown, "@button[Read the guide](/guide/){secondary}") {
		t.Errorf("MPD-to-Markdown dropped the inline compact button metadata:\n%s", markdown)
	}
	for _, want := range []string{
		`<a class="mpress-button mpress-button-secondary" href="/guide/">Read the guide</a>`,
		"Use ",
		" inside prose with ",
		`mpress-badge-info">Beta`,
		`href="/docs/"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered output missing %q:\n%s", want, html)
		}
	}
	if got := strings.Count(html, `class="mpress-button`); got != 1 {
		t.Errorf("rendered %d live buttons, want only the prose button:\n%s", got, html)
	}
}
