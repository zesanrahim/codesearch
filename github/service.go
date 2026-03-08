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

	currentCommit, err := GetCurrentCommitHash(repo)
	if err != nil {
		fmt.Printf("Warning: couldn't get commit hash: %v\n", err)
	}

	cachedGob := fmt.Sprintf("/tmp/index_%s.gob", repo.Name)
	if _, err := os.Stat(cachedGob); err == nil {
		fmt.Printf("Loading index from cache for repo %s\n", repo.Name)
		idx, err := engine.LoadIndex(cachedGob)
		if err == nil {
		
			if idx.CommitHash == currentCommit {
				return idx, nil
			}
			fmt.Printf("Commit changed, rebuilding index...\n")
		}
		fmt.Printf("Failed to load cached index: %v. Rebuilding...\n", err)
	}

	idx := &engine.Index{}

    tempFile := fmt.Sprintf("/tmp/index_%s.txt", repo.Name)
    file, err := os.Create(tempFile)
    if err != nil {
        return nil, fmt.Errorf("failed to create temp file: %w", err)
    }
    defer file.Close()
    defer os.Remove(tempFile)

    writer := bufio.NewWriter(file)

    fileChan := make(chan string, 128)
    resultChan := make(chan struct {
        content  []byte
        filePath string
    }, 128)

    var wg sync.WaitGroup
    fileCount := 0
    countMutex := sync.Mutex{}

    var fileBoundaries []engine.FileBoundary
    boundaryMutex := sync.Mutex{}
    currentOffset := 0

    numWorkers := 8

    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for filePath := range fileChan {
                content, err := os.ReadFile(filePath)
                if err != nil {
                    continue
                }
                resultChan <- struct {
                    content  []byte
                    filePath string
                }{content, filePath}
            }
        }()
    }

    var writerWg sync.WaitGroup
    writerWg.Add(1)
    go func() {
        defer writerWg.Done()
        for result := range resultChan {
            fileStart := currentOffset
            writer.Write(result.content)
            writer.WriteString("\n")
            currentOffset += len(result.content) + 1

            boundaryMutex.Lock()
            relPath := GetRelativePath(result.filePath, repo.RepoPath)
            fileBoundaries = append(fileBoundaries, engine.FileBoundary{
                FilePath:    relPath,
                StartOffset: fileStart,
                EndOffset:   currentOffset,
            })
            boundaryMutex.Unlock()

            countMutex.Lock()
            fileCount++
            if fileCount%500 == 0 {
                writer.Flush()
            }
            countMutex.Unlock()
        }
    }()

    err = filepath.Walk(repo.RepoPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }

        if info.IsDir() {
            return nil
        }

        if isBinaryFile(path) {
            return nil
        }

        if info.Size() > 5*1024*1024 {
            return nil
        }

        if shouldIgnore(path) {
            return nil
        }

        fileChan <- path
        return nil
    })

    if err != nil {
        return nil, err
    }

    close(fileChan)
    wg.Wait()

    close(resultChan)
    writerWg.Wait()


    writer.Flush()

    idx.MapBoundaries(tempFile)
    idx.BuildTrigrams()

    idx.FileBoundaries = fileBoundaries


    idx.CommitHash = currentCommit
    idx.RepoURL = GetRepoURL(repo)

    if err := idx.SaveIndex(cachedGob); err != nil {
        fmt.Printf("Failed to save index to cache: %v\n", err)
    }

    return idx, nil
}

func SearchRepo(repo *Repo, query string) ([]engine.SearchResult, error) {
    idx, err := IndexRepo(repo)
    if err != nil {
        return nil, err
    }

    commitHash := idx.CommitHash
    repoURL := idx.RepoURL

    matchLineNums := idx.Search(query)
    var results []engine.SearchResult

    for _, lineNum := range matchLineNums {
    
        if lineNum < 0 || lineNum >= len(idx.LineOffsets) {
            continue
        }
        byteOffset := idx.LineOffsets[lineNum]


        filePath, err := GetFileFromOffset(byteOffset, idx.FileBoundaries)
        if err != nil {
            continue
        }

        
        var fileStartOffset int
        for _, boundary := range idx.FileBoundaries {
            if boundary.FilePath == filePath {
                fileStartOffset = boundary.StartOffset
                break
            }
        }

        
        offsetInFile := byteOffset - fileStartOffset

        
        absPath := filepath.Join(repo.RepoPath, filePath)
        content, err := os.ReadFile(absPath)
        if err != nil {
            continue
        }

        
        if offsetInFile > len(content) {
            offsetInFile = len(content)
        }
        if offsetInFile < 0 {
            offsetInFile = 0
        }

    
        fileLine := 1
        for i := 0; i < offsetInFile; i++ {
            if content[i] == '\n' {
                fileLine++
            }
        }

        result := engine.SearchResult{
            FilePath:   filePath,
            Line:       fileLine,
            Offset:     byteOffset,
            CommitHash: commitHash,
            RepoURL:    repoURL,
        }

        results = append(results, result)
    }

    return results, nil
}

func loadGitignore() {
	gitignorePath := "/tmp/repos/.gitignore"

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
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
	