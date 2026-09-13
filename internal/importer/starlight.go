package importer

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/leaanthony/mpress/internal/content"
)

// ImportStarlight converts a Starlight/Astro docs project to M-Press format.
// Imports from src/content/docs/ (Starlight default), converts MDX to Markdown,
// and maps Starlight-specific frontmatter to M-Press equivalents.
func ImportStarlight(sourcePath, outputDir string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("source path not found: %s", sourcePath)
	}
	sourceDir := sourcePath
	if !info.IsDir() {
		sourceDir = filepath.Dir(sourcePath)
	}

	// Find content directory: src/content/docs/ (Starlight) or docs/
	contentDir := ""
	for _, candidate := range []string{
		filepath.Join("src", "content", "docs"),
		"docs",
		"src",
	} {
		dir := filepath.Join(sourceDir, candidate)
		if _, err := os.Stat(dir); err == nil {
			hasMD := false
			filepath.Walk(dir, func(p string, fi os.FileInfo, e error) error {
				if e == nil && !fi.IsDir() && (strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".mdx")) {
					hasMD = true
					return filepath.SkipAll
				}
				return nil
			})
			if hasMD {
				contentDir = dir
				break
			}
		}
	}
	if contentDir == "" {
		return fmt.Errorf("no content found in %s (checked src/content/docs/, docs/)", sourceDir)
	}
	// Extract config
	siteConfig := parseStarlightProjectConfig(sourceDir)
	siteConfig = augmentStarlightLanguagesFromContent(siteConfig, contentDir)
	title := siteConfig.Title

	// Create output
	mpressContent := filepath.Join(outputDir, "content")
	os.MkdirAll(mpressContent, 0o755)

	count := 0
	var findings []starlightMigrationFinding
	var pages []starlightImportPage
	err = filepath.Walk(contentDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdx") {
			return nil
		}

		rel, _ := filepath.Rel(contentDir, path)
		if isStarlightPrivateContent(rel) {
			findings = append(findings, starlightMigrationFinding{
				File:    rel,
				Pattern: "private-content",
				Detail:  "Skipped underscore-prefixed Starlight content that is not normally routed.",
			})
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		source := string(data)
		locale := starlightLocaleFromRel(rel, siteConfig.Languages)
		pages = append(pages, starlightImportPage{
			SourceRel: rel,
			OutputRel: starlightOutputRel(rel, source, siteConfig.Languages),
			Source:    source,
			Locale:    locale,
			Modified:  fi.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("scanning Starlight content: %w", err)
	}

	pages = appendStarlightSectionIndexes(pages)
	pages = dedupeStarlightPages(pages, &findings)
	routes := buildStarlightRouteMap(pages)
	sidebar, sidebarFindings := parseStarlightSidebar(sourceDir, routes)
	findings = append(findings, sidebarFindings...)
	convertedPages := make([]starlightConvertedPage, 0, len(pages))
	for _, page := range pages {
		findings = append(findings, detectStarlightMigrationFindings(page.SourceRel, page.Source)...)
		converted := convertStarlightContentWithRoutes(page.Source, starlightMPressRoute(page), routes)
		if strings.Contains(converted, "\ntemplate: splash\n") {
			converted = convertStarlightLandingTerminals(converted)
		}
		converted = ensureStarlightExplicitSlug(converted, page)
		if strings.HasPrefix(filepath.ToSlash(page.OutputRel), "blog/") {
			converted = ensureStarlightBlogImage(converted)
		}
		converted = ensureStarlightSourcePath(converted, page.SourceRel)
		convertedPages = append(convertedPages, starlightConvertedPage{Page: page, Content: converted})
	}

	anchors := starlightAnchorMap(convertedPages)
	for _, convertedPage := range convertedPages {
		page := convertedPage.Page
		converted := rewriteInvalidStarlightFragments(convertedPage.Content, starlightMPressRoute(page), routes, anchors, page.SourceRel, &findings)
		converted, err = markdownToMPD(converted)
		if err != nil {
			return fmt.Errorf("convert %s to MPD: %w", page.SourceRel, err)
		}
		converted = normaliseImportedHeadingLevels(converted)

		outPath := filepath.Join(mpressContent, page.OutputRel)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, []byte(converted), 0o644); err != nil {
			return err
		}
		if !page.Modified.IsZero() {
			if err := os.Chtimes(outPath, page.Modified, page.Modified); err != nil {
				return fmt.Errorf("preserve modification time for %s: %w", page.SourceRel, err)
			}
		}
		count++
	}
	if len(sidebar) > 0 {
		if err := os.WriteFile(filepath.Join(mpressContent, "_nav.yaml"), []byte(renderStarlightNavYAML(sidebar)), 0o644); err != nil {
			return err
		}
	}

	// Copy public/ assets
	staticAssets := 0
	publicDir := filepath.Join(sourceDir, "public")
	if _, err := os.Stat(publicDir); err == nil {
		copied, err := copyTree(publicDir, filepath.Join(outputDir, "static"))
		if err != nil {
			return err
		}
		staticAssets += copied
	}
	sourceAssetsDir := filepath.Join(sourceDir, "src", "assets")
	if _, err := os.Stat(sourceAssetsDir); err == nil {
		copied, err := copyTree(sourceAssetsDir, filepath.Join(outputDir, "static", "assets"))
		if err != nil {
			return err
		}
		staticAssets += copied
	}
	if siteConfig.RSS != "" {
		rssPath := strings.TrimPrefix(siteConfig.RSS, "/")
		rssOutput := filepath.Join(outputDir, "static", filepath.FromSlash(rssPath))
		if err := os.MkdirAll(filepath.Dir(rssOutput), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(rssOutput, []byte(starlightRSSDocument(siteConfig, title)), 0o644); err != nil {
			return err
		}
		staticAssets++
	}

	customCSS := starlightCustomCSS(sourceDir)
	if err := os.WriteFile(filepath.Join(outputDir, "custom.css"), []byte(customCSS), 0o644); err != nil {
		return err
	}

	// Generate config
	if title == "" {
		title = inferStarlightSiteTitle(sourceDir)
	}
	if len(siteConfig.Languages) == 0 {
		siteConfig.Languages = []string{"en"}
	}
	configYAML := renderStarlightMPressConfig(siteConfig, title)
	os.WriteFile(filepath.Join(outputDir, "mpress.yaml"), []byte(configYAML), 0o644)

	if err := writeStarlightMigrationReport(outputDir, starlightMigrationReport{
		SourceDir:    sourceDir,
		ContentDir:   contentDir,
		Pages:        count,
		StaticAssets: staticAssets,
		Findings:     findings,
	}); err != nil {
		return err
	}

	fmt.Printf("  Imported %d pages from Starlight\n", count)
	return nil
}

func normaliseImportedHeadingLevels(source string) string {
	lines := strings.Split(source, "\n")
	previous := 1 // The generated page shell supplies the document H1.
	inFence := false
	fence := ""
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence, fence = true, marker
			} else if marker == fence {
				inFence, fence = false, ""
			}
			continue
		}
		if inFence {
			continue
		}
		leading := len(line) - len(strings.TrimLeft(line, " \t"))
		heading := line[leading:]
		level := 0
		for level < len(heading) && level < 6 && heading[level] == '#' {
			level++
		}
		if level == 0 || len(heading) <= level || heading[level] != ' ' {
			continue
		}
		originalLevel := level
		if level == 1 {
			level = 2
			lines[index] = line[:leading] + "##" + heading[originalLevel:]
		} else if level > previous+1 {
			level = previous + 1
			lines[index] = line[:leading] + strings.Repeat("#", level) + heading[originalLevel:]
		}
		previous = level
	}
	return strings.Join(lines, "\n")
}

func ensureStarlightSourcePath(source, sourceRel string) string {
	extension := strings.ToLower(filepath.Ext(sourceRel))
	if extension != ".md" && extension != ".mdx" {
		return source
	}
	path := filepath.ToSlash(filepath.Clean(sourceRel))
	if path == "." || path == ".." || strings.HasPrefix(path, "../") || strings.HasPrefix(path, "/") {
		return source
	}
	return upsertStarlightFrontmatterField(source, "sourcePath", yamlQuote(path))
}

func upsertStarlightFrontmatterField(source, key, value string) string {
	field := key + ": " + value
	if !strings.HasPrefix(source, "---\n") {
		return "---\n" + field + "\n---\n\n" + source
	}
	endOffset := strings.Index(source[4:], "\n---")
	if endOffset < 0 {
		return source
	}
	end := 4 + endOffset
	lines := strings.Split(source[4:end], "\n")
	prefix := key + ":"
	for index, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[index] = field
			return source[:4] + strings.Join(lines, "\n") + source[end:]
		}
	}
	return source[:end] + "\n" + field + source[end:]
}

func starlightRSSDocument(siteConfig starlightProjectConfig, title string) string {
	baseURL := strings.TrimRight(siteConfig.BaseURL, "/")
	blogURL := baseURL + "/blog/"
	description := siteConfig.Description
	if description == "" {
		description = title + " updates"
	}
	return fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rss version=\"2.0\"><channel><title>%s</title><link>%s</link><description>%s</description></channel></rss>\n", html.EscapeString(title), html.EscapeString(blogURL), html.EscapeString(description))
}

func ensureStarlightExplicitSlug(content string, page starlightImportPage) string {
	slug := starlightLogicalSlug(page)
	if slug == "" {
		return content
	}
	if strings.HasPrefix(content, "---\n") {
		if end := strings.Index(content[4:], "\n---\n"); end >= 0 {
			insert := 4 + end
			return content[:insert] + "\nslug: " + slug + content[insert:]
		}
	}
	return "---\nslug: " + slug + "\n---\n\n" + content
}

func starlightLogicalSlug(page starlightImportPage) string {
	slug := strings.Trim(starlightRouteFromOutputRel(page.OutputRel), "/")
	if page.Locale != "" {
		locale := strings.Trim(page.Locale, "/")
		if slug == locale {
			return ""
		}
		slug = strings.TrimPrefix(slug, locale+"/")
	}
	return slug
}

var slTitleRegex = regexp.MustCompile(`title\s*:\s*['"]([^'"]*)['"]`)
var slSiteRegex = regexp.MustCompile(`site\s*:\s*['"]([^'"]+)['"]`)
var slDescriptionRegex = regexp.MustCompile(`description\s*:\s*['"]([^'"]+)['"]`)
var slFaviconRegex = regexp.MustCompile(`favicon\s*:\s*['"]([^'"]+)['"]`)
var slLogoLightRegex = regexp.MustCompile(`light\s*:\s*['"]([^'"]+)['"]`)
var slLogoDarkRegex = regexp.MustCompile(`dark\s*:\s*['"]([^'"]+)['"]`)
var slEditBaseURLRegex = regexp.MustCompile(`baseUrl\s*:\s*['"]([^'"]+)['"]`)
var slSocialRegex = regexp.MustCompile(`\{\s*icon:\s*['"]([^'"]+)['"]\s*,\s*label:\s*['"][^'"]+['"]\s*,\s*href:\s*['"]([^'"]+)['"]\s*\}`)
var slDefaultLocaleRegex = regexp.MustCompile(`defaultLocale\s*:\s*['"]([^'"]+)['"]`)
var slLocaleEntryRegex = regexp.MustCompile(`["']?([A-Za-z0-9_-]+)["']?\s*:\s*\{\s*label:\s*["'][^"']+["'][^}]*lang:\s*["']([^"']+)["']`)
var slLocaleLabelEntryRegex = regexp.MustCompile(`["']?([A-Za-z0-9_-]+)["']?\s*:\s*\{\s*label:\s*["']([^"']+)["'][^}]*lang:\s*["']([^"']+)["']`)
var slSlugRegex = regexp.MustCompile(`(?m)^slug:\s*["']?([^"'\n]+)["']?\s*$`)

type starlightProjectConfig struct {
	Title           string
	BlogTitle       string
	Description     string
	BaseURL         string
	DefaultLanguage string
	Languages       []string
	LanguageLabels  map[string]string
	LogoLight       string
	LogoDark        string
	Favicon         string
	GitHub          string
	Discord         string
	Reddit          string
	X               string
	RSS             string
	Sponsor         string
	EditURL         string
	HasBlog         bool
}

type starlightSidebarItem struct {
	Label        string
	Link         string
	Directory    string
	Collapsed    *bool
	BadgeText    string
	BadgeVariant string
	Children     []starlightSidebarItem
	External     bool
	Unresolved   bool
}

func parseStarlightConfig(sourceDir string) string {
	return parseStarlightProjectConfig(sourceDir).Title
}

func starlightPluginCall(text, marker string) string {
	start := strings.Index(text, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	depth := 1
	inString := rune(0)
	escaped := false
	for i, r := range text[start:] {
		if inString != 0 {
			if escaped {
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == inString {
				inString = 0
			}
			continue
		}
		switch r {
		case '\'', '"', '`':
			inString = r
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[start : start+i]
			}
		}
	}
	return text[start:]
}

func parseStarlightProjectConfig(sourceDir string) starlightProjectConfig {
	cfg := starlightProjectConfig{
		DefaultLanguage: "en",
		Languages:       []string{"en"},
		LanguageLabels:  map[string]string{},
	}
	for _, name := range []string{"astro.config.mjs", "astro.config.ts", "astro.config.js"} {
		data, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			continue
		}
		text := string(data)
		if strings.Contains(text, "starlightBlog(") {
			cfg.HasBlog = true
			if blogBlock := starlightPluginCall(text, "starlightBlog("); blogBlock != "" {
				if m := slTitleRegex.FindStringSubmatch(blogBlock); len(m) > 1 {
					cfg.BlogTitle = strings.TrimSpace(m[1])
				}
			}
		}
		starlightBlock := text
		if starlightStart := strings.Index(text, "starlight({"); starlightStart >= 0 {
			end := len(text)
			if blogStart := strings.Index(text[starlightStart:], "starlightBlog("); blogStart >= 0 {
				end = starlightStart + blogStart
			}
			starlightBlock = text[starlightStart:end]
			if m := slTitleRegex.FindStringSubmatch(starlightBlock); len(m) > 1 {
				cfg.Title = strings.TrimSpace(m[1])
			}
		}
		if cfg.Title == "" {
			if m := slTitleRegex.FindStringSubmatch(text); len(m) > 1 {
				cfg.Title = strings.TrimSpace(m[1])
			}
		}
		if m := slSiteRegex.FindStringSubmatch(text); len(m) > 1 {
			cfg.BaseURL = strings.TrimSpace(m[1])
		}
		if m := slDescriptionRegex.FindStringSubmatch(starlightBlock); len(m) > 1 {
			cfg.Description = strings.TrimSpace(m[1])
		}
		if m := slFaviconRegex.FindStringSubmatch(starlightBlock); len(m) > 1 {
			cfg.Favicon = strings.TrimPrefix(starlightAssetPath(m[1]), "/")
		}
		if m := slLogoLightRegex.FindStringSubmatch(starlightBlock); len(m) > 1 {
			cfg.LogoLight = strings.TrimPrefix(starlightAssetPath(m[1]), "/")
		}
		if m := slLogoDarkRegex.FindStringSubmatch(starlightBlock); len(m) > 1 {
			cfg.LogoDark = strings.TrimPrefix(starlightAssetPath(m[1]), "/")
		}
		if m := slEditBaseURLRegex.FindStringSubmatch(starlightBlock); len(m) > 1 {
			cfg.EditURL = strings.TrimSpace(m[1])
		}
		for _, m := range slSocialRegex.FindAllStringSubmatch(starlightBlock, -1) {
			if len(m) != 3 {
				continue
			}
			switch strings.ToLower(m[1]) {
			case "github":
				cfg.GitHub = m[2]
			case "discord":
				cfg.Discord = m[2]
			case "reddit":
				cfg.Reddit = m[2]
			case "x", "x.com", "twitter":
				cfg.X = m[2]
			case "rss":
				cfg.RSS = m[2]
			}
		}
		if cfg.HasBlog && cfg.RSS == "" {
			cfg.RSS = "/blog/rss.xml"
		}
		if m := regexp.MustCompile(`href\s*=\s*['"](https://github\.com/sponsors/[^'"]+)['"]`).FindStringSubmatch(starlightBlock); len(m) > 1 {
			cfg.Sponsor = m[1]
		}
		cfg.DefaultLanguage, cfg.Languages, cfg.LanguageLabels = parseStarlightLanguages(starlightBlock)
		if _, err := os.Stat(filepath.Join(sourceDir, "src", "content", "docs", "blog")); err == nil {
			cfg.HasBlog = true
		}
		if cfg.Title != "" || cfg.Description != "" || cfg.BaseURL != "" {
			return cfg
		}
	}
	return cfg
}

