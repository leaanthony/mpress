// Package highlight provides the small, dependency-free syntax highlighter
// shared by code frames and terminal components.
package highlight

import (
	stdhtml "html"
	"strings"
)

var keywords = map[string]bool{
	"as": true, "async": true, "await": true, "break": true, "case": true, "catch": true,
	"class": true, "const": true, "continue": true, "default": true, "defer": true, "else": true,
	"export": true, "fallthrough": true, "for": true, "from": true, "func": true, "function": true,
	"go": true, "goto": true, "if": true, "import": true, "in": true, "interface": true, "let": true,
	"map": true, "new": true, "package": true, "range": true, "return": true, "select": true,
	"struct": true, "switch": true, "throw": true, "try": true, "type": true, "var": true, "while": true,
}

var shellKeywords = map[string]bool{
	"case": true, "do": true, "done": true, "elif": true, "else": true, "esac": true,
	"export": true, "fi": true, "for": true, "function": true, "if": true, "in": true,
	"local": true, "return": true, "select": true, "then": true, "time": true, "until": true,
	"while": true,
}

// WriteLine appends highlighted HTML for one source line.
func WriteLine(b *strings.Builder, language string, line string) {
	language = strings.ToLower(language)
	if language == "text" || language == "plaintext" || language == "md" || language == "markdown" {
		b.WriteString(stdhtml.EscapeString(line))
		return
	}
	if isShell(language) {
		writeShellLine(b, line)
		return
	}
	writeCodeLine(b, language, line)
}

func isShell(language string) bool {
	switch language {
	case "bash", "sh", "shell", "zsh", "fish", "powershell", "pwsh", "console", "terminal":
		return true
	default:
		return false
	}
}

func writeShellLine(b *strings.Builder, line string) {
	expectCommand := true
	for i := 0; i < len(line); {
		if line[i] == '#' && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
			writeToken(b, "comment", line[i:])
			break
		}
		if line[i] == '\'' || line[i] == '"' || line[i] == '`' {
			end := quotedEnd(line, i)
			writeToken(b, "string", line[i:end])
			i = end
			expectCommand = false
			continue
		}
		if line[i] == '$' {
			end := shellVariableEnd(line, i)
			writeToken(b, "type", line[i:end])
			i = end
			expectCommand = false
			continue
		}
		if line[i] == '-' && i+1 < len(line) && line[i+1] != ' ' && line[i+1] != '\t' {
			end := i + 2
			for end < len(line) && line[end] != ' ' && line[end] != '\t' && !strings.ContainsRune("|&;", rune(line[end])) {
				end++
			}
			writeToken(b, "keyword", line[i:end])
			i = end
			expectCommand = false
			continue
		}
		if isDigit(line[i]) {
			end := numberEnd(line, i)
			writeToken(b, "number", line[i:end])
			i = end
			expectCommand = false
			continue
		}
		if isIdentifierStart(line[i]) {
			end := i + 1
			for end < len(line) && (isIdentifierPart(line[end]) || strings.ContainsRune("./\\@", rune(line[end]))) {
				end++
			}
			word := line[i:end]
			if shellKeywords[word] {
				writeToken(b, "keyword", word)
			} else if expectCommand {
				writeToken(b, "function", word)
			} else {
				b.WriteString(stdhtml.EscapeString(word))
			}
			i = end
			expectCommand = false
			continue
		}
		if strings.ContainsRune("=:+*%!&|<>~;", rune(line[i])) {
			end := i + 1
			for end < len(line) && strings.ContainsRune("=:+*%!&|<>~;", rune(line[end])) {
				end++
			}
			operator := line[i:end]
			writeToken(b, "operator", operator)
			if strings.Contains(operator, "|") || strings.Contains(operator, "&") || strings.Contains(operator, ";") {
				expectCommand = true
			}
			i = end
			continue
		}
		b.WriteString(stdhtml.EscapeString(line[i : i+1]))
		i++
	}
}

func writeCodeLine(b *strings.Builder, language, line string) {
	for i := 0; i < len(line); {
		if strings.HasPrefix(line[i:], "//") || ((language == "yaml" || language == "yml" || language == "python") && line[i] == '#') {
			writeToken(b, "comment", line[i:])
			break
		}
		if line[i] == '\'' || line[i] == '"' || line[i] == '`' {
			end := quotedEnd(line, i)
			writeToken(b, "string", line[i:end])
			i = end
			continue
		}
		if isDigit(line[i]) {
			end := numberEnd(line, i)
			writeToken(b, "number", line[i:end])
			i = end
			continue
		}
		if isIdentifierStart(line[i]) {
			end := i + 1
			for end < len(line) && isIdentifierPart(line[end]) {
				end++
			}
			word := line[i:end]
			class := ""
			if keywords[word] || word == "true" || word == "false" || word == "nil" || word == "null" || word == "undefined" {
				class = "keyword"
			} else {
				next := end
				for next < len(line) && (line[next] == ' ' || line[next] == '\t') {
					next++
				}
				if next < len(line) && line[next] == '(' {
					class = "function"
				} else if word[0] >= 'A' && word[0] <= 'Z' {
					class = "type"
				}
			}
			if class == "" {
				b.WriteString(stdhtml.EscapeString(word))
			} else {
				writeToken(b, class, word)
			}
			i = end
			continue
		}
		if strings.ContainsRune("=:+-*%!&|<>~", rune(line[i])) {
			end := i + 1
			for end < len(line) && strings.ContainsRune("=:+-*%!&|<>~", rune(line[end])) {
				end++
			}
			writeToken(b, "operator", line[i:end])
			i = end
			continue
		}
		b.WriteString(stdhtml.EscapeString(line[i : i+1]))
		i++
	}
}

func writeToken(b *strings.Builder, class, value string) {
	b.WriteString(`<span class="mpress-token-` + class + `">` + stdhtml.EscapeString(value) + `</span>`)
}

func quotedEnd(line string, start int) int {
	quote := line[start]
	end := start + 1
	for end < len(line) {
		if line[end] == '\\' {
			end += 2
			continue
		}
		end++
		if line[end-1] == quote {
			break
		}
	}
	if end > len(line) {
		return len(line)
	}
	return end
}

func shellVariableEnd(line string, start int) int {
	end := start + 1
	if end < len(line) && line[end] == '{' {
		end++
		for end < len(line) && line[end] != '}' {
			end++
		}
		if end < len(line) {
			end++
		}
		return end
	}
	for end < len(line) && (isIdentifierPart(line[end]) || line[end] == ':') {
		end++
	}
	return end
}

func numberEnd(line string, start int) int {
	end := start + 1
	for end < len(line) && (isDigit(line[end]) || strings.ContainsRune("._xabcdefABCDEF", rune(line[end]))) {
		end++
	}
	return end
}

func isDigit(value byte) bool { return value >= '0' && value <= '9' }
func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}
func isIdentifierPart(value byte) bool {
	return isIdentifierStart(value) || isDigit(value) || value == '-'
}
