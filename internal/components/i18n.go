package components

import (
	"fmt"
	"strings"
)

// ActiveLanguage is the current build language. Set by the build pipeline
// before processing content for each language.
var ActiveLanguage string

// componentTranslations maps language → key → translated string.
// Keys are component label identifiers used internally.
// Plural keys use ".one" and ".other" suffixes (e.g., "results.one", "results.other").
// Variable interpolation uses {varName} placeholders in translation strings.
var componentTranslations = map[string]map[string]string{
	"en": {
		"note.info":      "Info",
		"note.caution":   "Caution",
		"note.warning":   "Warning",
		"note.tip":       "Tip",
		"note.danger":    "Danger",
		"note.important": "Important",
		"details.title":  "Details",
		"step.prefix":    "Step",
		// Plural forms
		"results.one":   "{count} result",
		"results.other": "{count} results",
		"pages.one":     "{count} page",
		"pages.other":   "{count} pages",
		"minutes.one":   "{count} min read",
		"minutes.other": "{count} min read",
		"items.one":     "{count} item",
		"items.other":   "{count} items",
		// Variable interpolation
		"search.results_for": "Results for \"{query}\"",
		"page.last_updated":  "Last updated: {date}",
		"page.by_author":     "By {author}",
	},
	"es": {
		"note.info":          "Nota",
		"note.caution":       "Precaución",
		"note.warning":       "Advertencia",
		"note.tip":           "Consejo",
		"note.danger":        "Peligro",
		"note.important":     "Importante",
		"details.title":      "Detalles",
		"step.prefix":        "Paso",
		"results.one":        "{count} resultado",
		"results.other":      "{count} resultados",
		"pages.one":          "{count} página",
		"pages.other":        "{count} páginas",
		"minutes.one":        "{count} min de lectura",
		"minutes.other":      "{count} min de lectura",
		"items.one":          "{count} elemento",
		"items.other":        "{count} elementos",
		"search.results_for": "Resultados para \"{query}\"",
		"page.last_updated":  "Última actualización: {date}",
		"page.by_author":     "Por {author}",
	},
	"fr": {
		"note.info":          "Info",
		"note.caution":       "Attention",
		"note.warning":       "Avertissement",
		"note.tip":           "Astuce",
		"note.danger":        "Danger",
		"note.important":     "Important",
		"details.title":      "Détails",
		"step.prefix":        "Étape",
		"results.one":        "{count} résultat",
		"results.other":      "{count} résultats",
		"pages.one":          "{count} page",
		"pages.other":        "{count} pages",
		"minutes.one":        "{count} min de lecture",
		"minutes.other":      "{count} min de lecture",
		"items.one":          "{count} élément",
		"items.other":        "{count} éléments",
		"search.results_for": "Résultats pour \"{query}\"",
		"page.last_updated":  "Dernière mise à jour : {date}",
		"page.by_author":     "Par {author}",
	},
	"de": {
		"note.info":          "Info",
		"note.caution":       "Achtung",
		"note.warning":       "Warnung",
		"note.tip":           "Tipp",
		"note.danger":        "Gefahr",
		"note.important":     "Wichtig",
		"details.title":      "Details",
		"step.prefix":        "Schritt",
		"results.one":        "{count} Ergebnis",
		"results.other":      "{count} Ergebnisse",
		"pages.one":          "{count} Seite",
		"pages.other":        "{count} Seiten",
		"minutes.one":        "{count} Min. Lesezeit",
		"minutes.other":      "{count} Min. Lesezeit",
		"items.one":          "{count} Element",
		"items.other":        "{count} Elemente",
		"search.results_for": "Ergebnisse für \"{query}\"",
		"page.last_updated":  "Zuletzt aktualisiert: {date}",
		"page.by_author":     "Von {author}",
	},
	"ja": {
		"note.info":          "情報",
		"note.caution":       "注意",
		"note.warning":       "警告",
		"note.tip":           "ヒント",
		"note.danger":        "危険",
		"note.important":     "重要",
		"details.title":      "詳細",
		"step.prefix":        "ステップ",
		"results.one":        "{count}件の結果",
		"results.other":      "{count}件の結果",
		"pages.one":          "{count}ページ",
		"pages.other":        "{count}ページ",
		"minutes.one":        "{count}分で読めます",
		"minutes.other":      "{count}分で読めます",
		"items.one":          "{count}件",
		"items.other":        "{count}件",
		"search.results_for": "「{query}」の検索結果",
		"page.last_updated":  "最終更新日: {date}",
		"page.by_author":     "{author}著",
	},
	"zh": {
		"note.info":          "信息",
		"note.caution":       "注意",
		"note.warning":       "警告",
		"note.tip":           "提示",
		"note.danger":        "危险",
		"note.important":     "重要",
		"details.title":      "详情",
		"step.prefix":        "步骤",
		"results.one":        "{count}个结果",
		"results.other":      "{count}个结果",
		"pages.one":          "{count}页",
		"pages.other":        "{count}页",
		"minutes.one":        "{count}分钟阅读",
		"minutes.other":      "{count}分钟阅读",
		"items.one":          "{count}项",
		"items.other":        "{count}项",
		"search.results_for": "\"{query}\" 的搜索结果",
		"page.last_updated":  "最后更新: {date}",
		"page.by_author":     "作者: {author}",
	},
	"ko": {
		"note.info":          "정보",
		"note.caution":       "주의",
		"note.warning":       "경고",
		"note.tip":           "팁",
		"note.danger":        "위험",
		"note.important":     "중요",
		"details.title":      "세부사항",
		"step.prefix":        "단계",
		"results.one":        "{count}개의 결과",
		"results.other":      "{count}개의 결과",
		"pages.one":          "{count}페이지",
		"pages.other":        "{count}페이지",
		"minutes.one":        "{count}분 소요",
		"minutes.other":      "{count}분 소요",
		"items.one":          "{count}개 항목",
		"items.other":        "{count}개 항목",
		"search.results_for": "\"{query}\" 검색 결과",
		"page.last_updated":  "마지막 업데이트: {date}",
		"page.by_author":     "{author} 작성",
	},
	"pt": {
		"note.info":          "Nota",
		"note.caution":       "Atenção",
		"note.warning":       "Aviso",
		"note.tip":           "Dica",
		"note.danger":        "Perigo",
		"note.important":     "Importante",
		"details.title":      "Detalhes",
		"step.prefix":        "Passo",
		"results.one":        "{count} resultado",
		"results.other":      "{count} resultados",
		"pages.one":          "{count} página",
		"pages.other":        "{count} páginas",
		"minutes.one":        "{count} min de leitura",
		"minutes.other":      "{count} min de leitura",
		"items.one":          "{count} item",
		"items.other":        "{count} itens",
		"search.results_for": "Resultados para \"{query}\"",
		"page.last_updated":  "Última atualização: {date}",
		"page.by_author":     "Por {author}",
	},
	"it": {
		"note.info":          "Info",
		"note.caution":       "Attenzione",
		"note.warning":       "Attenzione",
		"note.tip":           "Suggerimento",
		"note.danger":        "Pericolo",
		"note.important":     "Importante",
		"details.title":      "Dettagli",
		"step.prefix":        "Passo",
		"results.one":        "{count} risultato",
		"results.other":      "{count} risultati",
		"pages.one":          "{count} pagina",
		"pages.other":        "{count} pagine",
		"minutes.one":        "{count} min di lettura",
		"minutes.other":      "{count} min di lettura",
		"items.one":          "{count} elemento",
		"items.other":        "{count} elementi",
		"search.results_for": "Risultati per \"{query}\"",
		"page.last_updated":  "Ultimo aggiornamento: {date}",
		"page.by_author":     "Di {author}",
	},
	"cy": {
		"note.info":          "Gwybodaeth",
		"note.caution":       "Rhybudd",
		"note.warning":       "Rhybudd",
		"note.tip":           "Awgrym",
		"note.danger":        "Perygl",
		"note.important":     "Pwysig",
		"details.title":      "Manylion",
		"step.prefix":        "Cam",
		"results.one":        "{count} canlyniad",
		"results.other":      "{count} canlyniad",
		"pages.one":          "{count} tudalen",
		"pages.other":        "{count} tudalen",
		"minutes.one":        "{count} mun i ddarllen",
		"minutes.other":      "{count} mun i ddarllen",
		"items.one":          "{count} eitem",
		"items.other":        "{count} eitem",
		"search.results_for": "Canlyniadau ar gyfer \"{query}\"",
		"page.last_updated":  "Diweddarwyd ddiwethaf: {date}",
		"page.by_author":     "Gan {author}",
	},
}

