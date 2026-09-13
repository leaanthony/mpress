package components

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Table renders an optionally searchable, filterable, and sortable data table.
// Each feature is opt-in so ordinary documentation tables remain static.
type Table struct {
	Meta    map[string]string
	Content string
	Rows    [][]string
}

func (t *Table) Parse(content string) error {
	t.Content = content
	t.Rows = nil
	width := 0
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		row, err := parseDataTableRow(line)
		if err != nil {
			return err
		}
		if width == 0 {
			width = len(row)
		}
		if len(row) != width {
			return fmt.Errorf("table row has %d cells; expected %d", len(row), width)
		}
		t.Rows = append(t.Rows, row)
	}
	if len(t.Rows) == 0 {
		return fmt.Errorf("table requires at least one row")
	}
	return nil
}

func (t *Table) Render() (string, error) {
	header := metadataBool(t.Meta, "header", true)
	searchable := metadataBool(t.Meta, "search", false)
	filterable := metadataBool(t.Meta, "filter", false)
	sortable := metadataBool(t.Meta, "sort", false)
	paginated := metadataBool(t.Meta, "paginate", false)
	columnSeparators := metadataBool(t.Meta, "column-separators", false)
	pageSize := 10
	if configured, err := strconv.Atoi(strings.TrimSpace(t.Meta["page-size"])); err == nil && configured > 0 {
		pageSize = configured
	}
	headerRows := 0
	if header {
		headerRows = 1
	}
	dataRows := t.Rows[headerRows:]
	id := tableID(t.Content, t.Meta)
	alignments := tableAlignments(t.Meta["align"], len(t.Rows[0]))

	var out strings.Builder
	classes := "mpress-data-table"
	if columnSeparators {
		classes += " mpress-table-column-separators"
	}
	fmt.Fprintf(&out, `<div class="%s" data-table-search="%t" data-table-filter="%t" data-table-sort="%t" data-table-paginate="%t" data-table-column-separators="%t" data-table-page-size="%d">`, classes, searchable, filterable, sortable, paginated, columnSeparators, pageSize)
	if searchable {
		out.WriteString(`<div class="mpress-table-toolbar">`)
		fmt.Fprintf(&out, `<label class="mpress-table-search"><span class="sr-only">Search table</span>%s<input type="search" placeholder="Search table" autocomplete="off" data-table-query aria-controls="%s"></label>`, lucide("search", 16), id)
		out.WriteString(`</div>`)
	}

	out.WriteString(`<div class="mpress-table-scroll">`)
	fmt.Fprintf(&out, `<table id="%s">`, id)
	if header {
		out.WriteString(`<thead><tr>`)
		for column, cell := range t.Rows[0] {
			label := plainInlineText(cell)
			fmt.Fprintf(&out, `<th scope="col"%s`, tableAlignAttribute(alignments[column]))
			if sortable {
				out.WriteString(` aria-sort="none">`)
			} else {
				out.WriteByte('>')
			}
			out.WriteString(`<div class="mpress-table-heading">`)
			if sortable {
				fmt.Fprintf(&out, `<button class="mpress-table-sort" type="button" data-table-sort-column="%d"><span>%s</span>%s</button>`, column, renderInlineMarkdown(cell), lucide("arrow-up-down", 14))
			} else {
				fmt.Fprintf(&out, `<span class="mpress-table-heading-label">%s</span>`, renderInlineMarkdown(cell))
			}
			if filterable {
				menuID := fmt.Sprintf("%s-filter-%d", id, column)
				fmt.Fprintf(&out, `<button class="mpress-table-filter-trigger utility-menu-trigger" type="button" popovertarget="%s" aria-label="Filter %s" aria-haspopup="menu" aria-expanded="false">%s</button>`, menuID, html.EscapeString(label), lucide("list-filter", 14))
				fmt.Fprintf(&out, `<menu id="%s" class="utility-menu-panel mpress-table-filter-menu" popover data-utility-menu data-table-filter-column="%d" data-table-filter-label="%s" data-table-filter-value=""><li><button type="button" role="menuitem" data-table-filter-value="" aria-current="true">All</button></li>`, menuID, column, html.EscapeString(label))
				for _, value := range distinctColumnValues(dataRows, column) {
					fmt.Fprintf(&out, `<li><button type="button" role="menuitem" data-table-filter-value="%s">%s</button></li>`, html.EscapeString(strings.ToLower(value)), html.EscapeString(value))
				}
				out.WriteString(`</menu>`)
			}
			out.WriteString(`</div></th>`)
		}
		out.WriteString(`</tr></thead>`)
	}
	out.WriteString(`<tbody>`)
	for index, row := range dataRows {
		fmt.Fprintf(&out, `<tr data-table-row data-table-index="%d">`, index)
		for column, cell := range row {
			fmt.Fprintf(&out, `<td%s data-table-value="%s">%s</td>`, tableAlignAttribute(alignments[column]), html.EscapeString(strings.ToLower(plainInlineText(cell))), renderInlineMarkdown(cell))
		}
		out.WriteString(`</tr>`)
	}
	out.WriteString(`</tbody></table></div>`)
	if searchable || filterable || paginated {
		out.WriteString(`<div class="mpress-table-footer">`)
		fmt.Fprintf(&out, `<p class="mpress-table-status" role="status" aria-live="polite">%d rows</p>`, len(dataRows))
		if paginated {
			pages := (len(dataRows) + pageSize - 1) / pageSize
			if pages < 1 {
				pages = 1
			}
			fmt.Fprintf(&out, `<nav class="mpress-table-pagination" aria-label="Table pages"><button type="button" data-table-page-prev aria-label="Previous page" disabled>%s</button><span data-table-page-status>Page 1 of %d</span><button type="button" data-table-page-next aria-label="Next page"%s>%s</button></nav>`, lucide("chevron-left", 16), pages, disabledAttribute(pages <= 1), lucide("chevron-right", 16))
		}
		out.WriteString(`</div>`)
	}
	out.WriteString(`</div>`)
	return out.String(), nil
}

