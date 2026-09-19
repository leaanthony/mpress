package translate

import (
	"bytes"
	"sort"

	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/mpd"
)

func rewriteMarkdownFragments(file string, source, target []byte, reverse bool) ([]byte, error) {
	if !bytes.Contains(target, []byte("#")) {
		return target, nil
	}
	renderer := content.NewRenderer()
	// Only headings are used here. Localized component labels render as note
	// paragraphs and disclosure summaries, not headings; use the neutral locale
	// for both documents, including normalization without language metadata.
	a, _, err := renderer.ParseBytes(file, "", source)
	if err != nil {
		return nil, err
	}
	b, _, err := renderer.ParseBytes(file, "", target)
	if err != nil {
		return nil, err
	}
	if len(a.Headings) != len(b.Headings) {
		return nil, errTranslationHeadingStructure
	}
	aliases := map[string]string{}
	for i, heading := range a.Headings {
		other := b.Headings[i]
		if heading.Level != other.Level {
			return nil, errTranslationHeadingStructure
		}
		from, to := heading.ID, other.ID
		if reverse {
			from, to = to, from
		}
		if from != "" && to != "" && from != to {
			aliases["#"+from] = "#" + to
		}
	}
	return patchMarkdownFragments(target, aliases), nil
}

// Parse link ranges rather than replacing strings globally: examples containing
// [text](#anchor) in code fences and inline code must remain byte-identical.
func patchMarkdownFragments(source []byte, aliases map[string]string) []byte {
	start := frontmatterEnd(source)
	view, _ := markdownDirectiveView(source, start)
	for i := 0; i < start; i++ {
		if view[i] != '\n' {
			view[i] = ' '
		}
	}
	parsed := mpd.Parse("fragments.mpd", view)
	var links []mpd.Range
	for _, node := range parsed.Nodes {
		if node.Kind == mpd.KindLink && node.Flags&mpd.FlagReference == 0 {
			links = append(links, node.Content)
		}
	}
	// Component declarations are masked above. Only changed lines can contain
	// component hrefs; this excludes component-looking code examples.
	for position := start; position < len(source); {
		end := position
		for end < len(source) && source[end] != '\n' {
			end++
		}
		line := source[position:end]
		if !bytes.Equal(line, view[position:end]) && markdownDirective.Match(line) {
			for _, attr := range markdownAttribute.FindAllSubmatchIndex(line, -1) {
				if string(line[attr[2]:attr[3]]) != "href" {
					continue
				}
				a, b := attr[4], attr[5]
				if a < b && (line[a] == '"' || line[a] == '\'') {
					a++
					b--
				}
				links = append(links, mpd.Range{Start: uint32(position + a), End: uint32(position + b)})
			}
			if match := markdownCompactLabel.FindSubmatchIndex(line); match != nil && string(line[match[2]:match[3]]) == "button" {
				after := match[1]
				if after < len(line) && line[after] == '(' {
					if close := bytes.IndexByte(line[after+1:], ')'); close >= 0 {
						links = append(links, mpd.Range{Start: uint32(position + after + 1), End: uint32(position + after + 1 + close)})
					}
				}
			}
		}
		position = end + 1
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Start > links[j].Start })
	output := append([]byte(nil), source...)
	for _, span := range links {
		if value, ok := aliases[string(source[span.Start:span.End])]; ok {
			output = append(output[:span.Start], append([]byte(value), output[span.End:]...)...)
		}
	}
	return output
}
