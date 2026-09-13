package dev

import (
	"strings"

	"github.com/leaanthony/mpress/internal/content"
)

type editableBlock struct {
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Kind     string `json:"kind"`
	Markdown string `json:"markdown"`
	Text     string `json:"text"`
}

type sourceLine struct {
	start, end int
	text       string
}

func editableBlocks(rel, source string) []editableBlock {
	lines := splitSourceLines(source)
	startLine := frontmatterEnd(lines)
	var blocks []editableBlock
	for i := startLine; i < len(lines); {
		line := strings.TrimSpace(lines[i].text)
		if line == "" || specialBlockStart(line) {
			i++
			continue
		}
		if level, prefix := headingPrefix(lines[i].text); level > 0 {
			start := lines[i].start + prefix
			end := lines[i].start + len(strings.TrimRight(lines[i].text, "\r\n"))
			markdown := source[start:end]
			blocks = append(blocks, editableBlock{Start: start, End: end, Kind: "heading", Markdown: markdown, Text: plainText(rel, strings.Repeat("#", level)+" "+markdown)})
			i++
			continue
		}
		start := lines[i].start
		end := lines[i].end
		j := i + 1
		for j < len(lines) {
			next := strings.TrimSpace(lines[j].text)
			if next == "" || specialBlockStart(next) {
				break
			}
			if level, _ := headingPrefix(lines[j].text); level > 0 {
				break
			}
			end = lines[j].end
			j++
		}
		end = trimLineEnd(source, end)
		markdown := source[start:end]
		text := plainText(rel, markdown)
		if text != "" {
			blocks = append(blocks, editableBlock{Start: start, End: end, Kind: "paragraph", Markdown: markdown, Text: text})
		}
		i = j
	}
	return blocks
}

func splitSourceLines(source string) []sourceLine {
	if source == "" {
		return nil
	}
	var lines []sourceLine
	start := 0
	for start < len(source) {
		next := strings.IndexByte(source[start:], '\n')
		end := len(source)
		if next >= 0 {
			end = start + next + 1
		}
		lines = append(lines, sourceLine{start: start, end: end, text: source[start:end]})
		start = end
	}
	return lines
}

func frontmatterEnd(lines []sourceLine) int {
	if len(lines) == 0 || strings.TrimSpace(lines[0].text) != "---" {
		return 0
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i].text) == "---" {
			return i + 1
		}
	}
	return len(lines)
}

func headingPrefix(line string) (int, int) {
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	level := 0
	for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, 0
	}
	return level, indent + level + 1
}

func specialBlockStart(line string) bool {
	return strings.HasPrefix(line, "@") || strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") ||
		strings.HasPrefix(line, "|") || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") ||
		strings.HasPrefix(line, "+") || (line[0] >= '0' && line[0] <= '9' && strings.Contains(line, ". ")) || strings.HasPrefix(line, "<")
}

func trimLineEnd(source string, end int) int {
	for end > 0 && (source[end-1] == '\n' || source[end-1] == '\r') {
		end--
	}
	return end
}

func plainText(rel, markdown string) string {
	page, _, err := content.NewRenderer().Parse(rel, "en", markdown)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(page.PlainText)
}
