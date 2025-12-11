package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestHelper provides utilities for testing checkpoint functionality
type TestHelper struct {
	tempDir    string
	repo       *git.Repository
	manager    *CheckpointManager
	cleanup    func()
	checkpoints []*Checkpoint
}

// NewTestHelper creates a new test helper with a temporary git repository
func NewTestHelper(t *testing.T) *TestHelper {
	tempDir, err := os.MkdirTemp("", "checkpoint-helper-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set environment variable
	os.Setenv("REPO_PATH", tempDir)

	// Create checkpoint manager
	manager, err := NewCheckpointManager()
	if err != nil {
		os.Unsetenv("REPO_PATH")
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	cleanup := func() {
		os.Unsetenv("REPO_PATH")
		os.RemoveAll(tempDir)
	}

	return &TestHelper{
		tempDir: tempDir,
		repo:    repo,
		manager: manager,
		cleanup: cleanup,
	}
}

// Cleanup performs final cleanup
func (th *TestHelper) Cleanup() {
	// Clean up created checkpoints
	for _, cp := range th.checkpoints {
		th.manager.DeleteCheckpoint(cp.ID)
	}
	th.cleanup()
}

// GetTempDir returns the temporary directory path
func (th *TestHelper) GetTempDir() string {
	return th.tempDir
}

// GetManager returns the checkpoint manager
func (th *TestHelper) GetManager() *CheckpointManager {
	return th.manager
}

// GetRepo returns the git repository
func (th *TestHelper) GetRepo() *git.Repository {
	return th.repo
}

// CreateTestFile creates a test file with the given content
func (th *TestHelper) CreateTestFile(name, content string, perm os.FileMode) error {
	filePath := filepath.Join(th.tempDir, name)

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if dir != th.tempDir {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), perm)
	if err != nil {
		return fmt.Errorf("failed to create test file: %v", err)
	}

	// Add to git if it's a regular file (not in .git directory)
	if !strings.HasPrefix(name, ".git") {
		w, err := th.repo.Worktree()
		if err != nil {
			return fmt.Errorf("failed to get worktree: %v", err)
		}
		_, err = w.Add(name)
		if err != nil {
			return fmt.Errorf("failed to add file to git: %v", err)
		}
	}

	return nil
}

// CreateTestFiles creates multiple test files
func (th *TestHelper) CreateTestFiles(files map[string]string) error {
	for name, content := range files {
		if err := th.CreateTestFile(name, content, 0644); err != nil {
			return err
		}
	}
	return nil
}

// CreateBinaryFile creates a test binary file with the given size
func (th *TestHelper) CreateBinaryFile(name string, size int) error {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	filePath := filepath.Join(th.tempDir, name)
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to create binary file: %v", err)
	}

	w, err := th.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}
	_, err = w.Add(name)
	if err != nil {
		return fmt.Errorf("failed to add binary file to git: %v", err)
	}

	return nil
}

// ModifyTestFile modifies an existing test file
func (th *TestHelper) ModifyTestFile(name, content string) error {
	filePath := filepath.Join(th.tempDir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to modify test file: %v", err)
	}
	return nil
}

// DeleteTestFile deletes a test file
func (th *TestHelper) DeleteTestFile(name string) error {
	filePath := filepath.Join(th.tempDir, name)
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete test file: %v", err)
	}
	return nil
}

// ReadTestFile reads the content of a test file
func (th *TestHelper) ReadTestFile(name string) (string, error) {
	filePath := filepath.Join(th.tempDir, name)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read test file: %v", err)
	}
	return string(content), nil
}

// FileExists checks if a file exists
func (th *TestHelper) FileExists(name string) bool {
	filePath := filepath.Join(th.tempDir, name)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// CreateCheckpoint creates a checkpoint and tracks it for cleanup
func (th *TestHelper) CreateCheckpoint(name, description string) (*Checkpoint, error) {
	checkpoint, err := th.manager.CreateCheckpoint(name, description)
	if err != nil {
		return nil, err
	}
	th.checkpoints = append(th.checkpoints, checkpoint)
	return checkpoint, nil
}

// AssertFileContent asserts that a file has the expected content
func (th *TestHelper) AssertFileContent(t *testing.T, name, expectedContent string) {
	content, err := th.ReadTestFile(name)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", name, err)
	}
	if content != expectedContent {
		t.Errorf("File %s content mismatch. Expected '%s', got '%s'", name, expectedContent, content)
	}
}

// AssertFileExists asserts that a file exists
func (th *TestHelper) AssertFileExists(t *testing.T, name string) {
	if !th.FileExists(name) {
		t.Errorf("File %s should exist but doesn't", name)
	}
}

// AssertFileNotExists asserts that a file does not exist
func (th *TestHelper) AssertFileNotExists(t *testing.T, name string) {
	if th.FileExists(name) {
		t.Errorf("File %s should not exist but does", name)
	}
}

// AssertCheckpointCount asserts the number of checkpoints
func (th *TestHelper) AssertCheckpointCount(t *testing.T, expectedCount int) {
	checkpoints, err := th.manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}
	if len(checkpoints) != expectedCount {
		t.Errorf("Expected %d checkpoints, got %d", expectedCount, len(checkpoints))
	}
}

