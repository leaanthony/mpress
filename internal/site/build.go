package site

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/content"
	"github.com/leaanthony/mpress/internal/knowledge"
	"github.com/leaanthony/mpress/internal/navigation"
	"github.com/leaanthony/mpress/internal/quickedit"
	docversion "github.com/leaanthony/mpress/internal/version"
)

type BuildOptions struct {
	Strict        bool
	IncludeDrafts bool
	// Development includes local-only authoring aids such as inline quick edit.
	// Production output deliberately omits their markup and page metadata.
	Development bool
	// MinifyAssets emits compact CSS and JavaScript for production builds.
	// Development builds leave assets readable unless this is explicitly set.
	MinifyAssets bool
	// PurgeUnusedCSS removes clearly unreferenced class and ID rules after all
	// pages have been rendered. It is opt-in because custom HTML can add styles
	// that are not visible in generated Markdown.
	PurgeUnusedCSS  bool
	OutputDir       string
	SourceOverrides map[string]string
	// LinkCollector receives generated HTML while pages are rendered. It is
	// optional so ordinary builds keep no validation state.
	LinkCollector *check.Collector
}
type BuildResult struct {
	Pages       int                  `json:"pages"`
	Files       int                  `json:"files"`
	Duration    time.Duration        `json:"duration"`
	Timings     []BuildTiming        `json:"timings,omitempty"`
	Diagnostics []content.Diagnostic `json:"diagnostics"`
}

type BuildTiming struct {
	Name       string  `json:"name"`
	Label      string  `json:"label"`
	OffsetMS   float64 `json:"offsetMs"`
	DurationMS float64 `json:"durationMs"`
	Status     string  `json:"status"`
}

