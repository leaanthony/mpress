package translate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

func TestExtractAndApplyPreserveMarkdownSyntax(t *testing.T) {
	source := []byte("---\ntitle: \"Build your first site\"\ndescription: Start with {product}.\n---\n\n" +
		"# Install **M-Press**\n\n" +
		"Read the [installation guide](https://example.com/install) for `mpress build`.\n\n" +
		":::note{type=\"info\" title=\"Keep this syntax\"}\nSave the file and run the build.\n:::\n\n" +
		"```go\nfmt.Println(\"do not translate\")\n```\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) < 7 {
		t.Fatalf("expected frontmatter and prose segments, got %#v", document.Segments)
	}
	for _, segment := range document.Segments {
		if strings.Contains(segment.Original, "https://") || strings.Contains(segment.Original, "mpress build") || strings.Contains(segment.Original, ":::note") || strings.Contains(segment.Original, "fmt.Println") {
			t.Fatalf("syntax or code was made translatable: %#v", segment)
		}
	}
	translations := map[string]string{}
	for _, segment := range document.Segments {
		value := "FR " + segment.Text
		translations[segment.ID] = value
	}
	output, err := Apply(document, translations)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, unchanged := range []string{"https://example.com/install", "`mpress build`", ":::note{type=\"info\" title=\"Keep this syntax\"}", "fmt.Println(\"do not translate\")", "{product}"} {
		if !strings.Contains(text, unchanged) {
			t.Errorf("output changed protected content %q:\n%s", unchanged, text)
		}
	}
}

