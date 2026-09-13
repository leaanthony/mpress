package translate

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/d2lang/d2/d2compiler"
	"github.com/d2lang/d2/d2format"
	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2oracle"
)

type translationDiagramFS struct{}

func (translationDiagramFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}

func compileTranslationDiagram(source string) (*d2graph.Graph, error) {
	graph, _, err := d2compiler.Compile("translation.d2", strings.NewReader(source), &d2compiler.CompileOptions{FS: translationDiagramFS{}})
	return graph, err
}

// Diagram labels share a containing source range. They are applied together
// through D2's editing API; identifiers, connections and executable code labels
// never become translation input.
func d2Segments(source []byte, start, end, ordinal int, section string) ([]Segment, error) {
	graph, err := compileTranslationDiagram(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse D2 translation labels: %w", err)
	}
	var result []Segment
	add := func(key, value, tag string) {
		if (tag != "" && tag != "markdown") || !translatableText(value) {
			return
		}
		text, placeholders := protect(value)
		// D2 can reorder implicit objects when the editor adds explicit labels.
		// Bind translation state to the graph key, never its traversal position.
		id := fmt.Sprintf("d%04d-label-%s", ordinal, Hash(key)[:16])
		result = append(result, Segment{ID: id, Kind: "diagram-label", Section: section, Original: value, Text: text, Start: start, End: end, SourceHash: Hash(value), Placeholders: placeholders, D2Key: key, D2Tag: tag})
	}
	if graph.Root.Label.Value != "" {
		add("label", graph.Root.Label.Value, graph.Root.Language)
	}
	for _, object := range graph.Objects {
		if object.Shape.Value == "code" || object.Shape.Value == "sql_table" || object.Shape.Value == "class" || (object.Language != "" && object.Language != "markdown") {
			continue
		}
		add(object.AbsID()+".label", object.Label.Value, object.Language)
	}
	for _, edge := range graph.Edges {
		add(edge.AbsID()+".label", edge.Label.Value, edge.Language)
	}
	return result, nil
}

func applyD2Labels(document *Document, container Segment, values map[string]string, restoreText bool) (string, error) {
	graph, err := compileTranslationDiagram(string(document.Source[container.Start:container.End]))
	if err != nil {
		return "", err
	}
	for _, segment := range document.Segments {
		if segment.D2Key == "" || segment.Start != container.Start {
			continue
		}
		value, ok := values[segment.ID]
		if !ok {
			continue
		}
		if restoreText {
			value, err = restore(segment, value, false)
			if err != nil {
				return "", err
			}
		}
		var tag *string
		if segment.D2Tag != "" {
			tag = &segment.D2Tag
		}
		graph, err = d2oracle.Set(graph, nil, segment.D2Key, tag, &value)
		if err != nil {
			return "", fmt.Errorf("translate diagram label %s: %w", segment.ID, err)
		}
	}
	return d2format.Format(graph.AST), nil
}
