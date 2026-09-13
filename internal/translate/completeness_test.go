package translate

import (
	"strings"
	"testing"
)

func TestNavigationAllowsLocalizedPunctuationAndQuotes(t *testing.T) {
	source := []byte("# Navigation\n\n- label: \"File dialogs (native)\"\n  link: /dialogs/\n")
	doc, err := ExtractNavigation(source)
	if err != nil {
		t.Fatal(err)
	}
	segment := doc.Segments[0]
	values := map[string]string{segment.ID: `文件对话框（原生）："打开"`}
	request := TranslationRequest{Format: doc.Format, Segments: []RequestSegment{{ID: segment.ID, Text: segment.Text}}}
	if _, err := validateTranslations(request, values); err != nil {
		t.Fatal(err)
	}
	output, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFile("_nav.yaml", "_nav.yaml", doc, output); err != nil {
		t.Fatal(err)
	}
	target, err := ExtractNavigation(output)
	if err != nil {
		t.Fatal(err)
	}
	if target.Segments[0].Original != values[segment.ID] {
		t.Fatalf("wrong label: %s", output)
	}
	if !strings.Contains(string(output), "link: /dialogs/") {
		t.Fatal("changed destination")
	}
}

func TestMPDNestedMetadataAndHTMLProse(t *testing.T) {
	source := []byte("---\nschema = 1\ntitle = \"Hello\"\nhero = {\"tagline\":\"Build desktop apps\",\"actions\":[{\"text\":\"Get started\",\"link\":\"/start/\"}],\"image\":{\"alt\":\"Wails logo\",\"src\":\"/logo.svg\"}}\nbanner = {\"content\":\"Read <a href=\\\"/v2/\\\">older docs</a> now\"}\n---\n\n@rawHTML\n<style>.x { color: red; }</style>\n<script>const x = 'Do not translate';</script>\n<p title=\"Useful help\">Read &amp; learn <code>wails3 dev</code> today.</p>\n<img src=\"/a.svg\" alt=\"Architecture diagram\">\n@end\n")
	doc, err := ExtractMPD("test.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	values := map[string]string{}
	for _, s := range doc.Segments {
		seen[s.Original] = true
		values[s.ID] = "FR " + s.Text
	}
	for _, want := range []string{"Build desktop apps", "Get started", "Wails logo", "Read <a href=\"/v2/\">older docs</a> now", "Useful help", "Read & learn ", " today.", "Architecture diagram"} {
		if !seen[want] {
			t.Errorf("missing %q: %#v", want, seen)
		}
	}
	output, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, output); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"const x = 'Do not translate';", ".x { color: red; }", "<code>wails3 dev</code>", `"link":"/start/"`, `src="/a.svg"`} {
		if !strings.Contains(string(output), want) {
			t.Errorf("changed protected content %q", want)
		}
	}
}

func TestHTMLPlaceholdersDoNotNest(t *testing.T) {
	value := `Read <a href="/v2/">version 2</a>`
	text, placeholders := protectHTMLMarkup(value)
	segment := Segment{ID: "html", Original: value, Text: text, Placeholders: placeholders}
	for i := 0; i < 30; i++ {
		got, err := restore(segment, text, false)
		if err != nil || got != value {
			t.Fatalf("restore=%q, %v", got, err)
		}
	}
}

func TestD2TranslationPreservesGraphAndCode(t *testing.T) {
	cases := []string{
		"direction: right\nfrontend -> backend: Sends request\nbackend: Go service {shape: rectangle}\n",
		"app: {\n shape: sequence_diagram\n user: User\n service: Go service\n Initialisation: {\n service.\"Prepare the window\"\n user -> service: Start application\n }\n}\n",
	}
	for _, diagram := range cases {
		source := []byte("# Diagram\n\n```d2\n" + diagram + "```\n\n```go\nfmt.Println(\"Hello\")\n```\n")
		doc, err := ExtractMPD("diagram.mpd", source)
		if err != nil {
			t.Fatal(err)
		}
		values := map[string]string{}
		labels := 0
		for _, s := range doc.Segments {
			values[s.ID] = "FR " + s.Text
			if s.D2Key != "" {
				labels++
			}
		}
		if labels < 3 {
			t.Fatalf("extracted only %d labels", labels)
		}
		output, err := Apply(doc, values)
		if err != nil {
			t.Fatal(err)
		}
		if err := Validate(doc, output); err != nil {
			t.Fatalf("%v\n%s", err, output)
		}
		if !strings.Contains(string(output), `fmt.Println("Hello")`) {
			t.Fatal("changed Go code")
		}
		if !strings.Contains(string(output), "FR Go service") {
			t.Fatalf("label not translated: %s", output)
		}
		// A changed connection must still fail the structural check.
		broken := strings.Replace(string(output), "frontend -> backend", "backend -> frontend", 1)
		if broken != string(output) && Validate(doc, []byte(broken)) == nil {
			t.Fatal("accepted changed diagram topology")
		}
	}
}

