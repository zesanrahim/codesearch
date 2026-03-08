package github

import (
	"codesearch/database"
	"codesearch/engine"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"bytes"
	"bufio"
)

var (
	ignoredPatterns []string
	ignoreMutex sync.RWMutex
	loadOnce	sync.Once
)

type Repo struct {
	Name     string
	RepoPath string
	CloneURL string
	// LastFetched time.Time
	// Size        int64
}

var (
	
	repoCache = make(map[string]*Repo)
	cacheLock sync.RWMutex
)

func GetRepo(name string) (repo *Repo, err error) {

	cacheLock.RLock()
	if repo, exists := repoCache[name]; exists {
		cacheLock.RUnlock()
		return repo, nil
	}
	cacheLock.RUnlock()
	database.GetClient()

	var repoPath, cloneURL string

	// TODO: Add Schema's and get data accorrdingly
	// sql query once db has schema
	// err := client.DB("query")

	// if err != nill {
	// 	return nil
	// }

	repo = &Repo{
		Name:     name,
		RepoPath: repoPath,
		CloneURL: cloneURL,
	}

	cacheLock.Lock()
	repoCache[name] = repo
	cacheLock.Unlock()

	return repo, nil
}

func CloneRepo(repo *Repo) error {

	if _, err := os.Stat(repo.RepoPath); !os.IsNotExist(err) {
		fmt.Printf("Repo %s already exists at %s\n", repo.Name, repo.RepoPath)
		return nil
	}

	if err := os.MkdirAll(repo.RepoPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	cmd := exec.Command("git", "clone", repo.CloneURL, repo.RepoPath)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Successfully cloned repo %s\n", repo.Name)
	return nil

}

func FetchRepo(repo *Repo) error {

	if _, err := os.Stat(repo.RepoPath); os.IsNotExist(err) {
		fmt.Printf("Repo %s does not exists at %s\n", repo.Name, repo.RepoPath)
		return nil
	}
	cmd := exec.Command("git", "-C", repo.RepoPath, "fetch", "--prune")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git fetch failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

//	func checkRepo(repo *Repo) error {
//		// TODO : add git fsck command if needed
//		return nil
//	}
func DeleteRepo(repo *Repo) error {

	if _, err := os.Stat(repo.RepoPath); os.IsNotExist(err) {
		return fmt.Errorf("Repo %s does not exists", repo.RepoPath)

	}

	if err := os.RemoveAll(repo.RepoPath); err != nil {
		return fmt.Errorf("failed to delete repo directory: %w", err)
	}
	cacheLock.Lock()
	delete(repoCache, repo.Name)
	cacheLock.Unlock()

	fmt.Printf("Repo has been deleted")

	return nil
}



func isBinaryFile(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    binaryExts := map[string]bool{
        ".exe": true, ".dll": true, ".so": true, ".bin": true,
        ".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
        ".pdf": true, ".zip": true, ".tar": true, ".gz": true,
    }
    if binaryExts[ext] {
        return true
    }

    return false
}

func IndexRepo(repo *Repo) (*engine.Index, error) {
    idx := &engine.Index{}

	

    tempFile := fmt.Sprintf("/tmp/index_%s.txt", repo.Name)
    file, err := os.Create(tempFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    writer := bufio.NewWriterWithSize(file, 4*1024*1024) // 4 mb

    fileCount := 0

    err = filepath.Walk(repo.RepoPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }
		if info.Size() > 5*1024*1024 { // skip files larger than 5MB
			return nil
		}

		if isBinaryFile(path) {
			return nil
		}

        if info.IsDir() || shouldIgnore(path) {
            return nil
        }

        content, err := os.ReadFile(path)
        if err != nil {
            return nil
        }

        writer.Write(content)
        writer.WriteString("\n")
        fileCount++

        if fileCount%500 == 0 {
            writer.Flush()
        }
        return nil
    })

    if err != nil {
        return nil, err
    }

    writer.Flush()

    idx.MapBoundaries(tempFile)
    idx.BuildTrigrams()

    os.Remove(tempFile)

    return idx, nil
}

func SearchRepo(repo *Repo, query string) ([]int, error) {
	idx, err := IndexRepo(repo)
	if err != nil {
		return nil, err
	}

	results := idx.Search(query)
	return results, nil
}

func loadGitignore() {
	gitignorePath := "/tmp/repos/.gitignore"

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		// .gitignore doesn't exist, use defaults
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ignoreMutex.Lock()
		ignoredPatterns = append(ignoredPatterns, line)
		ignoreMutex.Unlock()
	}
}



func shouldIgnore(path string) bool {

	loadOnce.Do(func() {
        loadGitignore()
    })

    ignoreMutex.RLock()
    patterns := ignoredPatterns
    ignoreMutex.RUnlock()

    if len(patterns) == 0 {
        loadGitignore()
        ignoreMutex.RLock()
        patterns = ignoredPatterns
        ignoreMutex.RUnlock()
    }

    for _, pattern := range patterns {
        if strings.Contains(path, pattern) {
            return true
        }
    }

    // Default patterns if .gitignore doesn't exist
    defaultIgnored := []string{".git", "node_modules", ".DS_Store", "vendor", "dist", "build"}
    for _, pattern := range defaultIgnored {
        if strings.Contains(path, pattern) {
            return true
        }
    }

    return false
}
	