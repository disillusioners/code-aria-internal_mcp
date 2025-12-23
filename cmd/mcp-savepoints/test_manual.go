//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
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

	fmt.Println("=== Manual MCP Savepoints Test ===")

	// Test 1: Create savepoint
	fmt.Println("\n1. Creating savepoint...")
	manager, err := NewSavepointManager()
	if err != nil {
		log.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint, err := manager.CreateSavepoint("manual-test", "Manual test savepoint")
	if err != nil {
		log.Fatalf("Failed to create savepoint: %v", err)
	}

	fmt.Printf("✓ Created savepoint: %s\n", savepoint.ID)
	fmt.Printf("  Name: %s\n", savepoint.Name)
	fmt.Printf("  Description: %s\n", savepoint.Description)
	fmt.Printf("  Files: %v\n", savepoint.Files)

	// Test 2: List savepoints
	fmt.Println("\n2. Listing savepoints...")
	savepoints, err := manager.ListSavepoints()
	if err != nil {
		log.Fatalf("Failed to list savepoints: %v", err)
	}

	fmt.Printf("✓ Found %d savepoints\n", len(savepoints))
	for _, cp := range savepoints {
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

	// Test 4: Restore savepoint
	fmt.Println("\n4. Restoring savepoint...")
	err = manager.RestoreSavepoint(savepoint.ID)
	if err != nil {
		log.Fatalf("Failed to restore savepoint: %v", err)
	}
	fmt.Printf("✓ Restored savepoint %s\n", savepoint.ID)

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

	// Test 6: Get savepoint info
	fmt.Println("\n6. Getting savepoint info...")
	args := map[string]interface{}{
		"savepoint_id": savepoint.ID,
	}

	result, err := toolGetSavepointInfo(args)
	if err != nil {
		log.Fatalf("Failed to get savepoint info: %v", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal([]byte(result), &info); err != nil {
		log.Fatalf("Failed to parse savepoint info: %v", err)
	}

	fmt.Printf("✓ Savepoint info retrieved:\n")
	fmt.Printf("  ID: %s\n", info["savepoint_id"])
	fmt.Printf("  Name: %s\n", info["name"])
	fmt.Printf("  Description: %s\n", info["description"])
	fmt.Printf("  File count: %v\n", info["file_count"])

	// Test 7: Delete savepoint
	fmt.Println("\n7. Deleting savepoint...")
	args = map[string]interface{}{
		"savepoint_id": savepoint.ID,
	}

	result, err = toolDeleteSavepoint(args)
	if err != nil {
		log.Fatalf("Failed to delete savepoint: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		log.Fatalf("Failed to parse delete response: %v", err)
	}

	if response["status"] != "success" {
		log.Fatalf("Expected success status, got: %v", response["status"])
	}

	fmt.Printf("✓ Deleted savepoint %s\n", savepoint.ID)

	// Test 8: Verify deletion
	fmt.Println("\n8. Verifying deletion...")
	savepoints, err = manager.ListSavepoints()
	if err != nil {
		log.Fatalf("Failed to list savepoints after deletion: %v", err)
	}

	if len(savepoints) != 0 {
		log.Fatalf("Expected 0 savepoints after deletion, got %d", len(savepoints))
	}

	fmt.Println("✓ Savepoint successfully deleted")

	fmt.Println("\n=== All Tests Passed! ===")
}
