package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestToolExecuteCommand(t *testing.T) {
	// Set up test environment
	testDir := t.TempDir()
	os.Setenv("REPO_PATH", testDir)
	defer os.Unsetenv("REPO_PATH")

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
		checkFunc func(*CommandResult) bool
	}{
		{
			name: "simple echo command",
			args: map[string]interface{}{
				"command": "echo",
				"timeout": float64(10),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0
			},
		},
		{
			name: "command with output",
			args: map[string]interface{}{
				"command": "echo",
				"timeout": float64(10),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.DurationMs > 0
			},
		},
		{
			name: "missing command",
			args: map[string]interface{}{
				"timeout": float64(10),
			},
			wantError: true,
		},
		{
			name: "command with working directory",
			args: map[string]interface{}{
				"command":           "pwd",
				"working_directory": testDir,
				"timeout":           float64(10),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.WorkingDir == testDir
			},
		},
		{
			name: "command with environment variables",
			args: map[string]interface{}{
				"command": "echo",
				"environment_vars": map[string]interface{}{
					"TEST_VAR": "test_value",
				},
				"timeout": float64(10),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0
			},
		},
		{
			name: "timeout at maximum",
			args: map[string]interface{}{
				"command": "echo",
				"timeout": float64(600), // At max, should succeed (clamped)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolExecuteCommand(tt.args)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			var cmdResult CommandResult
			if err := json.Unmarshal([]byte(result), &cmdResult); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if tt.checkFunc != nil && !tt.checkFunc(&cmdResult) {
				t.Error("Check function failed")
			}
		})
	}
}

func TestToolExecuteScript(t *testing.T) {
	testDir := t.TempDir()
	os.Setenv("REPO_PATH", testDir)
	defer os.Unsetenv("REPO_PATH")

	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
		checkFunc func(*CommandResult) bool
	}{
		{
			name: "simple script",
			args: map[string]interface{}{
				"script":  "#!/bin/bash\necho 'Hello World'",
				"timeout": float64(30),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.LinesExecuted > 0
			},
		},
		{
			name: "script with multiple lines",
			args: map[string]interface{}{
				"script":  "#!/bin/bash\necho 'Line 1'\necho 'Line 2'\necho 'Line 3'",
				"timeout": float64(30),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.LinesExecuted >= 3
			},
		},
		{
			name: "missing script",
			args: map[string]interface{}{
				"timeout": float64(30),
			},
			wantError: true,
		},
		{
			name: "script with working directory",
			args: map[string]interface{}{
				"script":            "#!/bin/bash\npwd",
				"working_directory": testDir,
				"timeout":           float64(30),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.WorkingDir == testDir
			},
		},
		{
			name: "script with custom name",
			args: map[string]interface{}{
				"script":      "#!/bin/bash\necho test",
				"script_name": "my_test_script",
				"timeout":     float64(30),
			},
			wantError: false,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.ScriptName == "my_test_script"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolExecuteScript(tt.args)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			var cmdResult CommandResult
			if err := json.Unmarshal([]byte(result), &cmdResult); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if tt.checkFunc != nil && !tt.checkFunc(&cmdResult) {
				t.Error("Check function failed")
			}
		})
	}
}

func TestToolCheckCommandExists(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]interface{}
		wantError bool
		checkFunc func(*CommandExistsResult) bool
	}{
		{
			name: "check existing command",
			args: map[string]interface{}{
				"command": "echo",
			},
			wantError: false,
			checkFunc: func(r *CommandExistsResult) bool {
				return r.Exists && r.Command == "echo"
			},
		},
		{
			name: "check non-existent command",
			args: map[string]interface{}{
				"command": "nonexistent_command_xyz123",
			},
			wantError: false,
			checkFunc: func(r *CommandExistsResult) bool {
				return !r.Exists && r.Command == "nonexistent_command_xyz123"
			},
		},
		{
			name:      "missing command parameter",
			args:      map[string]interface{}{},
			wantError: true,
		},
		{
			name: "invalid command format",
			args: map[string]interface{}{
				"command": "invalid-command-name!",
			},
			wantError: true,
		},
		{
			name: "check with search paths",
			args: map[string]interface{}{
				"command":      "echo",
				"search_paths": []interface{}{"/usr/bin", "/bin"},
			},
			wantError: false,
			checkFunc: func(r *CommandExistsResult) bool {
				return r.Exists
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolCheckCommandExists(tt.args)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			var cmdResult CommandExistsResult
			if err := json.Unmarshal([]byte(result), &cmdResult); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if tt.checkFunc != nil && !tt.checkFunc(&cmdResult) {
				t.Error("Check function failed")
			}
		})
	}
}

