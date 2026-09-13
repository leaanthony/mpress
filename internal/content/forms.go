package content

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	legacycomponents "github.com/leaanthony/mpress/internal/components"
)

// Forms deliberately use a small amount of syntax inside an explicit form
// block. The surrounding document remains ordinary Markdown. A field reads
// like prose followed by a control declaration:
//
//	Site title*: [text](site.title)
//	[ ] Enable search (features.search)
//
// Keeping the field grammar scoped to @form blocks means normal Markdown links,
// task lists, and examples never acquire form semantics by accident.
var (
	formOpenRE      = regexp.MustCompile(`^@form(?:\[([^\]]*)\])?(?:\{([^}]*)\})?\s*$`)
	formFieldRE     = regexp.MustCompile(`^([^:\n]+?)(\*)?:\s*\[([A-Za-z][A-Za-z0-9_-]*)\]\(([^)]+)\)(?:\s*\{([^}]*)\})?\s*$`)
	formCheckRE     = regexp.MustCompile(`^\[([ xX])\]\s+(.+?)(?:\s+\(([^)]+)\))?\s*$`)
	formActionRE    = regexp.MustCompile(`^\[([^\]]+)\]\((submit|reset)\)\s*$`)
	formOptionRE    = regexp.MustCompile(`^\[([ xX])\]\s+(.+)$`)
	formRadioRE     = regexp.MustCompile(`^\(([ xX])\)\s+(.+)$`)
	formMetaRE      = regexp.MustCompile(`([A-Za-z][A-Za-z0-9_-]*)(?:=(?:"([^"]*)"|'([^']*)'|([^\s}]+)))?`)
	formComponentRE = regexp.MustCompile(`^@([a-z][a-z0-9-]*)(?:\[[^\]]*\])?(?:\{[^}]*\})?\s*$`)
)

// processFormBlocks lowers text-first @form blocks to accessible HTML. It is
// called before the regular component pipeline so controls can contain normal
// Markdown and other M-Press components without changing their meaning.
func (r *Renderer) processFormBlocks(file, language, source string) (string, []Diagnostic) {
	// Keep examples literal while looking for live form delimiters. The regular
	// component pipeline applies the same protection later, but forms must be
	// discovered first so their complete Markdown region can be rendered.
	protected, blocks := protectFences(source)
	lines := strings.SplitAfter(protected, "\n")
	var out strings.Builder
	var diagnostics []Diagnostic

	for i := 0; i < len(lines); {
		trimmed := strings.TrimSpace(strings.TrimSuffix(lines[i], "\n"))
		open := formOpenRE.FindStringSubmatch(trimmed)
		if open == nil {
			out.WriteString(lines[i])
			i++
			continue
		}

		start := i
		depth := 1
		end := i + 1
		for ; end < len(lines); end++ {
			candidate := strings.TrimSpace(strings.TrimSuffix(lines[end], "\n"))
			if formOpenRE.MatchString(candidate) || formContainerLine(candidate) {
				depth++
				continue
			}
			if candidate == "@end" {
				depth--
				if depth == 0 {
					break
				}
			}
		}
		if end >= len(lines) {
			diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "invalid-form", File: file, Line: start + 1, Message: "form block is missing @end"})
			out.WriteString(lines[i])
			i++
			continue
		}

		// The outer protection is only for delimiter discovery. Restore the
		// body's fences before rendering so Goldmark can keep code examples as
		// code instead of seeing an internal placeholder as prose.
		body := restoreFences(strings.Join(lines[i+1:end], ""), blocks)
		rendered, bodyDiagnostics := r.renderFormBody(file, language, body)
		diagnostics = append(diagnostics, bodyDiagnostics...)
		attrs := parseFormMeta(open[2])
		if open[1] != "" {
			attrs["action"] = strings.TrimSpace(open[1])
		}
		out.WriteString(renderForm(attrs, rendered))
		if end < len(lines)-1 {
			out.WriteByte('\n')
		}
		i = end + 1
	}
	return restoreFences(out.String(), blocks), diagnostics
}

