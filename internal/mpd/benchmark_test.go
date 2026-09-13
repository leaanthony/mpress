package mpd

import (
	"testing"

	"github.com/leaanthony/mpress/internal/mpdcorpus"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

var (
	benchmarkDocument *Document
	benchmarkGoldmark any
)

func BenchmarkParsePage(b *testing.B) {
	source := mpdcorpus.Source(9)
	b.ReportAllocs()
	b.SetBytes(int64(len(source)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkDocument = Parse("benchmark.mpd", source)
	}
}

func BenchmarkParseCorpus10000(b *testing.B) {
	sources, size := benchmarkSources(10_000)
	b.ReportAllocs()
	b.SetBytes(size)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		for index, source := range sources {
			benchmarkDocument = Parse(benchmarkFilename(index), source)
		}
	}
	b.ReportMetric(float64(len(sources))*float64(b.N)/b.Elapsed().Seconds(), "pages/s")
}

func BenchmarkGoldmarkCorpus10000(b *testing.B) {
	sources, size := benchmarkSources(10_000)
	markdown := goldmark.New()
	parser := markdown.Parser()
	b.ReportAllocs()
	b.SetBytes(size)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		for _, source := range sources {
			benchmarkGoldmark = parser.Parse(text.NewReader(source))
		}
	}
	b.ReportMetric(float64(len(sources))*float64(b.N)/b.Elapsed().Seconds(), "pages/s")
}

func benchmarkSources(count int) ([][]byte, int64) {
	sources := make([][]byte, count)
	var size int64
	for index := range sources {
		sources[index] = mpdcorpus.Source(index)
		size += int64(len(sources[index]))
	}
	return sources, size
}

func benchmarkFilename(index int) string {
	// Avoid benchmark-side formatting allocations in the measured parser path.
	return "generated.mpd"
}
