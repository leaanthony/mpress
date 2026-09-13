package components

import (
	"fmt"
	"html"
	"strings"
)

// Button renders the canonical fenced button component. The older
// @button[Label](href) spelling remains supported by processNativeSyntax.
type Button struct {
	Meta    map[string]string
	Content string
}

func (b *Button) Parse(content string) error {
	b.Content = strings.TrimSpace(content)
	return nil
}

func (b *Button) Render() (string, error) {
	label := strings.TrimSpace(b.Meta["label"])
	if label == "" {
		label = b.Content
	}
	if label == "" {
		return "", fmt.Errorf("button component requires body content or a label attribute")
	}
	href := strings.TrimSpace(b.Meta["href"])
	if href == "" {
		return "", fmt.Errorf("button component requires an href attribute")
	}
	variant := safeToken(b.Meta["variant"])
	if variant == "" {
		variant = "secondary"
	}
	return renderButton(label, href, variant, b.Meta), nil
}

// ThemeImage renders one image for light mode and another for dark mode. A
// single light or dark source is also valid and becomes an ordinary image.
type ThemeImage struct {
	Meta map[string]string
}

func (i *ThemeImage) Parse(string) error { return nil }

func (i *ThemeImage) Render() (string, error) {
	return renderThemeImage(i.Meta)
}

func renderThemeImage(meta map[string]string) (string, error) {
	alt := html.EscapeString(strings.TrimSpace(meta["alt"]))
	source := html.EscapeString(strings.TrimSpace(meta["src"]))
	light := html.EscapeString(strings.TrimSpace(meta["light"]))
	dark := html.EscapeString(strings.TrimSpace(meta["dark"]))
	if light == "" {
		light = source
	}
	var image string
	if light != "" && dark != "" {
		image = fmt.Sprintf(`<span class="mpress-theme-image"><img class="mpress-theme-image-light" src="%s" alt="%s" loading="lazy"><img class="mpress-theme-image-dark" src="%s" alt="%s" loading="lazy"></span>`, light, alt, dark, alt)
	} else {
		if light == "" {
			light = dark
		}
		if light == "" {
			return "", fmt.Errorf("image component requires a src, light, or dark source")
		}
		image = fmt.Sprintf(`<img src="%s" alt="%s" loading="lazy">`, light, alt)
	}
	if meta["expand"] != "true" {
		return image, nil
	}
	label := "Expand image"
	if strings.TrimSpace(meta["alt"]) != "" {
		label += ": " + strings.TrimSpace(meta["alt"])
	}
	return fmt.Sprintf(`<button class="mpress-image-expand" type="button" aria-haspopup="dialog" aria-label="%s"><span class="mpress-image-expand-content">%s</span></button>`, html.EscapeString(label), image), nil
}