func formContainerLine(line string) bool {
	match := formComponentRE.FindStringSubmatch(line)
	if match == nil || match[1] == "form" {
		return false
	}
	// These declarations render themselves without an @end marker.
	switch match[1] {
	case "button", "image", "badge", "input", "computed":
		return false
	}
	return legacycomponents.Registry[match[1]] != nil || isFormAdmonition(match[1])
}

func isFormAdmonition(name string) bool {
	switch name {
	case "info", "tip", "warning", "warn", "caution", "danger", "important", "bug", "example", "note":
		return true
	default:
		return false
	}
}

func (r *Renderer) renderFormBody(file, language, body string) (string, []Diagnostic) {
	// Process nested M-Press blocks first. This also protects code fences while
	// components are expanded. Field syntax is applied afterwards, with a
	// second fence protection pass so examples remain literal.
	processed, diagnostics := r.processComponents(file, language, body)
	protected, blocks := protectFences(processed)
	transformed := transformFormFields(protected)
	transformed = restoreFences(transformed, blocks)
	var rendered bytes.Buffer
	if err := r.md.Convert(stringToBytes(transformed), &rendered); err != nil {
		diagnostics = append(diagnostics, Diagnostic{Severity: "error", Code: "invalid-form", File: file, Message: fmt.Sprintf("could not render form body: %v", err)})
		return transformed, diagnostics
	}
	return rendered.String(), diagnostics
}

func parseFormMeta(raw string) map[string]string {
	attrs := map[string]string{}
	for _, match := range formMetaRE.FindAllStringSubmatch(raw, -1) {
		value := "true"
		for _, candidate := range match[2:] {
			if candidate != "" {
				value = candidate
				break
			}
		}
		attrs[match[1]] = value
	}
	return attrs
}

func renderForm(attrs map[string]string, body string) string {
	action := strings.TrimSpace(attrs["action"])
	method := strings.ToLower(strings.TrimSpace(attrs["method"]))
	if method != "get" && method != "post" {
		method = "post"
	}
	var b strings.Builder
	classes := "mpress-form"
	if custom := strings.TrimSpace(attrs["class"]); custom != "" {
		classes += " " + custom
	}
	fmt.Fprintf(&b, `<form class="%s"`, html.EscapeString(classes))
	if action != "" {
		if strings.HasPrefix(action, "/") || strings.HasPrefix(action, "#") || strings.HasPrefix(action, "http://") || strings.HasPrefix(action, "https://") {
			fmt.Fprintf(&b, ` action="%s"`, html.EscapeString(action))
		} else {
			fmt.Fprintf(&b, ` data-mpress-action="%s"`, html.EscapeString(action))
		}
	}
	fmt.Fprintf(&b, ` method="%s"`, method)
	for _, key := range []string{"id", "class", "autocomplete", "target"} {
		if value := strings.TrimSpace(attrs[key]); value != "" && key != "class" {
			fmt.Fprintf(&b, ` %s="%s"`, key, html.EscapeString(value))
		}
	}
	b.WriteString(">\n")
	b.WriteString(body)
	b.WriteString("\n</form>\n\n")
	return b.String()
}

