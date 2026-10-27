package components

import (
	"strings"
	"testing"
)

func TestProtectInlineCodeSpansMatchesDelimiterRuns(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "plain", input: "text", want: nil},
		{name: "single", input: "a `@button[X](#)` b", want: []string{"`@button[X](#)`"}},
		{name: "double with inner single", input: "a ``b ` @button[X](#) c`` d", want: []string{"``b ` @button[X](#) c``"}},
		{name: "single with inner double", input: "a `b `` @button[X](#) `` c` d", want: []string{"`b `` @button[X](#) `` c`"}},
		{name: "two spans", input: "``a`` and `b`", want: []string{"``a``", "`b`"}},
		{name: "unmatched run stays literal", input: "one ` @button[X](#)", want: nil},
		{name: "unequal runs do not pair", input: "``@button[X](#)` live", want: nil},
		{name: "unmatched double then single span", input: "`` stray `@button[X](#)`", want: []string{"`@button[X](#)`"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			guard := newInlineCodeGuard(test.input)
			protected := guard.protectDocument(test.input)
			spans := guard.spans
			if len(spans) != len(test.want) {
				t.Fatalf("got %d spans, want %d: %#v", len(spans), len(test.want), spans)
			}
			for index, want := range test.want {
				if spans[index].text != want {
					t.Errorf("span %d = %q, want %q", index, spans[index].text, want)
				}
			}
			if got := guard.restore(protected); got != test.input {
				t.Errorf("restore = %q, want %q", got, test.input)
			}
		})
	}
}

func TestProtectInlineCodeSpansBeforeFenceExtraction(t *testing.T) {
	input := "````md\n@button[Fenced](#)\n````\n\nInline `@button[Literal](#)` here.\n"
	guard := newInlineCodeGuard(input)
	protected := guard.protectDocument(input)
	if len(guard.spans) != 1 || guard.spans[0].text != "`@button[Literal](#)`" {
		t.Fatalf("inline spans = %#v", guard.spans)
	}
	protected, fences := protectFencedCodeBlocks(protected, "CODEBLOCK_")
	if len(fences) != 1 {
		t.Fatalf("expected one fence, got %d", len(fences))
	}
	restored := guard.restore(protected)
	restored = restoreCodeBlocks(restored, "CODEBLOCK_", fences)
	if restored != input {
		t.Fatalf("round trip = %q, want %q", restored, input)
	}
}

func TestProtectInlineCodeSpansUsesSourceSegments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "trimmed edge spaces", input: "` abc `", want: []string{"` abc `"}},
		{name: "all spaces", input: "``  ``", want: []string{"``  ``"}},
		{
			name:  "quote continuation keeps physical prefix",
			input: "> Start `example\n> @button[Literal](#){primary}` ends.\n",
			want:  []string{"`example\n> @button[Literal](#){primary}`"},
		},
		{
			name:  "list continuation keeps physical indent",
			input: "- Start `example\n  @button[Literal](#){primary}` ends.\n",
			want:  []string{"`example\n  @button[Literal](#){primary}`"},
		},
		{name: "fenced content is not an inline span", input: "```\n`@button[X](#)`\n```\n", want: nil},
		{name: "escaped opener", input: "\\` @button[X](#) `", want: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			guard := newInlineCodeGuard(test.input)
			guard.protectDocument(test.input)
			if len(guard.spans) != len(test.want) {
				t.Fatalf("got %d spans, want %d: %#v", len(guard.spans), len(test.want), guard.spans)
			}
			for index, want := range test.want {
				if guard.spans[index].text != want {
					t.Errorf("span %d = %q, want %q", index, guard.spans[index].text, want)
				}
			}
		})
	}
}

func TestInlineCodeKeepsNativeDirectivesLiteral(t *testing.T) {
	input := "- Use `@button[Literal](#){primary}` in a list.\n\n" +
		"> Quote ``@image{src=literal.png}`` and `{badge.primary:Literal}`.\n\n" +
		"```md\n@button[Fenced](#){primary}\n```\n\n" +
		"Click @button[Live](/live/){secondary} or see @image{src=live.png alt=Live image}.\n"
	got, warnings := ProcessWithLanguage(input, "en")
	if len(warnings) != 0 {
		t.Fatalf("warnings: %#v", warnings)
	}
	for _, literal := range []string{
		"`@button[Literal](#){primary}`",
		"``@image{src=literal.png}``",
		"`{badge.primary:Literal}`",
		"```md\n@button[Fenced](#){primary}\n```",
	} {
		if !strings.Contains(got, literal) {
			t.Errorf("literal example %q was rewritten:\n%s", literal, got)
		}
	}
	if count := strings.Count(got, `class="mpress-button`); count != 1 {
		t.Errorf("expected only the adjacent live button, got %d:\n%s", count, got)
	}
	if !strings.Contains(got, `<a class="mpress-button mpress-button-secondary" href="/live/">Live</a>`) {
		t.Errorf("adjacent live button was lost:\n%s", got)
	}
	if count := strings.Count(got, "<img "); count != 1 {
		t.Errorf("expected only the adjacent live image, got %d:\n%s", count, got)
	}
	if strings.Contains(got, "mpress-badge") {
		t.Errorf("literal badge directive was rendered:\n%s", got)
	}
}

func TestUnmatchedBacktickDoesNotHideLiveDirective(t *testing.T) {
	for _, input := range []string{
		"An unmatched ` before @button[Live](/live/){primary} stays live.\n",
		"Unequal `` run before @button[Live](/live/){primary} stays live.\n",
	} {
		got, warnings := ProcessWithLanguage(input, "en")
		if len(warnings) != 0 {
			t.Fatalf("warnings: %#v", warnings)
		}
		if !strings.Contains(got, `<a class="mpress-button mpress-button-primary" href="/live/">Live</a>`) {
			t.Errorf("live directive after unmatched punctuation was hidden:\n%s", got)
		}
	}
}

func TestInlineCodeGuardAvoidsAuthoredPlaceholderText(t *testing.T) {
	input := "Authored \x00MPRESS_INLINE_0\x00 stays; code `@button[Literal](#)` stays.\n"
	guard := newInlineCodeGuard(input)
	if guard.marker == "\x00MPRESS_INLINE_" {
		t.Fatal("guard reused a marker that already occurs in the source")
	}
	protected := guard.protectDocument(input)
	if len(guard.spans) != 1 {
		t.Fatalf("expected one inline span, got %#v", guard.spans)
	}
	if got := guard.restore(protected); got != input {
		t.Fatalf("authored placeholder text changed: %q", got)
	}
}
