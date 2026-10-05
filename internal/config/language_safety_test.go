package config

import (
	"testing"
)

func TestValidateRejectsLanguagePathComponents(t *testing.T) {
	for _, lang := range []string{".", "..", "../outside", "fr/ca", `fr\ca`, `C:\fr`, "C:", "/fr", "%2e%2e", "fr?query", "fr#hash", "NUL", "LPT¹", "fr.", "fr ", "fr\x00ca"} {
		t.Run(lang, func(t *testing.T) {
			cfg := Default()
			cfg.Site.Languages = []string{"en", lang}
			if err := cfg.Validate(); err == nil {
				t.Error("unsafe language component accepted")
			}
		})
	}
	for _, lang := range []string{"zh-Hans", "pt-BR", "en_US", "日本語"} {
		cfg := Default()
		cfg.Site.DefaultLanguage, cfg.Site.Languages = lang, []string{lang}
		if err := cfg.Validate(); err != nil {
			t.Errorf("valid language %q: %v", lang, err)
		}
	}
}
