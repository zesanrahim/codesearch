package github

import (
	"bufio"
	"encoding/base64"

	"codesearch/internal/database"
	"codesearch/internal/engine"
	"codesearch/internal/paths"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

var (
	ignoredPatterns []string
	ignoreMutex     sync.RWMutex
	loadOnce        sync.Once
)

type Repo struct {
	Org      string
	Name     string
	RepoPath string
	CloneURL string
	// LastFetched time.Time
	// Size        int64
}

func NewRepo(cloneURL string) (*Repo, error) {
	org, name, err := ParseRepoURL(cloneURL)
	if err != nil {
		return nil, err
	}

	return &Repo{
		Org:      org,
		Name:     name,
		RepoPath: paths.RepoDir(org, name),
		CloneURL: cloneURL,
	}, nil
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

	if _, err := database.GetClient(); err != nil {
		return nil, fmt.Errorf("supabase client unavailable: %w", err)
	}

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

func MultiCloneRepos(ctx context.Context, repos []*Repo) error {
	if len(repos) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	workers := make(chan *Repo, len(repos))
	var wg sync.WaitGroup

	var (
		errs      []error
		errsMutex sync.Mutex
	)

	for w := 1; w <= 3; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range workers {
				if ctx.Err() != nil {
					return
				}
				if err := CloneRepo(ctx, repo); err != nil {
					errsMutex.Lock()
					errs = append(errs, fmt.Errorf("failed to clone %s: %w", repo.Name, err))
					errsMutex.Unlock()
				}
			}
		}()
	}

	for _, repo := range repos {
		workers <- repo
	}
	close(workers)
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return err
	}
	return errors.Join(errs...)
}

func CloneRepo(ctx context.Context, repo *Repo) error {
	return CloneRepoAuth(ctx, repo, "")
}

func CloneRepoAuth(ctx context.Context, repo *Repo, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := os.Stat(repo.RepoPath); !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Repo %s already exists at %s\n", repo.Name, repo.RepoPath)
		return nil
	}

	if err := os.MkdirAll(repo.RepoPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	args := []string{}
	if token != "" {
		auth := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
		args = append(args, "-c", "http.extraHeader=Authorization: Basic "+auth)
	}
	args = append(args, "clone", "--depth", "1", "--single-branch", repo.CloneURL, repo.RepoPath)

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(repo.RepoPath)
		if ctx.Err() != nil {
			return fmt.Errorf("clone aborted: %w", ctx.Err())
		}
		return fmt.Errorf("git clone failed: %w\nOutput: %s", err, scrub(string(output), token))
	}

	fmt.Fprintf(os.Stderr, "Successfully cloned repo %s\n", repo.Name)
	return nil
}

func scrub(s, token string) string {
	if token == "" {
		return s
	}
	return strings.ReplaceAll(s, token, "***")
}

func FetchRepo(repo *Repo) error {

	if _, err := os.Stat(repo.RepoPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Repo %s does not exists at %s\n", repo.Name, repo.RepoPath)
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

	fmt.Fprintf(os.Stderr, "Repo has been deleted")

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

func IndexMultiRepo(ctx context.Context, repos []*Repo) (map[string]*engine.Index, error) {
	results := make(map[string]*engine.Index)
	taskChan := make(chan *Repo, len(repos))
	var mu sync.Mutex
	var wg sync.WaitGroup
	numWorkers := runtime.NumCPU() * 2

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for r := range taskChan {
				select {
				case <-ctx.Done():
					return
				default:
					idx, err := IndexRepo(ctx, r)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to index repo %s: %v\n", r.Name, err)
						continue
					}

					mu.Lock()
					results[r.Name] = idx
					mu.Unlock()
				}
			}
		}()
	}

	for _, repo := range repos {
		taskChan <- repo
	}
	close(taskChan)

	wg.Wait()
	return results, ctx.Err()
}

func IndexRepo(ctx context.Context, repo *Repo) (*engine.Index, error) {
	return IndexRepoWithProgress(ctx, repo, nil)
}

func IndexRepoWithProgress(ctx context.Context, repo *Repo, onProgress func(processed, total int)) (*engine.Index, error) {

	cachedGob := paths.IndexFile(repo.Org, repo.Name)
	tempFile := paths.CorpusFile(repo.Org, repo.Name)

	for _, dir := range []string{filepath.Dir(cachedGob), filepath.Dir(tempFile)} {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	currentCommit, err := GetCurrentCommitHash(repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't get commit hash: %v\n", err)
	}

	if _, err := os.Stat(cachedGob); err == nil {
		fmt.Fprintf(os.Stderr, "Loading index from cache for repo %s\n", repo.Name)
		idx, err := engine.LoadIndex(cachedGob)
		if err == nil {

			if idx.CommitHash == currentCommit {
				return idx, nil
			}
			fmt.Fprintf(os.Stderr, "Commit changed, rebuilding index...\n")
		}
		fmt.Fprintf(os.Stderr, "Failed to load cached index: %v. Rebuilding...\n", err)
	}

	idx := &engine.Index{}

	file, err := os.Create(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	totalFiles := 0
	filepath.Walk(repo.RepoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || isBinaryFile(path) || info.Size() > 5*1024*1024 || shouldIgnore(path) {
			return nil
		}
		totalFiles++
		return nil
	})

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
			if onProgress != nil {
				onProgress(fileCount, totalFiles)
			}
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

	if err := idx.MapBoundaries(tempFile); err != nil {
		return nil, fmt.Errorf("failed to map corpus %s: %w", tempFile, err)
	}
	idx.BuildTrigrams()

	idx.FileBoundaries = fileBoundaries

	idx.CommitHash = currentCommit
	idx.RepoURL = GetRepoURL(repo)

	if err := idx.SaveIndex(cachedGob); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save index to cache: %v\n", err)
	}

	return idx, nil
}

func SearchRepo(repo *Repo, query string) ([]engine.SearchResult, error) {
	idx, err := IndexRepo(context.Background(), repo)
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

		contextStr := extractContext(content, fileLine)

		result := engine.SearchResult{
			FilePath:   filePath,
			Line:       fileLine,
			Offset:     byteOffset,
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
	gitignorePath := filepath.Join(paths.ReposDir(), ".gitignore")

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
			FilePath:         filePath,
			Line:             fileLine,
			Offset:           byteOffset,
			Context:          contextStr,
			CommitHash:       commitHash,
			RepoURL:          repoURL,
			MatchedLines:     matchCount,
			TotalInputLines:  len(validLines),
			ConsecutiveBonus: consecutiveScore,
		}

		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		scoreI := (float64(results[i].MatchedLines) / float64(results[i].TotalInputLines)) * 100
		scoreJ := (float64(results[j].MatchedLines) / float64(results[j].TotalInputLines)) * 100

		scoreI += results[i].ConsecutiveBonus
		scoreJ += results[j].ConsecutiveBonus

		if scoreI != scoreJ {
			return scoreI > scoreJ
		}

		if results[i].FilePath != results[j].FilePath {
			return results[i].FilePath < results[j].FilePath
		}
		return results[i].Line < results[j].Line
	})

	return results, nil
}
