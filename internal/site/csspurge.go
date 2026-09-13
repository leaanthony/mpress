package site

import (
	"regexp"
	"strings"
)

var cssSelectorToken = regexp.MustCompile(`[.#]([A-Za-z_][A-Za-z0-9_-]*)`)
var runtimeStringToken = regexp.MustCompile(`["']([A-Za-z_][A-Za-z0-9_-]*)["']`)

// purgeUnusedCSS removes only ordinary rules whose class and ID tokens are
// absent from every generated HTML page and the generated runtime script. It
// intentionally preserves at-rules and rules without class/ID selectors.
func purgeUnusedCSS(source string, generatedHTML []string, runtimeScript string) string {
	used := make(map[string]struct{})
	// Scan each page in place. Concatenating every page into one buffer first
	// copied the entire generated site (tens of megabytes) only to read it once,
	// which dominated the optimize phase; the token set is identical either way.
	for _, html := range generatedHTML {
		addHTMLTokens(used, html)
	}
	return purgeUnusedCSSWithTokens(source, used, runtimeScript)
}

// purgeUnusedCSSWithTokens purges from an already-collected set of HTML class
// and id tokens. The build collects these tokens as pages are generated, so the
// optimize phase no longer re-reads the whole site. The set is copied before the
// runtime script tokens are added so the caller's accumulator is not mutated.
func purgeUnusedCSSWithTokens(source string, htmlTokens map[string]struct{}, runtimeScript string) string {
	used := make(map[string]struct{}, len(htmlTokens))
	for token := range htmlTokens {
		used[token] = struct{}{}
	}
	// The page boot script adds this enhancement marker before the external
	// runtime loads. The marker is emitted inline by the template, so it is not
	// present in runtimeScript and must be retained explicitly during purging.
	addToken(used, "js")
	addRuntimeTokens(used, runtimeScript)
	return purgeCSSBlock(source, used)
}

func addHTMLTokens(used map[string]struct{}, source string) {
	scanHTMLAttribute(source, "class", used, true)
	scanHTMLAttribute(source, "id", used, false)
}

// addToken records a class or id token in the used set. The token is often a
// substring of a rendered page that is only borrowed for the scan (see
// bytesToString), so it is cloned on first insertion. Storing the substring
// directly would pin the entire page allocation for as long as the set lives.
// The lookup uses the borrowed key without allocating; only a genuinely new
// token is copied.
func addToken(used map[string]struct{}, token string) {
	if _, ok := used[token]; !ok {
		used[strings.Clone(token)] = struct{}{}
	}
}

func scanHTMLAttribute(source, name string, used map[string]struct{}, split bool) {
	for offset := 0; offset < len(source); {
		match := strings.Index(source[offset:], name)
		if match < 0 {
			return
		}
		position := offset + match
		if position > 0 && isAttributeNameChar(source[position-1]) {
			offset = position + len(name)
			continue
		}
		cursor := position + len(name)
		for cursor < len(source) && (source[cursor] == ' ' || source[cursor] == '\t' || source[cursor] == '\n' || source[cursor] == '\r') {
			cursor++
		}
		if cursor >= len(source) || source[cursor] != '=' {
			offset = position + len(name)
			continue
		}
		cursor++
		for cursor < len(source) && (source[cursor] == ' ' || source[cursor] == '\t' || source[cursor] == '\n' || source[cursor] == '\r') {
			cursor++
		}
		if cursor >= len(source) || (source[cursor] != '\'' && source[cursor] != '"') {
			offset = position + len(name)
			continue
		}
		quote := source[cursor]
		start := cursor + 1
		end := strings.IndexByte(source[start:], quote)
		if end < 0 {
			return
		}
		value := source[start : start+end]
		if split {
			for _, token := range strings.Fields(value) {
				addToken(used, token)
			}
		} else if value != "" {
			addToken(used, value)
		}
		offset = start + end + 1
	}
}

