package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestNewCheckpointManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "checkpoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize a git repository
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set REPO_PATH for the test
	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	if manager.repoPath != tempDir {
		t.Errorf("Expected repoPath %s, got %s", tempDir, manager.repoPath)
	}

	expectedCheckpointDir := filepath.Join(tempDir, ".mcp-checkpoints")
	if manager.checkpointDir != expectedCheckpointDir {
		t.Errorf("Expected checkpointDir %s, got %s", expectedCheckpointDir, manager.checkpointDir)
	}
}

func TestCreateCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository and create some changes
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get worktree and add file
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	// Modify the file to create working changes
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint, err := manager.CreateCheckpoint("test-checkpoint", "A test checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Verify checkpoint metadata
	if checkpoint.Name != "test-checkpoint" {
		t.Errorf("Expected name 'test-checkpoint', got '%s'", checkpoint.Name)
	}

	if checkpoint.Description != "A test checkpoint" {
		t.Errorf("Expected description 'A test checkpoint', got '%s'", checkpoint.Description)
	}

	if len(checkpoint.ID) != 8 {
		t.Errorf("Expected 8-character ID, got '%s'", checkpoint.ID)
	}

	if len(checkpoint.Files) == 0 {
		t.Error("Expected at least one file in checkpoint")
	}

	// Verify checkpoint directory exists
	checkpointPath := filepath.Join(manager.checkpointDir, checkpoint.ID)
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Errorf("Checkpoint directory does not exist: %s", checkpointPath)
	}

	// Verify file was copied to checkpoint
	checkpointFile := filepath.Join(checkpointPath, "test.txt")
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		t.Errorf("Checkpoint file does not exist: %s", checkpointFile)
	}

	// Verify file content
	content, err := os.ReadFile(checkpointFile)
	if err != nil {
		t.Fatalf("Failed to read checkpoint file: %v", err)
	}

	expectedContent := "modified content"
	if string(content) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
	}

	// Verify metadata file exists
	metadataFile := filepath.Join(checkpointPath, METADATA_FILE)
	if _, err := os.Stat(metadataFile); os.IsNotExist(err) {
		t.Errorf("Metadata file does not exist: %s", metadataFile)
	}
}

func TestCreateCheckpointNoChanges(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	_, err = manager.CreateCheckpoint("test-checkpoint", "A test checkpoint")
	if err == nil {
		t.Error("Expected error when creating checkpoint with no changes")
	}

	if !strings.Contains(err.Error(), "no changes to checkpoint") {
		t.Errorf("Expected 'no changes to checkpoint' error, got: %v", err)
	}
}

func TestListCheckpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository and create changes
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	// Initially should be empty
	checkpoints, err := manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints, got %d", len(checkpoints))
	}

	// Create a checkpoint
	checkpoint1, err := manager.CreateCheckpoint("checkpoint-1", "First checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Should have one checkpoint
	checkpoints, err = manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
	}

	if checkpoints[0].ID != checkpoint1.ID {
		t.Errorf("Expected checkpoint ID %s, got %s", checkpoint1.ID, checkpoints[0].ID)
	}

	// Create another checkpoint
	err = os.WriteFile(testFile, []byte("different content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	_, err = manager.CreateCheckpoint("checkpoint-2", "Second checkpoint")
	if err != nil {
		t.Fatalf("Failed to create second checkpoint: %v", err)
	}

	// Should have two checkpoints
	checkpoints, err = manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 2 {
		t.Errorf("Expected 2 checkpoints, got %d", len(checkpoints))
	}
}

func TestRestoreCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository and create initial state
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	initialContent := "initial content"
	err = os.WriteFile(testFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	// Modify file and create checkpoint
	modifiedContent := "modified content"
	err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	checkpoint, err := manager.CreateCheckpoint("test-checkpoint", "Test checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Modify file again
	againModifiedContent := "again modified content"
	err = os.WriteFile(testFile, []byte(againModifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file again: %v", err)
	}

	// Verify current content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != againModifiedContent {
		t.Errorf("Expected content '%s', got '%s'", againModifiedContent, string(content))
	}

	// Restore checkpoint
	err = manager.RestoreCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore checkpoint: %v", err)
	}

	// Verify checkpoint content was restored
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file after restore: %v", err)
	}
	if string(content) != modifiedContent {
		t.Errorf("Expected restored content '%s', got '%s'", modifiedContent, string(content))
	}
}

func TestDeleteCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository and create changes
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint, err := manager.CreateCheckpoint("test-checkpoint", "Test checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Verify checkpoint exists
	checkpointPath := filepath.Join(manager.checkpointDir, checkpoint.ID)
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		t.Errorf("Checkpoint directory does not exist: %s", checkpointPath)
	}

	// Delete checkpoint
	err = manager.DeleteCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify checkpoint directory was deleted
	if _, err := os.Stat(checkpointPath); !os.IsNotExist(err) {
		t.Errorf("Checkpoint directory still exists after deletion: %s", checkpointPath)
	}

	// Verify checkpoint is no longer in list
	checkpoints, err := manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints after deletion, got %d", len(checkpoints))
	}
}

func TestGenerateID(t *testing.T) {
	id1, err := generateID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if len(id1) != 8 {
		t.Errorf("Expected 8-character ID, got %d characters: %s", len(id1), id1)
	}

	// Generate another ID
	id2, err := generateID()
	if err != nil {
		t.Fatalf("Failed to generate second ID: %v", err)
	}

	// IDs should be different
	if id1 == id2 {
		t.Errorf("Generated IDs should be unique, but got the same: %s", id1)
	}

	// IDs should be lowercase
	if id1 != strings.ToLower(id1) {
		t.Errorf("ID should be lowercase, got: %s", id1)
	}

	if id2 != strings.ToLower(id2) {
		t.Errorf("ID should be lowercase, got: %s", id2)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "copy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcFile := filepath.Join(tempDir, "src.txt")
	dstFile := filepath.Join(tempDir, "dst.txt")

	content := "test file content with some unicode: 🚀"
	err = os.WriteFile(srcFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	size, err := copyFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	if size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), size)
	}

	// Verify content
	copiedContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != content {
		t.Errorf("Expected content '%s', got '%s'", content, string(copiedContent))
	}

	// Verify file permissions
	srcInfo, err := os.Stat(srcFile)
	if err != nil {
		t.Fatalf("Failed to stat source file: %v", err)
	}

	dstInfo, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("Failed to stat destination file: %v", err)
	}

	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("File permissions not preserved. Expected %v, got %v", srcInfo.Mode(), dstInfo.Mode())
	}
}