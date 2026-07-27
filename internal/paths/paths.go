package paths

import (
	"os"
	"path/filepath"
)

const EnvCacheDir = "CODESEARCH_CACHE_DIR"

func Root() string {
	if dir := os.Getenv(EnvCacheDir); dir != "" {
		return dir
	}
	if dir, err := os.UserCacheDir(); err == nil {
		return filepath.Join(dir, "codesearch")
	}
	return filepath.Join(os.TempDir(), "codesearch")
}

func ReposDir() string { return filepath.Join(Root(), "repos") }

func RepoDir(org, name string) string {
	return filepath.Join(ReposDir(), org, name)
}

func IndexDir() string { return filepath.Join(Root(), "index") }

func IndexFile(org, name string) string {
	return filepath.Join(IndexDir(), org, name+".gob")
}

func CorpusDir() string { return filepath.Join(Root(), "corpus") }

func CorpusFile(org, name string) string {
	return filepath.Join(CorpusDir(), org, name+".txt")
}
