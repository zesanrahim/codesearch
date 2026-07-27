package github

import (
	"os"
	"path/filepath"
	"strings"

	"codesearch/internal/engine"
	"codesearch/internal/paths"
)

type CachedRepo struct {
	Repo       *Repo
	Files      int
	CommitHash string
}

func ListCachedRepos() ([]CachedRepo, error) {
	indexDir := paths.IndexDir()

	orgs, err := os.ReadDir(indexDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cached []CachedRepo
	for _, orgEntry := range orgs {
		if !orgEntry.IsDir() {
			continue
		}
		org := orgEntry.Name()

		entries, err := os.ReadDir(filepath.Join(indexDir, org))
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gob") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".gob")

			idx, err := engine.LoadIndex(paths.IndexFile(org, name))
			if err != nil {
				continue
			}

			cached = append(cached, CachedRepo{
				Repo: &Repo{
					Org:      org,
					Name:     name,
					RepoPath: paths.RepoDir(org, name),
					CloneURL: idx.RepoURL,
				},
				Files:      len(idx.FileBoundaries),
				CommitHash: idx.CommitHash,
			})

			idx.Close()
		}
	}

	return cached, nil
}
