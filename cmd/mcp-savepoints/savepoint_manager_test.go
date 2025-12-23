package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestNewSavepointManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "savepoint-test")
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

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	if manager.repoPath != tempDir {
		t.Errorf("Expected repoPath %s, got %s", tempDir, manager.repoPath)
	}

	expectedSavepointDir := filepath.Join(tempDir, ".mcp-savepoints")
	if manager.savepointDir != expectedSavepointDir {
		t.Errorf("Expected savepointDir %s, got %s", expectedSavepointDir, manager.savepointDir)
	}
}

func TestCreateSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-test")
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

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint, err := manager.CreateSavepoint("test-savepoint", "A test savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Verify savepoint metadata
	if savepoint.Name != "test-savepoint" {
		t.Errorf("Expected name 'test-savepoint', got '%s'", savepoint.Name)
	}

	if savepoint.Description != "A test savepoint" {
		t.Errorf("Expected description 'A test savepoint', got '%s'", savepoint.Description)
	}

	if len(savepoint.ID) != 8 {
		t.Errorf("Expected 8-character ID, got '%s'", savepoint.ID)
	}

	if len(savepoint.Files) == 0 {
		t.Error("Expected at least one file in savepoint")
	}

	// Verify savepoint directory exists
	savepointPath := filepath.Join(manager.savepointDir, savepoint.ID)
	if _, err := os.Stat(savepointPath); os.IsNotExist(err) {
		t.Errorf("Savepoint directory does not exist: %s", savepointPath)
	}

	// Verify file was copied to savepoint
	savepointFile := filepath.Join(savepointPath, "test.txt")
	if _, err := os.Stat(savepointFile); os.IsNotExist(err) {
		t.Errorf("Savepoint file does not exist: %s", savepointFile)
	}

	// Verify file content
	content, err := os.ReadFile(savepointFile)
	if err != nil {
		t.Fatalf("Failed to read savepoint file: %v", err)
	}

	expectedContent := "modified content"
	if string(content) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
	}

	// Verify database contains savepoint metadata
	// (Metadata is now stored in database, not a file)
	savepointInfo, err := manager.GetSavepoint(savepoint.ID)
	if err != nil {
		t.Errorf("Failed to get savepoint from database: %v", err)
	}
	if savepointInfo == nil {
		t.Error("Savepoint not found in database")
	}
}

func TestCreateSavepointNoChanges(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-test")
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

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	_, err = manager.CreateSavepoint("test-savepoint", "A test savepoint")
	if err == nil {
		t.Error("Expected error when creating savepoint with no changes")
	}

	if !strings.Contains(err.Error(), "no changes to savepoint") {
		t.Errorf("Expected 'no changes to savepoint' error, got: %v", err)
	}
}

func TestListSavepoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-test")
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

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	// Initially should be empty
	savepoints, err := manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	if len(savepoints) != 0 {
		t.Errorf("Expected 0 savepoints, got %d", len(savepoints))
	}

	// Create a savepoint
	savepoint1, err := manager.CreateSavepoint("savepoint-1", "First savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Should have one savepoint
	savepoints, err = manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	if len(savepoints) != 1 {
		t.Errorf("Expected 1 savepoint, got %d", len(savepoints))
	}

	if savepoints[0].ID != savepoint1.ID {
		t.Errorf("Expected savepoint ID %s, got %s", savepoint1.ID, savepoints[0].ID)
	}

	// Create another savepoint
	err = os.WriteFile(testFile, []byte("different content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	_, err = manager.CreateSavepoint("savepoint-2", "Second savepoint")
	if err != nil {
		t.Fatalf("Failed to create second savepoint: %v", err)
	}

	// Should have two savepoints
	savepoints, err = manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	if len(savepoints) != 2 {
		t.Errorf("Expected 2 savepoints, got %d", len(savepoints))
	}
}

func TestRestoreSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-test")
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

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	// Modify file and create savepoint
	modifiedContent := "modified content"
	err = os.WriteFile(testFile, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	savepoint, err := manager.CreateSavepoint("test-savepoint", "Test savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
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

	// Restore savepoint
	err = manager.RestoreSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	// Verify savepoint content was restored
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file after restore: %v", err)
	}
	if string(content) != modifiedContent {
		t.Errorf("Expected restored content '%s', got '%s'", modifiedContent, string(content))
	}
}

func TestDeleteSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-test")
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

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint, err := manager.CreateSavepoint("test-savepoint", "Test savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Verify savepoint exists
	savepointPath := filepath.Join(manager.savepointDir, savepoint.ID)
	if _, err := os.Stat(savepointPath); os.IsNotExist(err) {
		t.Errorf("Savepoint directory does not exist: %s", savepointPath)
	}

	// Delete savepoint
	err = manager.DeleteSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to delete savepoint: %v", err)
	}

	// Verify savepoint directory was deleted
	if _, err := os.Stat(savepointPath); !os.IsNotExist(err) {
		t.Errorf("Savepoint directory still exists after deletion: %s", savepointPath)
	}

	// Verify savepoint is no longer in list
	savepoints, err := manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	if len(savepoints) != 0 {
		t.Errorf("Expected 0 savepoints after deletion, got %d", len(savepoints))
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
