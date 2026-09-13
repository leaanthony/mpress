package lighthouse

import (
	"strings"
	"testing"
)

func TestParseSummarisesScoresMetricsAndDiagnostics(t *testing.T) {
	report := `{
  "finalDisplayedUrl":"http://127.0.0.1:4191/guide/",
  "lighthouseVersion":"13.0.0",
  "fetchTime":"2026-08-06T01:02:03.000Z",
  "categories":{
    "performance":{"title":"Performance","score":0.94},
    "accessibility":{"title":"Accessibility","score":1},
    "best-practices":{"title":"Best Practices","score":0.88},
    "seo":{"title":"SEO","score":0.97}
  },
  "audits":{
    "first-contentful-paint":{"title":"First Contentful Paint","score":0.91,"scoreDisplayMode":"numeric","displayValue":"1.2 s"},
    "largest-contentful-paint":{"title":"Largest Contentful Paint","score":0.75,"scoreDisplayMode":"numeric","displayValue":"2.8 s"},
    "unused-css-rules":{"title":"Reduce unused CSS","score":0.4,"scoreDisplayMode":"metricSavings","displayValue":"Potential savings of 12 KiB"},
    "manual-check":{"title":"Check this manually","score":0,"scoreDisplayMode":"manual"}
  }
}`
	result, err := Parse([]byte(report))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Categories) != 4 || result.Categories[0].Score != 94 || result.Categories[2].ID != "best-practices" {
		t.Fatalf("unexpected categories: %#v", result.Categories)
	}
	if len(result.Metrics) != 2 || result.Metrics[1].DisplayValue != "2.8 s" {
		t.Fatalf("unexpected metrics: %#v", result.Metrics)
	}
	if len(result.Diagnostics) != 1 || !strings.Contains(result.Diagnostics[0].Title, "unused CSS") {
		t.Fatalf("unexpected diagnostics: %#v", result.Diagnostics)
	}
}

func TestParseRejectsReportWithoutCategoryScores(t *testing.T) {
	if _, err := Parse([]byte(`{"categories":{},"audits":{}}`)); err == nil {
		t.Fatal("expected a missing scores error")
	}
}
