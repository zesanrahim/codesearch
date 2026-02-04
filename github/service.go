package main

import (
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

// repo functions
func cloneRepo(repo Repo) error  {}
func fetchRepo(repo Repo) error  {}
func checkRepo(repo Repo) error  {}
func deleteRepo(repo Repo) error {}
