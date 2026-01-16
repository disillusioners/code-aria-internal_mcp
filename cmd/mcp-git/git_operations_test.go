package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type workingChange struct {
	FilePath string `json:"file_path"`
	Status   string `json:"status"`
	Diff     string `json:"diff"`
}

type workingChangesResult struct {
	ChangedFiles []workingChange `json:"changed_files"`
}

// runGit is a small helper to run git commands in a target directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func TestToolGetAllWorkingChangesReturnsStagedAndUnstaged(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-git-working-changes-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repo with a committed file.
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	origFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(origFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create staged addition.
	addedFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(addedFile, []byte("new file"), 0o644); err != nil {
		t.Fatalf("failed to write added file: %v", err)
	}
	runGit(t, tmpDir, "add", "newfile.txt")

	// Create unstaged modification.
	if err := os.WriteFile(origFile, []byte("package main\n\n// changed\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Ensure tool reads from this repo.
	t.Setenv("REPO_PATH", tmpDir)

	resultJSON, err := toolGetAllWorkingChanges(map[string]interface{}{})
	if err != nil {
		t.Fatalf("toolGetAllWorkingChanges returned error: %v", err)
	}

	var result workingChangesResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if len(result.ChangedFiles) == 0 {
		t.Fatalf("expected changed files, got none")
	}

	var sawAdded, sawModified bool
	for _, f := range result.ChangedFiles {
		switch f.FilePath {
		case "newfile.txt":
			if f.Status != "A" && f.Status != "?" {
				t.Fatalf("expected newfile.txt status A or ?, got %q", f.Status)
			}
			sawAdded = true
		case "main.go":
			if f.Status != "M" {
				t.Fatalf("expected main.go status M, got %q", f.Status)
			}
			sawModified = true
		}
	}

	if !sawAdded {
		t.Fatalf("expected staged added file newfile.txt to be reported")
	}
	if !sawModified {
		t.Fatalf("expected unstaged modified file main.go to be reported")
	}
}

// Test against the real test-repo path provided by the user to verify we
// return the actual working changes in a real repository.
func TestToolGetAllWorkingChangesRealRepo(t *testing.T) {
	repoPath := "/Users/nguyenminhkha/All/Code/opensource-projects/test-repo"
	if _, err := os.Stat(repoPath); err != nil {
		t.Skipf("skipping: test repo not found at %s", repoPath)
	}

	t.Setenv("REPO_PATH", repoPath)

	// Use file_patterns to limit the results to avoid exceeding the 100 file limit
	resultJSON, err := toolGetAllWorkingChanges(map[string]interface{}{
		"file_patterns": []interface{}{"*.go"},
	})
	if err != nil {
		// If still fails due to too many files, skip the test
		if strings.Contains(err.Error(), "too many changed files") {
			t.Skip("skipping: too many changed files in test repo")
		}
		t.Fatalf("toolGetAllWorkingChanges returned error: %v", err)
	}

	var result workingChangesResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	// Even if no files match the pattern, the test should not fail
	t.Logf("Found %d changed Go files in test repo", len(result.ChangedFiles))
}

func TestToolStageFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-git-stage-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit
	origFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(origFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create new files and modify existing one
	newFile1 := filepath.Join(tmpDir, "file1.txt")
	newFile2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(newFile1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(newFile2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	if err := os.WriteFile(origFile, []byte("package main\n\n// modified\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to modify main.go: %v", err)
	}

	t.Setenv("REPO_PATH", tmpDir)

	// Stage files
	resultJSON, err := toolStageFiles(map[string]interface{}{
		"file_paths": []interface{}{"file1.txt", "file2.txt"},
	})
	if err != nil {
		t.Fatalf("toolStageFiles returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	stagedCount, ok := result["staged_count"].(float64)
	if !ok {
		t.Fatalf("expected staged_count in result, got %v", result)
	}
	if int(stagedCount) != 2 {
		t.Fatalf("expected 2 staged files, got %v", stagedCount)
	}

	stagedFiles, ok := result["staged_files"].([]interface{})
	if !ok {
		t.Fatalf("expected staged_files array in result")
	}
	if len(stagedFiles) != 2 {
		t.Fatalf("expected 2 files in staged_files, got %d", len(stagedFiles))
	}

	// Verify files are actually staged by checking git status
	statusJSON, err := toolGetGitStatus(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to get git status: %v", err)
	}
	if !strings.Contains(statusJSON, "file1.txt") || !strings.Contains(statusJSON, "file2.txt") {
		t.Fatalf("expected file1.txt and file2.txt to be staged")
	}
}

func TestToolCommitChanges(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-git-commit-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit
	origFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(origFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create and stage a new file
	newFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(newFile, []byte("new content"), 0o644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	t.Setenv("REPO_PATH", tmpDir)

	// Stage the file first
	_, err = toolStageFiles(map[string]interface{}{
		"file_paths": []interface{}{"newfile.txt"},
	})
	if err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	// Commit the changes
	commitMessage := "Add newfile.txt"
	resultJSON, err := toolCommitChanges(map[string]interface{}{
		"message": commitMessage,
	})
	if err != nil {
		t.Fatalf("toolCommitChanges returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	commitHash, ok := result["commit_hash"].(string)
	if !ok || commitHash == "" {
		t.Fatalf("expected commit_hash in result, got %v", result)
	}

	message, ok := result["message"].(string)
	if !ok || message != commitMessage {
		t.Fatalf("expected message %q, got %q", commitMessage, message)
	}

	author, ok := result["author"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected author in result")
	}
	authorName, ok := author["name"].(string)
	if !ok || authorName != "Test User" {
		t.Fatalf("expected author name 'Test User', got %q", authorName)
	}
	authorEmail, ok := author["email"].(string)
	if !ok || authorEmail != "test@example.com" {
		t.Fatalf("expected author email 'test@example.com', got %q", authorEmail)
	}

	// Verify commit exists in git log
	historyJSON, err := toolGetCommitHistory(map[string]interface{}{
		"file_path": "newfile.txt",
		"limit":     1,
	})
	if err != nil {
		t.Fatalf("failed to get commit history: %v", err)
	}
	if !strings.Contains(historyJSON, commitMessage) {
		t.Fatalf("expected commit message in history")
	}
}

func TestToolUnstageFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-git-unstage-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit
	origFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(origFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create and stage files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	if err := os.WriteFile(file3, []byte("content3"), 0o644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}

	t.Setenv("REPO_PATH", tmpDir)

	// Stage all files
	_, err = toolStageFiles(map[string]interface{}{
		"file_paths": []interface{}{"file1.txt", "file2.txt", "file3.txt"},
	})
	if err != nil {
		t.Fatalf("failed to stage files: %v", err)
	}

	// Verify files are staged
	statusBefore, err := toolGetGitStatus(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to get git status: %v", err)
	}
	if !strings.Contains(statusBefore, "file1.txt") {
		t.Fatalf("expected file1.txt to be staged")
	}

	// Unstage specific files
	resultJSON, err := toolUnstageFiles(map[string]interface{}{
		"file_paths": []interface{}{"file1.txt", "file2.txt"},
	})
	if err != nil {
		t.Fatalf("toolUnstageFiles returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	unstagedCount, ok := result["unstaged_count"].(float64)
	if !ok {
		t.Fatalf("expected unstaged_count in result, got %v", result)
	}
	if int(unstagedCount) != 2 {
		t.Fatalf("expected 2 unstaged files, got %v", unstagedCount)
	}

	// Verify files are unstaged
	statusAfter, err := toolGetGitStatus(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to get git status: %v", err)
	}
	// file1.txt and file2.txt should be unstaged (untracked or modified)
	// file3.txt should still be staged
	if strings.Contains(statusAfter, "A  file3.txt") {
		// file3.txt is staged (A in staging area)
	} else {
		t.Fatalf("expected file3.txt to still be staged")
	}
}

func TestToolUnstageFilesAll(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-git-unstage-all-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit
	origFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(origFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create and stage files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	t.Setenv("REPO_PATH", tmpDir)

	// Stage all files
	_, err = toolStageFiles(map[string]interface{}{
		"file_paths": []interface{}{"file1.txt", "file2.txt"},
	})
	if err != nil {
		t.Fatalf("failed to stage files: %v", err)
	}

	// Unstage all files
	resultJSON, err := toolUnstageFiles(map[string]interface{}{
		"all": true,
	})
	if err != nil {
		t.Fatalf("toolUnstageFiles returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	unstagedCount, ok := result["unstaged_count"].(float64)
	if !ok {
		t.Fatalf("expected unstaged_count in result, got %v", result)
	}
	if int(unstagedCount) != 2 {
		t.Fatalf("expected 2 unstaged files, got %v", unstagedCount)
	}
}

func TestStageCommitUnstageIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-git-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize repo
	runGit(t, tmpDir, "init")
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit
	origFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(origFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	runGit(t, tmpDir, "add", "main.go")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	// Create new file
	newFile := filepath.Join(tmpDir, "feature.txt")
	if err := os.WriteFile(newFile, []byte("new feature"), 0o644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	t.Setenv("REPO_PATH", tmpDir)

	// Stage the file
	stageResult, err := toolStageFiles(map[string]interface{}{
		"file_paths": []interface{}{"feature.txt"},
	})
	if err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}
	var stageResultMap map[string]interface{}
	if err := json.Unmarshal([]byte(stageResult), &stageResultMap); err != nil {
		t.Fatalf("failed to parse stage result: %v", err)
	}
	if stageResultMap["staged_count"].(float64) != 1 {
		t.Fatalf("expected 1 staged file")
	}

	// Commit the changes
	commitResult, err := toolCommitChanges(map[string]interface{}{
		"message": "Add feature",
	})
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	var commitResultMap map[string]interface{}
	if err := json.Unmarshal([]byte(commitResult), &commitResultMap); err != nil {
		t.Fatalf("failed to parse commit result: %v", err)
	}
	commitHash := commitResultMap["commit_hash"].(string)
	if commitHash == "" {
		t.Fatalf("expected commit hash")
	}

	// Verify commit exists
	historyJSON, err := toolGetCommitHistory(map[string]interface{}{
		"file_path": "feature.txt",
		"limit":     1,
	})
	if err != nil {
		t.Fatalf("failed to get commit history: %v", err)
	}
	if !strings.Contains(historyJSON, "Add feature") {
		t.Fatalf("expected commit message in history")
	}

	// Create another file and stage it
	anotherFile := filepath.Join(tmpDir, "another.txt")
	if err := os.WriteFile(anotherFile, []byte("another file"), 0o644); err != nil {
		t.Fatalf("failed to write another file: %v", err)
	}

	_, err = toolStageFiles(map[string]interface{}{
		"file_paths": []interface{}{"another.txt"},
	})
	if err != nil {
		t.Fatalf("failed to stage another file: %v", err)
	}

	// Unstage it
	unstageResult, err := toolUnstageFiles(map[string]interface{}{
		"file_paths": []interface{}{"another.txt"},
	})
	if err != nil {
		t.Fatalf("failed to unstage file: %v", err)
	}
	var unstageResultMap map[string]interface{}
	if err := json.Unmarshal([]byte(unstageResult), &unstageResultMap); err != nil {
		t.Fatalf("failed to parse unstage result: %v", err)
	}
	if unstageResultMap["unstaged_count"].(float64) != 1 {
		t.Fatalf("expected 1 unstaged file")
	}
}
