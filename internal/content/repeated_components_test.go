package content

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"

	"golang.org/x/net/html"
)

const repeatedInteractivePage = `---
title: Repeated components
---

# Repeated components

:::tabs{sync-key=os alpha="one" zeta="last"}
[One]

Same tab content.

[Two]

Same tab content again.
:::

:::tabs{sync-key=os alpha="one" zeta="last"}
[One]

Same tab content.

[Two]

Same tab content again.
:::

:::carousel{label="Demo"}
First slide.
---
Second slide.
:::

:::carousel{label="Demo"}
First slide.
---
Second slide.
:::

:::table{header=true search=true filter=true sort=true}
| Name | Value |
| A | 1 |
| B | 2 |
:::

:::table{header=true search=true filter=true sort=true}
| Name | Value |
| A | 1 |
| B | 2 |
:::

:::preview-tabs
[One]

Same preview content.

[Two]

Same preview content again.
:::

:::preview-tabs
[One]

Same preview content.

[Two]

Same preview content again.
:::

:::file-tabs
[first.txt]

Same file content.
:::

:::file-tabs
[first.txt]

Same file content.
:::

:::api-playground{method="GET" path="/health" baseUrl="http://127.0.0.1:1"}
Fixture-only playground.
:::

:::api-playground{method="GET" path="/health" baseUrl="http://127.0.0.1:1"}
Fixture-only playground.
:::
`

func parsePageMarkup(t *testing.T, markup string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(markup))
	if err != nil {
		t.Fatalf("parse rendered markup: %v", err)
	}
	return doc
}

