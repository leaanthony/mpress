package knowledge

import "testing"

func TestEvaluateReportsRankAndCitationMetrics(t *testing.T) {
	pages := []Page{
		{PageSummary: PageSummary{ID: "install", ResourceURI: "mpress://knowledge/page/install", Language: "en", Version: "current", Route: "/install/", URL: "https://docs.example.test/install/", Title: "Install"}},
		{PageSummary: PageSummary{ID: "menus", ResourceURI: "mpress://knowledge/page/menus", Language: "en", Version: "current", Route: "/menus/", URL: "https://docs.example.test/menus/", Title: "Menus"}},
	}
	chunks := []Chunk{
		{ID: "install-section", PageID: "install", ResourceURI: "mpress://knowledge/section/install-section", Language: "en", Version: "current", Route: "/install/", URL: "https://docs.example.test/install/#linux", PageTitle: "Install", Title: "Linux", Text: "Install the Linux package."},
		{ID: "menu-section", PageID: "menus", ResourceURI: "mpress://knowledge/section/menu-section", Language: "en", Version: "current", Route: "/menus/", URL: "https://docs.example.test/menus/#context", PageTitle: "Menus", Title: "Context menus", Text: "Create a context menu."},
	}
	store := newStore(Manifest{}, pages, chunks, makeIndex(chunks))
	report := store.Evaluate(EvaluationSuite{Name: "test", Cases: []EvaluationCase{
		{ID: "install", Query: "Linux package", ExpectedRoutes: []string{"/install/"}},
		{ID: "menu", Query: "context menu", ExpectedRoutes: []string{"/menus/"}},
	}}, 5)
	if report.Passed != 2 || report.RecallAt1 != 1 || report.MRR != 1 || report.CitationRate != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}

func TestEvaluationSuiteValidation(t *testing.T) {
	path := t.TempDir() + "/suite.json"
	if _, err := LoadEvaluationSuite(path); err == nil {
		t.Fatal("missing suite did not fail")
	}
	if matchesExpectedRoute("https://docs.example.test/install/#linux", []string{"/install/"}) != true {
		t.Fatal("citation route did not match")
	}
}