func Build(projectDir string, opts BuildOptions) (result BuildResult, buildErr error) {
	started := time.Now()
	var timingName, timingLabel string
	var timingStarted time.Time
	finishTiming := func(status string) {
		if timingName == "" {
			return
		}
		result.Timings = append(result.Timings, BuildTiming{
			Name:       timingName,
			Label:      timingLabel,
			OffsetMS:   durationMilliseconds(timingStarted.Sub(started)),
			DurationMS: durationMilliseconds(time.Since(timingStarted)),
			Status:     status,
		})
		timingName = ""
	}
	startTiming := func(name, label string) {
		finishTiming("passed")
		timingName = name
		timingLabel = label
		timingStarted = time.Now()
	}
	defer func() {
		status := "passed"
		if buildErr != nil {
			status = "failed"
		}
		finishTiming(status)
		result.Duration = time.Since(started)
	}()
	startTiming("discover", "Discover project")
	cfg, err := config.Load(projectDir)
	if err != nil {
		return result, err
	}
	contentDir := cfg.ContentPath(projectDir)
	info, err := os.Stat(contentDir)
	if err != nil || !info.IsDir() {
		return result, fmt.Errorf("content directory not found: %s", contentDir)
	}
	discovered, err := content.Discover(contentDir, cfg.Site.Languages, cfg.Site.DefaultLanguage)
	if err != nil {
		return result, err
	}
	startTiming("parse", "Parse Markdown")
	pagesByLang := map[string][]*content.Page{}
	routesByLang := map[string]map[string]*content.Page{}
	quickEdits := make(map[*content.Page]*quickedit.Document)
	for _, lang := range cfg.Site.Languages {
		routesByLang[lang] = map[string]*content.Page{}
	}
	for _, parsed := range parseDiscoveredPages(projectDir, contentDir, cfg, discovered, opts) {
		lang, rel := parsed.job.lang, parsed.job.rel
		page, diags, parseErr := parsed.page, parsed.diags, parsed.err
		if lang != cfg.Site.DefaultLanguage {
			for i := range diags {
				diags[i].File = filepath.ToSlash(filepath.Join(lang, diags[i].File))
			}
		}
		result.Diagnostics = append(result.Diagnostics, diags...)
		if parseErr != nil {
			diagnosticFile := rel
			if lang != cfg.Site.DefaultLanguage {
				diagnosticFile = filepath.ToSlash(filepath.Join(lang, rel))
			}
			result.Diagnostics = append(result.Diagnostics, content.Diagnostic{Severity: "error", Code: "parse", File: diagnosticFile, Message: parseErr.Error()})
			continue
		}
		if page.Draft && !opts.IncludeDrafts {
			continue
		}
		if !opts.IncludeDrafts && strings.HasPrefix(filepath.ToSlash(page.SourcePath), "blog/") {
			publicationDate := content.ParseDate(page.Meta.Date)
			today, _ := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
			if !publicationDate.IsZero() && publicationDate.After(today) {
				continue
			}
		}
		if prev := routesByLang[lang][page.URLPath]; prev != nil {
			diagnosticFile := rel
			if lang != cfg.Site.DefaultLanguage {
				diagnosticFile = filepath.ToSlash(filepath.Join(lang, rel))
			}
			result.Diagnostics = append(result.Diagnostics, content.Diagnostic{Severity: "error", Code: "duplicate-route", File: diagnosticFile, Message: fmt.Sprintf("route /%s already used by %s", page.URLPath, prev.SourcePath)})
			continue
		}
		routesByLang[lang][page.URLPath] = page
		pagesByLang[lang] = append(pagesByLang[lang], page)
		if parsed.quickEdit != nil {
			quickEdits[page] = parsed.quickEdit
		}
	}
	if len(pagesByLang[cfg.Site.DefaultLanguage]) == 0 {
		return result, fmt.Errorf("no pages found for default language %s", cfg.Site.DefaultLanguage)
	}
	startTiming("prepare", "Prepare output")
	outputDir := cfg.OutputPath(projectDir)
	if strings.TrimSpace(opts.OutputDir) != "" {
		outputDir = opts.OutputDir
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(projectDir, outputDir)
		}
	}
	if err := safeOutput(projectDir, outputDir); err != nil {
		return result, err
	}
	if err := os.RemoveAll(outputDir); err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "assets"), 0o755); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.Reset(outputDir)
	}
	// usedCSSTokens accumulates the class and id tokens referenced by every
	// generated and copied HTML file. Collecting the set as pages are produced
	// avoids buffering the whole site in memory only to rescan it during the CSS
	// purge phase.
	var usedCSSTokens map[string]struct{}
	if opts.PurgeUnusedCSS {
		usedCSSTokens = make(map[string]struct{})
	}
	themeCSS := strings.Replace(defaultThemeCSS, "#5375f6", cfg.Theme.AccentColor, 1)
	themeCSS = strings.Replace(themeCSS, "#7593ff", cfg.Theme.HoverColorLight, 1)
	themeCSS = strings.Replace(themeCSS, "#95aaff", cfg.Theme.HoverColorDark, 1)
	themeCSS += documentationLayoutCSS(cfg.Theme.Layout)
	if cfg.Build.CustomCSS != "" {
		customCSS, err := os.ReadFile(filepath.Join(projectDir, cfg.Build.CustomCSS))
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, content.Diagnostic{Severity: "error", Code: "custom-css", Message: err.Error()})
		} else {
			// Keep the theme and project overrides in one critical stylesheet. This
			// preserves cascade order while removing a second render-blocking request.
			themeCSS += "\n/* Project custom CSS */\n" + string(customCSS)
		}
	}
	if cfg.Accessibility.Enabled {
		themeCSS += accessibilityCSS
	}
	cssAsset := themeCSS
	jsAsset := defaultThemeJS
	if cfg.Accessibility.Enabled {
		jsAsset += accessibilityJS
	}
	cssVersion := fmt.Sprintf("%x", sha256.Sum256([]byte(cssAsset)))[:12]
	jsVersion := fmt.Sprintf("%x", sha256.Sum256([]byte(jsAsset)))[:12]
	if err := os.WriteFile(filepath.Join(outputDir, "assets", "mpress.css"), []byte(cssAsset), 0o644); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.AddFile("assets/mpress.css")
	}
	if err := os.WriteFile(filepath.Join(outputDir, "assets", "mpress.js"), []byte(jsAsset), 0o644); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.AddFile("assets/mpress.js")
	}
	if err := copyTree(cfg.StaticPath(projectDir), outputDir, func(relative, target string) error {
		if opts.LinkCollector == nil && !opts.PurgeUnusedCSS {
			return nil
		}
		if filepath.Ext(target) != ".html" {
			if opts.LinkCollector != nil {
				opts.LinkCollector.AddFile(relative)
			}
			return nil
		}
		data, err := os.ReadFile(target)
		if err != nil {
			return err
		}
		if opts.PurgeUnusedCSS {
			addHTMLTokens(usedCSSTokens, bytesToString(data))
		}
		if opts.LinkCollector != nil {
			opts.LinkCollector.AddHTML(relative, data)
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return result, err
	}
	// Emit host-neutral cache rules for the static adapters supported by
	// M-Press. Version snapshots can be cached for a year because their paths
	// are immutable; the generated shell and HTML remain revalidatable.
	headersPath := filepath.Join(outputDir, "_headers")
	existingHeaders, _ := os.ReadFile(headersPath)
	headers := string(existingHeaders)
	if !strings.Contains(headers, "# M-Press cache defaults") {
		if headers != "" && !strings.HasSuffix(headers, "\n") {
			headers += "\n"
		}
		headers += staticCacheHeaders
	}
	if err := os.WriteFile(headersPath, []byte(headers), 0o644); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.AddFile("_headers")
	}
	if cfg.Contribution.Enabled {
		for name, contents := range map[string]string{
			"mpress-contribute.sh":  renderContributionInstallShell(cfg.Contribution.Repository, cfg.Contribution.Branch),
			"mpress-contribute.ps1": renderContributionInstallPowerShell(cfg.Contribution.Repository, cfg.Contribution.Branch),
		} {
			if err := os.WriteFile(filepath.Join(outputDir, name), []byte(contents), 0o644); err != nil {
				return result, err
			}
			if opts.LinkCollector != nil {
				opts.LinkCollector.AddFile(name)
			}
		}
	}
	defaultRoutes := routesByLang[cfg.Site.DefaultLanguage]
	var versionLabels []string
	if cfg.Version.Enabled {
		versionLabels, err = docversion.List(projectDir)
		if err != nil {
			return result, err
		}
	}
	renderLabel := "Render pages and search"
	if opts.LinkCollector != nil {
		renderLabel = "Render pages, search, and link index"
	}
	startTiming("render", renderLabel)
	for _, lang := range cfg.Site.Languages {
		pages := pagesByLang[lang]
		blogs, blogArticles := blogArchives(pages, cfg.Blog)
		sort.SliceStable(pages, func(i, j int) bool {
			if pages[i].Order != pages[j].Order {
				return pages[i].Order < pages[j].Order
			}
			return pages[i].SourcePath < pages[j].SourcePath
		})
		navPath := filepath.Join(contentDir, cfg.Build.NavFile)
		if lang != cfg.Site.DefaultLanguage {
			candidate := filepath.Join(contentDir, lang, cfg.Build.NavFile)
			if _, e := os.Stat(candidate); e == nil {
				navPath = candidate
			}
		}
		nav, navErr := navigation.Load(navPath, pages)
		if navErr != nil {
			return result, navErr
		}
		validateNavigation(nav, defaultRoutes, lang, routesByLang[lang], &result.Diagnostics)
		flat := flattenNav(nav)
		compiledNav := compileNav(nav, lang, cfg.Site.DefaultLanguage, cfg.Site.DefaultAtRoot, routesByLang[lang])
		rendered, tokens, renderErr := renderPages(outputDir, len(pages), opts.PurgeUnusedCSS, opts.LinkCollector != nil, func(i int) (string, templateData, error) {
			page := pages[i]
			pageOut := page.OutputPath
			if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
				pageOut = filepath.ToSlash(filepath.Join(lang, pageOut))
			}
			root := relativeRoot(pageOut)
			prev, next := neighbors(flat, page.URLPath)
			links := languageLinks(cfg, page.URLPath, routesByLang)
			versions := versionLinks(cfg, page.URLPath, versionLabels)
			searchURL := root + "search-index.json"
			if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
				searchURL = root + lang + "/search-index.json"
			}
			quickEdit := quickEdits[page]
			// Quick edit currently maps byte ranges in Markdown source. Native MPD
			// pages deliberately do not create that Markdown-specific metadata.
			if opts.Development && cfg.Contribution.Enabled && quickEdit == nil && !strings.EqualFold(filepath.Ext(page.SourcePath), ".mpd") {
				return "", templateData{}, fmt.Errorf("quick-edit metadata missing for %s", page.SourcePath)
			}
			data := templateData{Config: cfg, Page: page, Nav: nav, CompiledNav: compiledNav, Root: root, LangLinks: links, VersionLinks: versions, Prev: prev, Next: next, SearchURL: searchURL, CSSVersion: cssVersion, JSVersion: jsVersion, Blog: blogs[page], BlogArticle: blogArticles[page], QuickEdit: quickEdit,
				CanonicalURL: sitePageURL(cfg, lang, page.URLPath), Alternates: alternateLinks(cfg, page.URLPath, routesByLang), CurrentRoutes: routesByLang[lang]}
			return pageOut, data, nil
		})
		if renderErr != nil {
			return result, renderErr
		}
		if opts.PurgeUnusedCSS {
			for token := range tokens {
				usedCSSTokens[token] = struct{}{}
			}
		}
		// Link parsing runs alongside each render worker. Commit the completed
		// indexes in input order so a collector still has deterministic output.
		if opts.LinkCollector != nil {
			for _, page := range rendered {
				opts.LinkCollector.AddParsedHTML(page.pageOut, page.linkIndex)
			}
		}
		result.Pages += len(rendered)
		if cfg.Search.Enabled {
			if err := writeSearchIndex(outputDir, lang, cfg, pages); err != nil {
				return result, err
			}
			if opts.LinkCollector != nil {
				opts.LinkCollector.AddFile(searchIndexOutputPath(lang, cfg))
			}
		}
	}
	notFoundPath := filepath.Join(outputDir, "404.html")
	if _, statErr := os.Stat(notFoundPath); os.IsNotExist(statErr) {
		defaultPages := pagesByLang[cfg.Site.DefaultLanguage]
		nav, navErr := navigation.Load(filepath.Join(contentDir, cfg.Build.NavFile), defaultPages)
		if navErr != nil {
			return result, navErr
		}
		notFound := &content.Page{
			Language: cfg.Site.DefaultLanguage, Title: "Page not found",
			Description: "The requested documentation page does not exist or may have moved.",
		}
		notFoundConfig := cfg
		notFoundConfig.Contribution.Enabled = false
		rendered, renderErr := renderPage(templateData{
			Config: notFoundConfig, Page: notFound, Nav: nav, Root: "./", NotFound: true,
			SearchURL: "./" + searchIndexOutputPath(cfg.Site.DefaultLanguage, cfg), CSSVersion: cssVersion, JSVersion: jsVersion, CurrentRoutes: routesByLang[cfg.Site.DefaultLanguage],
		})
		if renderErr != nil {
			return result, renderErr
		}
		if err := os.WriteFile(notFoundPath, rendered, 0o644); err != nil {
			return result, err
		}
		if opts.PurgeUnusedCSS {
			addHTMLTokens(usedCSSTokens, bytesToString(rendered))
		}
		if opts.LinkCollector != nil {
			opts.LinkCollector.AddHTML("404.html", rendered)
		}
	} else if statErr != nil {
		return result, statErr
	}
	if opts.MinifyAssets || opts.PurgeUnusedCSS {
		startTiming("optimize", "Optimize CSS and JavaScript")
		cssAsset = themeCSS
		jsAsset = defaultThemeJS
		if cfg.Accessibility.Enabled {
			jsAsset += accessibilityJS
		}
		if opts.PurgeUnusedCSS {
			cssAsset = purgeUnusedCSSWithTokens(themeCSS, usedCSSTokens, jsAsset)
		}
		if opts.MinifyAssets {
			cssAsset, err = minifyStylesheet(cssAsset)
			if err != nil {
				return result, err
			}
			jsAsset, err = minifyScript(jsAsset)
			if err != nil {
				return result, err
			}
		}
		if err := os.WriteFile(filepath.Join(outputDir, "assets", "mpress.css"), []byte(cssAsset), 0o644); err != nil {
			return result, err
		}
		if err := os.WriteFile(filepath.Join(outputDir, "assets", "mpress.js"), []byte(jsAsset), 0o644); err != nil {
			return result, err
		}
	}
	finalCSSVersion := fmt.Sprintf("%x", sha256.Sum256([]byte(cssAsset)))[:12]
	finalJSVersion := fmt.Sprintf("%x", sha256.Sum256([]byte(jsAsset)))[:12]
	if err := rewriteAssetVersions(outputDir, cssVersion, finalCSSVersion, jsVersion, finalJSVersion); err != nil {
		return result, err
	}
	startTiming("finalize", "Finalize static site")
	if err := writeSitemap(outputDir, cfg, pagesByLang); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil && cfg.Site.BaseURL != "" {
		opts.LinkCollector.AddFile("sitemap.xml")
	}
	if err := writeLLMsText(outputDir, cfg, pagesByLang); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.AddFile("llms.txt")
	}
	if err := writeRobots(outputDir, cfg); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.AddFile("robots.txt")
	}
	if err := writeManifest(outputDir, cfg, pagesByLang); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil {
		opts.LinkCollector.AddFile("mpress-manifest.json")
	}
	if cfg.Knowledge.Enabled {
		if err := knowledge.Generate(outputDir, cfg, pagesByLang); err != nil {
			return result, err
		}
		if opts.LinkCollector != nil {
			for _, name := range []string{knowledge.ManifestFile, knowledge.PagesFile, knowledge.ChunksFile, knowledge.IndexFile} {
				opts.LinkCollector.AddFile(filepath.ToSlash(filepath.Join(knowledge.Directory, name)))
			}
		}
	}
	if _, err := docversion.Mount(projectDir, outputDir); err != nil {
		return result, err
	}
	if opts.LinkCollector != nil && cfg.Version.Enabled {
		opts.LinkCollector.AddFile("versions/versions.json")
		if err := opts.LinkCollector.IndexDirectory("versions"); err != nil {
			return result, err
		}
	}
	if opts.LinkCollector != nil {
		result.Files = opts.LinkCollector.FileCount()
	} else {
		result.Files = countFiles(outputDir)
	}
	if opts.Strict && hasErrors(result.Diagnostics) {
		return result, fmt.Errorf("strict build failed with %d error(s)", countSeverity(result.Diagnostics, "error"))
	}
	return result, nil
}