func transformFormFields(source string) string {
	lines := strings.SplitAfter(source, "\n")
	var out strings.Builder
	usedIDs := map[string]int{}
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSuffix(lines[i], "\n")
		if match := formFieldRE.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			field := formField{label: strings.TrimSpace(match[1]), required: match[2] == "*", typ: strings.ToLower(match[3]), name: strings.TrimSpace(match[4]), attrs: parseFormMeta(match[5])}
			var optionLines []string
			j := i + 1
			if field.typ == "select" || field.typ == "radio" || field.typ == "checkboxes" || field.typ == "checklist" {
				for ; j < len(lines); j++ {
					candidate := strings.TrimSpace(strings.TrimSuffix(lines[j], "\n"))
					if !strings.HasPrefix(candidate, "-") {
						break
					}
					optionLines = append(optionLines, strings.TrimSpace(strings.TrimPrefix(candidate, "-")))
				}
			}
			out.WriteString(renderFormField(field, optionLines, usedIDs))
			i = j - 1
			continue
		}
		if match := formCheckRE.FindStringSubmatch(strings.TrimSpace(line)); match != nil && strings.TrimSpace(match[3]) != "" {
			field := formField{label: strings.TrimSpace(match[2]), typ: "checkbox", name: strings.TrimSpace(match[3]), checked: match[1] != " "}
			out.WriteString(renderFormField(field, nil, usedIDs))
			continue
		}
		if match := formActionRE.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			kind := match[2]
			class := "mpress-form-button"
			if kind == "submit" {
				class += " mpress-form-button-primary"
			}
			fmt.Fprintf(&out, "<div class=\"mpress-form-actions\"><button class=\"%s\" type=\"%s\">%s</button></div>\n\n", class, kind, html.EscapeString(strings.TrimSpace(match[1])))
			continue
		}
		out.WriteString(lines[i])
	}
	return out.String()
}

type formField struct {
	label    string
	required bool
	typ      string
	name     string
	attrs    map[string]string
	checked  bool
}

func renderFormField(field formField, rawOptions []string, usedIDs map[string]int) string {
	typ := field.typ
	if typ == "" {
		typ = "text"
	}
	if typ == "checklist" {
		typ = "checkboxes"
	}
	id := uniqueFormFieldID(formFieldID(field.name), usedIDs)
	name := html.EscapeString(field.name)
	label := html.EscapeString(field.label)
	if field.required {
		label += ` <sup class="mpress-form-required" aria-hidden="true">*</sup>`
	}
	attrs := fieldHTMLAttrs(field)
	var b strings.Builder

	switch typ {
	case "checkbox":
		fmt.Fprintf(&b, `<div class="mpress-form-check"><label><input type="checkbox" id="%s" name="%s"%s%s><span>%s</span></label></div>`, id, name, attrs, checkedAttr(field.checked), label)
		return b.String() + "\n\n"
	case "checkboxes", "radio":
		fmt.Fprintf(&b, `<fieldset class="mpress-form-choice-group"><legend>%s</legend><div class="mpress-form-choices">`, label)
		for index, raw := range rawOptions {
			option := parseFormOption(raw)
			value := option.value
			if value == "" {
				value = option.label
			}
			optionID := id + "-" + strconv.Itoa(index+1)
			inputType := "radio"
			if typ == "checkboxes" {
				inputType = "checkbox"
			}
			checked := ""
			if option.checked {
				checked = " checked"
			}
			fmt.Fprintf(&b, `<label for="%s"><input type="%s" id="%s" name="%s" value="%s"%s><span>%s</span></label>`, optionID, inputType, optionID, name, html.EscapeString(value), checked, html.EscapeString(option.label))
		}
		b.WriteString("</div></fieldset>\n\n")
		return b.String()
	case "select":
		fmt.Fprintf(&b, `<div class="mpress-form-field"><label for="%s"><span>%s</span><select id="%s" name="%s"%s>`, id, label, id, name, attrs)
		for _, raw := range rawOptions {
			option := parseFormOption(raw)
			value := option.value
			if value == "" && !option.hasValue {
				value = option.label
			}
			selected := ""
			if field.attrs["value"] != "" && field.attrs["value"] == value {
				selected = " selected"
			}
			fmt.Fprintf(&b, `<option value="%s"%s>%s</option>`, html.EscapeString(value), selected, html.EscapeString(option.label))
		}
		b.WriteString("</select></label></div>\n\n")
		return b.String()
	case "textarea":
		fmt.Fprintf(&b, `<div class="mpress-form-field"><label for="%s"><span>%s</span><textarea id="%s" name="%s"%s`, id, label, id, name, attrs)
		if rows := field.attrs["rows"]; rows != "" {
			fmt.Fprintf(&b, ` rows="%s"`, html.EscapeString(rows))
		}
		b.WriteString(">")
		b.WriteString(html.EscapeString(field.attrs["value"]))
		b.WriteString("</textarea></label></div>\n\n")
		return b.String()
	case "shortcut":
		// Keep the captured value in a native input so form submission and
		// assistive technology work without JavaScript-specific state. The
		// development client makes the explicit Change button enter capture mode.
		fmt.Fprintf(&b, `<div class="mpress-form-field mpress-shortcut-field"><label for="%s"><span>%s</span></label><span class="mpress-shortcut-control"><input type="text" id="%s" name="%s"%s data-mpress-control="shortcut" inputmode="none" autocomplete="off" spellcheck="false" aria-describedby="%s-status" readonly><button class="button mpress-shortcut-change" type="button" data-shortcut-change aria-controls="%s">Change key</button></span><small id="%s-status" class="mpress-shortcut-status" data-shortcut-status hidden></small></div>`, id, label, id, name, attrs, id, id, id)
		return b.String() + "\n\n"
	case "hidden":
		return fmt.Sprintf("<input type=\"hidden\" id=\"%s\" name=\"%s\" value=\"%s\">\n\n", id, name, html.EscapeString(field.attrs["value"]))
	default:
		if !validFormInputType(typ) {
			typ = "text"
		}
		fmt.Fprintf(&b, "<div class=\"mpress-form-field\"><label for=\"%s\"><span>%s</span><input type=\"%s\" id=\"%s\" name=\"%s\"%s></label></div>\n\n", id, label, html.EscapeString(typ), id, name, attrs)
		return b.String()
	}
}