func TestD2TranslationDoesNotReadImports(t *testing.T) {
	_, err := ExtractMPD("import.mpd", []byte("```d2\nx: @/etc/passwd\n```\n"))
	if err == nil {
		t.Fatal("expected an import to be denied")
	}
}

func TestD2SequenceLabelIdentitySurvivesTranslationAndRefinement(t *testing.T) {
	source := []byte("```d2\napp: {\n shape: sequence_diagram\n frontend: Frontend\n backend: Backend\n Communication: {\n  shape: sequence_diagram\n  frontend.\"Make request\"\n  frontend -> backend.a: JSON\n  backend.a.\"Process request\"\n  backend.a.\"Generate response\"\n  backend.a -> frontend: JSON\n  frontend.\"Process response\"\n }\n}\n```\n")
	doc, err := ExtractMPD("diagram.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, segment := range doc.Segments {
		values[segment.ID] = "FR " + segment.Text
	}
	output, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	target, err := ExtractMPD("diagram.mpd", output)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Segment{}
	for _, segment := range target.Segments {
		byID[segment.ID] = segment
	}
	for _, segment := range doc.Segments {
		got := byID[segment.ID]
		if got.D2Key != segment.D2Key || got.Original != "FR "+segment.Original {
			t.Fatalf("label %s moved: source %s=%q, target %s=%q", segment.ID, segment.D2Key, segment.Original, got.D2Key, got.Original)
		}
	}
	if err := Validate(doc, output); err != nil {
		t.Fatal(err)
	}
	var selected Segment
	for _, segment := range target.Segments {
		if segment.Original == "FR Process request" {
			selected = segment
			break
		}
	}
	if selected.ID == "" {
		t.Fatal("missing request label")
	}
	refined, err := Apply(target, map[string]string{selected.ID: "Traiter la requête"})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := ExtractMPD("diagram.mpd", refined)
	if err != nil {
		t.Fatal(err)
	}
	for _, segment := range updated.Segments {
		want := byID[segment.ID].Original
		if segment.ID == selected.ID {
			want = "Traiter la requête"
		}
		if segment.Original != want {
			t.Fatalf("refinement changed %s: %q, want %q", segment.D2Key, segment.Original, want)
		}
	}
}

func TestDiagramStateUsesGraphIdentityAndRejectsOldOrdinalState(t *testing.T) {
	current := []Segment{{ID: "d0001-label-graph-key", D2Key: "node.label", SourceHash: Hash("Open")}}
	old := map[string]SegmentState{"d0001-label-001": {SourceHash: Hash("Open")}}
	if got := matchSegments(current, old); len(got) != 0 {
		t.Fatalf("reused ambiguous ordinal diagram state: %v", got)
	}
	old[current[0].ID] = SegmentState{SourceHash: Hash("Closed")}
	if got := matchSegments(current, old)[current[0].ID]; got != current[0].ID {
		t.Fatalf("graph identity should be retained for stale labels: %q", got)
	}
}

func TestTranslationStatePrefersExactIdentityForRepeatedProse(t *testing.T) {
	current := []Segment{{ID: "first", SourceHash: Hash("Open")}, {ID: "second", SourceHash: Hash("Open")}}
	old := map[string]SegmentState{"first": {SourceHash: Hash("Open")}, "second": {SourceHash: Hash("Open")}}
	for range 20 {
		got := matchSegments(current, old)
		if got["first"] != "first" || got["second"] != "second" {
			t.Fatalf("swapped repeated labels: %v", got)
		}
	}
}

