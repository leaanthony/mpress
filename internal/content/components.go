package content

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	legacycomponents "github.com/leaanthony/mpress/internal/components"
)

var (
	importRE         = regexp.MustCompile(`(?m)^\s*import\s+.*(?:@astrojs/starlight|starlight-|astro:assets).*$`)
	asideRE          = regexp.MustCompile(`(?s)<Aside(?:\s+([^>]*))?>(.*?)</Aside>`)
	tabsRE           = regexp.MustCompile(`(?s)<Tabs(?:\s+[^>]*)?>(.*?)</Tabs>`)
	tabItemRE        = regexp.MustCompile(`(?s)<TabItem\s+([^>]*)>(.*?)</TabItem>`)
	cardGridRE       = regexp.MustCompile(`(?s)<CardGrid(?:\s+[^>]*)?>(.*?)</CardGrid>`)
	cardRE           = regexp.MustCompile(`(?s)<Card\s+([^>]*)>(.*?)</Card>`)
	linkCardRE       = regexp.MustCompile(`(?s)<LinkCard\s+([^>]*)/?>`)
	stepsRE          = regexp.MustCompile(`(?s)<Steps(?:\s+[^>]*)?>(.*?)</Steps>`)
	fileTreeRE       = regexp.MustCompile(`(?s)<FileTree(?:\s+[^>]*)?>(.*?)</FileTree>`)
	badgeRE          = regexp.MustCompile(`<Badge\s+([^>]*)/?>`)
	imageRE          = regexp.MustCompile(`<Image\s+([^>]*)/?>`)
	unsupportedRE    = regexp.MustCompile(`<\/?([A-Z][A-Za-z0-9]*)\b[^>]*>`)
	inlineCodeRE     = regexp.MustCompile("`+[^`\\n]+`+")
	nativeAsideRE    = regexp.MustCompile(`(?ms)^:::(note|tip|warning|caution|danger|info)(?:\[([^\]]+)\])?(?:\{([^\n}]*)\})?(?:[ \t]+([^\n{]+))?\n(.*?)^:::[ \t]*$`)
	nativeTabsRE     = regexp.MustCompile(`(?ms)^:::tabs[^\n]*\n(.*?)^(:::endtabs|:::)[ \t]*$`)
	nativeTabLabelRE = regexp.MustCompile(`(?m)^\[([^\]]+)\]\s*$`)
	nativeCardsRE    = regexp.MustCompile(`(?ms)^:::cards[^\n]*\n(.*?)^:::[ \t]*$`)
	nativeFileTreeRE = regexp.MustCompile(`(?ms)^:::filetree[^\n]*\n(.*?)^:::[ \t]*$`)
	nativeStepsRE    = regexp.MustCompile(`(?ms)^:::steps[^\n]*\n(.*?)^:::[ \t]*$`)
	attrRE           = regexp.MustCompile(`([A-Za-z_:][A-Za-z0-9_:-]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|\{?"([^"]*)"\}?)`)
	componentLineRE  = regexp.MustCompile(`(?m)^[ \t]*(?:@[A-Za-z][A-Za-z0-9-]*(?:\[|\{|[ \t]*$)|:::)`)
	componentHTMLRE  = regexp.MustCompile(`<[A-Z][A-Za-z0-9]*\b`)
	inlineNativeRE   = regexp.MustCompile(`(?:@|:::)button\[[^\]]*\]\([^)]+\)(?:\{[^}]*\})?|@image\{[^}]*\}`)
	nativeNameRE     = regexp.MustCompile(`^@([A-Za-z][A-Za-z0-9-]*)`)
)

