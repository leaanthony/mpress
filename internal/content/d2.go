package content

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"io/fs"
	"strings"
	"time"

	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2layouts/d2dagrelayout"
	"github.com/d2lang/d2/d2lib"
	"github.com/d2lang/d2/d2renderers/d2svg"
	d2log "github.com/d2lang/d2/lib/log"
	"github.com/d2lang/d2/lib/textmeasure"
)

// Fenced diagrams are self-contained: imports must never read ambient files
// from the developer's machine or the CI checkout.
type diagramFiles struct{}

func (diagramFiles) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

func renderD2Diagram(info, source string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ctx = d2log.WithDefault(ctx)
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return "", fmt.Errorf("D2 diagram text measurement: %w", err)
	}
	options := &d2lib.CompileOptions{
		FS:    diagramFiles{},
		Ruler: ruler,
		LayoutResolver: func(engine string) (d2graph.LayoutGraph, error) {
			if engine != "dagre" {
				return nil, fmt.Errorf("unsupported D2 layout %q; use dagre", engine)
			}
			return d2dagrelayout.DefaultLayout, nil
		},
	}
	scale := 1.0
	renderOptions := &d2svg.RenderOpts{Scale: &scale}
	diagram, _, err := d2lib.Compile(ctx, source, options, renderOptions)
	if err != nil {
		return "", fmt.Errorf("D2 diagram: %w", err)
	}
	svg, err := d2svg.Render(diagram, renderOptions)
	if err != nil {
		return "", fmt.Errorf("D2 diagram SVG: %w", err)
	}
	label := fenceMetadataValue(codeFenceTitleRE, info)
	if label == "" {
		var labels []string
		for _, shape := range diagram.Shapes {
			if text := strings.Join(strings.Fields(shape.Label), " "); text != "" {
				labels = append(labels, text)
			}
		}
		// The figure already supplies the diagram semantics. Its description
		// must contain only the document's labels, without an English prefix.
		label = strings.Join(labels, ", ")
	}
	// An image isolates SVG IDs/styles from other diagrams and prevents its
	// markup from becoming active document content. Fonts are embedded by D2.
	return `<figure class="mpress-diagram mpress-diagram-d2" tabindex="0" aria-label="` + html.EscapeString(label) + `"><img alt="` + html.EscapeString(label) + `" src="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(svg) + `"></figure>`, nil
}
