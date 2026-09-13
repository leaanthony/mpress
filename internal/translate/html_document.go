package translate

import (
	"bytes"
	"fmt"
	stdhtml "html"
	"strings"

	"github.com/tidwall/gjson"
	"golang.org/x/net/html"
)

// HTML token offsets let translations change visible text without serializing
// tags, URLs, styles or scripts. Attribute values remain inside their quotes.
func htmlSegments(source []byte, base int, prefix, section string) []Segment {
	tokenizer := html.NewTokenizer(bytes.NewReader(source))
	var result []Segment
	offset, skipped := 0, 0
	for {
		kind := tokenizer.Next()
		raw := append([]byte(nil), tokenizer.Raw()...)
		start := offset
		offset += len(raw)
		if kind == html.ErrorToken {
			break
		}
		switch kind {
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			if nonProseHTML(token.Data) && kind == html.StartTagToken {
				skipped++
			}
			if skipped != 0 {
				continue
			}
			for _, attr := range htmlProseAttributeRanges(raw) {
				previous := len(result)
				result = appendHTMLSegment(result, source, base, start+attr.start, start+attr.end, prefix, section)
				if !attr.quoted && len(result) > previous {
					result[len(result)-1].Encoding = "html-unquoted"
				}
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			if nonProseHTML(token.Data) && skipped > 0 {
				skipped--
			}
		case html.TextToken:
			if skipped == 0 {
				result = appendHTMLSegment(result, source, base, start, offset, prefix, section)
			}
		}
	}
	return result
}

type htmlAttributeRange struct {
	start, end int
	quoted     bool
}

// Scan every attribute value, including those we do not translate, so text
// inside a quoted data or event-handler attribute cannot be mistaken for markup.
func htmlProseAttributeRanges(raw []byte) []htmlAttributeRange {
	var result []htmlAttributeRange
	i := 1
	for i < len(raw) && !htmlAttributeSpace(raw[i]) && raw[i] != '>' {
		i++
	}
	for i < len(raw) {
		for i < len(raw) && (htmlAttributeSpace(raw[i]) || raw[i] == '/') {
			i++
		}
		nameStart := i
		for i < len(raw) && !htmlAttributeSpace(raw[i]) && raw[i] != '=' && raw[i] != '>' && raw[i] != '/' {
			i++
		}
		name := strings.ToLower(string(raw[nameStart:i]))
		for i < len(raw) && htmlAttributeSpace(raw[i]) {
			i++
		}
		if i >= len(raw) || raw[i] == '>' {
			break
		}
		if raw[i] != '=' {
			continue
		}
		i++
		for i < len(raw) && htmlAttributeSpace(raw[i]) {
			i++
		}
		attr, next := htmlAttributeValue(raw, i)
		i = next
		if name == "alt" || name == "title" || name == "aria-label" {
			result = append(result, attr)
		}
	}
	return result
}

func htmlAttributeValue(raw []byte, start int) (htmlAttributeRange, int) {
	attr := htmlAttributeRange{start: start, end: start}
	if start >= len(raw) {
		return attr, start
	}
	quote := raw[start]
	attr.quoted = quote == '\'' || quote == '"'
	if attr.quoted {
		attr.start++
		attr.end++
	}
	for attr.end < len(raw) {
		c := raw[attr.end]
		if attr.quoted && c == quote {
			return attr, attr.end + 1
		}
		if !attr.quoted && (htmlAttributeSpace(c) || c == '>') {
			break
		}
		attr.end++
	}
	return attr, attr.end
}

func htmlAttributeSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f'
}

func nonProseHTML(tag string) bool {
	switch tag {
	case "script", "style", "code", "pre", "textarea":
		return true
	}
	return false
}

func appendHTMLSegment(result []Segment, source []byte, base, start, end int, prefix, section string) []Segment {
	value := stdhtml.UnescapeString(string(source[start:end]))
	if !translatableText(value) {
		return result
	}
	text, placeholders := protect(value)
	return append(result, Segment{ID: fmt.Sprintf("%s-text-%03d", prefix, len(result)+1), Kind: "html", Section: section, Original: value, Text: text, Start: base + start, End: base + end, SourceHash: Hash(value), Placeholders: placeholders, Encoding: "html-text"})
}

func escapeHTMLTranslation(value string) string { return stdhtml.EscapeString(value) }

// Nested hero/banner metadata is JSON, not TOML. gjson retains exact offsets
// so a translated string can be patched without rewriting the surrounding data.
func nestedMetadataSegments(source []byte, base int, prefix string) []Segment {
	var result []Segment
	var visit func(gjson.Result, string, string)
	visit = func(value gjson.Result, path, key string) {
		if value.IsObject() || value.IsArray() {
			value.ForEach(func(k, v gjson.Result) bool {
				name := k.String()
				visit(v, path+"-"+name, name)
				return true
			})
			return
		}
		if value.Type != gjson.String || !(mpdTranslatableMetadata[key] || key == "alt" || key == "label") || !translatableText(value.Str) {
			return
		}
		text, placeholders := protectHTMLMarkup(value.Str)
		result = append(result, Segment{ID: path, Kind: "attribute", Original: value.Str, Text: text, Start: base + value.Index + 1, End: base + value.Index + len(value.Raw) - 1, SourceHash: Hash(value.Str), Placeholders: placeholders, Encoding: "json-string"})
	}
	visit(gjson.ParseBytes(source), prefix, "")
	return result
}

func protectHTMLMarkup(value string) (string, map[string]string) {
	if !strings.Contains(value, "<") {
		return protect(value)
	}
	placeholders := map[string]string{}
	tokenizer := html.NewTokenizer(strings.NewReader(value))
	var output strings.Builder
	for {
		kind := tokenizer.Next()
		raw := string(tokenizer.Raw())
		if kind == html.ErrorToken {
			break
		}
		if kind == html.TextToken {
			output.WriteString(raw)
			continue
		}
		key := fmt.Sprintf("⟪MPRESS_HTML_%d⟫", len(placeholders))
		placeholders[key] = raw
		output.WriteString(key)
	}
	text, plain := protect(output.String())
	for key, value := range plain {
		placeholders[key] = value
	}
	return text, placeholders
}
