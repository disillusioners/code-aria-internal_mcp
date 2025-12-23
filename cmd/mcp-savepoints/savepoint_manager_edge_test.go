package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
)

func TestSavepointManagerEdgeCases(t *testing.T) {
	// Test with non-existent repository path
	t.Run("NonExistentRepoPath", func(t *testing.T) {
		os.Setenv("REPO_PATH", "/non/existent/path")
		defer os.Unsetenv("REPO_PATH")

		_, err := NewSavepointManager()
		if err == nil {
			t.Error("Expected error when creating manager with non-existent path")
		}
		if !strings.Contains(err.Error(), "repository not found") {
			t.Errorf("Expected repository not found error, got: %v", err)
		}
	})

	// Test with non-git directory
	t.Run("NonGitDirectory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "non-git-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		os.Setenv("REPO_PATH", tempDir)
		defer os.Unsetenv("REPO_PATH")

		_, err = NewSavepointManager()
		if err == nil {
			t.Error("Expected error when creating manager with non-git directory")
		}
	})

	// Test creating savepoint with special characters in name
	t.Run("SpecialCharactersInName", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "special-char-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

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

		specialName := "savepoint-with-special-chars-@#$%^&*()_+-=[]{}|;':\",./<>?"
		savepoint, err := manager.CreateSavepoint(specialName, "Testing special characters")
		if err != nil {
			t.Fatalf("Failed to create savepoint with special characters: %v", err)
		}

		if savepoint.Name != specialName {
			t.Errorf("Expected name with special chars, got: %s", savepoint.Name)
		}
	})

	// Test with very long description
	t.Run("LongDescription", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "long-desc-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

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

		longDesc := strings.Repeat("This is a very long description. ", 100)
		savepoint, err := manager.CreateSavepoint("test", longDesc)
		if err != nil {
			t.Fatalf("Failed to create savepoint with long description: %v", err)
		}

		if savepoint.Description != longDesc {
			t.Errorf("Description length mismatch. Expected %d, got %d", len(longDesc), len(savepoint.Description))
		}
	})

	// Test with binary files
	t.Run("BinaryFiles", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "binary-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		repo, err := git.PlainInit(tempDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create a binary file (simulated)
		binaryFile := filepath.Join(tempDir, "binary.bin")
		binaryData := make([]byte, 1024)
		for i := range binaryData {
			binaryData[i] = byte(i % 256)
		}
		err = os.WriteFile(binaryFile, binaryData, 0644)
		if err != nil {
			t.Fatalf("Failed to create binary file: %v", err)
		}

		w, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Failed to get worktree: %v", err)
		}
		_, err = w.Add("binary.bin")
		if err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}

		os.Setenv("REPO_PATH", tempDir)
		defer os.Unsetenv("REPO_PATH")

		manager, err := NewSavepointManager()
		if err != nil {
			t.Fatalf("Failed to create savepoint manager: %v", err)
		}

		savepoint, err := manager.CreateSavepoint("binary-test", "Testing binary files")
		if err != nil {
			t.Fatalf("Failed to create savepoint with binary file: %v", err)
		}

		// Verify binary file was copied correctly
		savepointPath := filepath.Join(manager.savepointDir, savepoint.ID)
		copiedFile := filepath.Join(savepointPath, "binary.bin")
		copiedData, err := os.ReadFile(copiedFile)
		if err != nil {
			t.Fatalf("Failed to read copied binary file: %v", err)
		}

		if len(copiedData) != len(binaryData) {
			t.Errorf("Binary file size mismatch. Expected %d, got %d", len(binaryData), len(copiedData))
		}
	})

	// Test with nested directories
	t.Run("NestedDirectories", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "nested-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		repo, err := git.PlainInit(tempDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create nested directory structure
		nestedDir := filepath.Join(tempDir, "dir1", "dir2", "dir3")
		err = os.MkdirAll(nestedDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create nested directories: %v", err)
		}

		nestedFile := filepath.Join(nestedDir, "nested.txt")
		err = os.WriteFile(nestedFile, []byte("nested content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create nested file: %v", err)
		}

		w, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Failed to get worktree: %v", err)
		}
		_, err = w.Add("dir1/dir2/dir3/nested.txt")
		if err != nil {
			t.Fatalf("Failed to add nested file: %v", err)
		}

		os.Setenv("REPO_PATH", tempDir)
		defer os.Unsetenv("REPO_PATH")

		manager, err := NewSavepointManager()
		if err != nil {
			t.Fatalf("Failed to create savepoint manager: %v", err)
		}

		savepoint, err := manager.CreateSavepoint("nested-test", "Testing nested directories")
		if err != nil {
			t.Fatalf("Failed to create savepoint with nested directories: %v", err)
		}

		// Verify nested structure was preserved
		savepointPath := filepath.Join(manager.savepointDir, savepoint.ID)
		copiedNestedFile := filepath.Join(savepointPath, "dir1", "dir2", "dir3", "nested.txt")
		if _, err := os.Stat(copiedNestedFile); os.IsNotExist(err) {
			t.Errorf("Nested file not found at expected location: %s", copiedNestedFile)
		}
	})

	// Test with permission variations
	t.Run("FilePermissions", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "perm-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		repo, err := git.PlainInit(tempDir, false)
		if err != nil {
			t.Fatalf("Failed to init git repo: %v", err)
		}

		// Create files with different permissions
		files := []struct {
			name string
			perm os.FileMode
		}{
			{"executable.sh", 0755},
			{"readonly.txt", 0444},
			{"private.json", 0600},
		}

		w, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Failed to get worktree: %v", err)
		}

		for _, file := range files {
			filePath := filepath.Join(tempDir, file.name)
			err = os.WriteFile(filePath, []byte("content"), file.perm)
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", file.name, err)
			}
			_, err = w.Add(file.name)
			if err != nil {
				t.Fatalf("Failed to add file %s: %v", file.name, err)
			}
		}

		os.Setenv("REPO_PATH", tempDir)
		defer os.Unsetenv("REPO_PATH")

		manager, err := NewSavepointManager()
		if err != nil {
			t.Fatalf("Failed to create savepoint manager: %v", err)
		}

		savepoint, err := manager.CreateSavepoint("perm-test", "Testing file permissions")
		if err != nil {
			t.Fatalf("Failed to create savepoint: %v", err)
		}

		// Verify permissions were preserved
		savepointPath := filepath.Join(manager.savepointDir, savepoint.ID)
		for _, file := range files {
			copiedFile := filepath.Join(savepointPath, file.name)
			info, err := os.Stat(copiedFile)
			if err != nil {
				t.Fatalf("Failed to stat copied file %s: %v", file.name, err)
			}

			// Note: On Windows, permission bits might not be preserved exactly
			// So we check if the file is executable on Unix-like systems
			if file.perm&0111 != 0 {
				if info.Mode()&0111 == 0 {
					t.Errorf("Executable bit not preserved for %s", file.name)
				}
			}
		}
	})
}

func TestSavepointConcurrentAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create multiple files for concurrent savepoint creation
	for i := 0; i < 10; i++ {
		fileName := filepath.Join(tempDir, "file"+string(rune('A'+i))+".txt")
		err = os.WriteFile(fileName, []byte("content "+string(rune('A'+i))), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	for i := 0; i < 10; i++ {
		fileName := "file" + string(rune('A'+i)) + ".txt"
		_, err = w.Add(fileName)
		if err != nil {
			t.Fatalf("Failed to add file %s: %v", fileName, err)
		}
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	// Create multiple savepoints concurrently
	resultChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			_, err := manager.CreateSavepoint(
				"concurrent-"+string(rune('A'+index)),
				"Concurrent savepoint "+string(rune('A'+index)),
			)
			resultChan <- err
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		if err := <-resultChan; err != nil {
			t.Errorf("Concurrent savepoint creation failed: %v", err)
		}
	}

	// Verify all savepoints were created
	savepoints, err := manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	if len(savepoints) != 10 {
		t.Errorf("Expected 10 savepoints, got %d", len(savepoints))
	}
}

func TestSavepointRestoreWithDeletions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "restore-delete-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create initial files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	for _, fileName := range files {
		filePath := filepath.Join(tempDir, fileName)
		err = os.WriteFile(filePath, []byte("original content of "+fileName), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fileName, err)
		}
		_, err = w.Add(fileName)
		if err != nil {
			t.Fatalf("Failed to add file %s: %v", fileName, err)
		}
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	// Create savepoint with original files
	savepoint, err := manager.CreateSavepoint("original", "Original state")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Modify files
	for _, fileName := range files {
		filePath := filepath.Join(tempDir, fileName)
		err = os.WriteFile(filePath, []byte("modified content of "+fileName), 0644)
		if err != nil {
			t.Fatalf("Failed to modify file %s: %v", fileName, err)
		}
	}

	// Delete one file
	err = os.Remove(filepath.Join(tempDir, "file2.txt"))
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Restore savepoint
	err = manager.RestoreSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	// Verify all files are restored with original content
	for _, fileName := range files {
		filePath := filepath.Join(tempDir, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read restored file %s: %v", fileName, err)
			continue
		}

		expected := "original content of " + fileName
		if string(content) != expected {
			t.Errorf("File %s content mismatch. Expected '%s', got '%s'", fileName, expected, string(content))
		}
	}
}

