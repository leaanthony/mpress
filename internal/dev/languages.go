package dev

import (
	"sort"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// languageChoice is a translation target that can be selected in the admin
// interface. The base list is the complete ISO 639-1 alpha-2 set. Common BCP
// 47 script and regional variants are included where translation output must
// distinguish between writing systems or national usage.
type languageChoice struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"nativeName,omitempty"`
}

var iso6391LanguageCodes = []string{
	"aa", "ab", "ae", "af", "ak", "am", "an", "ar", "as", "av", "ay", "az", "ba", "be", "bg", "bi", "bm", "bn", "bo", "br", "bs", "ca", "ce", "ch", "co", "cr", "cs", "cu", "cv", "cy", "da", "de", "dv", "dz", "ee", "el", "en", "eo", "es", "et", "eu", "fa", "ff", "fi", "fj", "fo", "fr", "fy", "ga", "gd", "gl", "gn", "gu", "gv", "ha", "he", "hi", "ho", "hr", "ht", "hu", "hy", "hz", "ia", "id", "ie", "ig", "ii", "ik", "io", "is", "it", "iu", "ja", "jv", "ka", "kg", "ki", "kj", "kk", "kl", "km", "kn", "ko", "kr", "ks", "ku", "kv", "kw", "ky", "la", "lb", "lg", "li", "ln", "lo", "lt", "lu", "lv", "mg", "mh", "mi", "mk", "ml", "mn", "mr", "ms", "mt", "my", "na", "nb", "nd", "ne", "ng", "nl", "nn", "no", "nr", "nv", "ny", "oc", "oj", "om", "or", "os", "pa", "pi", "pl", "ps", "pt", "qu", "rm", "rn", "ro", "ru", "rw", "sa", "sc", "sd", "se", "sg", "si", "sk", "sl", "sm", "sn", "so", "sq", "sr", "ss", "st", "su", "sv", "sw", "ta", "te", "tg", "th", "ti", "tk", "tl", "tn", "to", "tr", "ts", "tt", "tw", "ty", "ug", "uk", "ur", "uz", "ve", "vi", "vo", "wa", "wo", "xh", "yi", "yo", "za", "zh", "zu",
}

var commonTranslationVariants = []string{
	"ar-001", "az-Cyrl", "az-Latn", "en-GB", "en-US", "es-ES", "es-419",
	"fr-CA", "fr-FR", "pt-BR", "pt-PT", "sr-Cyrl", "sr-Latn", "uz-Cyrl",
	"uz-Latn", "zh-CN", "zh-TW", "zh-Hans", "zh-Hant",
}

func standardTranslationLanguages() []languageChoice {
	codes := append(append([]string{}, iso6391LanguageCodes...), commonTranslationVariants...)
	choices := make([]languageChoice, 0, len(codes))
	seen := map[string]bool{}
	for _, code := range codes {
		tag, err := language.Parse(code)
		if err != nil {
			continue
		}
		canonical := tag.String()
		key := strings.ToLower(canonical)
		if seen[key] {
			continue
		}
		seen[key] = true
		name := display.English.Tags().Name(tag)
		native := display.Self.Name(tag)
		if nativeNamer := display.Tags(tag); nativeNamer != nil {
			native = nativeNamer.Name(tag)
		}
		if native == name {
			native = ""
		}
		choices = append(choices, languageChoice{Code: canonical, Name: name, NativeName: native})
	}
	sort.SliceStable(choices, func(i, j int) bool {
		if choices[i].Name == choices[j].Name {
			return choices[i].Code < choices[j].Code
		}
		return choices[i].Name < choices[j].Name
	})
	return choices
}

func standardLanguageChoice(code string) (languageChoice, bool) {
	code = strings.TrimSpace(code)
	for _, choice := range standardTranslationLanguages() {
		if strings.EqualFold(choice.Code, code) {
			return choice, true
		}
	}
	return languageChoice{}, false
}
