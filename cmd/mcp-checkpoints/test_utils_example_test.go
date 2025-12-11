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

	// Create a checkpoint
	checkpoint, err := helper.CreateCheckpoint("initial", "Initial checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Verify checkpoint count
	helper.AssertCheckpointCount(t, 1)

	// Modify a file
	err = helper.ModifyTestFile("main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}")
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Create another checkpoint
	_, err = helper.CreateCheckpoint("modified", "Modified main.go")
	if err != nil {
		t.Fatalf("Failed to create second checkpoint: %v", err)
	}

	helper.AssertCheckpointCount(t, 2)

	// Restore the first checkpoint
	err = helper.GetManager().RestoreCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore checkpoint: %v", err)
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

	// Create checkpoint after workday
	checkpoint, err := helper.CreateCheckpoint("workday-end", "End of workday changes")
	if err != nil {
		t.Fatalf("Failed to create workday checkpoint: %v", err)
	}

	// Print checkpoint summary (useful for debugging)
	helper.PrintCheckpointSummary()

	// Verify checkpoint contains expected files
	if len(checkpoint.Files) < 5 {
		t.Errorf("Expected at least 5 files in checkpoint, got %d", len(checkpoint.Files))
	}

	// Test restoration by making additional changes
	err = helper.ModifyTestFile("src/main.go", "package main\n\n// RESTORED\nfunc main() {}")
	if err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Wait a bit to ensure file timestamps are different
	time.Sleep(10 * time.Millisecond)

	// Restore checkpoint
	err = helper.GetManager().RestoreCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore checkpoint: %v", err)
	}

	// Verify the workday state was restored
	helper.AssertFileContent(t, "src/main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, Enhanced World!\")\n}")
}

func TestExampleErrorHandling(t *testing.T) {
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Try to restore a non-existent checkpoint
	err := helper.GetManager().RestoreCheckpoint("non-existent-id")
	if err == nil {
		t.Error("Expected error when restoring non-existent checkpoint")
	}

	// Try to get a non-existent checkpoint
	_, err = helper.GetManager().GetCheckpoint("non-existent-id")
	if err == nil {
		t.Error("Expected error when getting non-existent checkpoint")
	}

	// Try to delete a non-existent checkpoint
	err = helper.GetManager().DeleteCheckpoint("non-existent-id")
	if err == nil {
		t.Error("Expected error when deleting non-existent checkpoint")
	}

	// Verify no checkpoints were created
	helper.AssertCheckpointCount(t, 0)
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

	// Create checkpoint
	checkpoint, err := helper.CreateCheckpoint("binary-test", "Test with binary files")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Modify binary files
	err = helper.CreateBinaryFile("image.png", 2048) // Different size
	if err != nil {
		t.Fatalf("Failed to modify binary file: %v", err)
	}

	// Restore checkpoint
	err = helper.GetManager().RestoreCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore checkpoint: %v", err)
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
		"app.py":   "print('Hello, World!')",
		"README":   "Test Application",
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

	// Create checkpoint before committing
	checkpoint1, err := helper.CreateCheckpoint("pre-commit", "Changes before commit")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
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

	// Create another checkpoint
	checkpoint2, err := helper.CreateCheckpoint("post-commit", "Changes after commit")
	if err != nil {
		t.Fatalf("Failed to create second checkpoint: %v", err)
	}

	// Verify we have 2 checkpoints
	helper.AssertCheckpointCount(t, 2)

	// Restore to pre-commit state
	err = helper.GetManager().RestoreCheckpoint(checkpoint1.ID)
	if err != nil {
		t.Fatalf("Failed to restore to pre-commit: %v", err)
	}

	// Verify the state
	helper.AssertFileContent(t, "app.py", "print('Hello, Enhanced World!')\nprint('Version 2.0')")
	helper.AssertFileExists(t, "requirements.txt")
	helper.AssertFileNotExists(t, ".git") // Git directory should not be in checkpoints
}