func (r *Renderer) processComponents(file, language, source string) (string, []Diagnostic) {
	// Most documentation pages contain ordinary Markdown only. Avoid the
	// component protection and regex pipeline when there is no syntax it can
	// transform. This also avoids allocating a second copy of the source.
	if !hasComponentSyntax(source) {
		return source, nil
	}
	// Forms own their complete Markdown region. Expand them before the legacy
	// component pass so ordinary headings, prose, and lists remain part of the
	// form while field-shaped lines acquire controls.
	formDiagnostics := []Diagnostic(nil)
	if hasFormBlockSyntax(source) {
		source, formDiagnostics = r.processFormBlocks(file, language, source)
	}
	// The legacy pass protects fences itself, because components such as
	// explained own the fence they wrap and have to read it.
	legacySource, warnings := legacycomponents.ProcessWithLanguage(source, language)
	diagnostics := make([]Diagnostic, 0, len(warnings)+len(formDiagnostics))
	diagnostics = append(diagnostics, formDiagnostics...)
	for _, warning := range warnings {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "invalid-component", File: file, Message: warning})
	}
	source = legacySource
	protected, blocks := protectFences(source)
	protected = importRE.ReplaceAllStringFunc(protected, func(s string) string { return "" })
	// Starlight pages commonly contain only an MDX import plus ordinary
	// Markdown. Once imports are removed there is no reason to run every
	// component regexp over those pages.
	if !hasComponentSyntax(protected) {
		return restoreFences(protected, blocks), diagnostics
	}
	protected = nativeTabsRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := nativeTabsRE.FindStringSubmatch(match)
		locs := nativeTabLabelRE.FindAllStringSubmatchIndex(m[1], -1)
		if len(locs) == 0 {
			diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "invalid-tabs", File: file, Message: "tabs block contains no [Label] sections"})
			return html.EscapeString(match)
		}
		var labels, bodies []string
		for i, loc := range locs {
			labels = append(labels, m[1][loc[2]:loc[3]])
			start := loc[1]
			end := len(m[1])
			if i+1 < len(locs) {
				end = locs[i+1][0]
			}
			bodies = append(bodies, m[1][start:end])
		}
		var b strings.Builder
		b.WriteString(`<div class="mpress-tabs"><div role="tablist">`)
		for i, label := range labels {
			fmt.Fprintf(&b, `<button type="button" role="tab" aria-selected="%t">%s</button>`, i == 0, html.EscapeString(label))
		}
		b.WriteString(`</div>`)
		for i, body := range bodies {
			hidden := " hidden"
			if i == 0 {
				hidden = ""
			}
			fmt.Fprintf(&b, `<section role="tabpanel"%s>%s</section>`, hidden, r.fragmentProtected(body, blocks))
		}
		b.WriteString("</div>\n\n")
		return b.String()
	})
	protected = nativeCardsRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := nativeCardsRE.FindStringSubmatch(match)
		parts := strings.Split(m[1], "\n---\n")
		var b strings.Builder
		b.WriteString(`<div class="mpress-card-grid">`)
		for _, part := range parts {
			lines := strings.Split(strings.TrimSpace(part), "\n")
			if len(lines) == 0 {
				continue
			}
			title := strings.TrimSpace(strings.TrimLeft(lines[0], "#* "))
			fmt.Fprintf(&b, `<article class="mpress-card"><h3>%s</h3>%s</article>`, html.EscapeString(title), r.fragmentProtected(strings.Join(lines[1:], "\n"), blocks))
		}
		b.WriteString(`</div>`)
		return b.String()
	})
	protected = nativeFileTreeRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := nativeFileTreeRE.FindStringSubmatch(match)
		return `<div class="mpress-file-tree">` + r.fragmentProtected(m[1], blocks) + "</div>\n\n"
	})
	protected = nativeStepsRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := nativeStepsRE.FindStringSubmatch(match)
		return `<div class="mpress-steps">` + r.fragmentProtected(m[1], blocks) + "</div>\n\n"
	})
	protected = asideRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := asideRE.FindStringSubmatch(match)
		attrs := attributes(m[1])
		kind := attrs["type"]
		if kind == "" {
			kind = "note"
		}
		title := attrs["title"]
		if title == "" {
			title = strings.Title(kind)
		}
		return fmt.Sprintf("<aside class=\"mpress-admonition %s\"><strong>%s</strong>%s</aside>", html.EscapeString(kind), html.EscapeString(title), r.fragmentProtected(m[2], blocks))
	})
	protected = nativeAsideRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := nativeAsideRE.FindStringSubmatch(match)
		attrs := attributes(m[3])
		kind := m[1]
		if attrs["type"] != "" {
			kind = attrs["type"]
		}
		title := strings.TrimSpace(m[2])
		if title == "" {
			title = strings.TrimSpace(m[4])
		}
		if attrs["title"] != "" {
			title = attrs["title"]
		}
		if title == "" {
			title = strings.Title(kind)
		}
		return fmt.Sprintf("<aside class=\"mpress-admonition %s\"><strong>%s</strong>%s</aside>", kind, html.EscapeString(title), r.fragmentProtected(m[5], blocks))
	})
	protected = tabsRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := tabsRE.FindStringSubmatch(match)
		items := tabItemRE.FindAllStringSubmatch(m[1], -1)
		if len(items) == 0 {
			diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "invalid-tabs", File: file, Message: "Tabs contains no TabItem children"})
			return html.EscapeString(match)
		}
		var b strings.Builder
		b.WriteString(`<div class="mpress-tabs"><div role="tablist">`)
		for i, item := range items {
			a := attributes(item[1])
			label := a["label"]
			if label == "" {
				label = a["title"]
			}
			fmt.Fprintf(&b, `<button type="button" role="tab" aria-selected="%t">%s</button>`, i == 0, html.EscapeString(label))
		}
		b.WriteString(`</div>`)
		for i, item := range items {
			fmt.Fprintf(&b, `<section role="tabpanel"%s>%s</section>`, map[bool]string{true: "", false: " hidden"}[i == 0], r.fragmentProtected(item[2], blocks))
		}
		b.WriteString("</div>\n\n")
		return b.String()
	})
	protected = cardGridRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := cardGridRE.FindStringSubmatch(match)
		body := cardRE.ReplaceAllStringFunc(m[1], func(card string) string {
			c := cardRE.FindStringSubmatch(card)
			a := attributes(c[1])
			return fmt.Sprintf(`<article class="mpress-card"><h3>%s</h3>%s</article>`, html.EscapeString(a["title"]), r.fragmentProtected(c[2], blocks))
		})
		body = linkCardRE.ReplaceAllStringFunc(body, linkCard)
		return `<div class="mpress-card-grid">` + body + "</div>\n\n"
	})
	protected = linkCardRE.ReplaceAllStringFunc(protected, linkCard)
	protected = stepsRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := stepsRE.FindStringSubmatch(match)
		return `<div class="mpress-steps">` + r.fragmentProtected(m[1], blocks) + "</div>\n\n"
	})
	protected = fileTreeRE.ReplaceAllStringFunc(protected, func(match string) string {
		m := fileTreeRE.FindStringSubmatch(match)
		return `<div class="mpress-file-tree">` + r.fragmentProtected(m[1], blocks) + "</div>\n\n"
	})
	protected = badgeRE.ReplaceAllStringFunc(protected, func(match string) string {
		a := attributes(badgeRE.FindStringSubmatch(match)[1])
		text := a["text"]
		if text == "" {
			text = a["label"]
		}
		return `<span class="mpress-badge">` + html.EscapeString(text) + `</span>`
	})
	protected = imageRE.ReplaceAllStringFunc(protected, func(match string) string {
		a := attributes(imageRE.FindStringSubmatch(match)[1])
		return fmt.Sprintf(`<img src="%s" alt="%s" loading="lazy">`, html.EscapeString(a["src"]), html.EscapeString(a["alt"]))
	})
	for _, m := range unsupportedRE.FindAllStringSubmatch(protected, -1) {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "unsupported-component", File: file, Message: fmt.Sprintf("unsupported MDX component <%s>", m[1])})
	}
	protected = unsupportedRE.ReplaceAllStringFunc(protected, func(s string) string { return `<code class="mpress-unsupported">` + html.EscapeString(s) + `</code>` })
	for _, directive := range unparsedComponentDirectives(protected) {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "unparsed-component",
			File:     file,
			Message:  fmt.Sprintf("component directive was not parsed: %s", strings.TrimSpace(directive)),
		})
	}
	return restoreFences(protected, blocks), diagnostics
}

