package engine

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"sync"

	mmap "github.com/edsrzf/mmap-go"
)

type u32 = uint32

// Sentinels wrap each line so trigrams can encode line boundaries.
const (
	sentinelStart = byte(0x02)
	sentinelEnd   = byte(0x03)
)

type FileBoundary struct {
	FilePath    string
	StartOffset int
	EndOffset   int
}

type Index struct {
	// Data and Mmap are views onto the memory-mapped corpus at CorpusPath.
	// They are rebuilt on load rather than persisted, so the corpus is never
	// written twice.
	Data []byte
	Mmap mmap.MMap

	CorpusPath     string
	LineOffsets    []int
	Trigrams       map[u32][]int
	FileBoundaries []FileBoundary
	CommitHash     string
	RepoURL        string
}

func bytesToTrigram(b []byte) u32 {
	if len(b) < 3 {
		return 0
	}
	return u32(b[0])<<16 | u32(b[1])<<8 | u32(b[2])
}

// mapCorpus memory-maps path and points the index at its contents.
func (idx *Index) mapCorpus(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	mmapData, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		return err
	}

	idx.Mmap = mmapData
	idx.Data = mmapData
	idx.CorpusPath = path
	return nil
}

// Close releases the memory-mapped corpus.
func (idx *Index) Close() error {
	if idx.Mmap == nil {
		return nil
	}

	err := idx.Mmap.Unmap()
	idx.Mmap = nil
	idx.Data = nil
	return err
}

// MapBoundaries maps the corpus at path and records where every line starts.
func (idx *Index) MapBoundaries(path string) error {
	if err := idx.mapCorpus(path); err != nil {
		return err
	}

	idx.LineOffsets = []int{0}

	offset := 0
	for {
		loc := bytes.IndexByte(idx.Data[offset:], '\n')
		if loc == -1 {
			break
		}

		offset += loc + 1
		idx.LineOffsets = append(idx.LineOffsets, offset)
	}

	return nil
}

// lineBytes returns the raw bytes of a line, trailing newline included.
func (idx *Index) lineBytes(lineNum int) []byte {
	lineStart := idx.LineOffsets[lineNum]
	lineEnd := len(idx.Data)
	if lineNum+1 < len(idx.LineOffsets) {
		lineEnd = idx.LineOffsets[lineNum+1]
	}
	return idx.Data[lineStart:lineEnd]
}

func (idx *Index) BuildTrigrams() {
	idx.Trigrams = make(map[u32][]int)

	totalLines := len(idx.LineOffsets)
	if totalLines == 0 {
		return
	}

	numWorkers := min(runtime.NumCPU(), totalLines)
	chunk := (totalLines + numWorkers - 1) / numWorkers

	partials := make([]map[u32][]int, numWorkers)

	var wg sync.WaitGroup
	for w := range numWorkers {
		start := w * chunk
		end := min(start+chunk, totalLines)
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(w, start, end int) {
			defer wg.Done()

			local := make(map[u32][]int)
			seen := make(map[u32]struct{})

			for lineNum := start; lineNum < end; lineNum++ {
				line := bytes.TrimRight(idx.lineBytes(lineNum), "\r\n")

				wrapped := make([]byte, 0, len(line)+2)
				wrapped = append(wrapped, sentinelStart)
				wrapped = append(wrapped, line...)
				wrapped = append(wrapped, sentinelEnd)

				// A trigram repeated within a line must only post that line
				// once, or the posting list grows duplicates.
				clear(seen)
				for i := 0; i <= len(wrapped)-3; i++ {
					tri := bytesToTrigram(wrapped[i : i+3])
					if _, dup := seen[tri]; dup {
						continue
					}
					seen[tri] = struct{}{}
					local[tri] = append(local[tri], lineNum)
				}
			}
			partials[w] = local
		}(w, start, end)
	}
	wg.Wait()

	// Workers own contiguous ascending line ranges, so merging them in worker
	// order leaves every posting list sorted for intersect.
	for _, local := range partials {
		for tri, lines := range local {
			idx.Trigrams[tri] = append(idx.Trigrams[tri], lines...)
		}
	}
}

func (idx *Index) GetLine(lineNum int) string {
	if lineNum < 0 || lineNum >= len(idx.LineOffsets) {
		return ""
	}
	return strings.TrimSpace(string(idx.lineBytes(lineNum)))
}
