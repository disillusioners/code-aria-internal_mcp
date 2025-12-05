package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// getWorkingDirDiff gets the diff between working directory and HEAD using go-git
func getWorkingDirDiff(repoPath, relPath string) (string, error) {
	// Open repository
	r, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	// Get HEAD reference
	headRef, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get HEAD commit
	headCommit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// Get HEAD tree
	headTree, err := headCommit.Tree()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	// Get worktree
	w, err := r.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get worktree status to see what files have changed
	status, err := w.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree status: %w", err)
	}

	// If a specific file path is provided, filter to that file
	if relPath != "" {
		// Check if this file has changes in worktree
		fileStatus, hasChanges := status[relPath]
		if !hasChanges || fileStatus.Worktree == ' ' {
			// File hasn't changed in worktree, return empty diff
			return "", nil
		}

		// Get the file from HEAD tree using go-git
		var headFile *object.File
		var headContent string
		headFile, err = headTree.File(relPath)
		if err == nil {
			headContent, err = headFile.Contents()
			if err != nil {
				return "", fmt.Errorf("failed to read HEAD file content: %w", err)
			}
		} else if err == object.ErrFileNotFound {
			// File doesn't exist in HEAD (new file)
			headContent = ""
		} else {
			return "", fmt.Errorf("failed to get file from HEAD tree: %w", err)
		}

		// Get the file from working directory
		fullPath := filepath.Join(repoPath, relPath)
		var workContent string
		if fileStatus.Worktree != 'D' {
			// File exists in working directory (not deleted)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return "", fmt.Errorf("failed to read working file: %w", err)
			}
			workContent = string(content)
		} else {
			// File is deleted in working directory
			workContent = ""
		}

		// Generate diff using go-git's objects for proper hash calculation
		return generateDiffWithGitObjects(relPath, headFile, headContent, workContent, r)
	}

	// If no specific file, get diff for all changed files
	var result strings.Builder
	for file, fileStatus := range status {
		// Skip if file is only in index (staged) but not in worktree
		if fileStatus.Worktree == ' ' {
			continue
		}

		fileDiff, err := getWorkingDirDiff(repoPath, file)
		if err != nil {
			continue
		}
		if fileDiff != "" {
			result.WriteString(fileDiff)
		}
	}

	return result.String(), nil
}

// generateDiffWithGitObjects generates a unified diff using go-git objects for proper hash calculation
func generateDiffWithGitObjects(filePath string, headFile *object.File, headContent, workContent string, r *git.Repository) (string, error) {
	if headContent == workContent {
		return "", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))

	// Calculate hashes using go-git's plumbing
	var headHash, workHash string
	if headFile != nil {
		headHash = headFile.Hash.String()[:7]
	} else {
		headHash = "0000000"
	}

	// Calculate hash for working directory content using go-git's plumbing
	if workContent == "" {
		workHash = "0000000"
	} else {
		// Create a blob object to calculate the proper git hash
		blob := &plumbing.MemoryObject{}
		blob.SetType(plumbing.BlobObject)
		_, err := blob.Write([]byte(workContent))
		if err != nil {
			return "", fmt.Errorf("failed to write blob content: %w", err)
		}
		workHashObj := blob.Hash()
		workHash = workHashObj.String()[:7]
	}

	// Determine file mode from HEAD if available
	fileMode := "100644"
	if headFile != nil {
		fileMode = fmt.Sprintf("%o", headFile.Mode)
	}

	// Write diff header
	if headContent == "" {
		result.WriteString(fmt.Sprintf("new file mode %s\n", fileMode))
		result.WriteString(fmt.Sprintf("index 0000000..%s\n", workHash))
		result.WriteString("--- /dev/null\n")
		result.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	} else if workContent == "" {
		result.WriteString(fmt.Sprintf("deleted file mode %s\n", fileMode))
		result.WriteString(fmt.Sprintf("index %s..0000000\n", headHash))
		result.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		result.WriteString("+++ /dev/null\n")
	} else {
		result.WriteString(fmt.Sprintf("index %s..%s\n", headHash, workHash))
		result.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		result.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	}

	// Generate the actual diff content
	headLines := strings.Split(headContent, "\n")
	workLines := strings.Split(workContent, "\n")

	// Simple unified diff format
	// In production, you'd use a proper diff algorithm (like Myers diff)
	result.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(headLines), len(workLines)))

	maxLen := len(headLines)
	if len(workLines) > maxLen {
		maxLen = len(workLines)
	}

	for i := 0; i < maxLen; i++ {
		if i < len(headLines) && i < len(workLines) {
			if headLines[i] != workLines[i] {
				result.WriteString(fmt.Sprintf("-%s\n", headLines[i]))
				result.WriteString(fmt.Sprintf("+%s\n", workLines[i]))
			} else {
				result.WriteString(fmt.Sprintf(" %s\n", headLines[i]))
			}
		} else if i < len(headLines) {
			result.WriteString(fmt.Sprintf("-%s\n", headLines[i]))
		} else if i < len(workLines) {
			result.WriteString(fmt.Sprintf("+%s\n", workLines[i]))
		}
	}

	return result.String(), nil
}

// formatPatch converts go-git patch objects to string format
func formatPatch(patch object.Changes) string {
	var result strings.Builder
	for _, change := range patch {
		patchStr, err := change.Patch()
		if err != nil {
			continue
		}
		result.WriteString(patchStr.String())
	}
	return result.String()
}
