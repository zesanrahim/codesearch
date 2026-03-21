package engine

import (
	"encoding/gob"
	"fmt"
	"os"
)

type SearchResult struct {
	FilePath          string
	Line              int
	Offset            int
	Context           string
	CommitHash        string
	RepoURL           string
	MatchedLines      int     
	TotalInputLines   int     
	ConsecutiveBonus  float64 
}

func (sr *SearchResult) GetBlobURL() string {
	if sr.CommitHash == "" || sr.RepoURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/blob/%s/%s#L%d", sr.RepoURL, sr.CommitHash, sr.FilePath, sr.Line)
}
	
	

func (idx *Index) SaveIndex(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	err = encoder.Encode(idx)
	if err != nil {
		return err
	}
	
	return nil
}

func LoadIndex(filename string) (*Index, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var idx Index
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&idx)
	if err != nil {
		return nil, err
	}

	return &idx, nil
}