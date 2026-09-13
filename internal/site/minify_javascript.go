package site

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type jsTokenKind uint8

const (
	jsIdentifier jsTokenKind = iota
	jsNumber
	jsString
	jsTemplate
	jsRegexp
	jsPunctuator
	jsComment
)

type jsGap uint8

const (
	jsNoGap jsGap = iota
	jsSpaceGap
	jsLineGap
)

type jsToken struct {
	kind jsTokenKind
	text string
	gap  jsGap
}

func minifyJavaScript(source string) (string, error) {
	tokens, err := lexJavaScript(source)
	if err != nil {
		return "", err
	}
	tokens = mangleJavaScriptIdentifiers(tokens)
	tokens = removeOptionalJSSemicolons(tokens)
	var output strings.Builder
	output.Grow(len(source))
	for index, token := range tokens {
		if index > 0 {
			previous := tokens[index-1]
			if token.gap == jsLineGap && preserveJSLine(previous, token) {
				output.WriteByte('\n')
			} else if token.gap != jsNoGap && jsNeedsSpace(previous, token) {
				output.WriteByte(' ')
			}
		}
		output.WriteString(token.text)
	}
	return strings.TrimSpace(output.String()), nil
}

func removeOptionalJSSemicolons(tokens []jsToken) []jsToken {
	output := tokens[:0]
	for index, token := range tokens {
		if token.text == ";" {
			next := nextJSToken(tokens, index)
			if next < 0 || tokens[next].text == "}" {
				continue
			}
		}
		output = append(output, token)
	}
	return output
}

func mangleJavaScriptIdentifiers(tokens []jsToken) []jsToken {
	declarationCounts := collectJSDeclarations(tokens)
	declared := make(map[string]bool, len(declarationCounts))
	for name, count := range declarationCounts {
		if count == 1 {
			declared[name] = true
		}
	}
	if len(declared) == 0 {
		return tokens
	}
	for _, token := range tokens {
		if token.kind == jsIdentifier && (token.text == "eval" || token.text == "with") {
			return tokens
		}
	}

	// A shorthand property carries its identifier into the public object key.
	// Leave that binding readable rather than expanding {value} to {value:a}.
	for index, token := range tokens {
		if token.kind == jsIdentifier && declared[token.text] {
			previous := previousJSToken(tokens, index)
			next := nextJSToken(tokens, index)
			labelReference := previous >= 0 && (tokens[previous].text == "break" || tokens[previous].text == "continue")
			labelDeclaration := next >= 0 && tokens[next].text == ":"
			if jsShorthandProperty(tokens, index) || labelReference || labelDeclaration {
				delete(declared, token.text)
			}
		}
	}

	counts := make(map[string]int, len(declared))
	used := make(map[string]bool, len(tokens))
	for index, token := range tokens {
		if token.kind != jsIdentifier {
			continue
		}
		canMangle := jsIdentifierCanMangle(tokens, index)
		if canMangle && (!declared[token.text] || len(token.text) == 1) {
			used[token.text] = true
		}
		if declared[token.text] && canMangle {
			counts[token.text]++
		}
	}
	type candidate struct {
		name  string
		score int
	}
	candidates := make([]candidate, 0, len(counts))
	for name, count := range counts {
		if len(name) <= 1 || count < 1 {
			continue
		}
		candidates = append(candidates, candidate{name: name, score: count * (len(name) - 1)})
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].score != candidates[right].score {
			return candidates[left].score > candidates[right].score
		}
		return candidates[left].name < candidates[right].name
	})

	replacements := make(map[string]string, len(candidates))
	nextName := 0
	for _, item := range candidates {
		var short string
		for {
			short = jsShortName(nextName)
			nextName++
			if !used[short] && !jsReservedWord(short) {
				break
			}
		}
		if len(short) >= len(item.name) {
			continue
		}
		replacements[item.name] = short
		used[short] = true
	}
	if len(replacements) == 0 {
		return tokens
	}
	for index := range tokens {
		if replacement := replacements[tokens[index].text]; replacement != "" && jsIdentifierCanMangle(tokens, index) {
			tokens[index].text = replacement
		}
	}
	return tokens
}