func walkHTML(node *html.Node, visit func(*html.Node)) {
	if node.Type == html.ElementNode {
		visit(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkHTML(child, visit)
	}
}

func htmlAttribute(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func htmlNodes(doc *html.Node, tag, class string) []*html.Node {
	var matches []*html.Node
	walkHTML(doc, func(node *html.Node) {
		if node.Data != tag {
			return
		}
		if class != "" && !hasHTMLClass(node, class) {
			return
		}
		matches = append(matches, node)
	})
	return matches
}

func hasHTMLClass(node *html.Node, class string) bool {
	for _, candidate := range strings.Fields(htmlAttribute(node, "class")) {
		if candidate == class {
			return true
		}
	}
	return false
}

func duplicateHTMLIDs(doc *html.Node) []string {
	counts := map[string]int{}
	walkHTML(doc, func(node *html.Node) {
		if id := htmlAttribute(node, "id"); id != "" {
			counts[id]++
		}
	})
	var duplicates []string
	for id, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, fmt.Sprintf("%s (%d)", id, count))
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

func TestRepeatedInteractiveComponentsUseUniqueIDs(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("repeated.md", "en", repeatedInteractivePage)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	doc := parsePageMarkup(t, page.HTML)
	if duplicates := duplicateHTMLIDs(doc); len(duplicates) != 0 {
		t.Fatalf("repeated components rendered duplicate IDs: %s\n%s", strings.Join(duplicates, ", "), page.HTML)
	}
	for _, tag := range []string{"ot-tabs", "section", "table", "div"} {
		for _, node := range htmlNodes(doc, tag, "") {
			switch {
			case tag == "ot-tabs" && hasHTMLClass(node, "mpress-tabs"):
			case tag == "section" && hasHTMLClass(node, "mpress-carousel"):
			case tag == "table":
			case tag == "div" && hasHTMLClass(node, "mpress-api-playground"):
			default:
				continue
			}
			if htmlAttribute(node, "id") == "" {
				t.Fatalf("repeated interactive component <%s> has no ID: %s", tag, page.HTML)
			}
		}
	}
}

func TestRepeatedTabsKeepLocalAriaRelationships(t *testing.T) {
	page, diagnostics, err := NewRenderer().Parse("repeated-tabs.md", "en", repeatedInteractivePage)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	doc := parsePageMarkup(t, page.HTML)
	groups := htmlNodes(doc, "ot-tabs", "mpress-tabs")
	// Two generated tabs groups plus two file-tabs groups, which share the tab
	// controller and therefore the same local reference contract.
	if len(groups) != 4 {
		t.Fatalf("rendered %d tab groups, want 4", len(groups))
	}
	seenTabs := map[string]bool{}
	seenPanels := map[string]bool{}
	for _, group := range groups {
		root := htmlAttribute(group, "id")
		if root == "" {
			t.Fatal("tab group has no id")
		}
		var tabs []*html.Node
		var panels []*html.Node
		walkHTML(group, func(node *html.Node) {
			if htmlAttribute(node, "role") == "tab" {
				tabs = append(tabs, node)
			}
			if htmlAttribute(node, "role") == "tabpanel" {
				panels = append(panels, node)
			}
		})
		if len(tabs) == 0 || len(tabs) != len(panels) {
			t.Fatalf("tab group %q rendered %d tabs and %d panels", root, len(tabs), len(panels))
		}
		if !strings.HasPrefix(root, "mpress-tabs-") {
			t.Fatalf("tab group id %q is not a generated tab ID", root)
		}
		panelsByID := map[string]*html.Node{}
		for _, panel := range panels {
			panelsByID[htmlAttribute(panel, "id")] = panel
		}
		for _, tab := range tabs {
			tabID := htmlAttribute(tab, "id")
			if tabID == "" || seenTabs[tabID] {
				t.Fatalf("tab id %q is empty or reused: %s", tabID, page.HTML)
			}
			seenTabs[tabID] = true
			controls := htmlAttribute(tab, "aria-controls")
			panel := panelsByID[controls]
			if panel == nil {
				t.Fatalf("tab %q controls %q, which is not a panel in the same group: %s", tabID, controls, page.HTML)
			}
			if labelledBy := htmlAttribute(panel, "aria-labelledby"); labelledBy != tabID {
				t.Fatalf("panel %q is labelled by %q, want %q", controls, labelledBy, tabID)
			}
			if seenPanels[controls] {
				t.Fatalf("panel id %q is reused", controls)
			}
			seenPanels[controls] = true
			if !strings.HasPrefix(controls, root+"-panel-") {
				t.Fatalf("panel %q is not local to tab group %q", controls, root)
			}
		}
	}
}

func TestRepeatedNestedFragmentsUseUniqueIDs(t *testing.T) {
	source := `---
title: Nested
---

# Nested

:::tabs
[Alpha]

:::table{header=true search=true filter=true}
| Name | Value |
| A | 1 |
:::

[Beta]

Beta body.
:::

:::tabs
[Alpha]

:::table{header=true search=true filter=true}
| Name | Value |
| A | 1 |
:::

[Beta]

Beta body.
:::
`
	page, diagnostics, err := NewRenderer().Parse("nested.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	doc := parsePageMarkup(t, page.HTML)
	if duplicates := duplicateHTMLIDs(doc); len(duplicates) != 0 {
		t.Fatalf("nested repeated components rendered duplicate IDs: %s\n%s", strings.Join(duplicates, ", "), page.HTML)
	}
	if tables := htmlNodes(doc, "table", ""); len(tables) != 2 {
		t.Fatalf("rendered %d nested tables, want 2: %s", len(tables), page.HTML)
	}
}

func TestAuthoredComponentIDsArePreservedWithDiagnostics(t *testing.T) {
	source := `---
title: Authored IDs
---

# Authored IDs

:::tabs{id="shared-tabs"}
[One]

Same content.
:::

:::tabs{id="shared-tabs"}
[One]

Same content.
:::
`
	page, diagnostics, err := NewRenderer().Parse("authored.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(page.HTML, `id="shared-tabs"`) != 2 {
		t.Fatalf("authored tab IDs were rewritten:\n%s", page.HTML)
	}
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "duplicate-id" && strings.Contains(diagnostic.Message, "shared-tabs") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing duplicate-ID diagnostic: %#v", diagnostics)
	}
	doc := parsePageMarkup(t, page.HTML)
	seenTabs := map[string]bool{}
	seenPanels := map[string]bool{}
	walkHTML(doc, func(node *html.Node) {
		switch htmlAttribute(node, "role") {
		case "tab":
			if id := htmlAttribute(node, "id"); id != "" {
				if seenTabs[id] {
					t.Fatalf("tab id %q was reused despite the authored root duplicate:\n%s", id, page.HTML)
				}
				seenTabs[id] = true
			}
		case "tabpanel":
			if id := htmlAttribute(node, "id"); id != "" {
				if seenPanels[id] {
					t.Fatalf("panel id %q was reused despite the authored root duplicate:\n%s", id, page.HTML)
				}
				seenPanels[id] = true
			}
		}
	})
}

func TestRepeatedRendersProduceStableBytes(t *testing.T) {
	render := func() string {
		page, diagnostics, err := NewRenderer().Parse("repeated.md", "en", repeatedInteractivePage)
		if err != nil {
			t.Fatal(err)
		}
		if len(diagnostics) != 0 {
			t.Fatalf("unexpected diagnostics: %#v", diagnostics)
		}
		return page.HTML
	}
	first := render()
	for i := 0; i < 24; i++ {
		if next := render(); next != first {
			t.Fatalf("render %d differed:\nfirst:\n%s\nnext:\n%s", i+2, first, next)
		}
	}

	const workers = 8
	results := make(chan string, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- render()
		}()
	}
	wg.Wait()
	close(results)
	for rendered := range results {
		if rendered != first {
			t.Fatalf("concurrent render differed:\nfirst:\n%s\nnext:\n%s", first, rendered)
		}
	}
}

func assertTabRelationships(t *testing.T, pageHTML string) {
	t.Helper()
	doc := parsePageMarkup(t, pageHTML)
	panelsByID := map[string]*html.Node{}
	walkHTML(doc, func(node *html.Node) {
		if htmlAttribute(node, "role") == "tabpanel" {
			id := htmlAttribute(node, "id")
			if id == "" {
				t.Fatalf("tabpanel without id: %s", pageHTML)
			}
			if panelsByID[id] != nil {
				t.Fatalf("tabpanel id %q is reused: %s", id, pageHTML)
			}
			panelsByID[id] = node
		}
	})
	seenTabs := map[string]bool{}
	walkHTML(doc, func(node *html.Node) {
		if htmlAttribute(node, "role") != "tab" {
			return
		}
		tabID := htmlAttribute(node, "id")
		if tabID == "" || seenTabs[tabID] {
			t.Fatalf("tab id %q is empty or reused: %s", tabID, pageHTML)
		}
		seenTabs[tabID] = true
		panel := panelsByID[htmlAttribute(node, "aria-controls")]
		if panel == nil {
			t.Fatalf("tab %q controls no panel: %s", tabID, pageHTML)
		}
		if got := htmlAttribute(panel, "aria-labelledby"); got != tabID {
			t.Fatalf("panel is labelled by %q, want %q: %s", got, tabID, pageHTML)
		}
	})
	if len(seenTabs) != len(panelsByID) {
		t.Fatalf("tab and panel counts differ: %d tabs, %d panels", len(seenTabs), len(panelsByID))
	}
}

func TestAuthoredIDCollidingWithGeneratedLocalIsReportedAndLocalized(t *testing.T) {
	const seed = `---
title: Seed
---

:::tabs
[One]

Same content.
:::
`
	seedPage, diagnostics, err := NewRenderer().Parse("seed.md", "en", seed)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected seed diagnostics: %#v", diagnostics)
	}
	seedDoc := parsePageMarkup(t, seedPage.HTML)
	var collision string
	walkHTML(seedDoc, func(node *html.Node) {
		if collision == "" && htmlAttribute(node, "role") == "tab" {
			collision = htmlAttribute(node, "id")
		}
	})
	if collision == "" {
		t.Fatalf("seed page has no tab id: %s", seedPage.HTML)
	}

	// The collision page contains the generated tabs followed by an authored
	// tabs instance using the generated local tab ID as its root.
	source := `---
title: Collision
---

:::tabs
[One]

Same content.
:::

:::tabs{id="` + collision + `"}
[One]

Same content.
:::
`
	page, diagnostics, err := NewRenderer().Parse("collision.md", "en", source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.HTML, `id="`+collision+`"`) {
		t.Fatalf("authored root was rewritten: %s", page.HTML)
	}
	if count := strings.Count(page.HTML, `id="`+collision+`"`); count != 2 {
		t.Fatalf("authored collision root count = %d, want 2: %s", count, page.HTML)
	}
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "duplicate-id" && strings.Contains(diagnostic.Message, collision) {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing authored/local collision diagnostic: %#v", diagnostics)
	}
	assertTabRelationships(t, page.HTML)
}
