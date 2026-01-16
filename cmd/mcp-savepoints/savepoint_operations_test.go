package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestToolCreateSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-tool-test")
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

	// Test successful savepoint creation
	args := map[string]interface{}{
		"name":        "test-savepoint",
		"description": "A test savepoint",
	}

	result, err := toolCreateSavepoint(args)
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	var savepoint Savepoint
	err = json.Unmarshal([]byte(result), &savepoint)
	if err != nil {
		t.Fatalf("Failed to unmarshal savepoint result: %v", err)
	}

	if savepoint.Name != "test-savepoint" {
		t.Errorf("Expected name 'test-savepoint', got '%s'", savepoint.Name)
	}

	if savepoint.Description != "A test savepoint" {
		t.Errorf("Expected description 'A test savepoint', got '%s'", savepoint.Description)
	}

	// Test missing name
	args = map[string]interface{}{
		"description": "A test savepoint",
	}

	_, err = toolCreateSavepoint(args)
	if err == nil {
		t.Error("Expected error when name is missing")
	}

	if !containsString(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}

	// Test empty name
	args = map[string]interface{}{
		"name":        "",
		"description": "A test savepoint",
	}

	_, err = toolCreateSavepoint(args)
	if err == nil {
		t.Error("Expected error when name is empty")
	}
}

func TestToolListSavepoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-tool-test")
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

	args := map[string]interface{}{}

	// Test with no savepoints
	result, err := toolListSavepoints(args)
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	var savepoints []*Savepoint
	err = json.Unmarshal([]byte(result), &savepoints)
	if err != nil {
		t.Fatalf("Failed to unmarshal savepoints result: %v", err)
	}

	if len(savepoints) != 0 {
		t.Errorf("Expected 0 savepoints, got %d", len(savepoints))
	}

	// Create a savepoint
	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint1, err := manager.CreateSavepoint("test-1", "First savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Test with one savepoint
	result, err = toolListSavepoints(args)
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	err = json.Unmarshal([]byte(result), &savepoints)
	if err != nil {
		t.Fatalf("Failed to unmarshal savepoints result: %v", err)
	}

	if len(savepoints) != 1 {
		t.Errorf("Expected 1 savepoint, got %d", len(savepoints))
	}

	if savepoints[0].ID != savepoint1.ID {
		t.Errorf("Expected savepoint ID %s, got %s", savepoint1.ID, savepoints[0].ID)
	}
}

func TestToolGetSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-tool-test")
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

	// Create a savepoint
	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint1, err := manager.CreateSavepoint("test-1", "First savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Test getting existing savepoint
	args := map[string]interface{}{
		"savepoint_id": savepoint1.ID,
	}

	result, err := toolGetSavepoint(args)
	if err != nil {
		t.Fatalf("Failed to get savepoint: %v", err)
	}

	var savepoint Savepoint
	err = json.Unmarshal([]byte(result), &savepoint)
	if err != nil {
		t.Fatalf("Failed to unmarshal savepoint result: %v", err)
	}

	if savepoint.ID != savepoint1.ID {
		t.Errorf("Expected savepoint ID %s, got %s", savepoint1.ID, savepoint.ID)
	}

	// Test getting non-existent savepoint
	args = map[string]interface{}{
		"savepoint_id": "nonexistent",
	}

	_, err = toolGetSavepoint(args)
	if err == nil {
		t.Error("Expected error when getting non-existent savepoint")
	}

	// Test missing savepoint_id
	args = map[string]interface{}{}

	_, err = toolGetSavepoint(args)
	if err == nil {
		t.Error("Expected error when savepoint_id is missing")
	}

	if !containsString(err.Error(), "savepoint_id is required") {
		t.Errorf("Expected 'savepoint_id is required' error, got: %v", err)
	}
}

func TestToolRestoreSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-tool-test")
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

	// Create a savepoint
	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint1, err := manager.CreateSavepoint("test-1", "First savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Modify file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Test restoring existing savepoint
	args := map[string]interface{}{
		"savepoint_id": savepoint1.ID,
	}

	result, err := toolRestoreSavepoint(args)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", response["status"])
	}

	// Verify file was restored
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected restored content 'test content', got '%s'", string(content))
	}

	// Test restoring non-existent savepoint
	args = map[string]interface{}{
		"savepoint_id": "nonexistent",
	}

	_, err = toolRestoreSavepoint(args)
	if err == nil {
		t.Error("Expected error when restoring non-existent savepoint")
	}
}

func TestToolDeleteSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "savepoint-tool-test")
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

	// Create a savepoint
	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint1, err := manager.CreateSavepoint("test-1", "First savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Test deleting existing savepoint
	args := map[string]interface{}{
		"savepoint_id": savepoint1.ID,
	}

	result, err := toolDeleteSavepoint(args)
	if err != nil {
		t.Fatalf("Failed to delete savepoint: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", response["status"])
	}

	// Verify savepoint is deleted
	_, err = manager.GetSavepoint(savepoint1.ID)
	if err == nil {
		t.Error("Expected savepoint to be deleted")
	}

	// Test deleting non-existent savepoint
	args = map[string]interface{}{
		"savepoint_id": "nonexistent",
	}

	_, err = toolDeleteSavepoint(args)
	if err == nil {
		t.Error("Expected error when deleting non-existent savepoint")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
