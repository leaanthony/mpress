package site

import "testing"

func TestMinifyStylesheetPreservesNestedSelectorMeaning(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "nested parent reference keeps the descendant pseudo-class",
			source: `.card { & :not(.disabled) { color: rgb(200, 10, 20); } }`,
			want:   `.card{& :not(.disabled){color:rgb(200,10,20)}}`,
		},
		{
			name:   "nested class keeps the descendant pseudo-class",
			source: `.card { .card :not(.disabled) { color: rgb(200, 10, 20); } }`,
			want:   `.card{.card :not(.disabled){color:rgb(200,10,20)}}`,
		},
		{
			name:   "nested element keeps the descendant pseudo-class",
			source: `.card { span :not(.disabled) { color: rgb(200, 10, 20); } }`,
			want:   `.card{span :not(.disabled){color:rgb(200,10,20)}}`,
		},
		{
			name:   "nested compound selector stays compound",
			source: `.card { &:not(.disabled) { color: rgb(200, 10, 20); } }`,
			want:   `.card{&:not(.disabled){color:rgb(200,10,20)}}`,
		},
		{
			name:   "nested comment boundary before a pseudo-class stays a boundary",
			source: `.card { &/**/:not(.disabled) { color: rgb(200, 10, 20); } }`,
			want:   `.card{&/**/:not(.disabled){color:rgb(200,10,20)}}`,
		},
		{
			name:   "declaration keeps authored property whitespace but drops post-colon whitespace",
			source: `.card { color :  rgb(200, 10, 20); }`,
			want:   `.card{color :rgb(200,10,20)}`,
		},
		{
			name:   "namespace selector is unchanged",
			source: `@namespace svg url(urn:svg); svg|a { color: red; }`,
			want:   `@namespace svg url(urn:svg);svg|a{color:red}`,
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
