package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const sampleCorpus = `package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
`

// buildIndex writes corpus to a temp file and indexes it the way the real
// pipeline does.
func buildIndex(t *testing.T, corpus string) *Index {
	t.Helper()

	path := filepath.Join(t.TempDir(), "corpus.txt")
	if err := os.WriteFile(path, []byte(corpus), 0o644); err != nil {
		t.Fatalf("writing corpus: %v", err)
	}

	idx := &Index{}
	if err := idx.MapBoundaries(path); err != nil {
		t.Fatalf("MapBoundaries: %v", err)
	}
	t.Cleanup(func() { idx.Close() })

	idx.BuildTrigrams()
	return idx
}

func TestBytesToTrigram(t *testing.T) {
	if got := bytesToTrigram([]byte("abc")); got != 0x616263 {
		t.Errorf("bytesToTrigram(abc) = %#x, want 0x616263", got)
	}
	if got := bytesToTrigram([]byte("ab")); got != 0 {
		t.Errorf("bytesToTrigram(ab) = %#x, want 0 for a short input", got)
	}
}

func TestGetTrigrams(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		{"", 0},
		{"ab", 0},
		{"abc", 1},
		{"abcd", 2},
		{"package", 5},
	}

	for _, tt := range tests {
		if got := len(getTrigrams(tt.query)); got != tt.want {
			t.Errorf("getTrigrams(%q) produced %d trigrams, want %d", tt.query, got, tt.want)
		}
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b []int
		want []int
	}{
		{"overlap", []int{1, 2, 3, 5}, []int{2, 3, 4}, []int{2, 3}},
		{"disjoint", []int{1, 3}, []int{2, 4}, nil},
		{"identical", []int{1, 2}, []int{1, 2}, []int{1, 2}},
		// An exhausted candidate set must stay exhausted; returning the other
		// operand would resurrect lines that failed an earlier trigram.
		{"empty left", nil, []int{1, 2}, nil},
		{"empty right", []int{1, 2}, nil, nil},
		{"both empty", nil, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intersect(tt.a, tt.b); !slices.Equal(got, tt.want) {
				t.Errorf("intersect(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSearch(t *testing.T) {
	idx := buildIndex(t, sampleCorpus)

	tests := []struct {
		name  string
		query string
		want  []int
	}{
		{"unique match", "Println", []int{5}},
		{"multiple matches", "main", []int{0, 4}},
		{"full line", `fmt.Println("hello world")`, []int{5}},
		{"no match", "nonexistent", nil},
		// Shorter than a trigram, so this exercises the full-scan fallback.
		{"short query", "import", []int{2}},
		{"single char", "{", []int{4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := idx.Search(tt.query); !slices.Equal(got, tt.want) {
				t.Errorf("Search(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

// A line can contain every trigram of a query without containing the query, so
// the index must only ever narrow the candidate set.
func TestSearchRejectsTrigramFalsePositives(t *testing.T) {
	idx := buildIndex(t, "abc\nabcxyz\nxyzabc\n")

	if got := idx.Search("abcxyz"); !slices.Equal(got, []int{1}) {
		t.Errorf("Search(abcxyz) = %v, want [1]", got)
	}
}

func TestSearchResultsAreSorted(t *testing.T) {
	var corpus strings.Builder
	for i := range 5000 {
		fmt.Fprintf(&corpus, "line %d needle\n", i)
	}

	idx := buildIndex(t, corpus.String())

	got := idx.Search("needle")
	if len(got) != 5000 {
		t.Fatalf("Search returned %d lines, want 5000", len(got))
	}
	if !slices.IsSorted(got) {
		t.Error("Search returned line numbers out of order")
	}
}

// A trigram repeated inside one line must post that line only once, or the
// posting lists bloat and intersection double-counts.
func TestBuildTrigramsDeduplicatesPostings(t *testing.T) {
	idx := buildIndex(t, "aaaaaaaa\nbbbb\n")

	tri := bytesToTrigram([]byte("aaa"))
	if got := idx.Trigrams[tri]; !slices.Equal(got, []int{0}) {
		t.Errorf("posting list for %q = %v, want [0]", "aaa", got)
	}
}

func TestPostingListsAreSorted(t *testing.T) {
	var corpus strings.Builder
	for range 5000 {
		corpus.WriteString("shared token\n")
	}

	idx := buildIndex(t, corpus.String())

	tri := bytesToTrigram([]byte("sha"))
	lines := idx.Trigrams[tri]
	if len(lines) != 5000 {
		t.Fatalf("posting list has %d entries, want 5000", len(lines))
	}
	if !slices.IsSorted(lines) {
		t.Error("posting list is not sorted, which breaks intersect")
	}
}

func TestSearchMultiple(t *testing.T) {
	idx := buildIndex(t, sampleCorpus)

	got := idx.SearchMultiple([]string{"package", "  ", "Println"})

	// The blank query is skipped, so "Println" keeps input index 2.
	want := map[int][]int{0: {0}, 5: {2}}
	if len(got) != len(want) {
		t.Fatalf("SearchMultiple returned %v, want %v", got, want)
	}
	for line, inputs := range want {
		if !slices.Equal(got[line], inputs) {
			t.Errorf("line %d matched inputs %v, want %v", line, got[line], inputs)
		}
	}
}

func TestCalculateConsecutiveBonus(t *testing.T) {
	tests := []struct {
		name    string
		indices []int
		want    int
	}{
		{"empty", nil, 0},
		{"single", []int{3}, 0},
		{"all consecutive", []int{0, 1, 2}, 2},
		{"unsorted but consecutive", []int{2, 0, 1}, 2},
		{"scattered", []int{0, 5, 9}, 0},
		{"partial run", []int{0, 1, 7}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateConsecutiveBonus(tt.indices); got != tt.want {
				t.Errorf("CalculateConsecutiveBonus(%v) = %d, want %d", tt.indices, got, tt.want)
			}
		})
	}
}

func TestCalculateConsecutiveBonusDoesNotMutateInput(t *testing.T) {
	indices := []int{5, 1, 3}
	CalculateConsecutiveBonus(indices)

	if !slices.Equal(indices, []int{5, 1, 3}) {
		t.Errorf("input was reordered to %v", indices)
	}
}

func TestGetLine(t *testing.T) {
	idx := buildIndex(t, sampleCorpus)

	if got := idx.GetLine(0); got != "package main" {
		t.Errorf("GetLine(0) = %q, want %q", got, "package main")
	}
	if got := idx.GetLine(-1); got != "" {
		t.Errorf("GetLine(-1) = %q, want empty", got)
	}
	if got := idx.GetLine(9999); got != "" {
		t.Errorf("GetLine(9999) = %q, want empty", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	idx := buildIndex(t, sampleCorpus)
	idx.CommitHash = "abc123"
	idx.RepoURL = "https://github.com/example/repo"
	idx.FileBoundaries = []FileBoundary{{FilePath: "main.go", StartOffset: 0, EndOffset: len(sampleCorpus)}}

	gobPath := filepath.Join(t.TempDir(), "index.gob")
	if err := idx.SaveIndex(gobPath); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}

	// The corpus must not be written into the gob; only offsets and postings.
	info, err := os.Stat(gobPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("SaveIndex wrote an empty file")
	}

	loaded, err := LoadIndex(gobPath)
	if err != nil {
		t.Fatalf("LoadIndex: %v", err)
	}
	t.Cleanup(func() { loaded.Close() })

	if loaded.CommitHash != "abc123" {
		t.Errorf("CommitHash = %q, want abc123", loaded.CommitHash)
	}
	if len(loaded.FileBoundaries) != 1 {
		t.Errorf("FileBoundaries = %v, want 1 entry", loaded.FileBoundaries)
	}
	// Search only works if the corpus was re-mapped on load.
	if got := loaded.Search("Println"); !slices.Equal(got, []int{5}) {
		t.Errorf("Search on loaded index = %v, want [5]", got)
	}
}

func TestLoadIndexFailsWhenCorpusIsGone(t *testing.T) {
	dir := t.TempDir()
	corpus := filepath.Join(dir, "corpus.txt")
	if err := os.WriteFile(corpus, []byte(sampleCorpus), 0o644); err != nil {
		t.Fatalf("writing corpus: %v", err)
	}

	idx := &Index{}
	if err := idx.MapBoundaries(corpus); err != nil {
		t.Fatalf("MapBoundaries: %v", err)
	}
	idx.BuildTrigrams()

	gobPath := filepath.Join(dir, "index.gob")
	if err := idx.SaveIndex(gobPath); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
	idx.Close()

	if err := os.Remove(corpus); err != nil {
		t.Fatalf("removing corpus: %v", err)
	}

	// Offsets without a corpus would silently return no results, so loading
	// has to fail loudly and let the caller rebuild.
	if _, err := LoadIndex(gobPath); err == nil {
		t.Error("LoadIndex succeeded despite a missing corpus")
	}
}

func TestGetBlobURL(t *testing.T) {
	sr := &SearchResult{
		FilePath:   "internal/engine/search.go",
		Line:       42,
		CommitHash: "abc123",
		RepoURL:    "https://github.com/example/repo",
	}

	want := "https://github.com/example/repo/blob/abc123/internal/engine/search.go#L42"
	if got := sr.GetBlobURL(); got != want {
		t.Errorf("GetBlobURL() = %q, want %q", got, want)
	}

	if got := (&SearchResult{FilePath: "a.go"}).GetBlobURL(); got != "" {
		t.Errorf("GetBlobURL() without commit metadata = %q, want empty", got)
	}
}
