package site

import "testing"

func FuzzMinifyCSSDoesNotPanic(f *testing.F) {
	for _, source := range []string{
		`.card { color: red; }`,
		`@media (width > 40rem) { article :not(pre) { margin: calc(100% - 2rem); } }`,
		`:root { --value: "a /* literal */ b"; }`,
	} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = minifyStylesheet(source)
	})
}

func FuzzMinifyJavaScriptDoesNotPanic(f *testing.F) {
	for _, source := range []string{
		`const value = state?.ready ?? false;`,
		`const pattern = /[a-z/]+/gi;`,
		"const message = `outer ${`inner ${value}`}`;",
		"function value() { return\n{ready: true}; }",
	} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = minifyScript(source)
	})
}