func TestSavepointTimestampConsistency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "timestamp-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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

	// Record time before creating savepoint
	before := time.Now()

	savepoint, err := manager.CreateSavepoint("timestamp-test", "Testing timestamps")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Record time after creating savepoint
	after := time.Now()

	// Parse savepoint timestamp
	savepointTime, err := time.Parse(time.RFC3339, savepoint.Timestamp)
	if err != nil {
		t.Fatalf("Failed to parse savepoint timestamp: %v", err)
	}

	// Verify timestamp is within expected range
	if savepointTime.Before(before) {
		t.Error("Savepoint timestamp is before creation time")
	}

	if savepointTime.After(after) {
		t.Error("Savepoint timestamp is after creation time")
	}
}

func TestSavepointWithSymlinks(t *testing.T) {
	// Skip on Windows as symlinks behave differently
	if os.PathSeparator == '\\' {
		t.Skip("Skipping symlink test on Windows")
	}

	tempDir, err := os.MkdirTemp("", "symlink-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create a target file
	targetFile := filepath.Join(tempDir, "target.txt")
	err = os.WriteFile(targetFile, []byte("target content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create a symlink
	symlinkPath := filepath.Join(tempDir, "symlink.txt")
	err = os.Symlink("target.txt", symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = w.Add("target.txt")
	if err != nil {
		t.Fatalf("Failed to add target file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint, err := manager.CreateSavepoint("symlink-test", "Testing symlinks")
	if err != nil {
		t.Fatalf("Failed to create savepoint with symlinks: %v", err)
	}

	// Verify symlink was copied as a file (not as a symlink)
	savepointPath := filepath.Join(manager.savepointDir, savepoint.ID)
	copiedSymlink := filepath.Join(savepointPath, "symlink.txt")

	// Check if the symlink was copied as a regular file
	info, err := os.Lstat(copiedSymlink)
	if err != nil {
		t.Fatalf("Failed to stat copied symlink: %v", err)
	}

	// The symlink should be copied as a regular file containing the target content
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("Symlink was copied as symlink instead of file content")
	}

	// Verify the content is the target's content
	content, err := os.ReadFile(copiedSymlink)
	if err != nil {
		t.Fatalf("Failed to read copied symlink: %v", err)
	}

	if string(content) != "target content" {
		t.Errorf("Expected symlink content 'target content', got '%s'", string(content))
	}
}

func TestSavepointWithIgnoredFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ignored-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create .gitignore file
	gitignoreContent := `
*.log
*.tmp
ignored/
.DS_Store
`
	err = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create files that should be ignored
	ignoredFiles := []string{"debug.log", "temp.tmp", "ignored/file.txt", ".DS_Store"}
	for _, fileName := range ignoredFiles {
		if fileName == "ignored/file.txt" {
			err = os.MkdirAll(filepath.Join(tempDir, "ignored"), 0755)
			if err != nil {
				t.Fatalf("Failed to create ignored directory: %v", err)
			}
		}
		filePath := filepath.Join(tempDir, fileName)
		err = os.WriteFile(filePath, []byte("ignored content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create ignored file %s: %v", fileName, err)
		}
	}

	// Create a file that should not be ignored
	trackedFile := filepath.Join(tempDir, "tracked.txt")
	err = os.WriteFile(trackedFile, []byte("tracked content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create tracked file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Add only the tracked file
	_, err = w.Add(".gitignore")
	if err != nil {
		t.Fatalf("Failed to add .gitignore: %v", err)
	}
	_, err = w.Add("tracked.txt")
	if err != nil {
		t.Fatalf("Failed to add tracked file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	savepoint, err := manager.CreateSavepoint("ignored-test", "Testing ignored files")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// Verify only tracked files are in savepoint
	if len(savepoint.Files) != 1 || savepoint.Files[0] != "tracked.txt" {
		t.Errorf("Expected only tracked.txt in savepoint, got: %v", savepoint.Files)
	}

	// Modify tracked file and create another savepoint
	err = os.WriteFile(trackedFile, []byte("modified tracked content"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify tracked file: %v", err)
	}

	savepoint2, err := manager.CreateSavepoint("ignored-test-2", "Testing with modifications")
	if err != nil {
		t.Fatalf("Failed to create second savepoint: %v", err)
	}

	// Verify still only tracked files are included
	if len(savepoint2.Files) != 1 || savepoint2.Files[0] != "tracked.txt" {
		t.Errorf("Expected only tracked.txt in second savepoint, got: %v", savepoint2.Files)
	}
}
