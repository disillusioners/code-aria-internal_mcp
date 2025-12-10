package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// detectRepositories detects version control repositories
func detectRepositories() ([]RepositoryInfo, error) {
	var repositories []RepositoryInfo

	// Start from current working directory
	startDir, err := os.Getwd()
	if err != nil {
		return repositories, err
	}

	// Look for repositories in current directory and parent directories
	repositories = scanDirectoryForRepos(startDir)

	// Also check REPO_PATH if set
	if repoPath := os.Getenv("REPO_PATH"); repoPath != "" && repoPath != startDir {
		if absPath, err := filepath.Abs(repoPath); err == nil {
			repos := scanDirectoryForRepos(absPath)
			// Merge with existing repos, avoiding duplicates
			for _, repo := range repos {
				found := false
				for _, existing := range repositories {
					if existing.Path == repo.Path {
						found = true
						break
					}
				}
				if !found {
					repositories = append(repositories, repo)
				}
			}
		}
	}

	return repositories, nil
}

// scanDirectoryForRepos scans a directory for version control repositories
func scanDirectoryForRepos(dir string) []RepositoryInfo {
	var repositories []RepositoryInfo

	// Check if current directory is a repository
	if repo := checkDirectoryRepo(dir); repo != nil {
		repositories = append(repositories, *repo)
	}

	// Check parent directories up to a reasonable limit
	currentDir := dir
	for i := 0; i < 5; i++ {
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached root
		}

		if repo := checkDirectoryRepo(parent); repo != nil {
			repositories = append(repositories, *repo)
		}
		currentDir = parent
	}

	return repositories
}

// checkDirectoryRepo checks if a directory contains a version control repository
func checkDirectoryRepo(dir string) *RepositoryInfo {
	// Check for Git repository
	if gitRepo := checkGitRepo(dir); gitRepo != nil {
		return gitRepo
	}

	// Check for SVN repository
	if svnRepo := checkSVNRepo(dir); svnRepo != nil {
		return svnRepo
	}

	// Check for Mercurial repository
	if hgRepo := checkMercurialRepo(dir); hgRepo != nil {
		return hgRepo
	}

	return nil
}

// checkGitRepo checks if directory is a Git repository
func checkGitRepo(dir string) *RepositoryInfo {
	gitDir := filepath.Join(dir, ".git")
	if stat, err := os.Stat(gitDir); err != nil || !stat.IsDir() {
		// Also check for git file (worktree)
		if stat, err := os.Stat(gitDir); err != nil || stat.IsDir() {
			return nil
		}
	}

	repo := &RepositoryInfo{
		Path: dir,
		Type: "git",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current branch
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		repo.Branch = strings.TrimSpace(stdout.String())
	}

	// Get current commit
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		repo.Commit = strings.TrimSpace(stdout.String())
	}

	// Get remote URL
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "config", "--get", "remote.origin.url")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		repo.RemoteURL = strings.TrimSpace(stdout.String())
	}

	// Get repository status
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		output := stdout.String()
		repo.Modified = len(output) > 0
		repo.Staged = strings.Contains(output, "M ") || strings.Contains(output, "A ")
		repo.Untracked = strings.Contains(output, "??")

		if len(output) == 0 {
			repo.Status = "clean"
		} else {
			repo.Status = "dirty"
		}
	}

	// Get last activity time
	if commit := repo.Commit; commit != "" {
		cmd = exec.CommandContext(ctx, "git", "-C", dir, "log", "-1", "--format=%ci", commit)
		stdout.Reset()
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			if timestamp, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(stdout.String())); err == nil {
				repo.LastActivity = timestamp
			}
		}
	}

	return repo
}

// checkSVNRepo checks if directory is an SVN repository
func checkSVNRepo(dir string) *RepositoryInfo {
	svnDir := filepath.Join(dir, ".svn")
	if _, err := os.Stat(svnDir); os.IsNotExist(err) {
		return nil
	}

	repo := &RepositoryInfo{
		Path: dir,
		Type: "svn",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get SVN info
	cmd := exec.CommandContext(ctx, "svn", "info", dir)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		output := stdout.String()
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "URL:") {
				repo.RemoteURL = strings.TrimSpace(strings.TrimPrefix(line, "URL:"))
			}
			if strings.HasPrefix(line, "Revision:") {
				repo.Commit = strings.TrimSpace(strings.TrimPrefix(line, "Revision:"))
			}
			if strings.HasPrefix(line, "Last Changed Date:") {
				dateStr := strings.TrimSpace(strings.TrimPrefix(line, "Last Changed Date:"))
				// Parse SVN date format
				if timestamp, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr[:25]); err == nil {
					repo.LastActivity = timestamp
				}
			}
		}
	}

	// Get SVN status
	cmd = exec.CommandContext(ctx, "svn", "status", dir)
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		output := stdout.String()
		repo.Modified = len(output) > 0
		if len(output) == 0 {
			repo.Status = "clean"
		} else {
			repo.Status = "dirty"
		}
	}

	return repo
}

