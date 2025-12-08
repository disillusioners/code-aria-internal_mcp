package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// toolGetGitStatus returns the git status in porcelain format
func toolGetGitStatus(args map[string]interface{}) (string, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	// Open repository using go-git
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get status
	status, err := w.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	// Convert status to porcelain format
	var result strings.Builder
	for file, s := range status {
		// Convert worktree status to porcelain format
		// X = staged status, Y = unstaged status
		x := string(s.Staging)
		y := string(s.Worktree)

		// Handle untracked files
		if s.Worktree == '?' {
			x = "?"
			y = "?"
		}

		result.WriteString(fmt.Sprintf("%s%s %s\n", x, y, file))
	}

	return result.String(), nil
}

// toolGetFileDiff returns the diff for a specific file
func toolGetFileDiff(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	fullPath := resolvePath(filePath)
	relPath, _ := filepath.Rel(repoPath, fullPath)

	// Priority 1: Check if compare_working is true (uncommitted changes)
	if compareWorking, ok := args["compare_working"].(bool); ok && compareWorking {
		// Get diff between working directory and HEAD using go-git
		return getWorkingDirDiff(repoPath, relPath)
	} else if baseCommit, ok := args["base_commit"].(string); ok && baseCommit != "" {
		// Priority 2: Commit comparison (base_commit and optionally target_commit)
		targetCommit := "HEAD"
		if tc, ok := args["target_commit"].(string); ok && tc != "" {
			targetCommit = tc
		}

		// Get diff between commits using git command for now
		// TODO: Implement proper go-git diff for commits
		cmd := exec.Command("git", "diff", baseCommit, targetCommit, "--", relPath)
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get diff: %w\nOutput: %s", err, string(output))
		}
		return string(output), nil
	} else {
		// Priority 3: Branch comparison (current behavior)
		baseBranch := "main"
		if bb, ok := args["base_branch"].(string); ok && bb != "" {
			baseBranch = bb
		}

		// If base_branch is "HEAD" or empty, compare working directory
		if baseBranch == "HEAD" || baseBranch == "" {
			// Get diff between working directory and HEAD using go-git
			return getWorkingDirDiff(repoPath, relPath)
		} else {
			// Get diff between branches using git command for now
			// TODO: Implement proper go-git diff for branches
			cmd := exec.Command("git", "diff", baseBranch, "--", relPath)
			cmd.Dir = repoPath
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("failed to get diff: %w\nOutput: %s", err, string(output))
			}
			return string(output), nil
		}
	}
}

// toolGetCommitHistory returns the commit history for a file
func toolGetCommitHistory(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Open repository using go-git
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	fullPath := resolvePath(filePath)
	relPath, _ := filepath.Rel(repoPath, fullPath)

	// Get commit history using go-git
	var result []map[string]interface{}

	// Get HEAD reference to start from
	head, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get commit iterator
	commitIter, err := r.Log(&git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get commit history: %w", err)
	}

	// Iterate through commits
	err = commitIter.ForEach(func(c *object.Commit) error {
		// Check if file was modified in this commit
		fileChanged := false
		if len(relPath) > 0 {
			// Check if file was modified in this commit by checking file stats
			stats, err := c.Stats()
			if err == nil {
				for _, stat := range stats {
					if strings.Contains(stat.Name, relPath) {
						fileChanged = true
						break
					}
				}
			}
		} else {
			fileChanged = true // If no specific file, include all commits
		}

		if fileChanged {
			result = append(result, map[string]interface{}{
				"hash":    c.Hash.String(),
				"author":  c.Author.Name,
				"email":   c.Author.Email,
				"date":    c.Author.When.Format(time.RFC3339),
				"message": c.Message,
			})
		}

		// Stop if we reached the limit
		if len(result) >= limit {
			return fmt.Errorf("limit reached")
		}

		return nil
	})

	// Ignore "limit reached" error as it's expected
	if err != nil && err.Error() != "limit reached" {
		return "", fmt.Errorf("failed to iterate commits: %w", err)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal commit history: %w", err)
	}
	return string(jsonResult), nil
}

// toolGetChangedFiles returns the list of changed files
func toolGetChangedFiles(args map[string]interface{}) (string, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	comparisonType, ok := args["comparison_type"].(string)
	if !ok {
		return "", fmt.Errorf("comparison_type is required (branch, commits, working, last_commit)")
	}

	includeStatus := true
	if is, ok := args["include_status"].(bool); ok {
		includeStatus = is
	}

	switch comparisonType {
	case "working":
		return getChangedFilesWorking(repoPath, includeStatus)

	case "branch":
		baseBranch, ok := args["base_branch"].(string)
		if !ok || baseBranch == "" {
			return "", fmt.Errorf("base_branch is required for branch comparison")
		}

		targetBranch := ""
		if tb, ok := args["target_branch"].(string); ok && tb != "" {
			targetBranch = tb
		}

		return getChangedFilesBranches(repoPath, baseBranch, targetBranch, includeStatus)

	case "commits":
		baseCommit, ok := args["base_commit"].(string)
		if !ok || baseCommit == "" {
			return "", fmt.Errorf("base_commit is required for commit comparison")
		}

		targetCommit := "HEAD"
		if tc, ok := args["target_commit"].(string); ok && tc != "" {
			targetCommit = tc
		}

		return getChangedFilesCommits(repoPath, baseCommit, targetCommit, includeStatus)

	case "last_commit":
		return getChangedFilesCommits(repoPath, "HEAD~1", "HEAD", includeStatus)

	default:
		return "", fmt.Errorf("invalid comparison_type: %s (must be: branch, commits, working, last_commit)", comparisonType)
	}
}

