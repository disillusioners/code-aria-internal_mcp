package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestInitAuditLogger tests the InitAuditLogger function
func TestInitAuditLogger(t *testing.T) {
	tests := []struct {
		name           string
		envAuditDisabled string
		envAuditFile    string
		wantErr        bool
	}{
		{
			name:            "Default initialization",
			envAuditDisabled: "",
			envAuditFile:     "",
			wantErr:         false,
		},
		{
			name:            "Audit disabled",
			envAuditDisabled: "true",
			envAuditFile:     "",
			wantErr:         false,
		},
		{
			name:            "Custom audit file",
			envAuditDisabled: "",
			envAuditFile:     "test-audit.log",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origAuditDisabled := os.Getenv("MCP_SYSTEMINFO_AUDIT_DISABLED")
			origAuditFile := os.Getenv("MCP_SYSTEMINFO_AUDIT_FILE")
			
			// Set test environment
			if tt.envAuditDisabled != "" {
				os.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", tt.envAuditDisabled)
			} else {
				os.Unsetenv("MCP_SYSTEMINFO_AUDIT_DISABLED")
			}
			
			if tt.envAuditFile != "" {
				os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", tt.envAuditFile)
			} else {
				os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
			}
			
			// Clean up after test
			defer func() {
				if origAuditDisabled != "" {
					os.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", origAuditDisabled)
				} else {
					os.Unsetenv("MCP_SYSTEMINFO_AUDIT_DISABLED")
				}
				
				if origAuditFile != "" {
					os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", origAuditFile)
				} else {
					os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
				}
				
				// Close audit logger to release file
				CloseAuditLogger()
				
				// Remove test audit files
				if tt.envAuditFile != "" {
					os.Remove(tt.envAuditFile)
				}
				os.Remove("mcp-systeminfo-audit.log")
			}()
			
			// Initialize audit logger
			err := InitAuditLogger()
			if (err != nil) != tt.wantErr {
				t.Errorf("InitAuditLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			// Check that audit file was created if not disabled
			if tt.envAuditDisabled != "true" {
				auditFile := "mcp-systeminfo-audit.log"
				if tt.envAuditFile != "" {
					auditFile = tt.envAuditFile
				}
				
				if _, err := os.Stat(auditFile); os.IsNotExist(err) {
					t.Errorf("InitAuditLogger() audit file not created: %s", auditFile)
				}
				
				// Check that startup message was written
				if data, err := os.ReadFile(auditFile); err == nil {
					content := string(data)
					if !strings.Contains(content, "startup") {
						t.Error("InitAuditLogger() startup message not written")
					}
					
					if !strings.Contains(content, "mcp-systeminfo") {
						t.Error("InitAuditLogger() server name not written")
					}
				}
			}
		})
	}
}

// TestCloseAuditLogger tests the CloseAuditLogger function
func TestCloseAuditLogger(t *testing.T) {
	// Create a temporary audit file
	tempFile := filepath.Join(t.TempDir(), "test-audit.log")
	os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", tempFile)
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	
	// Close audit logger
	CloseAuditLogger()
	
	// Check that shutdown message was written
	if data, err := os.ReadFile(tempFile); err == nil {
		content := string(data)
		if !strings.Contains(content, "shutdown") {
			t.Error("CloseAuditLogger() shutdown message not written")
		}
	}
}

// TestAuditLog tests the auditLog function
func TestAuditLog(t *testing.T) {
	// Create a temporary audit file
	tempFile := filepath.Join(t.TempDir(), "test-audit.log")
	os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", tempFile)
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	defer CloseAuditLogger()
	
	// Test parameters
	operation := "test_operation"
	command := "test_command"
	workingDir := "/test/dir"
	user := "test_user"
	result := &CommandResult{
		ExitCode:   0,
		Stdout:     "test output",
		Stderr:     "",
		DurationMs: 100,
		Command:    command,
		Timeout:    false,
	}
	security := &SecurityResult{
		Valid:  true,
		Reason: "",
		Rule:   "",
	}
	durationMs := int64(100)
	success := true
	errorCode := 0
	errorType := ""
	
	// Log audit entry
	auditLog(operation, command, workingDir, user, result, security, durationMs, success, errorCode, errorType)
	
	// Read audit file and verify content
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}
	
	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	
	// Should have at least 2 lines (startup + our entry)
	if len(lines) < 2 {
		t.Errorf("Audit log has too few lines: got %d, want at least 2", len(lines))
	}
	
	// Parse the last line (our entry)
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Errorf("Failed to parse audit entry: %v", err)
	}
	
	// Verify entry fields
	if entry["event_type"] != "operation" {
		t.Errorf("auditLog() event_type = %v, want %v", entry["event_type"], "operation")
	}
	
	if entry["operation"] != operation {
		t.Errorf("auditLog() operation = %v, want %v", entry["operation"], operation)
	}
	
	if entry["command"] != command {
		t.Errorf("auditLog() command = %v, want %v", entry["command"], command)
	}
	
	if entry["working_dir"] != workingDir {
		t.Errorf("auditLog() working_dir = %v, want %v", entry["working_dir"], workingDir)
	}
	
	if entry["user"] != user {
		t.Errorf("auditLog() user = %v, want %v", entry["user"], user)
	}
	
	if entry["duration_ms"] != float64(durationMs) {
		t.Errorf("auditLog() duration_ms = %v, want %v", entry["duration_ms"], durationMs)
	}
	
	if entry["success"] != success {
		t.Errorf("auditLog() success = %v, want %v", entry["success"], success)
	}
	
	if entry["error_code"] != float64(errorCode) {
		t.Errorf("auditLog() error_code = %v, want %v", entry["error_code"], errorCode)
	}
	
	if entry["error_type"] != errorType {
		t.Errorf("auditLog() error_type = %v, want %v", entry["error_type"], errorType)
	}
	
	if entry["server_name"] != "mcp-systeminfo" {
		t.Errorf("auditLog() server_name = %v, want %v", entry["server_name"], "mcp-systeminfo")
	}
	
	if entry["server_version"] != "1.0.0" {
		t.Errorf("auditLog() server_version = %v, want %v", entry["server_version"], "1.0.0")
	}
	
	// Check nested objects
	if resultObj, ok := entry["result"].(map[string]interface{}); ok {
		if resultObj["exit_code"] != float64(result.ExitCode) {
			t.Errorf("auditLog() result.exit_code = %v, want %v", resultObj["exit_code"], result.ExitCode)
		}
		
		if resultObj["stdout"] != result.Stdout {
			t.Errorf("auditLog() result.stdout = %v, want %v", resultObj["stdout"], result.Stdout)
		}
		
		if resultObj["command"] != result.Command {
			t.Errorf("auditLog() result.command = %v, want %v", resultObj["command"], result.Command)
		}
	} else {
		t.Error("auditLog() result is not a map")
	}
	
	if securityObj, ok := entry["security"].(map[string]interface{}); ok {
		if securityObj["valid"] != security.Valid {
			t.Errorf("auditLog() security.valid = %v, want %v", securityObj["valid"], security.Valid)
		}
	} else {
		t.Error("auditLog() security is not a map")
	}
}

