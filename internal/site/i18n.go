package site

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed locales/*.json
var localeFiles embed.FS

var readerLocales = loadReaderLocales()

func loadReaderLocales() map[string]map[string]string {
	entries, err := localeFiles.ReadDir("locales")
	if err != nil {
		panic(err)
	}
	locales := map[string]map[string]string{}
	for _, entry := range entries {
		data, err := localeFiles.ReadFile("locales/" + entry.Name())
		if err != nil {
			panic(err)
		}
		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			panic(fmt.Errorf("reader locale %s: %w", entry.Name(), err))
		}
		locales[strings.TrimSuffix(entry.Name(), ".json")] = messages
	}
	return locales
}

func readerMessages(language string) map[string]string {
	language = strings.ToLower(strings.ReplaceAll(language, "_", "-"))
	if messages := readerLocales[language]; messages != nil {
		return messages
	}
	switch language {
	case "zh-hans":
		language = "zh-cn"
	case "zh-hant":
		language = "zh-tw"
	}
	if messages := readerLocales[language]; messages != nil {
		return messages
	}
	if prefix, _, ok := strings.Cut(language, "-"); ok {
		return readerLocales[prefix]
	}
	return nil
}

func readerMessage(language, message string) string {
	if translated := readerMessages(language)[message]; translated != "" {
		return translated
	}
	return message
}

func (r *pageRenderer) uiText(message string) { r.text(readerMessage(r.data.Page.Language, message)) }

func (r *pageRenderer) uiTextf(message string, values ...any) {
	pairs := make([]string, 0, len(values)*2)
	for index, value := range values {
		pairs = append(pairs, fmt.Sprintf("{%d}", index), fmt.Sprint(value))
	}
	r.text(strings.NewReplacer(pairs...).Replace(readerMessage(r.data.Page.Language, message)))
}

func (r *pageRenderer) renderUIMessages() {
	if r.data.Page.Language == "en" {
		return
	}
	messages := readerMessages(r.data.Page.Language)
	if messages == nil {
		return
	}
	data, err := json.Marshal(messages)
	if err != nil {
		panic(err)
	}
	r.raw(`<script type="application/json" id="mpress-ui-messages">`)
	r.raw(string(data))
	r.raw(`</script>`)
}

const readerLocaleJS = `
const mpressUI = (() => {
  const node = document.getElementById('mpress-ui-messages');
  const messages = node ? JSON.parse(node.textContent) : {};
  return (message, ...values) => {
    const text = Object.prototype.hasOwnProperty.call(messages, message) ? messages[message] : message;
    return text.replace(/\{(\d+)\}/g, (token, index) => values[index] === undefined ? token : String(values[index]));
  };
})();
const mpressUIHTML = (message, ...values) => mpressUI(message, ...values).replace(/[&<>"']/g,
  character => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[character]));
`

func (r *pageRenderer) renderFooterCredit() {
	parts := strings.SplitN(readerMessage(r.data.Page.Language, "Built with {0}"), "{0}", 2)
	r.text(parts[0])
	r.raw(`<strong>M-Press</strong>`)
	if len(parts) == 2 {
		r.text(parts[1])
	}
}

func (r *pageRenderer) readerDate(value time.Time) string {
	if r.data.Page.Language == "en" {
		return displayDate(value)
	}
	return value.UTC().Format("2006-01-02")
}

func (r *pageRenderer) renderBlogDate(post blogPostData) {
	if r.data.Page.Language == "en" {
		r.text(post.DateDisplay)
	} else {
		r.text(post.Date)
	}
}

func (r *pageRenderer) renderVersionLabel(label string) {
	if label == "Latest" {
		r.uiText("Latest version")
	} else {
		r.text(label)
	}
}