func rewriteAssetVersions(root, oldCSS, newCSS, oldJS, newJS string) error {
	if oldCSS == newCSS && oldJS == newJS {
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		original := string(data)
		updated := original
		if oldCSS != newCSS {
			updated = strings.ReplaceAll(updated, "mpress.css?v="+oldCSS, "mpress.css?v="+newCSS)
		}
		if oldJS != newJS {
			updated = strings.ReplaceAll(updated, "mpress.js?v="+oldJS, "mpress.js?v="+newJS)
		}
		if updated == original {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(path, []byte(updated), info.Mode().Perm())
	})
}

const staticCacheHeaders = `# M-Press cache defaults
/assets/*
  Cache-Control: public, max-age=3600, stale-while-revalidate=86400

/versions/*
  Cache-Control: public, max-age=31536000, immutable

/*.html
  Cache-Control: public, max-age=300, must-revalidate
`

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func validateNavigation(items []navigation.Item, defaults map[string]*content.Page, lang string, routes map[string]*content.Page, diags *[]content.Diagnostic) {
	for _, link := range navigation.Links(items) {
		if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "#") {
			continue
		}
		route := strings.Trim(strings.TrimSuffix(link, "/"), "/")
		if route == "" {
			route = ""
		}
		if routes[route] == nil && defaults[route] == nil {
			*diags = append(*diags, content.Diagnostic{Severity: "error", Code: "unresolved-navigation", File: "_nav.yaml", Message: fmt.Sprintf("navigation link %q has no page for %s or default language", link, lang)})
		}
	}
}
func languageLinks(cfg config.Config, route string, routes map[string]map[string]*content.Page) []languageLink {
	var out []languageLink
	fallbackLabel := cfg.Site.DefaultLanguage
	if label := cfg.Site.LanguageLabels[cfg.Site.DefaultLanguage]; label != "" {
		fallbackLabel = label
	}
	for _, lang := range cfg.Site.Languages {
		available := routes[lang][route] != nil
		if !available && cfg.Site.MissingTranslation == "omit" {
			continue
		}
		targetLang := lang
		if !available {
			targetLang = cfg.Site.DefaultLanguage
		}
		prefix := ""
		if targetLang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
			prefix = "/" + targetLang
		}
		path := prefix + "/" + strings.Trim(route, "/")
		if route != "" {
			path += "/"
		}
		label := lang
		if cfg.Site.LanguageLabels[lang] != "" {
			label = cfg.Site.LanguageLabels[lang]
		}
		out = append(out, languageLink{Code: lang, Label: label, URL: path, Available: available, FallbackLabel: fallbackLabel})
	}
	return out
}