func parseStarlightLanguages(configText string) (string, []string, map[string]string) {
	defaultLocale := "en"
	if m := slDefaultLocaleRegex.FindStringSubmatch(configText); len(m) > 1 && m[1] != "root" {
		defaultLocale = strings.ToLower(m[1])
	}
	var languages []string
	seen := map[string]bool{}
	labels := map[string]string{}
	for _, m := range slLocaleLabelEntryRegex.FindAllStringSubmatch(configText, -1) {
		if len(m) != 4 {
			continue
		}
		code := strings.ToLower(m[1])
		if code == "root" {
			code = strings.ToLower(m[3])
		}
		labels[code] = m[2]
	}
	for _, m := range slLocaleEntryRegex.FindAllStringSubmatch(configText, -1) {
		if len(m) != 3 {
			continue
		}
		code := strings.ToLower(m[1])
		if code == "root" {
			code = strings.ToLower(m[2])
		}
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		languages = append(languages, code)
	}
	if len(languages) == 0 {
		return defaultLocale, []string{defaultLocale}, labels
	}
	if !seen[defaultLocale] {
		languages = append([]string{defaultLocale}, languages...)
	}
	return defaultLocale, languages, labels
}

func augmentStarlightLanguagesFromContent(cfg starlightProjectConfig, contentDir string) starlightProjectConfig {
	if cfg.LanguageLabels == nil {
		cfg.LanguageLabels = map[string]string{}
	}
	known := []struct{ Code, Label string }{
		{"zh-cn", "简体中文"}, {"zh-tw", "繁體中文"}, {"ja", "日本語"},
		{"ko", "한국어"}, {"ru", "Русский"}, {"fr", "Français"},
		{"pt", "Português (Brasil)"}, {"de", "Deutsch"}, {"id", "Bahasa Indonesia"},
		{"es", "Español"}, {"it", "Italiano"}, {"nl", "Nederlands"},
	}
	seen := map[string]bool{}
	for _, language := range cfg.Languages {
		seen[strings.ToLower(language)] = true
	}
	for _, language := range known {
		dir := filepath.Join(contentDir, language.Code)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			continue
		}
		if !seen[language.Code] {
			cfg.Languages = append(cfg.Languages, language.Code)
			seen[language.Code] = true
		}
		if cfg.LanguageLabels[language.Code] == "" {
			cfg.LanguageLabels[language.Code] = language.Label
		}
	}
	if cfg.LanguageLabels[cfg.DefaultLanguage] == "" {
		cfg.LanguageLabels[cfg.DefaultLanguage] = strings.ToUpper(cfg.DefaultLanguage)
		if cfg.DefaultLanguage == "en" {
			cfg.LanguageLabels[cfg.DefaultLanguage] = "English"
		}
	}
	return cfg
}

func parseStarlightSidebar(sourceDir string, routes map[string]string) ([]starlightSidebarItem, []starlightMigrationFinding) {
	var findings []starlightMigrationFinding
	for _, name := range []string{"astro.config.mjs", "astro.config.ts", "astro.config.js"} {
		data, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			continue
		}
		arrayText, ok := extractStarlightSidebarArray(string(data))
		if !ok {
			continue
		}
		parser := starlightJSParser{input: arrayText}
		items := parser.parseSidebarArray()
		if len(parser.errors) > 0 {
			findings = append(findings, starlightMigrationFinding{
				File:    name,
				Pattern: "sidebar-parse",
				Detail:  strings.Join(parser.errors, "; "),
			})
		}
		converted := resolveStarlightSidebarItems(items, routes, &findings, name)
		if len(converted) > 0 {
			return converted, findings
		}
	}
	return nil, findings
}

func extractStarlightSidebarArray(text string) (string, bool) {
	sidebarAt := strings.Index(text, "sidebar")
	for sidebarAt >= 0 {
		beforeOK := sidebarAt == 0 || !isStarlightIdentByte(text[sidebarAt-1])
		after := sidebarAt + len("sidebar")
		afterOK := after >= len(text) || !isStarlightIdentByte(text[after])
		if beforeOK && afterOK {
			colon := strings.Index(text[after:], ":")
			if colon >= 0 {
				arrayStart := after + colon + 1 + strings.Index(text[after+colon+1:], "[")
				if arrayStart >= after+colon+1 {
					if arrayEnd := matchingStarlightBracket(text, arrayStart, '[', ']'); arrayEnd >= 0 {
						return text[arrayStart : arrayEnd+1], true
					}
				}
			}
		}
		next := strings.Index(text[after:], "sidebar")
		if next < 0 {
			break
		}
		sidebarAt = after + next
	}
	return "", false
}