func unparsedComponentDirectives(source string) []string {
	var directives []string
	for _, line := range strings.Split(source, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "@end" {
			directives = append(directives, trimmed)
			continue
		}
		if strings.HasPrefix(trimmed, ":::") {
			directives = append(directives, trimmed)
			continue
		}
		match := nativeNameRE.FindStringSubmatch(trimmed)
		if match == nil {
			continue
		}
		name := match[1]
		if legacycomponents.Registry[name] != nil || name == "info" || name == "tip" || name == "warning" || name == "warn" || name == "caution" || name == "danger" || name == "important" || name == "bug" || name == "example" {
			directives = append(directives, trimmed)
		}
	}
	return directives
}

func (r *Renderer) fragment(markdown string) string {
	var b bytes.Buffer
	_ = r.md.Convert(stringToBytes(strings.TrimSpace(markdown)), &b)
	return b.String()
}

func (r *Renderer) fragmentProtected(markdown string, blocks []protectedFence) string {
	return r.fragment(restoreFences(markdown, blocks))
}
func linkCard(match string) string {
	a := attributes(linkCardRE.FindStringSubmatch(match)[1])
	title := a["title"]
	desc := a["description"]
	href := a["href"]
	return fmt.Sprintf(`<a class="mpress-card mpress-link-card" href="%s"><strong>%s</strong><span>%s</span></a>`, html.EscapeString(href), html.EscapeString(title), html.EscapeString(desc))
}
func attributes(s string) map[string]string {
	result := map[string]string{}
	for _, m := range attrRE.FindAllStringSubmatch(s, -1) {
		v := m[2]
		if v == "" {
			v = m[3]
		}
		if v == "" {
			v = m[4]
		}
		result[m[1]] = v
	}
	return result
}

