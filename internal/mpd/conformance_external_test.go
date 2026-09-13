package mpd_test

import (
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/mpd"
	"github.com/leaanthony/mpress/internal/mpdconformance"
)

func TestParseConformanceCorpus(t *testing.T) {
	corpus, err := mpdconformance.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range corpus.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			document := mpd.Parse(fixture.ID+".mpd", []byte(fixture.MPD))
			if document.Schema != 1 {
				t.Fatalf("schema = %d, want 1", document.Schema)
			}
			for _, diagnostic := range document.Diagnostics {
				if diagnostic.Severity == mpd.SeverityError {
					t.Errorf("unexpected %s at %d:%d: %s", diagnostic.Code, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Message)
				}
			}
			if got := document.Nodes[document.Root].Source.End; got != uint32(len(fixture.MPD)) {
				t.Errorf("root source end = %d, want %d", got, len(fixture.MPD))
			}
		})
	}
}

func TestNativeMPDConformanceCorpusRendersThroughProductionPipeline(t *testing.T) {
	corpus, err := mpdconformance.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	renderer := content.NewRenderer()
	for _, fixture := range corpus.Fixtures {
		fixture := fixture
		t.Run(fixture.ID, func(t *testing.T) {
			document := mpd.Parse(fixture.ID+".mpd", []byte(fixture.MPD))
			markdown, err := mpd.Markdown(document)
			if err != nil {
				t.Fatal(err)
			}
			page, diagnostics, err := renderer.ParseBytes(fixture.ID+".mpd", "en", markdown)
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range diagnostics {
				if diagnostic.Severity == "error" {
					t.Fatalf("render diagnostic %s: %s\n%s", diagnostic.Code, diagnostic.Message, markdown)
				}
			}
			if strings.TrimSpace(fixture.HTML) != "" && strings.TrimSpace(page.HTML) == "" {
				t.Fatalf("native MPD produced empty HTML\nMarkdown adapter output:\n%s", markdown)
			}
			if strings.Contains(page.HTML, "mpress-unsupported") {
				t.Fatalf("native MPD leaked unsupported component markup\nMarkdown adapter output:\n%s\n\nHTML:\n%s", markdown, page.HTML)
			}
		})
	}
}

func TestConformanceCorpusProducesEveryNativeNodeKind(t *testing.T) {
	corpus, err := mpdconformance.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[mpd.Kind]bool)
	for _, fixture := range corpus.Fixtures {
		document := mpd.Parse(fixture.ID+".mpd", []byte(fixture.MPD))
		for _, node := range document.Nodes {
			seen[node.Kind] = true
		}
	}
	for kind := mpd.KindDocument; kind <= mpd.KindInlineRole; kind++ {
		if !seen[kind] {
			t.Errorf("conformance corpus did not produce a %s node", kind)
		}
	}
}