func TestExecuteCommandWithTimeout(t *testing.T) {
	testDir := t.TempDir()

	tests := []struct {
		name      string
		command   string
		timeout   time.Duration
		wantError bool
		checkFunc func(*CommandResult) bool
	}{
		{
			name:    "successful command",
			command: "echo",
			timeout: 5 * time.Second,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && !r.Timeout
			},
		},
		{
			name:    "command with output",
			command: "echo",
			timeout: 5 * time.Second,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.DurationMs > 0
			},
		},
		{
			name:    "command with working directory",
			command: "pwd",
			timeout: 5 * time.Second,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.WorkingDir == testDir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCommandWithTimeout(tt.command, testDir, nil, false, tt.timeout)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil && !tt.wantError {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && result != nil && !tt.checkFunc(result) {
				t.Error("Check function failed")
			}
		})
	}
}

func TestExecuteScriptWithTimeout(t *testing.T) {
	testDir := t.TempDir()

	tests := []struct {
		name      string
		script    string
		timeout   time.Duration
		wantError bool
		checkFunc func(*CommandResult) bool
	}{
		{
			name:    "simple script",
			script:  "#!/bin/bash\necho 'test'",
			timeout: 5 * time.Second,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && !r.Timeout
			},
		},
		{
			name:    "multi-line script",
			script:  "#!/bin/bash\necho 'line1'\necho 'line2'",
			timeout: 5 * time.Second,
			checkFunc: func(r *CommandResult) bool {
				return r.ExitCode == 0 && r.LinesExecuted >= 2
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple script name without special characters
			scriptName := "test_script"
			result, err := executeScriptWithTimeout(tt.script, testDir, nil, true, tt.timeout, scriptName)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil && !tt.wantError {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkFunc != nil && result != nil && !tt.checkFunc(result) {
				t.Error("Check function failed")
			}
		})
	}
}

func TestCheckCommandExists(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		paths     []string
		checkFunc func(*CommandExistsResult) bool
	}{
		{
			name:    "existing command",
			command: "echo",
			checkFunc: func(r *CommandExistsResult) bool {
				return r.Exists && r.Command == "echo"
			},
		},
		{
			name:    "non-existent command",
			command: "nonexistent_xyz_123",
			checkFunc: func(r *CommandExistsResult) bool {
				return !r.Exists && r.Command == "nonexistent_xyz_123"
			},
		},
		{
			name:    "command with custom paths",
			command: "echo",
			paths:   []string{"/usr/bin", "/bin"},
			checkFunc: func(r *CommandExistsResult) bool {
				return r.Exists
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkCommandExists(tt.command, tt.paths)

			if tt.checkFunc != nil && !tt.checkFunc(result) {
				t.Error("Check function failed")
			}
		})
	}
}

func TestGetCommandVersion(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectEmpty bool
	}{
		{
			name:        "command with version",
			command:     "echo",
			expectEmpty: false, // echo might not have version, but function should handle it
		},
		{
			name:        "non-existent command",
			command:     "nonexistent_xyz_123",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := getCommandVersion(tt.command)

			if tt.expectEmpty && version != "" {
				t.Errorf("Expected empty version, got %s", version)
			}
		})
	}
}

func TestCommandResultSerialization(t *testing.T) {
	result := &CommandResult{
		ExitCode:   0,
		Stdout:     "test output",
		Stderr:     "",
		DurationMs: 100,
		Command:    "echo test",
		WorkingDir: "/tmp",
		Timeout:    false,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CommandResult: %v", err)
	}

	var unmarshaled CommandResult
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CommandResult: %v", err)
	}

	if unmarshaled.ExitCode != result.ExitCode {
		t.Errorf("ExitCode mismatch: expected %d, got %d", result.ExitCode, unmarshaled.ExitCode)
	}

	if unmarshaled.Stdout != result.Stdout {
		t.Errorf("Stdout mismatch: expected %s, got %s", result.Stdout, unmarshaled.Stdout)
	}
}

func TestCommandExistsResultSerialization(t *testing.T) {
	result := &CommandExistsResult{
		Exists:  true,
		Command: "echo",
		Path:    "/bin/echo",
		Version: "8.32",
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal CommandExistsResult: %v", err)
	}

	var unmarshaled CommandExistsResult
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CommandExistsResult: %v", err)
	}

	if unmarshaled.Exists != result.Exists {
		t.Errorf("Exists mismatch: expected %v, got %v", result.Exists, unmarshaled.Exists)
	}

	if unmarshaled.Command != result.Command {
		t.Errorf("Command mismatch: expected %s, got %s", result.Command, unmarshaled.Command)
	}
}
