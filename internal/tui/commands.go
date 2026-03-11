package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codesearch/internal/engine"
	"codesearch/internal/github"

	tea "github.com/charmbracelet/bubbletea"
)

type IndexResultMsg struct {
	RepoName string
	Result   string
	Repo     *github.Repo
	Err      error
}

type SearchResultMsg struct {
	Query   string
	Results []engine.SearchResult
	Err     error
}

type CachedReposMsg struct {
	Repos   []*github.Repo
	Results []string
}

func LoadCachedRepos() tea.Cmd {
	return func() tea.Msg {
		indexDir := "/tmp/codesearch/index"
		entries, err := os.ReadDir(indexDir)
		if err != nil {
			return CachedReposMsg{}
		}

		var repos []*github.Repo
		var results []string

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gob") {
				continue
			}

			gobPath := filepath.Join(indexDir, entry.Name())
			idx, err := engine.LoadIndex(gobPath)
			if err != nil {
				continue
			}

			repoName := strings.TrimSuffix(entry.Name(), ".gob")

			var org string
			repoURL := idx.RepoURL
			if repoURL != "" {
				parts := strings.Split(strings.TrimSuffix(repoURL, "/"), "/")
				if len(parts) >= 2 {
					org = parts[len(parts)-2]
				}
			}

			repoPath := fmt.Sprintf("/tmp/codesearch/repos/%s/%s", org, repoName)

			repo := &github.Repo{
				Name:     repoName,
				RepoPath: repoPath,
				CloneURL: repoURL,
			}
			repos = append(repos, repo)
			results = append(results, fmt.Sprintf("%s (Files: %d, Commit: %s)",
				repoName, len(idx.FileBoundaries), idx.CommitHash))
		}

		return CachedReposMsg{Repos: repos, Results: results}
	}
}

type IndexProgressMsg struct {
	Processed int
	Total     int
	Percent   float64
}

func CloneAndIndex(repoURL string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Split(strings.TrimSuffix(strings.TrimSuffix(repoURL, "/"), ".git"), "/")
		if len(parts) < 2 {
			return IndexResultMsg{RepoName: repoURL, Err: fmt.Errorf("invalid repo url")}
		}
		org := parts[len(parts)-2]
		repoName := parts[len(parts)-1]

		repo := &github.Repo{
			Name:     repoName,
			RepoPath: fmt.Sprintf("/tmp/codesearch/repos/%s/%s", org, repoName),
			CloneURL: repoURL,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := github.CloneRepo(ctx, repo); err != nil {
			return IndexResultMsg{RepoName: repoName, Err: err}
		}

		progressChan <- IndexProgressMsg{Processed: 0, Total: 0, Percent: 0}

		idx, err := github.IndexRepoWithProgress(ctx, repo, func(processed, total int) {
			if total > 0 {
				pct := float64(processed) / float64(total)
				progressChan <- IndexProgressMsg{
					Processed: processed,
					Total:     total,
					Percent:   pct,
				}
			}
		})
		if err != nil {
			return IndexResultMsg{RepoName: repoName, Err: err}
		}

		return IndexResultMsg{
			RepoName: repoName,
			Result:   fmt.Sprintf("%s (Files: %d, Commit: %s)", repoName, len(idx.FileBoundaries), idx.CommitHash),
			Repo:     repo,
		}
	}
}

var progressChan = make(chan IndexProgressMsg, 100)

func WaitForProgress() tea.Cmd {
	return func() tea.Msg {
		return <-progressChan
	}
}

func SearchRepos(query string, repos []*github.Repo) tea.Cmd {
	return func() tea.Msg {
		var allResults []engine.SearchResult
		for _, repo := range repos {
			results, err := github.SearchRepo(repo, query)
			if err != nil {
				continue
			}
			allResults = append(allResults, results...)
		}
		return SearchResultMsg{
			Query:   query,
			Results: allResults,
		}
	}
}
