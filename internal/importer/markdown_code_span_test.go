package importer

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/mpd"
)

const multilineCodeSpanExample = "# Examples\n\n" +
	"> Start `example\n> @button[Literal](#){primary}` ends.\n\n" +
	"- Start `example\n  @button[Literal](#){primary}` ends.\n"

const multilineCodeSpanText = "example @button[Literal](#){primary}"

// codeSpansByContainer returns the text of every code span in document, keyed
// by the innermost blockquote or list item that contains it.
func codeSpansByContainer(document *mpd.Document) map[string][]string {
	spans := make(map[string][]string)
	var walk func(uint32, string)
	walk = func(index uint32, container string) {
		node := &document.Nodes[index]
		switch node.Kind {
		case mpd.KindQuote:
			container = "quote"
		case mpd.KindItem:
			container = "item"
		case mpd.KindCodeSpan:
			spans[container] = append(spans[container], string(document.Text(node.Content)))
			return
		}
		iterator := document.Children(index)
		for {
			child, _, ok := iterator.Next()
			if !ok {
				return
			}
			walk(child, container)
		}
	}
	walk(document.Root, "")
	return spans
}

func parseImportedMPD(t *testing.T, source string) *mpd.Document {
	t.Helper()
	document := mpd.Parse("imported.mpd", []byte(source))
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == mpd.SeverityError {
			t.Fatalf("imported MPD is invalid: %s at line %d: %s\n%s", diagnostic.Code, diagnostic.Position.Line, diagnostic.Message, source)
		}
	}
	return document
}

// A code span may wrap onto another physical line inside a blockquote or list
// item. Converted MPD must keep it as one literal span with the quote/list
// meaning intact instead of escaping each physical line independently.
func TestMarkdownToMPDJoinsMultilineCodeSpansInsideContainers(t *testing.T) {
	native, markdown, html := renderImportedTable(t, multilineCodeSpanExample)

	spans := codeSpansByContainer(parseImportedMPD(t, native))
	if got := len(spans["quote"]); got != 1 {
		t.Errorf("quote has %d code spans, want 1: %v\n%s", got, spans["quote"], native)
	}
	if got := len(spans["item"]); got != 1 {
		t.Errorf("list item has %d code spans, want 1: %v\n%s", got, spans["item"], native)
	}
	for container, texts := range spans {
		for _, text := range texts {
			if text != multilineCodeSpanText {
				t.Errorf("%s code span text = %q, want %q\n%s", container, text, multilineCodeSpanText, native)
			}
		}
	}
	if got := len(spans["quote"]) + len(spans["item"]); got != 2 {
		t.Fatalf("code spans in containers = %d, want 2: %v\n%s", got, spans, native)
	}

	if !strings.Contains(html, "<blockquote>\n<p>Start <code>"+multilineCodeSpanText+"</code> ends.</p>\n</blockquote>") {
		t.Errorf("quote is not rendered as one literal span:\n%s", html)
	}
	if !strings.Contains(html, "<li>Start <code>"+multilineCodeSpanText+"</code> ends.</li>") {
		t.Errorf("list item is not rendered as one literal span:\n%s", html)
	}
	if strings.Contains(html, "mpress-button") {
		t.Errorf("literal component example became live:\n%s", html)
	}

	if got := strings.Count(markdown, "`"+multilineCodeSpanText+"`"); got != 2 {
		t.Errorf("portable Markdown has %d literal spans, want 2:\n%s", got, markdown)
	}
	reimported := parseImportedMPD(t, mustMarkdownToMPD(t, markdown))
	roundTrip := codeSpansByContainer(reimported)
	if len(roundTrip["quote"]) != 1 || roundTrip["quote"][0] != multilineCodeSpanText {
		t.Errorf("quote changed through the round trip: %v\n%s", roundTrip, markdown)
	}
	if len(roundTrip["item"]) != 1 || roundTrip["item"][0] != multilineCodeSpanText {
		t.Errorf("list item changed through the round trip: %v\n%s", roundTrip, markdown)
	}
}

func mustMarkdownToMPD(t *testing.T, source string) string {
	t.Helper()
	converted, err := MarkdownToMPD(source)
	if err != nil {
		t.Fatalf("MarkdownToMPD failed: %v", err)
	}
	return converted
}

func TestMarkdownToMPDKeepsSingleLineAndDoubleBacktickCodeSpans(t *testing.T) {
	source := "# Spans\n\n> Keep `a @button[Literal](#){primary}` literal.\n\n" +
		"- Keep ``a ` b`` literal.\n"
	native, _, html := renderImportedTable(t, source)
	spans := codeSpansByContainer(parseImportedMPD(t, native))
	if len(spans["quote"]) != 1 || spans["quote"][0] != "a @button[Literal](#){primary}" {
		t.Errorf("single backtick span changed: %v\n%s", spans, native)
	}
	if len(spans["item"]) != 1 || spans["item"][0] != "a ` b" {
		t.Errorf("double backtick span changed: %v\n%s", spans, native)
	}
	if strings.Contains(html, "mpress-button") {
		t.Errorf("literal component example became live:\n%s", html)
	}
}