func matchingStarlightBracket(text string, start int, open byte, close byte) int {
	depth := 0
	var quote byte
	escaped := false
	lineComment := false
	blockComment := false
	for i := start; i < len(text); i++ {
		c := text[i]
		if lineComment {
			if c == '\n' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if c == '*' && i+1 < len(text) && text[i+1] == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '/' {
			lineComment = true
			i++
			continue
		}
		if c == '/' && i+1 < len(text) && text[i+1] == '*' {
			blockComment = true
			i++
			continue
		}
		if c == '"' || c == '\'' || c == '`' {
			quote = c
			continue
		}
		if c == open {
			depth++
		} else if c == close {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func isStarlightIdentByte(b byte) bool {
	return b == '_' || b == '$' || b >= '0' && b <= '9' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z'
}

type starlightJSParser struct {
	input  string
	pos    int
	errors []string
}

func (p *starlightJSParser) parseSidebarArray() []starlightSidebarItem {
	p.skipSpace()
	if !p.consume('[') {
		p.errors = append(p.errors, "sidebar is not an array")
		return nil
	}
	var items []starlightSidebarItem
	for {
		p.skipSpace()
		if p.consume(']') {
			break
		}
		if p.peek() == '{' {
			items = append(items, p.parseSidebarObject())
		} else {
			p.skipValue()
		}
		p.skipSpace()
		p.consume(',')
	}
	return items
}

func (p *starlightJSParser) parseSidebarObject() starlightSidebarItem {
	var item starlightSidebarItem
	if !p.consume('{') {
		return item
	}
	for {
		p.skipSpace()
		if p.consume('}') {
			break
		}
		key := p.parseKey()
		p.skipSpace()
		if !p.consume(':') {
			p.skipValue()
			p.consume(',')
			continue
		}
		p.skipSpace()
		switch key {
		case "label":
			item.Label = p.parseStringValue()
		case "link":
			item.Link = p.parseStringValue()
		case "collapsed":
			if value, ok := p.parseBoolValue(); ok {
				item.Collapsed = &value
			} else {
				p.skipValue()
			}
		case "items":
			item.Children = p.parseSidebarArray()
		case "autogenerate":
			item.Directory, item.Collapsed = p.parseAutogenerateObject(item.Collapsed)
		case "badge":
			item.BadgeText, item.BadgeVariant = p.parseBadge()
		default:
			p.skipValue()
		}
		p.skipSpace()
		p.consume(',')
	}
	return item
}

func (p *starlightJSParser) parseAutogenerateObject(existingCollapsed *bool) (string, *bool) {
	collapsed := existingCollapsed
	directory := ""
	if !p.consume('{') {
		p.skipValue()
		return directory, collapsed
	}
	for {
		p.skipSpace()
		if p.consume('}') {
			break
		}
		key := p.parseKey()
		p.skipSpace()
		if !p.consume(':') {
			p.skipValue()
			p.consume(',')
			continue
		}
		switch key {
		case "directory":
			directory = p.parseStringValue()
		case "collapsed":
			if value, ok := p.parseBoolValue(); ok {
				collapsed = &value
			} else {
				p.skipValue()
			}
		default:
			p.skipValue()
		}
		p.skipSpace()
		p.consume(',')
	}
	return directory, collapsed
}

func (p *starlightJSParser) parseBadge() (string, string) {
	if !p.consume('{') {
		p.skipValue()
		return "", ""
	}
	text := ""
	variant := ""
	for {
		p.skipSpace()
		if p.consume('}') {
			break
		}
		key := p.parseKey()
		p.skipSpace()
		if !p.consume(':') {
			p.skipValue()
			p.consume(',')
			continue
		}
		if key == "text" {
			text = p.parseStringValue()
		} else if key == "variant" {
			variant = p.parseStringValue()
		} else {
			p.skipValue()
		}
		p.skipSpace()
		p.consume(',')
	}
	return text, variant
}

func (p *starlightJSParser) parseKey() string {
	p.skipSpace()
	if quote := p.peek(); quote == '"' || quote == '\'' || quote == '`' {
		return p.parseStringValue()
	}
	start := p.pos
	for p.pos < len(p.input) && isStarlightIdentByte(p.input[p.pos]) {
		p.pos++
	}
	return p.input[start:p.pos]
}

func (p *starlightJSParser) parseStringValue() string {
	p.skipSpace()
	quote := p.peek()
	if quote != '"' && quote != '\'' && quote != '`' {
		p.skipValue()
		return ""
	}
	p.pos++
	var b strings.Builder
	escaped := false
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		p.pos++
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == quote {
			break
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (p *starlightJSParser) parseBoolValue() (bool, bool) {
	p.skipSpace()
	if strings.HasPrefix(p.input[p.pos:], "true") {
		p.pos += len("true")
		return true, true
	}
	if strings.HasPrefix(p.input[p.pos:], "false") {
		p.pos += len("false")
		return false, true
	}
	return false, false
}

func (p *starlightJSParser) skipValue() {
	p.skipSpace()
	switch p.peek() {
	case '{':
		if end := matchingStarlightBracket(p.input, p.pos, '{', '}'); end >= 0 {
			p.pos = end + 1
			return
		}
	case '[':
		if end := matchingStarlightBracket(p.input, p.pos, '[', ']'); end >= 0 {
			p.pos = end + 1
			return
		}
	case '"', '\'', '`':
		_ = p.parseStringValue()
		return
	}
	for p.pos < len(p.input) && p.input[p.pos] != ',' && p.input[p.pos] != '}' && p.input[p.pos] != ']' {
		p.pos++
	}
}

func (p *starlightJSParser) skipSpace() {
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			p.pos++
			continue
		}
		if c == '/' && p.pos+1 < len(p.input) && p.input[p.pos+1] == '/' {
			p.pos += 2
			for p.pos < len(p.input) && p.input[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		if c == '/' && p.pos+1 < len(p.input) && p.input[p.pos+1] == '*' {
			p.pos += 2
			for p.pos+1 < len(p.input) && !(p.input[p.pos] == '*' && p.input[p.pos+1] == '/') {
				p.pos++
			}
			if p.pos+1 < len(p.input) {
				p.pos += 2
			}
			continue
		}
		break
	}
}

func (p *starlightJSParser) consume(c byte) bool {
	p.skipSpace()
	if p.peek() == c {
		p.pos++
		return true
	}
	return false
}

func (p *starlightJSParser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func resolveStarlightSidebarItems(items []starlightSidebarItem, routes map[string]string, findings *[]starlightMigrationFinding, sourceFile string) []starlightSidebarItem {
	var resolved []starlightSidebarItem
	for _, item := range items {
		if item.Directory != "" {
			item.Directory = strings.Trim(strings.TrimPrefix(filepath.ToSlash(item.Directory), "/"), "/")
			resolved = append(resolved, item)
			continue
		}
		if item.Link != "" {
			if strings.HasPrefix(item.Link, "http://") || strings.HasPrefix(item.Link, "https://") || strings.HasPrefix(item.Link, "mailto:") {
				item.External = true
				resolved = append(resolved, item)
				continue
			}
			route, ok := routes[normalizeStarlightRouteAlias(item.Link)]
			if !ok {
				*findings = append(*findings, starlightMigrationFinding{
					File:    sourceFile,
					Pattern: "sidebar-link",
					Detail:  fmt.Sprintf("Skipped unresolved sidebar link %q (%s).", item.Link, item.Label),
				})
				continue
			}
			item.Link = starlightNavPageForRoute(route, routes)
			resolved = append(resolved, item)
			continue
		}
		hadChildren := len(item.Children) > 0
		item.Children = resolveStarlightSidebarItems(item.Children, routes, findings, sourceFile)
		if len(item.Children) > 0 || item.Label != "" && !hadChildren {
			resolved = append(resolved, item)
		}
	}
	return resolved
}

func starlightNavPageForRoute(route string, routes map[string]string) string {
	route = normalizeStarlightRouteAlias(route)
	if out, ok := routes[route]; ok {
		route = out
	}
	route = strings.Trim(route, "/")
	if route == "" {
		return "index.md"
	}
	return route + ".md"
}

func renderStarlightNavYAML(items []starlightSidebarItem) string {
	var b strings.Builder
	writeStarlightNavItems(&b, items, 0)
	return b.String()
}

func writeStarlightNavItems(b *strings.Builder, items []starlightSidebarItem, indent int) {
	prefix := strings.Repeat(" ", indent)
	for _, item := range items {
		if item.Link != "" {
			link := item.Link
			if !item.External {
				link = starlightNavLink(item.Link)
			}
			fmt.Fprintf(b, "%s- label: %s\n", prefix, yamlQuote(item.Label))
			fmt.Fprintf(b, "%s  link: %s\n", prefix, yamlQuote(link))
			continue
		}
		fmt.Fprintf(b, "%s- label: %s\n", prefix, yamlQuote(item.Label))
		if item.Directory != "" {
			fmt.Fprintf(b, "%s  autogenerate:\n", prefix)
			fmt.Fprintf(b, "%s    directory: %s\n", prefix, yamlQuote(strings.Trim(item.Directory, "/")))
		}
		if item.Collapsed != nil {
			fmt.Fprintf(b, "%s  collapsed: %t\n", prefix, *item.Collapsed)
		}
		if len(item.Children) > 0 {
			fmt.Fprintf(b, "%s  items:\n", prefix)
			writeStarlightNavItems(b, item.Children, indent+4)
		}
	}
}

func starlightNavLink(page string) string {
	page = filepath.ToSlash(page)
	page = strings.TrimSuffix(page, filepath.Ext(page))
	page = strings.TrimSuffix(page, "/index")
	page = strings.Trim(page, "/")
	if page == "index" || page == "" {
		return "/"
	}
	return "/" + page + "/"
}

func writeStarlightNavBadge(b *strings.Builder, item starlightSidebarItem, indent int) {
	if item.BadgeText == "" {
		return
	}
	prefix := strings.Repeat(" ", indent)
	fmt.Fprintf(b, "%s  badge:\n", prefix)
	fmt.Fprintf(b, "%s    text: %s\n", prefix, yamlQuote(item.BadgeText))
	if item.BadgeVariant != "" {
		fmt.Fprintf(b, "%s    variant: %s\n", prefix, yamlQuote(item.BadgeVariant))
	}
}

func starlightNavSectionName(directory string) string {
	directory = strings.Trim(strings.TrimPrefix(filepath.ToSlash(directory), "/"), "/")
	if directory == "" {
		return directory
	}
	return filepath.Base(directory)
}

func writeStarlightNavTitle(b *strings.Builder, title string, indent int) {
	if title != "" {
		fmt.Fprintf(b, "%s  title: %s\n", strings.Repeat(" ", indent), yamlQuote(title))
	}
}

func yamlQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func renderStarlightMPressConfig(siteConfig starlightProjectConfig, title string) string {
	var b strings.Builder
	b.WriteString("site:\n")
	b.WriteString(fmt.Sprintf("  title: %q\n", title))
	if siteConfig.Description != "" {
		b.WriteString(fmt.Sprintf("  description: %q\n", siteConfig.Description))
	}
	if siteConfig.BaseURL != "" {
		b.WriteString(fmt.Sprintf("  baseURL: %q\n", siteConfig.BaseURL))
	}
	b.WriteString(fmt.Sprintf("  defaultLanguage: %s\n", siteConfig.DefaultLanguage))
	b.WriteString(fmt.Sprintf("  languages: [%s]\n", strings.Join(siteConfig.Languages, ", ")))
	if len(siteConfig.LanguageLabels) > 0 {
		b.WriteString("  languageLabels:\n")
		for _, language := range siteConfig.Languages {
			if label := siteConfig.LanguageLabels[language]; label != "" {
				b.WriteString(fmt.Sprintf("    %s: %s\n", language, yamlQuote(label)))
			}
		}
	}
	b.WriteString("  defaultLanguageAtRoot: true\n")
	b.WriteString("  missingTranslation: link-to-default\n")
	if siteConfig.HasBlog {
		b.WriteString("  headerLinks:\n")
		b.WriteString("    - label: \"Blog\"\n")
		b.WriteString("      url: \"/blog/\"\n")
	}
	if siteConfig.LogoLight != "" {
		b.WriteString(fmt.Sprintf("  logoLight: %q\n", siteConfig.LogoLight))
	}
	if siteConfig.LogoDark != "" {
		b.WriteString(fmt.Sprintf("  logoDark: %q\n", siteConfig.LogoDark))
	}
	if siteConfig.Favicon != "" {
		b.WriteString(fmt.Sprintf("  favicon: %q\n", siteConfig.Favicon))
	}
	if siteConfig.GitHub != "" || siteConfig.Discord != "" || siteConfig.Reddit != "" || siteConfig.X != "" || siteConfig.RSS != "" || siteConfig.Sponsor != "" || siteConfig.EditURL != "" {
		b.WriteString("social:\n")
		if siteConfig.GitHub != "" {
			b.WriteString(fmt.Sprintf("  github: %q\n", siteConfig.GitHub))
		}
		if siteConfig.Discord != "" {
			b.WriteString(fmt.Sprintf("  discord: %q\n", siteConfig.Discord))
		}
		if siteConfig.Reddit != "" {
			b.WriteString(fmt.Sprintf("  reddit: %q\n", siteConfig.Reddit))
		}
		if siteConfig.X != "" {
			b.WriteString(fmt.Sprintf("  x: %q\n", siteConfig.X))
		}
		if siteConfig.RSS != "" {
			b.WriteString(fmt.Sprintf("  rss: %q\n", siteConfig.RSS))
		}
		if siteConfig.Sponsor != "" {
			b.WriteString(fmt.Sprintf("  sponsor: %q\n", siteConfig.Sponsor))
		}
		if siteConfig.EditURL != "" {
			editURL := strings.TrimRight(siteConfig.EditURL, "/")
			if !strings.HasSuffix(editURL, "/src/content/docs") {
				editURL += "/src/content/docs"
			}
			b.WriteString(fmt.Sprintf("  editURL: %q\n", editURL))
		}
	}
	if repository, branch := contributionFromEditURL(siteConfig.EditURL); repository != "" {
		b.WriteString("contribution:\n")
		b.WriteString("  enabled: false\n")
		b.WriteString(fmt.Sprintf("  repository: %q\n", repository))
		b.WriteString(fmt.Sprintf("  branch: %q\n", branch))
		b.WriteString("  quickEdit: false\n")
		b.WriteString("  guide: CONTRIBUTING.md\n")
	}
	b.WriteString(`theme:
  accentColor: "#5375F6"
  layout:
    preset: starlight
build:
  contentDir: content
  staticDir: static
  outputDir: site
  navFile: _nav.yaml
  customCSS: custom.css
`)
	return b.String()
}

func contributionFromEditURL(raw string) (repository, branch string) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Host, "github.com") {
		return "", ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	marker := parts[2]
	if marker != "edit" && marker != "blob" && marker != "tree" {
		return "", ""
	}
	branch = strings.TrimSpace(parts[3])
	if branch == "" {
		branch = "main"
	}
	return "https://github.com/" + parts[0] + "/" + parts[1] + ".git", branch
}

func starlightCustomCSS(sourceDir string) string {
	var sourceCSS string
	for _, candidate := range []string{
		filepath.Join(sourceDir, "src", "stylesheets", "extra.css"),
		filepath.Join(sourceDir, "src", "styles", "custom.css"),
	} {
		if data, err := os.ReadFile(candidate); err == nil {
			sourceCSS = string(data)
			break
		}
	}
	lightAccent := "#5375f6"
	darkAccent := lightAccent
	lightAccentStrong := ""
	darkAccentStrong := ""
	if value := starlightCSSVariable(sourceCSS, `(?s):root\s*\{.*?--sl-color-accent:\s*([^;]+);`); value != "" {
		lightAccent = value
	}
	if value := starlightCSSVariable(sourceCSS, `(?s):root\s*\{.*?--sl-color-accent-high:\s*([^;]+);`); value != "" {
		lightAccentStrong = value
	}
	if value := starlightCSSVariable(sourceCSS, `(?s)\[data-theme=['"]dark['"]\]\s*\{.*?--sl-color-accent:\s*([^;]+);`); value != "" {
		darkAccent = value
	}
	if value := starlightCSSVariable(sourceCSS, `(?s)\[data-theme=['"]dark['"]\]\s*\{.*?--sl-color-accent-high:\s*([^;]+);`); value != "" {
		darkAccentStrong = value
	}
	if lightAccentStrong == "" {
		lightAccentStrong = lightAccent
	}
	if darkAccentStrong == "" {
		darkAccentStrong = darkAccent
	}

	return sourceCSS + fmt.Sprintf(`
:root {
  --accent: %s;
  --wails-accent-strong: %s;
  --wails-accent-link: %s;
  --wails-sidebar-root: #17181c;
  --wails-toc-muted: #555962;
  --bg: #ffffff;
  --surface: #f6f7f9;
  --surface-solid: #ffffff;
  --panel: #f6f7f9;
  --text: #17181c;
  --muted: #353841;
  --border: #edeef3;
  --code: #f6f7f9;
  --code-text: #17181c;
  --sl-color-bg: var(--bg);
  --sl-color-white: var(--text);
  --sl-color-gray-3: var(--muted);
  --sl-color-gray-5: var(--border);
  --sl-color-gray-6: var(--surface);
  --sl-color-gray-6-rgb: 35, 39, 49;
}

html[data-theme="dark"] {
  --accent: %s;
  --wails-accent-strong: %s;
  --wails-accent-link: %s;
  --wails-sidebar-root: #ffffff;
  --wails-toc-muted: #888c96;
  --bg: #17181c;
  --surface: #23262f;
  --surface-solid: #23262f;
  --panel: #23262f;
  --text: #ffffff;
  --muted: #c1c3c8;
  --border: #353841;
  --code: #23262f;
  --code-text: #ffffff;
}
@media (prefers-color-scheme: dark) {
  html[data-theme="system"] {
    --accent: %s;
    --wails-accent-strong: %s;
    --wails-accent-link: %s;
    --wails-sidebar-root: #ffffff;
    --wails-toc-muted: #888c96;
    --bg: #17181c;
    --surface: #23262f;
    --surface-solid: #23262f;
    --panel: #23262f;
    --text: #ffffff;
    --muted: #c1c3c8;
    --border: #353841;
    --code: #23262f;
    --code-text: #ffffff;
  }
}

body { color: var(--muted); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", "Noto Sans", Arial, sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"; }
body > header { height: 64px; padding-inline: 24px; background: var(--surface); border-bottom-color: var(--bg); }
.sidebar, .toc { top: 64px; height: calc(100vh - 64px - var(--mpress-devbar-height, 0px)); }

.docs-page .layout {
  gap: 0;
  max-width: none;
  margin: 0;
}
.docs-page .docs-stage { container-type: inline-size; grid-template-columns: minmax(0, 1fr) minmax(0, var(--layout-content-width)) minmax(0, 1fr) minmax(0, var(--layout-toc-width)); width: 100%%; }
.docs-page .docs-stage > main { grid-column: 2; }
.docs-page .docs-stage > .toc { grid-column: 4; }

:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .brand { order: 1; width: 188.45px; height: 40px; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-logo { flex-basis: 188.45px; width: 188.45px; height: 40px; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .primary-links { order: 4; height: 32px; margin-right: -.9rem; padding-inline: .65rem; justify-content: center; border-left: 1px solid var(--border); }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .primary-links a { padding: 0; color: var(--accent-text); font-weight: 600; line-height: 32px; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .search { order: 2; flex: 0 1 352px; width: 352px; margin-left: auto; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .search input { border-color: var(--border); border-radius: 8px; background: var(--bg); color: var(--muted); }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .header-links { display: contents; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .social-links { order: 3; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .header-utility-cluster { order: 5; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .header-utility-cluster > .header-group { margin: 0; padding: 0; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .header-utility-cluster > .header-group::before { display: none; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .version-select { width: auto; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .accessibility-select,
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .language-select,
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-toggle { width: 34px; min-width: 34px; height: 34px; min-height: 34px; justify-content: center; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .language-select .utility-menu-trigger { width: 34px; min-width: 34px; justify-content: center; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .social-links { width: auto; gap: 16px; margin-left: 0; padding-left: 0; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .social-links::before { display: none; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .social-link { width: 32px; height: 32px; margin: -8px; padding: 8px; border-radius: 0; color: var(--wails-accent-strong); }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .social-link[aria-label="Sponsor"] { width: 16px; height: 16px; margin: 0; padding: 0; }
:is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .social-link:hover { background: transparent; color: var(--hover); opacity: 1; }

.docs-page .sidebar { z-index: 1; padding: 0; background: var(--surface); border-right-color: var(--bg); scrollbar-gutter: auto; }
.docs-page .sidebar nav { padding: 16px 16px 0; }
.docs-page .sidebar a,
.docs-page .sidebar summary,
.docs-page .sidebar .nav-label { margin: 0; padding: 3.2px 8px; border-radius: 0; color: var(--muted); line-height: 1.4; }
.docs-page .sidebar nav > * + * { margin-top: 8px; }
.docs-page .sidebar nav > a { padding-block: 4.8px; color: var(--wails-sidebar-root); font-weight: 600; }
.docs-page .sidebar nav > a,
.docs-page .sidebar nav > .nav-label,
.docs-page .sidebar summary { color: var(--wails-sidebar-root); font-weight: 600; }
.docs-page .sidebar summary .lucide { width: 20px; height: 20px; }
.docs-page .sidebar details > div { padding-left: 17px; }
.docs-page .sidebar details > div a { padding-block: 4.2px; }
.docs-page .sidebar a.active { padding: 4.2px 8px; border-radius: 4px; background: var(--accent-fill); color: var(--accent-fill-text); }
html[data-theme="light"] .docs-page .sidebar { background: #ffffff; }
html[data-theme="light"] .docs-page .sidebar a.active { color: #ffffff; }

.docs-page main { padding: 40px 24px 6rem; color: var(--muted); }
.docs-page article { width: 100%%; font-size: 16px; line-height: 1.75; }
.docs-page article > h1 { display: block; max-width: none; margin: 0 0 12px; padding: 0; color: var(--text); font-size: 42px; line-height: 1.2; }
.docs-page article > h1 + .page-lead { position: relative; box-sizing: border-box; width: calc(100%% + 48px + max(0px, (100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)); max-width: none; margin: 0 0 48px -24px; padding: 0 calc(24px + max(0px, (100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)) 22px 24px; border-bottom: 1px solid color-mix(in srgb, var(--text) 18%%, transparent); font-size: 16px; line-height: 28px; }
.docs-page article > h1:not(:has(+ .page-lead)) { position: relative; box-sizing: border-box; width: calc(100%% + 48px + max(0px, (100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)); margin: 0 0 48px -24px; padding: 0 calc(24px + max(0px, (100cqw - var(--layout-content-width) - var(--layout-toc-width)) / 2)) 22px 24px; border-bottom: 1px solid color-mix(in srgb, var(--text) 18%%, transparent); }
.docs-page article > h1 + .page-lead::after,
.docs-page article > h1:not(:has(+ .page-lead))::after { content: ""; position: absolute; right: 100%%; bottom: -1px; width: 100vw; border-bottom: 1px solid color-mix(in srgb, var(--text) 18%%, transparent); }
.docs-page article > .page-lead + h2 { margin-top: 0; }
.docs-page article > h1 + h2 { margin-top: 0; }
.docs-page article h2 { margin-top: 64px; margin-bottom: 0; color: var(--text); font-size: 35px; line-height: 42px; }
.docs-page article h3,
.docs-page article h4 { color: var(--text); }
.docs-page article h3 { margin-top: 43.5px; margin-bottom: 20px; font-size: 29px; line-height: 34.8px; }
.docs-page article p,
.docs-page article li { color: var(--muted); line-height: 1.75; }

.docs-page .toc { min-width: 0; gap: 0; padding: 16px 17px; overflow-x: hidden; border-left: 1px solid var(--surface); }
.docs-page .toc strong { margin-bottom: 8px; color: var(--text); }
.docs-page .toc a { width: 100%%; max-width: 265px; margin: 0; padding: 4px 8px; overflow-wrap: anywhere; border-left: 0; color: var(--wails-toc-muted); font-size: 13px; line-height: 16.25px; }
.docs-page .toc a.toc-level-3 { padding-left: 24px; }
.docs-page .toc a.active { border-left: 0; background: transparent; color: var(--accent-text); }

.docs-page .mpress-admonition { --notice: hsl(234, 100%%, 60%%); --notice-bg: hsl(234, 54%%, 20%%); --notice-ink: hsl(234, 100%%, 87%%); --notice-text: #fff; margin: 16px 0 0; padding: 16px; overflow: visible; border: 0; border-left: 4px solid var(--notice); border-radius: 0; background: var(--notice-bg); color: var(--notice-text); }
.docs-page .mpress-admonition::before { display: none; }
.docs-page .mpress-admonition-info,
.docs-page .mpress-admonition-note { --notice: hsl(234, 100%%, 60%%); --notice-bg: hsl(234, 54%%, 20%%); --notice-ink: hsl(234, 100%%, 87%%); }
.docs-page .mpress-admonition-tip { --notice: hsl(281, 82%%, 63%%); --notice-bg: hsl(281, 39%%, 22%%); --notice-ink: hsl(281, 82%%, 89%%); }
.docs-page .mpress-admonition-warning,
.docs-page .mpress-admonition-caution { --notice: hsl(41, 82%%, 63%%); --notice-bg: hsl(41, 39%%, 22%%); --notice-ink: hsl(41, 82%%, 87%%); }
.docs-page .mpress-admonition-danger { --notice: hsl(339, 82%%, 63%%); --notice-bg: hsl(339, 39%%, 22%%); --notice-ink: hsl(339, 82%%, 87%%); }
.docs-page .mpress-admonition-titlebar { gap: 8px; margin: 0; color: var(--notice-ink); font-size: 18px; font-weight: 600; line-height: 21.6px; letter-spacing: 0; text-transform: none; }
.docs-page .mpress-admonition-titlebar > svg { width: 24px; height: 24px; }
.docs-page .mpress-admonition-body { margin-top: 8px; }
.docs-page .mpress-admonition-body :is(p, li) { color: var(--notice-text); }
.docs-page .mpress-admonition-body :is(a, em) { color: var(--notice-ink); }
.docs-page .mpress-admonition-body a { text-decoration-color: currentColor; }
.docs-page .mpress-admonition-body code:not(pre code) { background: color-mix(in srgb, var(--notice-ink) 12%%, transparent); color: var(--notice-text); }
.docs-page .mpress-admonition-body > pre:first-child { margin-top: 0; }

html[data-theme="light"] .docs-page .mpress-admonition { --notice-text: hsl(224, 10%%, 10%%); }
html[data-theme="light"] .docs-page .mpress-admonition-info,
html[data-theme="light"] .docs-page .mpress-admonition-note { --notice: hsl(234, 90%%, 60%%); --notice-bg: hsl(234, 88%%, 90%%); --notice-ink: hsl(234, 80%%, 30%%); }
html[data-theme="light"] .docs-page .mpress-admonition-tip { --notice: hsl(281, 90%%, 60%%); --notice-bg: hsl(281, 80%%, 90%%); --notice-ink: hsl(281, 90%%, 30%%); }
html[data-theme="light"] .docs-page .mpress-admonition-warning,
html[data-theme="light"] .docs-page .mpress-admonition-caution { --notice: hsl(41, 90%%, 60%%); --notice-bg: hsl(41, 90%%, 88%%); --notice-ink: hsl(41, 80%%, 25%%); }
html[data-theme="light"] .docs-page .mpress-admonition-danger { --notice: hsl(339, 90%%, 60%%); --notice-bg: hsl(339, 80%%, 90%%); --notice-ink: hsl(339, 80%%, 30%%); }
@media (prefers-color-scheme: light) {
  html[data-theme="system"] .docs-page .sidebar { background: #ffffff; }
  html[data-theme="system"] .docs-page .sidebar a.active { color: #ffffff; }
  html[data-theme="system"] .docs-page .mpress-admonition { --notice-text: hsl(224, 10%%, 10%%); }
  html[data-theme="system"] .docs-page .mpress-admonition-info,
  html[data-theme="system"] .docs-page .mpress-admonition-note { --notice: hsl(234, 90%%, 60%%); --notice-bg: hsl(234, 88%%, 90%%); --notice-ink: hsl(234, 80%%, 30%%); }
  html[data-theme="system"] .docs-page .mpress-admonition-tip { --notice: hsl(281, 90%%, 60%%); --notice-bg: hsl(281, 80%%, 90%%); --notice-ink: hsl(281, 90%%, 30%%); }
  html[data-theme="system"] .docs-page .mpress-admonition-warning,
  html[data-theme="system"] .docs-page .mpress-admonition-caution { --notice: hsl(41, 90%%, 60%%); --notice-bg: hsl(41, 90%%, 88%%); --notice-ink: hsl(41, 80%%, 25%%); }
  html[data-theme="system"] .docs-page .mpress-admonition-danger { --notice: hsl(339, 90%%, 60%%); --notice-bg: hsl(339, 80%%, 90%%); --notice-ink: hsl(339, 80%%, 30%%); }
}
@media (prefers-color-scheme: dark) {
  html[data-theme="system"] .docs-page .mpress-admonition-titlebar { color: color-mix(in srgb, var(--notice) 30%%, #fff); }
}

.docs-page pre { padding: 13.6px 16px; border-color: color-mix(in srgb, var(--border) 75%%, transparent); border-radius: 1px; background: var(--surface); color: var(--text); font-size: 14.4px; line-height: 23.76px; }
.docs-page .mpress-terminal { margin: 16px 0; }
.docs-page .mpress-terminal pre { min-height: 0; margin: 0; padding: 14px 16px; border: 0; background: transparent; }
.docs-page .mpress-codeframe { margin: 16px 0 0; border-color: color-mix(in srgb, var(--border) 75%%, transparent); border-radius: 1px; background: var(--bg); }
.docs-page .mpress-codeframe-header { min-height: 40px; border-bottom-color: var(--border); background: var(--surface); box-shadow: inset 0 2px var(--accent); }
.docs-page .mpress-codeframe-title { padding: 8px 11px 8px 16px; color: var(--text); font-size: 14.4px; font-weight: 500; line-height: 23.76px; letter-spacing: .0143em; }
.docs-page .mpress-codeframe-language { border-color: var(--border); color: var(--muted); }
.docs-page .mpress-codeframe-header > .mpress-copy { background: var(--surface); }
.docs-page .mpress-codeframe pre { margin: 0; padding: 12px 16px; border: 0; background: var(--surface); }
.docs-page .mpress-codeframe .mpress-code-line { margin-inline: -16px; padding-inline: 16px; }

.docs-page article h2 > code,
.docs-page article h3 > code,
.docs-page article h4 > code { padding: .05em .16em; color: inherit; font: inherit; }

.docs-page article table { margin: 16px 0 0; overflow: visible; border: 0; border-radius: 0; }
.docs-page article th,
.docs-page article td { padding: 10px 12px; border-bottom: 1px solid var(--border); }
.docs-page article th { background: transparent; color: var(--text); font-size: 16px; font-weight: 600; letter-spacing: 0; text-transform: none; }

.docs-page .mpress-steps { margin: 16px 0 0; }
.docs-page .mpress-step { grid-template-columns: 28px minmax(0, 1fr); gap: 16px; padding-bottom: 20px; }
.docs-page .mpress-step:last-child { padding-bottom: 0; }
.docs-page .mpress-step::before { left: 13.5px; top: 32px; border-left-color: var(--border); }
.docs-page .mpress-step-number,
.docs-page .mpress-step-num { width: 28px; height: 28px; border-color: var(--border); background: var(--surface); color: var(--text); font-size: 12px; }
.docs-page .mpress-step-title { margin: 0 0 8px; color: var(--muted); font-size: 21px; font-weight: 600; line-height: 30px; }
.docs-page .mpress-step-content > :last-child { margin-bottom: 0; }

.docs-page .mpress-tabs { margin: 16px 0 0; }
.docs-page .mpress-tabs [role="tablist"] { gap: 0; min-height: 30px; border-bottom: 2px solid var(--border); color: var(--wails-toc-muted); font-size: 16px; font-weight: 400; }
.docs-page .mpress-tabs [role="tab"] { display: flex; width: auto; min-width: 0; height: 28px; align-items: center; justify-content: center; gap: 8px; padding: 4.4px 20px; border-bottom: 0; color: var(--wails-toc-muted); line-height: 19.2px; }
.docs-page .mpress-tabs [role="tab"][aria-selected="true"] { color: var(--text); font-weight: 600; box-shadow: 0 2px 0 var(--wails-accent-link); }
.docs-page .mpress-tabs [role="tabpanel"] { margin-top: 16px; padding: 0; }

.mpress-site-banner { position: relative; z-index: 1; padding: 8px 16px; background: var(--wails-accent-strong); color: #470606; box-shadow: none; font-size: 16px; font-weight: 400; }
.landing-page .landing-main { padding: 24px 0 30px; }
.mpress-frontmatter-hero {
  width: min(1080px, calc(100%% - 3rem));
  min-height: 0;
  margin: 0 auto;
  padding: 48px 0 32px;
  border-bottom: 0;
}
.mpress-frontmatter-hero-inner {
  position: relative;
  width: 61.728%%;
  min-height: 391px;
  margin: 0;
  padding: 0;
  align-items: flex-start;
  justify-content: flex-start;
  text-align: left;
}
.mpress-frontmatter-hero-image { position: absolute; top: 44px; left: calc(100%% + 32px); display: block; width: 381px; height: 301px; margin: 0; }
.mpress-frontmatter-hero-title {
  max-width: 464px;
  margin: 0 0 0;
  color: var(--text);
  font-size: 64px;
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: 0;
  text-wrap: wrap;
}
.mpress-frontmatter-hero-tagline {
  max-width: 572px;
  margin: 16px 0 0;
  font-size: 20px;
  line-height: 1.75;
}
.landing-page .mpress-frontmatter-hero-tagline { margin-top: 16px; }
.mpress-frontmatter-hero-actions { width: 100%%; justify-content: flex-start; margin-top: 32px; }
.mpress-frontmatter-hero-action { min-height: 56px; padding: .75rem 1.25rem; border-radius: 999px; }
.mpress-frontmatter-hero-action-primary { border-color: var(--wails-accent-strong); background: var(--wails-accent-strong); color: #17181c; }
.mpress-frontmatter-hero-action:not(.mpress-frontmatter-hero-action-primary) { border-color: var(--text); background: transparent; color: var(--text); }
.landing-page .landing-main > .mpress-terminal { width: min(1080px, calc(100%% - 3rem)); margin: 0 auto 16px; }
.morph-title-line { display: inline-flex; align-items: baseline; gap: .26em; white-space: nowrap; }
.morph-word-title { display: inline-block; width: 4.5em; text-align: left; }
.morph-word-title span { display: inline-block; transition: opacity .65s ease, filter .65s ease, transform .65s ease; }
.morph-word-title span.morph-word-out { opacity: 0; filter: blur(8px); transform: translateY(-.18em); }
.browser-footnote { display: block; margin-top: .45rem; font-size: .65em; font-style: italic; opacity: .65; }

.landing-page .landing-main > h2,
.landing-page .landing-main > p,
.landing-page .landing-main > pre,
.landing-page .landing-main > hr,
.landing-page .landing-main > aside,
.landing-page .landing-main > .mpress-card-grid {
  width: min(760px, calc(100%% - 2rem));
  margin-left: auto;
  margin-right: auto;
}
.landing-page .landing-main > h2,
.landing-page .landing-main > p,
.landing-page .landing-main > pre,
.landing-page .landing-main > hr,
.landing-page .landing-main > aside,
.landing-page .landing-main > .mpress-card-grid {
  width: min(1080px, calc(100%% - 3rem));
}
.landing-page .landing-main > h2 { margin-top: 3rem; margin-bottom: 19px; color: var(--text); font-size: 35px; font-weight: 600; line-height: 42px; }
.landing-page .landing-main > .mpress-card-grid { margin-top: 1.5rem; }
.landing-page > footer { width: min(1080px, calc(100%% - 3rem)); max-width: none; padding-inline: 0; }

@media (max-width: 1050px) {
  .docs-page .layout { grid-template-columns: 295px minmax(0, 1fr); gap: 0; }
  .docs-page .docs-stage { display: block; width: 100%%; }
  .docs-page main { padding: 96px 16px 6rem; }
  .docs-page article > h1 + .page-lead { width: calc(100%% + 32px); margin-left: -16px; padding-right: 16px; padding-left: 16px; }
  .docs-page article > h1:not(:has(+ .page-lead)) { width: calc(100%% + 32px); margin-left: -16px; padding-right: 16px; padding-left: 16px; }
  .docs-page .toc { position: fixed; z-index: 8; top: 64px; right: 0; left: 295px; display: flex; width: auto; height: 46px; align-items: center; flex-direction: row; gap: 8px; padding: 7px 16px; overflow: hidden; border: 0; border-bottom: 1px solid var(--surface); background: var(--surface); }
  .docs-page .toc strong { display: inline-flex; min-width: 126px; height: 30px; align-items: center; justify-content: space-between; gap: 10px; margin: 0; padding: 5px 10px; border: 1px solid var(--border); border-radius: 6px; background: var(--bg); color: var(--muted); font-size: 13px; font-weight: 500; }
  .docs-page .toc-mobile-icon { display: inline-flex; }
  .docs-page .toc > a { display: none; flex: 0 0 auto; margin: 0; padding: 5px 0; border: 0; font-size: 13px; white-space: nowrap; }
  .docs-page .toc > a.active { display: block; }
}

@media (max-width: 720px) {
  body > header { height: 58px; padding-inline: .9rem; }
  .docs-page #menu { order: 4; margin-left: .55rem; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .brand { order: 1; width: 139px; height: 32px; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-logo { width: 139px; height: 32px; flex-basis: 139px; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .search { order: 2; flex: 0 0 42px; width: 42px; margin-left: auto; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .search input { width: 42px; }
  .docs-page .header-links { display: flex; }
  .docs-page main { width: 100%%; min-width: 0; padding: 38px 16px 6rem; overflow: hidden; }
  .docs-page article { width: 100%%; max-width: 100%%; min-width: 0; }
  .docs-page article > h1 { font-size: 35px; line-height: 42px; }
  .docs-page article h2 { margin-top: 48px; font-size: 29px; line-height: 34.8px; overflow-wrap: break-word; }
  .docs-page .mpress-admonition,
  .docs-page .mpress-codeframe,
  .docs-page pre { max-width: 100%%; }
  .docs-page .mpress-tabs [role="tablist"] { max-width: 100%%; overflow-x: auto; scrollbar-width: thin; }
  .docs-page .mpress-tabs [role="tab"] { min-width: 108px; padding-inline: 12px; }
  .docs-page .mpress-step { grid-template-columns: 28px minmax(0, 1fr); gap: 14px; }
  .docs-page .toc { position: fixed; z-index: 8; top: 58px; right: 0; left: 0; display: flex; width: 100%%; height: 42px; align-items: center; flex-direction: row; gap: 8px; padding: 7px 16px; overflow: hidden; border: 0; border-bottom: 1px solid var(--bg); background: var(--surface); }
  .docs-page .toc strong { display: inline-flex; min-width: 126px; height: 30px; align-items: center; justify-content: space-between; gap: 10px; margin: 0; padding: 5px 10px; border: 1px solid var(--border); border-radius: 6px; background: var(--bg); color: var(--muted); font-size: 13px; font-weight: 500; }
  .docs-page .toc-mobile-icon { display: inline-flex; }
  .docs-page .toc > a { display: none; flex: 0 0 auto; margin: 0; padding: 5px 0; border: 0; font-size: 13px; white-space: nowrap; }
  .docs-page .toc > a.active { display: block; }
  .docs-page main { padding-top: 86px; }
  :is(.landing-page, .blog-index-page, .blog-article-page) .search:not(:focus-within) input { border-color: transparent; background: transparent; box-shadow: none; }
  :is(.landing-page, .blog-index-page, .blog-article-page) .header-links { padding-right: 0; }
  .landing-page .landing-main { padding: 0 1rem 30px; }
  .mpress-frontmatter-hero { width: 100%%; padding: 4.5rem 0 2rem; }
  .mpress-frontmatter-hero-inner { width: 100%%; min-height: 0; padding: 0; align-items: center; text-align: center; }
  .mpress-frontmatter-hero-image { position: static; display: block; width: 220px; height: 174px; margin-bottom: 1rem; }
  .mpress-frontmatter-hero-title { max-width: 340px; font-size: 32px; line-height: 1.08; text-align: center; }
  .landing-page .mpress-frontmatter-hero-tagline { max-width: 360px; margin-top: 1.25rem; font-size: 16px; line-height: 1.65; text-align: center; }
  .morph-word-title { display: inline-block; width: auto; text-align: center; }
  .mpress-frontmatter-hero-actions { width: 100%%; justify-content: center; }
  .mpress-frontmatter-hero-action { min-height: 40px; flex: 0 0 auto; justify-content: center; padding: .45rem .9rem; }
  .landing-page .landing-main > h2,
  .landing-page .landing-main > p,
  .landing-page .landing-main > pre,
  .landing-page .landing-main > hr,
  .landing-page .landing-main > aside,
  .landing-page .landing-main > .mpress-card-grid { width: 100%%; }
  .landing-page .landing-main > h2 { margin-top: 2.5rem; font-size: 28px; }
}

@media (max-width: 420px) {
  body > header { gap: .2rem; padding-inline: .3rem; }
  .docs-page .header-links { gap: .2rem; }
  .docs-page #menu { width: 40px; height: 40px; margin-left: .1rem; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .brand { width: 96px; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-logo { width: 96px; flex-basis: 96px; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .search { width: 40px; flex-basis: 40px; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .search input { width: 40px; height: 40px; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .accessibility-select,
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-toggle { width: 34px; min-width: 34px; margin: 0; padding: 0; }
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .accessibility-select::before,
  :is(.docs-page, .landing-page, .blog-index-page, .blog-article-page) .theme-toggle::before { display: none; }
}

.mpress-showcase-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 1rem;
  margin: 1.5rem 0 2rem;
}

.mpress-showcase-card {
  display: block;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  color: var(--foreground);
  text-decoration: none;
  box-shadow: var(--shadow-small);
  transition: transform .15s ease, box-shadow .15s ease, border-color .15s ease;
}

.mpress-showcase-card:hover {
  transform: translateY(-2px);
  border-color: var(--primary);
  box-shadow: var(--shadow-medium);
}

.mpress-showcase-card img {
  width: 100%%;
  aspect-ratio: 16 / 10;
  object-fit: cover;
  display: block;
  background: var(--muted);
}

.mpress-showcase-card span {
  display: block;
  padding: .7rem .8rem;
  font-weight: 600;
  font-size: .9rem;
}
`, lightAccent, lightAccentStrong, lightAccent, darkAccent, darkAccentStrong, darkAccentStrong, darkAccent, darkAccentStrong, darkAccentStrong)
}

func convertStarlightLandingTerminals(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	for i := 0; i < len(lines); {
		indent, marker, ok := starlightFenceOpening(lines[i])
		if !ok {
			out = append(out, lines[i])
			i++
			continue
		}
		trimmed := strings.TrimSpace(lines[i])
		info := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, marker)))
		language := strings.Fields(info)
		if len(language) == 0 || language[0] != "bash" && language[0] != "shell" && language[0] != "sh" && language[0] != "powershell" {
			out = append(out, lines[i])
			i++
			continue
		}
		end := i + 1
		for end < len(lines) && !starlightFenceClosing(lines[end], marker) {
			end++
		}
		if end >= len(lines) {
			out = append(out, lines[i:]...)
			break
		}
		out = append(out, indent+`@terminal{frame=macos|prompt=none}`)
		out = append(out, lines[i+1:end]...)
		out = append(out, indent+"@end")
		i = end + 1
	}
	return strings.Join(out, "\n")
}

func starlightCSSVariable(source string, pattern string) string {
	match := regexp.MustCompile(pattern).FindStringSubmatch(source)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func inferStarlightSiteTitle(sourceDir string) string {
	for _, name := range []string{"astro.config.mjs", "astro.config.ts", "astro.config.js"} {
		data, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			continue
		}
		if m := regexp.MustCompile(`property:\s*['"]og:site_name['"]\s*,\s*content:\s*['"]([^'"]+)['"]`).FindStringSubmatch(string(data)); len(m) > 1 {
			return m[1]
		}
	}
	if filepath.Base(sourceDir) == "docs" {
		parent := filepath.Base(filepath.Dir(sourceDir))
		if parent != "." && parent != string(filepath.Separator) && parent != "" {
			return strings.Title(strings.ReplaceAll(parent, "-", " "))
		}
	}
	return "Documentation"
}

func copyTree(sourceDir string, outputDir string) (int, error) {
	count := 0
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return 0, err
	}
	err := filepath.Walk(sourceDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(sourceDir, path)
		dst := filepath.Join(outputDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

func extractStarlightSlug(source string) string {
	if m := slSlugRegex.FindStringSubmatch(source); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func isStarlightPrivateContent(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.HasPrefix(part, "_") {
			return true
		}
	}
	return false
}

func starlightOutputRel(rel string, source string, languages []string) string {
	outRel := strings.TrimSuffix(rel, ".mdx") + ".mpd"
	if strings.HasSuffix(rel, ".md") {
		outRel = strings.TrimSuffix(rel, ".md") + ".mpd"
	}
	if slug := extractStarlightSlug(source); slug != "" {
		outRel = strings.Trim(strings.TrimPrefix(slug, "/"), "/") + ".mpd"
		if locale := starlightLocaleFromRel(rel, languages); locale != "" && !strings.HasPrefix(filepath.ToSlash(outRel), locale+"/") {
			outRel = filepath.ToSlash(filepath.Join(locale, outRel))
		}
	}
	return filepath.ToSlash(outRel)
}

func starlightLocaleFromRel(rel string, languages []string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return ""
	}
	first := strings.ToLower(parts[0])
	for _, language := range languages {
		if first == strings.ToLower(language) && first != "en" {
			return parts[0]
		}
	}
	return ""
}

func starlightRouteFromOutputRel(outRel string) string {
	route := strings.TrimSuffix(filepath.ToSlash(outRel), filepath.Ext(outRel))
	if strings.HasSuffix(route, "/index") {
		route = strings.TrimSuffix(route, "/index")
	}
	if route == "index" {
		route = ""
	}
	return "/" + strings.Trim(route, "/")
}

func buildStarlightRouteMap(pages []starlightImportPage) map[string]string {
	routes := make(map[string]string)
	for _, page := range pages {
		sourceRoute := starlightRouteFromOutputRel(page.OutputRel)
		route := starlightMPressRoute(page)
		addStarlightRouteAlias(routes, sourceRoute, route)
		addStarlightRouteAlias(routes, route, route)

		trimmed := strings.Trim(route, "/")
		if trimmed == "" {
			continue
		}
		parts := strings.Split(trimmed, "/")
		base := parts[len(parts)-1]
		parent := "/" + strings.Join(parts[:len(parts)-1], "/")
		if parent == "/" {
			parent = ""
		}

		switch base {
		case "index", "overview", "basics":
			addStarlightRouteAlias(routes, parent, route)
		case "shortcuts":
			if len(parts) >= 2 && parts[len(parts)-2] == "keyboard" {
				addStarlightRouteAlias(routes, parent, route)
			}
		}

		if len(parts) >= 3 && (parts[1] == "build" || parts[1] == "distribution") {
			addStarlightRouteAlias(routes, "/"+parts[0]+"/"+base, route)
		}
		if len(parts) == 2 && parts[0] == "guides" && (base == "performance" || base == "security") {
			addStarlightRouteAlias(routes, "/guides/advanced/"+base, route)
		}
		if len(parts) == 2 && parts[0] == "guides" && base == "installers" {
			addStarlightRouteAlias(routes, "/guides/packaging", route)
		}
	}
	addStarlightLegacyAliases(routes)
	return routes
}

func appendStarlightSectionIndexes(pages []starlightImportPage) []starlightImportPage {
	sections := make(map[string]bool)
	sectionModified := make(map[string]time.Time)
	topLevelRoutes := make(map[string]bool)
	blogPosts := make(map[string][]starlightImportPage)
	blogIndexes := make(map[string]bool)
	for _, page := range pages {
		outRel := filepath.ToSlash(page.OutputRel)
		localizedRel := outRel
		if page.Locale != "" {
			localizedRel = strings.TrimPrefix(localizedRel, page.Locale+"/")
		}
		localizedRoute := normalizeStarlightRouteAlias(starlightRouteFromOutputRel(localizedRel))
		if localizedRoute == "/blog" {
			blogIndexes[page.Locale] = true
		} else if strings.HasPrefix(localizedRel, "blog/") {
			blogPosts[page.Locale] = append(blogPosts[page.Locale], page)
		}
		// Locale roots are complete sites of their own. They must never be
		// mistaken for missing sections in the default-language site.
		if page.Locale != "" {
			continue
		}
		route := strings.Trim(normalizeStarlightRouteAlias(starlightRouteFromOutputRel(outRel)), "/")
		parts := strings.Split(route, "/")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		if len(parts) == 1 {
			topLevelRoutes[parts[0]] = true
		} else {
			sections[parts[0]] = true
			if page.Modified.After(sectionModified[parts[0]]) {
				sectionModified[parts[0]] = page.Modified
			}
		}
	}

	var blogLocales []string
	for locale := range blogPosts {
		if !blogIndexes[locale] {
			blogLocales = append(blogLocales, locale)
		}
	}
	sort.Strings(blogLocales)
	for _, locale := range blogLocales {
		posts := blogPosts[locale]
		sort.SliceStable(posts, func(i, j int) bool { return posts[i].OutputRel > posts[j].OutputRel })
		var source strings.Builder
		source.WriteString("---\ntitle: Blog\ndescription: News, releases, and project updates.\nlayout: blog-index\ngenerated: true\n---\n\n")
		for _, post := range posts {
			title := "Article"
			if match := slTitleRegex.FindStringSubmatch(post.Source); len(match) > 1 {
				title = strings.TrimSpace(match[1])
			}
			fmt.Fprintf(&source, "- [%s](%s/)\n", title, strings.TrimSuffix(starlightMPressRoute(post), "/"))
		}
		outputRel := "blog/index.mpd"
		if locale != "" {
			outputRel = locale + "/blog/index.mpd"
		} else {
			topLevelRoutes["blog"] = true
		}
		pages = append(pages, starlightImportPage{
			SourceRel: outputRel,
			OutputRel: outputRel,
			Source:    source.String(),
			Locale:    locale,
			Modified:  latestStarlightPageModification(posts),
		})
	}

	var names []string
	for section := range sections {
		if !topLevelRoutes[section] {
			names = append(names, section)
		}
	}
	sort.Strings(names)
	for _, section := range names {
		title := strings.Title(strings.ReplaceAll(section, "-", " "))
		pages = append(pages, starlightImportPage{
			SourceRel: section + "/index.mpd",
			OutputRel: section + "/index.mpd",
			Modified:  sectionModified[section],
			Source: fmt.Sprintf(`---
title: %s
generated: true
---

# %s

`, title, title),
		})
	}
	return pages
}

func latestStarlightPageModification(pages []starlightImportPage) time.Time {
	var latest time.Time
	for _, page := range pages {
		if page.Modified.After(latest) {
			latest = page.Modified
		}
	}
	return latest
}

func addStarlightLegacyAliases(routes map[string]string) {
	addStarlightAliasIfTargetExists(routes, "/features/services", "/features/bindings/services")
	addStarlightAliasIfTargetExists(routes, "/docs/concepts/services", "/features/bindings/services")
	addStarlightAliasIfTargetExists(routes, "/features/events/custom", "/features/events/system")
	addStarlightAliasIfTargetExists(routes, "/features/events/patterns", "/features/events/system")
	addStarlightAliasIfTargetExists(routes, "/guides/patterns/menus", "/guides/menus")
	addStarlightAliasIfTargetExists(routes, "/guides/patterns/database", "/tutorials/02-todo-vanilla")
	addStarlightAliasIfTargetExists(routes, "/tutorials/system-tray", "/features/menus/systray")
	addStarlightAliasIfTargetExists(routes, "/examples/notifications", "/features/notifications/overview")
	addStarlightAliasIfTargetExists(routes, "/guides/ci-cd", "/guides/distribution/auto-updates")
	addStarlightAliasIfTargetExists(routes, "/guides/distribution/file-associations", "/guides/file-associations")
	addStarlightAliasIfTargetExists(routes, "/guides/distribution/single-instance", "/guides/single-instance")
	addStarlightAliasIfTargetExists(routes, "/guides/auto-updates", "/guides/distribution/auto-updates")
	addStarlightAliasIfTargetExists(routes, "/learn/runtime", "/reference/frontend-runtime")
}

func addStarlightAliasIfTargetExists(routes map[string]string, alias string, target string) {
	if route, ok := routes[normalizeStarlightRouteAlias(target)]; ok {
		addStarlightRouteAlias(routes, alias, route)
	}
}

func addStarlightRouteAlias(routes map[string]string, alias string, route string) {
	alias = normalizeStarlightRouteAlias(alias)
	route = normalizeStarlightRouteAlias(route)
	if _, exists := routes[alias]; !exists {
		routes[alias] = route
	}
	if strings.HasPrefix(alias, "/") && !strings.HasPrefix(alias, "/docs/") && alias != "/docs" {
		if _, exists := routes["/docs"+alias]; !exists {
			routes["/docs"+alias] = route
		}
	}
}

func normalizeStarlightRouteAlias(route string) string {
	route = strings.TrimSpace(route)
	if route == "" || route == "." {
		return "/"
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	route = filepath.ToSlash(filepath.Clean(route))
	if route == "." {
		return "/"
	}
	route = strings.TrimSuffix(route, ".html")
	route = strings.TrimSuffix(route, ".mdx")
	route = strings.TrimSuffix(route, ".md")
	if strings.HasSuffix(route, "/index") {
		route = strings.TrimSuffix(route, "/index")
	}
	if route != "/" {
		route = strings.TrimRight(route, "/")
	}
	return route
}

type starlightMigrationFinding struct {
	File    string
	Pattern string
	Detail  string
}

type starlightImportPage struct {
	SourceRel string
	OutputRel string
	Source    string
	Locale    string
	Modified  time.Time
}

type starlightConvertedPage struct {
	Page    starlightImportPage
	Content string
}

func dedupeStarlightPages(pages []starlightImportPage, findings *[]starlightMigrationFinding) []starlightImportPage {
	seen := map[string]starlightImportPage{}
	out := make([]starlightImportPage, 0, len(pages))
	for _, page := range pages {
		route := starlightMPressRoute(page)
		if kept, exists := seen[route]; exists {
			originalRoute := route
			page.OutputRel = starlightCollisionOutputRel(page.OutputRel, seen)
			route = starlightMPressRoute(page)
			*findings = append(*findings, starlightMigrationFinding{
				File:    page.SourceRel,
				Pattern: "route-collision",
				Detail:  fmt.Sprintf("Preserved at %q because %q is already provided by %s.", route, originalRoute, kept.SourceRel),
			})
		}
		seen[route] = page
		out = append(out, page)
	}
	return out
}

func starlightCollisionOutputRel(outputRel string, seen map[string]starlightImportPage) string {
	ext := filepath.Ext(outputRel)
	base := strings.TrimSuffix(outputRel, ext)
	if strings.HasSuffix(filepath.ToSlash(base), "/index") {
		base = strings.TrimSuffix(filepath.ToSlash(base), "/index") + "-migrated"
	} else {
		base += "-migrated"
	}
	for index := 1; ; index++ {
		candidate := base + ext
		if index > 1 {
			candidate = fmt.Sprintf("%s-%d%s", base, index, ext)
		}
		if _, exists := seen[normalizeStarlightRouteAlias(starlightRouteFromOutputRel(candidate))]; !exists {
			return candidate
		}
	}
}

func starlightMPressRoute(page starlightImportPage) string {
	route := strings.Trim(starlightRouteFromOutputRel(page.OutputRel), "/")
	if route == "" {
		return "/"
	}
	return normalizeStarlightRouteAlias(route)
}

type starlightMigrationReport struct {
	SourceDir    string
	ContentDir   string
	Pages        int
	StaticAssets int
	Findings     []starlightMigrationFinding
}

// Starlight/MDX conversion patterns
var (
	// JSX import statements: import { Component } from '...'
	jsxImportRegex      = regexp.MustCompile(`(?m)^import\s+.*from\s+['"].*['"];?\s*$`)
	jsxAssetImportRegex = regexp.MustCompile(`(?m)^import\s+(\w+)\s+from\s+['"]([^'"]+)['"];?\s*$`)
	// JSX self-closing tags: <Component prop="val" />
	jsxSelfCloseRegex = regexp.MustCompile(`<(\w+)\s+([^>]*?)/>`)
	// Starlight <Aside type="tip"> component
	slAsideOpenRegex  = regexp.MustCompile(`<Aside\s+type=["'](\w+)["']\s*(?:title=["']([^"']*)["'])?\s*>`)
	slAsideCloseRegex = regexp.MustCompile(`</Aside>`)
	// Starlight <Tabs>/<TabItem> and <Card>/<CardGrid> components
	slTabsOpenRegex      = regexp.MustCompile(`<Tabs\b([^>]*)>`)
	slTabsCloseRegex     = regexp.MustCompile(`</Tabs>`)
	slTabItemOpenRegex   = regexp.MustCompile(`<TabItem\s+label=["']([^"']+)["'][^>]*>`)
	slTabItemCloseRegex  = regexp.MustCompile(`</TabItem>`)
	slCardGridRegex      = regexp.MustCompile(`(?s)<CardGrid>\s*(.*?)\s*</CardGrid>`)
	slCardRegex          = regexp.MustCompile(`(?s)<Card\s+([^>]*)>\s*(.*?)\s*</Card>`)
	slLinkCardRegex      = regexp.MustCompile(`(?s)<LinkCard\b([^>]*?)/>`)
	slAttrRegex          = regexp.MustCompile(`(\w+)=(?:"([^"]*)"|'([^']*)')`)
	slExprAttrRegex      = regexp.MustCompile(`(\w+)=\{([^}]+)\}`)
	slInnerLinkRegex     = regexp.MustCompile(`(?m)\[([^\]]+)\]\(([^)]+)\)`)
	slStepsBlockRegex    = regexp.MustCompile(`(?s)<Steps>\s*(.*?)\s*</Steps>`)
	slStepsOpenRegex     = regexp.MustCompile(`<Steps>`)
	slStepsCloseRegex    = regexp.MustCompile(`</Steps>`)
	slFileTreeOpenRegex  = regexp.MustCompile(`<FileTree>`)
	slFileTreeCloseRegex = regexp.MustCompile(`</FileTree>`)
	slBadgeRegex         = regexp.MustCompile(`<Badge\s+([^>]*)/>`)
	slImageRegex         = regexp.MustCompile(`<Image\s+([^>]*)/>`)
	slMarkdownImageRegex = regexp.MustCompile(`!\[[^\]]*\]\((/assets/[^)\s]+)[^)]*\)`)
	slLinkButtonRegex    = regexp.MustCompile(`(?s)<LinkButton\s+([^>]*)>\s*(.*?)\s*</LinkButton>`)
	slContributorsRegex  = regexp.MustCompile(`<Contributors\s*/>`)
	slShowcaseRegex      = regexp.MustCompile(`(?s)<ShowcaseImage\s+entries=\{\[(.*?)\]\}\s*/>`)
	slShowcaseEntryRegex = regexp.MustCompile(`(?s)\{\s*thumbnail:\s*import\(\s*["']([^"']+)["']\s*\),\s*href:\s*["']([^"']+)["']\s*,\s*title:\s*["']([^"']+)["']\s*,?\s*\}`)
	slCardClassSelector  = regexp.MustCompile(`(^|[^A-Za-z0-9_-])\.card([^A-Za-z0-9_-]|$)`)
	// Starlight native Markdown admonition: :::tip[Title] or :::tip (no title)
	slAdmonitionTitled = regexp.MustCompile(`(?m)^[ \t]*:::(tip|note|caution|warning|danger|info|important)\[([^\]]+)\]\s*$`)
	slAdmonitionPlain  = regexp.MustCompile(`(?m)^[ \t]*:::(tip|note|caution|warning|danger|info|important)\s*$`)
	slAdmonitionClose  = regexp.MustCompile(`(?m)^[ \t]+:::\s*$`)
	// Starlight-specific frontmatter
	slSidebarFM         = regexp.MustCompile(`(?m)^sidebar:\s*\n(?:\s+.*\n)*`)
	slHeroFM            = regexp.MustCompile(`(?m)^hero:\s*\n(?:\s+.*\n)*`)
	slHeroAssetPathLine = regexp.MustCompile(`(?m)^(\s+(?:src|light|dark):\s*)(["']?)([^"'\n#]+)(["']?)\s*$`)
	slTableOfContentsFM = regexp.MustCompile(`(?m)^tableOfContents:\s.*$`)
	slSlugFM            = regexp.MustCompile(`(?m)^slug:\s*["']?[^"'\n]+["']?\s*$`)
	jsxElementRegex     = regexp.MustCompile(`<([A-Z][A-Za-z0-9_.]*)\b[^>]*>`)
	markdownLinkRegex   = regexp.MustCompile(`(!?\[[^\]]*\]\()([^)\s]+)((?:\s+"[^"]*")?\))`)
	htmlHrefRegex       = regexp.MustCompile(`(?i)(href\s*=\s*)(["'])([^"']+)(["'])`)
	jsxCommentRegex     = regexp.MustCompile(`(?s)\{/\*.*?\*/\}`)
	slMorphTextRegex    = regexp.MustCompile(`(?m)^\s*<MorphText\s*/>\s*$`)
	slReactStyleRegex   = regexp.MustCompile(`style=\{\{\s*([^}]*)\s*\}\}`)
	slReactStyleEntry   = regexp.MustCompile(`([A-Za-z][A-Za-z0-9]*):\s*["']([^"']*)["']`)
)

// starlightAdmonitionType maps Starlight type names to M-Press note types.
var starlightAdmonitionType = map[string]string{
	"tip": "tip", "note": "info", "info": "info",
	"caution": "caution", "warning": "warning",
	"danger": "danger", "important": "important",
}

func convertStarlightContent(content string) string {
	return convertStarlightContentWithRoutes(content, "", nil)
}

func convertStarlightContentWithRoutes(content string, currentRoute string, routes map[string]string) string {
	content, fencedCode := protectStarlightFencedCode(content)
	assetImports := parseStarlightAssetImports(content)

	content = convertStarlightAssetComponents(content, assetImports, currentRoute)
	content = convertStarlightHeroAssetPaths(content)
	content = convertStarlightStyleBlocks(content)
	content = convertStarlightReactStyles(content)
	content = convertStarlightShowcase(content)
	// MorphText is the Wails homepage word rotation layered on top of the
	// frontmatter-driven hero. Preserve the interaction as framework-free JS.
	content = slMorphTextRegex.ReplaceAllString(content, starlightMorphTextScript())
	content = stripStarlightJSXComments(content)

	// Strip JSX imports (not supported in plain Markdown)
	content = jsxImportRegex.ReplaceAllString(content, "")

	// Convert Starlight native Markdown admonitions: :::tip[Title] → :::note{type="tip" title="Title"}
	content = slAdmonitionTitled.ReplaceAllStringFunc(content, func(match string) string {
		m := slAdmonitionTitled.FindStringSubmatch(match)
		mtype := starlightAdmonitionType[strings.ToLower(m[1])]
		if mtype == "" {
			mtype = strings.ToLower(m[1])
		}
		return fmt.Sprintf(`:::note{type="%s" title="%s"}`, mtype, m[2])
	})
	// :::tip (no title) → :::note{type="tip"}
	content = slAdmonitionPlain.ReplaceAllStringFunc(content, func(match string) string {
		m := slAdmonitionPlain.FindStringSubmatch(match)
		mtype := starlightAdmonitionType[strings.ToLower(m[1])]
		if mtype == "" {
			mtype = strings.ToLower(m[1])
		}
		return fmt.Sprintf(`:::note{type="%s"}`, mtype)
	})
	content = slAdmonitionClose.ReplaceAllString(content, ":::")

	// Convert <Aside type="tip"> → :::note{type="tip"}
	content = slAsideOpenRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slAsideOpenRegex.FindStringSubmatch(match)
		atype := "info"
		if len(m) > 1 {
			atype = m[1]
		}
		atitle := ""
		if len(m) > 2 {
			atitle = m[2]
		}
		if atitle != "" {
			return fmt.Sprintf(":::note{type=\"%s\" title=\"%s\"}\n", atype, atitle)
		}
		return fmt.Sprintf(":::note{type=\"%s\"}\n", atype)
	})
	content = slAsideCloseRegex.ReplaceAllString(content, "\n:::")

	content = convertStarlightSteps(content)
	content = slFileTreeOpenRegex.ReplaceAllString(content, ":::filetree")
	content = slFileTreeCloseRegex.ReplaceAllString(content, ":::")
	content = slBadgeRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slBadgeRegex.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		text := starlightAttrValue(m[1], "text")
		variant := starlightAttrValue(m[1], "variant")
		if variant == "" {
			variant = "default"
		}
		if text == "" {
			text = variant
		}
		return fmt.Sprintf(`<span class="mpress-badge mpress-badge-%s">%s</span>`, variant, text)
	})

	content = convertStarlightLinkCards(content)
	content = convertStarlightCards(content)

	// Convert <Tabs>/<TabItem> → :::tabs with [Label] syntax
	content = slTabsOpenRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slTabsOpenRegex.FindStringSubmatch(match)
		if len(m) < 2 {
			return ":::tabs"
		}
		if syncKey := starlightAttrValue(m[1], "syncKey"); syncKey != "" {
			return fmt.Sprintf(":::tabs{sync-key=%s}\n", syncKey)
		}
		return ":::tabs\n"
	})
	content = slTabsCloseRegex.ReplaceAllString(content, "\n:::endtabs")
	content = slTabItemOpenRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slTabItemOpenRegex.FindStringSubmatch(match)
		if len(m) > 1 {
			return "\n[" + m[1] + "]\n"
		}
		return match
	})
	content = slTabItemCloseRegex.ReplaceAllString(content, "\n")
	content = normalizeStarlightTabBlocks(content)

	// Strip Starlight-specific frontmatter blocks
	content = slSidebarFM.ReplaceAllString(content, "")
	content = slTableOfContentsFM.ReplaceAllString(content, "")
	content = slSlugFM.ReplaceAllString(content, "")

	content = rewriteStarlightMarkdownLinks(content, currentRoute, routes)
	content = rewriteStarlightHTMLLinks(content, currentRoute, routes)
	content = normalizeStarlightDirectiveBlocks(content)

	// Clean up excessive blank lines from stripping
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}

	return restoreStarlightFencedCode(content, fencedCode)
}

