package components

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
)

var (
	nativeOpenRE   = regexp.MustCompile(`^@([a-z][a-z0-9-]*)(?:\[([^\]]*)\])?(?:\{([^}]*)\})?\s*$`)
	nativeButtonRE = regexp.MustCompile(`@button\[([^\]]*)\]\(([^)]+)\)(?:\{([^}]*)\})?`)
	legacyButtonRE = regexp.MustCompile(`:::button\[([^\]]*)\]\(([^)]+)\)(?:\{([^}]*)\})?`)
	nativeImageRE  = regexp.MustCompile(`@image\{([^}]*)\}`)
	nativeVideoRE  = regexp.MustCompile(`@video\{([^}]*)\}`)
	inlineBadgeRE  = regexp.MustCompile(`\{badge(?:\.([a-z0-9-]+))?:([^}]+)\}`)
	nativeMetaRE   = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_-]*)(?:=(?:"((?:\\.|[^"\\])*)"|'((?:\\.|[^'\\])*)'|([^|\s]+)))?`)
)

// processNativeSyntax lowers the canonical @ syntax into the older fenced
// component representation used by the renderer. It runs after fenced code has
// been protected, so examples in documentation remain literal.
func processNativeSyntax(source string) string {
	if !strings.Contains(source, "@") && !strings.Contains(source, ":::button[") && !strings.Contains(source, "{badge") {
		return source
	}
	lines := strings.Split(source, "\n")
	var out strings.Builder
	for i := 0; i < len(lines); i++ {
		trimmedLine := strings.TrimSpace(lines[i])
		var match []string
		if strings.HasPrefix(trimmedLine, "@") {
			match = nativeOpenRE.FindStringSubmatch(trimmedLine)
		}
		if match != nil && isNativeLeafComponent(match[1]) {
			out.WriteString(renderNativeLeaf(lines[i], match))
			if i+1 < len(lines) {
				out.WriteByte('\n')
			}
			continue
		}
		if match == nil || !isNativeBlock(match[1]) {
			out.WriteString(processNativeInline(lines[i]))
			if i+1 < len(lines) {
				out.WriteByte('\n')
			}
			continue
		}
		end := i + 1
		depth := 0
		for ; end < len(lines); end++ {
			trimmed := strings.TrimSpace(lines[end])
			if trimmed != "@end" {
				if nested := nativeOpenRE.FindStringSubmatch(trimmed); nested != nil && isNativeBlock(nested[1]) {
					depth++
				}
				continue
			}
			if depth == 0 {
				break
			}
			depth--
		}
		if end == len(lines) {
			// Compact leaf components such as @button and @image do not need an
			// @end marker. If no matching close exists, give the inline renderer
			// a chance to handle the declaration before preserving it literally.
			out.WriteString(processNativeInline(lines[i]))
			if i+1 < len(lines) {
				out.WriteByte('\n')
			}
			continue
		}

		name, title, rawMeta := match[1], match[2], match[3]
		body := processNativeSyntax(strings.Join(lines[i+1:end], "\n"))
		if isAdmonition(name) {
			if name == "warn" {
				name = "warning"
			}
			additions := map[string]string{"title": title}
			// @note is the generic component and can carry an explicit type.
			// Shorthand components such as @tip derive their type from the
			// component name. Do not replace type="tip" with type="note".
			if name != "note" {
				additions["type"] = name
			}
			rawMeta = mergeNativeMeta(rawMeta, additions)
			name = "note"
		} else if title != "" {
			rawMeta = mergeNativeMeta(rawMeta, map[string]string{"title": title})
		}
		fmt.Fprintf(&out, ":::%s%s\n%s\n:::", name, normalizedMeta(rawMeta), body)
		if end+1 < len(lines) {
			out.WriteByte('\n')
		}
		i = end
	}
	return out.String()
}

