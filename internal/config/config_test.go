package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDefaultConfigurationMarshalsEverySupportedSection(t *testing.T) {
	data, err := yaml.Marshal(Default())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"socialImage:", "hoverColorLight:", "hoverColorDark:", "contentWidth:", "wideContentWidth:",
		"logoWidth:",
		"sidebarWidth:", "tocWidth:", "contentTocGap:", "alignment:", "toc:", "accessibility:",
		"shortcut: Mod+K", "shortcut: Mod+A", "placeholder: Search documentation", "maxResults: 12", "rememberRecent: true",
		"repository:", "reddit:", "rss:", "sponsor:", "model:", "glossary:", "styleGuide:",
		"inputPricePerMillion:", "outputPricePerMillion:",
		"deploy:", "default:", "targets:",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("marshalled default configuration is missing %q:\n%s", want, data)
		}
	}
}

func TestValidateTranslationPrices(t *testing.T) {
	cfg := Default()
	cfg.Translation.InputPrice = -1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "prices per million") {
		t.Fatalf("negative translation input price was accepted: %v", err)
	}
	cfg = Default()
	cfg.Translation.OutputPrice = -1
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "prices per million") {
		t.Fatalf("negative translation output price was accepted: %v", err)
	}
}

func TestLoadAppliesDefaultsAndRejectsUnknownFields(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		root := t.TempDir()
		write(t, filepath.Join(root, Filename), "site:\n  title: Test\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n")
		cfg, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Site.DefaultLanguage != "en" || cfg.Site.LogoWidth != "160px" || cfg.Build.NavFile != "_nav.yaml" || cfg.Version.Artifacts != ".mpress/versions" || cfg.Theme.AccentColor != "#5375f6" || cfg.Theme.HoverColor != "#7593ff" || cfg.Theme.HoverColorLight != "#7593ff" || cfg.Theme.HoverColorDark != "#7593ff" || !cfg.Accessibility.Enabled || cfg.Search.Shortcut != "Mod+K" || cfg.Search.Placeholder != "Search documentation" || cfg.Search.MaxResults != 12 || !cfg.Search.RememberRecent || cfg.Accessibility.Shortcut != "Mod+A" {
			t.Fatalf("defaults not applied: %#v", cfg)
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		root := t.TempDir()
		write(t, filepath.Join(root, Filename), "site:\n  title: Test\n  typo: true\n")
		_, err := Load(root)
		if err == nil || !strings.Contains(err.Error(), "field typo not found") {
			t.Fatalf("expected useful unknown-field error, got %v", err)
		}
	})
}

func TestValidateKeyboardShortcuts(t *testing.T) {
	cfg := Default()
	cfg.Search.Shortcut = "ctrl + shift + f"
	cfg.Accessibility.Shortcut = "command+a"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Search.Shortcut != "Control+Shift+F" || cfg.Accessibility.Shortcut != "Meta+A" {
		t.Fatalf("shortcuts were not normalised: search=%q accessibility=%q", cfg.Search.Shortcut, cfg.Accessibility.Shortcut)
	}

	for name, mutate := range map[string]func(*Config){
		"missing modifier": func(cfg *Config) { cfg.Search.Shortcut = "K" },
		"multiple keys":    func(cfg *Config) { cfg.Search.Shortcut = "Mod+K+P" },
		"duplicate":        func(cfg *Config) { cfg.Accessibility.Shortcut = "Mod+K" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := Default()
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatalf("invalid shortcuts were accepted: %#v", invalid)
			}
		})
	}

	disabled := Default()
	disabled.Search.Shortcut = "none"
	if err := disabled.Validate(); err != nil || disabled.Search.Shortcut != "None" {
		t.Fatalf("disabled shortcut was rejected: %q, %v", disabled.Search.Shortcut, err)
	}
}

func TestValidateSearchPresentation(t *testing.T) {
	cfg := Default()
	cfg.Search.Placeholder = "  Find these docs  "
	cfg.Search.MaxResults = 18
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Search.Placeholder != "Find these docs" || cfg.Search.MaxResults != 18 {
		t.Fatalf("search presentation was not normalised: %#v", cfg.Search)
	}

	for name, mutate := range map[string]func(*Config){
		"too few results":  func(cfg *Config) { cfg.Search.MaxResults = 3 },
		"too many results": func(cfg *Config) { cfg.Search.MaxResults = 25 },
		"long prompt": func(cfg *Config) {
			cfg.Search.Placeholder = strings.Repeat("x", 81)
		},
	} {
		t.Run(name, func(t *testing.T) {
			invalid := Default()
			mutate(&invalid)
			if err := invalid.Validate(); err == nil {
				t.Fatalf("invalid search settings were accepted: %#v", invalid.Search)
			}
		})
	}
}

