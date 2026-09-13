package site

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

func TestReaderLocalesHaveEveryMessageAndPreservePlaceholders(t *testing.T) {
	placeholder := regexp.MustCompile(`\{\d+\}`)
	english := readerLocales["en"]
	for _, language := range []string{"fr", "de", "pt", "ru", "ja", "ko", "zh-cn", "zh-tw", "id"} {
		t.Run(language, func(t *testing.T) {
			messages := readerLocales[language]
			if len(messages) != len(english) {
				t.Errorf("got %d messages, want %d", len(messages), len(english))
			}
			for key := range english {
				value := messages[key]
				if strings.TrimSpace(value) == "" {
					t.Errorf("missing translation: %q", key)
				}
				want := placeholder.FindAllString(key, -1)
				got := placeholder.FindAllString(value, -1)
				slices.Sort(want)
				slices.Sort(got)
				if !slices.Equal(got, want) {
					t.Errorf("%q placeholders = %v, want %v", key, got, want)
				}
			}
		})
	}
}

func TestReaderLocaleSelection(t *testing.T) {
	for _, test := range []struct{ language, canonical string }{
		{"fr-FR", "fr"}, {"pt_BR", "pt"}, {"zh-CN", "zh-cn"}, {"zh-Hans", "zh-cn"}, {"zh-Hant", "zh-tw"},
	} {
		if got := readerMessage(test.language, "Search"); got != readerMessage(test.canonical, "Search") {
			t.Errorf("%s selected %q", test.language, got)
		}
	}
	if got := readerMessage("unknown", "Custom label"); got != "Custom label" {
		t.Errorf("unknown locale lost custom label: %q", got)
	}
}

func TestLocalizedReaderShellAndFormattedText(t *testing.T) {
	page := &content.Page{Language: "fr", Title: "Guide", HTML: "<p>Bonjour</p>"}
	markup, err := renderPage(templateData{Config: config.Default(), Page: page})
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	for _, want := range []string{`id="mpress-ui-messages"`, `lang="fr"`, `>Rechercher<`, `>Sur cette page<`} {
		if !strings.Contains(got, want) {
			t.Errorf("French reader shell missing %q", want)
		}
	}
	// Substitution must never turn user-provided values into markup.
	r := &pageRenderer{data: &templateData{Page: page}, out: &bytes.Buffer{}}
	r.uiTextf("{0} articles", `<script>alert(1)</script>`)
	if strings.Contains(r.out.String(), "<script>") || !strings.Contains(r.out.String(), "&lt;script&gt;") {
		t.Fatalf("formatted reader message was not escaped: %s", r.out.String())
	}
}
