package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const Filename = "mpress.yaml"

type Config struct {
	Site          SiteConfig          `yaml:"site" json:"site"`
	Build         BuildConfig         `yaml:"build" json:"build"`
	Theme         ThemeConfig         `yaml:"theme" json:"theme"`
	Search        SearchConfig        `yaml:"search" json:"search"`
	Knowledge     KnowledgeConfig     `yaml:"knowledge" json:"knowledge"`
	Accessibility AccessibilityConfig `yaml:"accessibility" json:"accessibility"`
	Contribution  ContributionConfig  `yaml:"contribution" json:"contribution"`
	Social        SocialConfig        `yaml:"social" json:"social"`
	Version       VersionConfig       `yaml:"versioning" json:"versioning"`
	Translation   TranslationConfig   `yaml:"translation" json:"translation"`
	Deploy        DeployConfig        `yaml:"deploy" json:"deploy"`
	Blog          BlogConfig          `yaml:"blog" json:"blog"`
}

type SiteConfig struct {
	Title              string            `yaml:"title" json:"title"`
	Description        string            `yaml:"description" json:"description"`
	BaseURL            string            `yaml:"baseURL" json:"baseURL"`
	DefaultLanguage    string            `yaml:"defaultLanguage" json:"defaultLanguage"`
	Languages          []string          `yaml:"languages" json:"languages"`
	LanguageLabels     map[string]string `yaml:"languageLabels" json:"languageLabels"`
	LogoLight          string            `yaml:"logoLight" json:"logoLight"`
	LogoDark           string            `yaml:"logoDark" json:"logoDark"`
	LogoWidth          string            `yaml:"logoWidth" json:"logoWidth"`
	Favicon            string            `yaml:"favicon" json:"favicon"`
	SocialImage        string            `yaml:"socialImage" json:"socialImage"`
	HeaderLinks        []HeaderLink      `yaml:"headerLinks" json:"headerLinks"`
	DefaultAtRoot      bool              `yaml:"defaultLanguageAtRoot" json:"defaultLanguageAtRoot"`
	MissingTranslation string            `yaml:"missingTranslation" json:"missingTranslation"`
}

