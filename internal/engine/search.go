package engine

import (
	"bytes"
	"runtime"
	"slices"
	"strings"
	"sync"
)

func getTrigrams(query string) []u32 {
	queryBytes := []byte(query)
	if len(queryBytes) < 3 {
		return nil
	}

	trigrams := make([]u32, 0, len(queryBytes)-2)
	for i := 0; i <= len(queryBytes)-3; i++ {
		trigrams = append(trigrams, bytesToTrigram(queryBytes[i:i+3]))
	}
	return trigrams
}

// Search returns the line numbers containing query, in ascending order.
// Queries shorter than a trigram can't use the index, so they fall back to a
// full scan.
func (idx *Index) Search(query string) []int {
	if len(query) < 3 {
		return idx.scanLines(nil, query)
	}

	trigrams := getTrigrams(query)

	candidates, exists := idx.Trigrams[trigrams[0]]
	if !exists {
		return nil
	}

	for _, tri := range trigrams[1:] {
		lines, exists := idx.Trigrams[tri]
		if !exists {
			return nil
		}
		candidates = intersect(candidates, lines)
		if len(candidates) == 0 {
			return nil
		}
	}

	// The trigram index only narrows the search: a line can hold every trigram
	// of the query without holding the query itself, so verify each candidate.
	return idx.scanLines(candidates, query)
}

// intersect returns the values present in both sorted slices.
func intersect(a, b []int) []int {
	var result []int

	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			result = append(result, a[i])
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return result
}

// scanLines reports which of the candidate lines contain query. A nil
// candidates slice scans the whole index.
func (idx *Index) scanLines(candidates []int, query string) []int {
	total := len(candidates)
	if candidates == nil {
		total = len(idx.LineOffsets)
	}
	if total == 0 || query == "" {
		return nil
	}

	needle := []byte(query)

	numWorkers := min(runtime.NumCPU(), total)
	chunk := (total + numWorkers - 1) / numWorkers

	// Each worker collects into its own slice so the scan doesn't serialise on
	// a shared mutex, then the chunks are concatenated in order.
	partials := make([][]int, numWorkers)

	var wg sync.WaitGroup
	for w := range numWorkers {
		start := w * chunk
		end := min(start+chunk, total)
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()

			var local []int
			for i := start; i < end; i++ {
				lineNum := i
				if candidates != nil {
					lineNum = candidates[i]
				}
				if bytes.Contains(idx.lineBytes(lineNum), needle) {
					local = append(local, lineNum)
				}
			}
			partials[w] = local
		}(w, start, end)
	}
	wg.Wait()

	var results []int
	for _, partial := range partials {
		results = append(results, partial...)
	}
	slices.Sort(results)
	return results
}

// SearchMultiple maps each matching line number to the indices of the input
// queries that matched it.
func (idx *Index) SearchMultiple(queries []string) map[int][]int {
	results := make(map[int][]int)

	for inputIdx, query := range queries {
		trimmed := strings.TrimSpace(query)
		if trimmed == "" {
			continue
		}
		for _, lineNum := range idx.Search(trimmed) {
			results[lineNum] = append(results[lineNum], inputIdx)
		}
	}
	return results
}

// CalculateConsecutiveBonus counts adjacent pairs among the matched input
// lines, so a block matched in its original order scores above scattered hits.
func CalculateConsecutiveBonus(matchedIndices []int) int {
	if len(matchedIndices) < 2 {
		return 0
	}

	sorted := slices.Clone(matchedIndices)
	slices.Sort(sorted)

	consecutive := 0
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1]+1 {
			consecutive++
		}
	}
	return consecutive
}
