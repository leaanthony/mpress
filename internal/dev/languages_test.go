package dev

import "testing"

func TestStandardTranslationLanguagesContainCompleteISO6391SetAndCommonVariants(t *testing.T) {
	choices := standardTranslationLanguages()
	if len(choices) < len(iso6391LanguageCodes) {
		t.Fatalf("language catalog has %d entries, want at least %d", len(choices), len(iso6391LanguageCodes))
	}
	wanted := map[string]string{"fr": "French", "ja": "Japanese", "pt-BR": "Brazilian Portuguese", "zh-Hant": "Traditional Chinese"}
	for _, choice := range choices {
		if name, ok := wanted[choice.Code]; ok {
			if choice.Name != name {
				t.Errorf("%s has unexpected standard name %q", choice.Code, choice.Name)
			}
			delete(wanted, choice.Code)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("language catalog is missing %#v", wanted)
	}
}

func TestStandardLanguageChoiceIsCaseInsensitive(t *testing.T) {
	choice, ok := standardLanguageChoice("zh-hant")
	if !ok || choice.Code != "zh-Hant" || choice.Name == "" {
		t.Fatalf("unexpected choice: %#v, %t", choice, ok)
	}
}
