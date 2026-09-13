package content

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Code fences are rendered from goldmark's own AST rather than expanded into
// raw HTML by a string pass before parsing.
//
// The superseded renderCodeFences pass rewrote the Markdown source, turning
// every fence into a <div class="mpress-codeframe"> blob, and handed the result
// to goldmark - which then had to re-scan that generated markup with its HTML
// block parser. Because the pass matched fences line by line it had no idea
// whether a fence sat inside a list item or a blockquote, so it lifted the
// block out of its container and broke the surrounding structure.
//
// Rendering from ast.FencedCodeBlock keeps every fence in the block context the
// author wrote it in, and removes both the extra source pass and the HTML
// re-parse.

type codeFenceRenderer struct{}

func (codeFenceRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, renderFencedCodeBlock)
}

// codeFenceExtension installs the fenced code renderer at a priority that wins
// over goldmark's built-in <pre><code> output.
type codeFenceExtension struct{}

func (codeFenceExtension) Extend(md goldmark.Markdown) {
	md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(codeFenceRenderer{}, 100),
	))
}

func renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	block := node.(*ast.FencedCodeBlock)

	info := ""
	if block.Info != nil {
		info = string(block.Info.Segment.Value(source))
	}

	// goldmark reports each line of the body with its terminator attached, while
	// the code frame renderers join lines themselves, so drop it here.
	segments := block.Lines()
	lines := make([]string, 0, segments.Len())
	for i := 0; i < segments.Len(); i++ {
		segment := segments.At(i)
		line := string(segment.Value(source))
		lines = append(lines, strings.TrimRight(line, "\r\n"))
	}

	// Render only the frame this fence actually needs; both walk the body and
	// escape it, so building the unused one would double that work per fence.
	var markup string
	if isTerminalFence(info) {
		markup = renderTerminalFence(info, lines)
	} else {
		markup = renderAnnotatedCodeFrame(info, lines)
	}
	_, _ = w.WriteString(markup)
	// The pre-pass emitted this markup as a Markdown HTML block, and goldmark
	// terminates an HTML block with a single newline. The terminal component
	// already ends with one, so only add it when missing.
	if !strings.HasSuffix(markup, "\n") {
		_ = w.WriteByte('\n')
	}
	return ast.WalkSkipChildren, nil
}
