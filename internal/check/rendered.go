package check

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/config"
	"golang.org/x/net/html"
)

// RenderedOptions selects publication checks beyond the regular link checker.
type RenderedOptions struct {
	Output          string
	CloudflarePages bool
}

type RenderedReport struct {
	Files     int            `json:"files"`
	HTMLPages int            `json:"html_pages"`
	Languages map[string]int `json:"languages"`
	Redirects int            `json:"redirects"`
	Errors    []string       `json:"errors"`
}

type renderedPage struct {
	HTMLIndex
	language, canonical, source string
	d2, fence                   bool
}

// Rendered validates an existing build without rebuilding or making network requests.
func Rendered(project string, cfg config.Config, options RenderedOptions) (RenderedReport, error) {
	report := RenderedReport{Languages: map[string]int{}, Errors: []string{}}
	output := options.Output
	if output == "" {
		output = cfg.OutputPath(project)
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return report, err
	}
	pages, index, err := indexRenderedOutput(output, options.CloudflarePages, &report)
	if err != nil {
		return report, err
	}

	aliases := readRedirects(output, &report)
	report.Redirects = len(aliases)
	baseURL := strings.TrimRight(cfg.Site.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://mpress.invalid"
	}
	base, err := url.Parse(baseURL + "/")
	if err != nil {
		return report, err
	}
	for rel, page := range pages {
		checkRenderedPage(project, cfg, rel, page, &report)
		current := *base
		current.Path = strings.TrimSuffix(base.Path, "/") + renderedRoute(rel)
		current.RawPath = ""
		for _, href := range page.Refs {
			if reason := renderedLinkProblem(&current, base, href, aliases, &index); reason != "" {
				report.Errors = append(report.Errors, rel+": "+reason+" "+href)
			}
		}
	}
	for source, target := range aliases {
		if reason := renderedLinkProblem(base, base, source, aliases, &index); reason != "" {
			report.Errors = append(report.Errors, "redirect "+source+" -> "+target+": "+reason)
		}
	}
	checkRenderedLanguages(cfg, pages, &report)
	checkRenderedAssets(output, cfg, &report)
	sort.Strings(report.Errors)
	return report, nil
}

func indexRenderedOutput(output string, cloudflare bool, report *RenderedReport) (map[string]renderedPage, siteIndex, error) {
	pages := map[string]renderedPage{}
	index := newSiteIndex(output)
	err := filepath.WalkDir(output, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(output, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		report.Files++
		if !info.Mode().IsRegular() {
			report.Errors = append(report.Errors, rel+": output must be a regular file")
			return nil
		}
		if cloudflare && info.Size() > 25*1024*1024 {
			report.Errors = append(report.Errors, rel+": file exceeds the Pages 25 MiB limit")
		}
		index.addFile(rel)
		if filepath.Ext(path) != ".html" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		page := parseRenderedPage(data)
		pages[rel] = page
		index.ids[rel] = page.IDs
		report.HTMLPages++
		report.Languages[page.language]++
		return nil
	})
	if err != nil {
		return nil, index, err
	}
	if report.Files == 0 {
		report.Errors = append(report.Errors, "site contains no files")
	}
	if cloudflare && report.Files > 20000 {
		report.Errors = append(report.Errors, "Pages upload must contain 1–20,000 files")
	}
	return pages, index, nil
}

func parseRenderedPage(data []byte) renderedPage {
	page := renderedPage{HTMLIndex: ParseHTML(data)}
	literal := map[string]int{}
	tokenizer := html.NewTokenizer(bytes.NewReader(data))
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			return page
		}
		token := tokenizer.Token()
		if kind == html.TextToken {
			if len(literal) == 0 && strings.Contains(token.Data, "```") {
				page.fence = true
			}
			continue
		}
		if kind == html.EndTagToken {
			if literal[token.Data] <= 1 {
				delete(literal, token.Data)
			} else {
				literal[token.Data]--
			}
			continue
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		if kind == html.StartTagToken {
			switch token.Data {
			case "pre", "code", "script", "style", "svg":
				literal[token.Data]++
			}
		}
		page.readMetadata(token)
	}
}

