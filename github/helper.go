package github

import (
	"codesearch/engine"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func GetCurrentCommitHash(repo *Repo) (string, error) {
    cmd := exec.Command("git", "-C", repo.RepoPath, "rev-parse", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("git rev-parse failed: %w\nOutput: %s", err, string(output))
    }

    commitHash := strings.TrimSpace(string(output))
    return commitHash, nil
}

func GetRepoURL(repo *Repo) string {
    cmd := exec.Command("git", "-C", repo.RepoPath, "config", "--get", "remote.origin.url")
    output, err := cmd.Output()
    if err != nil {
        fmt.Printf("git config failed: %v\nOutput: %s", err, string(output))
        return ""
    }

    url := strings.TrimSpace(string(output))

    if strings.HasPrefix(url, "git@") {
        re := regexp.MustCompile(`git@github\.com:(.+)/(.+?)(?:\.git)?$`)
        if matches := re.FindStringSubmatch(url); matches != nil {
            return fmt.Sprintf("https://github.com/%s/%s", matches[1], matches[2])
        }
    }

    url = strings.TrimSuffix(url, ".git")
    return url
}

func CountLinesInFile(filePath string, offset int) (int, error) {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return 0, err
    }

    if offset > len(content) {
        offset = len(content)
    }
    if offset < 0 {
        offset = 0
    }

    lineCount := 1
    for i := 0; i < offset; i++ {
        if content[i] == '\n' {
            lineCount++
        }
    }

    return lineCount, nil
}

func GetFileFromOffset(offset int, boundaries []engine.FileBoundary) (string, error) {
    for _, boundary := range boundaries {
        if offset >= boundary.StartOffset && offset < boundary.EndOffset {
            return boundary.FilePath, nil
        }
    }
    return "", fmt.Errorf("offset %d not found in any file", offset)
}

func GetRelativePath(fullPath string, repoPath string) string {
    return strings.TrimPrefix(fullPath, repoPath+"/")
}

func IsGitRepo(path string) bool {
    cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
    err := cmd.Run()
    return err == nil
}

func GetBranchName(repo *Repo) (string, error) {
    cmd := exec.Command("git", "-C", repo.RepoPath, "rev-parse", "--abbrev-ref", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func GetCommitMessage(repo *Repo, commitHash string) (string, error) {
    cmd := exec.Command("git", "-C", repo.RepoPath, "log", "-1", "--pretty=%B", commitHash)
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func GetCommitAuthor(repo *Repo, commitHash string) (string, error) {
    cmd := exec.Command("git", "-C", repo.RepoPath, "log", "-1", "--pretty=%an", commitHash)
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func GetCommitDate(repo *Repo, commitHash string) (string, error) {
    cmd := exec.Command("git", "-C", repo.RepoPath, "log", "-1", "--pretty=%ai", commitHash)
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}