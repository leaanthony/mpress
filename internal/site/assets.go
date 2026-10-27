package site

// The production asset minifiers live in this package so M-Press remains a
// single Go binary. The lexical scanners remove comments and redundant
// whitespace; JavaScript also shortens eligible binding names. Scripts using
// eval, with, escaped identifiers or template interpolation skip renaming.

func minifyStylesheet(source string) (string, error) {
	return minifyCSS(source)
}

func minifyScript(source string) (string, error) {
	return minifyJavaScript(source)
}