func validFormInputType(typ string) bool {
	switch typ {
	case "text", "url", "email", "number", "date", "datetime-local", "month", "time", "week", "color", "password", "search", "file", "range", "tel":
		return true
	default:
		return false
	}
}

func fieldHTMLAttrs(field formField) string {
	var b strings.Builder
	for _, key := range []string{"placeholder", "value", "min", "max", "step", "minlength", "maxlength", "pattern", "autocomplete", "accept"} {
		if value := strings.TrimSpace(field.attrs[key]); value != "" && key != "value" {
			fmt.Fprintf(&b, ` %s="%s"`, key, html.EscapeString(value))
		}
	}
	if value := strings.TrimSpace(field.attrs["value"]); value != "" {
		fmt.Fprintf(&b, ` value="%s"`, html.EscapeString(value))
	}
	if field.required {
		b.WriteString(" required")
	}
	if field.attrs["disabled"] == "true" {
		b.WriteString(" disabled")
	}
	if field.attrs["readonly"] == "true" {
		b.WriteString(" readonly")
	}
	return b.String()
}

func checkedAttr(checked bool) string {
	if checked {
		return " checked"
	}
	return ""
}

type formOption struct {
	label    string
	value    string
	hasValue bool
	checked  bool
}

func parseFormOption(raw string) formOption {
	raw = strings.TrimSpace(raw)
	option := formOption{}
	if match := formOptionRE.FindStringSubmatch(raw); match != nil {
		option.checked = strings.EqualFold(match[1], "x")
		raw = strings.TrimSpace(match[2])
	}
	if match := formRadioRE.FindStringSubmatch(raw); match != nil {
		option.checked = strings.EqualFold(match[1], "x")
		raw = strings.TrimSpace(match[2])
	}
	if start := strings.LastIndex(raw, "{value="); start >= 0 && strings.HasSuffix(raw, "}") {
		option.label = strings.TrimSpace(raw[:start])
		option.value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(raw[start+len("{value="):], " "), "}"))
		option.hasValue = true
		return option
	}
	if parts := strings.SplitN(raw, "=", 2); len(parts) == 2 {
		option.label = strings.TrimSpace(parts[0])
		option.value = strings.TrimSpace(parts[1])
		option.hasValue = true
		return option
	}
	option.label = strings.TrimSpace(raw)
	return option
}

func formFieldID(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
			b.WriteByte('-')
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		return "field"
	}
	return id
}

func uniqueFormFieldID(base string, used map[string]int) string {
	used[base]++
	if used[base] == 1 {
		return base
	}
	return base + "-" + strconv.Itoa(used[base])
}