// containerPrefix splits a line into the container markers it opens with -
// indentation and blockquote arrows - and the content that follows them. A
// fence written inside a list item or a blockquote carries such a prefix on
// every line, so fence detection has to look past it.
func containerPrefix(line string) (prefix, content string) {
	cursor := 0
	for cursor < len(line) && (line[cursor] == ' ' || line[cursor] == '\t' || line[cursor] == '>') {
		cursor++
	}
	return line[:cursor], line[cursor:]
}

// protectedFence is a code block or inline span held aside while components are
// expanded. prefix records the container markers the fence opened with, so the
// block can follow its token if a component reindents the surrounding body.
type protectedFence struct {
	text   string
	prefix string
	token  string
	block  bool
}

// protectFences replaces fenced code blocks and inline code spans with opaque
// tokens so component expansion cannot rewrite example source that an author
// meant to display literally. Fences are matched behind their container prefix,
// and the token inherits that prefix so the surrounding list item or blockquote
// survives expansion intact.
func protectFences(source string) (string, []protectedFence) {
	lines := strings.Split(source, "\n")
	var out strings.Builder
	var blocks []protectedFence
	in := false
	marker := ""
	openPrefix := ""
	var block strings.Builder
	for _, line := range lines {
		_, content := containerPrefix(line)
		trim := strings.TrimSpace(content)
		if !in {
			opening, _, ok := annotatedFenceOpening(trim)
			if !ok {
				out.WriteString(line + "\n")
				continue
			}
			in = true
			marker = opening
			openPrefix, _ = containerPrefix(line)
			block.Reset()
			block.WriteString(line + "\n")
			continue
		}
		if in {
			block.WriteString(line + "\n")
			if annotatedFenceClosing(trim, marker) {
				token := fenceToken(len(blocks))
				blocks = append(blocks, protectedFence{text: block.String(), prefix: openPrefix, token: token, block: true})
				// Keep the token inside whatever container opened the fence.
				// restoreFences swaps the whole token line back for the block.
				out.WriteString(openPrefix + token + "\n")
				in = false
			}
			continue
		}
	}
	if in {
		out.WriteString(block.String())
	}
	protected := inlineCodeRE.ReplaceAllStringFunc(out.String(), func(code string) string {
		token := fenceToken(len(blocks))
		blocks = append(blocks, protectedFence{text: code, token: token})
		return token
	})
	return protected, blocks
}

