package site

// The production asset minifiers live in this package so M-Press remains a
// single Go binary. They are lexical minifiers: they remove comments and
// redundant whitespace without renaming identifiers or rewriting expressions.
// This is intentionally more conservative than a compiler-style optimizer.

func minifyStylesheet(source string) (string, error) {
	return minifyCSS(source)
}

func minifyScript(source string) (string, error) {
	return minifyJavaScript(source)
}