func isAttributeNameChar(value byte) bool {
	return value == '-' || value == '_' || value >= '0' && value <= '9' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func addRuntimeTokens(used map[string]struct{}, source string) {
	for _, match := range cssSelectorToken.FindAllStringSubmatch(source, -1) {
		addToken(used, match[1])
	}
	for _, match := range runtimeStringToken.FindAllStringSubmatch(source, -1) {
		// Runtime class names are often short state tokens such as active,
		// open, or hidden. Keep every identifier-like string from the generated
		// script rather than trying to guess which strings are CSS classes.
		addToken(used, match[1])
	}
}

func purgeCSSBlock(source string, used map[string]struct{}) string {
	var output strings.Builder
	for position := 0; position < len(source); {
		open, semicolon := nextCSSDelimiter(source, position)
		if semicolon >= 0 && (open < 0 || semicolon < open) {
			output.WriteString(source[position : semicolon+1])
			position = semicolon + 1
			continue
		}
		if open < 0 {
			output.WriteString(source[position:])
			break
		}
		end := matchingCSSBrace(source, open)
		if end < 0 {
			output.WriteString(source[position:])
			break
		}
		header := source[position:open]
		trimmed := strings.TrimSpace(header)
		if strings.HasPrefix(trimmed, "@") {
			if purgeNestedAtRule(trimmed) {
				nested := purgeCSSBlock(source[open+1:end], used)
				if strings.TrimSpace(nested) != "" {
					output.WriteString(header)
					output.WriteByte('{')
					output.WriteString(nested)
					output.WriteByte('}')
				}
			} else {
				output.WriteString(header)
				output.WriteByte('{')
				output.WriteString(source[open+1 : end])
				output.WriteByte('}')
			}
		} else if cssRuleUsed(trimmed, used) {
			output.WriteString(header)
			output.WriteByte('{')
			output.WriteString(source[open+1 : end])
			output.WriteByte('}')
		}
		position = end + 1
	}
	return output.String()
}

func purgeNestedAtRule(header string) bool {
	header = strings.ToLower(strings.TrimSpace(header))
	for _, prefix := range []string{"@media", "@supports", "@container", "@layer", "@scope", "@document"} {
		if strings.HasPrefix(header, prefix) {
			return true
		}
	}
	return false
}

func cssRuleUsed(selector string, used map[string]struct{}) bool {
	// A selector list is one rule. Keep it when any simple selector in the
	// list is used, otherwise a live selector could lose its shared declaration
	// merely because a sibling selector is unused.
	if !strings.ContainsAny(selector, "([:") {
		for _, part := range strings.Split(selector, ",") {
			matches := cssSelectorToken.FindAllStringSubmatch(part, -1)
			if len(matches) == 0 {
				return true
			}
			allUsed := true
			for _, match := range matches {
				if _, ok := used[match[1]]; !ok {
					allUsed = false
					break
				}
			}
			if allUsed {
				return true
			}
		}
		return false
	}
	// Pseudo classes, attributes, and functional selectors can encode state
	// that is not present in static markup. Keep these rules when any token is
	// known rather than attempting to evaluate CSS selector semantics.
	matches := cssSelectorToken.FindAllStringSubmatch(selector, -1)
	if len(matches) == 0 {
		return true
	}
	for _, match := range matches {
		if _, ok := used[match[1]]; ok {
			return true
		}
	}
	return false
}

func nextCSSDelimiter(source string, start int) (open, semicolon int) {
	open, semicolon = -1, -1
	inString := byte(0)
	for i := start; i < len(source); i++ {
		switch {
		case inString != 0:
			if source[i] == '\\' {
				i++
			} else if source[i] == inString {
				inString = 0
			}
		case source[i] == '\'', source[i] == '"':
			inString = source[i]
		case source[i] == '/' && i+1 < len(source) && source[i+1] == '*':
			i += 2
			for i+1 < len(source) && !(source[i] == '*' && source[i+1] == '/') {
				i++
			}
			i++
		case source[i] == '{':
			return i, semicolon
		case source[i] == ';':
			if semicolon < 0 {
				semicolon = i
			}
		}
	}
	return open, semicolon
}

func matchingCSSBrace(source string, open int) int {
	depth := 0
	inString := byte(0)
	for i := open; i < len(source); i++ {
		switch {
		case inString != 0:
			if source[i] == '\\' {
				i++
			} else if source[i] == inString {
				inString = 0
			}
		case source[i] == '\'', source[i] == '"':
			inString = source[i]
		case source[i] == '/' && i+1 < len(source) && source[i+1] == '*':
			i += 2
			for i+1 < len(source) && !(source[i] == '*' && source[i+1] == '/') {
				i++
			}
			i++
		case source[i] == '{':
			depth++
		case source[i] == '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