type HeaderLink struct {
	Label   string `yaml:"label" json:"label"`
	URL     string `yaml:"url" json:"url"`
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`
	Variant string `yaml:"variant,omitempty" json:"variant,omitempty"`
	Color   string `yaml:"color,omitempty" json:"color,omitempty"`
}

type BuildConfig struct {
	ContentDir string `yaml:"contentDir" json:"contentDir"`
	StaticDir  string `yaml:"staticDir" json:"staticDir"`
	OutputDir  string `yaml:"outputDir" json:"outputDir"`
	NavFile    string `yaml:"navFile" json:"navFile"`
	CustomCSS  string `yaml:"customCSS" json:"customCSS"`
}

type ThemeConfig struct {
	AccentColor     string       `yaml:"accentColor" json:"accentColor"`
	HoverColor      string       `yaml:"hoverColor" json:"hoverColor"`
	HoverColorLight string       `yaml:"hoverColorLight" json:"hoverColorLight"`
	HoverColorDark  string       `yaml:"hoverColorDark" json:"hoverColorDark"`
	ColorScheme     string       `yaml:"colorScheme" json:"colorScheme"`
	Layout          LayoutConfig `yaml:"layout" json:"layout"`
}

// LayoutConfig controls the documentation shell without changing landing or
// blog layouts. Presets provide useful defaults, while custom accepts safe CSS
// lengths for projects that need a specific reading width.
type LayoutConfig struct {
	Preset           string `yaml:"preset" json:"preset"`
	ContentWidth     string `yaml:"contentWidth" json:"contentWidth"`
	WideContentWidth string `yaml:"wideContentWidth" json:"wideContentWidth"`
	SidebarWidth     string `yaml:"sidebarWidth" json:"sidebarWidth"`
	TOCWidth         string `yaml:"tocWidth" json:"tocWidth"`
	ContentTOCGap    string `yaml:"contentTocGap" json:"contentTocGap"`
	Alignment        string `yaml:"alignment" json:"alignment"`
	TOC              string `yaml:"toc" json:"toc"`
}

type SearchConfig struct {
	Enabled        bool   `yaml:"enabled" json:"enabled"`
	Shortcut       string `yaml:"shortcut" json:"shortcut"`
	Placeholder    string `yaml:"placeholder" json:"placeholder"`
	MaxResults     int    `yaml:"maxResults" json:"maxResults"`
	RememberRecent bool   `yaml:"rememberRecent" json:"rememberRecent"`
}

// KnowledgeConfig controls the portable, read-only knowledge artifact used by
// local and hosted MCP servers. It is enabled by default and contains no
// credentials or authoring capabilities.
type KnowledgeConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// AccessibilityConfig controls the optional visitor-facing reading tools.
// Preferences are stored only in each visitor's browser.
type AccessibilityConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Shortcut string `yaml:"shortcut" json:"shortcut"`
}

// ContributionConfig controls the public handoff from generated documentation
// to a safe local checkout. The generated site contains repository metadata,
// never credentials.
type ContributionConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Repository string `yaml:"repository" json:"repository"`
	Branch     string `yaml:"branch" json:"branch"`
	QuickEdit  bool   `yaml:"quickEdit" json:"quickEdit"`
	Guide      string `yaml:"guide,omitempty" json:"guide,omitempty"`
}

type SocialConfig struct {
	GitHub  string `yaml:"github" json:"github"`
	Discord string `yaml:"discord" json:"discord"`
	Reddit  string `yaml:"reddit" json:"reddit"`
	X       string `yaml:"x" json:"x"`
	RSS     string `yaml:"rss" json:"rss"`
	Sponsor string `yaml:"sponsor" json:"sponsor"`
	EditURL string `yaml:"editURL" json:"editURL"`
}

type VersionConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	Current   string `yaml:"current" json:"current"`
	Artifacts string `yaml:"artifactsDir" json:"artifactsDir"`
}

// TranslationConfig configures the optional local translation workflow. API
// keys are always read from APIKeyEnv and are never stored in mpress.yaml.
type TranslationConfig struct {
	Provider          string                              `yaml:"provider" json:"provider"`
	Model             string                              `yaml:"model" json:"model"`
	ReasoningEffort   string                              `yaml:"reasoningEffort,omitempty" json:"reasoningEffort,omitempty"`
	LanguageModels    map[string]TranslationLanguageModel `yaml:"languageModels,omitempty" json:"languageModels,omitempty"`
	Command           string                              `yaml:"command,omitempty" json:"command,omitempty"`
	BaseURL           string                              `yaml:"baseURL" json:"baseURL"`
	APIKeyEnv         string                              `yaml:"apiKeyEnv" json:"apiKeyEnv"`
	SourceLanguage    string                              `yaml:"sourceLanguage" json:"sourceLanguage"`
	Glossary          string                              `yaml:"glossary" json:"glossary"`
	StyleGuide        string                              `yaml:"styleGuide" json:"styleGuide"`
	StateDir          string                              `yaml:"stateDir" json:"stateDir"`
	DataCollection    string                              `yaml:"dataCollection" json:"dataCollection"`
	RequireParameters bool                                `yaml:"requireParameters" json:"requireParameters"`
	InputPrice        float64                             `yaml:"inputPricePerMillion" json:"inputPricePerMillion"`
	OutputPrice       float64                             `yaml:"outputPricePerMillion" json:"outputPricePerMillion"`
}

type TranslationLanguageModel struct {
	Model           string `yaml:"model" json:"model"`
	ReasoningEffort string `yaml:"reasoningEffort,omitempty" json:"reasoningEffort,omitempty"`
}

type DeployConfig struct {
	Default string                  `yaml:"default" json:"default"`
	Targets map[string]DeployTarget `yaml:"targets" json:"targets"`
}

type DeployTarget struct {
	Provider         string `yaml:"provider" json:"provider"`
	AccountID        string `yaml:"accountID" json:"accountID"`
	Project          string `yaml:"project" json:"project"`
	ProductionBranch string `yaml:"productionBranch" json:"productionBranch"`
	Domain           string `yaml:"domain" json:"domain"`
}

// BlogConfig controls the defaults used by the generated blog archive. A
// post can override these values in its frontmatter.
type BlogConfig struct {
	LandingStyle    string `yaml:"landingStyle" json:"landingStyle"`
	Tagline         string `yaml:"tagline" json:"tagline"`
	ShowTags        bool   `yaml:"showTags" json:"showTags"`
	HeadingSize     string `yaml:"headingSize" json:"headingSize"`
	ImageMode       string `yaml:"imageMode" json:"imageMode"`
	ImageFit        string `yaml:"imageFit" json:"imageFit"`
	ImageWidth      int    `yaml:"imageWidth" json:"imageWidth"`
	ImageBackground string `yaml:"imageBackground" json:"imageBackground"`
}

func Default() Config {
	return Config{
		Site: SiteConfig{
			Title: "Documentation", DefaultLanguage: "en", Languages: []string{"en"},
			DefaultAtRoot: true, MissingTranslation: "link-to-default", LogoWidth: "160px",
		},
		Build: BuildConfig{ContentDir: "content", StaticDir: "static", OutputDir: "site", NavFile: "_nav.yaml"},
		Theme: ThemeConfig{
			AccentColor: "#5375f6", HoverColor: "#7593ff", ColorScheme: "system",
			Layout: layoutPreset("starlight"),
		},
		Search:        SearchConfig{Enabled: true, Shortcut: "Mod+K", Placeholder: "Search documentation", MaxResults: 12, RememberRecent: true},
		Knowledge:     KnowledgeConfig{Enabled: true},
		Accessibility: AccessibilityConfig{Enabled: true, Shortcut: "Mod+A"},
		Contribution:  ContributionConfig{Branch: "main", Guide: "CONTRIBUTING.md"},
		Version:       VersionConfig{Artifacts: ".mpress/versions"},
		Translation: TranslationConfig{
			Provider: "openrouter", BaseURL: "https://openrouter.ai/api/v1",
			APIKeyEnv: "OPENROUTER_API_KEY", StateDir: ".mpress/translations",
			DataCollection: "deny", RequireParameters: true,
		},
		Deploy: DeployConfig{Targets: map[string]DeployTarget{}},
		Blog:   BlogConfig{LandingStyle: "featured", ShowTags: true, HeadingSize: "default", ImageMode: "panel", ImageFit: "cover", ImageWidth: 100},
	}
}

func layoutPreset(name string) LayoutConfig {
	switch name {
	case "wide":
		return LayoutConfig{Preset: "wide", ContentWidth: "60rem", WideContentWidth: "72rem", SidebarWidth: "18.75rem", TOCWidth: "16rem", ContentTOCGap: "2.5rem", Alignment: "cluster", TOC: "right"}
	case "reading":
		return LayoutConfig{Preset: "reading", ContentWidth: "65ch", WideContentWidth: "52rem", SidebarWidth: "17rem", TOCWidth: "15rem", ContentTOCGap: "2rem", Alignment: "cluster", TOC: "right"}
	default:
		return LayoutConfig{Preset: "starlight", ContentWidth: "50rem", WideContentWidth: "64rem", SidebarWidth: "18.75rem", TOCWidth: "16rem", ContentTOCGap: "2.5rem", Alignment: "cluster", TOC: "right"}
	}
}

func validateLayout(layout *LayoutConfig) error {
	if layout.Preset == "" {
		*layout = layoutPreset("starlight")
		return nil
	}
	switch layout.Preset {
	case "starlight", "wide", "reading":
		*layout = layoutPreset(layout.Preset)
		return nil
	case "custom":
		defaults := layoutPreset("starlight")
		if layout.ContentWidth == "" {
			layout.ContentWidth = defaults.ContentWidth
		}
		if layout.WideContentWidth == "" {
			layout.WideContentWidth = defaults.WideContentWidth
		}
		if layout.SidebarWidth == "" {
			layout.SidebarWidth = defaults.SidebarWidth
		}
		if layout.TOCWidth == "" {
			layout.TOCWidth = defaults.TOCWidth
		}
		if layout.ContentTOCGap == "" {
			layout.ContentTOCGap = defaults.ContentTOCGap
		}
		if layout.Alignment == "" {
			layout.Alignment = defaults.Alignment
		}
		if layout.TOC == "" {
			layout.TOC = defaults.TOC
		}
	default:
		return errors.New("theme.layout.preset must be starlight, wide, reading or custom")
	}
	for field, value := range map[string]string{
		"contentWidth": layout.ContentWidth, "wideContentWidth": layout.WideContentWidth,
		"sidebarWidth": layout.SidebarWidth, "tocWidth": layout.TOCWidth, "contentTocGap": layout.ContentTOCGap,
	} {
		allowPercent := field == "contentWidth" || field == "wideContentWidth"
		if !validLayoutLength(value, allowPercent) {
			suffix := ""
			if allowPercent {
				suffix = ", or a percentage from 40% to 100%"
			}
			return fmt.Errorf("theme.layout.%s must be a positive px, rem or ch length%s", field, suffix)
		}
	}
	if layout.Alignment != "cluster" && layout.Alignment != "left" {
		return errors.New("theme.layout.alignment must be cluster or left")
	}
	if layout.TOC != "right" && layout.TOC != "hidden" {
		return errors.New("theme.layout.toc must be right or hidden")
	}
	return nil
}

var layoutLength = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]+)?|\.[0-9]+)(px|rem|ch|%)$`)

