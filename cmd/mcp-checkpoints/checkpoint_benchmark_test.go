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

// BenchmarkCreateCheckpoint benchmarks checkpoint creation with different scenarios
func BenchmarkCreateCheckpoint(b *testing.B) {
	scenarios := []struct {
		name      string
		numFiles  int
		fileSize  int
		nested    bool
	}{
		{"Small_Files", 10, 1024, false},
		{"Medium_Files", 100, 10240, false},
		{"Large_Files", 10, 1048576, false}, // 1MB files
		{"Many_Small_Files", 1000, 512, false},
		{"Nested_Directories", 50, 2048, true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			tempDir, manager, cleanup := setupBenchmarkEnvironment(b, scenario.numFiles, scenario.fileSize, scenario.nested)
			defer cleanup()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				checkpointName := fmt.Sprintf("bench-checkpoint-%d", i)
				_, err := manager.CreateCheckpoint(checkpointName, "Benchmark checkpoint")
				if err != nil {
					b.Fatalf("Failed to create checkpoint: %v", err)
				}
			}
			b.StopTimer()

			// Clean up checkpoints between runs
			checkpoints, _ := manager.ListCheckpoints()
			for _, checkpoint := range checkpoints {
				manager.DeleteCheckpoint(checkpoint.ID)
			}
		})
	}
}

// BenchmarkListCheckpoints benchmarks listing checkpoints with different numbers of checkpoints
func BenchmarkListCheckpoints(b *testing.B) {
	tempDir, manager, cleanup := setupBenchmarkEnvironment(b, 10, 1024, false)
	defer cleanup()

	// Pre-create checkpoints for listing
	checkpoints := make([]*Checkpoint, 0, 1000)
	for i := 0; i < 1000; i++ {
		cp, err := manager.CreateCheckpoint(fmt.Sprintf("list-test-%d", i), "Test checkpoint for listing")
		if err != nil {
			b.Fatalf("Failed to create checkpoint %d: %v", i, err)
		}
		checkpoints = append(checkpoints, cp)
	}
	defer func() {
		for _, cp := range checkpoints {
			manager.DeleteCheckpoint(cp.ID)
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ListCheckpoints()
		if err != nil {
			b.Fatalf("Failed to list checkpoints: %v", err)
		}
	}
	b.StopTimer()
}

// BenchmarkRestoreCheckpoint benchmarks restoration with different file sizes
func BenchmarkRestoreCheckpoint(b *testing.B) {
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

			// Create a checkpoint
			checkpoint, err := manager.CreateCheckpoint("restore-test", "Checkpoint for restoration testing")
			if err != nil {
				b.Fatalf("Failed to create checkpoint: %v", err)
			}
			defer manager.DeleteCheckpoint(checkpoint.ID)

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
				err = manager.RestoreCheckpoint(checkpoint.ID)
				if err != nil {
					b.Fatalf("Failed to restore checkpoint: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

// BenchmarkDeleteCheckpoint benchmarks deletion of checkpoints
func BenchmarkDeleteCheckpoint(b *testing.B) {
	tempDir, manager, cleanup := setupBenchmarkEnvironment(b, 100, 10240, false)
	defer cleanup()

	// Pre-create checkpoints to delete
	checkpointIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		cp, err := manager.CreateCheckpoint(fmt.Sprintf("delete-test-%d", i), "Checkpoint for deletion testing")
		if err != nil {
			b.Fatalf("Failed to create checkpoint %d: %v", i, err)
		}
		checkpointIDs[i] = cp.ID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := manager.DeleteCheckpoint(checkpointIDs[i])
		if err != nil {
			b.Fatalf("Failed to delete checkpoint %d: %v", i, err)
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
		1024,      // 1KB
		10240,     // 10KB
		102400,    // 100KB
		1048576,   // 1MB
		10485760,  // 10MB
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

// BenchmarkConcurrentOperations benchmarks concurrent checkpoint operations
func BenchmarkConcurrentOperations(b *testing.B) {
	tempDir, manager, cleanup := setupBenchmarkEnvironment(b, 50, 2048, false)
	defer cleanup()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			checkpointName := fmt.Sprintf("concurrent-%d", i)
			_, err := manager.CreateCheckpoint(checkpointName, "Concurrent checkpoint")
			if err != nil {
				b.Fatalf("Failed to create concurrent checkpoint: %v", err)
			}
			i++
		}
	})
	b.StopTimer()
}

// BenchmarkMemoryUsage benchmarks memory usage during checkpoint operations
func BenchmarkMemoryUsage(b *testing.B) {
	tempDir, manager, cleanup := setupBenchmarkEnvironment(b, 100, 10240, false)
	defer cleanup()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checkpointName := fmt.Sprintf("memory-test-%d", i)
		_, err := manager.CreateCheckpoint(checkpointName, "Memory test checkpoint")
		if err != nil {
			b.Fatalf("Failed to create checkpoint: %v", err)
		}
	}
	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc)/float64(b.N), "bytes/op")
}

// setupBenchmarkEnvironment creates a test environment for benchmarking
func setupBenchmarkEnvironment(b *testing.B, numFiles, fileSize int, nested bool) (string, *CheckpointManager, func()) {
	tempDir, err := os.MkdirTemp("", "checkpoint-bench")
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

	// Create checkpoint manager
	manager, err := NewCheckpointManager()
	if err != nil {
		os.Unsetenv("REPO_PATH")
		os.RemoveAll(tempDir)
		b.Fatalf("Failed to create checkpoint manager: %v", err)
	}

	cleanup := func() {
		os.Unsetenv("REPO_PATH")
		os.RemoveAll(tempDir)
	}

	return tempDir, manager, cleanup
}

// TestBenchmarkSanity runs a quick sanity check to ensure benchmarks work
func TestBenchmarkSanity(t *testing.T) {
	tempDir, manager, cleanup := setupBenchmarkEnvironment(t, 5, 1024, false)
	defer cleanup()

	// Create a checkpoint
	checkpoint, err := manager.CreateCheckpoint("sanity-test", "Sanity check checkpoint")
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// List checkpoints
	checkpoints, err := manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
	}

	// Restore checkpoint
	err = manager.RestoreCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to restore checkpoint: %v", err)
	}

	// Delete checkpoint
	err = manager.DeleteCheckpoint(checkpoint.ID)
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify deletion
	checkpoints, err = manager.ListCheckpoints()
	if err != nil {
		t.Fatalf("Failed to list checkpoints after deletion: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints after deletion, got %d", len(checkpoints))
	}
}