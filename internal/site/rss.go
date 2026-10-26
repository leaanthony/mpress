package site

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/routes"
)

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        string   `xml:"guid"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate,omitempty"`
	Categories  []string `xml:"category,omitempty"`
}

func rssOutputPath(cfg config.Config) (string, error) {
	value := strings.TrimSpace(cfg.Social.RSS)
	if value == "" {
		return "", nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid RSS URL: %w", err)
	}
	// External feeds are links, not output destinations.
	if parsed.IsAbs() || parsed.Host != "" {
		return "", nil
	}
	filename := strings.TrimPrefix(parsed.Path, "/")
	if err := routes.Output(filename); err != nil {
		return "", fmt.Errorf("invalid RSS output: %w", err)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("RSS output must be a file path without a query or fragment")
	}
	return filename, nil
}

func writeRSSFeed(output string, cfg config.Config, archives map[*content.Page]*blogArchiveData) error {
	filename, err := rssOutputPath(cfg)
	if err != nil || filename == "" || len(archives) == 0 {
		return err
	}
	if cfg.Site.BaseURL == "" {
		cfg.Site.BaseURL = "/"
	}
	channel := rssChannel{Title: cfg.Site.Title, Description: cfg.Site.Description, Language: cfg.Site.DefaultLanguage, Link: sitePageURL(cfg, cfg.Site.DefaultLanguage, "")}
	posts := make(map[string]blogPostData)
	for index, archive := range archives {
		if len(archives) == 1 {
			channel.Link = sitePageURL(cfg, cfg.Site.DefaultLanguage, index.URLPath)
		}
		for _, post := range archive.Posts {
			posts[post.Route] = post
		}
	}
	ordered := make([]blogPostData, 0, len(posts))
	for _, post := range posts {
		ordered = append(ordered, post)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := content.ParseDate(ordered[i].Date), content.ParseDate(ordered[j].Date)
		if !left.Equal(right) {
			return left.After(right)
		}
		return ordered[i].Route > ordered[j].Route
	})
	for _, post := range ordered {
		link := sitePageURL(cfg, cfg.Site.DefaultLanguage, post.Route)
		item := rssItem{Title: post.Title, Link: link, GUID: link, Description: post.Excerpt, Categories: post.Tags}
		if date := content.ParseDate(post.Date); !date.IsZero() {
			item.PubDate = date.Format(time.RFC1123Z)
		}
		channel.Items = append(channel.Items, item)
	}
	data, err := xml.MarshalIndent(rssDocument{Version: "2.0", Channel: channel}, "", "  ")
	if err != nil {
		return err
	}
	destination := filepath.Join(output, filepath.FromSlash(filename))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, append([]byte(xml.Header), append(data, '\n')...), 0o644)
}

func writeLanguageRSSFeed(output string, cfg config.Config, lang string, archives map[*content.Page]*blogArchiveData, collector *check.Collector) error {
	if lang != cfg.Site.DefaultLanguage {
		return nil
	}
	if err := writeRSSFeed(output, cfg, archives); err != nil {
		return err
	}
	if collector != nil && len(archives) > 0 {
		if filename, _ := rssOutputPath(cfg); filename != "" {
			collector.AddFile(filename)
		}
	}
	return nil
}

func registerRSSOutput(registry outputRegistry, cfg config.Config) error {
	filename, err := rssOutputPath(cfg)
	if err != nil || filename == "" {
		return err
	}
	for _, generated := range generatedOutputFiles(cfg) {
		if filename == generated {
			return fmt.Errorf("RSS output collision with generated file %q", filename)
		}
	}
	return registry.add(outputEntry{name: filename, owner: "RSS feed"})
}
