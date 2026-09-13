package mpd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Markdown renders a parsed MPress Document into the portable Markdown
// interchange syntax consumed by the production content renderer. Keeping
// this adapter beside the native parser gives .mpd files the same components,
// highlighting, search text, and page metadata as imported Markdown without
// routing MPD source through Goldmark.
func Markdown(document *Document) ([]byte, error) {
	if document == nil || int(document.Root) >= len(document.Nodes) {
		return nil, fmt.Errorf("invalid MPD document")
	}
	for _, diagnostic := range document.Diagnostics {
		if diagnostic.Severity == SeverityError {
			return nil, fmt.Errorf("%s:%d:%d: %s", document.Filename, diagnostic.Position.Line, diagnostic.Position.Column, diagnostic.Message)
		}
	}
	renderer := markdownRenderer{document: document}
	renderer.children(document.Root, 0)
	return []byte(strings.TrimRight(renderer.output.String(), "\n") + "\n"), nil
}

type markdownRenderer struct {
	document         *Document
	output           strings.Builder
	literalHardBreak bool
}

func (r *markdownRenderer) children(parent uint32, indent int) {
	iterator := r.document.Children(parent)
	for {
		index, node, ok := iterator.Next()
		if !ok {
			return
		}
		r.block(index, node, indent)
	}
}

func (r *markdownRenderer) block(index uint32, node *Node, indent int) {
	switch node.Kind {
	case KindMetadata:
		r.metadata(index, node)
	case KindParagraph:
		r.writeIndent(indent)
		r.inlineChildren(index)
		r.output.WriteString("\n\n")
	case KindHeading:
		r.writeIndent(indent)
		r.output.WriteString(strings.Repeat("#", int(node.Level)))
		r.output.WriteByte(' ')
		r.inlineChildren(index)
		r.output.WriteString("\n\n")
	case KindRule:
		r.writeIndent(indent)
		r.output.WriteString("---\n\n")
	case KindList:
		r.list(index, node, indent)
	case KindQuote:
		r.quote(index, node, indent)
	case KindCode:
		r.code(index, node, indent)
	case KindRawHTML:
		r.writeIndent(indent)
		r.output.Write(r.document.Text(node.Content))
		if !strings.HasSuffix(string(r.document.Text(node.Content)), "\n") {
			r.output.WriteByte('\n')
		}
		r.output.WriteByte('\n')
	case KindComment:
		return
	case KindTable:
		r.table(index, node, indent)
	case KindFootnote:
		r.footnote(index, node, indent)
	case KindComponent, KindImport, KindInclude:
		r.component(index, node, indent)
	default:
		r.writeIndent(indent)
		r.output.Write(r.document.Text(node.Source))
		if !strings.HasSuffix(string(r.document.Text(node.Source)), "\n") {
			r.output.WriteByte('\n')
		}
	}
}

func (r *markdownRenderer) metadata(_ uint32, node *Node) {
	r.output.WriteString("---\n")
	start := int(node.FirstAttr)
	for offset := 0; offset < int(node.AttrCount); offset++ {
		attribute := r.document.Attributes[start+offset]
		name := string(r.document.Text(attribute.Name))
		if name == "schema" {
			continue
		}
		value := string(r.document.Text(attribute.Value))
		r.output.WriteString(name)
		r.output.WriteString(": ")
		r.output.WriteString(value)
		r.output.WriteByte('\n')
	}
	r.output.WriteString("---\n\n")
}

func (r *markdownRenderer) inlineChildren(parent uint32) {
	iterator := r.document.Children(parent)
	for {
		index, node, ok := iterator.Next()
		if !ok {
			return
		}
		r.inline(index, node)
	}
}