// injectPreviewTabSources adds an authored-source panel to every tab in a
// @preview-tabs{source} block. It runs while fenced code is still protected,
// then restores each fence only in the escaped display copy. This keeps source
// examples inert even when they contain live component directives or nested
// code fences.
func injectPreviewTabSources(source string, codeBlocks []protectedCodeBlock) string {
	if !strings.Contains(source, "@preview-tabs") || !strings.Contains(source, "source") {
		return source
	}
	lines := strings.Split(source, "\n")
	var out strings.Builder
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		match := nativeOpenRE.FindStringSubmatch(trimmed)
		if match == nil || match[1] != "preview-tabs" || parseNativeMeta(match[3])["source"] != "true" {
			out.WriteString(lines[i])
			if i+1 < len(lines) {
				out.WriteByte('\n')
			}
			continue
		}

		end, depth := i+1, 0
		for ; end < len(lines); end++ {
			current := strings.TrimSpace(lines[end])
			if current == "@end" {
				if depth == 0 {
					break
				}
				depth--
				continue
			}
			if nested := nativeOpenRE.FindStringSubmatch(current); nested != nil && isNativeBlock(nested[1]) {
				depth++
			}
		}
		if end == len(lines) {
			out.WriteString(lines[i])
			if i+1 < len(lines) {
				out.WriteByte('\n')
			}
			continue
		}

		body := strings.Join(lines[i+1:end], "\n")
		tabs := &Tabs{}
		if err := tabs.Parse(body); err != nil {
			out.WriteString(strings.Join(lines[i:end+1], "\n"))
			if end+1 < len(lines) {
				out.WriteByte('\n')
			}
			i = end
			continue
		}

		out.WriteString(lines[i])
		out.WriteByte('\n')
		for index, section := range tabs.Sections {
			fmt.Fprintf(&out, "[%s]\n%s\n\n", section.Label, section.Content)
			display := restoreCodeBlocks(section.Content, "CODEBLOCK_", codeBlocks)
			escaped := html.EscapeString(strings.TrimSpace(display))
			escaped = strings.ReplaceAll(escaped, "\r\n", "&#10;")
			escaped = strings.ReplaceAll(escaped, "\n", "&#10;")
			out.WriteString(`<div class="mpress-preview-source"><div class="mpress-preview-source-label">Markdown</div><pre><code class="language-markdown">`)
			out.WriteString(escaped)
			out.WriteString("</code></pre></div>")
			if index+1 < len(tabs.Sections) {
				out.WriteByte('\n')
			}
		}
		out.WriteByte('\n')
		out.WriteString(lines[end])
		if end+1 < len(lines) {
			out.WriteByte('\n')
		}
		i = end
	}
	return out.String()
}

func renderNativeLeaf(line string, match []string) string {
	name := match[1]
	if name == "image" || name == "video" {
		return processNativeInline(line)
	}
	factory := Registry[name]
	if factory == nil {
		return line
	}
	meta := parseNativeMeta(match[3])
	component := factory(meta)
	if err := component.Parse(""); err != nil {
		return line
	}
	rendered, err := component.Render()
	if err != nil {
		return line
	}
	return strings.TrimRight(rendered, "\r\n")
}

func isNativeBlock(name string) bool {
	// Leaf components are self-contained declarations. Counting one as a block
	// opener makes it consume the next @end while scanning a parent layout,
	// which leaves the surrounding container unclosed and leaks its directive.
	if isNativeLeafComponent(name) {
		return false
	}
	if Registry[name] != nil {
		return true
	}
	return isAdmonition(name) || name == "warn"
}

func isNativeLeafComponent(name string) bool {
	switch name {
	case "computed", "image", "input", "linkcard", "qr", "video":
		return true
	default:
		return false
	}
}

