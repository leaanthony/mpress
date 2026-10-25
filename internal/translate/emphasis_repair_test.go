package translate

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leaanthony/mpress/internal/config"
)

type koreanEmphasisProvider struct{ calls int }

func (*koreanEmphasisProvider) Name() string  { return "korean-fixture" }
func (*koreanEmphasisProvider) Model() string { return "fixture" }
func (p *koreanEmphasisProvider) Translate(_ context.Context, request TranslationRequest) (map[string]string, error) {
	p.calls++
	result := map[string]string{}
	for _, segment := range request.Segments {
		tokens := translationPlaceholderPattern.FindAllString(segment.Text, -1)
		if len(tokens) != 4 {
			return nil, fmt.Errorf("expected four bold tokens, got %d", len(tokens))
		}
		result[segment.ID] = tokens[0] + "상태: 소유자 검토용 초안." + tokens[1] + " " + tokens[2] + "(결정 보류)" + tokens[3] + "로 표시된 항목은 확정되지 않았습니다."
	}
	return result, nil
}
func TestTranslationRepairsCJKEmphasisWithoutAnotherProviderCall(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Site.Languages = []string{"en", "ko"}
	cfg.Build.ContentDir = "content"
	if err := os.MkdirAll(filepath.Join(root, "content"), 0755); err != nil {
		t.Fatal(err)
	}
	source := "> **Status: draft for owner review.** Items marked **(decision pending)** are not confirmed.\n"
	if err := os.WriteFile(filepath.Join(root, "content", "policy.md"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	provider := &koreanEmphasisProvider{}
	engine := NewEngine(root, cfg, provider)
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := engine.Run(context.Background(), Options{Language: "ko", File: "policy.md"}); err != nil {
			t.Fatal(err)
		}
	}
	if provider.calls != 1 {
		t.Fatalf("repair made %d provider calls", provider.calls)
	}
	output, err := os.ReadFile(filepath.Join(root, "content", "ko", "policy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "<strong>(결정 보류)</strong>로") {
		t.Fatalf("bold was not repaired: %s", output)
	}
	doc, err := ExtractMarkdown([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, output); err != nil {
		t.Fatal(err)
	}
}

func TestEmphasisRepairDoesNotGuessMissingOrReorderedMarkers(t *testing.T) {
	doc, err := ExtractMarkdown([]byte("**Status.** Items marked **(pending)** are not confirmed.\n"))
	if err != nil {
		t.Fatal(err)
	}
	segment := doc.Segments[0]
	tokens := orderedPlaceholderTokens(segment)
	translated := tokens[0] + "상태." + tokens[1] + " " + tokens[2] + "(결정 보류)" + tokens[3] + "로 표시됩니다."
	for _, value := range []string{
		segment.Text,
		strings.Replace(translated, "상태.", "", 1),
		strings.Replace(translated, tokens[2], "", 1),
		translated + tokens[0],
		strings.ReplaceAll(strings.ReplaceAll(translated, tokens[0], "SWAP"), tokens[1], tokens[0]),
	} {
		adjusted := repairMarkdownEmphasis(markdownTranslationFormat, segment, value)
		if !maps.Equal(adjusted.Placeholders, segment.Placeholders) {
			t.Fatal("repaired valid, missing or ambiguous markers")
		}
	}
	if adjusted := repairMarkdownEmphasis("mpd", segment, translated); !maps.Equal(adjusted.Placeholders, segment.Placeholders) {
		t.Fatal("changed another format")
	}
	adjusted := repairMarkdownEmphasis(markdownTranslationFormat, segment, translated)
	if maps.Equal(adjusted.Placeholders, segment.Placeholders) {
		t.Fatal("fixture did not trigger repair")
	}
	for _, value := range segment.Placeholders {
		if value != "**" {
			t.Fatal("repair mutated source placeholders")
		}
	}
	for _, extra := range []string{" [잘못된 링크](/wrong/)", " `unexpected command`"} {
		altered := *doc
		altered.Segments = append([]Segment(nil), doc.Segments...)
		altered.Segments[0] = repairMarkdownEmphasis(markdownTranslationFormat, segment, translated+extra)
		output, err := Apply(&altered, map[string]string{segment.ID: translated + extra})
		if err == nil && Validate(doc, output) == nil {
			t.Fatalf("repair accepted additional protected content: %s", extra)
		}
	}
}

func TestEmphasisRepairKeepsBoldInlineCode(t *testing.T) {
	doc, err := ExtractMarkdown([]byte("Templates live in **`internal/templates/`**.\n"))
	if err != nil {
		t.Fatal(err)
	}
	segment := doc.Segments[0]
	tokens := orderedPlaceholderTokens(segment)
	value := "템플릿은 " + tokens[0] + tokens[1] + tokens[2] + "에 있습니다."
	adjusted := repairMarkdownEmphasis(markdownTranslationFormat, segment, value)
	changed := *doc
	changed.Segments = []Segment{adjusted}
	output, err := Apply(&changed, map[string]string{segment.ID: value})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(doc, output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "<strong>`internal/templates/`</strong>") {
		t.Fatalf("bold code was not preserved: %s", output)
	}
	missing := strings.Replace(value, tokens[1], "", 1)
	empty := repairMarkdownEmphasis(markdownTranslationFormat, segment, missing)
	if !maps.Equal(empty.Placeholders, segment.Placeholders) {
		t.Fatal("repaired missing code")
	}
}

func TestEmphasisRepairDoesNotFillEmptySpanWithMovedCode(t *testing.T) {
	doc, err := ExtractMarkdown([]byte("**Status** uses `private`.\n"))
	if err != nil {
		t.Fatal(err)
	}
	segment := doc.Segments[0]
	tokens := orderedPlaceholderTokens(segment)
	value := tokens[0] + tokens[2] + tokens[1] + "를 사용합니다."
	adjusted := repairMarkdownEmphasis(markdownTranslationFormat, segment, value)
	if !maps.Equal(adjusted.Placeholders, segment.Placeholders) {
		t.Fatal("repaired an empty prose span using code from outside it")
	}
}