// TestAuditLogDisabled tests that auditLog does nothing when audit is disabled
func TestAuditLogDisabled(t *testing.T) {
	// Disable audit
	os.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "true")
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_DISABLED")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	defer CloseAuditLogger()
	
	// Log audit entry
	auditLog("test_operation", "test_command", "/test/dir", "test_user", nil, nil, 100, true, 0, "")
	
	// Since audit is disabled, we can't verify the file doesn't exist
	// But we can at least verify the function doesn't panic
}

// TestWriteAuditEntry tests the writeAuditEntry function
func TestWriteAuditEntry(t *testing.T) {
	// Create a temporary audit file
	tempFile := filepath.Join(t.TempDir(), "test-audit.log")
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("Failed to create test audit file: %v", err)
	}
	defer file.Close()
	
	// Save original audit file and mutex
	origAuditFile := auditFile
	origAuditMutex := auditMutex
	defer func() {
		auditFile = origAuditFile
		auditMutex = origAuditMutex
	}()
	
	// Set test values
	auditFile = file
	auditMutex = sync.Mutex{}
	
	// Test entry
	entry := map[string]interface{}{
		"test_field": "test_value",
		"number":     42,
		"boolean":    true,
	}
	
	// Write entry
	err = writeAuditEntry(entry)
	if err != nil {
		t.Errorf("writeAuditEntry() error = %v", err)
	}
	
	// Read file and verify content
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}
	
	content := string(data)
	if !strings.Contains(content, "test_field") {
		t.Error("writeAuditEntry() field not written")
	}
	
	if !strings.Contains(content, "test_value") {
		t.Error("writeAuditEntry() value not written")
	}
	
	if !strings.Contains(content, "42") {
		t.Error("writeAuditEntry() number not written")
	}
	
	if !strings.Contains(content, "true") {
		t.Error("writeAuditEntry() boolean not written")
	}
	
	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("writeAuditEntry() wrote invalid JSON: %v", err)
	}
}

