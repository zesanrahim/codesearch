package main

import (
	"context"
	"fmt"
	"os"

	"codesearch/internal/github"
)

const usage = `codesearch — trigram code search over cloned repositories.

Usage:
  codesearch index <repo-url>   Clone and index a repository
  codesearch search <query>     Search every indexed repository
  codesearch list               List indexed repositories
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "index":
		err = runIndex(os.Args[2:])
	case "search":
		err = runSearch(os.Args[2:])
	case "list":
		err = runList()
	case "help", "-h", "--help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runIndex(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("index takes exactly one repository URL")
	}

	repo, err := github.NewRepo(args[0])
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := github.CloneRepo(ctx, repo); err != nil {
		return fmt.Errorf("cloning %s: %w", repo.Name, err)
	}

	idx, err := github.IndexRepoWithProgress(ctx, repo, func(processed, total int) {
		if total > 0 {
			fmt.Fprintf(os.Stderr, "\rindexing %s: %d/%d files", repo.Name, processed, total)
		}
	})
	if err != nil {
		return fmt.Errorf("indexing %s: %w", repo.Name, err)
	}
	fmt.Fprintln(os.Stderr)

	fmt.Printf("indexed %s/%s (%d files, commit %s)\n",
		repo.Org, repo.Name, len(idx.FileBoundaries), idx.CommitHash)
	return nil
}

func runSearch(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("search takes exactly one query")
	}
	query := args[0]

	cached, err := github.ListCachedRepos()
	if err != nil {
		return err
	}
	if len(cached) == 0 {
		return fmt.Errorf("no repositories indexed yet; run: codesearch index <repo-url>")
	}

	total := 0
	for _, c := range cached {
		results, err := github.SearchRepo(c.Repo, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", c.Repo.Name, err)
			continue
		}

		for _, r := range results {
			fmt.Printf("%s/%s:%d\n", c.Repo.Name, r.FilePath, r.Line)
			if url := r.GetBlobURL(); url != "" {
				fmt.Printf("  %s\n", url)
			}
			total++
		}
	}

	fmt.Fprintf(os.Stderr, "%d match(es) for %q\n", total, query)
	return nil
}

func runList() error {
	cached, err := github.ListCachedRepos()
	if err != nil {
		return err
	}
	if len(cached) == 0 {
		fmt.Fprintln(os.Stderr, "no repositories indexed yet")
		return nil
	}

	for _, c := range cached {
		fmt.Printf("%s/%s\t%d files\tcommit %s\n",
			c.Repo.Org, c.Repo.Name, c.Files, c.CommitHash)
	}
	return nil
}