func sitePageURL(cfg config.Config, lang, route string) string {
	if strings.TrimSpace(cfg.Site.BaseURL) == "" {
		return ""
	}
	prefix := ""
	if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
		prefix = "/" + lang
	}
	path := prefix + "/" + strings.Trim(route, "/")
	if route != "" {
		path += "/"
	}
	return strings.TrimRight(cfg.Site.BaseURL, "/") + path
}

func alternateLinks(cfg config.Config, route string, routes map[string]map[string]*content.Page) []alternateLink {
	if strings.TrimSpace(cfg.Site.BaseURL) == "" {
		return nil
	}
	var out []alternateLink
	for _, lang := range cfg.Site.Languages {
		if routes[lang][route] == nil {
			continue
		}
		out = append(out, alternateLink{Language: lang, URL: sitePageURL(cfg, lang, route)})
	}
	if routes[cfg.Site.DefaultLanguage][route] != nil {
		out = append(out, alternateLink{Language: "x-default", URL: sitePageURL(cfg, cfg.Site.DefaultLanguage, route)})
	}
	return out
}
func versionLinks(cfg config.Config, route string, labels []string) []versionLink {
	if !cfg.Version.Enabled || len(labels) == 0 {
		return nil
	}
	current := cfg.Version.Current
	if current == "" {
		current = "Current"
	}
	path := "/" + strings.Trim(route, "/")
	if route != "" {
		path += "/"
	}
	out := []versionLink{{Label: current, URL: path, Current: true}}
	for _, label := range labels {
		out = append(out, versionLink{Label: label, URL: "/versions/" + label + path})
	}
	return out
}
func flattenNav(items []navigation.Item) []navigation.Item {
	var out []navigation.Item
	var walk func([]navigation.Item)
	walk = func(xs []navigation.Item) {
		for _, x := range xs {
			if x.Link != "" {
				out = append(out, x)
			}
			walk(x.Items)
		}
	}
	walk(items)
	return out
}
func neighbors(items []navigation.Item, route string) (*navigation.Item, *navigation.Item) {
	for i := range items {
		if strings.Trim(items[i].Link, "/") == route {
			var p, n *navigation.Item
			if i > 0 {
				p = &items[i-1]
			}
			if i+1 < len(items) {
				n = &items[i+1]
			}
			return p, n
		}
	}
	return nil, nil
}
func relativeRoot(output string) string {
	dir := filepath.ToSlash(filepath.Dir(output))
	if dir == "." {
		return "./"
	}
	return strings.Repeat("../", len(strings.Split(dir, "/")))
}

