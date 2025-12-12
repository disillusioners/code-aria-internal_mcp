package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitAuditLogger(t *testing.T) {
	// Save original state
	originalEnabled := auditEnabled
	originalLogFile := auditLogFile
	originalLogger := auditLogger

	defer func() {
		auditEnabled = originalEnabled
		auditLogFile = originalLogFile
		auditLogger = originalLogger
		if auditLogger != nil {
			CloseAuditLogger()
		}
	}()

	tests := []struct {
		name        string
		enabled     bool
		logFile     string
		expectError bool
	}{
		{
			name:        "audit disabled",
			enabled:     false,
			expectError: false,
		},
		{
			name:        "audit enabled with default path",
			enabled:     true,
			logFile:     "",
			expectError: false,
		},
		{
			name:        "audit enabled with custom path",
			enabled:     true,
			logFile:     filepath.Join(t.TempDir(), "test_audit.log"),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditEnabled = tt.enabled
			auditLogFile = tt.logFile

			// Set REPO_PATH for default log file path
			if tt.logFile == "" {
				testDir := t.TempDir()
				os.Setenv("REPO_PATH", testDir)
				defer os.Unsetenv("REPO_PATH")
			}

			err := InitAuditLogger()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if auditLogger != nil {
				CloseAuditLogger()
			}
		})
	}
}

func TestAuditLog(t *testing.T) {
	testDir := t.TempDir()
	testLogFile := filepath.Join(testDir, "test_audit.log")

	// Save original state
	originalEnabled := auditEnabled
	originalLogFile := auditLogFile
	originalLogger := auditLogger

	defer func() {
		auditEnabled = originalEnabled
		auditLogFile = originalLogFile
		auditLogger = originalLogger
		if auditLogger != nil {
			CloseAuditLogger()
		}
	}()

	// Initialize audit logger
	auditEnabled = true
	auditLogFile = testLogFile
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("Failed to initialize audit logger: %v", err)
	}
	defer CloseAuditLogger()

	// Test audit logging
	commandResult := &CommandResult{
		ExitCode:   0,
		Stdout:     "test output",
		Stderr:     "",
		DurationMs: 100,
		Command:    "echo test",
	}

	securityResult := &SecurityResult{
		Valid: true,
	}

	auditLog("execute_command", "echo test", "", testDir, nil, commandResult, securityResult, 100, true, 0, "")

	// Verify log file was created
	if _, err := os.Stat(testLogFile); os.IsNotExist(err) {
		t.Fatal("Audit log file was not created")
	}

	// Read and verify log entry
	data, err := os.ReadFile(testLogFile)
	if err != nil {
		t.Fatalf("Failed to read audit log file: %v", err)
	}

	var entry AuditLog
	lines := []byte{}
	for _, b := range data {
		if b == '\n' {
			if len(lines) > 0 {
				if err := json.Unmarshal(lines, &entry); err != nil {
					t.Fatalf("Failed to unmarshal audit log entry: %v", err)
				}
				break
			}
		} else {
			lines = append(lines, b)
		}
	}

	if entry.Operation != "execute_command" {
		t.Errorf("Expected operation 'execute_command', got '%s'", entry.Operation)
	}

	if entry.Command != "echo test" {
		t.Errorf("Expected command 'echo test', got '%s'", entry.Command)
	}

	if !entry.Success {
		t.Error("Expected success=true")
	}
}

