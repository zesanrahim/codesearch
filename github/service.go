package main

import (
	"codesearch/database"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type Repo struct {
	Name     string
	RepoPath string
	CloneURL string
	Lock     sync.RWMutex
	// LastFetched time.Time
	// Size        int64
}

var (
	repoCache = make(map[string]*Repo)
	cacheLock sync.RWMutex
)

func getRepo(name string) (*Repo, error) {

	cacheLock.RLock()

	if repo, exists := repoCache[name]; exists {
		cacheLock.RUnlock()
		return repo, nil
	}

	database.GetClient()

	var repoPath, cloneURL string

	// TODO: Add Schema's and get data accorrdingly
	// sql query once db has schema
	// err := client.DB("query")

	// if err != nill {
	// 	return nil
	// }

	repo := &Repo{
		Name:     name,
		RepoPath: repoPath,
		CloneURL: cloneURL,
	}

	cacheLock.Lock()
	repoCache[name] = repo
	cacheLock.Unlock()

	return repo, nil
}

func cloneRepo(repo *Repo) error {

	repo.Lock.Lock()
	defer repo.Lock.Unlock()

	if _, err := os.Stat(repo.RepoPath); !os.IsNotExist(err) {
		fmt.Printf("Repo %s already exists at %s\n", repo.Name, repo.RepoPath)
		return nil
	}

	if err := os.MkdirAll(repo.RepoPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "--bare", repo.Name, repo.RepoPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Successfully cloned repo %s\n", repo.Name)
	return nil

}
func fetchRepo(repo *Repo) error  { return nil }
func checkRepo(repo *Repo) error  { return nil }
func deleteRepo(repo *Repo) error { return nil }
