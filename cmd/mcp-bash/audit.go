package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Global audit configuration
var (
	auditEnabled = true
	auditLogFile = ""
	auditLogger  *AuditLogger
)

// AuditLogger handles audit logging
type AuditLogger struct {
	logFile *os.File
}

// InitAuditLogger initializes the audit logger
func InitAuditLogger() error {
	if !auditEnabled {
		return nil
	}

	// Determine audit log file path
	if auditLogFile == "" {
		// Default to REPO_PATH/.mcp_audit.log
		repoPath := os.Getenv("REPO_PATH")
		if repoPath == "" {
			repoPath = os.TempDir()
		}
		auditLogFile = filepath.Join(repoPath, ".mcp_audit.log")
	}

	// Create/open audit log file
	file, err := os.OpenFile(auditLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	auditLogger = &AuditLogger{
		logFile: file,
	}

	return nil
}

// CloseAuditLogger closes the audit logger
func CloseAuditLogger() {
	if auditLogger != nil && auditLogger.logFile != nil {
		auditLogger.logFile.Close()
	}
}

// auditLog logs an audit entry
func auditLog(operation, command, script, workingDir string, envVars map[string]string, result *CommandResult, security *SecurityResult, durationMs int64, success bool, errorCode int, errorType string) {
	if !auditEnabled || auditLogger == nil {
		return
	}

	// Get current user
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	if user == "" {
		user = "unknown"
	}

	// Create audit entry
	entry := AuditLog{
		Timestamp:   time.Now().UTC(),
		Operation:   operation,
		Command:     command,
		Script:      script,
		User:        user,
		WorkingDir:  workingDir,
		Environment: envVars,
		Result:      result,
		Security:    security,
		DurationMs:  durationMs,
		Success:     success,
		ErrorCode:   errorCode,
		ErrorType:   errorType,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal audit log entry: %v\n", err)
		return
	}

	// Write to log file
	logLine := string(jsonData) + "\n"
	if _, err := auditLogger.logFile.WriteString(logLine); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write audit log entry: %v\n", err)
	}

	// Also write to stderr for immediate visibility
	fmt.Fprintf(os.Stderr, "[AUDIT] %s\n", logLine)
}

// GetAuditStats returns audit statistics
func GetAuditStats() (map[string]interface{}, error) {
	if !auditEnabled || auditLogFile == "" {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	// Read audit log file
	file, err := os.Open(auditLogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer file.Close()

	// Parse log entries
	var entries []AuditLog
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var entry AuditLog
		if err := decoder.Decode(&entry); err == nil {
			entries = append(entries, entry)
		}
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"enabled":        true,
		"total_entries":  len(entries),
		"log_file":      auditLogFile,
		"operations":     make(map[string]int),
		"success_rate":   0.0,
		"error_types":    make(map[string]int),
		"security_violations": 0,
		"avg_duration_ms": 0,
		"last_activity":  nil,
	}

	if len(entries) > 0 {
		var successCount, totalDuration int64
		operations := make(map[string]int)
		errorTypes := make(map[string]int)
		securityViolations := 0

		for _, entry := range entries {
			// Count operations
			operations[entry.Operation]++
			
			// Count successes
			if entry.Success {
				successCount++
			}
			
			// Count error types
			if entry.ErrorType != "" {
				errorTypes[entry.ErrorType]++
			}
			
			// Count security violations
			if entry.Security != nil && !entry.Security.Valid {
				securityViolations++
			}
			
			// Sum duration
			totalDuration += entry.DurationMs
			
			// Track last activity
			if stats["last_activity"] == nil || entry.Timestamp.After(stats["last_activity"].(time.Time)) {
				stats["last_activity"] = entry.Timestamp
			}
		}

		stats["operations"] = operations
		stats["success_rate"] = float64(successCount) / float64(len(entries))
		stats["error_types"] = errorTypes
		stats["security_violations"] = securityViolations
		stats["avg_duration_ms"] = totalDuration / int64(len(entries))
	}

	return stats, nil
}

// ClearAuditLog clears the audit log
func ClearAuditLog() error {
	if !auditEnabled || auditLogFile == "" {
		return nil
	}

	// Close current log file
	if auditLogger != nil && auditLogger.logFile != nil {
		auditLogger.logFile.Close()
	}

	// Remove log file
	if err := os.Remove(auditLogFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove audit log file: %w", err)
	}

	// Reinitialize audit logger
	return InitAuditLogger()
}

// SearchAuditLog searches the audit log for specific criteria
func SearchAuditLog(criteria map[string]interface{}) ([]AuditLog, error) {
	if !auditEnabled || auditLogFile == "" {
		return nil, fmt.Errorf("audit logging is not enabled")
	}

	// Read audit log file
	file, err := os.Open(auditLogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}
	defer file.Close()

	// Parse log entries
	var entries []AuditLog
	decoder := json.NewDecoder(file)
	for decoder.More() {
		var entry AuditLog
		if err := decoder.Decode(&entry); err == nil {
			entries = append(entries, entry)
		}
	}

	// Filter entries based on criteria
	var filteredEntries []AuditLog
	for _, entry := range entries {
		if matchesCriteria(entry, criteria) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	return filteredEntries, nil
}

// matchesCriteria checks if an audit entry matches the search criteria
func matchesCriteria(entry AuditLog, criteria map[string]interface{}) bool {
	// Check operation
	if op, ok := criteria["operation"].(string); ok && op != "" {
		if entry.Operation != op {
			return false
		}
	}

	// Check user
	if user, ok := criteria["user"].(string); ok && user != "" {
		if entry.User != user {
			return false
		}
	}

	// Check success
	if success, ok := criteria["success"].(bool); ok {
		if entry.Success != success {
			return false
		}
	}

	// Check error type
	if errorType, ok := criteria["error_type"].(string); ok && errorType != "" {
		if entry.ErrorType != errorType {
			return false
		}
	}

	// Check time range
	if startTime, ok := criteria["start_time"].(time.Time); ok {
		if entry.Timestamp.Before(startTime) {
			return false
		}
	}

	if endTime, ok := criteria["end_time"].(time.Time); ok {
		if entry.Timestamp.After(endTime) {
			return false
		}
	}

	// Check command contains
	if command, ok := criteria["command_contains"].(string); ok && command != "" {
		if entry.Command == "" || !containsString(entry.Command, command) {
			return false
		}
	}

	// Check script contains
	if script, ok := criteria["script_contains"].(string); ok && script != "" {
		if entry.Script == "" || !containsString(entry.Script, script) {
			return false
		}
	}

	// Check security violation
	if securityViolation, ok := criteria["security_violation"].(bool); ok {
		if securityViolation {
			if entry.Security == nil || entry.Security.Valid {
				return false
			}
		} else {
			if entry.Security != nil && !entry.Security.Valid {
				return false
			}
		}
	}

	return true
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr ||
		     containsSubstring(s, substr)))
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SetAuditConfiguration configures audit logging settings
func SetAuditConfiguration(enabled bool, logFile string) {
	auditEnabled = enabled
	auditLogFile = logFile
	
	// Reinitialize if already initialized
	if auditLogger != nil {
		CloseAuditLogger()
		if enabled {
			InitAuditLogger()
		}
	}
}