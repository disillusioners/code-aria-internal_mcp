package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	_ "modernc.org/sqlite"
)

const SAVEPOINT_DIR = ".mcp-savepoints"
const DB_FILE = "savepoints.db"

// SavepointManager handles savepoint operations
type SavepointManager struct {
	repoPath     string
	savepointDir string
	db           *sql.DB
}

// NewSavepointManager creates a new savepoint manager
func NewSavepointManager() (*SavepointManager, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return nil, fmt.Errorf("REPO_PATH environment variable not set")
	}

	savepointDir := filepath.Join(repoPath, SAVEPOINT_DIR)
	if err := os.MkdirAll(savepointDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create savepoint directory: %w", err)
	}

	// Open SQLite database
	dbPath := filepath.Join(savepointDir, DB_FILE)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	cm := &SavepointManager{
		repoPath:     repoPath,
		savepointDir: savepointDir,
		db:           db,
	}

	// Initialize database schema
	if err := cm.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return cm, nil
}

// Close closes the database connection
func (cm *SavepointManager) Close() error {
	if cm.db != nil {
		return cm.db.Close()
	}
	return nil
}

// initSchema creates the database schema if it doesn't exist
func (cm *SavepointManager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS savepoints (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		timestamp TEXT NOT NULL,
		total_size INTEGER NOT NULL,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS savepoint_files (
		savepoint_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_status TEXT NOT NULL CHECK(file_status IN ('new', 'modified', 'deleted')),
		file_size INTEGER NOT NULL,
		PRIMARY KEY (savepoint_id, file_path),
		FOREIGN KEY (savepoint_id) REFERENCES savepoints(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_savepoint_files_id ON savepoint_files(savepoint_id);
	CREATE INDEX IF NOT EXISTS idx_savepoint_files_status ON savepoint_files(savepoint_id, file_status);
	CREATE INDEX IF NOT EXISTS idx_savepoints_timestamp ON savepoints(timestamp);
	`

	_, err := cm.db.Exec(schema)
	return err
}

// CreateSavepoint creates a new savepoint of the current working directory changes
func (cm *SavepointManager) CreateSavepoint(name, description string) (*Savepoint, error) {
	// Generate unique savepoint ID
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate savepoint ID: %w", err)
	}

	// Get working directory changes with status
	fileChanges, err := cm.getWorkingChanges()
	if err != nil {
		return nil, fmt.Errorf("failed to get working changes: %w", err)
	}

	if len(fileChanges) == 0 {
		return nil, fmt.Errorf("no changes to savepoint")
	}

	// Create savepoint directory
	savepointPath := filepath.Join(cm.savepointDir, id)
	if err := os.MkdirAll(savepointPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create savepoint directory: %w", err)
	}

	var savepointFiles []string
	var totalSize int64

	// Begin transaction
	tx, err := cm.db.Begin()
	if err != nil {
		os.RemoveAll(savepointPath)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Process each file change
	for _, fileChange := range fileChanges {
		var fileSize int64

		if fileChange.Status == "deleted" {
			// For deleted files, don't copy, just record
			fileSize = 0
		} else {
			// Copy file to savepoint directory
			srcPath := filepath.Join(cm.repoPath, fileChange.Path)
			dstPath := filepath.Join(savepointPath, fileChange.Path)

			// Verify source exists for new/modified files
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				os.RemoveAll(savepointPath)
				return nil, fmt.Errorf("source file %s does not exist", fileChange.Path)
			}

			// Create destination directory if needed
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				os.RemoveAll(savepointPath)
				return nil, fmt.Errorf("failed to create destination directory: %w", err)
			}

			// Copy file
			size, err := copyFile(srcPath, dstPath)
			if err != nil {
				os.RemoveAll(savepointPath)
				return nil, fmt.Errorf("failed to copy file %s: %w", fileChange.Path, err)
			}

			fileSize = size
		}

		savepointFiles = append(savepointFiles, fileChange.Path)
		totalSize += fileSize

		// Insert into savepoint_files table
		_, err := tx.Exec(
			"INSERT INTO savepoint_files (savepoint_id, file_path, file_status, file_size) VALUES (?, ?, ?, ?)",
			id, fileChange.Path, fileChange.Status, fileSize,
		)
		if err != nil {
			os.RemoveAll(savepointPath)
			return nil, fmt.Errorf("failed to insert file record: %w", err)
		}
	}

	// Insert savepoint metadata
	timestamp := time.Now().Format(time.RFC3339)
	_, err = tx.Exec(
		"INSERT INTO savepoints (id, name, description, timestamp, total_size) VALUES (?, ?, ?, ?, ?)",
		id, name, description, timestamp, totalSize,
	)
	if err != nil {
		os.RemoveAll(savepointPath)
		return nil, fmt.Errorf("failed to insert savepoint: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		os.RemoveAll(savepointPath)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	savepoint := &Savepoint{
		ID:          id,
		Name:        name,
		Description: description,
		Timestamp:   timestamp,
		Files:       savepointFiles,
		Size:        totalSize,
	}

	return savepoint, nil
}

// ListSavepoints returns all available savepoints
func (cm *SavepointManager) ListSavepoints() ([]*Savepoint, error) {
	rows, err := cm.db.Query(`
		SELECT c.id, c.name, c.description, c.timestamp, c.total_size
		FROM savepoints c
		ORDER BY c.timestamp DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query savepoints: %w", err)
	}
	defer rows.Close()

	var savepoints []*Savepoint
	for rows.Next() {
		var cp Savepoint
		err := rows.Scan(&cp.ID, &cp.Name, &cp.Description, &cp.Timestamp, &cp.Size)
		if err != nil {
			continue
		}

		// Get file list for this savepoint
		files, err := cm.getSavepointFiles(cp.ID)
		if err != nil {
			continue
		}
		cp.Files = files

		savepoints = append(savepoints, &cp)
	}

	return savepoints, nil
}

// GetSavepoint returns a specific savepoint by ID
func (cm *SavepointManager) GetSavepoint(id string) (*Savepoint, error) {
	var cp Savepoint
	err := cm.db.QueryRow(
		"SELECT id, name, description, timestamp, total_size FROM savepoints WHERE id = ?",
		id,
	).Scan(&cp.ID, &cp.Name, &cp.Description, &cp.Timestamp, &cp.Size)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("savepoint %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query savepoint: %w", err)
	}

	// Get file list
	files, err := cm.getSavepointFiles(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get savepoint files: %w", err)
	}
	cp.Files = files

	return &cp, nil
}

// GetSavepointWithStatus returns a savepoint with file status information
func (cm *SavepointManager) GetSavepointWithStatus(id string) (*SavepointWithStatus, error) {
	savepoint, err := cm.GetSavepoint(id)
	if err != nil {
		return nil, err
	}

	// Get files with status
	rows, err := cm.db.Query(
		"SELECT file_path, file_status FROM savepoint_files WHERE savepoint_id = ?",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query savepoint files: %w", err)
	}
	defer rows.Close()

	var filesWithStatus []FileStatusEntry
	for rows.Next() {
		var entry FileStatusEntry
		if err := rows.Scan(&entry.Path, &entry.Status); err != nil {
			continue
		}
		filesWithStatus = append(filesWithStatus, entry)
	}

	return &SavepointWithStatus{
		Savepoint:       *savepoint,
		FilesWithStatus: filesWithStatus,
	}, nil
}

// getSavepointFiles returns the list of file paths for a savepoint
func (cm *SavepointManager) getSavepointFiles(savepointID string) ([]string, error) {
	rows, err := cm.db.Query(
		"SELECT file_path FROM savepoint_files WHERE savepoint_id = ?",
		savepointID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []string
	for rows.Next() {
		var file string
		if err := rows.Scan(&file); err != nil {
			continue
		}
		files = append(files, file)
	}

	return files, nil
}

// RestoreSavepoint restores a savepoint to the working directory
func (cm *SavepointManager) RestoreSavepoint(id string) error {
	// Load savepoint with status
	savepoint, err := cm.GetSavepointWithStatus(id)
	if err != nil {
		return fmt.Errorf("failed to load savepoint: %w", err)
	}

	var operations []FileRestoreOperation
	savepointPath := filepath.Join(cm.savepointDir, id)

	// Process each file based on status
	for _, fileEntry := range savepoint.FilesWithStatus {
		switch fileEntry.Status {
		case "new", "modified":
			// Restore file from savepoint
			srcPath := filepath.Join(savepointPath, fileEntry.Path)

			// Verify source exists
			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				cm.rollbackRestore(operations)
				return fmt.Errorf("savepoint corrupted: file %s missing", fileEntry.Path)
			}

			dstPath := filepath.Join(cm.repoPath, fileEntry.Path)

			// Create backup if file exists (for rollback)
			var backupPath string
			if _, err := os.Stat(dstPath); err == nil {
				backupPath = dstPath + ".savepoint_backup"
				if _, err := copyFile(dstPath, backupPath); err != nil {
					cm.rollbackRestore(operations)
					return fmt.Errorf("failed to backup file %s: %w", fileEntry.Path, err)
				}
			}

			// Create directory
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				cm.rollbackRestore(operations)
				return fmt.Errorf("failed to create destination directory: %w", err)
			}

			// Copy file
			if _, err := copyFile(srcPath, dstPath); err != nil {
				cm.rollbackRestore(operations)
				return fmt.Errorf("failed to restore file %s: %w", fileEntry.Path, err)
			}

			operations = append(operations, FileRestoreOperation{
				Type:     "copy",
				FilePath: fileEntry.Path,
				Backup:   backupPath,
			})

		case "deleted":
			// Delete file from working directory
			dstPath := filepath.Join(cm.repoPath, fileEntry.Path)

			// Check if exists
			if _, err := os.Stat(dstPath); err == nil {
				// Create backup for rollback
				backupPath := dstPath + ".savepoint_backup"
				if _, err := copyFile(dstPath, backupPath); err != nil {
					cm.rollbackRestore(operations)
					return fmt.Errorf("failed to backup file for deletion %s: %w", fileEntry.Path, err)
				}

				// Delete file/directory
				if err := os.RemoveAll(dstPath); err != nil {
					cm.rollbackRestore(operations)
					return fmt.Errorf("failed to delete file %s: %w", fileEntry.Path, err)
				}

				operations = append(operations, FileRestoreOperation{
					Type:     "delete",
					FilePath: fileEntry.Path,
					Backup:   backupPath,
				})
			}
			// If file doesn't exist, no-op (idempotent)
		}
	}

	// Cleanup backup files after successful restore
	for _, op := range operations {
		if op.Backup != "" {
			os.Remove(op.Backup) // Ignore errors
		}
	}

	return nil
}

