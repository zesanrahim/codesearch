package engine

import "bytes"

func (idx *Index) getTrigrams(query string) []string {
    var trigrams []string
    for i := 0; i <= len(query)-3; i++ {
        tri := query[i : i+3]
        trigrams = append(trigrams, tri)
    }
    return trigrams
}

func (idx *Index) Search(query string) []int {
    if len(query) < 3 {
        return idx.linearSearch(query)
    }

    trigrams := idx.getTrigrams(query)

    var candidateLines []int
    for _, tri := range trigrams {
        if lines, exists := idx.Trigrams[tri]; exists {
            candidateLines = idx.intersect(candidateLines, lines)
        } else {
            return []int{}
        }
    }

    results := idx.linearSearch(query)

    for _, lineNum := range results {
        lineStart := idx.LineOffsets[lineNum]
        lineEnd := len(idx.data)
        if lineNum+1 < len(idx.LineOffsets) {
            lineEnd = idx.LineOffsets[lineNum+1]
        }
        line := string(idx.data[lineStart:lineEnd])
        _ = line // Use it or remove
    }
    return results
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

func (idx *Index) linearSearch(query string) []int {
    var results []int
    for lineNum := range idx.LineOffsets {
        lineStart := idx.LineOffsets[lineNum]
        lineEnd := len(idx.data)
        if lineNum+1 < len(idx.LineOffsets) {
            lineEnd = idx.LineOffsets[lineNum+1]
        }
        line := idx.data[lineStart:lineEnd]
        if bytes.Contains(line, []byte(query)) {
            results = append(results, lineNum)
        }
    }
    return results
}