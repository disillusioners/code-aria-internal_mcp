// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

func main() {
	// Create test directory
	testDir := "manual-test"
	if err := os.RemoveAll(testDir); err != nil {
		log.Printf("Warning: Could not remove existing test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Fatalf("Failed to create test dir: %v", err)
	}

	// Initialize git repo
	_, err := git.PlainInit(testDir, false)
	if err != nil {
		log.Fatalf("Failed to init git repo: %v", err)
	}

	// Create test files
	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}

	newFile := filepath.Join(testDir, "new.txt")
	err = os.WriteFile(newFile, []byte("new file content"), 0644)
	if err != nil {
		log.Fatalf("Failed to create new file: %v", err)
	}

	// Set environment
	os.Setenv("REPO_PATH", testDir)
	defer os.Unsetenv("REPO_PATH")

	fmt.Println("=== Manual MCP Checkpoints Test ===")

	// Test 1: Create checkpoint
	fmt.Println("\n1. Creating checkpoint...")
	manager, err := NewCheckpointManager()
	if err != nil {
		log.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	checkpoint, err := manager.CreateCheckpoint("manual-test", "Manual test checkpoint")
	if err != nil {
		log.Fatalf("Failed to create checkpoint: %v", err)
	}

	fmt.Printf("✓ Created checkpoint: %s\n", checkpoint.ID)
	fmt.Printf("  Name: %s\n", checkpoint.Name)
	fmt.Printf("  Description: %s\n", checkpoint.Description)
	fmt.Printf("  Files: %v\n", checkpoint.Files)

	// Test 2: List checkpoints
	fmt.Println("\n2. Listing checkpoints...")
	checkpoints, err := manager.ListCheckpoints()
	if err != nil {
		log.Fatalf("Failed to list checkpoints: %v", err)
	}

	fmt.Printf("✓ Found %d checkpoints\n", len(checkpoints))
	for _, cp := range checkpoints {
		fmt.Printf("  - %s: %s\n", cp.ID, cp.Name)
	}

	// Test 3: Modify files
	fmt.Println("\n3. Modifying files...")
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	if err != nil {
		log.Fatalf("Failed to modify test file: %v", err)
	}
	err = os.Remove(newFile)
	if err != nil {
		log.Fatalf("Failed to remove new file: %v", err)
	}
	fmt.Println("✓ Modified test.txt and removed new.txt")

	// Test 4: Restore checkpoint
	fmt.Println("\n4. Restoring checkpoint...")
	err = manager.RestoreCheckpoint(checkpoint.ID)
	if err != nil {
		log.Fatalf("Failed to restore checkpoint: %v", err)
	}
	fmt.Printf("✓ Restored checkpoint %s\n", checkpoint.ID)

	// Test 5: Verify restoration
	fmt.Println("\n5. Verifying restoration...")
	content, err := os.ReadFile(testFile)
	if err != nil {
		log.Fatalf("Failed to read test file: %v", err)
	}
	if string(content) != "initial content" {
		log.Fatalf("Expected 'initial content', got '%s'", string(content))
	}

	if _, err := os.Stat(newFile); os.IsNotExist(err) {
		log.Fatalf("Expected new.txt to be restored, but it doesn't exist")
	}

	fmt.Println("✓ Files restored correctly")

	// Test 6: Get checkpoint info
	fmt.Println("\n6. Getting checkpoint info...")
	args := map[string]interface{}{
		"checkpoint_id": checkpoint.ID,
	}

	result, err := toolGetCheckpointInfo(args)
	if err != nil {
		log.Fatalf("Failed to get checkpoint info: %v", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal([]byte(result), &info); err != nil {
		log.Fatalf("Failed to parse checkpoint info: %v", err)
	}

	fmt.Printf("✓ Checkpoint info retrieved:\n")
	fmt.Printf("  ID: %s\n", info["checkpoint_id"])
	fmt.Printf("  Name: %s\n", info["name"])
	fmt.Printf("  Description: %s\n", info["description"])
	fmt.Printf("  File count: %v\n", info["file_count"])

	// Test 7: Delete checkpoint
	fmt.Println("\n7. Deleting checkpoint...")
	args = map[string]interface{}{
		"checkpoint_id": checkpoint.ID,
	}

	result, err = toolDeleteCheckpoint(args)
	if err != nil {
		log.Fatalf("Failed to delete checkpoint: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		log.Fatalf("Failed to parse delete response: %v", err)
	}

	if response["status"] != "success" {
		log.Fatalf("Expected success status, got: %v", response["status"])
	}

	fmt.Printf("✓ Deleted checkpoint %s\n", checkpoint.ID)

	// Test 8: Verify deletion
	fmt.Println("\n8. Verifying deletion...")
	checkpoints, err = manager.ListCheckpoints()
	if err != nil {
		log.Fatalf("Failed to list checkpoints after deletion: %v", err)
	}

	if len(checkpoints) != 0 {
		log.Fatalf("Expected 0 checkpoints after deletion, got %d", len(checkpoints))
	}

	fmt.Println("✓ Checkpoint successfully deleted")

	fmt.Println("\n=== All Tests Passed! ===")
}