func validLayoutLength(value string, allowPercent bool) bool {
	match := layoutLength.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return false
	}
	numberText := strings.TrimSuffix(strings.TrimSpace(value), match[1])
	number, err := strconv.ParseFloat(numberText, 64)
	if err != nil || number <= 0 {
		return false
	}
	if match[1] == "%" {
		return allowPercent && number >= 40 && number <= 100
	}
	return true
}

func Load(projectDir string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(projectDir, Filename))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), fmt.Errorf("%s not found", Filename)
		}
		return Default(), err
	}
	return Parse(data)
}

// Parse loads and validates configuration data while applying the same defaults
// as Load. Development tools use it before a configuration edit reaches disk.
func Parse(data []byte) (Config, error) {
	cfg := Default()
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", Filename, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(projectDir string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	path := filepath.Join(projectDir, Filename)
	tmp, err := os.CreateTemp(projectDir, ".mpress-config-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Site.Title) == "" {
		return errors.New("site.title is required")
	}
	if c.Site.BaseURL != "" {
		base, err := url.Parse(c.Site.BaseURL)
		if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.RawQuery != "" || base.Fragment != "" {
			return errors.New("site.baseURL must be an absolute HTTP or HTTPS URL without a query or fragment")
		}
		c.Site.BaseURL = strings.TrimRight(c.Site.BaseURL, "/")
	}
	if c.Site.DefaultLanguage == "" {
		c.Site.DefaultLanguage = "en"
	}
	if c.Site.LogoWidth == "" {
		c.Site.LogoWidth = "160px"
	}
	if !validLayoutLength(c.Site.LogoWidth, false) {
		return errors.New("site.logoWidth must be a positive px, rem or ch length")
	}
	if c.Theme.AccentColor == "" {
		c.Theme.AccentColor = "#5375f6"
	}
	if c.Theme.HoverColor == "" {
		c.Theme.HoverColor = "#7593ff"
	}
	if c.Theme.HoverColorLight == "" {
		c.Theme.HoverColorLight = c.Theme.HoverColor
	}
	if c.Theme.HoverColorDark == "" {
		c.Theme.HoverColorDark = c.Theme.HoverColor
	}
	colour := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if !colour.MatchString(c.Theme.AccentColor) {
		return errors.New("theme.accentColor must be a six-digit hex colour")
	}
	if !colour.MatchString(c.Theme.HoverColor) {
		return errors.New("theme.hoverColor must be a six-digit hex colour")
	}
	if !colour.MatchString(c.Theme.HoverColorLight) {
		return errors.New("theme.hoverColorLight must be a six-digit hex colour")
	}
	if !colour.MatchString(c.Theme.HoverColorDark) {
		return errors.New("theme.hoverColorDark must be a six-digit hex colour")
	}
	if c.Theme.ColorScheme == "" {
		c.Theme.ColorScheme = "system"
	}
	switch c.Theme.ColorScheme {
	case "system", "light", "dark":
	default:
		return errors.New("theme.colorScheme must be system, light or dark")
	}
	if err := validateLayout(&c.Theme.Layout); err != nil {
		return err
	}
	if c.Search.Shortcut == "" {
		c.Search.Shortcut = "Mod+K"
	}
	c.Search.Placeholder = strings.TrimSpace(c.Search.Placeholder)
	if c.Search.Placeholder == "" {
		c.Search.Placeholder = "Search documentation"
	}
	if len([]rune(c.Search.Placeholder)) > 80 {
		return errors.New("search.placeholder must be 80 characters or fewer")
	}
	if c.Search.MaxResults == 0 {
		c.Search.MaxResults = 12
	}
	if c.Search.MaxResults < 4 || c.Search.MaxResults > 24 {
		return errors.New("search.maxResults must be from 4 to 24")
	}
	if c.Accessibility.Shortcut == "" {
		c.Accessibility.Shortcut = "Mod+A"
	}
	var err error
	if c.Search.Shortcut, err = normaliseShortcut(c.Search.Shortcut); err != nil {
		return fmt.Errorf("search.shortcut: %w", err)
	}
	if c.Accessibility.Shortcut, err = normaliseShortcut(c.Accessibility.Shortcut); err != nil {
		return fmt.Errorf("accessibility.shortcut: %w", err)
	}
	if c.Search.Shortcut != "None" && c.Search.Shortcut == c.Accessibility.Shortcut {
		return errors.New("search.shortcut and accessibility.shortcut must be different")
	}
	if c.Translation.Provider == "" {
		c.Translation.Provider = "openrouter"
	}
	if c.Translation.BaseURL == "" {
		switch c.Translation.Provider {
		case "openrouter":
			c.Translation.BaseURL = "https://openrouter.ai/api/v1"
		case "openai-compatible", "openai":
			c.Translation.BaseURL = "https://api.openai.com/v1"
		}
	}
	if c.Translation.APIKeyEnv == "" && c.Translation.Provider != "codex" && c.Translation.Provider != "claude" {
		if c.Translation.Provider == "openrouter" {
			c.Translation.APIKeyEnv = "OPENROUTER_API_KEY"
		} else {
			c.Translation.APIKeyEnv = "OPENAI_API_KEY"
		}
	}
	if c.Translation.SourceLanguage == "" {
		c.Translation.SourceLanguage = c.Site.DefaultLanguage
	}
	if c.Translation.StateDir == "" {
		c.Translation.StateDir = ".mpress/translations"
	}
	if c.Translation.DataCollection == "" && c.Translation.Provider == "openrouter" {
		c.Translation.DataCollection = "deny"
	}
	switch c.Translation.Provider {
	case "openrouter", "openai", "openai-compatible", "codex", "claude":
	default:
		return fmt.Errorf("translation.provider %q is not supported", c.Translation.Provider)
	}
	if c.Translation.SourceLanguage != c.Site.DefaultLanguage {
		return fmt.Errorf("translation.sourceLanguage must match site.defaultLanguage %q", c.Site.DefaultLanguage)
	}
	if c.Translation.APIKeyEnv != "" && !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(c.Translation.APIKeyEnv) {
		return errors.New("translation.apiKeyEnv must be an environment variable name")
	}
	if c.Translation.DataCollection != "" && c.Translation.DataCollection != "deny" && c.Translation.DataCollection != "allow" {
		return errors.New("translation.dataCollection must be deny, allow, or empty")
	}
	if c.Translation.InputPrice < 0 || c.Translation.OutputPrice < 0 {
		return errors.New("translation prices per million tokens must not be negative")
	}
	stateDir := filepath.Clean(filepath.FromSlash(c.Translation.StateDir))
	if stateDir == "." || stateDir == ".." || filepath.IsAbs(stateDir) || strings.HasPrefix(stateDir, ".."+string(filepath.Separator)) {
		return errors.New("translation.stateDir must stay inside the project")
	}
	if c.Translation.Model == "" {
		// A model is only required when translation is run. Keeping it empty lets
		// new projects build without choosing a paid provider.
	}
	if !validReasoningEffort(c.Translation.ReasoningEffort) {
		return errors.New("translation.reasoningEffort must be none, minimal, low, medium, high, xhigh, max, or empty")
	}
	if len(c.Site.Languages) == 0 {
		c.Site.Languages = []string{c.Site.DefaultLanguage}
	}
	found := false
	seen := map[string]bool{}
	for _, lang := range c.Site.Languages {
		if lang == "" || seen[lang] {
			return fmt.Errorf("site.languages contains an empty or duplicate language %q", lang)
		}
		seen[lang] = true
		if lang == c.Site.DefaultLanguage {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("site.defaultLanguage %q is not in site.languages", c.Site.DefaultLanguage)
	}
	for lang, selection := range c.Translation.LanguageModels {
		if !seen[lang] || lang == c.Site.DefaultLanguage {
			return fmt.Errorf("translation.languageModels contains unknown target language %q", lang)
		}
		if strings.TrimSpace(selection.Model) == "" {
			return fmt.Errorf("translation.languageModels.%s.model is required", lang)
		}
		if !validReasoningEffort(selection.ReasoningEffort) {
			return fmt.Errorf("translation.languageModels.%s.reasoningEffort is invalid", lang)
		}
	}
	if c.Build.ContentDir == "" || c.Build.StaticDir == "" || c.Build.OutputDir == "" {
		return errors.New("build contentDir, staticDir and outputDir must not be empty")
	}
	if c.Build.NavFile == "" {
		c.Build.NavFile = "_nav.yaml"
	}
	if c.Blog.HeadingSize == "" {
		c.Blog.HeadingSize = "default"
	}
	if c.Blog.LandingStyle == "" {
		c.Blog.LandingStyle = "featured"
	}
	if c.Blog.ImageMode == "" {
		c.Blog.ImageMode = "panel"
	}
	if c.Blog.ImageFit == "" {
		c.Blog.ImageFit = "cover"
	}
	if c.Blog.ImageWidth == 0 {
		c.Blog.ImageWidth = 100
	}
	if c.Blog.HeadingSize != "default" && c.Blog.HeadingSize != "compact" && c.Blog.HeadingSize != "large" {
		return errors.New("blog.headingSize must be default, compact or large")
	}
	if c.Blog.LandingStyle != "featured" && c.Blog.LandingStyle != "grid" && c.Blog.LandingStyle != "list" {
		return errors.New("blog.landingStyle must be featured, grid or list")
	}
	if c.Blog.ImageMode != "panel" && c.Blog.ImageMode != "floating" {
		return errors.New("blog.imageMode must be panel or floating")
	}
	if c.Blog.ImageFit != "cover" && c.Blog.ImageFit != "contain" {
		return errors.New("blog.imageFit must be cover or contain")
	}
	if c.Blog.ImageWidth < 50 || c.Blog.ImageWidth > 100 {
		return errors.New("blog.imageWidth must be between 50 and 100")
	}
	if c.Blog.ImageBackground != "" && !regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(c.Blog.ImageBackground) {
		return errors.New("blog.imageBackground must be a six-digit hex colour")
	}
	if c.Site.MissingTranslation == "" {
		c.Site.MissingTranslation = "link-to-default"
	}
	if c.Site.MissingTranslation != "link-to-default" && c.Site.MissingTranslation != "omit" {
		return fmt.Errorf("site.missingTranslation must be link-to-default or omit")
	}
	if c.Contribution.Branch == "" {
		c.Contribution.Branch = "main"
	}
	if c.Contribution.Guide == "" {
		c.Contribution.Guide = "CONTRIBUTING.md"
	}
	if c.Contribution.Enabled && strings.TrimSpace(c.Contribution.Repository) == "" {
		return errors.New("contribution.repository is required when contribution is enabled")
	}
	if strings.ContainsAny(c.Contribution.Repository, "\r\n") {
		return errors.New("contribution.repository must be a single URL")
	}
	if strings.ContainsAny(c.Contribution.Branch, "\r\n") {
		return errors.New("contribution.branch must be a single branch name")
	}
	if strings.ContainsAny(c.Contribution.Guide, "\r\n") || filepath.IsAbs(c.Contribution.Guide) || strings.HasPrefix(filepath.ToSlash(filepath.Clean(c.Contribution.Guide)), "../") {
		return errors.New("contribution.guide must be a safe project-relative path")
	}
	for index, link := range c.Site.HeaderLinks {
		if strings.TrimSpace(link.Label) == "" || strings.TrimSpace(link.URL) == "" {
			return fmt.Errorf("site.headerLinks[%d] requires label and url", index)
		}
		if link.Type != "" && link.Type != "link" && link.Type != "button" {
			return fmt.Errorf("site.headerLinks[%d].type must be link or button", index)
		}
		if link.Variant != "" && link.Type != "button" {
			return fmt.Errorf("site.headerLinks[%d].variant requires type button", index)
		}
		switch link.Variant {
		case "", "primary", "secondary", "outline", "custom":
		default:
			return fmt.Errorf("site.headerLinks[%d].variant must be primary, secondary, outline or custom", index)
		}
		if link.Color != "" && (link.Type != "button" || link.Variant != "custom") {
			return fmt.Errorf("site.headerLinks[%d].color requires type button and variant custom", index)
		}
		if link.Variant == "custom" && !colour.MatchString(link.Color) {
			return fmt.Errorf("site.headerLinks[%d].color must be a six-digit hex colour for a custom button", index)
		}
	}
	if c.Deploy.Targets == nil {
		c.Deploy.Targets = map[string]DeployTarget{}
	}
	if c.Deploy.Default != "" {
		if _, ok := c.Deploy.Targets[c.Deploy.Default]; !ok {
			return fmt.Errorf("deploy.default %q does not name a deploy target", c.Deploy.Default)
		}
	}
	for name, target := range c.Deploy.Targets {
		if strings.TrimSpace(name) == "" {
			return errors.New("deploy.targets contains an empty target name")
		}
		switch target.Provider {
		case "cloudflare-pages":
			if strings.TrimSpace(target.AccountID) == "" || strings.TrimSpace(target.Project) == "" {
				return fmt.Errorf("deploy target %q requires accountID and project", name)
			}
		case "netlify":
			if strings.TrimSpace(target.Project) == "" {
				return fmt.Errorf("deploy target %q requires a Netlify site ID, domain, or name", name)
			}
		case "":
			return fmt.Errorf("deploy target %q requires a provider", name)
		default:
			return fmt.Errorf("deploy target %q uses unsupported provider %q", name, target.Provider)
		}
	}
	return nil
}

func validReasoningEffort(value string) bool {
	switch value {
	case "", "none", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	default:
		return false
	}
}

func normaliseShortcut(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "none") {
		return "None", nil
	}
	parts := strings.Split(value, "+")
	if len(parts) < 2 {
		return "", errors.New("must contain a modifier and a key, or be None")
	}
	modifiers := map[string]bool{}
	key := ""
	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			return "", errors.New("contains an empty key")
		}
		canonical := ""
		switch strings.ToLower(part) {
		case "mod":
			canonical = "Mod"
		case "command", "cmd", "meta":
			canonical = "Meta"
		case "control", "ctrl":
			canonical = "Control"
		case "option", "alt":
			canonical = "Alt"
		case "shift":
			canonical = "Shift"
		}
		if canonical != "" {
			if modifiers[canonical] {
				return "", fmt.Errorf("contains duplicate modifier %s", canonical)
			}
			modifiers[canonical] = true
			continue
		}
		if key != "" {
			return "", errors.New("must contain exactly one non-modifier key")
		}
		runes := []rune(part)
		if len(runes) == 1 {
			key = strings.ToUpper(part)
			continue
		}
		switch strings.ToLower(part) {
		case "enter":
			key = "Enter"
		case "escape", "esc":
			key = "Escape"
		case "space":
			key = "Space"
		default:
			upper := strings.ToUpper(part)
			if matched, _ := regexp.MatchString(`^F(?:[1-9]|1[0-2])$`, upper); matched {
				key = upper
			} else {
				return "", fmt.Errorf("unsupported key %q", part)
			}
		}
	}
	if len(modifiers) == 0 || key == "" {
		return "", errors.New("must contain a modifier and exactly one key")
	}
	ordered := make([]string, 0, len(modifiers)+1)
	for _, modifier := range []string{"Mod", "Control", "Meta", "Alt", "Shift"} {
		if modifiers[modifier] {
			ordered = append(ordered, modifier)
		}
	}
	return strings.Join(append(ordered, key), "+"), nil
}

func FindProject(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, Filename)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no %s found from %s", Filename, start)
}

func (c Config) ContentPath(root string) string   { return join(root, c.Build.ContentDir) }
func (c Config) StaticPath(root string) string    { return join(root, c.Build.StaticDir) }
func (c Config) OutputPath(root string) string    { return join(root, c.Build.OutputDir) }
func (c Config) ArtifactsPath(root string) string { return join(root, c.Version.Artifacts) }
func join(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}
