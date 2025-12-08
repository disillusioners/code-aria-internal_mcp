package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

	resultJSON, err := toolGetAllWorkingChanges(map[string]interface{}{})
	if err != nil {
		t.Fatalf("toolGetAllWorkingChanges returned error: %v", err)
	}

	var result workingChangesResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		t.Fatalf("failed to parse result JSON: %v", err)
	}

	if len(result.ChangedFiles) == 0 {
		t.Fatalf("expected changed files in test repo %s, got none", repoPath)
	}
}