func TestApplyRejectsMissingPlaceholder(t *testing.T) {
	document, err := Extract([]byte("Hello {name}.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 {
		t.Fatalf("got %d segments", len(document.Segments))
	}
	_, err = Apply(document, map[string]string{document.Segments[0].ID: "Bonjour."})
	if err == nil || !strings.Contains(err.Error(), "placeholder") {
		t.Fatalf("expected placeholder error, got %v", err)
	}
}

func TestExtractsMultilineAndNestedFrontmatterProse(t *testing.T) {
	source := []byte("---\ntitle: Example\ndescription: >-\n  First description line.\n  Second description line.\nhero:\n  tagline: Ship documentation.\n  actions:\n    - text: Get started\n      link: /start/\n---\n\nBody.\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, segment := range document.Segments {
		seen[segment.ID] = segment.Original
	}
	for id, value := range map[string]string{
		"fm-description-01": "First description line.",
		"fm-description-02": "Second description line.",
		"fm-tagline-01":     "Ship documentation.",
		"fm-text-01":        "Get started",
	} {
		if seen[id] != value {
			t.Errorf("%s = %q, want %q (all: %#v)", id, seen[id], value, seen)
		}
	}
	for _, value := range seen {
		if value == "/start/" {
			t.Fatal("frontmatter link was made translatable")
		}
	}
}

func TestValidateRejectsChangedCode(t *testing.T) {
	source, err := Extract([]byte("Hello.\n\n```go\nold()\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	err = Validate(source, []byte("Bonjour.\n\n```go\nnew()\n```\n"))
	if err == nil {
		t.Fatal("expected structural validation failure")
	}
}

func TestValidateAllowsTranslatedProsePunctuation(t *testing.T) {
	source, err := Extract([]byte("Command line interface\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(source, []byte("Interface de ligne de commande :\n")); err != nil {
		t.Fatalf("French prose punctuation was treated as structure: %v", err)
	}
	if err := Validate(source, []byte("@note translated text\n")); err == nil {
		t.Fatal("component syntax introduced by a translation was accepted")
	}
}

func TestPunctuationAfterInlineCodeIsInsideTheFollowingProseSegment(t *testing.T) {
	source := []byte("This writes to `mpress.yaml`, then translates the site.\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	values := map[string]string{}
	for _, segment := range document.Segments {
		if strings.HasPrefix(segment.Original, ", then") {
			found = true
		}
		values[segment.ID] = segment.Text
	}
	if !found {
		t.Fatalf("comma after inline code is outside the prose segments: %#v", document.Segments)
	}
	if _, err := Apply(document, values); err != nil {
		t.Fatalf("unchanged prose did not round trip: %v", err)
	}
	editable, err := ExtractEditable(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(editable) != len(document.Segments) {
		t.Fatalf("quick-edit and translation segment counts differ: %d != %d", len(editable), len(document.Segments))
	}
	for index := range editable {
		if editable[index].Start != document.Segments[index].Start || editable[index].End != document.Segments[index].End {
			t.Fatalf("segment %d ranges differ: %#v != %#v", index, editable[index], document.Segments[index])
		}
	}
}

func TestClosingParenthesisAfterInlineCodeHasStableChineseBoundary(t *testing.T) {
	source := []byte("Set `frame` (`macos`, `windows`, or `plain`) as metadata.\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, segment := range document.Segments {
		values[segment.ID] = segment.Text
		if strings.Contains(segment.Original, ") as metadata.") {
			values[segment.ID] = ") 作为元数据。"
		}
	}
	output, err := Apply(document, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatalf("equivalent Chinese punctuation changed the structure: %v\n%s", err, output)
	}
}

func TestExplainedCodeMarkerHasStableChineseBoundary(t *testing.T) {
	source := []byte("(2) One call turns the content tree into the static site.\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 || !strings.HasPrefix(document.Segments[0].Original, "(2) ") {
		t.Fatalf("explanation marker is outside source prose: %#v", document.Segments)
	}
	translated := strings.Replace(document.Segments[0].Text, "One call turns the content tree into the static site.", "一次调用即可将内容树转换为静态网站。", 1)
	output, err := Apply(document, map[string]string{document.Segments[0].ID: translated})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatalf("equivalent Chinese explanation marker changed the structure: %v\n%s", err, output)
	}
}

func TestPricingMarkerHasStableChineseBoundary(t *testing.T) {
	source := []byte("- ✓ Complete generator\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 || !strings.HasPrefix(document.Segments[0].Original, "✓ ") {
		t.Fatalf("pricing marker is outside source prose: %#v", document.Segments)
	}
	output, err := Apply(document, map[string]string{document.Segments[0].ID: "✓ 完整生成器"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatalf("equivalent Chinese pricing marker changed the structure: %v\n%s", err, output)
	}
}

func TestNumberedHeadingHasStableChineseBoundary(t *testing.T) {
	source := []byte("## 2. Translation hardening\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 || !strings.HasPrefix(document.Segments[0].Original, "2. ") {
		t.Fatalf("heading number is outside source prose: %#v", document.Segments)
	}
	translated := strings.Replace(document.Segments[0].Text, "Translation hardening", "翻译强化", 1)
	output, err := Apply(document, map[string]string{document.Segments[0].ID: translated})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatalf("equivalent numbered Chinese heading changed the structure: %v\n%s", err, output)
	}
}

func TestTranslationCanReorderProtectedNumbers(t *testing.T) {
	document, err := Extract([]byte("M-Press version 0.1 is ready.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 {
		t.Fatalf("number was split from prose: %#v", document.Segments)
	}
	segment := document.Segments[0]
	if !strings.Contains(segment.Text, "⟪MPRESS_") || strings.Contains(segment.Text, "0.1") {
		t.Fatalf("version number was not protected: %#v", segment)
	}
	output, err := Apply(document, map[string]string{segment.ID: "La version ⟪MPRESS_0⟫ de M-Press est prête."})
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != "La version 0.1 de M-Press est prête.\n" {
		t.Fatalf("number was not restored after reordering: %s", output)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
}

func TestMarkdownEscapesAreProtectedFromTranslation(t *testing.T) {
	document, err := Extract([]byte("Use escaped \\*asterisks\\* in prose.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 {
		t.Fatalf("escaped prose was split: %#v", document.Segments)
	}
	segment := document.Segments[0]
	if strings.Contains(segment.Text, `\*`) || len(segment.Placeholders) != 2 {
		t.Fatalf("Markdown escapes were exposed to the provider: %#v", segment)
	}
	output, err := Apply(document, map[string]string{segment.ID: "Utilisez des ⟪MPRESS_0⟫astérisques⟪MPRESS_1⟫ dans le texte."})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `\*astérisques\*`) {
		t.Fatalf("Markdown escapes were not restored: %s", output)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
}

func TestNavigationTranslationPreservesLinksAndNesting(t *testing.T) {
	source := []byte("- label: Home\n  link: /\n- label: Learn\n  items:\n    - label: Build your first site\n      link: /start/\n")
	document, err := ExtractNavigation(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 3 {
		t.Fatalf("segments = %#v", document.Segments)
	}
	values := map[string]string{}
	for _, segment := range document.Segments {
		values[segment.ID] = "FR " + segment.Text
	}
	output, err := Apply(document, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFile("_nav.yaml", "_nav.yaml", document, output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "link: /start/") || !strings.Contains(string(output), `label: "FR Build your first site"`) {
		t.Fatalf("navigation translation is wrong:\n%s", output)
	}
}

func TestComponentListMarkersRemainOutsideTranslationRanges(t *testing.T) {
	source := []byte(":::timeline\n1. **Plan** Define the work.\n2. **Build** Ship it.\n:::\n\n:::capabilities\n- **Fast** One binary.\n- **Portable** Static output.\n:::\n")
	document, err := Extract(source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, segment := range document.Segments {
		if strings.HasPrefix(segment.Original, "1. ") || strings.HasPrefix(segment.Original, "2. ") || strings.HasPrefix(segment.Original, "- ") {
			t.Fatalf("component list marker is translatable: %#v", segment)
		}
		values[segment.ID] = "FR " + segment.Text
	}
	output, err := Apply(document, values)
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"1. **", "2. **", "- **"} {
		if !strings.Contains(string(output), marker) {
			t.Fatalf("component marker %q changed:\n%s", marker, output)
		}
	}
}

func TestFastSegmentFormattingMatchesFormattedOutput(t *testing.T) {
	for _, value := range []int{0, 1, 9, 10, 99, 100, 9999, 10000} {
		if got, want := blockSegmentID(value, "paragraph", value), fmt.Sprintf("b%04d-paragraph-%02d", value, value); got != want {
			t.Fatalf("blockSegmentID(%d) = %q, want %q", value, got, want)
		}
		if got, want := numberedID("b0001-text-", value, 2), fmt.Sprintf("b0001-text-%02d", value); got != want {
			t.Fatalf("numberedID(%d) = %q, want %q", value, got, want)
		}
	}
}

func TestHashMatchesSHA256Hex(t *testing.T) {
	for _, value := range []string{"", "M-Press", "Unicode: Cymraeg 日本語"} {
		want := fmt.Sprintf("%x", sha256.Sum256([]byte(value)))
		if got := Hash(value); got != want {
			t.Fatalf("Hash(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestExtractMPDTranslatesVisibleContentAndPreservesSyntax(t *testing.T) {
	source := []byte(`---
schema = 1
title = "Build the site"
description = "A fast guide"
translationKey = "build-guide"
---

# Install **M-Press**

Use ` + "`mpress build`" + ` with {product}.

@steps
@step title="Create a project"
Run the command.
@end
@step title="Publish the site"
@button href="/deploy/" title="Deployment guide"
Continue
@end
@end
@end

@tabs
@tab label="Linux"
Install the package.
@end
@tab label="Windows"
Install the archive.
@end
@end

@terminal title="Build command"
$ mpress build --strict
@end
`)
	document, err := ExtractMPD("guide.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	if document.Format != "mpd" || document.TranslationKey != "build-guide" {
		t.Fatalf("unexpected MPD identity: %#v", document)
	}
	seen := map[string]bool{}
	translations := map[string]string{}
	for _, segment := range document.Segments {
		seen[segment.Original] = true
		if strings.Contains(segment.Text, "mpress build") {
			t.Fatalf("code or terminal command became translatable: %#v", segment)
		}
		translations[segment.ID] = "FR " + segment.Text
	}
	for _, want := range []string{"Build the site", "A fast guide", "Install **M-Press**", "Create a project", "Run the command.", "Publish the site", "Deployment guide", "Continue", "Linux", "Windows", "Install the archive.", "Build command"} {
		if !seen[want] {
			t.Errorf("visible MPD text %q was not extracted; got %#v", want, seen)
		}
	}
	output, err := Apply(document, translations)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, unchanged := range []string{"href=\"/deploy/\"", "`mpress build`", "$ mpress build --strict", "{product}", "translationKey = \"build-guide\""} {
		if !strings.Contains(text, unchanged) {
			t.Errorf("protected MPD syntax %q changed:\n%s", unchanged, text)
		}
	}
	for _, translated := range []string{"title=\"FR Create a project\"", "label=\"FR Linux\"", "FR Run the command."} {
		if !strings.Contains(text, translated) {
			t.Errorf("translation %q missing:\n%s", translated, text)
		}
	}
}

func TestApplyMPDTranslationMayReflowSoftLines(t *testing.T) {
	source := []byte("# Guide\n\nFirst source line.\nSecond source line.\n")
	document, err := ExtractMPD("guide.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	var paragraph Segment
	for _, segment := range document.Segments {
		if segment.Kind == "text" {
			paragraph = segment
			break
		}
	}
	if paragraph.ID == "" {
		t.Fatal("missing MPD paragraph segment")
	}
	output, err := Apply(document, map[string]string{paragraph.ID: "翻译后的段落可以重新排版为一行。"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractMPD("translated.mpd", output); err != nil {
		t.Fatalf("reflowed MPD is invalid: %v\n%s", err, output)
	}
}

func TestApplyMPDTranslationDoesNotUseMarkdownPunctuationRules(t *testing.T) {
	source := []byte("# Guide\n\nChoose A > B for this workflow.\n")
	document, err := ExtractMPD("guide.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	var paragraph Segment
	for _, segment := range document.Segments {
		if segment.Kind == "text" {
			paragraph = segment
			break
		}
	}
	output, err := Apply(document, map[string]string{paragraph.ID: "此工作流请选择 A，而不是 B。"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(document, output); err != nil {
		t.Fatalf("translated MPD is invalid: %v\n%s", err, output)
	}
}

func TestApplyMPDTranslationPreservesSegmentBoundaryWhitespace(t *testing.T) {
	source := []byte("Run `mpress dev` and continue.\n")
	document, err := ExtractMPD("guide.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	translations := map[string]string{}
	for _, segment := range document.Segments {
		if segment.Kind == "text" {
			translations[segment.ID] = "  続行するには " + placeholderForValue(t, segment, "`mpress dev`") + " を実行します  "
		}
	}
	output, err := Apply(document, translations)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(output), "続行するには `mpress dev` を実行します\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
}

func TestExtractMPDUsesOneReorderableSentenceAroundInlineCode(t *testing.T) {
	source := []byte("M-Press requires only the `mpress` executable at runtime.\n")
	document, err := ExtractMPD("guide.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 1 {
		t.Fatalf("got %d segments, want one: %#v", len(document.Segments), document.Segments)
	}
	segment := document.Segments[0]
	placeholder := placeholderForValue(t, segment, "`mpress`")
	if strings.Contains(segment.Text, "`mpress`") {
		t.Fatalf("inline code was sent to the provider: %q", segment.Text)
	}
	translated := "À l’exécution, M-Press nécessite uniquement l’exécutable " + placeholder + "."
	output, err := Apply(document, map[string]string{segment.ID: translated})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(output), "À l’exécution, M-Press nécessite uniquement l’exécutable `mpress`.\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
}

func TestExtractMPDHeadingContextIsPlainText(t *testing.T) {
	document, err := ExtractMPD("guide.mpd", []byte("# Install **M-Press** with `go install`\n"))
	if err != nil {
		t.Fatal(err)
	}
	if document.Title != "Install M-Press with go install" || len(document.Outline) != 1 || document.Outline[0] != document.Title {
		t.Fatalf("heading context contains syntax: title=%q outline=%#v", document.Title, document.Outline)
	}
}

func TestExtractMPDFiletreeTranslatesDescriptionsOnly(t *testing.T) {
	source := []byte("@filetree\ndocs/\n  index.mpd  Home page\n  guide/\n    install.mpd  Installation guide\nassets/\n  logo.svg  # Project mark\n@end\n")
	document, err := ExtractMPD("tree.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Home page", "Installation guide", "Project mark"}
	if len(document.Segments) != len(want) {
		t.Fatalf("got segments %#v, want %q", document.Segments, want)
	}
	translations := map[string]string{}
	for index, segment := range document.Segments {
		if segment.Original != want[index] || segment.Kind != "filetree" {
			t.Fatalf("segment %d = %#v, want %q filetree", index, segment, want[index])
		}
		translations[segment.ID] = []string{"Page d’accueil", "Guide d’installation", "Marque du projet"}[index]
	}
	output, err := Apply(document, translations)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, path := range []string{"index.mpd", "guide/", "install.mpd", "logo.svg"} {
		if !strings.Contains(text, path) {
			t.Fatalf("filetree path %q changed:\n%s", path, text)
		}
	}
	for _, translated := range translations {
		if !strings.Contains(text, translated) {
			t.Fatalf("translation %q missing:\n%s", translated, text)
		}
	}
}

func TestExtractMPDExplainedTranslatesAnnotationsOnly(t *testing.T) {
	source := []byte("@explained\n```go\nfunc main() { // (1)\n  build() // (2)\n}\n```\n\n(1) Start with the entry point.\n\n(2) Build the complete\nsite without changing code.\n@end\n")
	document, err := ExtractMPD("explained.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Segments) != 2 {
		t.Fatalf("got segments %#v, want two annotations", document.Segments)
	}
	translations := map[string]string{
		document.Segments[0].ID: "Commencez par le point d’entrée.",
		document.Segments[1].ID: "Générez le site complet sans modifier le code.",
	}
	output, err := Apply(document, translations)
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, unchanged := range []string{"func main() { // (1)", "  build() // (2)", "(1) Commencez", "(2) Générez le site complet sans modifier le code."} {
		if !strings.Contains(text, unchanged) {
			t.Fatalf("expected %q in translated explained component:\n%s", unchanged, text)
		}
	}
}

func placeholderForValue(t *testing.T, segment Segment, value string) string {
	t.Helper()
	for placeholder, original := range segment.Placeholders {
		if original == value {
			return placeholder
		}
	}
	t.Fatalf("no placeholder for %q in %#v", value, segment.Placeholders)
	return ""
}

func TestMPDAttributeTranslationEscapesJSON(t *testing.T) {
	document, err := ExtractMPD("quote.mpd", []byte("@step title=\"Create a site\"\nContent.\n@end\n"))
	if err != nil {
		t.Fatal(err)
	}
	var title Segment
	for _, segment := range document.Segments {
		if segment.Original == "Create a site" {
			title = segment
		}
	}
	if title.ID == "" {
		t.Fatal("step title was not extracted")
	}
	output, err := Apply(document, map[string]string{title.ID: `Créer le site "exemple"`})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), `title="Créer le site \"exemple\""`) {
		t.Fatalf("translated JSON attribute was not escaped:\n%s", output)
	}
	if err := Validate(document, output); err != nil {
		t.Fatal(err)
	}
}

func TestEveryMPDFixtureSurvivesStructuralTranslation(t *testing.T) {
	root := filepath.Join("..", "mpdconformance", "testdata", "corpus")
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".mpd") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			document, err := ExtractMPD(rel, source)
			if err != nil {
				t.Fatal(err)
			}
			translations := make(map[string]string, len(document.Segments))
			for _, segment := range document.Segments {
				if segment.Protected {
					continue
				}
				translations[segment.ID] = segment.Text + " TR"
			}
			output, err := Apply(document, translations)
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(document, output); err != nil {
				t.Fatalf("%v\n--- source ---\n%s\n--- target ---\n%s", err, source, output)
			}
			parsed := mpd.Parse(rel, output)
			if _, err := mpd.Markdown(parsed); err != nil {
				t.Fatalf("translated fixture does not compile: %v", err)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