func TestValidateLogoWidth(t *testing.T) {
	cfg := Default()
	for _, value := range []string{"0px", "100%", "calc(100% - 1rem)"} {
		cfg.Site.LogoWidth = value
		if err := cfg.Validate(); err == nil {
			t.Fatalf("invalid logo width %q was accepted", value)
		}
	}
	cfg = Default()
	cfg.Site.LogoWidth = "12rem"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid logo width was rejected: %v", err)
	}
}

func TestValidateThemeColoursAndScheme(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"invalid accent":      func(cfg *Config) { cfg.Theme.AccentColor = "blue" },
		"invalid hover":       func(cfg *Config) { cfg.Theme.HoverColor = "#123" },
		"invalid light hover": func(cfg *Config) { cfg.Theme.HoverColorLight = "#123" },
		"invalid dark hover":  func(cfg *Config) { cfg.Theme.HoverColorDark = "white" },
		"invalid scheme":      func(cfg *Config) { cfg.Theme.ColorScheme = "auto" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Default()
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("invalid theme configuration was accepted: %#v", cfg.Theme)
			}
		})
	}
}

func TestValidatePublicBaseURL(t *testing.T) {
	for _, value := range []string{"docs.example.com", "javascript:alert(1)", "https://docs.example.com/?preview=1", "https://docs.example.com/#top"} {
		cfg := Default()
		cfg.Site.BaseURL = value
		if err := cfg.Validate(); err == nil {
			t.Fatalf("invalid site.baseURL %q was accepted", value)
		}
	}
	cfg := Default()
	cfg.Site.BaseURL = "https://docs.example.com/product/"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Site.BaseURL != "https://docs.example.com/product" {
		t.Fatalf("site.baseURL was not normalised: %q", cfg.Site.BaseURL)
	}
}

func TestContributionConfigurationRequiresARepository(t *testing.T) {
	cfg := Default()
	cfg.Contribution.Enabled = true
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "contribution.repository") {
		t.Fatalf("enabled contribution without a repository was accepted: %v", err)
	}
	cfg.Contribution.Repository = "https://github.com/example/docs.git"
	cfg.Contribution.Branch = ""
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Contribution.Branch != "main" {
		t.Fatalf("default contribution branch = %q", cfg.Contribution.Branch)
	}
	if cfg.Contribution.Guide != "CONTRIBUTING.md" {
		t.Fatalf("default contributor guide = %q", cfg.Contribution.Guide)
	}
	cfg.Contribution.Guide = "../private.md"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "contribution.guide") {
		t.Fatalf("contributor guide outside the project was accepted: %v", err)
	}
}