func collectJSDeclarations(tokens []jsToken) map[string]int {
	declared := map[string]int{}
	for index, token := range tokens {
		if token.text == "=>" {
			previous := previousJSToken(tokens, index)
			if previous >= 0 && tokens[previous].kind == jsIdentifier {
				declared[tokens[previous].text]++
			} else if previous >= 0 && tokens[previous].text == ")" {
				if open := matchingJSOpenParen(tokens, previous); open >= 0 {
					collectJSParameters(tokens, open, declared)
				}
			}
			continue
		}
		if token.kind != jsIdentifier {
			continue
		}
		switch token.text {
		case "const", "let", "var":
			if next := nextJSToken(tokens, index); next >= 0 && tokens[next].kind == jsIdentifier {
				declared[tokens[next].text]++
			}
		case "function":
			next := nextJSToken(tokens, index)
			if next >= 0 && tokens[next].kind == jsIdentifier {
				declared[tokens[next].text]++
				next = nextJSToken(tokens, next)
			}
			if next >= 0 && tokens[next].text == "(" {
				collectJSParameters(tokens, next, declared)
			}
		case "catch":
			next := nextJSToken(tokens, index)
			if next >= 0 && tokens[next].text == "(" {
				collectJSParameters(tokens, next, declared)
			}
		}
	}
	// Object literal methods are the only non-function parameter form used by
	// the generated runtime. Recognise `name(args) {}` without treating calls
	// or control statements as declarations.
	for close := range tokens {
		if tokens[close].text != ")" {
			continue
		}
		open := matchingJSOpenParen(tokens, close)
		if open < 1 {
			continue
		}
		name := previousJSToken(tokens, open)
		after := nextJSToken(tokens, close)
		beforeName := previousJSToken(tokens, name)
		if name >= 0 && after >= 0 && beforeName >= 0 && tokens[name].kind == jsIdentifier && tokens[after].text == "{" && (tokens[beforeName].text == "{" || tokens[beforeName].text == ",") {
			collectJSParameters(tokens, open, declared)
		}
	}
	return declared
}

func collectJSParameters(tokens []jsToken, open int, declared map[string]int) {
	depth := 0
	for index := open + 1; index < len(tokens); index++ {
		switch tokens[index].text {
		case "(", "[", "{":
			depth++
		case ")":
			if depth == 0 {
				return
			}
			depth--
		case "]", "}":
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && tokens[index].kind == jsIdentifier {
				previous := previousJSToken(tokens, index)
				if previous == open || previous >= 0 && tokens[previous].text == "," {
					declared[tokens[index].text]++
				}
			}
		}
	}
}

func jsIdentifierCanMangle(tokens []jsToken, index int) bool {
	previous := previousJSToken(tokens, index)
	next := nextJSToken(tokens, index)
	if previous >= 0 && (tokens[previous].text == "." || tokens[previous].text == "?.") {
		return false
	}
	if next >= 0 && tokens[next].text == ":" {
		return false
	}
	if jsObjectMethodName(tokens, index) {
		return false
	}
	return tokens[index].kind == jsIdentifier && !jsReservedWord(tokens[index].text)
}

func jsObjectMethodName(tokens []jsToken, index int) bool {
	previous := previousJSToken(tokens, index)
	open := nextJSToken(tokens, index)
	if previous < 0 || open < 0 || !(tokens[previous].text == "{" || tokens[previous].text == ",") || tokens[open].text != "(" {
		return false
	}
	depth := 0
	for cursor := open; cursor < len(tokens); cursor++ {
		switch tokens[cursor].text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				after := nextJSToken(tokens, cursor)
				return after >= 0 && tokens[after].text == "{"
			}
		}
	}
	return false
}

func jsShorthandProperty(tokens []jsToken, index int) bool {
	previous := previousJSToken(tokens, index)
	next := nextJSToken(tokens, index)
	if previous < 0 || next < 0 || !(tokens[previous].text == "{" || tokens[previous].text == ",") || !(tokens[next].text == "," || tokens[next].text == "}") {
		return false
	}
	return true
}

