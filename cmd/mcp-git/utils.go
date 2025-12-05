package main

import (
	"os"
	"path/filepath"
)

// resolvePath resolves a file path relative to REPO_PATH or returns absolute path
func resolvePath(path string) string {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return path
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(repoPath, path)
}