// getChangedFilesWorking returns changed files in working directory using go-git
func getChangedFilesWorking(repoPath string, includeStatus bool) (string, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	var result []map[string]interface{}
	for file, s := range status {
		// Only include files with worktree changes (unstaged)
		if s.Worktree == ' ' {
			continue
		}

		entry := map[string]interface{}{
			"file_path": file,
		}

		if includeStatus {
			// Map worktree status to git status format
			statusStr := getStatusString(s.Worktree)
			entry["status"] = statusStr
		}

		result = append(result, entry)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal changed files: %w", err)
	}
	return string(jsonResult), nil
}

// getChangedFilesBranches returns changed files between two branches using go-git
func getChangedFilesBranches(repoPath, baseBranch, targetBranch string, includeStatus bool) (string, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get target branch commit (default to HEAD if not specified)
	var targetCommit *object.Commit
	if targetBranch == "" {
		head, err := r.Head()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}
		targetCommit, err = r.CommitObject(head.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD commit: %w", err)
		}
	} else {
		ref, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", targetBranch)), true)
		if err != nil {
			return "", fmt.Errorf("failed to get target branch reference: %w", err)
		}
		targetCommit, err = r.CommitObject(ref.Hash())
		if err != nil {
			return "", fmt.Errorf("failed to get target branch commit: %w", err)
		}
	}

	// Get base branch commit
	baseRef, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", baseBranch)), true)
	if err != nil {
		return "", fmt.Errorf("failed to get base branch reference: %w", err)
	}
	baseCommit, err := r.CommitObject(baseRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get base branch commit: %w", err)
	}

	return getChangedFilesFromCommits(baseCommit, targetCommit, includeStatus)
}

// getChangedFilesCommits returns changed files between two commits using go-git
func getChangedFilesCommits(repoPath, baseCommitStr, targetCommitStr string, includeStatus bool) (string, error) {
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve commit references
	baseCommit, err := resolveCommit(r, baseCommitStr)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base commit: %w", err)
	}

	targetCommit, err := resolveCommit(r, targetCommitStr)
	if err != nil {
		return "", fmt.Errorf("failed to resolve target commit: %w", err)
	}

	return getChangedFilesFromCommits(baseCommit, targetCommit, includeStatus)
}

// resolveCommit resolves a commit reference (hash, HEAD, HEAD~1, etc.) to a commit object
func resolveCommit(r *git.Repository, ref string) (*object.Commit, error) {
	// Handle special references
	if ref == "HEAD" {
		head, err := r.Head()
		if err != nil {
			return nil, err
		}
		return r.CommitObject(head.Hash())
	}

	// Handle HEAD~n syntax
	if strings.HasPrefix(ref, "HEAD~") {
		head, err := r.Head()
		if err != nil {
			return nil, err
		}
		commit, err := r.CommitObject(head.Hash())
		if err != nil {
			return nil, err
		}

		n := 1
		if len(ref) > 5 {
			_, err := fmt.Sscanf(ref, "HEAD~%d", &n)
			if err != nil {
				return nil, fmt.Errorf("invalid HEAD~ syntax: %s", ref)
			}
		}

		for i := 0; i < n; i++ {
			if len(commit.ParentHashes) == 0 {
				return nil, fmt.Errorf("commit has no parent")
			}
			commit, err = r.CommitObject(commit.ParentHashes[0])
			if err != nil {
				return nil, err
			}
		}

		return commit, nil
	}

	// Try as branch reference
	if refs, err := r.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", ref)), true); err == nil {
		return r.CommitObject(refs.Hash())
	}

	// Try as commit hash
	hash := plumbing.NewHash(ref)
	if hash != plumbing.ZeroHash {
		return r.CommitObject(hash)
	}

	return nil, fmt.Errorf("could not resolve commit reference: %s", ref)
}