func (r *markdownRenderer) inline(index uint32, node *Node) {
	switch node.Kind {
	case KindText:
		r.output.Write(r.document.Text(node.Content))
	case KindSoftBreak:
		r.output.WriteByte(' ')
	case KindHardBreak:
		if r.literalHardBreak {
			r.output.WriteByte('\n')
		} else {
			r.output.WriteString("  \n")
		}
	case KindEmphasis:
		r.output.WriteByte('*')
		r.inlineChildren(index)
		r.output.WriteByte('*')
	case KindStrong:
		r.output.WriteString("**")
		r.inlineChildren(index)
		r.output.WriteString("**")
	case KindCodeSpan:
		content := string(r.document.Text(node.Content))
		marker := "`"
		if strings.Contains(content, "`") {
			marker = "``"
		}
		r.output.WriteString(marker)
		r.output.WriteString(content)
		r.output.WriteString(marker)
	case KindLink, KindImage:
		if node.Kind == KindImage {
			r.output.WriteByte('!')
		}
		r.output.WriteByte('[')
		r.inlineChildren(index)
		r.output.WriteString("](")
		r.output.Write(r.document.Text(node.Content))
		r.output.WriteByte(')')
	case KindAutomaticLink:
		r.output.WriteByte('<')
		r.output.Write(r.document.Text(node.Content))
		r.output.WriteByte('>')
	case KindEmoji:
		r.output.WriteByte(':')
		r.output.Write(r.document.Text(node.Name))
		r.output.WriteByte(':')
	case KindFootnoteReference:
		r.output.WriteString("[^")
		r.output.Write(r.document.Text(node.Name))
		r.output.WriteByte(']')
	case KindMetadataReference:
		r.output.WriteString(r.metadataValue(string(r.document.Text(node.Content))))
	case KindInlineRole:
		r.inlineRole(index, string(r.document.Text(node.Name)))
	default:
		r.output.Write(r.document.Text(node.Content))
	}
}

func (r *markdownRenderer) inlineRole(index uint32, name string) {
	tag := map[string]string{"mark": "mark", "delete": "del", "sub": "sub", "sup": "sup", "key": "kbd"}[name]
	if tag == "" {
		r.inlineChildren(index)
		return
	}
	r.output.WriteByte('<')
	r.output.WriteString(tag)
	r.output.WriteByte('>')
	r.inlineChildren(index)
	r.output.WriteString("</")
	r.output.WriteString(tag)
	r.output.WriteByte('>')
}

func (r *markdownRenderer) metadataValue(name string) string {
	root := &r.document.Nodes[r.document.Root]
	iterator := r.document.Children(r.document.Root)
	for {
		_, node, ok := iterator.Next()
		if !ok {
			break
		}
		if node.Kind == KindMetadata {
			root = node
			break
		}
	}
	start := int(root.FirstAttr)
	for offset := 0; offset < int(root.AttrCount); offset++ {
		attribute := r.document.Attributes[start+offset]
		if string(r.document.Text(attribute.Name)) != name {
			continue
		}
		var value any
		if json.Unmarshal(r.document.Text(attribute.Value), &value) != nil {
			return string(r.document.Text(attribute.Value))
		}
		switch typed := value.(type) {
		case string:
			return typed
		case nil:
			return ""
		default:
			encoded, _ := json.Marshal(typed)
			return string(encoded)
		}
	}
	return ""
}

func (r *markdownRenderer) code(index uint32, node *Node, indent int) {
	marker := "```"
	content := string(r.document.Text(node.Content))
	for strings.Contains(content, marker) {
		marker += "`"
	}
	r.writeIndent(indent)
	r.output.WriteString(marker)
	r.output.Write(r.document.Text(node.Name))
	attributes := r.attributes(node)
	if len(attributes) > 0 {
		r.output.WriteString(" {")
		for offset, attribute := range attributes {
			if offset > 0 {
				r.output.WriteByte(' ')
			}
			r.output.WriteString(attribute)
		}
		r.output.WriteByte('}')
	}
	r.output.WriteByte('\n')
	r.output.WriteString(content)
	if content != "" && !strings.HasSuffix(content, "\n") {
		r.output.WriteByte('\n')
	}
	r.writeIndent(indent)
	r.output.WriteString(marker)
	r.output.WriteString("\n\n")
	_ = index
}

func (r *markdownRenderer) list(index uint32, node *Node, indent int) {
	iterator := r.document.Children(index)
	position := 1
	for {
		itemIndex, item, ok := iterator.Next()
		if !ok {
			break
		}
		r.writeIndent(indent)
		if node.Flags&FlagOrdered != 0 {
			number := string(r.document.Text(item.Name))
			if number == "" {
				number = strconv.Itoa(position)
			}
			r.output.WriteString(number + ". ")
		} else {
			r.output.WriteString("- ")
		}
		if item.Flags&FlagChecked != 0 {
			r.output.WriteString("[x] ")
		} else if item.Flags&FlagUnchecked != 0 {
			r.output.WriteString("[ ] ")
		}
		children := r.document.Children(itemIndex)
		first := true
		for {
			childIndex, child, childOK := children.Next()
			if !childOK {
				break
			}
			if first && child.Kind == KindParagraph {
				r.inlineChildren(childIndex)
				r.output.WriteByte('\n')
				first = false
				continue
			}
			if first {
				r.output.WriteByte('\n')
				first = false
			}
			r.block(childIndex, child, indent+2)
		}
		position++
	}
	r.output.WriteByte('\n')
}

