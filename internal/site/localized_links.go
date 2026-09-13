package site

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"golang.org/x/net/html"
)

// Localize rendered links, keeping the authoritative source and translation
// placeholders unchanged. Only links to an existing translated page are moved.
func localizePageLinks(cfg config.Config, page *content.Page, sourceRoutes, targetRoutes map[string]*content.Page) (string, error) {
	if page.Language == cfg.Site.DefaultLanguage || len(sourceRoutes) == 0 {
		return page.HTML, nil
	}
	if cfg.Site.BaseURL == "" {
		cfg.Site.BaseURL = "https://mpress.invalid"
	}
	base, err := url.Parse(sitePageURL(cfg, cfg.Site.DefaultLanguage, page.URLPath))
	if err != nil {
		return "", err
	}
	tokenizer := html.NewTokenizer(strings.NewReader(page.HTML))
	var output strings.Builder
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			if tokenizer.Err() != io.EOF {
				return "", tokenizer.Err()
			}
			break
		}
		raw := string(tokenizer.Raw())
		if kind == html.StartTagToken {
			token := tokenizer.Token()
			if token.Data == "a" {
				for index, attr := range token.Attr {
					if attr.Key != "href" {
						continue
					}
					localized, linkErr := localizedDocumentLink(cfg, page.Language, base, attr.Val, sourceRoutes, targetRoutes)
					if linkErr != nil {
						return "", linkErr
					}
					if localized != attr.Val {
						token.Attr[index].Val = localized
						raw = token.String()
					}
				}
			}
		}
		output.WriteString(raw)
	}
	return output.String(), nil
}

func localizedDocumentLink(cfg config.Config, language string, base *url.URL, link string, sourceRoutes, targetRoutes map[string]*content.Page) (string, error) {
	if link == "" || strings.HasPrefix(link, "#") {
		return link, nil
	}
	parsed, err := url.Parse(link)
	if err != nil {
		return link, nil
	}
	resolved := base.ResolveReference(parsed)
	if resolved.Host != base.Host || resolved.Scheme != base.Scheme {
		return link, nil
	}
	siteRoot, err := url.Parse(cfg.Site.BaseURL)
	if err != nil {
		return link, nil
	}
	root := strings.TrimRight(siteRoot.Path, "/")
	if root != "" && resolved.Path != root && !strings.HasPrefix(resolved.Path, root+"/") {
		return link, nil
	}
	route := strings.TrimPrefix(resolved.Path, root)
	route = strings.Trim(route, "/")
	if !cfg.Site.DefaultAtRoot {
		if route == cfg.Site.DefaultLanguage {
			route = ""
		} else {
			route = strings.TrimPrefix(route, cfg.Site.DefaultLanguage+"/")
		}
	}
	source := sourceRoutes[route]
	if source == nil {
		return link, nil
	}
	target := targetRoutes[source.URLPath]
	if target == nil {
		return link, nil
	}
	destination, err := url.Parse(sitePageURL(cfg, language, target.URLPath))
	if err != nil {
		return "", err
	}
	destination.RawQuery = parsed.RawQuery
	destination.Fragment, err = translatedHeadingFragment(parsed.Fragment, source, target)
	if err != nil {
		return "", fmt.Errorf("localize %s link %q: %w", language, link, err)
	}
	if !parsed.IsAbs() && parsed.Host == "" {
		destination.Scheme = ""
		destination.Host = ""
	}
	return destination.String(), nil
}

func translatedHeadingFragment(fragment string, source, target *content.Page) (string, error) {
	if fragment == "" {
		return "", nil
	}
	for index, heading := range source.Headings {
		if heading.ID != fragment {
			continue
		}
		if len(source.Headings) != len(target.Headings) || target.Headings[index].Level != heading.Level {
			return "", fmt.Errorf("cannot align translated heading %q in %s", fragment, target.URLPath)
		}
		return target.Headings[index].ID, nil
	}
	// Explicit HTML anchors retain their identifiers across translations.
	return fragment, nil
}