func (page *renderedPage) readMetadata(token html.Token) {
	attrs := map[string]string{}
	for _, attr := range token.Attr {
		attrs[attr.Key] = attr.Val
	}
	switch token.Data {
	case "html":
		page.language = attrs["lang"]
	case "link":
		if slices.Contains(strings.Fields(attrs["rel"]), "canonical") {
			page.canonical = attrs["href"]
		}
	case "meta":
		if attrs["name"] == "mpress:source" {
			page.source = attrs["content"]
		}
	case "code":
		if slices.Contains(strings.Fields(attrs["class"]), "language-d2") {
			page.d2 = true
		}
	}
}

func renderedRoute(relative string) string { return "/" + strings.TrimSuffix(relative, "index.html") }

func checkRenderedPage(project string, cfg config.Config, relative string, page renderedPage, report *RenderedReport) {
	add := func(message string) { report.Errors = append(report.Errors, relative+": "+message) }
	if page.language == "" {
		add("missing language")
	}
	if page.d2 {
		add("D2 diagram was displayed as source code")
	}
	if page.fence {
		add("Markdown code fence was displayed as prose")
	}
	if relative == "404.html" {
		return
	}
	if cfg.Site.BaseURL != "" && page.canonical != strings.TrimRight(cfg.Site.BaseURL, "/")+renderedRoute(relative) {
		add("incorrect canonical " + page.canonical)
	}
	if page.source != "" && (!filepath.IsLocal(filepath.FromSlash(page.source)) || !fileWithin(cfg.ContentPath(project), filepath.Join(project, filepath.FromSlash(page.source)))) {
		add("invalid contribution source " + page.source)
	}
}