func TestMarkdownToMPDDoesNotJoinUnmatchedBackticks(t *testing.T) {
	source := "# Plain\n\n> Not a span: `open\n> next line stays prose.\n"
	native := mustMarkdownToMPD(t, source)
	if !strings.Contains(native, "> Not a span: \\`open\n> next line stays prose.") {
		t.Errorf("unmatched backtick changed the paragraph shape:\n%s", native)
	}
	if spans := codeSpansByContainer(parseImportedMPD(t, native)); len(spans) != 0 {
		t.Errorf("unmatched backtick became a code span: %v\n%s", spans, native)
	}
}

func TestMarkdownToMPDKeepsFencedMultilineBackticksLiteral(t *testing.T) {
	source := "# Fence\n\n```md\n> Start `example\n> @button[Literal](#){primary}` ends.\n```\n"
	native := mustMarkdownToMPD(t, source)
	if !strings.Contains(native, "> Start `example\n> @button[Literal](#){primary}` ends.\n") {
		t.Errorf("fenced example changed:\n%s", native)
	}
}

func TestMarkdownToMPDKeepsPreservedComponentLinesIntact(t *testing.T) {
	source := "@terminal{title=\"T\"}\nresult=`some\ncommand`\n@end\n"
	native := mustMarkdownToMPD(t, source)
	if !strings.Contains(native, "result=`some\ncommand`\n") {
		t.Errorf("preserved component lines were joined:\n%s", native)
	}
}

func TestMarkdownToMPDJoinsChainedMultilineCodeSpansInProse(t *testing.T) {
	source := "Use `one\ntwo` and `three\nfour` examples.\n"
	native, markdown, html := renderImportedTable(t, source)
	if !strings.Contains(native, "Use `one two` and `three four` examples.") {
		t.Errorf("chained spans were not joined:\n%s", native)
	}
	spans := codeSpansByContainer(parseImportedMPD(t, native))[""]
	if len(spans) != 2 || spans[0] != "one two" || spans[1] != "three four" {
		t.Errorf("chained code spans = %v\n%s", spans, native)
	}
	if !strings.Contains(html, "<p>Use <code>one two</code> and <code>three four</code> examples.</p>") {
		t.Errorf("chained spans did not render literally:\n%s", html)
	}
	roundTrip := codeSpansByContainer(parseImportedMPD(t, mustMarkdownToMPD(t, markdown)))[""]
	if len(roundTrip) != 2 || roundTrip[0] != "one two" || roundTrip[1] != "three four" {
		t.Errorf("chained spans changed through the round trip: %v\n%s", roundTrip, markdown)
	}
}

func mpdCodeSpanText(document *mpd.Document) string {
	var texts []string
	for index := range document.Nodes {
		if document.Nodes[index].Kind == mpd.KindCodeSpan {
			texts = append(texts, string(document.Text(document.Nodes[index].Content)))
		}
	}
	return strings.Join(texts, "\n")
}

func TestMarkdownToMPDDoesNotJoinCodeSpansAcrossConverterBlocks(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantNative string
		payload    string
	}{
		{
			name:       "terminal directive",
			source:     "Before `open\n@terminal{title=\"Build\"}\necho hello\n@end\nclose` after.\n",
			wantNative: "@terminal title=\"Build\"",
			payload:    "echo hello",
		},
		{
			name:       "fence",
			source:     "Before `open\n```sh\necho hi\n```\nclose` after.\n",
			wantNative: "```sh\necho hi\n```",
			payload:    "echo hi",
		},
		{
			name:       "table",
			source:     "Before `open\n| A | B |\n| --- | --- |\n| 1 | 2 |\nclose` after.\n",
			wantNative: "@table header=true",
			payload:    "| --- | --- |",
		},
		{
			name:       "raw HTML",
			source:     "Before `open\n<span>hi</span>\nclose` after.\n",
			wantNative: "@rawHTML",
			payload:    "<span>hi</span>",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			native, _, _ := renderImportedTable(t, test.source)
			if !strings.Contains(native, test.wantNative) {
				t.Errorf("converter-owned block was consumed:\n%s", native)
			}
			if strings.Contains(mpdCodeSpanText(parseImportedMPD(t, native)), test.payload) {
				t.Errorf("code span consumed %q:\n%s", test.payload, native)
			}
		})
	}
}
