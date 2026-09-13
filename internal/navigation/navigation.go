package navigation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leaanthony/mpress/internal/content"
	"gopkg.in/yaml.v3"
)

type Item struct {
	Label        string            `yaml:"label" json:"label"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Link         string            `yaml:"link,omitempty" json:"link,omitempty"`
	Collapsed    bool              `yaml:"collapsed,omitempty" json:"collapsed,omitempty"`
	Items        []Item            `yaml:"items,omitempty" json:"items,omitempty"`
	Autogenerate *struct {
		Directory string `yaml:"directory"`
	} `yaml:"autogenerate,omitempty" json:"-"`
}

func (item Item) LabelFor(language string) (string, bool) {
	if label := strings.TrimSpace(item.Labels[language]); label != "" {
		return label, true
	}
	return item.Label, false
}

func Load(path string, pages []*content.Page) ([]Item, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Generate(pages), nil
		}
		return nil, err
	}
	var items []Item
	if err := yaml.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse navigation %s: %w", path, err)
	}
	return expand(items, pages), nil
}

func Generate(pages []*content.Page) []Item {
	type group struct {
		label string
		pages []*content.Page
	}
	groups := map[string]*group{}
	var roots []*content.Page
	for _, p := range pages {
		parts := strings.Split(p.URLPath, "/")
		if len(parts) == 1 {
			roots = append(roots, p)
			continue
		}
		key := parts[0]
		if groups[key] == nil {
			groups[key] = &group{label: title(key)}
		}
		groups[key].pages = append(groups[key].pages, p)
	}
	sortPages(roots)
	var result []Item
	for _, p := range roots {
		result = append(result, Item{Label: p.Title, Link: "/" + p.URLPath})
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g := groups[k]
		sortPages(g.pages)
		item := Item{Label: g.label, Collapsed: true}
		for _, p := range g.pages {
			item.Items = append(item.Items, Item{Label: p.Title, Link: "/" + p.URLPath})
		}
		result = append(result, item)
	}
	return result
}

func expand(items []Item, pages []*content.Page) []Item {
	for i := range items {
		if items[i].Autogenerate != nil {
			dir := strings.Trim(items[i].Autogenerate.Directory, "/")
			for _, p := range pages {
				if strings.HasPrefix(p.URLPath, dir+"/") {
					items[i].Items = append(items[i].Items, Item{Label: p.Title, Link: "/" + p.URLPath})
				}
			}
			sort.SliceStable(items[i].Items, func(a, b int) bool { return items[i].Items[a].Label < items[i].Items[b].Label })
		}
		items[i].Items = expand(items[i].Items, pages)
	}
	return items
}
func sortPages(p []*content.Page) {
	sort.SliceStable(p, func(i, j int) bool {
		if p[i].Order != p[j].Order {
			return p[i].Order < p[j].Order
		}
		return p[i].SourcePath < p[j].SourcePath
	})
}
func title(s string) string {
	s = strings.ReplaceAll(filepath.Base(s), "-", " ")
	return strings.Title(s)
}

func Links(items []Item) []string {
	var out []string
	var walk func([]Item)
	walk = func(xs []Item) {
		for _, x := range xs {
			if x.Link != "" {
				out = append(out, x.Link)
			}
			walk(x.Items)
		}
	}
	walk(items)
	return out
}
