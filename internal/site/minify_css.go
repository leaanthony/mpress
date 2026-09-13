package site

import (
	"fmt"
	"strings"
)

type cssTokenKind uint8

const (
	cssWord cssTokenKind = iota
	cssSymbol
	cssString
	cssComment
)

type cssToken struct {
	kind cssTokenKind
	text string
	gap  bool
}

func minifyCSS(source string) (string, error) {
	tokens, err := lexCSS(source)
	if err != nil {
		return "", err
	}
	if len(tokens) == 0 {
		return "", nil
	}

	output := make([]byte, 0, len(source))
	var previous cssToken
	havePrevious := false
	var blocks []bool // true means the block contains nested rules.
	for index, token := range tokens {
		if token.text == "}" && havePrevious && previous.text == ";" {
			if len(output) > 0 && output[len(output)-1] == ';' {
				output = output[:len(output)-1]
			}
			havePrevious = false
		}
		inSelector := len(blocks) == 0 || blocks[len(blocks)-1]
		if havePrevious && token.gap && cssNeedsSpace(previous, token, !inSelector) {
			output = append(output, ' ')
		}
		output = append(output, token.text...)
		if token.text == "{" {
			blocks = append(blocks, cssHeaderContainsRules(tokens, index))
		} else if token.text == "}" {
			if len(blocks) > 0 {
				blocks = blocks[:len(blocks)-1]
			}
		}
		previous = token
		havePrevious = true
	}
	return strings.TrimSpace(string(output)), nil
}

func cssHeaderContainsRules(tokens []cssToken, open int) bool {
	start := open - 1
	for start >= 0 && tokens[start].text != "{" && tokens[start].text != "}" && tokens[start].text != ";" {
		start--
	}
	start++
	for start < open && tokens[start].kind == cssComment {
		start++
	}
	if start+1 >= open || tokens[start].text != "@" {
		return false
	}
	name := tokens[start+1].text
	for _, candidate := range []string{"media", "supports", "container", "layer", "scope", "document", "starting-style", "keyframes", "-webkit-keyframes"} {
		if strings.EqualFold(name, candidate) {
			return true
		}
	}
	return false
}

func lexCSS(source string) ([]cssToken, error) {
	tokens := make([]cssToken, 0, len(source)/4)
	gap := false
	braces, brackets, parentheses := 0, 0, 0
	for index := 0; index < len(source); {
		if isCSSSpace(source[index]) {
			gap = true
			index++
			for index < len(source) && isCSSSpace(source[index]) {
				index++
			}
			continue
		}
		if index+1 < len(source) && source[index] == '/' && source[index+1] == '*' {
			end := strings.Index(source[index+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("minify CSS: unterminated comment at byte %d", index)
			}
			end += index + 4
			if index+2 < len(source) && source[index+2] == '!' {
				tokens = append(tokens, cssToken{kind: cssComment, text: source[index:end], gap: gap})
				gap = false
			} else {
				gap = true
			}
			index = end
			continue
		}
		if source[index] == '\'' || source[index] == '"' {
			end, err := scanCSSString(source, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, cssToken{kind: cssString, text: source[index:end], gap: gap})
			gap = false
			index = end
			continue
		}
		if isCSSSymbol(source[index]) {
			text := source[index : index+1]
			switch source[index] {
			case '{':
				braces++
			case '}':
				braces--
			case '[':
				brackets++
			case ']':
				brackets--
			case '(':
				parentheses++
			case ')':
				parentheses--
			}
			if braces < 0 || brackets < 0 || parentheses < 0 {
				return nil, fmt.Errorf("minify CSS: unexpected %q at byte %d", text, index)
			}
			tokens = append(tokens, cssToken{kind: cssSymbol, text: text, gap: gap})
			gap = false
			index++
			continue
		}
		start := index
		for index < len(source) && !isCSSSpace(source[index]) && !isCSSSymbol(source[index]) && source[index] != '\'' && source[index] != '"' {
			if source[index] == '\\' && index+1 < len(source) {
				index += 2
				continue
			}
			if index+1 < len(source) && source[index] == '/' && source[index+1] == '*' {
				break
			}
			index++
		}
		if start == index {
			index++
			continue
		}
		tokens = append(tokens, cssToken{kind: cssWord, text: source[start:index], gap: gap})
		gap = false
	}
	if braces != 0 || brackets != 0 || parentheses != 0 {
		return nil, fmt.Errorf("minify CSS: unbalanced delimiters (braces %d, brackets %d, parentheses %d)", braces, brackets, parentheses)
	}
	return tokens, nil
}

func scanCSSString(source string, start int) (int, error) {
	quote := source[start]
	for index := start + 1; index < len(source); index++ {
		switch source[index] {
		case '\\':
			index++
		case quote:
			return index + 1, nil
		case '\n', '\r', '\f':
			return 0, fmt.Errorf("minify CSS: unescaped newline in string at byte %d", index)
		}
	}
	return 0, fmt.Errorf("minify CSS: unterminated string at byte %d", start)
}

func cssNeedsSpace(previous, next cssToken, inBlock bool) bool {
	if previous.kind == cssComment || next.kind == cssComment {
		return true
	}
	if previous.kind == cssWord && (next.kind == cssWord || next.kind == cssString) {
		return true
	}
	if previous.kind == cssString && (next.kind == cssWord || next.kind == cssString) {
		return true
	}
	if previous.text == ")" && next.kind == cssWord {
		return true
	}
	if previous.kind == cssWord && next.text == "(" {
		// A gap distinguishes an at-rule prelude from a function token.
		return true
	}
	if next.text == "{" || next.text == "}" || next.text == ";" || next.text == "," || next.text == ")" || next.text == "]" {
		return false
	}
	if previous.text == "{" || previous.text == "}" || previous.text == ";" || previous.text == "," || previous.text == "(" || previous.text == "[" {
		return false
	}
	if inBlock && (previous.text == ":" || next.text == ":" || previous.text == "!" || next.text == "!") {
		return false
	}
	if previous.text == ">" || previous.text == "~" || next.text == ">" || next.text == "~" {
		return false
	}
	return true
}

func isCSSSpace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t' || value == '\f'
}

func isCSSSymbol(value byte) bool {
	// Keep + and - inside word tokens. A standalone operator retains its source
	// gap, which protects calc(), while --custom-properties and negative values
	// no longer acquire an unnecessary leading space.
	switch value {
	case '{', '}', '[', ']', '(', ')', ':', ';', ',', '>', '~', '=', '|', '^', '$', '*', '!', '/', '@', '#', '%', '.':
		return true
	}
	return false
}
