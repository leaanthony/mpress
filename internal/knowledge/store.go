package knowledge

import (
	"sort"
	"strings"
)

type Store struct {
	Manifest Manifest
	Pages    []Page
	Chunks   []Chunk
	pages    map[string]*Page
	chunks   map[string]*Chunk
	terms    map[string][]Posting
}

type SearchOptions struct {
	Query    string   `json:"query"`
	Language string   `json:"language,omitempty"`
	Version  string   `json:"version,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

type SearchResult struct {
	ChunkID      string   `json:"chunkId"`
	PageID       string   `json:"pageId"`
	ResourceURI  string   `json:"resourceUri"`
	PageResource string   `json:"pageResource"`
	Title        string   `json:"title"`
	PageTitle    string   `json:"pageTitle"`
	URL          string   `json:"url"`
	Language     string   `json:"language"`
	Version      string   `json:"version"`
	HeadingPath  []string `json:"headingPath,omitempty"`
	Snippet      string   `json:"snippet"`
	Score        int      `json:"score"`
}

func newStore(manifest Manifest, pages []Page, chunks []Chunk, index Index) *Store {
	store := &Store{Manifest: manifest, Pages: pages, Chunks: chunks, pages: make(map[string]*Page, len(pages)), chunks: make(map[string]*Chunk, len(chunks)), terms: make(map[string][]Posting, len(index.Terms))}
	for index := range store.Pages {
		store.pages[store.Pages[index].ID] = &store.Pages[index]
	}
	for index := range store.Chunks {
		store.chunks[store.Chunks[index].ID] = &store.Chunks[index]
	}
	for _, term := range index.Terms {
		store.terms[term.Term] = term.Postings
	}
	return store
}

func (s *Store) Page(id string) (*Page, bool) {
	page, ok := s.pages[id]
	return page, ok
}

func (s *Store) Chunk(id string) (*Chunk, bool) {
	chunk, ok := s.chunks[id]
	return chunk, ok
}

func (s *Store) Search(options SearchOptions) []SearchResult {
	query := strings.TrimSpace(options.Query)
	queryTokens := tokens(query)
	if query == "" || len(queryTokens) == 0 {
		return nil
	}
	if options.Limit <= 0 {
		options.Limit = 8
	}
	if options.Limit > 50 {
		options.Limit = 50
	}
	type score struct {
		value   int
		matched map[string]bool
	}
	scores := make(map[string]*score)
	type pageScore struct {
		terms   map[string]int
		matched map[string]bool
	}
	pageScores := make(map[string]*pageScore)
	for _, token := range queryTokens {
		postings := s.terms[token]
		rarity := 1
		if len(postings) > 0 {
			rarity = len(s.Chunks) / len(postings)
			if rarity > 12 {
				rarity = 12
			}
		}
		for _, posting := range postings {
			chunk := s.chunks[posting.ChunkID]
			if chunk == nil || !matchesFilters(chunk, options) {
				continue
			}
			current := scores[posting.ChunkID]
			if current == nil {
				current = &score{matched: make(map[string]bool)}
				scores[posting.ChunkID] = current
			}
			textScore := posting.Text
			if textScore > 8 {
				textScore = 8
			}
			contribution := textScore + posting.Page*100 + posting.Title*20 + posting.Route*75 + posting.Tags*8 + rarity
			current.value += contribution
			current.matched[token] = true
			page := pageScores[chunk.PageID]
			if page == nil {
				page = &pageScore{terms: make(map[string]int), matched: make(map[string]bool)}
				pageScores[chunk.PageID] = page
			}
			if contribution > page.terms[token] {
				page.terms[token] = contribution
			}
			page.matched[token] = true
		}
	}
	lowerQuery := strings.ToLower(query)
	results := make([]SearchResult, 0, len(scores))
	for id, score := range scores {
		chunk := s.chunks[id]
		pageCoverage := pageScores[chunk.PageID]
		if pageCoverage != nil {
			for _, value := range pageCoverage.terms {
				score.value += value
			}
			score.value += len(pageCoverage.matched) * 8
			if len(pageCoverage.matched) == len(queryTokens) {
				score.value += 40
			}
		}
		score.value += len(score.matched) * 5
		if len(score.matched) == len(queryTokens) {
			score.value += 25
		}
		if strings.Contains(strings.ToLower(chunk.Title), lowerQuery) || strings.Contains(strings.ToLower(chunk.PageTitle), lowerQuery) {
			score.value += 20
		} else if strings.Contains(strings.ToLower(chunk.Text), lowerQuery) {
			score.value += 8
		}
		page := s.pages[chunk.PageID]
		pageResource := ""
		if page != nil {
			pageResource = page.ResourceURI
		}
		results = append(results, SearchResult{
			ChunkID: chunk.ID, PageID: chunk.PageID, ResourceURI: chunk.ResourceURI,
			PageResource: pageResource, Title: chunk.Title, PageTitle: chunk.PageTitle,
			URL: chunk.URL, Language: chunk.Language, Version: chunk.Version,
			HeadingPath: append([]string(nil), chunk.HeadingPath...), Snippet: snippet(chunk.Text), Score: score.value,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].URL != results[j].URL {
			return results[i].URL < results[j].URL
		}
		return results[i].ChunkID < results[j].ChunkID
	})
	// A long tutorial can contain many matching sections. Return the strongest
	// section from each page first so focused reference pages remain visible.
	diverse := make([]SearchResult, 0, options.Limit)
	seenPages := make(map[string]bool, options.Limit)
	for _, result := range results {
		if seenPages[result.PageID] {
			continue
		}
		seenPages[result.PageID] = true
		diverse = append(diverse, result)
		if len(diverse) == options.Limit {
			break
		}
	}
	return diverse
}

func matchesFilters(chunk *Chunk, options SearchOptions) bool {
	if options.Language != "" && chunk.Language != options.Language {
		return false
	}
	if options.Version != "" && chunk.Version != options.Version {
		return false
	}
	if len(options.Tags) == 0 {
		return true
	}
	available := make(map[string]bool, len(chunk.Tags))
	for _, tag := range chunk.Tags {
		available[strings.ToLower(tag)] = true
	}
	for _, tag := range options.Tags {
		if !available[strings.ToLower(strings.TrimSpace(tag))] {
			return false
		}
	}
	return true
}

func snippet(text string) string {
	runes := []rune(text)
	if len(runes) <= 260 {
		return text
	}
	return strings.TrimSpace(string(runes[:257])) + "..."
}