func TestGetAuditStats(t *testing.T) {
	testDir := t.TempDir()
	testLogFile := filepath.Join(testDir, "test_audit.log")

	// Save original state
	originalEnabled := auditEnabled
	originalLogFile := auditLogFile
	originalLogger := auditLogger

	defer func() {
		auditEnabled = originalEnabled
		auditLogFile = originalLogFile
		auditLogger = originalLogger
		if auditLogger != nil {
			CloseAuditLogger()
		}
	}()

	// Initialize audit logger
	auditEnabled = true
	auditLogFile = testLogFile
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("Failed to initialize audit logger: %v", err)
	}
	defer CloseAuditLogger()

	// Create some audit entries
	commandResult := &CommandResult{
		ExitCode:   0,
		Stdout:     "test",
		DurationMs: 50,
		Command:    "echo test",
	}

	securityResult := &SecurityResult{Valid: true}

	auditLog("execute_command", "echo test1", "", testDir, nil, commandResult, securityResult, 50, true, 0, "")
	auditLog("execute_command", "echo test2", "", testDir, nil, commandResult, securityResult, 50, true, 0, "")
	auditLog("execute_script", "", "script content", testDir, nil, commandResult, securityResult, 100, false, -32003, "Execution")

	// Get stats
	stats, err := GetAuditStats()
	if err != nil {
		t.Fatalf("Failed to get audit stats: %v", err)
	}

	if stats["enabled"] != true {
		t.Error("Expected enabled=true")
	}

	if stats["total_entries"].(int) != 3 {
		t.Errorf("Expected 3 entries, got %d", stats["total_entries"])
	}

	operations, ok := stats["operations"].(map[string]int)
	if !ok {
		t.Fatal("Operations is not a map")
	}

	if operations["execute_command"] != 2 {
		t.Errorf("Expected 2 execute_command operations, got %d", operations["execute_command"])
	}

	if operations["execute_script"] != 1 {
		t.Errorf("Expected 1 execute_script operation, got %d", operations["execute_script"])
	}
}

func TestClearAuditLog(t *testing.T) {
	testDir := t.TempDir()
	testLogFile := filepath.Join(testDir, "test_audit.log")

	// Save original state
	originalEnabled := auditEnabled
	originalLogFile := auditLogFile
	originalLogger := auditLogger

	defer func() {
		auditEnabled = originalEnabled
		auditLogFile = originalLogFile
		auditLogger = originalLogger
		if auditLogger != nil {
			CloseAuditLogger()
		}
	}()

	// Initialize audit logger
	auditEnabled = true
	auditLogFile = testLogFile
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("Failed to initialize audit logger: %v", err)
	}

	// Create some entries
	commandResult := &CommandResult{
		ExitCode:   0,
		DurationMs: 50,
		Command:    "echo test",
	}
	securityResult := &SecurityResult{Valid: true}

	auditLog("execute_command", "echo test", "", testDir, nil, commandResult, securityResult, 50, true, 0, "")

	// Verify file exists
	if _, err := os.Stat(testLogFile); os.IsNotExist(err) {
		t.Fatal("Audit log file was not created")
	}

	// Clear log
	if err := ClearAuditLog(); err != nil {
		t.Fatalf("Failed to clear audit log: %v", err)
	}

	// Verify file was removed and recreated
	if _, err := os.Stat(testLogFile); os.IsNotExist(err) {
		t.Fatal("Audit log file should be recreated after clearing")
	}

	// Verify it's empty
	stats, err := GetAuditStats()
	if err != nil {
		t.Fatalf("Failed to get audit stats: %v", err)
	}

	if stats["total_entries"].(int) != 0 {
		t.Errorf("Expected 0 entries after clearing, got %d", stats["total_entries"])
	}
}

func TestSearchAuditLog(t *testing.T) {
	testDir := t.TempDir()
	testLogFile := filepath.Join(testDir, "test_audit.log")

	// Save original state
	originalEnabled := auditEnabled
	originalLogFile := auditLogFile
	originalLogger := auditLogger

	defer func() {
		auditEnabled = originalEnabled
		auditLogFile = originalLogFile
		auditLogger = originalLogger
		if auditLogger != nil {
			CloseAuditLogger()
		}
	}()

	// Initialize audit logger
	auditEnabled = true
	auditLogFile = testLogFile
	if err := InitAuditLogger(); err != nil {
		t.Fatalf("Failed to initialize audit logger: %v", err)
	}
	defer CloseAuditLogger()

	// Create test entries
	commandResult1 := &CommandResult{
		ExitCode:   0,
		DurationMs: 50,
		Command:    "echo success",
	}
	commandResult2 := &CommandResult{
		ExitCode:   1,
		DurationMs: 100,
		Command:    "echo failure",
	}
	securityResult := &SecurityResult{Valid: true}

	auditLog("execute_command", "echo success", "", testDir, nil, commandResult1, securityResult, 50, true, 0, "")
	auditLog("execute_command", "echo failure", "", testDir, nil, commandResult2, securityResult, 100, false, -32003, "Execution")
	auditLog("execute_script", "", "script content", testDir, nil, commandResult1, securityResult, 50, true, 0, "")

	tests := []struct {
		name          string
		criteria      map[string]interface{}
		expectedCount int
	}{
		{
			name: "search by operation",
			criteria: map[string]interface{}{
				"operation": "execute_command",
			},
			expectedCount: 2,
		},
		{
			name: "search by success",
			criteria: map[string]interface{}{
				"success": true,
			},
			expectedCount: 2,
		},
		{
			name: "search by error type",
			criteria: map[string]interface{}{
				"error_type": "Execution",
			},
			expectedCount: 1,
		},
		{
			name: "search by command contains",
			criteria: map[string]interface{}{
				"command_contains": "success",
			},
			expectedCount: 1,
		},
		{
			name: "combined criteria",
			criteria: map[string]interface{}{
				"operation": "execute_command",
				"success":   true,
			},
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := SearchAuditLog(tt.criteria)
			if err != nil {
				t.Fatalf("Failed to search audit log: %v", err)
			}

			if len(entries) != tt.expectedCount {
				t.Errorf("Expected %d entries, got %d", tt.expectedCount, len(entries))
			}
		})
	}
}