func matchingJSOpenParen(tokens []jsToken, close int) int {
	depth := 0
	for index := close; index >= 0; index-- {
		switch tokens[index].text {
		case ")":
			depth++
		case "(":
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

func previousJSToken(tokens []jsToken, index int) int {
	for index--; index >= 0; index-- {
		if tokens[index].kind != jsComment {
			return index
		}
	}
	return -1
}

func nextJSToken(tokens []jsToken, index int) int {
	for index++; index < len(tokens); index++ {
		if tokens[index].kind != jsComment {
			return index
		}
	}
	return -1
}

func jsShortName(index int) string {
	const first = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ$_"
	const rest = first + "0123456789"
	name := string(first[index%len(first)])
	index /= len(first)
	for index > 0 {
		index--
		name += string(rest[index%len(rest)])
		index /= len(rest)
	}
	return name
}

func jsReservedWord(value string) bool {
	switch value {
	case "await", "break", "case", "catch", "class", "const", "continue", "debugger", "default", "delete", "do", "else", "enum", "export", "extends", "false", "finally", "for", "function", "if", "implements", "import", "in", "instanceof", "interface", "let", "new", "null", "package", "private", "protected", "public", "return", "static", "super", "switch", "this", "throw", "true", "try", "typeof", "var", "void", "while", "with", "yield":
		return true
	}
	return false
}

func lexJavaScript(source string) ([]jsToken, error) {
	tokens := make([]jsToken, 0, len(source)/4)
	gap := jsNoGap
	for index := 0; index < len(source); {
		if isJSSpace(source[index]) {
			if source[index] == '\n' || source[index] == '\r' {
				gap = jsLineGap
			} else if gap == jsNoGap {
				gap = jsSpaceGap
			}
			index++
			continue
		}
		if index == 0 && strings.HasPrefix(source, "#!") {
			end := strings.IndexByte(source, '\n')
			if end < 0 {
				end = len(source)
			}
			tokens = append(tokens, jsToken{kind: jsComment, text: source[:end]})
			index = end
			gap = jsLineGap
			continue
		}
		if index+1 < len(source) && source[index] == '/' && source[index+1] == '/' {
			end := strings.IndexByte(source[index+2:], '\n')
			if end < 0 {
				break
			}
			index += end + 2
			gap = jsLineGap
			continue
		}
		if index+1 < len(source) && source[index] == '/' && source[index+1] == '*' {
			end := strings.Index(source[index+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("minify JavaScript: unterminated comment at byte %d", index)
			}
			end += index + 4
			comment := source[index:end]
			if strings.HasPrefix(comment, "/*!") || strings.Contains(comment, "@license") || strings.Contains(comment, "@preserve") {
				tokens = append(tokens, jsToken{kind: jsComment, text: comment, gap: gap})
				gap = jsSpaceGap
			} else if strings.ContainsAny(comment, "\r\n") {
				gap = jsLineGap
			} else if gap == jsNoGap {
				gap = jsSpaceGap
			}
			index = end
			continue
		}
		value := source[index]
		if value == '\'' || value == '"' {
			end, err := scanJSString(source, index, value)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, jsToken{kind: jsString, text: source[index:end], gap: gap})
			index, gap = end, jsNoGap
			continue
		}
		if value == '`' {
			end, err := scanJSTemplate(source, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, jsToken{kind: jsTemplate, text: source[index:end], gap: gap})
			index, gap = end, jsNoGap
			continue
		}
		if isJSIdentifierStart(source, index) {
			end := scanJSIdentifier(source, index)
			tokens = append(tokens, jsToken{kind: jsIdentifier, text: source[index:end], gap: gap})
			index, gap = end, jsNoGap
			continue
		}
		if isJSNumberStart(source, index) {
			end := scanJSNumber(source, index)
			tokens = append(tokens, jsToken{kind: jsNumber, text: source[index:end], gap: gap})
			index, gap = end, jsNoGap
			continue
		}
		if value == '/' && regexpCanStart(tokens) {
			end, err := scanJSRegexp(source, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, jsToken{kind: jsRegexp, text: source[index:end], gap: gap})
			index, gap = end, jsNoGap
			continue
		}
		punctuator := scanJSPunctuator(source[index:])
		tokens = append(tokens, jsToken{kind: jsPunctuator, text: punctuator, gap: gap})
		index += len(punctuator)
		gap = jsNoGap
	}
	return tokens, nil
}

func scanJSString(source string, start int, quote byte) (int, error) {
	for index := start + 1; index < len(source); index++ {
		switch source[index] {
		case '\\':
			index++
		case quote:
			return index + 1, nil
		case '\n', '\r':
			return 0, fmt.Errorf("minify JavaScript: unescaped newline in string at byte %d", index)
		}
	}
	return 0, fmt.Errorf("minify JavaScript: unterminated string at byte %d", start)
}

func scanJSTemplate(source string, start int) (int, error) {
	for index := start + 1; index < len(source); index++ {
		if source[index] == '\\' {
			index++
			continue
		}
		if source[index] == '`' {
			return index + 1, nil
		}
		if source[index] == '$' && index+1 < len(source) && source[index+1] == '{' {
			end, err := scanJSTemplateExpression(source, index+2)
			if err != nil {
				return 0, err
			}
			index = end - 1
		}
	}
	return 0, fmt.Errorf("minify JavaScript: unterminated template literal at byte %d", start)
}

func scanJSTemplateExpression(source string, start int) (int, error) {
	depth := 1
	for index := start; index < len(source); index++ {
		switch source[index] {
		case '\'', '"':
			end, err := scanJSString(source, index, source[index])
			if err != nil {
				return 0, err
			}
			index = end - 1
		case '`':
			end, err := scanJSTemplate(source, index)
			if err != nil {
				return 0, err
			}
			index = end - 1
		case '/':
			if index+1 < len(source) && source[index+1] == '/' {
				if end := strings.IndexByte(source[index+2:], '\n'); end >= 0 {
					index += end + 1
				} else {
					return 0, fmt.Errorf("minify JavaScript: unterminated template expression at byte %d", start)
				}
			} else if index+1 < len(source) && source[index+1] == '*' {
				end := strings.Index(source[index+2:], "*/")
				if end < 0 {
					return 0, fmt.Errorf("minify JavaScript: unterminated comment at byte %d", index)
				}
				index += end + 3
			} else if templateRegexpCanStart(source, start, index) {
				end, err := scanJSRegexp(source, index)
				if err != nil {
					return 0, err
				}
				index = end - 1
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("minify JavaScript: unterminated template expression at byte %d", start)
}

func templateRegexpCanStart(source string, start, slash int) bool {
	for index := slash - 1; index >= start; index-- {
		if isJSSpace(source[index]) {
			continue
		}
		return strings.ContainsRune("([{=,:;!?&|+-*%^~<>", rune(source[index]))
	}
	return true
}

func scanJSRegexp(source string, start int) (int, error) {
	inClass := false
	for index := start + 1; index < len(source); index++ {
		switch source[index] {
		case '\\':
			index++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				index++
				for index < len(source) && isASCIIIdentifierPart(source[index]) {
					index++
				}
				return index, nil
			}
		case '\n', '\r':
			return 0, fmt.Errorf("minify JavaScript: unterminated regular expression at byte %d", start)
		}
	}
	return 0, fmt.Errorf("minify JavaScript: unterminated regular expression at byte %d", start)
}

func scanJSIdentifier(source string, start int) int {
	index := start
	for index < len(source) {
		if source[index] == '\\' && index+1 < len(source) {
			index += 2
			continue
		}
		if source[index] < utf8.RuneSelf {
			if !isASCIIIdentifierPart(source[index]) {
				break
			}
			index++
			continue
		}
		_, size := utf8.DecodeRuneInString(source[index:])
		index += size
	}
	return index
}

func scanJSNumber(source string, start int) int {
	index := start
	if source[index] == '.' {
		index++
	}
	for index < len(source) {
		value := source[index]
		if isASCIIDigit(value) || isASCIIHex(value) || value == '_' || value == '.' {
			index++
			continue
		}
		if value == 'e' || value == 'E' || value == 'p' || value == 'P' {
			index++
			if index < len(source) && (source[index] == '+' || source[index] == '-') {
				index++
			}
			continue
		}
		if value == 'x' || value == 'X' || value == 'o' || value == 'O' || value == 'b' || value == 'B' || value == 'n' {
			index++
			continue
		}
		break
	}
	return index
}

func scanJSPunctuator(source string) string {
	for _, width := range []int{4, 3, 2} {
		if len(source) < width {
			continue
		}
		candidate := source[:width]
		if jsPunctuators[candidate] {
			return candidate
		}
	}
	return source[:1]
}

var jsPunctuators = map[string]bool{
	">>>=": true,
	"===":  true, "!==": true, ">>>": true, "**=": true, "&&=": true, "||=": true, "??=": true, "...": true, "<<=": true, ">>=": true,
	"=>": true, "++": true, "--": true, "==": true, "!=": true, "<=": true, ">=": true, "&&": true, "||": true, "??": true,
	"?.": true, "**": true, "<<": true, ">>": true, "+=": true, "-=": true, "*=": true, "/=": true, "%=": true, "&=": true, "|=": true, "^=": true,
}

func regexpCanStart(tokens []jsToken) bool {
	if len(tokens) == 0 {
		return true
	}
	previous := tokens[len(tokens)-1]
	if previous.kind == jsIdentifier {
		switch previous.text {
		case "return", "throw", "case", "delete", "void", "typeof", "new", "in", "of", "yield", "await", "else", "do", "instanceof":
			return true
		}
		return false
	}
	if previous.kind == jsNumber || previous.kind == jsString || previous.kind == jsTemplate || previous.kind == jsRegexp {
		return false
	}
	switch previous.text {
	case ")":
		open := matchingJSOpenParen(tokens, len(tokens)-1)
		if open > 0 {
			before := previousJSToken(tokens, open)
			if before >= 0 {
				switch tokens[before].text {
				case "if", "while", "for", "with", "switch", "catch":
					return true
				}
			}
		}
		return false
	case "]", "}", "++", "--":
		return false
	}
	return true
}

func preserveJSLine(previous, next jsToken) bool {
	if previous.kind == jsComment && strings.HasPrefix(previous.text, "#!") {
		return true
	}
	if previous.kind == jsIdentifier {
		switch previous.text {
		case "return", "throw", "break", "continue", "yield", "async":
			return true
		}
	}
	if previous.text == "++" || previous.text == "--" || next.text == "++" || next.text == "--" {
		return true
	}
	if next.kind == jsIdentifier && jsStartsStatement(next.text) {
		switch previous.kind {
		case jsIdentifier, jsNumber, jsString, jsTemplate, jsRegexp:
			return true
		}
		return previous.text == ")" || previous.text == "]" || previous.text == "}"
	}
	if (jsCanEndExpression(previous) || previous.text == ")" || previous.text == "]" || previous.text == "}") && (jsWordLike(next.kind) || next.kind == jsString || next.kind == jsTemplate || next.kind == jsRegexp) {
		return true
	}
	return false
}

func jsCanEndExpression(token jsToken) bool {
	return token.kind == jsIdentifier || token.kind == jsNumber || token.kind == jsString || token.kind == jsTemplate || token.kind == jsRegexp
}

func jsStartsStatement(value string) bool {
	switch value {
	case "const", "let", "var", "class", "function", "if", "for", "while", "do", "switch", "try", "throw", "return", "break", "continue", "debugger", "with", "import", "export":
		return true
	}
	return false
}

func jsNeedsSpace(previous, next jsToken) bool {
	if previous.kind == jsComment || next.kind == jsComment {
		return true
	}
	if jsWordLike(previous.kind) && jsWordLike(next.kind) {
		return true
	}
	if previous.kind == jsIdentifier && (next.kind == jsString || next.kind == jsTemplate || next.kind == jsRegexp) {
		return true
	}
	if previous.kind == jsNumber && next.text == "." {
		return true
	}
	joined := previous.text + next.text
	if jsPunctuators[joined] || strings.HasPrefix(joined, "//") || strings.HasPrefix(joined, "/*") {
		return true
	}
	return false
}

func jsWordLike(kind jsTokenKind) bool {
	return kind == jsIdentifier || kind == jsNumber
}

func isJSOperator(value string) bool {
	switch value {
	case "=", "+", "-", "*", "/", "%", "**", "==", "===", "!=", "!==", "<", ">", "<=", ">=", "<<", ">>", ">>>", "&", "|", "^", "!", "~", "&&", "||", "??", "?", "=>", "+=", "-=", "*=", "/=", "%=", "**=", "&&=", "||=", "??=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
		return true
	}
	return false
}

func isJSSpace(value byte) bool {
	return value == ' ' || value == '\n' || value == '\r' || value == '\t' || value == '\f' || value == '\v'
}

func isJSIdentifierStart(source string, index int) bool {
	value := source[index]
	return value == '$' || value == '_' || value == '\\' || isASCIIAlpha(value) || value >= utf8.RuneSelf
}

func isJSNumberStart(source string, index int) bool {
	if isASCIIDigit(source[index]) {
		return true
	}
	return source[index] == '.' && index+1 < len(source) && isASCIIDigit(source[index+1])
}

func isASCIIIdentifierPart(value byte) bool {
	return isASCIIAlpha(value) || isASCIIDigit(value) || value == '$' || value == '_'
}

func isASCIIAlpha(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isASCIIHex(value byte) bool {
	return value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}
