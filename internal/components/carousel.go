package components

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"regexp"
	"strings"
)

// Carousel presents a small set of Markdown-authored panels one at a time.
// Slides are separated by a Markdown thematic break. Each slide can contain
// normal Markdown and M-Press components.
type Carousel struct {
	Meta    map[string]string
	Content string
}

var carouselSeparatorRE = regexp.MustCompile(`(?m)^\s*---\s*$`)

func (c *Carousel) Parse(content string) error {
	c.Content = content
	return nil
}

func (c *Carousel) Render() (string, error) {
	var slides []string
	for _, source := range carouselSeparatorRE.Split(strings.TrimSpace(c.Content), -1) {
		if source = strings.TrimSpace(source); source != "" {
			slides = append(slides, source)
		}
	}
	if len(slides) == 0 {
		return "", fmt.Errorf("carousel requires at least one slide")
	}

	label := strings.TrimSpace(c.Meta["label"])
	if label == "" {
		label = "Carousel"
	}
	digest := sha256.Sum256([]byte(label + "\x00" + c.Content))
	id := "mpress-carousel-" + hex.EncodeToString(digest[:5])

	var out strings.Builder
	fmt.Fprintf(&out, `<section id="%s" class="mpress-carousel" aria-roledescription="carousel" aria-label="%s" data-carousel>`, id, html.EscapeString(label))
	out.WriteString(`<div class="mpress-carousel-viewport" aria-live="polite">`)
	for index, slide := range slides {
		hidden := ""
		if index > 0 {
			hidden = ` hidden`
		}
		fmt.Fprintf(&out, `<article class="mpress-carousel-slide" data-carousel-slide aria-label="%d of %d"%s>%s</article>`, index+1, len(slides), hidden, renderMarkdownFragment(slide))
	}
	out.WriteString(`</div>`)
	if len(slides) > 1 {
		out.WriteString(`<footer class="mpress-carousel-controls"><button type="button" data-carousel-previous aria-label="Previous slide">`)
		out.WriteString(lucide("chevron-left", 16))
		out.WriteString(`</button><span data-carousel-status>1 / `)
		out.WriteString(fmt.Sprint(len(slides)))
		out.WriteString(`</span><div class="mpress-carousel-dots" role="group" aria-label="Choose a slide">`)
		for index := range slides {
			current := ""
			if index == 0 {
				current = ` aria-current="true"`
			}
			fmt.Fprintf(&out, `<button type="button" data-carousel-dot="%d" aria-label="Show slide %d"%s></button>`, index, index+1, current)
		}
		out.WriteString(`</div><button type="button" data-carousel-next aria-label="Next slide">`)
		out.WriteString(lucide("chevron-right", 16))
		out.WriteString(`</button></footer>`)
	}
	out.WriteString(`</section>`)
	return out.String(), nil
}
