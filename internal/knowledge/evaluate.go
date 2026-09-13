package knowledge

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
)

type EvaluationSuite struct {
	Name  string           `json:"name"`
	Cases []EvaluationCase `json:"cases"`
}

type EvaluationCase struct {
	ID             string   `json:"id"`
	Query          string   `json:"query"`
	Language       string   `json:"language,omitempty"`
	Version        string   `json:"version,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	ExpectedRoutes []string `json:"expectedRoutes"`
}

type EvaluationReport struct {
	Suite        string             `json:"suite"`
	Cases        int                `json:"cases"`
	Passed       int                `json:"passed"`
	RecallAt1    float64            `json:"recallAt1"`
	RecallAt3    float64            `json:"recallAt3"`
	RecallAt5    float64            `json:"recallAt5"`
	MRR          float64            `json:"mrr"`
	CitationRate float64            `json:"citationRate"`
	Results      []EvaluationResult `json:"results"`
}

type EvaluationResult struct {
	ID             string   `json:"id"`
	Query          string   `json:"query"`
	Passed         bool     `json:"passed"`
	Rank           int      `json:"rank,omitempty"`
	ExpectedRoutes []string `json:"expectedRoutes"`
	TopURLs        []string `json:"topUrls,omitempty"`
}

func LoadEvaluationSuite(path string) (EvaluationSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EvaluationSuite{}, err
	}
	var suite EvaluationSuite
	if err := json.Unmarshal(data, &suite); err != nil {
		return EvaluationSuite{}, err
	}
	if strings.TrimSpace(suite.Name) == "" {
		return EvaluationSuite{}, fmt.Errorf("evaluation suite name is required")
	}
	if len(suite.Cases) == 0 {
		return EvaluationSuite{}, fmt.Errorf("evaluation suite must contain at least one case")
	}
	seen := make(map[string]bool, len(suite.Cases))
	for index, test := range suite.Cases {
		if test.ID == "" || seen[test.ID] {
			return EvaluationSuite{}, fmt.Errorf("evaluation case %d has an empty or duplicate ID", index+1)
		}
		seen[test.ID] = true
		if strings.TrimSpace(test.Query) == "" || len(test.ExpectedRoutes) == 0 {
			return EvaluationSuite{}, fmt.Errorf("evaluation case %q requires a query and expectedRoutes", test.ID)
		}
	}
	return suite, nil
}

func (s *Store) Evaluate(suite EvaluationSuite, limit int) EvaluationReport {
	if limit < 5 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}
	report := EvaluationReport{Suite: suite.Name, Cases: len(suite.Cases), Results: make([]EvaluationResult, 0, len(suite.Cases))}
	var at1, at3, at5, reciprocal, cited float64
	for _, test := range suite.Cases {
		results := s.Search(SearchOptions{Query: test.Query, Language: test.Language, Version: test.Version, Tags: test.Tags, Limit: limit})
		result := EvaluationResult{ID: test.ID, Query: test.Query, ExpectedRoutes: append([]string(nil), test.ExpectedRoutes...)}
		for index, candidate := range results {
			result.TopURLs = append(result.TopURLs, candidate.URL)
			if candidate.URL != "" {
				cited++
			}
			if result.Rank == 0 && matchesExpectedRoute(candidate.URL, test.ExpectedRoutes) {
				result.Rank = index + 1
			}
		}
		if result.Rank > 0 {
			result.Passed = true
			report.Passed++
			reciprocal += 1 / float64(result.Rank)
			if result.Rank <= 1 {
				at1++
			}
			if result.Rank <= 3 {
				at3++
			}
			if result.Rank <= 5 {
				at5++
			}
		}
		report.Results = append(report.Results, result)
	}
	total := float64(report.Cases)
	report.RecallAt1, report.RecallAt3, report.RecallAt5, report.MRR = at1/total, at3/total, at5/total, reciprocal/total
	resultCount := 0
	for _, result := range report.Results {
		resultCount += len(result.TopURLs)
	}
	if resultCount > 0 {
		report.CitationRate = cited / float64(resultCount)
	}
	sort.SliceStable(report.Results, func(i, j int) bool {
		if report.Results[i].Passed != report.Results[j].Passed {
			return !report.Results[i].Passed
		}
		return report.Results[i].ID < report.Results[j].ID
	})
	return report
}

func matchesExpectedRoute(rawURL string, expected []string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	path := strings.TrimRight(parsed.Path, "/") + "/"
	for _, route := range expected {
		if path == strings.TrimRight(route, "/")+"/" {
			return true
		}
	}
	return false
}
