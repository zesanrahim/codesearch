package engine

import (
	"encoding/gob"
	"fmt"
	"os"
)

type SearchResult struct {
	FilePath         string
	Line             int
	Offset           int
	Context          string
	CommitHash       string
	RepoURL          string
	MatchedLines     int
	TotalInputLines  int
	ConsecutiveBonus float64
}

func (sr *SearchResult) GetBlobURL() string {
	if sr.CommitHash == "" || sr.RepoURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/blob/%s/%s#L%d", sr.RepoURL, sr.CommitHash, sr.FilePath, sr.Line)
}

// indexPayload is the on-disk form of an Index. It deliberately omits Data and
// Mmap: those are views onto the corpus file, and encoding them would write a
// second full copy of the repository into the gob.
type indexPayload struct {
	CorpusPath     string
	LineOffsets    []int
	Trigrams       map[u32][]int
	FileBoundaries []FileBoundary
	CommitHash     string
	RepoURL        string
}

func (idx *Index) SaveIndex(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := indexPayload{
		CorpusPath:     idx.CorpusPath,
		LineOffsets:    idx.LineOffsets,
		Trigrams:       idx.Trigrams,
		FileBoundaries: idx.FileBoundaries,
		CommitHash:     idx.CommitHash,
		RepoURL:        idx.RepoURL,
	}

	return gob.NewEncoder(file).Encode(payload)
}

// LoadIndex reads a saved index and re-maps its corpus. It fails if the corpus
// is gone, since the offsets would no longer refer to anything; callers should
// treat that as a cache miss and rebuild.
func LoadIndex(filename string) (*Index, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var payload indexPayload
	if err := gob.NewDecoder(file).Decode(&payload); err != nil {
		return nil, err
	}

	idx := &Index{
		LineOffsets:    payload.LineOffsets,
		Trigrams:       payload.Trigrams,
		FileBoundaries: payload.FileBoundaries,
		CommitHash:     payload.CommitHash,
		RepoURL:        payload.RepoURL,
	}

	if err := idx.mapCorpus(payload.CorpusPath); err != nil {
		return nil, fmt.Errorf("corpus %q unavailable: %w", payload.CorpusPath, err)
	}

	return idx, nil
}