// convertStarlightReactStyles makes ordinary HTML elements in MDX portable to
// Markdown renderers. Wails uses React-style object attributes on sponsor
// images and objects, which otherwise leak into the generated HTML as text.
func convertStarlightReactStyles(content string) string {
	return slReactStyleRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := slReactStyleEntry.FindAllStringSubmatch(match, -1)
		if len(parts) == 0 {
			return match
		}
		styles := make([]string, 0, len(parts))
		for _, part := range parts {
			property := starlightCSSProperty(part[1])
			styles = append(styles, property+": "+part[2])
		}
		return `style="` + strings.Join(styles, "; ") + `"`
	})
}

func starlightCSSProperty(property string) string {
	var b strings.Builder
	for i, r := range property {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func starlightMorphTextScript() string {
	return `<script>
(() => {
  const words = ['Desktop', 'Server', 'Mobile'];
  let index = 0;
  const init = () => {
    const title = document.querySelector('.mpress-frontmatter-hero-title');
    if (!title || title.querySelector('.morph-word-title')) return;
    if (title.textContent.trim() !== 'Build Desktop Apps with Go') return;
    const titleLine = document.createElement('span');
    titleLine.className = 'morph-title-line';
    const prefix = document.createElement('span');
    prefix.textContent = 'Build';
    const wordWrap = document.createElement('span');
    wordWrap.className = 'morph-word-title';
    const activeWord = document.createElement('span');
    activeWord.textContent = words[0];
    wordWrap.append(activeWord);
    titleLine.append(prefix, wordWrap);
    title.replaceChildren(titleLine, document.createElement('br'), document.createTextNode('Apps with Go'));
    const tagline = document.querySelector('.mpress-frontmatter-hero-tagline');
    if (tagline && !tagline.querySelector('.browser-footnote')) {
      const note = document.createElement('span');
      note.className = 'browser-footnote';
      note.textContent = '* unless you really want to';
      tagline.appendChild(note);
    }
    const rotate = () => {
      const word = title.querySelector('.morph-word-title span');
      if (!word) return;
      word.classList.add('morph-word-out');
      setTimeout(() => {
        index = (index + 1) % words.length;
        word.textContent = words[index];
        word.classList.remove('morph-word-out');
        setTimeout(rotate, 3600);
      }, 650);
    };
    setTimeout(rotate, 3600);
  };
  document.readyState === 'loading' ? document.addEventListener('DOMContentLoaded', init) : init();
})();
</script>`
}

type starlightFencedCodeBlock struct {
	Token string
	Lines []string
}

// protectStarlightFencedCode keeps migration rewrites out of examples. MDX and
// JavaScript import syntax is common in Wails code samples, and treating those
// lines as document-level MDX silently deletes real documentation.
func protectStarlightFencedCode(content string) (string, []starlightFencedCodeBlock) {
	lines := strings.Split(content, "\n")
	var blocks []starlightFencedCodeBlock
	var out []string
	for i := 0; i < len(lines); {
		indent, marker, ok := starlightFenceOpening(lines[i])
		if !ok {
			out = append(out, lines[i])
			i++
			continue
		}

		end := i + 1
		for end < len(lines) && !starlightFenceClosing(lines[end], marker) {
			end++
		}
		if end >= len(lines) {
			out = append(out, lines[i:]...)
			break
		}

		token := fmt.Sprintf("MPRESS_STARLIGHT_FENCED_CODE_%06d", len(blocks))
		blockLines := append([]string(nil), lines[i:end+1]...)
		for j, line := range blockLines {
			if strings.HasPrefix(line, indent) {
				blockLines[j] = strings.TrimPrefix(line, indent)
			}
		}
		blocks = append(blocks, starlightFencedCodeBlock{Token: token, Lines: blockLines})
		out = append(out, indent+token)
		i = end + 1
	}
	return strings.Join(out, "\n"), blocks
}

func starlightFenceOpening(line string) (string, string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	if trimmed == "" || trimmed[0] != '`' && trimmed[0] != '~' {
		return "", "", false
	}
	markerLength := 0
	for markerLength < len(trimmed) && trimmed[markerLength] == trimmed[0] {
		markerLength++
	}
	if markerLength < 3 {
		return "", "", false
	}
	return indent, trimmed[:markerLength], true
}

func starlightFenceClosing(line string, marker string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < len(marker) || trimmed[0] != marker[0] {
		return false
	}
	markerLength := 0
	for markerLength < len(trimmed) && trimmed[markerLength] == marker[0] {
		markerLength++
	}
	return markerLength >= len(marker) && strings.TrimSpace(trimmed[markerLength:]) == ""
}

func restoreStarlightFencedCode(content string, blocks []starlightFencedCodeBlock) string {
	for _, block := range blocks {
		tokenLine := regexp.MustCompile(`(?m)^([ \t]*)` + regexp.QuoteMeta(block.Token) + `$`)
		content = tokenLine.ReplaceAllStringFunc(content, func(match string) string {
			indent := strings.TrimSuffix(match, block.Token)
			restored := make([]string, len(block.Lines))
			for i, line := range block.Lines {
				if line == "" {
					restored[i] = ""
				} else {
					restored[i] = indent + line
				}
			}
			return strings.Join(restored, "\n")
		})
	}
	return content
}

func convertStarlightStyleBlocks(content string) string {
	content = strings.ReplaceAll(content, "<style>{`", "<style>")
	content = strings.ReplaceAll(content, "`}</style>", "</style>")
	content = rewriteStarlightStyleSelectors(content)
	return content
}

func rewriteStarlightStyleSelectors(content string) string {
	content = strings.ReplaceAll(content, ".card-grid", ".mpress-cards")
	content = slCardClassSelector.ReplaceAllString(content, `${1}.mpress-card${2}`)
	return content
}

func convertStarlightHeroAssetPaths(content string) string {
	return slHeroAssetPathLine.ReplaceAllStringFunc(content, func(line string) string {
		m := slHeroAssetPathLine.FindStringSubmatch(line)
		if len(m) != 5 {
			return line
		}
		path := strings.TrimSpace(m[3])
		asset := starlightAssetPath(path)
		if asset == "" {
			return line
		}
		return m[1] + m[2] + asset + m[4]
	})
}

func convertStarlightCards(content string) string {
	content = slCardGridRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slCardGridRegex.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		cards := parseStarlightCards(m[1])
		if len(cards) == 0 {
			body := strings.TrimSpace(m[1])
			if strings.Contains(body, ":::linkcard") {
				return ":::container{display=\"grid\" columns=\"2\" gap=\"1rem\"}\n" + body + "\n:::"
			}
			return body
		}
		var b strings.Builder
		b.WriteString(":::cards{cols=\"2\"}\n")
		for i, card := range cards {
			if i > 0 {
				b.WriteString("\n\n---\n\n")
			}
			b.WriteString(card.toMarkdown())
		}
		b.WriteString("\n:::")
		return b.String()
	})

	return slCardRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slCardRegex.FindStringSubmatch(match)
		if len(m) != 3 {
			return match
		}
		card := buildStarlightCard(m[1], m[2])
		return ":::cards{cols=\"1\"}\n" + card.toMarkdown() + "\n:::"
	})
}

