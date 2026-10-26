package config

import "testing"

func TestConfigCloneCopiesEveryMutableField(t *testing.T) {
	original := Default()
	original.Site.Languages = []string{"en", "fr"}
	original.Site.LanguageLabels = map[string]string{"en": "English", "fr": "Français"}
	original.Site.HeaderLinks = []HeaderLink{{Label: "Home", URL: "/"}}
	original.Translation.LanguageModels = map[string]TranslationLanguageModel{"fr": {Model: "model-a"}}
	original.Deploy.Targets = map[string]DeployTarget{"production": {Provider: "cloudflare-pages", Project: "docs"}}

	clone := original.Clone()
	clone.Site.Languages[0] = "de"
	clone.Site.LanguageLabels["fr"] = "changed"
	clone.Site.HeaderLinks[0].Label = "Changed"
	clone.Translation.LanguageModels["fr"] = TranslationLanguageModel{Model: "model-b"}
	clone.Deploy.Targets["production"] = DeployTarget{Project: "other"}

	if original.Site.Languages[0] != "en" {
		t.Fatalf("languages slice was shared: %v", original.Site.Languages)
	}
	if original.Site.LanguageLabels["fr"] != "Français" {
		t.Fatalf("language labels map was shared: %v", original.Site.LanguageLabels)
	}
	if original.Site.HeaderLinks[0].Label != "Home" {
		t.Fatalf("header links slice was shared: %v", original.Site.HeaderLinks)
	}
	if original.Translation.LanguageModels["fr"].Model != "model-a" {
		t.Fatalf("language models map was shared: %v", original.Translation.LanguageModels)
	}
	if original.Deploy.Targets["production"].Project != "docs" {
		t.Fatalf("deploy targets map was shared: %v", original.Deploy.Targets)
	}
}