func TestMatchesCriteria(t *testing.T) {
	entry := AuditLog{
		Timestamp:  time.Now(),
		Operation:  "execute_command",
		Command:    "echo test",
		User:       "testuser",
		Success:    true,
		ErrorType:  "",
		Security:   &SecurityResult{Valid: true},
	}

	tests := []struct {
		name     string
		criteria map[string]interface{}
		expected bool
	}{
		{
			name: "match by operation",
			criteria: map[string]interface{}{
				"operation": "execute_command",
			},
			expected: true,
		},
		{
			name: "mismatch by operation",
			criteria: map[string]interface{}{
				"operation": "execute_script",
			},
			expected: false,
		},
		{
			name: "match by success",
			criteria: map[string]interface{}{
				"success": true,
			},
			expected: true,
		},
		{
			name: "mismatch by success",
			criteria: map[string]interface{}{
				"success": false,
			},
			expected: false,
		},
		{
			name: "match by command contains",
			criteria: map[string]interface{}{
				"command_contains": "test",
			},
			expected: true,
		},
		{
			name: "mismatch by command contains",
			criteria: map[string]interface{}{
				"command_contains": "nonexistent",
			},
			expected: false,
		},
		{
			name: "match by security violation false",
			criteria: map[string]interface{}{
				"security_violation": false,
			},
			expected: true,
		},
		{
			name: "match by security violation true",
			criteria: map[string]interface{}{
				"security_violation": true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesCriteria(entry, tt.criteria)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetAuditConfiguration(t *testing.T) {
	// Save original state
	originalEnabled := auditEnabled
	originalLogFile := auditLogFile
	originalLogger := auditLogger

	defer func() {
		auditEnabled = originalEnabled
		auditLogFile = originalLogFile
		auditLogger = originalLogger
		if auditLogger != nil {
			CloseAuditLogger()
		}
	}()

	testLogFile := filepath.Join(t.TempDir(), "test_audit.log")

	SetAuditConfiguration(true, testLogFile)

	if !auditEnabled {
		t.Error("Expected audit to be enabled")
	}

	if auditLogFile != testLogFile {
		t.Errorf("Expected log file %s, got %s", testLogFile, auditLogFile)
	}

	SetAuditConfiguration(false, "")

	if auditEnabled {
		t.Error("Expected audit to be disabled")
	}
}

func TestAuditLogEntrySerialization(t *testing.T) {
	entry := AuditLog{
		Timestamp:  time.Now().UTC(),
		Operation:  "execute_command",
		Command:    "echo test",
		User:       "testuser",
		WorkingDir: "/tmp",
		Environment: map[string]string{
			"VAR1": "value1",
		},
		Result: &CommandResult{
			ExitCode:   0,
			Stdout:     "test",
			DurationMs: 100,
			Command:    "echo test",
		},
		Security:   &SecurityResult{Valid: true},
		DurationMs: 100,
		Success:    true,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal AuditLog: %v", err)
	}

	var unmarshaled AuditLog
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal AuditLog: %v", err)
	}

	if unmarshaled.Operation != entry.Operation {
		t.Errorf("Operation mismatch: expected %s, got %s", entry.Operation, unmarshaled.Operation)
	}

	if unmarshaled.Command != entry.Command {
		t.Errorf("Command mismatch: expected %s, got %s", entry.Command, unmarshaled.Command)
	}

	if unmarshaled.Success != entry.Success {
		t.Errorf("Success mismatch: expected %v, got %v", entry.Success, unmarshaled.Success)
	}
}