func convertStarlightLinkCards(content string) string {
	return slLinkCardRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := slLinkCardRegex.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		attrs := parts[1]
		values := [][2]string{
			{"title", starlightAttrValue(attrs, "title")},
			{"href", starlightAttrValue(attrs, "href")},
			{"description", starlightAttrValue(attrs, "description")},
			{"icon", starlightAttrValue(attrs, "icon")},
		}
		var directive strings.Builder
		directive.WriteString(":::linkcard{")
		written := 0
		for _, value := range values {
			if value[1] == "" {
				continue
			}
			if written > 0 {
				directive.WriteByte(' ')
			}
			directive.WriteString(value[0])
			directive.WriteByte('=')
			directive.WriteString(strconv.Quote(strings.TrimSpace(value[1])))
			written++
		}
		directive.WriteString("}\n:::")
		return directive.String()
	})
}

func convertStarlightSteps(content string) string {
	return slStepsBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slStepsBlockRegex.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		body := strings.TrimSpace(m[1])
		return ":::steps\n\n" + body + "\n\n:::"
	})
}

func parseStarlightAssetImports(content string) map[string]string {
	assets := make(map[string]string)
	for _, match := range jsxAssetImportRegex.FindAllStringSubmatch(content, -1) {
		if len(match) != 3 {
			continue
		}
		if path := starlightAssetPath(match[2]); path != "" {
			assets[match[1]] = path
		}
	}
	return assets
}

