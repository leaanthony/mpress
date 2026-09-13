package components

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"path"
	"strconv"
	"strings"
)

const (
	defaultVideoWidth  = 1280
	defaultVideoHeight = 720
)

// Video renders an accessible native video player. Its compact source API is
// intentional: browsers own playback, while MPress supplies safe defaults and
// infers source types from filenames.
type Video struct {
	Meta map[string]string
}

func (v *Video) Parse(content string) error {
	if strings.TrimSpace(content) != "" {
		return fmt.Errorf("video is a leaf component and does not accept body content")
	}
	return nil
}

func (v *Video) Render() (string, error) {
	sources := mediaList(v.Meta["src"])
	if len(sources) == 0 {
		return "", fmt.Errorf("video component requires src")
	}
	base := strings.TrimSpace(v.Meta["base"])
	for index, source := range sources {
		resolved, err := resolveMediaURL(base, source)
		if err != nil {
			return "", fmt.Errorf("video source %q: %w", source, err)
		}
		sources[index] = resolved
	}

	width := positiveDimension(v.Meta["width"], defaultVideoWidth)
	height := positiveDimension(v.Meta["height"], defaultVideoHeight)
	poster, err := resolveOptionalMediaURL(base, v.Meta["poster"])
	if err != nil {
		return "", fmt.Errorf("video poster: %w", err)
	}
	transcript, err := resolveOptionalMediaURL(base, v.Meta["transcript"])
	if err != nil {
		return "", fmt.Errorf("video transcript: %w", err)
	}

	captionFiles := mediaList(v.Meta["captions"])
	for index, caption := range captionFiles {
		resolved, resolveErr := resolveMediaURL(base, caption)
		if resolveErr != nil {
			return "", fmt.Errorf("video captions %q: %w", caption, resolveErr)
		}
		captionFiles[index] = resolved
	}

	title := strings.TrimSpace(v.Meta["title"])
	defaultLanguage := strings.TrimSpace(v.Meta["lang"])
	if defaultLanguage == "" {
		defaultLanguage = "en"
	}

	var out strings.Builder
	out.WriteString(`<figure class="mpress-video">`)
	fmt.Fprintf(&out, `<video controls playsinline preload="metadata" width="%d" height="%d"`, width, height)
	if poster != "" {
		fmt.Fprintf(&out, ` poster="%s"`, html.EscapeString(poster))
	}
	if title != "" {
		fmt.Fprintf(&out, ` aria-label="%s"`, html.EscapeString(title))
	}
	out.WriteString(`>`)
	for _, source := range sources {
		fmt.Fprintf(&out, `<source src="%s"`, html.EscapeString(source))
		if mediaType := inferredVideoType(source); mediaType != "" {
			fmt.Fprintf(&out, ` type="%s"`, mediaType)
		}
		out.WriteString(`>`)
	}
	for index, caption := range captionFiles {
		language := inferredCaptionLanguage(caption)
		if language == "" {
			language = defaultLanguage
		}
		fmt.Fprintf(&out, `<track kind="captions" src="%s" srclang="%s" label="%s"`, html.EscapeString(caption), html.EscapeString(language), html.EscapeString(languageLabel(language)))
		if index == 0 {
			out.WriteString(` default`)
		}
		out.WriteString(`>`)
	}
	fmt.Fprintf(&out, `Your browser cannot play this video. <a href="%s">Open the video file</a>.`, html.EscapeString(sources[0]))
	out.WriteString(`</video>`)
	if title != "" || transcript != "" {
		out.WriteString(`<figcaption>`)
		if title != "" {
			out.WriteString(html.EscapeString(title))
		}
		if transcript != "" {
			if title != "" {
				out.WriteString(` · `)
			}
			fmt.Fprintf(&out, `<a href="%s">Transcript</a>`, html.EscapeString(transcript))
		}
		out.WriteString(`</figcaption>`)
	}
	out.WriteString(`</figure>`)
	return out.String(), nil
}

func mediaList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var decoded []string
	if strings.HasPrefix(raw, "[") && json.Unmarshal([]byte(raw), &decoded) == nil {
		return cleanMediaList(decoded)
	}
	return cleanMediaList(strings.Split(raw, ","))
}

func cleanMediaList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func positiveDimension(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func resolveOptionalMediaURL(base, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	return resolveMediaURL(base, value)
}

func resolveMediaURL(base, value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	if parsed.IsAbs() || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.TrimSpace(base) == "" {
		return value, nil
	}
	resolved := strings.TrimRight(strings.TrimSpace(base), "/") + "/" + strings.TrimLeft(value, "/")
	parsed, err = url.Parse(resolved)
	if err != nil {
		return "", fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}
	return resolved, nil
}

func inferredVideoType(source string) string {
	clean := source
	if index := strings.IndexAny(clean, "?#"); index >= 0 {
		clean = clean[:index]
	}
	switch strings.ToLower(path.Ext(clean)) {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ogv", ".ogg":
		return "video/ogg"
	case ".mov":
		return "video/quicktime"
	default:
		return ""
	}
}

func inferredCaptionLanguage(source string) string {
	clean := source
	if index := strings.IndexAny(clean, "?#"); index >= 0 {
		clean = clean[:index]
	}
	name := strings.TrimSuffix(path.Base(clean), path.Ext(clean))
	separator := strings.LastIndexByte(name, '.')
	if separator < 0 {
		return ""
	}
	candidate := name[separator+1:]
	parts := strings.Split(candidate, "-")
	if len(parts[0]) < 2 || len(parts[0]) > 3 {
		return ""
	}
	for _, part := range parts {
		if len(part) < 2 || len(part) > 8 {
			return ""
		}
		for _, r := range part {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
				return ""
			}
		}
	}
	return candidate
}

func languageLabel(language string) string {
	switch strings.ToLower(language) {
	case "en":
		return "English"
	default:
		return language
	}
}