// WaitForFile waits for a file to appear (useful for async operations)
func (th *TestHelper) WaitForFile(t *testing.T, name string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		if th.FileExists(name) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("File %s did not appear within timeout", name)
}

// CreateComplexFileStructure creates a complex file structure for testing
func (th *TestHelper) CreateComplexFileStructure() error {
	// Create nested directories
	dirs := []string{
		"src",
		"src/utils",
		"src/components",
		"docs",
		"tests",
		"config",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(th.tempDir, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create files with various content types
	files := map[string]string{
		"README.md":           "# Test Project\n\nThis is a test project.",
		"src/main.go":         "package main\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}",
		"src/utils/helpers.go": "package utils\n\nfunc Helper() string { return \"helper\" }",
		"src/components/ui.go": "package components\n\ntype UI struct { Name string }",
		"docs/api.md":          "# API Documentation\n\nEndpoints...",
		"tests/main_test.go":   "package tests\n\nfunc TestMain(t *testing.T) { t.Log(\"test\") }",
		"config/config.json":   `{"debug": true, "port": 8080}`,
		".gitignore":           "*.log\n*.tmp\nbuild/",
	}

	for name, content := range files {
		if err := th.CreateTestFile(name, content, 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %v", name, err)
		}
	}

	// Create some executable files
	executables := map[string]string{
		"build.sh": "#!/bin/bash\necho 'Building...'\n",
		"run.sh":   "#!/bin/bash\necho 'Running...'\n",
	}

	for name, content := range executables {
		if err := th.CreateTestFile(name, content, 0755); err != nil {
			return fmt.Errorf("failed to create executable %s: %v", name, err)
		}
	}

	return nil
}

// CreateGitCommit creates a git commit with current changes
func (th *TestHelper) CreateGitCommit(message string) error {
	w, err := th.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get status: %v", err)
	}

	// Add all changes
	for file := range status {
		_, err := w.Add(file)
		if err != nil {
			return fmt.Errorf("failed to add file %s: %v", file, err)
		}
	}

	// Create commit
	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create commit: %v", err)
	}

	return nil
}

// SimulateWorkday simulates a typical workday with file changes
func (th *TestHelper) SimulateWorkday() error {
	// Morning: Create initial structure
	if err := th.CreateComplexFileStructure(); err != nil {
		return fmt.Errorf("failed to create initial structure: %v", err)
	}

	// Morning commit
	if err := th.CreateGitCommit("Initial project structure"); err != nil {
		return fmt.Errorf("failed to create morning commit: %v", err)
	}

	// Mid-day: Make some changes
	changes := map[string]string{
		"src/main.go":           "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Enhanced World!\")\n}",
		"src/utils/helpers.go":  "package utils\n\nimport \"strings\"\n\nfunc Helper() string { return strings.ToUpper(\"helper\") }",
		"config/config.json":    `{"debug": false, "port": 9090, "env": "production"}`,
		"docs/api.md":           "# API Documentation\n\n## Endpoints\n\nGET /api/users\nPOST /api/users",
	}

	for name, content := range changes {
		if err := th.ModifyTestFile(name, content); err != nil {
			return fmt.Errorf("failed to modify file %s: %v", name, err)
		}
	}

	// Create new files
	newFiles := map[string]string{
		"src/services/api.go":   "package services\n\ntype APIClient struct { URL string }",
		"docs/README.md":        "# Documentation\n\nSee api.md for API docs.",
		"scripts/deploy.sh":     "#!/bin/bash\necho 'Deploying...'\ndocker build .",
	}

	for name, content := range newFiles {
		if err := th.CreateTestFile(name, content, 0644); err != nil {
			return fmt.Errorf("failed to create new file %s: %v", name, err)
		}
	}

	return nil
}

// GetCheckpointDiskUsage calculates total disk usage of all checkpoints
func (th *TestHelper) GetCheckpointDiskUsage() (int64, error) {
	var totalSize int64

	checkpointDir := th.manager.checkpointDir
	err := filepath.Walk(checkpointDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// PrintCheckpointSummary prints a summary of all checkpoints
func (th *TestHelper) PrintCheckpointSummary() {
	checkpoints, err := th.manager.ListCheckpoints()
	if err != nil {
		fmt.Printf("Error listing checkpoints: %v\n", err)
		return
	}

	fmt.Printf("Checkpoint Summary (%d checkpoints):\n", len(checkpoints))
	for i, cp := range checkpoints {
		fmt.Printf("  %d. %s (%s) - %s\n", i+1, cp.Name, cp.ID, cp.Timestamp)
		fmt.Printf("     Files: %d, Size: %d bytes\n", len(cp.Files), cp.Size)
	}

	diskUsage, err := th.GetCheckpointDiskUsage()
	if err == nil {
		fmt.Printf("Total disk usage: %d bytes (%.2f MB)\n", diskUsage, float64(diskUsage)/(1024*1024))
	}
}