func starlightAssetPath(importPath string) string {
	cleanPath := filepath.ToSlash(filepath.Clean(importPath))
	if idx := strings.Index(cleanPath, "assets/"); idx >= 0 {
		return "/" + cleanPath[idx:]
	}
	if idx := strings.Index(cleanPath, "public/"); idx >= 0 {
		return "/" + strings.TrimPrefix(cleanPath[idx+len("public/"):], "/")
	}
	return ""
}

func ensureStarlightBlogImage(content string) string {
	if regexp.MustCompile(`(?m)^image:\s*`).MatchString(content) {
		return content
	}
	m := slMarkdownImageRegex.FindStringSubmatch(content)
	if len(m) != 2 {
		return content
	}
	image := m[1]
	if strings.HasPrefix(content, "---\n") {
		end := strings.Index(content[4:], "\n---")
		if end >= 0 {
			insertAt := 4 + end
			return content[:insertAt] + "\nimage: " + image + content[insertAt:]
		}
	}
	return fmt.Sprintf("---\nimage: %s\n---\n\n%s", image, content)
}

func convertStarlightAssetComponents(content string, assets map[string]string, currentRoute string) string {
	content = slImageRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slImageRegex.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		attrs := m[1]
		src := starlightAttrValue(attrs, "src")
		if src == "" {
			src = starlightExprAttrValue(attrs, "src")
			src = assets[strings.TrimSpace(src)]
		}
		if src == "" {
			return match
		}
		alt := starlightAttrValue(attrs, "alt")
		return fmt.Sprintf("![%s](%s)", alt, src)
	})

	content = slLinkButtonRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slLinkButtonRegex.FindStringSubmatch(match)
		if len(m) != 3 {
			return match
		}
		attrs := m[1]
		href := starlightAttrValue(attrs, "href")
		label := strings.TrimSpace(m[2])
		if href == "" || label == "" {
			return label
		}
		variant := starlightAttrValue(attrs, "variant")
		if variant == "" {
			variant = "primary"
		}
		icon := starlightAttrValue(attrs, "icon")
		return starlightLinkButtonHTML(href, label, variant, icon)
	})

	content = slContributorsRegex.ReplaceAllString(content, starlightContributorsInclude(currentRoute))

	return slExprAttrRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slExprAttrRegex.FindStringSubmatch(match)
		if len(m) != 3 || m[1] != "src" {
			return match
		}
		if asset := assets[strings.TrimSpace(m[2])]; asset != "" {
			return fmt.Sprintf(`src="%s"`, asset)
		}
		return match
	})
}