func TestProtectedQuantitiesKeepDecimalsAndRangesBeforeUnits(t *testing.T) {
	doc, err := ExtractMPD("quantities.mpd", []byte("Startup &lt;0.5s versus 2-3s, with 2.2K downloads and 150MB memory.\n"))
	if err != nil {
		t.Fatal(err)
	}
	segment := doc.Segments[0]
	for _, quantity := range []string{"0.5", "2-3", "2.2", "150"} {
		found := false
		for _, value := range segment.Placeholders {
			if strings.Contains(value, quantity) {
				found = true
			}
		}
		if !found {
			t.Errorf("quantity %s was only partially protected: %v", quantity, segment.Placeholders)
		}
	}
	prepared, err := prepareExisting(segment, "FR "+segment.Original)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restore(segment, prepared, false)
	if err != nil || restored != "FR "+segment.Original {
		t.Fatalf("quantity round trip: %q, %v", restored, err)
	}
}

func TestExistingQuantityCannotMatchPartOfAnotherNumber(t *testing.T) {
	text, placeholders := protect("Wait 2 seconds.")
	segment := Segment{ID: "quantity", Original: "Wait 2 seconds.", Text: text, Placeholders: placeholders}
	for _, target := range []string{"Wait 20ms.", "Wait 1.2s.", "Wait 2.5s.", "Wait v2."} {
		if _, err := prepareExisting(segment, target); err == nil {
			t.Errorf("accepted changed quantity: %s", target)
		}
	}
	if _, err := prepareExisting(segment, "Attendre 2s."); err != nil {
		t.Fatal(err)
	}
}

func TestTranslatedMentionAtStartOfParagraphRemainsText(t *testing.T) {
	doc, err := ExtractMPD("mention.mpd", []byte("Thanks to @lea for this fix.\n"))
	if err != nil {
		t.Fatal(err)
	}
	output, err := Apply(doc, map[string]string{doc.Segments[0].ID: "@lea 님이 수정했습니다."})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, output); err != nil {
		t.Fatalf("mention became a directive: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "\\@lea") {
		t.Fatalf("mention was not escaped: %s", output)
	}
}

