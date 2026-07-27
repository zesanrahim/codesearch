package engine

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// vocab is a spread of real-looking source lines. Synthetic corpora made of one
// repeated line give misleadingly good numbers, because every posting list ends
// up either empty or complete.
var vocab = []string{
	"func handler(w http.ResponseWriter, r *http.Request) {",
	"if err != nil {", "return nil, err", "defer file.Close()",
	"for i := range items {", "var buf bytes.Buffer", "import (",
	"type Config struct {", "result = append(result, item)",
	"ctx context.Context", "mu sync.Mutex", "http.HandleFunc(\"/\", h)",
	"json.Unmarshal(data, &out)", "log.Printf(\"failed: %v\", err)",
	"cache[key] = value", "close(ch)", "wg.Wait()",
	"t.Fatalf(\"got %v, want %v\", got, want)", "switch v := x.(type) {",
	"const maxRetries = 3", "// TODO: handle the error path",
}

// genCorpus builds a corpus of the given line count, planting needle every
// needleEvery lines so a query's selectivity is known up front.
func genCorpus(lines int, needle string, needleEvery int) string {
	rng := rand.New(rand.NewSource(42))

	var b strings.Builder
	b.Grow(lines * 48)

	for i := range lines {
		if needleEvery > 0 && i%needleEvery == 0 {
			fmt.Fprintf(&b, "\t%s // line %d\n", needle, i)
			continue
		}
		fmt.Fprintf(&b, "\t%s%d\n", vocab[rng.Intn(len(vocab))], i)
	}
	return b.String()
}

// writeCorpus persists a corpus and returns its path.
func writeCorpus(tb testing.TB, corpus string) string {
	tb.Helper()

	path := filepath.Join(tb.TempDir(), "corpus.txt")
	if err := os.WriteFile(path, []byte(corpus), 0o644); err != nil {
		tb.Fatalf("writing corpus: %v", err)
	}
	return path
}

// indexCorpus builds a ready-to-search index, excluded from the caller's timer.
func indexCorpus(tb testing.TB, corpus string) *Index {
	tb.Helper()

	idx := &Index{}
	if err := idx.MapBoundaries(writeCorpus(tb, corpus)); err != nil {
		tb.Fatalf("MapBoundaries: %v", err)
	}
	tb.Cleanup(func() { idx.Close() })

	idx.BuildTrigrams()
	return idx
}

const benchNeedle = "http.ResponseWriter"

// BenchmarkBuildTrigrams tracks indexing throughput, reported as MB/s of source
// consumed. This dominates the time a user waits on a new repo.
func BenchmarkBuildTrigrams(b *testing.B) {
	for _, lines := range []int{10_000, 100_000, 500_000} {
		b.Run(fmt.Sprintf("lines=%d", lines), func(b *testing.B) {
			corpus := genCorpus(lines, benchNeedle, 5000)
			path := writeCorpus(b, corpus)

			b.SetBytes(int64(len(corpus)))
			b.ReportAllocs()

			for b.Loop() {
				idx := &Index{}
				if err := idx.MapBoundaries(path); err != nil {
					b.Fatalf("MapBoundaries: %v", err)
				}
				idx.BuildTrigrams()
				idx.Close()
			}
		})
	}
}

// BenchmarkMapBoundaries isolates the line-offset scan from trigram building.
func BenchmarkMapBoundaries(b *testing.B) {
	corpus := genCorpus(200_000, benchNeedle, 5000)
	path := writeCorpus(b, corpus)

	b.SetBytes(int64(len(corpus)))
	b.ReportAllocs()

	for b.Loop() {
		idx := &Index{}
		if err := idx.MapBoundaries(path); err != nil {
			b.Fatalf("MapBoundaries: %v", err)
		}
		idx.Close()
	}
}

// BenchmarkSearch covers the selectivity range that drives cost: a miss should
// be nearly free, a rare term cheap, and a common term is the worst case
// because every trigram carries a long posting list.
func BenchmarkSearch(b *testing.B) {
	idx := indexCorpus(b, genCorpus(200_000, benchNeedle, 5000))

	cases := []struct {
		name  string
		query string
	}{
		{"miss", "zzzznotpresentzzzz"},
		{"rare", benchNeedle},
		{"common", "if err != nil"},
		{"long_query", "func handler(w http.ResponseWriter, r *http.Request) {"},
		// Below three bytes the index can't help, so this is the full scan.
		{"short_query", "wg"},
	}

	for _, tt := range cases {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				idx.Search(tt.query)
			}
		})
	}
}

// BenchmarkSearchMultiple exercises the multi-line path used by paste-a-block
// search, where cost scales with the number of input lines.
func BenchmarkSearchMultiple(b *testing.B) {
	idx := indexCorpus(b, genCorpus(200_000, benchNeedle, 5000))

	queries := []string{
		"func handler(w http.ResponseWriter, r *http.Request) {",
		"if err != nil {",
		"return nil, err",
		"defer file.Close()",
	}

	b.ReportAllocs()
	for b.Loop() {
		idx.SearchMultiple(queries)
	}
}

// BenchmarkIntersect measures posting-list intersection on its own. Search
// allocates a fresh slice per trigram step, so this is where that shows up.
func BenchmarkIntersect(b *testing.B) {
	sizes := []struct {
		name string
		a, b int
	}{
		{"both_large", 100_000, 100_000},
		{"large_small", 100_000, 100},
		{"small_large", 100, 100_000},
	}

	for _, tt := range sizes {
		b.Run(tt.name, func(b *testing.B) {
			left := make([]int, tt.a)
			for i := range left {
				left[i] = i
			}
			right := make([]int, tt.b)
			for i := range right {
				right[i] = i * 2
			}

			b.ReportAllocs()
			for b.Loop() {
				intersect(left, right)
			}
		})
	}
}

// BenchmarkGetLine covers result rendering, run once per displayed row.
func BenchmarkGetLine(b *testing.B) {
	idx := indexCorpus(b, genCorpus(200_000, benchNeedle, 5000))

	b.ReportAllocs()
	for b.Loop() {
		idx.GetLine(1000)
	}
}

// BenchmarkSaveLoadIndex covers the cache path taken on every repo reopen.
func BenchmarkSaveLoadIndex(b *testing.B) {
	idx := indexCorpus(b, genCorpus(100_000, benchNeedle, 5000))
	gobPath := filepath.Join(b.TempDir(), "index.gob")

	b.Run("save", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := idx.SaveIndex(gobPath); err != nil {
				b.Fatalf("SaveIndex: %v", err)
			}
		}
	})

	if err := idx.SaveIndex(gobPath); err != nil {
		b.Fatalf("SaveIndex: %v", err)
	}

	b.Run("load", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			loaded, err := LoadIndex(gobPath)
			if err != nil {
				b.Fatalf("LoadIndex: %v", err)
			}
			loaded.Close()
		}
	})
}
