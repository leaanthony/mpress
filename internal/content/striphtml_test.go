package content

import (
	stdhtml "html"
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

var referenceTagRE = regexp.MustCompile(`<[^>]+>`)

// referenceStripHTML is the previous regexp-based implementation. stripHTML
// must keep producing identical output, because PlainText feeds the search
// index, blog excerpts, and the parse cache.
func referenceStripHTML(s string) string {
	s = referenceTagRE.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(stdhtml.UnescapeString(s)), " ")
}

func referenceStripTags(s, sep string) string {
	return referenceTagRE.ReplaceAllString(s, sep)
}

func TestStripHTMLMatchesReferenceImplementation(t *testing.T) {
	cases := []string{
		"",
		"plain text with no markup",
		"<p>Hello <strong>world</strong></p>",
		"a &amp; b &lt;tag&gt; &nbsp; c",
		"<p>entity whitespace&#10;newline&nbsp;nbsp</p>",
		"unclosed <tag never ends",
		"empty <> brackets < and > loose",
		"<<b>double open</b>>",
		"a < b > c",
		"<pre><code>x&quot;y&#39;z</code></pre>",
		"tabs\tand\nnewlines  collapse",
		"unicode   　 spaces",
		"trailing space <br> ",
		" <p> leading </p>",
		"<h2 id=\"x\">Heading <em>text</em></h2>body",
	}
	for _, input := range cases {
		if got, want := stripHTML(input), referenceStripHTML(input); got != want {
			t.Errorf("stripHTML(%q) = %q, want %q", input, got, want)
		}
		if got, want := stripTags(input, ""), referenceStripTags(input, ""); got != want {
			t.Errorf("stripTags(%q, \"\") = %q, want %q", input, got, want)
		}
	}

	// Property check across random soups of the characters that drive the
	// scanner's state machine.
	alphabet := []rune{'<', '>', '&', ';', 'a', 'b', ' ', '\n', '\t', '#', '"', 'm', 'p', ';', ' ', '世'}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		var b strings.Builder
		for n := rng.Intn(60); n > 0; n-- {
			b.WriteRune(alphabet[rng.Intn(len(alphabet))])
		}
		input := b.String()
		if got, want := stripHTML(input), referenceStripHTML(input); got != want {
			t.Fatalf("stripHTML(%q) = %q, want %q", input, got, want)
		}
		if got, want := stripTags(input, " "), referenceStripTags(input, " "); got != want {
			t.Fatalf("stripTags(%q) = %q, want %q", input, got, want)
		}
	}
}
