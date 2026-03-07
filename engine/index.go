package engine

import (
    "bytes"
    "os"
    "github.com/edsrzf/mmap-go"
    "sync"
    "strings"
)

var (
    start = 0x02
    end   = 0x03
)

type Index struct {
    data        []byte
    Mmap        mmap.MMap
    LineOffsets []int   
    Trigrams    map[string][]int
}

func (idx *Index) MapBoundaries(path string) error {
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
    idx.data = mmapData

    idx.LineOffsets = []int{0}

    offset := 0
    for {
        loc := bytes.IndexByte(idx.data[offset:], '\n')
        if loc == -1 {
            break
        }

        nextStart := offset + loc + 1

        idx.LineOffsets = append(idx.LineOffsets, nextStart)

        offset = nextStart
    }

    return nil
}

func (idx *Index) BuildTrigrams() {
    idx.Trigrams = make(map[string][]int)

    numWorkers := 8
    lineChan := make(chan int, 100)
    var wg sync.WaitGroup
    var mu sync.Mutex

    // Worker goroutines
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for lineNum := range lineChan {
                startOffset := idx.LineOffsets[lineNum]
                endOffset := len(idx.data)
                if lineNum+1 < len(idx.LineOffsets) {
                    endOffset = idx.LineOffsets[lineNum+1]
                }

                line := bytes.TrimRight(idx.data[startOffset:endOffset], "\r\n")

                wrapped := make([]byte, 0, len(line)+2)
                wrapped = append(wrapped, byte(start))
                wrapped = append(wrapped, line...)
                wrapped = append(wrapped, byte(end))

                lineTrigrams := make(map[string][]int)
                for i := 0; i <= len(wrapped)-3; i++ {
                    tri := string(wrapped[i : i+3])
                    lineTrigrams[tri] = append(lineTrigrams[tri], lineNum)
                }

                mu.Lock()
                for tri, lines := range lineTrigrams {
                    if list := idx.Trigrams[tri]; len(list) == 0 || list[len(list)-1] != lineNum {
                        idx.Trigrams[tri] = append(idx.Trigrams[tri], lines...)
                    }
                }
                mu.Unlock()
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
}

func (idx *Index) GetLine(lineNum int) string {
    if lineNum < 0 || lineNum >= len(idx.LineOffsets) {
        return ""
    }

    lineStart := idx.LineOffsets[lineNum]
    lineEnd := len(idx.data)
    if lineNum+1 < len(idx.LineOffsets) {
        lineEnd = idx.LineOffsets[lineNum+1]
    }

    return strings.TrimSpace(string(idx.data[lineStart:lineEnd]))
}