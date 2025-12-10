package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	auditFile     *os.File
	auditMutex    sync.Mutex
	auditEnabled  bool
	auditFilePath string
)

// InitAuditLogger initializes the audit logging system
func InitAuditLogger() error {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	// Check if audit logging is disabled
	if os.Getenv("MCP_POWERSHELL_AUDIT_DISABLED") == "true" {
		auditEnabled = false
		return nil
	}

	auditEnabled = true

	// Determine audit file path
	auditFilePath = os.Getenv("MCP_POWERSHELL_AUDIT_FILE")
	if auditFilePath == "" {
		// Default to current directory
		auditFilePath = "mcp-powershell-audit.log"
	}

	// Open audit file in append mode
	var err error
	auditFile, err = os.OpenFile(auditFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}

	// Write startup message
	startupMsg := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"event_type":  "startup",
		"server_name": "mcp-powershell",
		"pid":         os.Getpid(),
		"version":     "1.0.0",
	}

	if err := writeAuditEntry(startupMsg); err != nil {
		auditFile.Close()
		auditFile = nil
		return fmt.Errorf("failed to write startup message: %w", err)
	}

	return nil
}

// CloseAuditLogger closes the audit logging system
func CloseAuditLogger() {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	if auditFile != nil {
		// Write shutdown message
		shutdownMsg := map[string]interface{}{
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"event_type":  "shutdown",
			"server_name": "mcp-powershell",
			"pid":         os.Getpid(),
		}

		writeAuditEntry(shutdownMsg)
		auditFile.Close()
		auditFile = nil
	}
}

// auditLog logs an audit entry for PowerShell operations
func auditLog(operation, command, script, workingDir string, envVars map[string]string, result *CommandResult, security *SecurityResult, durationMs int64, success bool, errorCode int, errorType string) {
	if !auditEnabled {
		return
	}

	auditMutex.Lock()
	defer auditMutex.Unlock()

	if auditFile == nil {
		return
	}

	// Build audit entry
	entry := AuditLog{
		Timestamp:  time.Now().UTC(),
		Operation:  operation,
		Command:    command,
		Script:     script,
		WorkingDir: workingDir,
		Environment: envVars,
		Result:     result,
		Security:   security,
		DurationMs: durationMs,
		Success:    success,
		ErrorCode:  errorCode,
		ErrorType:  errorType,
		User:       os.Getenv("USER"),
	}

	// Convert to JSON and write
	auditData := map[string]interface{}{
		"timestamp":     entry.Timestamp.Format(time.RFC3339),
		"event_type":    "operation",
		"operation":     entry.Operation,
		"command":       entry.Command,
		"script":        entry.Script,
		"user":          entry.User,
		"working_dir":   entry.WorkingDir,
		"environment":   entry.Environment,
		"result":        entry.Result,
		"security":      entry.Security,
		"duration_ms":   entry.DurationMs,
		"success":       entry.Success,
		"error_code":    entry.ErrorCode,
		"error_type":    entry.ErrorType,
		"server_name":   "mcp-powershell",
		"server_version": "1.0.0",
	}

	writeAuditEntry(auditData)
}

// writeAuditEntry writes an audit entry to the audit file
func writeAuditEntry(entry map[string]interface{}) error {
	if auditFile == nil {
		return fmt.Errorf("audit file not open")
	}

	// Convert to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	// Write to file with newline
	if _, err := auditFile.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write audit entry: %w", err)
	}

	// Sync to ensure data is written to disk
	return auditFile.Sync()
}

// GetAuditStats returns audit logging statistics
func GetAuditStats() map[string]interface{} {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	stats := map[string]interface{}{
		"enabled":     auditEnabled,
		"audit_file":  auditFilePath,
		"server_name": "mcp-powershell",
	}

	if auditFile != nil {
		// Get file info
		if fileInfo, err := auditFile.Stat(); err == nil {
			stats["file_size"] = fileInfo.Size()
			stats["file_modified"] = fileInfo.ModTime().UTC().Format(time.RFC3339)
		}
	}

	return stats
}