type searchItem struct {
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Text        string          `json:"text"`
	URL         string          `json:"url"`
	Headings    []searchHeading `json:"headings,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

type searchHeading struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Level int    `json:"level"`
}

func writeSearchIndex(output, lang string, cfg config.Config, pages []*content.Page) error {
	items := make([]searchItem, 0, len(pages))
	prefix := ""
	if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
		prefix = "/" + lang
	}
	for _, p := range pages {
		path := prefix + "/" + strings.Trim(p.URLPath, "/")
		if p.URLPath != "" {
			path += "/"
		}
		headings := make([]searchHeading, 0, len(p.Headings)+1)
		hasTitleHeading := false
		for _, heading := range p.Headings {
			if heading.Level == 1 && strings.EqualFold(strings.TrimSpace(heading.Text), strings.TrimSpace(p.Title)) {
				hasTitleHeading = true
				break
			}
		}
		// Documentation pages render their title in the page chrome and remove a
		// matching body H1 to avoid displaying it twice. Keep that title in the
		// search contract so keyboard search can still offer the page itself as
		// the first heading result. #content is a stable target on every page.
		if strings.TrimSpace(p.Title) != "" && !hasTitleHeading {
			headings = append(headings, searchHeading{ID: "content", Text: p.Title, Level: 1})
		}
		for _, heading := range p.Headings {
			headings = append(headings, searchHeading{ID: heading.ID, Text: heading.Text, Level: heading.Level})
		}
		items = append(items, searchItem{
			Title:       p.Title,
			Description: p.Description,
			Text:        p.PlainText,
			URL:         path,
			Headings:    headings,
			Tags:        p.Meta.Tags,
		})
	}
	data, _ := json.Marshal(items)
	dest := filepath.Join(output, filepath.FromSlash(searchIndexOutputPath(lang, cfg)))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0o644)
}

func searchIndexOutputPath(lang string, cfg config.Config) string {
	if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
		return filepath.ToSlash(filepath.Join(lang, "search-index.json"))
	}
	return "search-index.json"
}
func writeSitemap(output string, cfg config.Config, pages map[string][]*content.Page) error {
	if cfg.Site.BaseURL == "" {
		return nil
	}
	base := strings.TrimRight(cfg.Site.BaseURL, "/")
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, lang := range cfg.Site.Languages {
		for _, p := range pages[lang] {
			prefix := ""
			if lang != cfg.Site.DefaultLanguage || !cfg.Site.DefaultAtRoot {
				prefix = "/" + lang
			}
			segments := strings.Split(strings.Trim(p.URLPath, "/"), "/")
			for i := range segments {
				segments[i] = url.PathEscape(segments[i])
			}
			path := strings.Join(segments, "/")
			if path != "" {
				path += "/"
			}
			fmt.Fprintf(&b, "<url><loc>%s%s/%s</loc></url>\n", base, prefix, path)
		}
	}
	b.WriteString("</urlset>\n")
	return os.WriteFile(filepath.Join(output, "sitemap.xml"), []byte(b.String()), 0o644)
}

func writeRobots(output string, cfg config.Config) error {
	path := filepath.Join(output, "robots.txt")
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	var body strings.Builder
	body.WriteString("User-agent: *\nAllow: /\n")
	if cfg.Site.BaseURL != "" {
		body.WriteString("Sitemap: ")
		body.WriteString(strings.TrimRight(cfg.Site.BaseURL, "/"))
		body.WriteString("/sitemap.xml\n")
	}
	return os.WriteFile(path, []byte(body.String()), 0o644)
}

func writeLLMsText(output string, cfg config.Config, pages map[string][]*content.Page) error {
	var body strings.Builder
	body.WriteString("# " + cleanLLMsText(cfg.Site.Title) + "\n\n")
	if description := cleanLLMsText(cfg.Site.Description); description != "" {
		body.WriteString("> " + description + "\n\n")
	}
	body.WriteString("## Documentation\n\n")
	base := strings.TrimRight(cfg.Site.BaseURL, "/")
	defaultLanguage := cfg.Site.DefaultLanguage
	for _, page := range pages[defaultLanguage] {
		path := "/" + strings.Trim(page.URLPath, "/")
		if path != "/" {
			path += "/"
		}
		if !cfg.Site.DefaultAtRoot {
			path = "/" + defaultLanguage + path
		}
		link := base + path
		if link == "" {
			link = "/"
		}
		body.WriteString("- [" + cleanLLMsText(page.Title) + "](" + link + ")")
		if description := cleanLLMsText(page.Description); description != "" {
			body.WriteString(": " + description)
		}
		body.WriteByte('\n')
	}
	return os.WriteFile(filepath.Join(output, "llms.txt"), []byte(body.String()), 0o644)
}

func cleanLLMsText(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = strings.ReplaceAll(value, "[", "\\[")
	return strings.ReplaceAll(value, "]", "\\]")
}

func writeManifest(output string, cfg config.Config, pages map[string][]*content.Page) error {
	type entry struct{ Language, Source, URL string }
	var es []entry
	for _, lang := range cfg.Site.Languages {
		for _, p := range pages[lang] {
			es = append(es, entry{lang, p.SourcePath, p.URLPath})
		}
	}
	data, _ := json.MarshalIndent(map[string]any{"schemaVersion": 1, "pages": es}, "", "  ")
	return os.WriteFile(filepath.Join(output, "mpress-manifest.json"), data, 0o644)
}
func safeOutput(root, out string) error {
	r, _ := filepath.Abs(root)
	o, _ := filepath.Abs(out)
	rel, err := filepath.Rel(r, o)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("unsafe output directory %s", out)
	}
	return nil
}
func copyTree(src, dst string, onFile func(relative, target string) error) error {
	return filepath.WalkDir(src, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := copyFile(path, target); err != nil {
			return err
		}
		if onFile != nil {
			return onFile(filepath.ToSlash(rel), target)
		}
		return nil
	})
}
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err = os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, cp := io.Copy(out, in)
	closeErr := out.Close()
	if cp != nil {
		return cp
	}
	return closeErr
}
func countFiles(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(_ string, e fs.DirEntry, err error) error {
		if err == nil && !e.IsDir() {
			n++
		}
		return nil
	})
	return n
}
func hasErrors(ds []content.Diagnostic) bool { return countSeverity(ds, "error") > 0 }
func countSeverity(ds []content.Diagnostic, s string) int {
	n := 0
	for _, d := range ds {
		if d.Severity == s {
			n++
		}
	}
	return n
}