// fileWithin resolves symlinks as well as lexical parent traversal.
func fileWithin(root, path string) bool {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return false
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || !filepath.IsLocal(rel) {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func readRedirects(output string, report *RenderedReport) map[string]string {
	aliases := map[string]string{}
	data, err := os.ReadFile(filepath.Join(output, "_redirects"))
	if os.IsNotExist(err) {
		return aliases
	}
	if err != nil {
		report.Errors = append(report.Errors, "read redirects: "+err.Error())
		return aliases
	}
	for line, raw := range strings.Split(string(data), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		fields := strings.Fields(raw)
		if len(fields) != 3 || (fields[2] != "301" && fields[2] != "302") || !strings.HasPrefix(fields[0], "/") || strings.ContainsAny(fields[0], "*:") {
			report.Errors = append(report.Errors, fmt.Sprintf("_redirects:%d: expected an exact path, target and 301 or 302 status", line+1))
			continue
		}
		if _, exists := aliases[fields[0]]; exists {
			report.Errors = append(report.Errors, "duplicate redirect: "+fields[0])
		}
		aliases[fields[0]] = fields[1]
	}
	return aliases
}

func renderedLinkProblem(current, base *url.URL, href string, aliases map[string]string, index *siteIndex) string {
	target, err := url.Parse(href)
	if err != nil {
		return "invalid link"
	}
	target = current.ResolveReference(target)
	target, problem := resolveRenderedRedirect(target, base, aliases)
	if problem != "" {
		return problem
	}
	if target.Scheme != "http" && target.Scheme != "https" || !strings.EqualFold(target.Host, base.Host) {
		return ""
	}

	prefix := strings.TrimSuffix(base.Path, "/")
	if prefix != "" && target.Path != prefix && !strings.HasPrefix(target.Path, prefix+"/") {
		return "link outside site base path"
	}
	path := strings.TrimPrefix(strings.TrimPrefix(target.Path, prefix), "/")
	if path != "" && !filepath.IsLocal(filepath.FromSlash(path)) {
		return "link escapes site"
	}
	key := index.normaliseTarget(cleanKey(path))
	if !index.hasTarget(key) {
		return "missing link"
	}
	if target.Fragment != "" {
		if _, isHTML := index.ids[key]; isHTML && !index.hasFragment(key, target.Fragment) {
			return "missing anchor"
		}
	}
	return ""
}

func resolveRenderedRedirect(target, base *url.URL, aliases map[string]string) (*url.URL, string) {
	seen := map[string]bool{}
	for {
		if target.Scheme != "http" && target.Scheme != "https" || !strings.EqualFold(target.Host, base.Host) {
			return target, ""
		}
		if seen[target.Path] {
			return nil, "redirect cycle"
		}
		seen[target.Path] = true
		alias, ok := aliases[target.Path]
		if !ok {
			break
		}
		next, err := url.Parse(alias)
		if err != nil {
			return nil, "invalid redirect"
		}
		fragment := target.Fragment
		target = target.ResolveReference(next)
		if next.Fragment == "" {
			target.Fragment = fragment
		}
	}
	return target, ""
}

func languagePrefix(cfg config.Config, language string) string {
	if language == cfg.Site.DefaultLanguage && cfg.Site.DefaultAtRoot {
		return ""
	}
	return language + "/"
}

func checkRenderedLanguages(cfg config.Config, pages map[string]renderedPage, report *RenderedReport) {
	sourcePrefix := languagePrefix(cfg, cfg.Site.DefaultLanguage)
	root := sourcePrefix + "index.html"
	if page, ok := pages[root]; !ok || page.language != cfg.Site.DefaultLanguage {
		report.Errors = append(report.Errors, root+": root page must use the default language "+cfg.Site.DefaultLanguage)
	}
	for relative, page := range pages {
		if relative == "404.html" {
			continue
		}
		expected := renderedLanguage(cfg, relative)
		if page.language != expected {
			report.Errors = append(report.Errors, relative+": incorrect route language (possible translation fallback)")
			continue
		}
		if expected != cfg.Site.DefaultLanguage || !strings.HasPrefix(relative, sourcePrefix) {
			continue
		}
		route := strings.TrimPrefix(relative, sourcePrefix)
		checkTranslatedRoute(cfg, route, pages, report)
	}
}

func renderedLanguage(cfg config.Config, relative string) string {
	for _, lang := range cfg.Site.Languages {
		if strings.HasPrefix(relative, lang+"/") {
			return lang
		}
	}
	return cfg.Site.DefaultLanguage
}

func checkTranslatedRoute(cfg config.Config, route string, pages map[string]renderedPage, report *RenderedReport) {
	for _, lang := range cfg.Site.Languages {
		if lang == cfg.Site.DefaultLanguage {
			continue
		}
		target := languagePrefix(cfg, lang) + route
		if translated, ok := pages[target]; !ok || translated.language != lang {
			report.Errors = append(report.Errors, target+": missing translated page")
		}
	}
}

func checkRenderedAssets(output string, cfg config.Config, report *RenderedReport) {
	assets := []string{"sitemap.xml", "robots.txt", "llms.txt"}
	if cfg.Search.Enabled {
		for _, lang := range cfg.Site.Languages {
			assets = append(assets, languagePrefix(cfg, lang)+"search-index.json")
		}
	}
	for _, name := range assets {
		if !fileWithin(output, filepath.Join(output, filepath.FromSlash(name))) {
			report.Errors = append(report.Errors, "missing generated asset: "+name)
		}
	}
	if !cfg.Knowledge.Enabled {
		return
	}
	directory := filepath.Join(output, "knowledge")
	data, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	var manifest struct {
		Artifacts map[string]string `json:"artifacts"`
	}
	if err == nil {
		err = json.Unmarshal(data, &manifest)
	}
	if err != nil {
		report.Errors = append(report.Errors, "invalid knowledge manifest: "+err.Error())
		return
	}
	for _, kind := range []string{"pages", "chunks", "index"} {
		name := manifest.Artifacts[kind]
		if !filepath.IsLocal(name) || !fileWithin(directory, filepath.Join(directory, name)) {
			report.Errors = append(report.Errors, "missing or invalid knowledge artifact: "+kind+" "+name)
		}
	}
}
