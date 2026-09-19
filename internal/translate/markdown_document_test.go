package translate

import (
	"strings"
	"testing"
)

func TestMarkdownBlockTranslations(t *testing.T) {
	source := "Run `wails3 dev` and read the [guide](/guide/).\n"
	target := "Lisez le [guide](/guide/), puis lancez `wails3 dev`.\n"
	for _, test := range []struct {
		name, target string
		bad          bool
	}{
		{"reordered", target, false},
		{"command", strings.ReplaceAll(target, "wails3 dev", "wails3 build"), true},
		{"link", strings.ReplaceAll(target, "/guide/", "/other/"), true},
		{"removed", strings.ReplaceAll(target, "[guide](/guide/)", "guide"), true},
		{"added", target + "\nExtra paragraph.\n", true},
		{"added command", strings.TrimSpace(target) + " `extra command`\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			doc, err := ExtractMarkdown([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
			if len(doc.Segments) != 1 {
				t.Fatalf("segments: %#v", doc.Segments)
			}
			err = Validate(doc, []byte(test.target))
			if (err != nil) != test.bad {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}

func TestMarkdownCJKEmphasisAndInlineCode(t *testing.T) {
	source := []byte("Read **the guide** and use <code>run command</code>.\n")
	target := []byte("<strong>ガイド</strong>を読み、<code>run command</code>を使ってください。\n")
	doc, err := ExtractMarkdown(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(doc, target); err != nil {
		t.Fatal(err)
	}
	targetDoc, err := ExtractMarkdown(target)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for i := range doc.Segments {
		value, err := prepareExistingForFormat(doc.Format, &doc.Segments[i], targetDoc.Segments[i])
		if err != nil {
			t.Fatal(err)
		}
		values[doc.Segments[i].ID] = value
	}
	reused, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if string(reused) != string(target) {
		t.Fatalf("lost target emphasis spelling:\n%s", reused)
	}
	if err = Validate(doc, []byte(strings.ReplaceAll(string(target), "run command", "destroy command"))); err == nil {
		t.Fatal("inline HTML code became translatable")
	}
}

func TestMarkdownURLProtectionAcrossPunctuation(t *testing.T) {
	for _, test := range []struct{ source, target string }{
		{"Read [the guide](https://example.com/guide).", "[ガイド](https://example.com/guide)を読んでください。"},
		{"Server URL (default: http://timestamp.digicert.com)", "URL（デフォルト：http://timestamp.digicert.com）"},
	} {
		doc, err := ExtractMarkdown([]byte(test.source))
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(doc, []byte(test.target)); err != nil {
			t.Fatal(err)
		}
		damaged := strings.ReplaceAll(strings.ReplaceAll(test.target, "example.com", "wrong.example"), "digicert.com", "wrong.example")
		if err = Validate(doc, []byte(damaged)); err == nil {
			t.Fatal("changed URL was accepted")
		}
	}
}

func TestMarkdownMetadataEscapingAndNestedCoverage(t *testing.T) {
	source := []byte("---\ntitle: 'Hello: world' # keep comment\ndescription: |\n  First line.\n  Second line.\nhero: {\"tagline\":\"Start here\",\"actions\":[{\"label\":\"Read guide\",\"link\":\"/guide/\"}]}\n---\n\nBody text.\n")
	doc, err := ExtractMarkdown(source)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, s := range doc.Segments {
		values[s.ID] = "Texte: \"traduit\" " + s.Text
	}
	output, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(doc, output); err != nil {
		t.Fatalf("invalid metadata: %v\n%s", err, output)
	}
	for _, want := range []string{"# keep comment", `"link":"/guide/"`, `"label":"Texte: \"traduit\" Read guide"`} {
		if !strings.Contains(string(output), want) {
			t.Errorf("missing %s\n%s", want, output)
		}
	}
}

func TestMarkdownFragmentRewritesLeaveExamplesAlone(t *testing.T) {
	source := []byte("## Guide\n\nRead [guide](#guide).\n\n`[example](#guide)`\n\n```md\n[example](#guide)\n```\n")
	target := []byte(strings.ReplaceAll(strings.ReplaceAll(string(source), "## Guide", "## Manuel"), "Read [guide]", "Lire [manuel]"))
	patched, err := rewriteMarkdownFragments("page.md", source, target, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(patched), "[manuel](#manuel)") || strings.Count(string(patched), "[example](#guide)") != 2 {
		t.Fatalf("wrong anchors:\n%s", patched)
	}
	normalized, err := rewriteMarkdownFragments("page.md", source, patched, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(normalized) != string(target) {
		t.Fatalf("reverse anchors:\n%s", normalized)
	}
}

func TestMarkdownVisibleContentAndOpaqueCode(t *testing.T) {
	source := "---\ntitle: Hello\n---\n\n@note{title=\"Visible heading\"}\nBody text.\n@end\n\n<p title=\"Tooltip text\">HTML prose</p>\n\n```sh\n@note{title=\"Code heading\"}\n```\n\n@terminal\nrun command\n@end\n\n| Name | Value |\n| --- | --- |\n| Alpha | Bravo |\n"
	doc, err := ExtractMarkdown([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var prose []string
	values := map[string]string{}
	for _, s := range doc.Segments {
		prose = append(prose, s.Original)
		values[s.ID] = "FR " + s.Text
	}
	joined := strings.Join(prose, "\n")
	for _, want := range []string{"Hello", "Visible heading", "Body text.", "Tooltip text", "HTML prose", "Name", "Bravo"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q: %s", want, joined)
		}
	}
	for _, bad := range []string{"Code heading", "run command", "@note", "/p>"} {
		if strings.Contains(joined, bad) {
			t.Errorf("non-prose %q: %s", bad, joined)
		}
	}
	output, err := Apply(doc, values)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(doc, output); err != nil {
		t.Fatalf("output validation: %v\n%s", err, output)
	}
	for _, want := range []string{"@note{title=\"FR Visible heading\"}", "<p title=\"FR Tooltip text\">FR HTML prose</p>", "@terminal\nrun command\n@end"} {
		if !strings.Contains(string(output), want) {
			t.Errorf("output missing %s\n%s", want, output)
		}
	}
}