// T returns the translated string for the given key in the active language.
// Falls back to English if the language or key is not found.
func T(key string) string {
	return TForLanguage(ActiveLanguage, key)
}

// TForLanguage returns the translated string for key in language.
func TForLanguage(language, key string) string {
	if lang, ok := componentTranslations[language]; ok {
		if val, ok := lang[key]; ok {
			return val
		}
	}
	// Fallback to English
	if en, ok := componentTranslations["en"]; ok {
		if val, ok := en[key]; ok {
			return val
		}
	}
	return key
}

// TVar returns the translated string for the given key with variable interpolation.
// Variables in the translation string use {varName} placeholders, which are replaced
// with values from the vars map. Falls back to English if the language or key is not found.
// Unknown placeholders are left as-is.
func TVar(key string, vars map[string]string) string {
	tmpl := T(key)
	return interpolate(tmpl, vars)
}

// TPlural returns the pluralized translated string for the given key and count.
// It looks up "key.one" when count == 1, and "key.other" otherwise.
// The {count} variable is automatically injected into the translation string.
// Falls back to English if the language or key is not found.
func TPlural(key string, count int) string {
	suffix := ".other"
	if count == 1 {
		suffix = ".one"
	}
	tmpl := T(key + suffix)
	return interpolate(tmpl, map[string]string{
		"count": fmt.Sprintf("%d", count),
	})
}

// interpolate replaces {varName} placeholders in s with values from vars.
func interpolate(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}
