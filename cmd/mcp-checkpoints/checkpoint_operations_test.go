package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestToolCreateCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-tool-test")
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

	// Test successful checkpoint creation
	args := map[string]interface{}{
		"name":        "test-checkpoint",
		"description": "A test checkpoint",
	}

	result, err := toolCreateCheckpoint(args)
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	var checkpoint Checkpoint
	err = json.Unmarshal([]byte(result), &checkpoint)
	if err != nil {
		t.Fatalf("Failed to unmarshal checkpoint result: %v", err)
	}

	if checkpoint.Name != "test-checkpoint" {
		t.Errorf("Expected name 'test-checkpoint', got '%s'", checkpoint.Name)
	}

	if checkpoint.Description != "A test checkpoint" {
		t.Errorf("Expected description 'A test checkpoint', got '%s'", checkpoint.Description)
	}

	// Test missing name
	args = map[string]interface{}{
		"description": "A test checkpoint",
	}

	_, err = toolCreateCheckpoint(args)
	if err == nil {
		t.Error("Expected error when name is missing")
	}

	if !containsString(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}

	// Test empty name
	args = map[string]interface{}{
		"name":        "",
		"description": "A test checkpoint",
	}

	_, err = toolCreateCheckpoint(args)
	if err == nil {
		t.Error("Expected error when name is empty")
	}
}

func TestToolListCheckpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-tool-test")
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

	// Test with no checkpoints
	result, err := toolListCheckpoints(args)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	var checkpoints []*Checkpoint
	err = json.Unmarshal([]byte(result), &checkpoints)
	if err != nil {
		t.Fatalf("Failed to unmarshal checkpoints result: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints, got %d", len(checkpoints))
	}

	// Create a checkpoint
	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint1, err := manager.CreateCheckpoint("test-1", "First checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Test with one checkpoint
	result, err = toolListCheckpoints(args)
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	err = json.Unmarshal([]byte(result), &checkpoints)
	if err != nil {
		t.Fatalf("Failed to unmarshal checkpoints result: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
	}

	if checkpoints[0].ID != checkpoint1.ID {
		t.Errorf("Expected checkpoint ID %s, got %s", checkpoint1.ID, checkpoints[0].ID)
	}
}

func TestToolGetCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-tool-test")
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

	// Create a checkpoint
	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint1, err := manager.CreateCheckpoint("test-1", "First checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Test getting existing checkpoint
	args := map[string]interface{}{
		"checkpoint_id": checkpoint1.ID,
	}

	result, err := toolGetCheckpoint(args)
	if err != nil {
		t.Fatalf("Failed to get checkpoint: %v", err)
	}

	var checkpoint Checkpoint
	err = json.Unmarshal([]byte(result), &checkpoint)
	if err != nil {
		t.Fatalf("Failed to unmarshal checkpoint result: %v", err)
	}

	if checkpoint.ID != checkpoint1.ID {
		t.Errorf("Expected checkpoint ID %s, got %s", checkpoint1.ID, checkpoint.ID)
	}

	// Test getting non-existent checkpoint
	args = map[string]interface{}{
		"checkpoint_id": "nonexistent",
	}

	_, err = toolGetCheckpoint(args)
	if err == nil {
		t.Error("Expected error when getting non-existent checkpoint")
	}

	// Test missing checkpoint_id
	args = map[string]interface{}{}

	_, err = toolGetCheckpoint(args)
	if err == nil {
		t.Error("Expected error when checkpoint_id is missing")
	}

	if !containsString(err.Error(), "checkpoint_id is required") {
		t.Errorf("Expected 'checkpoint_id is required' error, got: %v", err)
	}
}

func TestToolRestoreCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-tool-test")
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

	// Create a checkpoint
	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint1, err := manager.CreateCheckpoint("test-1", "First checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Modify file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Test restoring existing checkpoint
	args := map[string]interface{}{
		"checkpoint_id": checkpoint1.ID,
	}

	result, err := toolRestoreCheckpoint(args)
	if err != nil {
		t.Fatalf("Failed to restore checkpoint: %v", err)
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

	// Test restoring non-existent checkpoint
	args = map[string]interface{}{
		"checkpoint_id": "nonexistent",
	}

	_, err = toolRestoreCheckpoint(args)
	if err == nil {
		t.Error("Expected error when restoring non-existent checkpoint")
	}
}

func TestToolDeleteCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-tool-test")
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

	// Create a checkpoint
	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint1, err := manager.CreateCheckpoint("test-1", "First checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Test deleting existing checkpoint
	args := map[string]interface{}{
		"checkpoint_id": checkpoint1.ID,
	}

	result, err := toolDeleteCheckpoint(args)
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", response["status"])
	}

	// Verify checkpoint is deleted
	_, err = manager.GetCheckpoint(checkpoint1.ID)
	if err == nil {
		t.Error("Expected checkpoint to be deleted")
	}

	// Test deleting non-existent checkpoint
	args = map[string]interface{}{
		"checkpoint_id": "nonexistent",
	}

	_, err = toolDeleteCheckpoint(args)
	if err == nil {
		t.Error("Expected error when deleting non-existent checkpoint")
	}
}

func TestToolGetCheckpointInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "checkpoint-tool-test")
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

	// Create a checkpoint
	manager, err := NewCheckpointManager()
	if err != nil {
		t.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint1, err := manager.CreateCheckpoint("test-1", "First checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Test getting checkpoint info
	args := map[string]interface{}{
		"checkpoint_id": checkpoint1.ID,
	}

	result, err := toolGetCheckpointInfo(args)
	if err != nil {
		t.Fatalf("Failed to get checkpoint info: %v", err)
	}

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["checkpoint_id"] != checkpoint1.ID {
		t.Errorf("Expected checkpoint_id %s, got %v", checkpoint1.ID, response["checkpoint_id"])
	}

	if response["name"] != "test-1" {
		t.Errorf("Expected name 'test-1', got %v", response["name"])
	}

	if response["description"] != "First checkpoint" {
		t.Errorf("Expected description 'First checkpoint', got %v", response["description"])
	}

	fileCount, ok := response["file_count"].(float64)
	if !ok || int(fileCount) != 1 {
		t.Errorf("Expected file_count 1, got %v", response["file_count"])
	}

	files, ok := response["files"].([]interface{})
	if !ok || len(files) != 1 {
		t.Errorf("Expected 1 file in files array, got %v", response["files"])
	}

	if len(files) > 0 && files[0] != "test.txt" {
		t.Errorf("Expected file 'test.txt', got %v", files[0])
	}

	// Test getting info for non-existent checkpoint
	args = map[string]interface{}{
		"checkpoint_id": "nonexistent",
	}

	_, err = toolGetCheckpointInfo(args)
	if err == nil {
		t.Error("Expected error when getting info for non-existent checkpoint")
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