func (r *markdownRenderer) quote(index uint32, _ *Node, indent int) {
	var nested markdownRenderer
	nested.document = r.document
	nested.children(index, 0)
	for _, line := range strings.Split(strings.TrimRight(nested.output.String(), "\n"), "\n") {
		r.writeIndent(indent)
		r.output.WriteString(">")
		if line != "" {
			r.output.WriteByte(' ')
			r.output.WriteString(line)
		}
		r.output.WriteByte('\n')
	}
	r.output.WriteByte('\n')
}

func (r *markdownRenderer) table(index uint32, node *Node, indent int) {
	rows := r.tableRows(index)
	if len(rows) == 0 {
		return
	}
	attributes := r.attributeMap(node)
	interactive := attributes["search"] != "" || attributes["filter"] != "" || attributes["sort"] != "" || attributes["paginate"] != "" || attributes["column-separators"] != ""
	if interactive {
		r.writeIndent(indent)
		r.output.WriteString("@table")
		r.writeMarkdownAttributes(node)
		r.output.WriteByte('\n')
	}
	for rowIndex, cells := range rows {
		r.writeIndent(indent)
		r.output.WriteString("| ")
		r.output.WriteString(strings.Join(cells, " | "))
		r.output.WriteString(" |\n")
		if rowIndex == 0 {
			align := r.tableAlign(attributes["align"], len(cells))
			r.writeIndent(indent)
			r.output.WriteString("| ")
			r.output.WriteString(strings.Join(align, " | "))
			r.output.WriteString(" |\n")
		}
	}
	if interactive {
		r.writeIndent(indent)
		r.output.WriteString("@end\n")
	}
	r.output.WriteByte('\n')
}

func (r *markdownRenderer) tableRows(index uint32) [][]string {
	var rows [][]string
	iterator := r.document.Children(index)
	for {
		rowIndex, row, ok := iterator.Next()
		if !ok {
			break
		}
		if row.Kind != KindTableRow {
			continue
		}
		var cells []string
		cellIterator := r.document.Children(rowIndex)
		for {
			cellIndex, _, cellOK := cellIterator.Next()
			if !cellOK {
				break
			}
			var nested markdownRenderer
			nested.document = r.document
			nested.inlineChildren(cellIndex)
			cells = append(cells, strings.TrimSpace(nested.output.String()))
		}
		rows = append(rows, cells)
	}
	return rows
}

func (r *markdownRenderer) tableAlign(value string, count int) []string {
	result := make([]string, count)
	for index := range result {
		result[index] = "---"
	}
	var align []string
	if json.Unmarshal([]byte(value), &align) != nil {
		align = strings.Split(value, ",")
	}
	for index := 0; index < len(align) && index < len(result); index++ {
		switch strings.TrimSpace(align[index]) {
		case "left":
			result[index] = ":---"
		case "center":
			result[index] = ":---:"
		case "right":
			result[index] = "---:"
		}
	}
	return result
}

func (r *markdownRenderer) footnote(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("[^")
	r.output.WriteString(r.attributeMap(node)["id"])
	r.output.WriteString("]: ")
	var nested markdownRenderer
	nested.document = r.document
	nested.children(index, 0)
	text := strings.TrimSpace(nested.output.String())
	r.output.WriteString(strings.ReplaceAll(text, "\n", "\n    "))
	r.output.WriteString("\n\n")
}

func (r *markdownRenderer) component(index uint32, node *Node, indent int) {
	name := string(r.document.Text(node.Name))
	switch name {
	case "hr":
		r.block(index, &Node{Kind: KindRule}, indent)
	case "link":
		return
	case "tabs", "preview-tabs", "file-tabs":
		r.tabs(index, node, indent, name)
	case "steps":
		r.steps(index, node, indent)
	case "tutorial":
		r.tutorial(index, node, indent)
	case "pricing":
		r.pricing(index, node, indent)
	case "testimonials":
		r.testimonials(index, node, indent)
	case "cards":
		r.cards(index, node, indent)
	case "timeline":
		r.namedList(index, node, indent, "event", true)
	case "capabilities":
		r.namedList(index, node, indent, "capability", false)
	case "resources":
		r.resources(index, node, indent)
	case "video":
		r.video(node, indent)
	case "diff", "explained", "filetree", "headline", "terminal":
		r.literalComponent(index, node, indent)
	case "image", "qr", "input", "computed":
		r.leafComponent(node, indent)
	default:
		r.writeIndent(indent)
		r.output.WriteByte('@')
		r.output.WriteString(name)
		r.writeMarkdownAttributes(node)
		r.output.WriteByte('\n')
		r.children(index, indent)
		if componentClassFor(r.document.Text(node.Name)) == componentContainer {
			r.writeIndent(indent)
			r.output.WriteString("@end\n\n")
		}
	}
}