func TestMPDReuseDistinguishesLiteralAndFormattingUnderscores(t *testing.T) {
	source, err := ExtractMPD("source.mpd", []byte("Update G_APPLICATION_NON_UNIQUE for Linux.\n"))
	if err != nil {
		t.Fatal(err)
	}
	target, err := ExtractMPD("target.mpd", []byte("Mettre à jour G_APPLICATION_NON_UNIQUE pour Linux.\n"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := prepareExistingSegment(source.Segments[0], target.Segments[0])
	if err != nil {
		t.Fatal(err)
	}
	output, err := Apply(source, map[string]string{source.Segments[0].ID: value})
	if err != nil || string(output) != string(target.Source) {
		t.Fatalf("reuse changed the translation: %s, %v", output, err)
	}
	if err := Validate(source, output); err != nil {
		t.Fatal(err)
	}
}

func TestMPDReuseRejectsChangedProtectedQuantity(t *testing.T) {
	source, err := ExtractMPD("source.mpd", []byte("Wait 2.5s before trying again.\n"))
	if err != nil {
		t.Fatal(err)
	}
	target, err := ExtractMPD("target.mpd", []byte("Attendre 12.5s avant de réessayer.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepareExistingSegment(source.Segments[0], target.Segments[0]); err == nil {
		t.Fatal("accepted a changed quantity")
	}
}

func TestNavigationQuotedMultilineLabels(t *testing.T) {
	source := []byte("- label: \"Getting\n    started\" # preserved\n  link: /start/\n- label: 'User''s\n    guide'\n  link: /guide/\n")
	doc, err := ExtractNavigation(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Segments) != 2 || doc.Segments[0].Original != "Getting started" || doc.Segments[1].Original != "User's guide" {
		t.Fatalf("segments: %#v", doc.Segments)
	}
	values := map[string]string{}
	for _, s := range doc.Segments {
		values[s.ID] = "FR " + s.Text
	}
	out, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFile("_nav.yaml", "_nav.yaml", doc, out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "# preserved") {
		t.Fatal("lost comment")
	}
}

func TestPrepareExistingNumbersDoNotMatchInsertedPlaceholders(t *testing.T) {
	text := "WebKit 6.0 on Ubuntu 22.04, Debian 12, Fedora 39, RHEL 9.x; versions 4.0, 20.04, 11 and 8."
	protected, placeholders := protect(text)
	segment := Segment{ID: "numbers", Original: text, Text: protected, Placeholders: placeholders}
	prepared, err := prepareExisting(segment, "FR "+text)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restore(segment, prepared, false)
	if err != nil || restored != "FR "+text {
		t.Fatalf("got %q, %v", restored, err)
	}
	if _, err := prepareExisting(segment, strings.Replace(text, "9.x", "10.x", 1)); err == nil {
		t.Fatal("accepted a changed version")
	}
}

func TestD2MarkdownLabelsRemainMarkdown(t *testing.T) {
	source := []byte("```d2\nbox: |md\n # Heading\n Read **this** now.\n|\n```\n")
	doc, err := ExtractMPD("diagram.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, s := range doc.Segments {
		values[s.ID] = strings.Replace(s.Text, "Heading", "Titre", 1)
	}
	out, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, out); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Titre") || !strings.Contains(string(out), "**this**") {
		t.Fatalf("lost markdown: %s", out)
	}
}

func TestTranslationProtectsEntitiesAndRejectsControlCharacters(t *testing.T) {
	doc, err := ExtractMPD("page.mpd", []byte("Startup &lt;0.5 seconds &amp; memory &#60;10 MB.\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := doc.Segments[0]
	if strings.Contains(s.Text, "&lt;") || strings.Contains(s.Text, "&amp;") || strings.Contains(s.Text, "&#60;") {
		t.Fatalf("unprotected entity: %s", s.Text)
	}
	values := map[string]string{s.ID: "FR " + s.Text}
	good, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, good); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"\x00", "\x12", "\x1c", "\x1e"} {
		values[s.ID] = "FR " + bad + s.Text
		if _, err := Apply(doc, values); err == nil {
			t.Fatal("wrote control character")
		}
		request := TranslationRequest{Format: "mpd", Segments: []RequestSegment{{ID: s.ID, Text: s.Text}}}
		if _, err := validateTranslations(request, values); err == nil {
			t.Fatal("provider accepted control character")
		}
		if err := Validate(doc, append([]byte(bad), good...)); err == nil {
			t.Fatal("audit accepted control character")
		}
	}
}

func TestHTMLAttributesDoNotTranslateAttributeLikeData(t *testing.T) {
	source := []byte("@rawHTML\n<p data-example=\"title='Keep this' alt='Also keep'\" title='Real title'>Hello</p><img alt=Diagram src=/a.svg>\n@end\n")
	doc, err := ExtractMPD("page.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, s := range doc.Segments {
		values[s.ID] = "FR " + s.Text
	}
	out, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `data-example="title='Keep this' alt='Also keep'"`) || !strings.Contains(string(out), `alt="FR Diagram"`) {
		t.Fatalf("wrong attributes: %s", out)
	}
	if err := Validate(doc, out); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}

func TestD2CodeAndMathLabelsRemainProtected(t *testing.T) {
	diagram := "label: |latex x^2 |\na -> b: |go fmt.Println(1) |\nc: |latex y^3 |\na: Request\n"
	source := []byte("```d2\n" + diagram + "```\n")
	doc, err := ExtractMPD("diagram.mpd", source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, s := range doc.Segments {
		if strings.Contains(s.Original, "fmt.Println") || strings.Contains(s.Original, "^") {
			t.Fatalf("code/math became prose: %#v", s)
		}
		values[s.ID] = "FR " + s.Text
	}
	out, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, out); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}

func TestPrepareExistingDistinguishesVersionNamesFromNumbers(t *testing.T) {
	source := "Wails v2 has reached version 2. More v2 releases follow."
	text, placeholders := protect(source)
	segment := Segment{ID: "version", Original: source, Text: text, Placeholders: placeholders}
	target := "Wails v2 hat Version 2 erreicht. Weitere v2-Releases folgen."
	prepared, err := prepareExisting(segment, target)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := restore(segment, prepared, false)
	if err != nil || restored != target {
		t.Fatalf("got %q, %v", restored, err)
	}
}

func TestUnchangedProseDoesNotCountProtectedImagePaths(t *testing.T) {
	doc, err := ExtractMPD("page.mpd", []byte("![ESP Studio](/assets/showcase-images/esp-studio.png)\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := doc.Segments[0]
	var terms []string
	for _, v := range s.Placeholders {
		terms = append(terms, v)
	}
	severity, found := unchangedProse(s.Kind, s.Original, s.Original, terms...)
	if !found || severity != "warning" {
		t.Fatalf("proper-name label should request review, not count URL prose: %s, %v", severity, found)
	}
}