// checkMercurialRepo checks if directory is a Mercurial repository
func checkMercurialRepo(dir string) *RepositoryInfo {
	hgDir := filepath.Join(dir, ".hg")
	if _, err := os.Stat(hgDir); os.IsNotExist(err) {
		return nil
	}

	repo := &RepositoryInfo{
		Path: dir,
		Type: "hg",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current branch
	cmd := exec.CommandContext(ctx, "hg", "-R", dir, "branch")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		repo.Branch = strings.TrimSpace(stdout.String())
	}

	// Get current commit
	cmd = exec.CommandContext(ctx, "hg", "-R", dir, "id", "-i")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		repo.Commit = strings.TrimSpace(stdout.String())
	}

	// Get default path (remote)
	cmd = exec.CommandContext(ctx, "hg", "-R", dir, "paths", "default")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		repo.RemoteURL = strings.TrimSpace(stdout.String())
	}

	// Get repository status
	cmd = exec.CommandContext(ctx, "hg", "-R", dir, "status")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		output := stdout.String()
		repo.Modified = len(output) > 0
		repo.Untracked = strings.Contains(output, "?")
		if len(output) == 0 {
			repo.Status = "clean"
		} else {
			repo.Status = "dirty"
		}
	}

	return repo
}

// getSystemRecommendations generates system-specific recommendations
func getSystemRecommendations(osInfo *OSInfo, hardwareInfo *HardwareInfo, devToolsInfo *DevelopmentToolsInfo) []string {
	var recommendations []string

	// OS-specific recommendations
	if osInfo != nil {
		switch strings.ToLower(osInfo.Name) {
		case "windows":
			recommendations = append(recommendations, "Consider using Windows Terminal for better shell experience")
			recommendations = append(recommendations, "Enable Windows Subsystem for Linux (WSL) for better Unix tool support")
			if devToolsInfo.PowerShell == nil || !devToolsInfo.PowerShell.Installed {
				recommendations = append(recommendations, "Install PowerShell 7+ for enhanced scripting capabilities")
			}
		case "linux":
			recommendations = append(recommendations, "Use a modern terminal emulator like Tilix, Alacritty, or Kitty")
			if osInfo.Distribution == "ubuntu" || osInfo.Distribution == "debian" {
				recommendations = append(recommendations, "Keep system updated: sudo apt update && sudo apt upgrade")
			}
		case "macos":
			recommendations = append(recommendations, "Use Homebrew for package management")
			recommendations = append(recommendations, "Consider installing iTerm2 for advanced terminal features")
		}
	}

	// Hardware-specific recommendations
	if hardwareInfo != nil {
		if hardwareInfo.Memory.UsagePercent > 80 {
			recommendations = append(recommendations, "High memory usage detected. Consider closing unused applications")
		}

		if hardwareInfo.CPU.Threads >= 8 {
			recommendations = append(recommendations, "Multi-core CPU detected. Parallel compilation enabled")
		}

		// Check disk space
		for _, storage := range hardwareInfo.Storage {
			if storage.UsagePercent > 90 {
				recommendations = append(recommendations, fmt.Sprintf("Low disk space on %s (%.1f%% used). Consider cleanup", storage.Mountpoint, storage.UsagePercent))
			}
		}
	}

	// Development tools recommendations
	if devToolsInfo != nil {
		missingTools := []string{}
		if devToolsInfo.Git == nil || !devToolsInfo.Git.Installed {
			missingTools = append(missingTools, "Git")
		}
		if devToolsInfo.Go == nil || !devToolsInfo.Go.Installed {
			missingTools = append(missingTools, "Go")
		}
		if devToolsInfo.Node == nil || !devToolsInfo.Node.Installed {
			missingTools = append(missingTools, "Node.js")
		}
		if devToolsInfo.Docker == nil || !devToolsInfo.Docker.Installed {
			missingTools = append(missingTools, "Docker")
		}

		if len(missingTools) > 0 {
			recommendations = append(recommendations, fmt.Sprintf("Consider installing: %s", strings.Join(missingTools, ", ")))
		}

		// Package manager recommendations
		if runtime.GOOS == "windows" {
			hasWindowsPkgMgr := false
			for _, pkg := range devToolsInfo.PackageMgrs {
				if pkg.Type == "chocolatey" || pkg.Type == "winget" || pkg.Type == "scoop" {
					hasWindowsPkgMgr = true
					break
				}
			}
			if !hasWindowsPkgMgr {
				recommendations = append(recommendations, "Consider installing a Windows package manager (Chocolatey, winget, or Scoop)")
			}
		} else if runtime.GOOS == "macos" {
			hasHomebrew := false
			for _, pkg := range devToolsInfo.PackageMgrs {
				if pkg.Type == "brew" {
					hasHomebrew = true
					break
				}
			}
			if !hasHomebrew {
				recommendations = append(recommendations, "Consider installing Homebrew for package management")
			}
		}
	}

	// General recommendations
	recommendations = append(recommendations, "Use MCP PowerShell server on Windows for better Windows command support")
	recommendations = append(recommendations, "Use MCP Bash server on Unix systems for Unix command support")

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System appears well-configured for development")
	}

	return recommendations
}