func starlightLinkButtonHTML(href, label, variant, icon string) string {
	class := "mpress-link-button mpress-link-button-" + html.EscapeString(variant)
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<a class="%s" href="%s">`, class, html.EscapeString(href)))
	b.WriteString(`<span>`)
	b.WriteString(html.EscapeString(label))
	b.WriteString(`</span>`)
	if icon != "" {
		b.WriteString(`<span class="mpress-link-button-icon" aria-hidden="true">`)
		b.WriteString(starlightLinkButtonIcon(icon))
		b.WriteString(`</span>`)
	}
	b.WriteString(`</a>`)
	return b.String()
}

func starlightLinkButtonIcon(icon string) string {
	switch icon {
	case "right-arrow", "arrow-right":
		return `<svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M17.92 11.62a1 1 0 0 0-.21-.33l-5-5a1 1 0 1 0-1.42 1.42l3.3 3.29H7a1 1 0 0 0 0 2h7.59l-3.3 3.29a1 1 0 1 0 1.42 1.42l5-5a1 1 0 0 0 .21-1.09Z"/></svg>`
	default:
		return html.EscapeString(icon)
	}
}

func starlightContributorsInclude(currentRoute string) string {
	depth := 0
	route := strings.Trim(currentRoute, "/")
	if route != "" {
		depth = strings.Count(route, "/")
	}
	return fmt.Sprintf(`{{include "%sstatic/assets/contributors.html"}}`, strings.Repeat("../", depth+1))
}

func convertStarlightShowcase(content string) string {
	return slShowcaseRegex.ReplaceAllStringFunc(content, func(match string) string {
		m := slShowcaseRegex.FindStringSubmatch(match)
		if len(m) != 2 {
			return match
		}
		entries := slShowcaseEntryRegex.FindAllStringSubmatch(m[1], -1)
		if len(entries) == 0 {
			return match
		}
		var b strings.Builder
		b.WriteString(`<div class="mpress-showcase-grid">` + "\n")
		for _, entry := range entries {
			if len(entry) != 4 {
				continue
			}
			image := strings.TrimPrefix(starlightAssetPath(entry[1]), "/")
			if image == "" {
				continue
			}
			href := htmlEscapeAttr(entry[2])
			title := htmlEscapeText(entry[3])
			b.WriteString(fmt.Sprintf(`<a class="mpress-showcase-card" href="%s"><img src="/%s" alt="%s screenshot" loading="lazy"><span>%s</span></a>`+"\n", href, htmlEscapeAttr(image), title, title))
		}
		b.WriteString("</div>")
		return b.String()
	})
}

func htmlEscapeText(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	return value
}

func htmlEscapeAttr(value string) string {
	value = htmlEscapeText(value)
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return value
}

func normalizeStarlightTabLabels(content string) string {
	return regexp.MustCompile(`(?m)^[ \t]+\[([^\]]+)\][ \t]*$`).ReplaceAllString(content, "[$1]")
}

