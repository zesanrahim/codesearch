package github

import (
    "runtime"
	"codesearch/internal/database"
	"fmt"
	"codesearch/internal/engine"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"bufio"
    "context"
	"sort"
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
    filepath.Walk(repo.RepoPath, func(path string, info os.FileInfo, err error) error {
            return nil
        }
        totalFiles++
    })

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
            if onProgress != nil {
                onProgress(fileCount, totalFiles)
            }
            if fileCount%500 == 0 {
                writer.Flush()
            }
            countMutex.Unlock()
        }
    }()


        if info.Size() > 5*1024*1024 {
            return nil
        }

        if shouldIgnore(path) {
            return nil
        }

        fileChan <- path
    if err != nil {
        return nil, err
    }

    commitHash := idx.CommitHash
    repoURL := idx.RepoURL

    matchLineNums := idx.Search(query)
    var results []engine.SearchResult
numWorkers := runtime.NumCPU()
    for _, lineNum := range matchLineNums {
    
        if lineNum < 0 || lineNum >= len(idx.LineOffsets) {
            continue
        }
        byteOffset := idx.LineOffsets[lineNum]

if err := CloneRepo(ctx, repo); err != nil {
                    // This assumes 'errs' (e.g., `var errs []error`) and 'errsMutex' (e.g., `var errsMutex sync.Mutex`)
                    // are declared in the outer scope of MultiCloneRepos.
                    // The full fix also requires returning 'errs' at the end of MultiCloneRepos.
                    errsMutex.Lock()
                    errs = append(errs, fmt.Errorf("failed to clone %s: %w", repo.Name, err))
                    errsMutex.Unlock()
                }
        for _, boundary := range idx.FileBoundaries {
            if boundary.FilePath == filePath {
                fileStartOffset = boundary.StartOffset
                break
            }
        }

        
        offsetInFile := byteOffset - fileStartOffset

        
            Context:    contextStr,
            CommitHash: commitHash,
            RepoURL:    repoURL,
        }

        results = append(results, result)
    }

    return results, nil
}


func extractContext(content []byte, matchLine int) string {
    lines := strings.Split(string(content), "\n")
    const window = 10
    start := matchLine - 1 - window 
    if start < 0 {
        start = 0
    }
    end := matchLine - 1 + window + 1
    if end > len(lines) {
        end = len(lines)
    }
    return strings.Join(lines[start:end], "\n")
}

func loadGitignore() {
	gitignorePath := "/tmp/codesearch/repos/.gitignore"

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

func SearchRepoMultiLine(repo *Repo, codeLines []string) ([]engine.SearchResult, error) {
	var validLines []string
	for _, line := range codeLines {
		if strings.TrimSpace(line) != "" {
			validLines = append(validLines, line)
		}
	}

	if len(validLines) == 0 {
		return nil, fmt.Errorf("no valid lines in query")
	}

	idx, err := IndexRepo(context.Background(), repo)
	if err != nil {
		return nil, err
	}

	commitHash := idx.CommitHash
	repoURL := idx.RepoURL

	matchData := idx.SearchMultiple(validLines)

	var results []engine.SearchResult
	for lineNum, matchedInputIndices := range matchData {
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

		contextStr := extractContext(content, fileLine)

		matchCount := len(matchedInputIndices)
		consecutiveBonus := engine.CalculateConsecutiveBonus(matchedInputIndices)
		consecutiveScore := float64(consecutiveBonus) / float64(len(validLines)-1) * 10.0
		if len(validLines) <= 1 {
			consecutiveScore = 0
		}

		result := engine.SearchResult{
			FilePath:        filePath,
			Line:            fileLine,
			Offset:          byteOffset,
			Context:         contextStr,
			CommitHash:      commitHash,
			RepoURL:         repoURL,
			MatchedLines:    matchCount,
			TotalInputLines: len(validLines),
			ConsecutiveBonus: consecutiveScore,
		}
