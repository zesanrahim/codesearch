package main

import (
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

func cloneRepo(repo *Repo) error {

	repo.Lock.Lock()
	defer repo.Lock.Unlock()

	if _, err := os.Stat(repo.RepoPath); !os.IsNotExist(err) {
		fmt.Printf("Repo %s already exists at %s\n", repo.Name, repo.RepoPath)
		return nil // already cloned
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
