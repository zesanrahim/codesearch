package engine

import (
	"encoding/gob"
	"os"
)

func SaveIndex(idx *Index, filename string) error {
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