// getChangedFilesFromCommits returns changed files between two commits using go-git
func getChangedFilesFromCommits(baseCommit, targetCommit *object.Commit, includeStatus bool) (string, error) {
	baseTree, err := baseCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get base tree: %w", err)
	}

	targetTree, err := targetCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get target tree: %w", err)
	}

	// Get diff between trees
	changes, err := object.DiffTree(baseTree, targetTree)
	if err != nil {
		return "", fmt.Errorf("failed to diff trees: %w", err)
	}

	var result []map[string]interface{}
	for _, change := range changes {
		entry := make(map[string]interface{})

		// Determine file path and status
		var filePath string
		var status string

		if change.From.Name != "" && change.To.Name != "" {
			// Modified or renamed
			filePath = change.To.Name
			if change.From.Name != change.To.Name {
				status = "R" // Renamed
			} else {
				status = "M" // Modified
			}
		} else if change.From.Name != "" {
			// Deleted
			filePath = change.From.Name
			status = "D"
		} else if change.To.Name != "" {
			// Added
			filePath = change.To.Name
			status = "A"
		} else {
			continue // Skip invalid changes
		}

		entry["file_path"] = filePath
		if includeStatus {
			entry["status"] = status
		}

		result = append(result, entry)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal changed files: %w", err)
	}
	return string(jsonResult), nil
}

// getStatusString converts git status code to string representation
func getStatusString(statusCode git.StatusCode) string {
	switch statusCode {
	case 'M':
		return "M" // Modified
	case 'A':
		return "A" // Added
	case 'D':
		return "D" // Deleted
	case 'R':
		return "R" // Renamed
	case 'C':
		return "C" // Copied
	case '?':
		return "?" // Untracked
	default:
		return string(statusCode)
	}
}

// parseDiffNameOutput parses the output of git diff --name-status or --name-only
func parseDiffNameOutput(output string, includeStatus bool) (string, error) {
	var result []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		entry := make(map[string]interface{})
		if includeStatus {
			// Format: STATUS\tfile_path
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				entry["status"] = parts[0]
				entry["file_path"] = parts[1]
			} else {
				entry["file_path"] = line
			}
		} else {
			entry["file_path"] = line
		}
		result = append(result, entry)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal changed files: %w", err)
	}
	return string(jsonResult), nil
}

// toolGetAllWorkingChanges returns all working directory changes with diffs
func toolGetAllWorkingChanges(args map[string]interface{}) (string, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	// Parse optional parameters
	includeStatus := true
	if is, ok := args["include_status"].(bool); ok {
		includeStatus = is
	}

	format := "unified"
	if f, ok := args["format"].(string); ok {
		format = f
	}

	var filePatterns []string
	if patterns, ok := args["file_patterns"].([]interface{}); ok {
		filePatterns = make([]string, len(patterns))
		for i, p := range patterns {
			if pattern, ok := p.(string); ok {
				filePatterns[i] = pattern
			}
		}
	}

	// Get changed files
	changedFilesJSON, err := getChangedFilesWorking(repoPath, includeStatus)
	if err != nil {
		return "", fmt.Errorf("failed to get changed files: %w", err)
	}

	var changedFiles []map[string]interface{}
	if err := json.Unmarshal([]byte(changedFilesJSON), &changedFiles); err != nil {
		return "", fmt.Errorf("failed to parse changed files: %w", err)
	}

	// Filter by file patterns if provided
	var filteredFiles []map[string]interface{}
	for _, file := range changedFiles {
		filePath := file["file_path"].(string)

		// Check if file matches any pattern
		if len(filePatterns) == 0 {
			filteredFiles = append(filteredFiles, file)
			continue
		}

		matched := false
		for _, pattern := range filePatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
				matched = true
				break
			}
			// Also support path patterns
			if matched, _ := filepath.Match(pattern, filePath); matched {
				matched = true
				break
			}
		}

		if matched {
			filteredFiles = append(filteredFiles, file)
		}
	}

	// Generate diffs for each file if format is "unified"
	var resultFiles []map[string]interface{}
	summary := map[string]interface{}{
		"total_files": 0,
		"modified":    0,
		"added":       0,
		"deleted":     0,
	}

	for _, file := range filteredFiles {
		filePath := file["file_path"].(string)
		status := ""
		if includeStatus {
			if s, ok := file["status"].(string); ok {
				status = s
			}
		}

		fileResult := map[string]interface{}{
			"file_path": filePath,
		}

		if includeStatus {
			fileResult["status"] = status
		}

		// Generate diff if format is "unified"
		if format == "unified" {
			diff, err := getWorkingDirDiff(repoPath, filePath)
			if err != nil {
				fileResult["diff"] = fmt.Sprintf("Error generating diff: %s", err.Error())
			} else {
				fileResult["diff"] = diff
			}
		}

		resultFiles = append(resultFiles, fileResult)

		// Update summary
		totalFiles := summary["total_files"].(int)
		summary["total_files"] = totalFiles + 1
		switch status {
		case "M":
			modified := summary["modified"].(int)
			summary["modified"] = modified + 1
		case "A":
			added := summary["added"].(int)
			summary["added"] = added + 1
		case "D":
			deleted := summary["deleted"].(int)
			summary["deleted"] = deleted + 1
		case "?":
			added := summary["added"].(int)
			summary["added"] = added + 1
		}
	}

	// Build final result
	finalResult := map[string]interface{}{
		"changed_files": resultFiles,
		"summary":       summary,
	}

	resultJSON, err := json.Marshal(finalResult)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