func (r *markdownRenderer) leafComponent(node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteByte('@')
	r.output.Write(r.document.Text(node.Name))
	r.writeMarkdownAttributes(node)
	r.output.WriteString("\n\n")
}

func (r *markdownRenderer) literalComponent(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteByte('@')
	r.output.Write(r.document.Text(node.Name))
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	body := ""
	if node.Content.End > node.Content.Start {
		body = strings.TrimRight(string(r.document.Text(node.Content)), "\r\n")
	} else {
		var nested markdownRenderer
		nested.document = r.document
		nested.literalHardBreak = true
		nested.children(index, 0)
		body = strings.TrimRight(nested.output.String(), "\n")
	}
	if body != "" {
		r.output.WriteString(body)
		r.output.WriteByte('\n')
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) tutorial(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@tutorial")
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	for {
		lessonIndex, lesson, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(lesson.Name)) != "lesson" {
			r.block(lessonIndex, lesson, indent)
			continue
		}
		r.writeIndent(indent)
		r.output.WriteString("### ")
		r.output.WriteString(r.attributeMap(lesson)["title"])
		r.output.WriteString("\n\n")
		r.children(lessonIndex, indent)
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) video(node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@video")
	if node.AttrCount > 0 {
		r.output.WriteByte('{')
		start := int(node.FirstAttr)
		for offset := 0; offset < int(node.AttrCount); offset++ {
			if offset > 0 {
				r.output.WriteByte(' ')
			}
			attribute := r.document.Attributes[start+offset]
			name := string(r.document.Text(attribute.Name))
			r.output.WriteString(name)
			if attribute.Flag {
				continue
			}
			value := string(r.document.Text(attribute.Value))
			var list []string
			if json.Unmarshal([]byte(value), &list) == nil {
				value = strings.Join(list, ",")
			} else {
				var scalar string
				if json.Unmarshal([]byte(value), &scalar) == nil {
					value = scalar
				}
			}
			r.output.WriteByte('=')
			r.output.WriteString(strconv.Quote(value))
		}
		r.output.WriteByte('}')
	}
	r.output.WriteString("\n\n")
}

func (r *markdownRenderer) pricing(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@pricing")
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	first := true
	for {
		planIndex, plan, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(plan.Name)) != "plan" {
			continue
		}
		if !first {
			r.writeIndent(indent)
			r.output.WriteString("---\n")
		}
		first = false
		var nested markdownRenderer
		nested.document = r.document
		nested.children(planIndex, 0)
		body := strings.TrimSpace(nested.output.String())
		if strings.HasPrefix(body, "## ") {
			body = "### " + strings.TrimPrefix(body, "## ")
		}
		r.output.WriteString(body)
		r.output.WriteByte('\n')
		if _, recommended := r.attributeMap(plan)["recommended"]; recommended {
			r.writeIndent(indent)
			r.output.WriteString("recommended\n")
		}
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) testimonials(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@testimonials")
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	first := true
	for {
		testimonialIndex, testimonial, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(testimonial.Name)) != "testimonial" {
			continue
		}
		if !first {
			r.writeIndent(indent)
			r.output.WriteString("---\n")
		}
		first = false
		var nested markdownRenderer
		nested.document = r.document
		nested.children(testimonialIndex, 0)
		r.output.WriteString(strings.TrimSpace(nested.output.String()))
		r.output.WriteByte('\n')
		attrs := r.attributeMap(testimonial)
		if author := attrs["author"]; author != "" {
			r.writeIndent(indent)
			r.output.WriteString("— " + author)
			if company := attrs["company"]; company != "" {
				r.output.WriteString(", " + company)
			}
			r.output.WriteByte('\n')
		}
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) tabs(index uint32, node *Node, indent int, name string) {
	r.writeIndent(indent)
	r.output.WriteByte('@')
	r.output.WriteString(name)
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	for {
		tabIndex, tab, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(tab.Name)) != "tab" {
			r.block(tabIndex, tab, indent)
			continue
		}
		r.writeIndent(indent)
		r.output.WriteByte('[')
		r.output.WriteString(r.attributeMap(tab)["label"])
		r.output.WriteString("]\n")
		r.children(tabIndex, indent)
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) steps(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@steps")
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	for {
		stepIndex, step, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(step.Name)) != "step" {
			r.block(stepIndex, step, indent)
			continue
		}
		r.writeIndent(indent)
		r.output.WriteString("### ")
		r.output.WriteString(r.attributeMap(step)["title"])
		r.output.WriteByte('\n')
		r.children(stepIndex, indent)
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) cards(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@cards")
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	first := true
	for {
		cardIndex, card, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(card.Name)) != "card" {
			continue
		}
		if !first {
			r.output.WriteString("---\n")
		}
		first = false
		attrs := r.attributeMap(card)
		r.writeIndent(indent)
		if attrs["href"] != "" {
			r.output.WriteString("[")
			r.output.WriteString(attrs["title"])
			r.output.WriteString("](")
			r.output.WriteString(attrs["href"])
			r.output.WriteString(")\n")
		} else {
			r.output.WriteString(attrs["title"])
			r.output.WriteByte('\n')
		}
		r.children(cardIndex, indent)
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) namedList(index uint32, node *Node, indent int, childName string, ordered bool) {
	r.writeIndent(indent)
	r.output.WriteByte('@')
	r.output.Write(r.document.Text(node.Name))
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	position := 1
	for {
		childIndex, child, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(child.Name)) != childName {
			continue
		}
		attrs := r.attributeMap(child)
		r.writeIndent(indent)
		if ordered {
			r.output.WriteString(strconv.Itoa(position) + ". **" + attrs["title"] + "** ")
		} else {
			r.output.WriteString("- **" + attrs["name"] + "** ")
		}
		var nested markdownRenderer
		nested.document = r.document
		nested.children(childIndex, 0)
		r.output.WriteString(strings.TrimSpace(nested.output.String()))
		r.output.WriteByte('\n')
		position++
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) resources(index uint32, node *Node, indent int) {
	r.writeIndent(indent)
	r.output.WriteString("@resources")
	r.writeMarkdownAttributes(node)
	r.output.WriteByte('\n')
	iterator := r.document.Children(index)
	first := true
	for {
		resourceIndex, resource, ok := iterator.Next()
		if !ok {
			break
		}
		if string(r.document.Text(resource.Name)) != "resource" {
			continue
		}
		if !first {
			r.output.WriteString("---\n")
		}
		first = false
		attrs := r.attributeMap(resource)
		r.output.WriteString("[" + attrs["label"] + "](" + attrs["href"] + ")\n### " + attrs["title"] + "\n")
		r.children(resourceIndex, indent)
	}
	r.writeIndent(indent)
	r.output.WriteString("@end\n\n")
}

func (r *markdownRenderer) attributes(node *Node) []string {
	result := make([]string, 0, node.AttrCount)
	start := int(node.FirstAttr)
	for offset := 0; offset < int(node.AttrCount); offset++ {
		attribute := r.document.Attributes[start+offset]
		name := string(r.document.Text(attribute.Name))
		if attribute.Flag {
			result = append(result, name)
			continue
		}
		result = append(result, name+"="+string(r.document.Text(attribute.Value)))
	}
	return result
}

func (r *markdownRenderer) attributeMap(node *Node) map[string]string {
	result := make(map[string]string, node.AttrCount)
	start := int(node.FirstAttr)
	for offset := 0; offset < int(node.AttrCount); offset++ {
		attribute := r.document.Attributes[start+offset]
		name := string(r.document.Text(attribute.Name))
		if attribute.Flag {
			result[name] = "true"
			continue
		}
		value := string(r.document.Text(attribute.Value))
		var decoded any
		if json.Unmarshal([]byte(value), &decoded) == nil {
			switch typed := decoded.(type) {
			case string:
				value = typed
			case bool:
				value = strconv.FormatBool(typed)
			case float64:
				value = strconv.FormatFloat(typed, 'f', -1, 64)
			}
		}
		result[name] = value
	}
	return result
}

func (r *markdownRenderer) writeMarkdownAttributes(node *Node) {
	attributes := r.attributes(node)
	if len(attributes) == 0 {
		return
	}
	r.output.WriteByte('{')
	for index, attribute := range attributes {
		if index > 0 {
			r.output.WriteByte(' ')
		}
		name, value, found := strings.Cut(attribute, "=")
		if !found {
			r.output.WriteString(name)
			continue
		}
		r.output.WriteString(name)
		r.output.WriteByte('=')
		if strings.HasPrefix(value, "\"") {
			r.output.WriteString(value)
		} else {
			r.output.WriteString(strconv.Quote(value))
		}
	}
	r.output.WriteByte('}')
}

func (r *markdownRenderer) writeIndent(indent int) {
	if indent > 0 {
		r.output.WriteString(strings.Repeat(" ", indent))
	}
}
