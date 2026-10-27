package site

import "testing"

func TestMinifyStylesheetPreservesCommentTokenBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "comment between compound class selectors stays a token boundary",
			source: `.card/**/.active { color: red; }`,
			want:   `.card/**/.active{color:red}`,
		},
		{
			name:   "comment before a pseudo-class keeps the compound selector",
			source: `article/**/:hover { color: red; }`,
			want:   `article/**/:hover{color:red}`,
		},
		{
			name:   "real descendant whitespace is still emitted",
			source: `article :hover { color: red; }`,
			want:   `article :hover{color:red}`,
		},
		{
			name:   "whitespace beside a comment keeps descendant meaning",
			source: `article /**/:hover { color: red; }`,
			want:   `article :hover{color:red}`,
		},
		{
			name:   "comment before whitespace keeps descendant meaning",
			source: `article/**/ :hover { color: red; }`,
			want:   `article :hover{color:red}`,
		},
		{
			name:   "comment between identifiers prevents token fusion",
			source: `re/**/d { color: red; }`,
			want:   `re/**/d{color:red}`,
		},
		{
			name:   "comment inside a custom property value stays a boundary",
			source: `:root { --gap: 1rem/**/2rem; }`,
			want:   `:root{--gap:1rem/**/2rem}`,
		},
		{
			name:   "comment beside punctuation needs no separator",
			source: `.card/**/{ color: red; }`,
			want:   `.card{color:red}`,
		},
		{
			name:   "escaped identifier keeps its escape across a comment boundary",
			source: `.icon\+name/**/.active { color: red; }`,
			want:   `.icon\+name/**/.active{color:red}`,
		},
		{
			name:   "nested rule keeps a compound selector",
			source: `@media (min-width: 40rem) { .card/**/.active { color: red; } }`,
			want:   `@media (min-width: 40rem){.card/**/.active{color:red}}`,
		},
		{
			name:   "removed comment before license keeps compound meaning",
			source: `.card/**//*! keep */.active { color: red; }`,
			want:   `.card/**//*! keep */.active{color:red}`,
		},
		{
			name:   "removed comment after license keeps compound meaning",
			source: `.card/*! keep *//**/.active { color: red; }`,
			want:   `.card/*! keep *//**/.active{color:red}`,
		},
		{
			name:   "license comment is preserved",
			source: "/*! keep */\n.card { color: red; }",
			want:   `/*! keep */ .card{color:red}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := minifyStylesheet(tc.source)
			if err != nil {
				t.Fatalf("minify stylesheet: %v", err)
			}
			if got != tc.want {
				t.Fatalf("minified stylesheet changed meaning:\nsource: %s\nwant:   %s\ngot:    %s", tc.source, tc.want, got)
			}
			again, err := minifyStylesheet(got)
			if err != nil {
				t.Fatalf("minify minified stylesheet: %v", err)
			}
			if again != got {
				t.Fatalf("minification is not idempotent:\nfirst:  %s\nsecond: %s", got, again)
			}
		})
	}
}
