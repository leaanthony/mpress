package mpd

import (
	"strings"
	"testing"
)

func TestCompactButtonsKeepInlineRoleBoundaries(t *testing.T) {
	for _, test := range []struct{ source, want string }{
		{"Use @button[Read](/guide/){secondary}.\n", "Use @button[Read](/guide/){secondary}."},
		{"Use @button[Read].\n", "Use Read."},
		{"Use @button[Read]().\n", "Use Read()."},
		{"Use @button[](/guide/).\n", "Use (/guide/)."},
		{"Use @button[Read](unfinished.\n", "Use Read(unfinished."},
		{"Use `@button[Literal](#){primary}`.\n", "Use `@button[Literal](#){primary}`."},
	} {
		document := Parse("inline.mpd", []byte(test.source))
		got, err := Markdown(document)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), test.want) {
			t.Fatalf("source %q: got %q, want %q", test.source, got, test.want)
		}
	}
}
