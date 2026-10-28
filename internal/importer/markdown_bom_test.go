package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

const bomTestFrontmatter = "title: \"Café Guide\"\nlayout: docs\nslug: /cafe\ndraft: true\n"

func requireBOMTestMetadata(t *testing.T, converted string) {
	t.Helper()
	for _, want := range []string{
		"title = \"Café Guide\"",
		"layout = \"docs\"",
		"slug = \"/cafe\"",
		"draft = true",
	} {
		if !strings.Contains(converted, want) {
			t.Errorf("converted MPD is missing %q:\n%s", want, converted)
		}
	}
	if !strings.HasPrefix(converted, "---\nschema = 1\n") {
		t.Errorf("converted MPD does not open with MPD frontmatter:\n%s", converted)
	}
}

func TestMarkdownToMPDIgnoresLeadingBOMWithLF(t *testing.T) {
	converted, err := MarkdownToMPD("\uFEFF---\n" + bomTestFrontmatter + "---\n# BOM Body\n")
	if err != nil {
		t.Fatal(err)
	}
	requireBOMTestMetadata(t, converted)
	if !strings.Contains(converted, "BOM Body") {
		t.Errorf("converted MPD dropped the body:\n%s", converted)
	}
	if strings.Contains(converted, "\uFEFF") {
		t.Errorf("converted MPD leaked the leading BOM:\n%s", converted)
	}
}

func TestMarkdownToMPDIgnoresLeadingBOMWithCRLF(t *testing.T) {
	converted, err := MarkdownToMPD("\uFEFF---\r\n" + strings.ReplaceAll(bomTestFrontmatter, "\n", "\r\n") + "---\r\n# BOM Body\r\n")
	if err != nil {
		t.Fatal(err)
	}
	requireBOMTestMetadata(t, converted)
	if !strings.Contains(converted, "BOM Body") {
		t.Errorf("converted MPD dropped the body:\n%s", converted)
	}
}

func TestMarkdownToMPDNoBOMMetadataUnchanged(t *testing.T) {
	converted, err := MarkdownToMPD("---\n" + bomTestFrontmatter + "---\n# Plain Body\n")
	if err != nil {
		t.Fatal(err)
	}
	requireBOMTestMetadata(t, converted)
}

func TestMarkdownToMPDBOMRepeatedConversionIsStable(t *testing.T) {
	first, err := MarkdownToMPD("\uFEFF---\n" + bomTestFrontmatter + "---\n# BOM Body\n")
	if err != nil {
		t.Fatal(err)
	}
	document := parseImportedMPD(t, first)
	markdown, err := mpd.Markdown(document)
	if err != nil {
		t.Fatal(err)
	}
	second, err := MarkdownToMPD(string(markdown))
	if err != nil {
		t.Fatal(err)
	}
	requireBOMTestMetadata(t, second)
	if second != first {
		t.Errorf("repeated conversion is unstable:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestMarkdownToMPDBOMWithoutFrontmatterKeepsHeading(t *testing.T) {
	converted, err := MarkdownToMPD("\uFEFF# BOM Heading\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(converted, "\uFEFF") {
		t.Errorf("converted MPD leaked the leading BOM:\n%s", converted)
	}
	if !strings.Contains(converted, "BOM Heading") {
		t.Errorf("converted MPD dropped the heading:\n%s", converted)
	}
	if strings.HasPrefix(converted, "---") {
		t.Errorf("plain heading gained frontmatter:\n%s", converted)
	}
}

func TestMarkdownToMPDKeepsInteriorFEFF(t *testing.T) {
	converted, err := MarkdownToMPD("---\ntitle: \"a\uFEFFb\"\n---\n# Body\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(converted, "a\uFEFFb") {
		t.Errorf("converted MPD altered interior U+FEFF:\n%s", converted)
	}
}

func TestMarkdownToMPDBOMMalformedYAMLStillErrors(t *testing.T) {
	if _, err := MarkdownToMPD("\uFEFF---\ntitle: [\n---\n# Body\n"); err == nil {
		t.Fatal("malformed BOM-prefixed frontmatter was accepted")
	}
	if _, err := MarkdownToMPD("\uFEFF---\ntitle: x\n"); err == nil {
		t.Fatal("unterminated BOM-prefixed frontmatter was accepted")
	}
}