// rollbackRestore rolls back restore operations in reverse order
func (cm *SavepointManager) rollbackRestore(operations []FileRestoreOperation) {
	// Rollback in reverse order
	for i := len(operations) - 1; i >= 0; i-- {
		op := operations[i]
		dstPath := filepath.Join(cm.repoPath, op.FilePath)

		switch op.Type {
		case "copy":
			// Restore from backup or delete if backup empty
			if op.Backup != "" {
				_, _ = copyFile(op.Backup, dstPath) // Ignore errors during rollback
				os.Remove(op.Backup)
			} else {
				os.Remove(dstPath)
			}
		case "delete":
			// Restore from backup
			if op.Backup != "" {
				os.MkdirAll(filepath.Dir(dstPath), 0755)
				_, _ = copyFile(op.Backup, dstPath) // Ignore errors during rollback
				os.Remove(op.Backup)
			}
		}
	}
}

// DeleteSavepoint removes a savepoint
func (cm *SavepointManager) DeleteSavepoint(id string) error {
	// Begin transaction
	tx, err := cm.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if savepoint exists
	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM savepoints WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check savepoint existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("savepoint %s not found", id)
	}

	// Delete from database (CASCADE will delete savepoint_files)
	_, err = tx.Exec("DELETE FROM savepoints WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete savepoint: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Remove savepoint directory
	savepointPath := filepath.Join(cm.savepointDir, id)
	if err := os.RemoveAll(savepointPath); err != nil {
		return fmt.Errorf("failed to remove savepoint directory: %w", err)
	}

	return nil
}

// getWorkingChanges returns a list of changed files with their status
func (cm *SavepointManager) getWorkingChanges() ([]FileChange, error) {
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

	var changes []FileChange
	for file, s := range status {
		var fileStatus string

		// Detect file status based on git status codes
		// Worktree: ' ' = unchanged, '?' = untracked, 'M' = modified, 'D' = deleted
		// Staging: ' ' = unchanged, 'A' = added, 'M' = modified, 'D' = deleted
		if s.Worktree == 'D' || s.Staging == 'D' {
			fileStatus = "deleted"
		} else if s.Worktree == '?' || s.Staging == 'A' {
			fileStatus = "new"
		} else if s.Worktree == 'M' || s.Staging == 'M' {
			fileStatus = "modified"
		} else if s.Worktree == ' ' && s.Staging == ' ' {
			continue // Skip unchanged files
		} else {
			// Handle other status codes as modified for safety
			fileStatus = "modified"
		}

		changes = append(changes, FileChange{
			Path:   file,
			Status: fileStatus,
		})
	}

	return changes, nil
}

// generateID generates a unique 8-character savepoint ID
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
