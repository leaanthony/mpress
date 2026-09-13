package components

import (
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var legacyFencedCodeRegex = regexp.MustCompile("(?ms)^[ \\t>]*(```|~~~)[^\n]*\n(.*?)\n[ \\t>]*(```|~~~)")

func legacyProtectFences(content, tokenPrefix string) (string, []protectedCodeBlock) {
	var blocks []protectedCodeBlock
	protected := legacyFencedCodeRegex.ReplaceAllStringFunc(content, func(match string) string {
		idx := len(blocks)
		prefix := legacyLeadingContainerPrefix(match)
		blocks = append(blocks, protectedCodeBlock{text: match, prefix: prefix})
		if tokenPrefix == "" {
			return ""
		}
		return prefix + fmt.Sprintf("\x00%s%d\x00", tokenPrefix, idx)
	})
	return protected, blocks
}

func legacyLeadingContainerPrefix(block string) string {
	line := block
	if end := strings.IndexByte(block, '\n'); end >= 0 {
		line = block[:end]
	}
	cursor := 0
	for cursor < len(line) && (line[cursor] == ' ' || line[cursor] == '\t' || line[cursor] == '>') {
		cursor++
	}
	return line[:cursor]
}

func legacyRestoreCodeBlock(content, placeholder string, block protectedCodeBlock) string {
	at := strings.Index(content, placeholder)
	if at < 0 {
		return content
	}
	lineStart := strings.LastIndexByte(content[:at], '\n') + 1
	current := content[lineStart:at]
	if strings.TrimLeft(current, " \t>") != "" {
		return strings.Replace(content, placeholder, block.text, 1)
	}
	return content[:lineStart] + reindentCodeBlock(block, current) + content[at+len(placeholder):]
}

func TestProtectFencedCodeBlocksMatchesLegacyBehaviour(t *testing.T) {
	tests := []string{
		"plain text\n",
		"```go\nbody\n```\n",
		"  ```go\n  body\n  ``` trailing\n",
		"> ```go\n> body\n> ~~~\n",
		"```\n```\n",
		"```\n```\n```\n",
		"~~~toml\n[section]\n```and-more\nremainder\n",
		"before\n```go\n@note\ntext\n@end\n```\nafter\n",
		"```unclosed\nbody\n",
		"```a\nx\n```tail\n~~~b\ny\n~~~\n",
	}
	for _, input := range tests {
		for _, prefix := range []string{"CODEBLOCK_", "TABCODE_", ""} {
			wantText, wantBlocks := legacyProtectFences(input, prefix)
			gotText, gotBlocks := protectFencedCodeBlocks(input, prefix)
			if gotText != wantText || !reflect.DeepEqual(gotBlocks, wantBlocks) {
				t.Fatalf("scanner mismatch for %q with prefix %q\nwant text: %q\n got text: %q\nwant blocks: %#v\n got blocks: %#v", input, prefix, wantText, gotText, wantBlocks, gotBlocks)
			}
		}
	}
}

func TestProtectFencedCodeBlocksGeneratedParity(t *testing.T) {
	rng := rand.New(rand.NewSource(84723))
	parts := []string{
		"text", "", "   ", "> quote", "@note", ":::", "```", "~~~",
		"```go", "~~~toml", "  ```", "> ```", "\t~~~", "```tail", "~~~ extra",
	}
	for trial := 0; trial < 1000; trial++ {
		var input strings.Builder
		lineCount := 1 + rng.Intn(40)
		for line := 0; line < lineCount; line++ {
			input.WriteString(parts[rng.Intn(len(parts))])
			if line+1 < lineCount || rng.Intn(2) == 0 {
				input.WriteByte('\n')
			}
		}
		source := input.String()
		wantText, wantBlocks := legacyProtectFences(source, "CODEBLOCK_")
		gotText, gotBlocks := protectFencedCodeBlocks(source, "CODEBLOCK_")
		if gotText != wantText || !reflect.DeepEqual(gotBlocks, wantBlocks) {
			t.Fatalf("generated scanner mismatch at trial %d\nsource: %q\nwant: %q %#v\ngot:  %q %#v", trial, source, wantText, wantBlocks, gotText, gotBlocks)
		}
	}
}

func TestRestoreCodeBlocksMatchesSequentialRestoration(t *testing.T) {
	input := "  \x00CODEBLOCK_0\x00\ntext\n> \x00CODEBLOCK_1\x00\ninline \x00CODEBLOCK_2\x00 tail"
	blocks := []protectedCodeBlock{
		{text: "```go\none\n```", prefix: ""},
		{text: "> ~~~\n> two\n> ~~~", prefix: "> "},
		{text: "```\nthree\n```", prefix: ""},
	}
	want := input
	for i, block := range blocks {
		want = legacyRestoreCodeBlock(want, fmt.Sprintf("\x00CODEBLOCK_%d\x00", i), block)
	}
	if got := restoreCodeBlocks(input, "CODEBLOCK_", blocks); got != want {
		t.Fatalf("batch restoration mismatch\nwant: %q\n got: %q", want, got)
	}
}
