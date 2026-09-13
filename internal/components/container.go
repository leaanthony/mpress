package components

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

type Container struct {
	Meta    map[string]string
	Content string
}

func init() {
	Registry["container"] = func(meta map[string]string) Component { return &Container{Meta: meta} }
}

func (c *Container) Parse(content string) error {
	c.Content = content
	return nil
}

func (c *Container) Render() (string, error) {
	allowed := map[string]bool{
		"display": true, "direction": true, "wrap": true, "justify": true,
		"align": true, "gap": true, "row-gap": true, "column-gap": true,
		"width": true, "height": true, "min-width": true, "max-width": true,
		"padding": true, "margin": true, "position": true, "top": true,
		"right": true, "bottom": true, "left": true, "z-index": true,
		"text-align": true, "color": true, "font-size": true,
	}
	keys := make([]string, 0, len(c.Meta))
	for key := range c.Meta {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var styles []string
	classes := []string{"mpress-container"}
	attrs := ""
	for _, key := range keys {
		value := strings.TrimSpace(c.Meta[key])
		if strings.ContainsAny(value, ";{}") {
			continue
		}
		switch key {
		case "class":
			for _, class := range strings.Fields(value) {
				if token := safeToken(class); token != "" {
					classes = append(classes, token)
				}
			}
		case "id":
			attrs += fmt.Sprintf(` id="%s"`, html.EscapeString(value))
		case "columns":
			styles = append(styles, "grid-template-columns:repeat("+value+",minmax(0,1fr))")
		default:
			if allowed[key] {
				property := key
				if key == "direction" {
					property = "flex-direction"
				} else if key == "wrap" {
					property = "flex-wrap"
				} else if key == "justify" {
					property = "justify-content"
				} else if key == "align" {
					property = "align-items"
				}
				styles = append(styles, property+":"+value)
			}
		}
	}
	if len(styles) > 0 {
		attrs += fmt.Sprintf(` style="%s"`, html.EscapeString(strings.Join(styles, ";")))
	}
	return fmt.Sprintf("<div class=\"%s\"%s>\n\n%s\n\n</div>\n", strings.Join(classes, " "), attrs, c.Content), nil
}