func processNativeInline(line string) string {
	if strings.Contains(line, "@button[") {
		line = nativeButtonRE.ReplaceAllStringFunc(line, renderButtonMatch)
	}
	if strings.Contains(line, ":::button[") {
		line = legacyButtonRE.ReplaceAllStringFunc(line, renderButtonMatch)
	}
	if strings.Contains(line, "@image{") {
		line = nativeImageRE.ReplaceAllStringFunc(line, func(match string) string {
			meta := parseNativeMeta(nativeImageRE.FindStringSubmatch(match)[1])
			rendered, err := renderThemeImage(meta)
			if err != nil {
				return match
			}
			return rendered
		})
	}
	if strings.Contains(line, "@video{") {
		line = nativeVideoRE.ReplaceAllStringFunc(line, func(match string) string {
			video := &Video{Meta: parseNativeMeta(nativeVideoRE.FindStringSubmatch(match)[1])}
			rendered, err := video.Render()
			if err != nil {
				return match
			}
			return rendered
		})
	}
	if !strings.Contains(line, "{badge") {
		return line
	}
	return inlineBadgeRE.ReplaceAllStringFunc(line, func(match string) string {
		parts := inlineBadgeRE.FindStringSubmatch(match)
		variant := parts[1]
		if variant == "" {
			variant = "default"
		}
		return fmt.Sprintf(`<span class="mpress-badge mpress-badge-%s">%s</span>`, html.EscapeString(variant), html.EscapeString(strings.TrimSpace(parts[2])))
	})
}

func renderButtonMatch(match string) string {
	parts := nativeButtonRE.FindStringSubmatch(match)
	if parts == nil {
		parts = legacyButtonRE.FindStringSubmatch(match)
	}
	meta := parseNativeMeta(parts[3])
	variant := "secondary"
	for _, candidate := range []string{"primary", "secondary", "outline"} {
		if meta[candidate] == "true" {
			variant = candidate
		}
	}
	return renderButton(parts[1], parts[2], variant, meta)
}

func renderButton(label, href, variant string, meta map[string]string) string {
	classes := []string{"mpress-button", "mpress-button-" + variant}
	if size := meta["size"]; size != "" {
		classes = append(classes, "mpress-button-"+safeToken(size))
	}
	label = html.EscapeString(label)
	if icon := strings.TrimSpace(meta["icon"]); icon != "" {
		iconHTML := fmt.Sprintf(`<span class="mpress-button-icon">%s</span>`, lucide(icon, 16))
		if meta["icon-right"] == "true" {
			label += iconHTML
		} else {
			label = iconHTML + label
		}
	}
	if meta["action"] == "accessibility" {
		return fmt.Sprintf(`<button class="%s" type="button" data-open-accessibility>%s</button>`, strings.Join(classes, " "), label)
	}
	return fmt.Sprintf(`<a class="%s" href="%s">%s</a>`, strings.Join(classes, " "), html.EscapeString(href), label)
}

func isAdmonition(name string) bool {
	switch name {
	case "info", "tip", "warning", "warn", "caution", "danger", "important", "bug", "example", "note":
		return true
	default:
		return false
	}
}

func normalizedMeta(raw string) string {
	meta := parseNativeMeta(raw)
	if len(meta) == 0 {
		return ""
	}
	keys := make([]string, 0, len(meta))
	for key := range meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, strings.ReplaceAll(meta[key], `"`, `\"`)))
	}
	return "{" + strings.Join(parts, " ") + "}"
}

func mergeNativeMeta(raw string, additions map[string]string) string {
	meta := parseNativeMeta(raw)
	for key, value := range additions {
		if value != "" {
			meta[key] = value
		}
	}
	var parts []string
	for key, value := range meta {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, key, strings.ReplaceAll(value, `"`, `\"`)))
	}
	return strings.Join(parts, "|")
}

func parseNativeMeta(raw string) map[string]string {
	result := map[string]string{}
	for _, match := range nativeMetaRE.FindAllStringSubmatch(raw, -1) {
		value := "true"
		for _, candidate := range match[2:] {
			if candidate != "" {
				value = unescapeNativeMeta(candidate)
				break
			}
		}
		result[match[1]] = value
	}
	return result
}

func unescapeNativeMeta(value string) string {
	var result strings.Builder
	for index := 0; index < len(value); index++ {
		if value[index] != '\\' || index+1 >= len(value) {
			result.WriteByte(value[index])
			continue
		}
		next := value[index+1]
		if next == '\\' || next == '"' || next == '\'' {
			result.WriteByte(next)
			index++
			continue
		}
		result.WriteByte(value[index])
	}
	return result.String()
}

func safeToken(value string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(value) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			out.WriteRune(r)
		}
	}
	return out.String()
}
