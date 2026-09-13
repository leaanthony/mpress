package dev

import (
	_ "embed"
	"strings"

	"github.com/leaanthony/mpress/internal/icons"
)

//go:embed assets/devbar.html
var devbarHTMLTemplate string

//go:embed assets/devbar.css
var devbarCSS string

//go:embed assets/devbar.js
var devbarJS string

//go:embed assets/mpress-mark.svg
var mpressMark string

func workspaceHTML(template string) string {
	replacer := strings.NewReplacer(
		"{{icon:eye}}", icons.Lucide("eye", 17),
		"{{icon:save}}", icons.Lucide("save", 17),
		"{{icon:more}}", icons.Lucide("more-vertical", 18),
		"{{icon:chevron-down}}", icons.Lucide("chevron-down", 15),
		"{{icon:chevrons-left}}", icons.Lucide("chevrons-left", 16),
		"{{icon:upload}}", icons.Lucide("upload", 19),
		"{{icon:zap}}", icons.Lucide("zap", 17),
		"{{icon:check-circle}}", icons.Lucide("check-circle", 17),
		"{{icon:play}}", icons.Lucide("play", 16),
		"{{icon:pencil}}", icons.Lucide("pencil", 16),
		"{{icon:file}}", icons.Lucide("file-text", 16),
		"{{icon:book-open}}", icons.Lucide("book-open", 16),
		"{{icon:folder}}", icons.Lucide("folder", 16),
		"{{icon:chevron-right}}", icons.Lucide("chevron-right", 15),
		"{{icon:x}}", icons.Lucide("x", 15),
		"{{icon:panel-left}}", icons.Lucide("panel-left", 18),
		"{{icon:columns}}", icons.Lucide("columns-3", 18),
		"{{icon:rows}}", icons.Lucide("rows-3", 18),
	)
	return replacer.Replace(template)
}

func devbarHTML() string {
	replacer := strings.NewReplacer(
		"{{icon:chevron-up}}", icons.Lucide("chevron-up", 14),
		"{{icon:chevron-right}}", icons.Lucide("chevron-right", 14),
		"{{icon:check-circle}}", icons.Lucide("check-circle", 16),
		"{{icon:pencil}}", icons.Lucide("pencil", 16),
		"{{icon:arrow-left}}", icons.Lucide("arrow-left", 16),
		"{{icon:star}}", icons.Lucide("star", 16),
		"{{icon:globe}}", icons.Lucide("globe-2", 16),
		"{{icon:image}}", icons.Lucide("image", 16),
		"{{icon:link}}", icons.Lucide("link", 16),
		"{{icon:bold}}", icons.Lucide("bold", 16),
		"{{icon:italic}}", icons.Lucide("italic", 16),
		"{{icon:code}}", icons.Lucide("code", 16),
		"{{icon:code-xml}}", icons.Lucide("code-xml", 16),
		"{{icon:file-text}}", icons.Lucide("file-text", 16),
		"{{icon:list}}", icons.Lucide("list", 16),
		"{{icon:list-ordered}}", icons.Lucide("list-ordered", 16),
		"{{icon:quote}}", icons.Lucide("quote", 16),
		"{{icon:grip-vertical}}", icons.Lucide("grip-vertical", 16),
		"{{icon:chevron-down}}", icons.Lucide("chevron-down", 15),
		"{{icon:file}}", icons.Lucide("file-text", 16),
		"{{icon:book-open}}", icons.Lucide("book-open", 16),
		"{{icon:sun}}", icons.Lucide("sun", 16),
		"{{icon:moon}}", icons.Lucide("moon", 16),
		"{{icon:accessibility}}", icons.Lucide("circle-user-round", 16),
		"{{icon:layout}}", icons.Lucide("columns-3", 16),
		"{{icon:versions}}", icons.Lucide("git-branch", 16),
		"{{icon:eye}}", icons.Lucide("eye", 16),
		"{{icon:settings}}", icons.Lucide("settings", 16),
		"{{icon:gauge}}", icons.Lucide("gauge", 16),
		"{{icon:languages}}", icons.Lucide("languages", 16),
		"{{icon:rocket}}", icons.Lucide("rocket", 16),
		"{{icon:package}}", icons.Lucide("package", 16),
		"{{icon:smartphone}}", icons.Lucide("smartphone", 15),
		"{{icon:tablet}}", icons.Lucide("tablet", 15),
		"{{icon:monitor}}", icons.Lucide("monitor", 15),
		"{{icon:zap}}", icons.Lucide("zap", 16),
		"{{icon:check}}", icons.Lucide("check", 17),
		"{{icon:x}}", icons.Lucide("x", 17),
		"{{brand:cloudflare}}", icons.Starlight("cloudflare", 26),
		"{{brand:netlify}}", icons.Starlight("netlify", 26),
		"{{brand:mpress-mark}}", mpressMark,
	)
	return replacer.Replace(devbarHTMLTemplate)
}
