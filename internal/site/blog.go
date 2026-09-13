package site

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
)

type blogPostData struct {
	Title           string
	Date            string
	DateDisplay     string
	Author          string
	Tags            []string
	Excerpt         string
	Route           string
	Image           string
	ImageMode       string
	ImageFit        string
	ImageBackground string
	ImageWidth      int
	ShowTags        bool
	HeadingSize     string
	ReadingTime     int
}

type blogArchiveData struct {
	Posts     []blogPostData
	Tags      []string
	MoreCount int
}

type blogPeriodData struct {
	Label string
	Posts []blogPostData
}

type blogArticleData struct {
	IndexRoute string
	Periods    []blogPeriodData
	Post       blogPostData
}

func blogArchives(pages []*content.Page, preferences config.BlogConfig) (map[*content.Page]*blogArchiveData, map[*content.Page]*blogArticleData) {
	archives := make(map[*content.Page]*blogArchiveData)
	articles := make(map[*content.Page]*blogArticleData)
	for _, index := range pages {
		if index.Layout != "blog-index" {
			continue
		}
		prefix := strings.TrimSuffix(index.URLPath, "/")
		if prefix != "" {
			prefix += "/"
		}
		archive := &blogArchiveData{}
		tags := make(map[string]bool)
		postPages := make(map[string]*content.Page)
		for _, page := range pages {
			if page == index || page.Layout == "blog-index" || !strings.HasPrefix(page.URLPath, prefix) {
				continue
			}
			remainder := strings.TrimPrefix(page.URLPath, prefix)
			if remainder == "" {
				continue
			}
			author := strings.TrimSpace(page.Meta.Author)
			if author == "" {
				author = strings.Join(page.Meta.Authors, ", ")
			}
			excerpt := strings.TrimSpace(page.Description)
			if excerpt == "" {
				excerpt = blogExcerpt(page.PlainText, 220)
			}
			wordCount := len(strings.Fields(page.PlainText))
			readingTime := (wordCount + 219) / 220
			if readingTime < 1 {
				readingTime = 1
			}
			dateDisplay := page.Meta.Date
			if date := content.ParseDate(page.Meta.Date); !date.IsZero() {
				dateDisplay = date.Format("2 January 2006")
			}
			imageMode := blogImageMode(page.Meta.ImageMode, preferences.ImageMode)
			imageBackground := blogImageBackground(page.Meta.ImageBackground)
			if imageBackground == "" {
				imageBackground = blogImageBackground(preferences.ImageBackground)
			}
			if imageMode == "floating" {
				imageBackground = ""
			}
			post := blogPostData{
				Title: page.Title, Date: page.Meta.Date, DateDisplay: dateDisplay,
				Author: author, Tags: page.Meta.Tags, Excerpt: excerpt,
				Route: page.URLPath, Image: page.Meta.Image, ImageMode: imageMode, ImageFit: blogImageFit(page.Meta.ImageFit, preferences.ImageFit), ImageBackground: imageBackground, ImageWidth: blogImageWidth(page.Meta.ImageWidth, preferences.ImageWidth),
				ShowTags: blogShowTags(page.Meta.ShowTags, preferences.ShowTags), HeadingSize: blogHeadingSize(page.Meta.HeadingSize, preferences.HeadingSize), ReadingTime: readingTime,
			}
			archive.Posts = append(archive.Posts, post)
			postPages[page.URLPath] = page
			for _, tag := range page.Meta.Tags {
				if tag = strings.TrimSpace(tag); tag != "" {
					tags[tag] = true
				}
			}
		}
		sort.SliceStable(archive.Posts, func(i, j int) bool {
			iDate := content.ParseDate(archive.Posts[i].Date)
			jDate := content.ParseDate(archive.Posts[j].Date)
			if !iDate.Equal(jDate) {
				return iDate.After(jDate)
			}
			return archive.Posts[i].Route > archive.Posts[j].Route
		})
		if len(archive.Posts) > 1 {
			archive.MoreCount = len(archive.Posts) - 1
		}
		for tag := range tags {
			archive.Tags = append(archive.Tags, tag)
		}
		sort.Strings(archive.Tags)
		archives[index] = archive

		periods := blogPeriods(archive.Posts)
		for route, page := range postPages {
			if _, exists := articles[page]; exists {
				continue
			}
			article := &blogArticleData{IndexRoute: index.URLPath, Periods: periods}
			for _, post := range archive.Posts {
				if post.Route == route {
					article.Post = post
					break
				}
			}
			articles[page] = article
		}
	}
	return archives, articles
}

func blogPeriods(posts []blogPostData) []blogPeriodData {
	var periods []blogPeriodData
	for _, post := range posts {
		label := "Undated"
		if date := content.ParseDate(post.Date); !date.IsZero() {
			label = date.Format("2006")
		}
		if len(periods) == 0 || periods[len(periods)-1].Label != label {
			periods = append(periods, blogPeriodData{Label: label})
		}
		periods[len(periods)-1].Posts = append(periods[len(periods)-1].Posts, post)
	}
	return periods
}

func blogShowTags(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func blogHeadingSize(value, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "compact":
		return "compact"
	case "large":
		return "large"
	default:
		if fallback == "compact" || fallback == "large" {
			return fallback
		}
		return "default"
	}
}

func blogImageFit(value, fallback string) string {
	if strings.EqualFold(strings.TrimSpace(value), "contain") {
		return "contain"
	}
	if fallback == "contain" {
		return "contain"
	}
	return "cover"
}

func blogImageMode(value, fallback string) string {
	if strings.EqualFold(strings.TrimSpace(value), "floating") {
		return "floating"
	}
	if fallback == "floating" {
		return "floating"
	}
	return "panel"
}

func blogImageWidth(value, fallback int) int {
	if value >= 50 && value <= 100 {
		return value
	}
	if fallback >= 50 && fallback <= 100 {
		return fallback
	}
	return 100
}

func blogImageBackground(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 7 || value[0] != '#' {
		return ""
	}
	for _, character := range value[1:] {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return ""
		}
	}
	return value
}

func blogExcerpt(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	cut := strings.TrimSpace(string(runes[:limit]))
	if lastSpace := strings.LastIndex(cut, " "); lastSpace > limit/2 {
		cut = cut[:lastSpace]
	}
	return strings.TrimSpace(cut) + "…"
}
