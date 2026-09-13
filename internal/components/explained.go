package components

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Explained renders a bidirectional code-explanation panel.
// Hovering prose highlights the corresponding code lines and vice versa.
//
// Usage:
//
//	:::explained
//	```go
//	func main() {           // (1)
//	    fmt.Println("hi")   // (2)
//	}
//	```
//
//	(1) The main function is the entry point.
//
//	(2) This prints "hi" to stdout.
//	:::
//
// Code lines are tagged with `// (N)` annotations.
// Prose paragraphs start with `(N)` to link to the annotated code lines.
// Hovering either side highlights the matching pair.
type Explained struct {
	Meta    map[string]string
	Content string
}

func (e *Explained) Parse(content string) error {
	e.Content = content
	return nil
}

// explainedCodeBlockRe matches backtick and tilde fenced code blocks.
var explainedCodeBlockRe = regexp.MustCompile("(?ms)^(?:```|~~~)(\\w*)\\n(.*?)\\n(?:```|~~~)")

// annotationRe matches // (N) or # (N) code annotations.
var annotationRe = regexp.MustCompile(`(?://|#)\s*\((\d+)\)\s*$`)

// proseRefRe matches (N) at the start of a prose paragraph.
var proseRefRe = regexp.MustCompile(`^\((\d+)\)\s+`)

func (e *Explained) Render() (string, error) {
	content := strings.TrimSpace(e.Content)
	if content == "" {
		return "", nil
	}

	// Split into code block and prose
	codeMatch := explainedCodeBlockRe.FindStringSubmatchIndex(content)
	if codeMatch == nil {
		// No code block found — render as plain content
		return fmt.Sprintf(`<div class="mpress-explained">%s</div>`, content), nil
	}

	lang := content[codeMatch[2]:codeMatch[3]]
	codeRaw := content[codeMatch[4]:codeMatch[5]]
	proseRaw := strings.TrimSpace(content[codeMatch[1]:])
	explanationByRef := make(map[string]string)
	for _, para := range splitParagraphs(proseRaw) {
		para = strings.TrimSpace(para)
		if match := proseRefRe.FindStringSubmatch(para); match != nil {
			explanationByRef[match[1]] = proseRefRe.ReplaceAllString(para, "")
		}
	}

	// Process code lines: extract annotations and build highlighted HTML
	codeLines := strings.Split(codeRaw, "\n")
	var codeHTML strings.Builder
	codeHTML.WriteString(fmt.Sprintf(`<div class="mpress-explained-code"><pre><code class="language-%s">`, html.EscapeString(lang)))

	for _, line := range codeLines {
		m := annotationRe.FindStringSubmatch(line)
		if m != nil {
			// Line has annotation. Preserve its authored indentation.
			ref := m[1]
			cleanLine := strings.TrimRight(annotationRe.ReplaceAllString(line, ""), " \t")
			codeHTML.WriteString(fmt.Sprintf(`<span class="mpress-explained-line" data-ref="%s" data-explanation="%s" tabindex="0">%s</span>`, ref, html.EscapeString(explanationByRef[ref]), html.EscapeString(cleanLine)))
		} else {
			codeHTML.WriteString(fmt.Sprintf(`<span class="mpress-explained-line">%s</span>`, html.EscapeString(line)))
		}
	}
	codeHTML.WriteString("</code></pre></div>")

	// Process prose paragraphs: split by blank lines, match (N) refs
	var proseHTML strings.Builder
	proseHTML.WriteString(`<div class="mpress-explained-prose">`)

	paragraphs := splitParagraphs(proseRaw)
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		m := proseRefRe.FindStringSubmatch(para)
		if m != nil {
			ref := m[1]
			text := proseRefRe.ReplaceAllString(para, "")
			proseHTML.WriteString(fmt.Sprintf(`<p class="mpress-explained-para" data-ref="%s"><span class="mpress-explained-ref">%s</span><span class="mpress-explained-copy">%s</span></p>`,
				ref, ref, html.EscapeString(text)))
		} else {
			proseHTML.WriteString(fmt.Sprintf(`<p>%s</p>`, html.EscapeString(para)))
		}
	}
	proseHTML.WriteString("</div>")

	// Combine into the explained container
	var b strings.Builder
	b.WriteString(`<div class="mpress-explained" role="group" aria-label="Code explanation">`)
	b.WriteString(codeHTML.String())
	b.WriteString(proseHTML.String())
	b.WriteString("</div>")

	return b.String(), nil
}

// splitParagraphs splits text by blank lines into paragraphs. A numbered
// explanation also starts a new paragraph so adjacent notes do not collapse
// into the first note.
func splitParagraphs(text string) []string {
	var result []string
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		result = append(result, current.String())
		current.Reset()
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		if current.Len() > 0 && proseRefRe.MatchString(trimmed) {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(trimmed)
	}
	flush()
	return result
}

// ExplainedScript returns the JS for bidirectional highlight interaction.
// Injected once per page when :::explained blocks are detected.
func ExplainedScript() string {
	return `<script>
(function(){
  document.querySelectorAll('.mpress-explained').forEach(function(el){
    var lines = el.querySelectorAll('.mpress-explained-line[data-ref]');
    var paras = el.querySelectorAll('.mpress-explained-para[data-ref]');
    var cls = 'mpress-explained-active';

    function highlight(ref) {
      lines.forEach(function(l){ l.classList.toggle(cls, l.dataset.ref === ref); });
      paras.forEach(function(p){ p.classList.toggle(cls, p.dataset.ref === ref); });
    }
    function clear() {
      lines.forEach(function(l){ l.classList.remove(cls); });
      paras.forEach(function(p){ p.classList.remove(cls); });
    }

    lines.forEach(function(l){
      l.addEventListener('mouseenter', function(){ highlight(l.dataset.ref); });
      l.addEventListener('mouseleave', clear);
      l.addEventListener('focus', function(){ highlight(l.dataset.ref); });
      l.addEventListener('blur', clear);
      l.setAttribute('tabindex', '0');
    });
    paras.forEach(function(p){
      p.addEventListener('mouseenter', function(){ highlight(p.dataset.ref); });
      p.addEventListener('mouseleave', clear);
    });
  });
})();
</script>`
}
