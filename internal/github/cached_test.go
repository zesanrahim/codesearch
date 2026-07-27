package github

import (
	"os"
	"path/filepath"
	"testing"

	"codesearch/internal/engine"
	"codesearch/internal/paths"
)

func writeIndex(t *testing.T, org, name, corpus string) {
	t.Helper()

	corpusPath := paths.CorpusFile(org, name)
	if err := os.MkdirAll(filepath.Dir(corpusPath), 0o755); err != nil {
		t.Fatalf("mkdir corpus: %v", err)
	}
	if err := os.WriteFile(corpusPath, []byte(corpus), 0o644); err != nil {
		t.Fatalf("write corpus: %v", err)
	}

	idx := &engine.Index{}
	if err := idx.MapBoundaries(corpusPath); err != nil {
		t.Fatalf("MapBoundaries: %v", err)
	}
	defer idx.Close()

	idx.BuildTrigrams()
	idx.CommitHash = "deadbeef"
	idx.RepoURL = "https://github.com/" + org + "/" + name
	idx.FileBoundaries = []engine.FileBoundary{
		{FilePath: "main.go", StartOffset: 0, EndOffset: len(corpus)},
	}

	indexPath := paths.IndexFile(org, name)
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("mkdir index: %v", err)
	}
	if err := idx.SaveIndex(indexPath); err != nil {
		t.Fatalf("SaveIndex: %v", err)
	}
}

func TestListCachedReposEmptyWhenNothingCached(t *testing.T) {
	t.Setenv(paths.EnvCacheDir, t.TempDir())

	cached, err := ListCachedRepos()
	if err != nil {
		t.Fatalf("ListCachedRepos: %v", err)
	}
	if len(cached) != 0 {
		t.Errorf("got %d cached repos, want 0", len(cached))
	}
}

func TestListCachedReposReadsIdentityFromLayout(t *testing.T) {
	t.Setenv(paths.EnvCacheDir, t.TempDir())
	writeIndex(t, "acme", "widget", "package main\n")

	cached, err := ListCachedRepos()
	if err != nil {
		t.Fatalf("ListCachedRepos: %v", err)
	}
	if len(cached) != 1 {
		t.Fatalf("got %d cached repos, want 1", len(cached))
	}

	got := cached[0]
	if got.Repo.Org != "acme" || got.Repo.Name != "widget" {
		t.Errorf("identity = %q/%q, want acme/widget", got.Repo.Org, got.Repo.Name)
	}
	if want := paths.RepoDir("acme", "widget"); got.Repo.RepoPath != want {
		t.Errorf("RepoPath = %q, want %q", got.Repo.RepoPath, want)
	}
	if got.CommitHash != "deadbeef" {
		t.Errorf("CommitHash = %q, want deadbeef", got.CommitHash)
	}
	if got.Files != 1 {
		t.Errorf("Files = %d, want 1", got.Files)
	}
}

func TestListCachedReposKeepsForksApart(t *testing.T) {
	t.Setenv(paths.EnvCacheDir, t.TempDir())
	writeIndex(t, "facebook", "react", "upstream\n")
	writeIndex(t, "myfork", "react", "fork\n")

	cached, err := ListCachedRepos()
	if err != nil {
		t.Fatalf("ListCachedRepos: %v", err)
	}
	if len(cached) != 2 {
		t.Fatalf("got %d cached repos, want 2 ; forks sharing a cache key would collapse to 1", len(cached))
	}

	orgs := map[string]bool{}
	for _, c := range cached {
		if c.Repo.Name != "react" {
			t.Errorf("name = %q, want react", c.Repo.Name)
		}
		orgs[c.Repo.Org] = true
	}
	for _, want := range []string{"facebook", "myfork"} {
		if !orgs[want] {
			t.Errorf("missing org %q, got %v", want, orgs)
		}
	}
}

func TestListCachedReposSkipsUnloadableIndex(t *testing.T) {
	t.Setenv(paths.EnvCacheDir, t.TempDir())

	indexPath := paths.IndexFile("acme", "broken")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("mkdir index: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("not a gob"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	cached, err := ListCachedRepos()
	if err != nil {
		t.Fatalf("ListCachedRepos should skip a corrupt cache, got error: %v", err)
	}
	if len(cached) != 0 {
		t.Errorf("got %d cached repos, want 0", len(cached))
	}
}