// TestGetAuditStats tests the GetAuditStats function
func TestGetAuditStats(t *testing.T) {
	// Create a temporary audit file
	tempFile := filepath.Join(t.TempDir(), "test-audit.log")
	os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", tempFile)
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	defer CloseAuditLogger()
	
	// Get stats
	stats := GetAuditStats()
	
	// Verify stats structure
	if stats == nil {
		t.Error("GetAuditStats() returned nil")
	}
	
	if enabled, ok := stats["enabled"].(bool); !ok || !enabled {
		t.Error("GetAuditStats() enabled should be true")
	}
	
	if auditFile, ok := stats["audit_file"].(string); !ok || auditFile != tempFile {
		t.Errorf("GetAuditStats() audit_file = %v, want %v", auditFile, tempFile)
	}
	
	if serverName, ok := stats["server_name"].(string); !ok || serverName != "mcp-systeminfo" {
		t.Errorf("GetAuditStats() server_name = %v, want %v", serverName, "mcp-systeminfo")
	}
	
	// File stats should be present
	if _, ok := stats["file_size"]; !ok {
		t.Error("GetAuditStats() missing file_size")
	}
	
	if _, ok := stats["file_modified"]; !ok {
		t.Error("GetAuditStats() missing file_modified")
	}
}

// TestGetAuditStatsDisabled tests GetAuditStats when audit is disabled
func TestGetAuditStatsDisabled(t *testing.T) {
	// Disable audit
	os.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "true")
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_DISABLED")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	defer CloseAuditLogger()
	
	// Get stats
	stats := GetAuditStats()
	
	// Verify stats
	if enabled, ok := stats["enabled"].(bool); !ok || enabled {
		t.Error("GetAuditStats() enabled should be false")
	}
	
	if serverName, ok := stats["server_name"].(string); !ok || serverName != "mcp-systeminfo" {
		t.Errorf("GetAuditStats() server_name = %v, want %v", serverName, "mcp-systeminfo")
	}
}

// TestAuditLogConcurrency tests concurrent audit logging
func TestAuditLogConcurrency(t *testing.T) {
	// Create a temporary audit file
	tempFile := filepath.Join(t.TempDir(), "test-audit.log")
	os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", tempFile)
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	defer CloseAuditLogger()
	
	// Number of goroutines
	const numGoroutines = 10
	const numLogsPerGoroutine = 5
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	// Start multiple goroutines logging concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < numLogsPerGoroutine; j++ {
				auditLog(
					"test_operation",
					"test_command",
					"/test/dir",
					"test_user",
					nil,
					nil,
					100,
					true,
					0,
					"",
				)
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	// Read audit file and verify all entries were written
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}
	
	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	
	// Should have 1 (startup) + numGoroutines * numLogsPerGoroutine entries
	expectedLines := 1 + numGoroutines*numLogsPerGoroutine
	if len(lines) < expectedLines {
		t.Errorf("Audit log has too few lines: got %d, want at least %d", len(lines), expectedLines)
	}
	
	// Verify all entries are valid JSON
	for i, line := range lines {
		if line == "" {
			continue
		}
		
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}
	}
}

// TestAuditLogTimestamp tests that audit entries have correct timestamps
func TestAuditLogTimestamp(t *testing.T) {
	// Create a temporary audit file
	tempFile := filepath.Join(t.TempDir(), "test-audit.log")
	os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", tempFile)
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("InitAuditLogger() failed: %v", err)
	}
	defer CloseAuditLogger()
	
	// Record time before logging
	before := time.Now().UTC()
	
	// Log audit entry
	auditLog("test_operation", "test_command", "/test/dir", "test_user", nil, nil, 100, true, 0, "")
	
	// Record time after logging
	after := time.Now().UTC()
	
	// Read audit file and verify timestamp
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read audit file: %v", err)
	}
	
	content := string(data)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	
	// Parse the last line (our entry)
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("Failed to parse audit entry: %v", err)
	}
	
	// Parse timestamp
	timestampStr, ok := entry["timestamp"].(string)
	if !ok {
		t.Fatal("auditLog() timestamp not found or not a string")
	}
	
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}
	
	// Verify timestamp is within expected range
	if timestamp.Before(before) {
		t.Error("auditLog() timestamp is before logging time")
	}
	
	if timestamp.After(after) {
		t.Error("auditLog() timestamp is after logging time")
	}
}