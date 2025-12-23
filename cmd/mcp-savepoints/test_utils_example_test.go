package main

import (
	"os"
	"testing"
	"time"
)

// Example of how to use the test utilities
func TestExampleUsingTestUtils(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create some test files
	err := helper.CreateTestFiles(map[string]string{
		"main.go":     "package main\n\nfunc main() {}",
		"config.json": `{"debug": true}`,
		"README.md":   "# Test Project",
	})
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}

	// Verify files exist
	helper.AssertFileExists(t, "main.go")
	helper.AssertFileExists(t, "config.json")
	helper.AssertFileExists(t, "README.md")

	// Create a savepoint
	savepoint, err := helper.CreateSavepoint("initial", "Initial savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Verify savepoint count
	helper.AssertSavepointCount(t, 1)

	// Modify a file
	err = helper.ModifyTestFile("main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}")
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Create another savepoint
	_, err = helper.CreateSavepoint("modified", "Modified main.go")
	if err != nil {
		t.Fatalf("Failed to create second savepoint: %v", err)
	}

	helper.AssertSavepointCount(t, 2)

	// Restore the first savepoint
	err = helper.GetManager().RestoreSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	// Verify the content was restored
	helper.AssertFileContent(t, "main.go", "package main\n\nfunc main() {}")
}

func TestExampleComplexScenario(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Simulate a complex workday scenario
	err := helper.SimulateWorkday()
	if err != nil {
		t.Fatalf("Failed to simulate workday: %v", err)
	}

	// Create savepoint after workday
	savepoint, err := helper.CreateSavepoint("workday-end", "End of workday changes")
	if err != nil {
		t.Fatalf("Failed to create workday savepoint: %v", err)
	}

	// Print savepoint summary (useful for debugging)
	helper.PrintSavepointSummary()

	// Verify savepoint contains expected files
	if len(savepoint.Files) < 5 {
		t.Errorf("Expected at least 5 files in savepoint, got %d", len(savepoint.Files))
	}

	// Test restoration by making additional changes
	err = helper.ModifyTestFile("src/main.go", "package main\n\n// RESTORED\nfunc main() {}")
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Wait a bit to ensure file timestamps are different
	time.Sleep(10 * time.Millisecond)

	// Restore savepoint
	err = helper.GetManager().RestoreSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	// Verify the workday state was restored
	helper.AssertFileContent(t, "src/main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Enhanced World!\")\n}")
}

func TestExampleErrorHandling(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Try to restore a non-existent savepoint
	err := helper.GetManager().RestoreSavepoint("non-existent-id")
	if err == nil {
		t.Error("Expected error when restoring non-existent savepoint")
	}

	// Try to get a non-existent savepoint
	_, err = helper.GetManager().GetSavepoint("non-existent-id")
	if err == nil {
		t.Error("Expected error when getting non-existent savepoint")
	}

	// Try to delete a non-existent savepoint
	err = helper.GetManager().DeleteSavepoint("non-existent-id")
	if err == nil {
		t.Error("Expected error when deleting non-existent savepoint")
	}

	// Verify no savepoints were created
	helper.AssertSavepointCount(t, 0)
}

func TestExampleBinaryFiles(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create some binary files
	err := helper.CreateBinaryFile("image.png", 1024)
	if err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	err = helper.CreateBinaryFile("data.bin", 2048)
	if err != nil {
		t.Fatalf("Failed to create second binary file: %v", err)
	}

	// Create savepoint
	savepoint, err := helper.CreateSavepoint("binary-test", "Test with binary files")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Modify binary files
	err = helper.CreateBinaryFile("image.png", 2048) // Different size
	if err != nil {
		t.Fatalf("Failed to modify binary file: %v", err)
	}

	// Restore savepoint
	err = helper.GetManager().RestoreSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	// Verify file sizes were restored
	info, err := os.Stat(helper.GetTempDir() + "/image.png")
	if err != nil {
		t.Fatalf("Failed to stat restored file: %v", err)
	}

	if info.Size() != 1024 {
		t.Errorf("Expected file size 1024, got %d", info.Size())
	}
}

func TestExampleGitIntegration(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Create initial files
	err := helper.CreateTestFiles(map[string]string{
		"app.py":     "print('Hello, World!')",
		"README":     "Test Application",
		".gitignore": "*.pyc\n__pycache__/",
	})
	if err != nil {
		t.Fatalf("Failed to create files: %v", err)
	}

	// Create initial commit
	err = helper.CreateGitCommit("Initial commit")
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Make some changes
	err = helper.ModifyTestFile("app.py", "print('Hello, Enhanced World!')\nprint('Version 2.0')")
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	err = helper.CreateTestFile("requirements.txt", "flask==2.0.0", 0644)
	if err != nil {
		t.Fatalf("Failed to create requirements file: %v", err)
	}

	// Create savepoint before committing
	savepoint1, err := helper.CreateSavepoint("pre-commit", "Changes before commit")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Commit the changes
	err = helper.CreateGitCommit("Add features and requirements")
	if err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	// Make more changes
	err = helper.ModifyTestFile("app.py", "print('Hello, Enhanced World!')\nprint('Version 2.0')\nprint('With bug fixes')")
	if err != nil {
		t.Fatalf("Failed to modify file again: %v", err)
	}

	// Create another savepoint
	savepoint2, err := helper.CreateSavepoint("post-commit", "Changes after commit")
	if err != nil {
		t.Fatalf("Failed to create second savepoint: %v", err)
	}

	// Verify we have 2 savepoints
	helper.AssertSavepointCount(t, 2)

	// Verify savepoint2 exists
	if savepoint2 == nil {
		t.Fatal("savepoint2 should not be nil")
	}
	if savepoint2.ID == "" {
		t.Fatal("savepoint2 should have an ID")
	}

	// Restore to pre-commit state
	err = helper.GetManager().RestoreSavepoint(savepoint1.ID)
	if err != nil {
		t.Fatalf("Failed to restore to pre-commit: %v", err)
	}

	// Verify the state
	helper.AssertFileContent(t, "app.py", "print('Hello, Enhanced World!')\nprint('Version 2.0')")
	helper.AssertFileExists(t, "requirements.txt")
	helper.AssertFileNotExists(t, ".git") // Git directory should not be in savepoints
}
