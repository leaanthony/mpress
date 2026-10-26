package components

import (
	"strings"
	"sync"
	"testing"
)

// Standalone component rendering has no document scope. It must keep producing
// the same identifiers as before, even when Render is called repeatedly.
func TestStandaloneComponentRenderingKeepsStableIdentifiers(t *testing.T) {
	tabs := &Tabs{Meta: map[string]string{"alpha": "one", "zeta": "last"}, Sections: []TabSection{
		{Label: "One", Content: "First."},
		{Label: "Two", Content: "Second."},
	}}
	first, err := tabs.Render()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 32; i++ {
		next, err := tabs.Render()
		if err != nil {
			t.Fatal(err)
		}
		if next != first {
			t.Fatalf("standalone tabs render %d changed:\nfirst:\n%s\nnext:\n%s", i+2, first, next)
		}
	}
	if strings.Contains(first, "-2-tab-") || strings.Contains(first, "-2-panel-") {
		t.Fatalf("standalone render acquired a page-scoped suffix:\n%s", first)
	}

	table := &Table{Meta: map[string]string{"search": "true", "filter": "true"}}
	if err := table.Parse("| Name | Value |\n| A | 1 |"); err != nil {
		t.Fatal(err)
	}
	tableFirst, err := table.Render()
	if err != nil {
		t.Fatal(err)
	}
	tableSecond, err := table.Render()
	if err != nil {
		t.Fatal(err)
	}
	if tableFirst != tableSecond {
		t.Fatalf("standalone table render changed:\nfirst:\n%s\nnext:\n%s", tableFirst, tableSecond)
	}

	carousel := &Carousel{Meta: map[string]string{"label": "Demo"}}
	if err := carousel.Parse("First.\n---\nSecond."); err != nil {
		t.Fatal(err)
	}
	carouselFirst, err := carousel.Render()
	if err != nil {
		t.Fatal(err)
	}
	carouselSecond, err := carousel.Render()
	if err != nil {
		t.Fatal(err)
	}
	if carouselFirst != carouselSecond {
		t.Fatalf("standalone carousel render changed:\nfirst:\n%s\nnext:\n%s", carouselFirst, carouselSecond)
	}
}

func TestProcessRepeatedComponentsGetIndependentIDs(t *testing.T) {
	source := `:::carousel{label="Demo"}
First.
---
Second.
:::

:::carousel{label="Demo"}
First.
---
Second.
:::

:::table{header=true filter=true}
| Name | Value |
| A | 1 |
:::

:::table{header=true filter=true}
| Name | Value |
| A | 1 |
:::

:::api-playground{method="GET" path="/fixture" baseUrl="http://127.0.0.1:1"}
Fixture.
:::

:::api-playground{method="GET" path="/fixture" baseUrl="http://127.0.0.1:1"}
Fixture.
:::
`
	first, warnings := Process(source)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	second, warnings := Process(source)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if first != second {
		t.Fatalf("repeated Process calls differ:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if !strings.Contains(first, `id="mpress-carousel-`) || !strings.Contains(first, `-2" class="mpress-carousel"`) {
		t.Fatalf("repeated carousels did not receive distinct IDs:\n%s", first)
	}
	if !strings.Contains(first, `-2-filter-0"`) {
		t.Fatalf("repeated table filters did not receive distinct IDs:\n%s", first)
	}
	if !strings.Contains(first, `id="api-pg-`) || !strings.Contains(first, `-2" data-method=`) {
		t.Fatalf("repeated API playgrounds did not receive distinct IDs:\n%s", first)
	}
}

func TestProcessConcurrentRendersAreDeterministic(t *testing.T) {
	source := `:::tabs{alpha="one" zeta="last"}
[One]
Same.
[Two]
Same.
:::

:::tabs{alpha="one" zeta="last"}
[One]
Same.
[Two]
Same.
:::
`
	want, warnings := Process(source)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	const workers = 16
	results := make(chan string, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, warnings := Process(source)
			if len(warnings) != 0 {
				results <- "warnings: " + strings.Join(warnings, "; ")
				return
			}
			results <- got
		}()
	}
	wg.Wait()
	close(results)
	for got := range results {
		if got != want {
			t.Fatalf("concurrent Process render differed:\nwant:\n%s\ngot:\n%s", want, got)
		}
	}
}

func TestGeneratedSuffixesAvoidEveryClaimedID(t *testing.T) {
	ids := NewInstanceIDs()
	if got := ids.claimAuthored("mpress-example-2"); got != "mpress-example-2" {
		t.Fatalf("first authored claim = %q", got)
	}
	if got := ids.claimGenerated("mpress-example"); got != "mpress-example" {
		t.Fatalf("first generated claim = %q", got)
	}
	if got := ids.claimGenerated("mpress-example"); got != "mpress-example-3" {
		t.Fatalf("second generated claim = %q, want mpress-example-3", got)
	}
	if warnings := ids.takeWarnings(); len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestAuthoredIDCollisionWithGeneratedRootAndLocalsIsReported(t *testing.T) {
	ids := NewInstanceIDs()
	if got := ids.claimGenerated("mpress-tabs-a"); got != "mpress-tabs-a" {
		t.Fatalf("generated root = %q", got)
	}
	local := ids.claimGenerated("mpress-tabs-a-tab-1")
	if local != "mpress-tabs-a-tab-1" {
		t.Fatalf("generated local = %q", local)
	}
	stem := ids.claimAuthored(local)
	if stem == local || !strings.HasPrefix(stem, local+"-") {
		t.Fatalf("authored stem %q did not move away from the generated local %q", stem, local)
	}
	warnings := ids.takeWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], `duplicate component id "mpress-tabs-a-tab-1"`) {
		t.Fatalf("missing authored-local collision diagnostic: %#v", warnings)
	}

	filter := ids.claimGenerated("mpress-table-b-filter-0")
	stem = ids.claimAuthored(filter)
	if stem == filter {
		t.Fatalf("authored stem reused generated filter ID %q", filter)
	}
	warnings = ids.takeWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], `duplicate component id "mpress-table-b-filter-0"`) {
		t.Fatalf("missing authored-filter collision diagnostic: %#v", warnings)
	}
}

func TestAuthoredRootCollidingWithGeneratedRootKeepsTextAndPairing(t *testing.T) {
	ids := NewInstanceIDs()
	generated := &Tabs{Meta: map[string]string{}, Sections: []TabSection{
		{Label: "One", Content: "Same."},
		{Label: "Two", Content: "Same."},
	}, instances: ids}
	if _, err := generated.Render(); err != nil {
		t.Fatal(err)
	}
	ids.claimGenerated("shared")
	authored := &Tabs{Meta: map[string]string{"id": "shared"}, Sections: []TabSection{
		{Label: "One", Content: "Same."},
		{Label: "Two", Content: "Same."},
	}, instances: ids}
	rendered, err := authored.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `id="shared"`) {
		t.Fatalf("authored root was rewritten:\n%s", rendered)
	}
	warnings := ids.takeWarnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0], `duplicate component id "shared"`) {
		t.Fatalf("missing authored-root collision diagnostic: %#v", warnings)
	}
	if strings.Contains(rendered, `id="shared-tab-1"`) || strings.Contains(rendered, `aria-controls="shared-panel-1"`) {
		t.Fatalf("authored collision reused generated local references:\n%s", rendered)
	}
}