func TestLegacyHoverColourIsTheThemeFallback(t *testing.T) {
	cfg, err := Parse([]byte("site:\n  title: Legacy\ntheme:\n  hoverColor: '#ffffff'\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme.HoverColorLight != "#ffffff" || cfg.Theme.HoverColorDark != "#ffffff" {
		t.Fatalf("legacy hover colour was not used as the fallback: %#v", cfg.Theme)
	}

	cfg, err = Parse([]byte("site:\n  title: Split\ntheme:\n  hoverColor: '#ffffff'\n  hoverColorLight: '#111111'\n  hoverColorDark: '#eeeeee'\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme.HoverColorLight != "#111111" || cfg.Theme.HoverColorDark != "#eeeeee" {
		t.Fatalf("theme-specific hover colours were not preserved: %#v", cfg.Theme)
	}
}

func TestLayoutPresetsAndCustomMeasurements(t *testing.T) {
	cfg := Default()
	if cfg.Theme.Layout.Preset != "starlight" || cfg.Theme.Layout.ContentWidth != "50rem" || cfg.Theme.Layout.SidebarWidth != "18.75rem" {
		t.Fatalf("unexpected default layout: %#v", cfg.Theme.Layout)
	}

	cfg.Theme.Layout = LayoutConfig{Preset: "wide", ContentWidth: "1px"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Theme.Layout.ContentWidth != "60rem" || cfg.Theme.Layout.WideContentWidth != "72rem" {
		t.Fatalf("wide preset was not resolved: %#v", cfg.Theme.Layout)
	}

	cfg.Theme.Layout = LayoutConfig{
		Preset: "custom", ContentWidth: "75%", WideContentWidth: "1100px",
		SidebarWidth: "19rem", TOCWidth: "240px", ContentTOCGap: "2rem",
		Alignment: "left", TOC: "hidden",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid custom layout was rejected: %v", err)
	}

	for name, value := range map[string]string{"css injection": "720px;display:none", "small percentage": "20%", "unsupported unit": "80vw"} {
		t.Run(name, func(t *testing.T) {
			invalid := Default()
			invalid.Theme.Layout = LayoutConfig{Preset: "custom", ContentWidth: value}
			if err := invalid.Validate(); err == nil {
				t.Fatalf("invalid content width %q was accepted", value)
			}
		})
	}
}

func TestParseAppliesDefaultsAndValidatesInMemoryConfiguration(t *testing.T) {
	cfg, err := Parse([]byte("site:\n  title: Preview\nbuild:\n  contentDir: content\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Site.Title != "Preview" || cfg.Site.DefaultLanguage != "en" || !cfg.Search.Enabled || cfg.Search.Placeholder != "Search documentation" || cfg.Search.MaxResults != 12 || !cfg.Search.RememberRecent || !cfg.Accessibility.Enabled || cfg.Blog.LandingStyle != "featured" {
		t.Fatalf("defaults not applied to parsed configuration: %#v", cfg)
	}
	disabled, err := Parse([]byte("site:\n  title: Preview\naccessibility:\n  enabled: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Accessibility.Enabled {
		t.Fatal("explicitly disabled accessibility menu was re-enabled")
	}
	if _, err := Parse([]byte("site:\n  title: Preview\n  unknownOption: true\n")); err == nil {
		t.Fatal("unknown configuration option was accepted")
	}
}

func TestValidateBlogLandingStyle(t *testing.T) {
	cfg := Default()
	cfg.Blog.LandingStyle = "magazine"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "blog.landingStyle") {
		t.Fatalf("unknown blog landing style was accepted: %v", err)
	}
}

func TestValidateLanguages(t *testing.T) {
	cfg := Default()
	cfg.Site.Languages = []string{"fr"}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "not in site.languages") {
		t.Fatalf("expected default-language error, got %v", err)
	}
}

func TestValidateHeaderLinks(t *testing.T) {
	cfg := Default()
	cfg.Site.HeaderLinks = []HeaderLink{{Label: "Docs"}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "requires label and url") {
		t.Fatalf("expected header-link error, got %v", err)
	}
	cfg = Default()
	cfg.Site.HeaderLinks = []HeaderLink{{Label: "Support", URL: "/support/", Type: "pill"}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "type must be link or button") {
		t.Fatalf("expected header-link type error, got %v", err)
	}
	cfg.Site.HeaderLinks[0].Type = "button"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("button header link was rejected: %v", err)
	}
	cfg.Site.HeaderLinks[0].Variant = "outline"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("outline header button was rejected: %v", err)
	}
	cfg.Site.HeaderLinks[0].Variant = "custom"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "six-digit hex colour") {
		t.Fatalf("custom header button without a colour was accepted: %v", err)
	}
	cfg.Site.HeaderLinks[0].Color = "#7c3aed"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("custom header button colour was rejected: %v", err)
	}
	cfg.Site.HeaderLinks[0].Variant = "rounded"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "variant must be") {
		t.Fatalf("unknown header button variant was accepted: %v", err)
	}
}

func TestValidateTranslationSafety(t *testing.T) {
	for name, mutate := range map[string]func(*Config){
		"state outside project":        func(cfg *Config) { cfg.Translation.StateDir = "../translations" },
		"invalid environment variable": func(cfg *Config) { cfg.Translation.APIKeyEnv = "not valid" },
		"wrong source language":        func(cfg *Config) { cfg.Translation.SourceLanguage = "fr" },
		"unknown privacy policy":       func(cfg *Config) { cfg.Translation.DataCollection = "maybe" },
		"unknown reasoning effort":     func(cfg *Config) { cfg.Translation.ReasoningEffort = "expensive" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Default()
			mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("unsafe translation configuration was accepted: %#v", cfg.Translation)
			}
		})
	}
	for name, selection := range map[string]TranslationLanguageModel{
		"unknown target": {Model: "example/model"},
		"missing model":  {},
		"bad reasoning":  {Model: "example/model", ReasoningEffort: "expensive"},
	} {
		t.Run("language model "+name, func(t *testing.T) {
			cfg := Default()
			cfg.Site.Languages = []string{"en", "fr"}
			key := "fr"
			if name == "unknown target" {
				key = "de"
			}
			cfg.Translation.LanguageModels = map[string]TranslationLanguageModel{key: selection}
			if err := cfg.Validate(); err == nil {
				t.Fatalf("unsafe language model configuration was accepted: %#v", cfg.Translation.LanguageModels)
			}
		})
	}
	cfg := Default()
	cfg.Translation.Provider = "codex"
	cfg.Translation.APIKeyEnv = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("local Codex configuration should not require an API key: %v", err)
	}
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