// reindentFence rewrites a held block so every line carries prefix in place of
// the container prefix it was captured with. Components such as steps and tabs
// dedent their bodies while the fence is opaque, so the block has to follow the
// indentation its token ended up with.
func reindentFence(fence protectedFence, prefix string) string {
	if prefix == fence.prefix {
		return fence.text
	}
	lines := strings.Split(fence.text, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		if fence.prefix != "" && strings.HasPrefix(line, fence.prefix) {
			lines[i] = prefix + line[len(fence.prefix):]
			continue
		}
		lines[i] = prefix + strings.TrimLeft(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func restoreFences(source string, blocks []protectedFence) string {
	if len(blocks) == 0 {
		return source
	}
	// A fence token owns its line and carries the container prefix of the fence
	// it replaced, while the stored block already repeats that prefix on every
	// line. Swapping the entire line avoids emitting the prefix twice, and lets
	// the block adopt any reindentation applied to the token. Inline code tokens
	// sit inside surrounding text and are substituted in place.
	byToken := make(map[string]int, len(blocks))
	for i, block := range blocks {
		byToken[block.token] = i
	}
	lines := strings.Split(source, "\n")
	for index, line := range lines {
		linePrefix, content := containerPrefix(line)
		trimmed := strings.TrimSpace(content)
		if i, ok := byToken[trimmed]; ok && blocks[i].block {
			lines[index] = strings.TrimSuffix(reindentFence(blocks[i], linePrefix), "\n")
		}
	}
	source = strings.Join(lines, "\n")
	replacements := make([]string, 0, len(blocks)*2)
	for _, block := range blocks {
		replacements = append(replacements, block.token, block.text)
	}
	return strings.NewReplacer(replacements...).Replace(source)
}

func fenceToken(index int) string {
	buffer := make([]byte, 0, len("@@MPRESS_FENCE_000000@@"))
	buffer = append(buffer, "@@MPRESS_FENCE_"...)
	width := 6
	value := strconv.AppendInt(nil, int64(index), 10)
	for i := len(value); i < width; i++ {
		buffer = append(buffer, '0')
	}
	buffer = append(buffer, value...)
	buffer = append(buffer, '@', '@')
	return string(buffer)
}

// hasComponentSyntax deliberately recognises component constructs rather than
// punctuation. In particular, a normal @mention in prose must not send an
// entire document through the expensive fence-protection and form passes.
func hasComponentSyntax(source string) bool {
	if strings.Contains(source, "{badge") {
		return true
	}
	if hasComponentLine(source) {
		return true
	}
	if (strings.Contains(source, "button[") || strings.Contains(source, "@image{")) && inlineNativeRE.MatchString(source) {
		return true
	}
	if strings.Contains(source, "<") && componentHTMLRE.MatchString(source) {
		return true
	}
	return strings.Contains(source, "import") && importRE.MatchString(source)
}

func hasComponentLine(source string) bool {
	for start := 0; start < len(source); {
		end := strings.IndexByte(source[start:], '\n')
		if end < 0 {
			end = len(source)
		} else {
			end += start
		}
		cursor := start
		for cursor < end && (source[cursor] == ' ' || source[cursor] == '\t') {
			cursor++
		}
		if cursor+3 <= end && source[cursor:cursor+3] == ":::" {
			return true
		}
		if cursor < end && source[cursor] == '@' {
			cursor++
			if cursor < end && asciiLetter(source[cursor]) {
				cursor++
				for cursor < end && (asciiLetter(source[cursor]) || source[cursor] >= '0' && source[cursor] <= '9' || source[cursor] == '-') {
					cursor++
				}
				if cursor < end && (source[cursor] == '[' || source[cursor] == '{') {
					return true
				}
				for cursor < end && (source[cursor] == ' ' || source[cursor] == '\t') {
					cursor++
				}
				if cursor == end {
					return true
				}
			}
		}
		if end == len(source) {
			break
		}
		start = end + 1
	}
	return false
}

func asciiLetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

// hasFormBlockSyntax recognises the opening delimiter at the start of a line.
// It keeps ordinary pages out of the form parser, whose fence protection is
// otherwise unnecessary work for every Markdown file with an @mention.
func hasFormBlockSyntax(source string) bool {
	for _, line := range strings.Split(source, "\n") {
		if formOpenRE.MatchString(strings.TrimSpace(line)) {
			return true
		}
	}
	return false
}