func normalizeStarlightTabBlocks(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	var block []string
	inTab := false
	directiveDepth := 0

	flush := func() {
		if len(block) == 0 {
			return
		}
		minIndent := -1
		for i, line := range block {
			if strings.TrimSpace(line) == "" {
				continue
			}
			trimmed := strings.TrimSpace(line)
			if i == 0 || (i == len(block)-1 && trimmed == ":::") {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if minIndent == -1 || indent < minIndent {
				minIndent = indent
			}
		}
		if minIndent < 0 {
			minIndent = 0
		}
		for i, line := range block {
			trimmed := strings.TrimSpace(line)
			if i == 0 || (i == len(block)-1 && trimmed == ":::") {
				out = append(out, trimmed)
			} else if len(line) >= minIndent {
				out = append(out, line[minIndent:])
			} else {
				out = append(out, strings.TrimLeft(line, " \t"))
			}
		}
		block = nil
		directiveDepth = 0
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inTab && strings.HasPrefix(trimmed, ":::tabs") {
			inTab = true
			block = append(block, line)
			continue
		}
		if inTab {
			block = append(block, line)
			if strings.HasPrefix(trimmed, ":::") && trimmed != ":::" {
				directiveDepth++
				continue
			}
			if trimmed == ":::" || strings.HasPrefix(trimmed, ":::endtabs") {
				if directiveDepth > 0 {
					directiveDepth--
					continue
				}
				flush()
				inTab = false
			}
			continue
		}
		out = append(out, line)
	}
	flush()
	return normalizeStarlightTabLabels(strings.Join(out, "\n"))
}

func normalizeStarlightDirectiveBlocks(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	var block []string
	inBlock := false

	flush := func() {
		if len(block) == 0 {
			return
		}
		minIndent := -1
		for i, line := range block {
			if strings.TrimSpace(line) == "" {
				continue
			}
			trimmed := strings.TrimSpace(line)
			if i == 0 || (i == len(block)-1 && trimmed == ":::") {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if minIndent == -1 || indent < minIndent {
				minIndent = indent
			}
		}
		if minIndent < 0 {
			minIndent = 0
		}
		for i, line := range block {
			trimmed := strings.TrimSpace(line)
			if i == 0 || (i == len(block)-1 && trimmed == ":::") {
				out = append(out, trimmed)
			} else if len(line) >= minIndent {
				out = append(out, line[minIndent:])
			} else {
				out = append(out, strings.TrimLeft(line, " \t"))
			}
		}
		block = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock && strings.HasPrefix(trimmed, ":::") && trimmed != ":::" {
			inBlock = true
			block = append(block, line)
			continue
		}
		if inBlock {
			block = append(block, line)
			if trimmed == ":::" {
				flush()
				inBlock = false
			}
			continue
		}
		out = append(out, line)
	}
	flush()
	return strings.Join(out, "\n")
}

func stripStarlightJSXComments(content string) string {
	return jsxCommentRegex.ReplaceAllStringFunc(content, func(match string) string {
		if strings.Contains(match, "\n") || regexp.MustCompile(`(?m)^[ \t]*`+regexp.QuoteMeta(match)+`[ \t]*$`).MatchString(content) {
			return ""
		}
		return match
	})
}

func flattenStarlightTabsRepeated(content string) string {
	for i := 0; i < 8; i++ {
		next := flattenStarlightTabs(normalizeStarlightTabBlocks(content))
		if next == content {
			return next
		}
		content = next
	}
	return content
}

func flattenStarlightTabs(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	var block []string
	inTab := false
	nestedDepth := 0

	flush := func() {
		if len(block) == 0 {
			return
		}
		tabBlockCount := 0
		for _, line := range block {
			if strings.HasPrefix(strings.TrimSpace(line), ":::tabs") {
				tabBlockCount++
			}
		}
		if tabBlockCount <= 1 {
			out = append(out, block...)
			block = nil
			nestedDepth = 0
			return
		}
		skipTabsDepth := 0
		for i, line := range block {
			trimmed := strings.TrimSpace(line)
			if i == 0 || (i == len(block)-1 && (trimmed == ":::" || strings.HasPrefix(trimmed, ":::endtabs"))) {
				continue
			}
			if strings.HasPrefix(trimmed, ":::tabs") {
				skipTabsDepth++
				continue
			}
			if trimmed == ":::" && skipTabsDepth > 0 {
				skipTabsDepth--
				continue
			}
			if m := regexp.MustCompile(`^\[([^\]]+)\]$`).FindStringSubmatch(trimmed); len(m) == 2 {
				out = append(out, "#### "+m[1])
				continue
			}
			line = strings.TrimPrefix(line, "    ")
			out = append(out, line)
		}
		block = nil
		nestedDepth = 0
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inTab && strings.HasPrefix(trimmed, ":::tabs") {
			inTab = true
			block = append(block, line)
			continue
		}
		if inTab {
			if strings.HasPrefix(trimmed, ":::tabs") {
				nestedDepth++
				block = append(block, line)
				continue
			}
			block = append(block, line)
			if strings.HasPrefix(trimmed, ":::note") {
				nestedDepth++
				continue
			}
			if trimmed == ":::" || strings.HasPrefix(trimmed, ":::endtabs") {
				if nestedDepth > 0 {
					nestedDepth--
					continue
				}
				flush()
				inTab = false
			}
			continue
		}
		out = append(out, line)
	}
	flush()
	return strings.Join(out, "\n")
}

type starlightCard struct {
	Title       string
	Icon        string
	Link        string
	Description string
}

func parseStarlightCards(content string) []starlightCard {
	matches := slCardRegex.FindAllStringSubmatch(content, -1)
	cards := make([]starlightCard, 0, len(matches))
	for _, match := range matches {
		if len(match) != 3 {
			continue
		}
		cards = append(cards, buildStarlightCard(match[1], match[2]))
	}
	return cards
}

func buildStarlightCard(attrs string, body string) starlightCard {
	card := starlightCard{
		Title: strings.TrimSpace(starlightAttrValue(attrs, "title")),
		Icon:  starlightIconEmoji(starlightAttrValue(attrs, "icon")),
		Link:  strings.TrimSpace(starlightAttrValue(attrs, "href")),
	}
	body = dedentStarlightCardBody(body)
	body = strings.TrimSpace(body)
	card.Description = strings.TrimSpace(body)
	if card.Title == "" && card.Description != "" {
		lines := strings.Split(card.Description, "\n")
		card.Title = strings.TrimSpace(lines[0])
		card.Description = strings.TrimSpace(strings.Join(lines[1:], "\n"))
	}
	return card
}

func dedentStarlightCardBody(body string) string {
	lines := strings.Split(strings.Trim(body, "\r\n"), "\n")
	minIndent := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if i == 0 && indent == 0 {
			continue
		}
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return strings.Join(lines, "\n")
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent >= minIndent && len(line) >= minIndent {
			lines[i] = line[minIndent:]
		} else {
			lines[i] = strings.TrimLeft(line, " \t")
		}
	}
	return strings.Join(lines, "\n")
}

func (c starlightCard) toMarkdown() string {
	title := c.Title
	if c.Icon != "" {
		title = c.Icon + " " + title
	}
	if c.Link != "" {
		title = fmt.Sprintf("[%s](%s)", title, c.Link)
	} else {
		title = strings.TrimSpace(title)
	}
	if c.Description == "" {
		return title
	}
	return title + "\n" + c.Description
}

func starlightAttrValue(attrs string, name string) string {
	for _, match := range slAttrRegex.FindAllStringSubmatch(attrs, -1) {
		if len(match) != 4 || match[1] != name {
			continue
		}
		if match[2] != "" {
			return match[2]
		}
		return match[3]
	}
	return ""
}

func starlightExprAttrValue(attrs string, name string) string {
	for _, match := range slExprAttrRegex.FindAllStringSubmatch(attrs, -1) {
		if len(match) == 3 && match[1] == name {
			return strings.TrimSpace(match[2])
		}
	}
	return ""
}

func starlightIconEmoji(icon string) string {
	switch strings.ToLower(strings.TrimSpace(icon)) {
	case "rocket":
		return "🚀"
	case "star":
		return "★"
	case "puzzle":
		return "◆"
	case "open-book", "document":
		return "📖"
	case "github":
		return "◆"
	case "discord":
		return "◆"
	case "information":
		return "ℹ"
	case "bell":
		return "●"
	case "laptop":
		return "▣"
	case "heart":
		return "♥"
	case "setting":
		return "⚙"
	case "approve-check":
		return "✓"
	case "seti:config":
		return "⚙"
	case "seti:go":
		return "◇"
	case "seti:html":
		return "▤"
	case "terminal":
		return "▸"
	case "list-format":
		return "☰"
	case "down-caret":
		return "▼"
	default:
		return ""
	}
}

func rewriteStarlightMarkdownLinks(content string, currentRoute string, routes map[string]string) string {
	return markdownLinkRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := markdownLinkRegex.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		target := parts[2]
		if shouldSkipLinkRewrite(target) {
			return match
		}
		rewritten := rewriteStarlightMarkdownLinkTarget(target, currentRoute, routes)
		if rewritten == target {
			return match
		}
		return parts[1] + rewritten + parts[3]
	})
}

// rewriteStarlightHTMLLinks handles links emitted by component conversions
// (notably LinkButton) after the original JSX has become portable HTML.
func rewriteStarlightHTMLLinks(content string, currentRoute string, routes map[string]string) string {
	return htmlHrefRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := htmlHrefRegex.FindStringSubmatch(match)
		if len(parts) != 5 || parts[2] != parts[4] || shouldSkipLinkRewrite(parts[3]) {
			return match
		}
		rewritten := rewriteStarlightMarkdownLinkTarget(parts[3], currentRoute, routes)
		if rewritten == parts[3] {
			return match
		}
		return parts[1] + parts[2] + rewritten + parts[4]
	})
}

func starlightAnchorMap(pages []starlightConvertedPage) map[string]map[string]bool {
	anchors := make(map[string]map[string]bool, len(pages))
	renderer := content.NewRenderer()
	for _, converted := range pages {
		page, _, err := renderer.Parse(converted.Page.OutputRel, "en", converted.Content)
		if err != nil {
			continue
		}
		route := normalizeStarlightRouteAlias(starlightMPressRoute(converted.Page))
		ids := map[string]bool{"content": true}
		for _, heading := range page.Headings {
			ids[heading.ID] = true
		}
		anchors[route] = ids
	}
	return anchors
}

// Starlight sites in the wild often retain stale heading fragments after a
// heading is translated or renamed. Preserve the page link, drop only a
// provably invalid fragment, and record the fallback for human review.
func rewriteInvalidStarlightFragments(source, currentRoute string, routes map[string]string, anchors map[string]map[string]bool, file string, findings *[]starlightMigrationFinding) string {
	rewrite := func(target string) string {
		if shouldSkipLinkRewrite(target) && !strings.HasPrefix(target, "#") {
			return target
		}
		hash := strings.Index(target, "#")
		if hash < 0 || hash == len(target)-1 {
			return target
		}
		fragment := target[hash+1:]
		decoded, err := url.PathUnescape(fragment)
		if err == nil {
			fragment = decoded
		}
		withoutFragment := target[:hash]
		pathTarget := withoutFragment
		if query := strings.Index(pathTarget, "?"); query >= 0 {
			pathTarget = pathTarget[:query]
		}
		route := normalizeStarlightRouteAlias(currentRoute)
		if pathTarget != "" {
			resolved, ok := resolveStarlightRouteTarget(pathTarget, currentRoute, routes)
			if !ok {
				return target
			}
			route = normalizeStarlightRouteAlias(resolved)
		}
		ids, exists := anchors[route]
		if !exists || ids[fragment] {
			return target
		}
		*findings = append(*findings, starlightMigrationFinding{
			File: file, Pattern: "stale-fragment",
			Detail: fmt.Sprintf("Removed missing fragment #%s from a link to %s; the page link was preserved.", fragment, route),
		})
		return withoutFragment
	}

	source = markdownLinkRegex.ReplaceAllStringFunc(source, func(match string) string {
		parts := markdownLinkRegex.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		return parts[1] + rewrite(parts[2]) + parts[3]
	})
	return htmlHrefRegex.ReplaceAllStringFunc(source, func(match string) string {
		parts := htmlHrefRegex.FindStringSubmatch(match)
		if len(parts) != 5 || parts[2] != parts[4] {
			return match
		}
		return parts[1] + parts[2] + rewrite(parts[3]) + parts[4]
	})
}

func shouldSkipLinkRewrite(target string) bool {
	lower := strings.ToLower(target)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(lower, "#")
}

func rewriteStarlightMarkdownLinkTarget(target string, currentRoute string, routes map[string]string) string {
	fragment := ""
	if idx := strings.Index(target, "#"); idx >= 0 {
		fragment = target[idx:]
		target = target[:idx]
	}
	if fragment == "#gtk4-support-experimental" {
		fragment = "#legacy-gtk3-support"
	}
	query := ""
	if idx := strings.Index(target, "?"); idx >= 0 {
		query = target[idx:]
		target = target[:idx]
	}
	ext := strings.ToLower(filepath.Ext(target))
	cleanTarget := filepath.ToSlash(filepath.Clean(target))
	if idx := strings.Index(cleanTarget, "assets/"); idx >= 0 {
		return "/" + cleanTarget[idx:] + query + fragment
	}
	if route, ok := resolveStarlightRouteTarget(target, currentRoute, routes); ok {
		if route != "/" {
			route = strings.TrimRight(route, "/") + "/"
		}
		return route + query + fragment
	}
	if ext != ".md" && ext != ".mdx" {
		return target + query + fragment
	}

	dir := filepath.Dir(target)
	base := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
	_, cleanName := parseNumericPrefix(base)
	slug := strings.ToLower(strings.ReplaceAll(cleanName, "_", "-"))
	rewritten := slug + ".html"
	if dir != "." && dir != "" {
		rewritten = filepath.ToSlash(filepath.Join(dir, rewritten))
	}
	return rewritten + query + fragment
}

func resolveStarlightRouteTarget(target string, currentRoute string, routes map[string]string) (string, bool) {
	if len(routes) == 0 || target == "" {
		return "", false
	}
	if strings.HasPrefix(target, "/") {
		if route, ok := routes[normalizeStarlightRouteAlias(target)]; ok {
			return route, true
		}
		return "", false
	}

	currentDir := filepath.Dir(strings.TrimPrefix(currentRoute, "/"))
	if currentDir == "." {
		currentDir = ""
	}
	combined := filepath.ToSlash(filepath.Clean(filepath.Join("/", currentDir, target)))
	if route, ok := routes[normalizeStarlightRouteAlias(combined)]; ok {
		return route, true
	}
	if strings.HasPrefix(target, "./") {
		rootCandidate := strings.TrimPrefix(target, "./")
		if route, ok := routes[normalizeStarlightRouteAlias(rootCandidate)]; ok {
			return route, true
		}
	}
	return "", false
}

func parseNumericPrefix(name string) (int, string) {
	m := regexp.MustCompile(`^(\d+)_(.+)$`).FindStringSubmatch(name)
	if len(m) != 3 {
		return 0, name
	}
	return 0, m[2]
}

func detectStarlightMigrationFindings(rel string, source string) []starlightMigrationFinding {
	source, _ = protectStarlightFencedCode(source)
	source = regexp.MustCompile("`[^`\\n]+`").ReplaceAllString(source, "MPRESS_STARLIGHT_INLINE_CODE")
	var findings []starlightMigrationFinding
	seen := make(map[string]bool)
	add := func(pattern, detail string) {
		key := pattern + "\x00" + detail
		if seen[key] {
			return
		}
		seen[key] = true
		findings = append(findings, starlightMigrationFinding{
			File:    rel,
			Pattern: pattern,
			Detail:  detail,
		})
	}
	for _, match := range jsxImportRegex.FindAllString(source, -1) {
		if isConvertedStarlightImport(match) {
			continue
		}
		add("mdx-import", strings.TrimSpace(match))
	}

	for _, match := range jsxSelfCloseRegex.FindAllStringSubmatch(source, -1) {
		component := match[1]
		if component == strings.ToLower(component) {
			continue
		}
		if isKnownConvertedStarlightComponent(component) {
			continue
		}
		add("jsx-component", strings.TrimSpace(match[0]))
	}

	for _, match := range jsxElementRegex.FindAllStringSubmatch(source, -1) {
		component := match[1]
		if isKnownConvertedStarlightComponent(component) {
			continue
		}
		add("jsx-component", strings.TrimSpace(match[0]))
	}

	return findings
}

func isConvertedStarlightImport(statement string) bool {
	for _, marker := range []string{
		"@astrojs/starlight/components",
		"astro:assets",
		"starlight-showcases",
		"@components/MorphText.astro",
		"/assets/",
	} {
		if strings.Contains(statement, marker) {
			return true
		}
	}
	return false
}

func isKnownConvertedStarlightComponent(name string) bool {
	switch name {
	case "Aside", "Tabs", "TabItem",
		"Card", "CardGrid", "LinkCard", "Steps", "FileTree", "Badge", "Image", "LinkButton", "Contributors", "ShowcaseImage", "MorphText":
		return true
	default:
		return false
	}
}

func writeStarlightMigrationReport(outputDir string, report starlightMigrationReport) error {
	sort.SliceStable(report.Findings, func(i, j int) bool {
		if report.Findings[i].File != report.Findings[j].File {
			return report.Findings[i].File < report.Findings[j].File
		}
		if report.Findings[i].Pattern != report.Findings[j].Pattern {
			return report.Findings[i].Pattern < report.Findings[j].Pattern
		}
		return report.Findings[i].Detail < report.Findings[j].Detail
	})

	var b strings.Builder
	b.WriteString("# Starlight Migration Report\n\n")
	b.WriteString(fmt.Sprintf("- Source: `%s`\n", report.SourceDir))
	b.WriteString(fmt.Sprintf("- Content source: `%s`\n", report.ContentDir))
	b.WriteString(fmt.Sprintf("- Pages imported: %d\n", report.Pages))
	b.WriteString(fmt.Sprintf("- Static assets copied: %d\n", report.StaticAssets))
	b.WriteString("- Config written: `mpress.yaml`\n\n")
	if len(report.Findings) == 0 {
		b.WriteString("No unsupported Starlight or MDX patterns were detected.\n")
	} else {
		b.WriteString("## Manual Review\n\n")
		b.WriteString("The importer found patterns that may need manual cleanup:\n\n")
		for _, finding := range report.Findings {
			b.WriteString(fmt.Sprintf("- `%s` — %s: %s\n", finding.File, finding.Pattern, finding.Detail))
		}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating migration report directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "migration-report.md"), []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("writing migration-report.md: %w", err)
	}
	return nil
}
