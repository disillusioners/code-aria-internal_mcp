package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

const CHECKPOINT_DIR = ".mcp-checkpoints"
const METADATA_FILE = "metadata.json"

// CheckpointManager handles checkpoint operations
type CheckpointManager struct {
	repoPath      string
	checkpointDir string
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager() (*CheckpointManager, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return nil, fmt.Errorf("REPO_PATH environment variable not set")
	}

	checkpointDir := filepath.Join(repoPath, CHECKPOINT_DIR)
	return &CheckpointManager{
		repoPath:      repoPath,
		checkpointDir: checkpointDir,
	}, nil
}

// CreateCheckpoint creates a new checkpoint of the current working directory changes
func (cm *CheckpointManager) CreateCheckpoint(name, description string) (*Checkpoint, error) {
	// Generate unique checkpoint ID
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate checkpoint ID: %w", err)
	}

	// Get working directory changes
	changedFiles, err := cm.getWorkingChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to get working changes: %w", err)
	}

	if len(changedFiles) == 0 {
		return nil, fmt.Errorf("no changes to checkpoint")
	}

	// Create checkpoint directory
	checkpointPath := filepath.Join(cm.checkpointDir, id)
	if err := os.MkdirAll(checkpointPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	var checkpointFiles []string
	var totalSize int64

	// Copy files to checkpoint directory
	for _, file := range changedFiles {
		srcPath := filepath.Join(cm.repoPath, file)
		dstPath := filepath.Join(checkpointPath, file)

		// Create destination directory if needed
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Copy file
		size, err := copyFile(srcPath, dstPath)
		if err != nil {
			// Cleanup on failure
			os.RemoveAll(checkpointPath)
			return nil, fmt.Errorf("failed to copy file %s: %w", file, err)
		}

		checkpointFiles = append(checkpointFiles, file)
		totalSize += size
	}

	// Create checkpoint metadata
	checkpoint := &Checkpoint{
		ID:          id,
		Name:        name,
		Description: description,
		Timestamp:   time.Now().Format(time.RFC3339),
		Files:       checkpointFiles,
		Size:        totalSize,
	}

	// Save metadata
	metadataPath := filepath.Join(checkpointPath, METADATA_FILE)
	if err := cm.saveCheckpointMetadata(checkpoint, metadataPath); err != nil {
		// Cleanup on failure
		os.RemoveAll(checkpointPath)
		return nil, fmt.Errorf("failed to save checkpoint metadata: %w", err)
	}

	return checkpoint, nil
}

// ListCheckpoints returns all available checkpoints
func (cm *CheckpointManager) ListCheckpoints() ([]*Checkpoint, error) {
	entries, err := os.ReadDir(cm.checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Checkpoint{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var checkpoints []*Checkpoint
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(cm.checkpointDir, entry.Name(), METADATA_FILE)
		checkpoint, err := cm.loadCheckpointMetadata(metadataPath)
		if err != nil {
			// Skip invalid checkpoints
			continue
		}

		checkpoints = append(checkpoints, checkpoint)
	}

	return checkpoints, nil
}

// GetCheckpoint returns a specific checkpoint by ID
func (cm *CheckpointManager) GetCheckpoint(id string) (*Checkpoint, error) {
	metadataPath := filepath.Join(cm.checkpointDir, id, METADATA_FILE)
	return cm.loadCheckpointMetadata(metadataPath)
}

// RestoreCheckpoint restores a checkpoint to the working directory
func (cm *CheckpointManager) RestoreCheckpoint(id string) error {
	checkpoint, err := cm.GetCheckpoint(id)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	checkpointPath := filepath.Join(cm.checkpointDir, id)

	// Restore files from checkpoint
	for _, file := range checkpoint.Files {
		srcPath := filepath.Join(checkpointPath, file)
		dstPath := filepath.Join(cm.repoPath, file)

		// Create destination directory if needed
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Copy file from checkpoint to working directory
		if _, err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to restore file %s: %w", file, err)
		}
	}

	return nil
}

// DeleteCheckpoint removes a checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(id string) error {
	checkpointPath := filepath.Join(cm.checkpointDir, id)
	if _, err := os.Stat(checkpointPath); os.IsNotExist(err) {
		return fmt.Errorf("checkpoint %s not found", id)
	}

	return os.RemoveAll(checkpointPath)
}

// getWorkingChanges returns a list of changed files in the working directory
func (cm *CheckpointManager) getWorkingChanges() ([]string, error) {
	// For now, we'll implement a basic version using go-git
	// In a production environment, you might want to integrate with the existing MCP git server

	// Open repository
	r, err := git.PlainOpen(cm.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get status
	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	var changedFiles []string
	for file, s := range status {
		// Include files with either staged or unstaged changes
		if s.Worktree == ' ' && s.Staging == ' ' {
			continue
		}

		changedFiles = append(changedFiles, file)
	}

	return changedFiles, nil
}

// saveCheckpointMetadata saves checkpoint metadata to file
func (cm *CheckpointManager) saveCheckpointMetadata(checkpoint *Checkpoint, path string) error {
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// loadCheckpointMetadata loads checkpoint metadata from file
func (cm *CheckpointManager) loadCheckpointMetadata(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, err
	}

	return &checkpoint, nil
}

// generateID generates a unique 8-character checkpoint ID
func generateID() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ToLower(hex.EncodeToString(bytes)), nil
}

// copyFile copies a file from src to dst and returns the size
func copyFile(src, dst string) (int64, error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destFile.Close()

	size, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return 0, err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	return size, os.Chmod(dst, sourceInfo.Mode())
}