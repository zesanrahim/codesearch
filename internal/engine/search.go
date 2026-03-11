package engine

import (
	"bytes"
	"sync"
	"runtime"
)

func (idx *Index) getTrigrams(query string) []u32 {
	var trigrams []u32
	queryBytes := []byte(query)
	for i := 0; i <= len(queryBytes)-3; i++ {
		tri := bytesToTrigram(queryBytes[i : i+3])
		trigrams = append(trigrams, tri)
	}
	return trigrams
}

func (idx *Index) Search(query string) []int {
	if len(query) < 3 {
		return idx.linearSearch(query)
	}

	trigrams := idx.getTrigrams(query)

	var results []int
	firstTrigram := trigrams[0]

	if lines, exists := idx.Trigrams[firstTrigram]; exists {
		results = lines
	} else {
		return []int{}
	}

	for i := 1; i < len(trigrams); i++ {
		if lines, exists := idx.Trigrams[trigrams[i]]; exists {
			results = idx.intersect(results, lines)
		} else {
			return []int{}
		}
	}

	return idx.filterResults(results, query)
}

func (idx *Index) intersect(a, b []int) []int {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}

	var result []int
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			result = append(result, a[i])
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}
	return result
}

func (idx *Index) filterResults(lineNums []int, query string) []int {
	var results []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	lineChan := make(chan int, 100)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for lineNum := range lineChan {
				lineStart := idx.LineOffsets[lineNum]
				lineEnd := len(idx.Data)
				if lineNum+1 < len(idx.LineOffsets) {
					lineEnd = idx.LineOffsets[lineNum+1]
				}
				line := idx.Data[lineStart:lineEnd]

				if bytes.Contains(line, []byte(query)) {
					mu.Lock()
					results = append(results, lineNum)
					mu.Unlock()
				}
			}
		}()
	}

	go func() {
		for _, lineNum := range lineNums {
			lineChan <- lineNum
		}
		close(lineChan)
	}()

	wg.Wait()
	return results
}

func (idx *Index) linearSearch(query string) []int {
	var results []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	lineChan := make(chan int, 100)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for lineNum := range lineChan {
				lineStart := idx.LineOffsets[lineNum]
				lineEnd := len(idx.Data)
				if lineNum+1 < len(idx.LineOffsets) {
					lineEnd = idx.LineOffsets[lineNum+1]
				}
				line := idx.Data[lineStart:lineEnd]

				if bytes.Contains(line, []byte(query)) {
					mu.Lock()
					results = append(results, lineNum)
					mu.Unlock()
				}
			}
		}()
	}

	go func() {
		for lineNum := range idx.LineOffsets {
			lineChan <- lineNum
		}
		close(lineChan)
	}()

	wg.Wait()
	return results
}