package github

import (
	"codesearch/internal/engine"
	"fmt"
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

func ParseRepoURL(rawURL string) (org, name string, err error) {
	s := strings.TrimSpace(rawURL)
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, ".git")
	if s == "" {
		return "", "", fmt.Errorf("empty repository URL")
	}

	if i := strings.Index(s, "://"); i != -1 {
		s = s[i+len("://"):]
	}
	if strings.Contains(s, "@") {
		s = strings.Replace(s, ":", "/", 1)
	}

	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '/' })
	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot parse organization and repository from %q", rawURL)
	}

	return parts[len(parts)-2], parts[len(parts)-1], nil
}