func disabledAttribute(disabled bool) string {
	if disabled {
		return " disabled"
	}
	return ""
}

func parseDataTableRow(line string) ([]string, error) {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '|' || line[len(line)-1] != '|' {
		return nil, fmt.Errorf("table rows must begin and end with |")
	}
	line = line[1 : len(line)-1]
	var cells []string
	var cell strings.Builder
	for index := 0; index < len(line); index++ {
		if line[index] == '\\' && index+1 < len(line) && line[index+1] == '|' {
			cell.WriteByte('|')
			index++
			continue
		}
		if line[index] == '|' {
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteByte(line[index])
	}
	cells = append(cells, strings.TrimSpace(cell.String()))
	return cells, nil
}

func metadataBool(meta map[string]string, key string, fallback bool) bool {
	value, ok := meta[key]
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func tableID(content string, meta map[string]string) string {
	keys := make([]string, 0, len(meta))
	for key := range meta {
		if !strings.HasPrefix(key, "__") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var seed strings.Builder
	seed.WriteString(content)
	for _, key := range keys {
		seed.WriteByte(0)
		seed.WriteString(key)
		seed.WriteByte('=')
		seed.WriteString(meta[key])
	}
	digest := sha256.Sum256([]byte(seed.String()))
	return "mpress-table-" + hex.EncodeToString(digest[:6])
}

func tableAlignments(raw string, width int) []string {
	result := make([]string, width)
	raw = strings.TrimSpace(strings.Trim(raw, "[]"))
	parts := strings.Split(raw, ",")
	for index := 0; index < width && index < len(parts); index++ {
		value := strings.ToLower(strings.Trim(strings.TrimSpace(parts[index]), `"'`))
		if value == "centre" {
			value = "center"
		}
		switch value {
		case "left", "center", "right":
			result[index] = value
		}
	}
	return result
}

func tableAlignAttribute(value string) string {
	if value == "" {
		return ""
	}
	return ` style="text-align:` + value + `"`
}

func distinctColumnValues(rows [][]string, column int) []string {
	seen := make(map[string]string)
	for _, row := range rows {
		value := plainInlineText(row[column])
		key := strings.ToLower(value)
		if value != "" {
			seen[key] = value
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := seen[keys[i]]
		right := seen[keys[j]]
		leftNumber, leftErr := strconv.ParseFloat(left, 64)
		rightNumber, rightErr := strconv.ParseFloat(right, 64)
		if leftErr == nil && rightErr == nil {
			return leftNumber < rightNumber
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, seen[key])
	}
	return values
}

var inlineHTMLTagRE = regexp.MustCompile(`<[^>]+>`)

func plainInlineText(source string) string {
	rendered := renderInlineMarkdown(source)
	return strings.TrimSpace(html.UnescapeString(inlineHTMLTagRE.ReplaceAllString(rendered, "")))
}
