package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

func TestMarkdownToMPDPreservesImageExpandRoundTrip(t *testing.T) {
	source := `@image{src="/images/diagram.png" alt="System diagram" expand}
`
	native, markdown, html := renderImportedTable(t, source)

	if !strings.Contains(native, `expand="true"`) {
		t.Errorf("converted MPD dropped the image expand flag:\n%s", native)
	}
	if !strings.Contains(markdown, `expand="true"`) {
		t.Errorf("MPD-to-Markdown dropped the image expand flag:\n%s", markdown)
	}
	if !strings.Contains(html, `class="mpress-image-expand"`) {
		t.Errorf("rendered image has no lightbox trigger:\n%s", html)
	}
	reimported, err := MarkdownToMPD(markdown)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := mpd.Markdown(mpd.Parse("roundtrip.mpd", []byte(reimported)))
	if err != nil {
		t.Fatal(err)
	}
	if string(roundTrip) != markdown {
		t.Errorf("round trip changed the interchange form:\nfirst:\n%s\nsecond:\n%s", markdown, roundTrip)
	}
}

func TestMarkdownToMPDPreservesPreviewTabSourceRoundTrip(t *testing.T) {
	source := `@preview-tabs{source}
[Terminal]

Run the build.

[Steps]

One, two, three.
@end
`
	native, markdown, html := renderImportedTable(t, source)

	if !strings.Contains(native, `source="true"`) {
		t.Errorf("converted MPD dropped the preview-tabs source flag:\n%s", native)
	}
	if !strings.Contains(markdown, `source="true"`) {
		t.Errorf("MPD-to-Markdown dropped the preview-tabs source flag:\n%s", markdown)
	}
	if strings.Count(html, `class="mpress-preview-source"`) != 2 {
		t.Errorf("rendered preview tabs lost their source panels:\n%s", html)
	}
	reimported, err := MarkdownToMPD(markdown)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := mpd.Markdown(mpd.Parse("roundtrip.mpd", []byte(reimported)))
	if err != nil {
		t.Fatal(err)
	}
	if string(roundTrip) != markdown {
		t.Errorf("round trip changed the interchange form:\nfirst:\n%s\nsecond:\n%s", markdown, roundTrip)
	}
}

func TestMarkdownToMPDKeepsExplicitFalseFlags(t *testing.T) {
	source := `@image{src="/images/plain.png" alt="Plain" expand="false"}

@preview-tabs{source=false}
[Only]

Just text.
@end
`
	native, markdown, html := renderImportedTable(t, source)

	if !strings.Contains(native, `expand="false"`) || !strings.Contains(markdown, `expand="false"`) {
		t.Errorf("explicit expand=false did not survive conversion:\n%s\n%s", native, markdown)
	}
	if !strings.Contains(native, `source="false"`) || !strings.Contains(markdown, `source="false"`) {
		t.Errorf("explicit source=false did not survive conversion:\n%s\n%s", native, markdown)
	}
	if strings.Contains(html, `class="mpress-image-expand"`) {
		t.Errorf("expand=false still rendered a lightbox trigger:\n%s", html)
	}
	if strings.Contains(html, `class="mpress-preview-source"`) {
		t.Errorf("source=false still rendered source panels:\n%s", html)
	}
}

func TestMarkdownToMPDDropsUnknownBareMetadata(t *testing.T) {
	for _, source := range []string{
		"@image{src=\"/images/plain.png\" alt=\"Plain\" glow}\n",
		"@note{expand source}\nBody.\n@end\n",
	} {
		native, _, _ := renderImportedTable(t, source)
		for _, flag := range []string{"glow", "expand", "source"} {
			if strings.Contains(native, flag) {
				t.Errorf("unknown bare metadata %q was not dropped:\n%s", flag, native)
			}
		}
	}
}

func TestMarkdownToMPDKeepsFencedFlagExamplesLiteral(t *testing.T) {
	source := "```md\n@image{src=\"/images/diagram.png\" alt=\"System diagram\" expand}\n```\n\n~~~md\n@preview-tabs{source}\n[Terminal]\nRun the build.\n@end\n~~~\n"
	native, markdown, html := renderImportedTable(t, source)

	for _, want := range []string{`@image{src="/images/diagram.png" alt="System diagram" expand}`, "@preview-tabs{source}"} {
		if !strings.Contains(native, want) {
			t.Errorf("fenced example %q did not stay literal:\n%s", want, native)
		}
		if !strings.Contains(markdown, want) {
			t.Errorf("fenced example %q did not survive the round trip:\n%s", want, markdown)
		}
	}
	if strings.Contains(html, `class="mpress-image-expand"`) || strings.Contains(html, `class="mpress-preview-source"`) {
		t.Errorf("fenced examples rendered live expansion or source panels:\n%s", html)
	}
}
