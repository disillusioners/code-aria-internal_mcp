package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

// BenchmarkCreateSavepoint benchmarks savepoint creation with different scenarios
func BenchmarkCreateSavepoint(b *testing.B) {
	scenarios := []struct {
		name     string
		numFiles int
		fileSize int
		nested   bool
	}{
		{"Small_Files", 10, 1024, false},
		{"Medium_Files", 100, 10240, false},
		{"Large_Files", 10, 1048576, false}, // 1MB files
		{"Many_Small_Files", 1000, 512, false},
		{"Nested_Directories", 50, 2048, true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			_, manager, cleanup := setupBenchmarkEnvironment(b, scenario.numFiles, scenario.fileSize, scenario.nested)
			defer cleanup()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				savepointName := fmt.Sprintf("bench-savepoint-%d", i)
				_, err := manager.CreateSavepoint(savepointName, "Benchmark savepoint")
				if err != nil {
					b.Fatalf("Failed to create savepoint: %v", err)
				}
			}
			b.StopTimer()

			// Clean up savepoints between runs
			savepoints, _ := manager.ListSavepoints()
			for _, savepoint := range savepoints {
				manager.DeleteSavepoint(savepoint.ID)
			}
		})
	}
}

// BenchmarkListSavepoints benchmarks listing savepoints with different numbers of savepoints
func BenchmarkListSavepoints(b *testing.B) {
	_, manager, cleanup := setupBenchmarkEnvironment(b, 10, 1024, false)
	defer cleanup()

	// Pre-create savepoints for listing
	savepoints := make([]*Savepoint, 0, 1000)
	for i := 0; i < 1000; i++ {
		cp, err := manager.CreateSavepoint(fmt.Sprintf("list-test-%d", i), "Test savepoint for listing")
		if err != nil {
			b.Fatalf("Failed to create savepoint %d: %v", i, err)
		}
		savepoints = append(savepoints, cp)
	}
	defer func() {
		for _, cp := range savepoints {
			manager.DeleteSavepoint(cp.ID)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ListSavepoints()
		if err != nil {
			b.Fatalf("Failed to list savepoints: %v", err)
		}
	}
	b.StopTimer()
}

// BenchmarkRestoreSavepoint benchmarks restoration with different file sizes
func BenchmarkRestoreSavepoint(b *testing.B) {
	scenarios := []struct {
		name     string
		numFiles int
		fileSize int
	}{
		{"Small_Restore", 10, 1024},
		{"Medium_Restore", 100, 10240},
		{"Large_Restore", 10, 1048576}, // 1MB files
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			tempDir, manager, cleanup := setupBenchmarkEnvironment(b, scenario.numFiles, scenario.fileSize, false)
			defer cleanup()

			// Create a savepoint
			savepoint, err := manager.CreateSavepoint("restore-test", "Savepoint for restoration testing")
			if err != nil {
				b.Fatalf("Failed to create savepoint: %v", err)
			}
			defer manager.DeleteSavepoint(savepoint.ID)

			// Modify files to ensure there's something to restore
			for i := 0; i < scenario.numFiles; i++ {
				fileName := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
				err = os.WriteFile(fileName, []byte(strings.Repeat("X", scenario.fileSize)), 0644)
				if err != nil {
					b.Fatalf("Failed to modify file %d: %v", i, err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				err = manager.RestoreSavepoint(savepoint.ID)
				if err != nil {
					b.Fatalf("Failed to restore savepoint: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkDeleteSavepoint benchmarks deletion of savepoints
func BenchmarkDeleteSavepoint(b *testing.B) {
	_, manager, cleanup := setupBenchmarkEnvironment(b, 100, 10240, false)
	defer cleanup()

	// Pre-create savepoints to delete
	savepointIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		cp, err := manager.CreateSavepoint(fmt.Sprintf("delete-test-%d", i), "Savepoint for deletion testing")
		if err != nil {
			b.Fatalf("Failed to create savepoint %d: %v", i, err)
		}
		savepointIDs[i] = cp.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := manager.DeleteSavepoint(savepointIDs[i])
		if err != nil {
			b.Fatalf("Failed to delete savepoint %d: %v", i, err)
		}
	}
	b.StopTimer()
}

// BenchmarkGenerateID benchmarks ID generation
func BenchmarkGenerateID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generateID()
		if err != nil {
			b.Fatalf("Failed to generate ID: %v", err)
		}
	}
	b.StopTimer()
}

// BenchmarkCopyFile benchmarks file copying with different sizes
func BenchmarkCopyFile(b *testing.B) {
	sizes := []int{
		1024,     // 1KB
		10240,    // 10KB
		102400,   // 100KB
		1048576,  // 1MB
		10485760, // 10MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			tempDir, err := os.MkdirTemp("", "copy-bench")
			if err != nil {
				b.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			srcFile := filepath.Join(tempDir, "src.bin")
			dstFile := filepath.Join(tempDir, "dst.bin")

			// Create source file with specified size
			content := strings.Repeat("A", size)
			err = os.WriteFile(srcFile, []byte(content), 0644)
			if err != nil {
				b.Fatalf("Failed to create source file: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := copyFile(srcFile, dstFile)
				if err != nil {
					b.Fatalf("Failed to copy file: %v", err)
				}
				// Remove destination file for next iteration
				os.Remove(dstFile)
			}
			b.StopTimer()
		})
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent savepoint operations
func BenchmarkConcurrentOperations(b *testing.B) {
	_, manager, cleanup := setupBenchmarkEnvironment(b, 50, 2048, false)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			savepointName := fmt.Sprintf("concurrent-%d", i)
			_, err := manager.CreateSavepoint(savepointName, "Concurrent savepoint")
			if err != nil {
				b.Fatalf("Failed to create concurrent savepoint: %v", err)
			}
			i++
		}
	})
	b.StopTimer()
}

// BenchmarkMemoryUsage benchmarks memory usage during savepoint operations
func BenchmarkMemoryUsage(b *testing.B) {
	_, manager, cleanup := setupBenchmarkEnvironment(b, 100, 10240, false)
	defer cleanup()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		savepointName := fmt.Sprintf("memory-test-%d", i)
		_, err := manager.CreateSavepoint(savepointName, "Memory test savepoint")
		if err != nil {
			b.Fatalf("Failed to create savepoint: %v", err)
		}
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
}

// setupBenchmarkEnvironment creates a test environment for benchmarking
func setupBenchmarkEnvironment(b *testing.B, numFiles, fileSize int, nested bool) (string, *SavepointManager, func()) {
	tempDir, err := os.MkdirTemp("", "savepoint-bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to init git repo: %v", err)
	}

	// Create test files
	w, err := repo.Worktree()
	if err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to get worktree: %v", err)
	}

	content := strings.Repeat("X", fileSize)
	for i := 0; i < numFiles; i++ {
		var fileName string
		if nested && i%10 == 0 {
			// Create nested directories
			dirName := fmt.Sprintf("dir%d", i/10)
			err = os.MkdirAll(filepath.Join(tempDir, dirName), 0755)
			if err != nil {
				os.RemoveAll(tempDir)
				b.Fatalf("Failed to create nested dir: %v", err)
			}
			fileName = filepath.Join(dirName, fmt.Sprintf("file%d.txt", i))
		} else {
			fileName = fmt.Sprintf("file%d.txt", i)
		}

		filePath := filepath.Join(tempDir, fileName)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			os.RemoveAll(tempDir)
			b.Fatalf("Failed to create file %d: %v", i, err)
		}

		_, err = w.Add(fileName)
		if err != nil {
			os.RemoveAll(tempDir)
			b.Fatalf("Failed to add file %d: %v", i, err)
		}
	}

	// Set environment variable
	os.Setenv("REPO_PATH", tempDir)

	// Create savepoint manager
	manager, err := NewSavepointManager()
	if err != nil {
		os.Unsetenv("REPO_PATH")
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create savepoint manager: %v", err)
	}

	cleanup := func() {
		os.Unsetenv("REPO_PATH")
		os.RemoveAll(tempDir)
	}

	return tempDir, manager, cleanup
}

// TestBenchmarkSanity runs a quick sanity check to ensure benchmarks work
func TestBenchmarkSanity(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "savepoint-bench-sanity")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Create test files
	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	content := strings.Repeat("X", 1024)
	for i := 0; i < 5; i++ {
		fileName := fmt.Sprintf("file%d.txt", i)
		filePath := filepath.Join(tempDir, fileName)
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}
		_, err = w.Add(fileName)
		if err != nil {
			t.Fatalf("Failed to add file %d: %v", i, err)
		}
	}

	// Set environment variable
	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	// Create savepoint manager
	manager, err := NewSavepointManager()
	if err != nil {
		t.Fatalf("Failed to create savepoint manager: %v", err)
	}

	// Create a savepoint
	savepoint, err := manager.CreateSavepoint("sanity-test", "Sanity check savepoint")
	if err != nil {
		t.Fatalf("Failed to create savepoint: %v", err)
	}

	// List savepoints
	savepoints, err := manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints: %v", err)
	}

	if len(savepoints) != 1 {
		t.Errorf("Expected 1 savepoint, got %d", len(savepoints))
	}

	// Restore savepoint
	err = manager.RestoreSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore savepoint: %v", err)
	}

	// Delete savepoint
	err = manager.DeleteSavepoint(savepoint.ID)
	if err != nil {
		t.Fatalf("Failed to delete savepoint: %v", err)
	}

	// Verify deletion
	savepoints, err = manager.ListSavepoints()
	if err != nil {
		t.Fatalf("Failed to list savepoints after deletion: %v", err)
	}

	if len(savepoints) != 0 {
		t.Errorf("Expected 0 savepoints after deletion, got %d", len(savepoints))
	}
}
