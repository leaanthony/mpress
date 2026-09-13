package mpd

import (
	"encoding/json"
	"testing"
)

func TestJSONScannerMatchesStandardLibrary(t *testing.T) {
	values := []string{
		`null`, `true`, `false`, `0`, `-0`, `42`, `-12.5`, `6.02e23`,
		`"text"`, `"escape\\n\\u263a"`, `[]`, `[1, true, {"x": null}]`,
		`{}`, `{"name":"MPress","values":[1,2,3]}`,
		``, `nil`, `True`, `01`, `1.`, `.1`, `1e`, `"unterminated`,
		`[1,]`, `{"x":}`, `{x:1}`, `{"x":1,}`, `{"x":"\\q"}`,
	}
	for _, value := range values {
		start := skipJSONWhitespace([]byte(value), 0)
		end, ok := scanJSONValue([]byte(value), start)
		for end < len(value) && (value[end] == ' ' || value[end] == '\t' || value[end] == '\r' || value[end] == '\n') {
			end++
		}
		got := ok && end == len(value)
		if want := json.Valid([]byte(value)); got != want {
			t.Errorf("scanJSONValue(%q) = %v at %d, want %v", value, got, end, want)
		}
	}
}

func FuzzJSONScannerMatchesStandardLibrary(f *testing.F) {
	for _, value := range []string{`null`, `"hello"`, `[1,{"x":true}]`, `-1.25e+3`, `{"bad":}`} {
		f.Add([]byte(value))
	}
	f.Fuzz(func(t *testing.T, value []byte) {
		start := skipJSONWhitespace(value, 0)
		end, ok := scanJSONValue(value, start)
		for end < len(value) && (value[end] == ' ' || value[end] == '\t' || value[end] == '\r' || value[end] == '\n') {
			end++
		}
		got := ok && end == len(value)
		if want := json.Valid(value); got != want {
			t.Fatalf("custom=%v standard=%v end=%d source=%q", got, want, end, value)
		}
	})
}

func skipJSONWhitespace(value []byte, offset int) int {
	for offset < len(value) {
		switch value[offset] {
		case ' ', '\t', '\r', '\n':
			offset++
		default:
			return offset
		}
	}
	return offset
}
