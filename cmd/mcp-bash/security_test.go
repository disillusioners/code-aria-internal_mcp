package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name            string
		command         string
		allowShellAccess bool
		wantValid       bool
		expectedRule    string
	}{
		{
			name:            "valid simple command",
			command:         "echo hello",
			allowShellAccess: false,
			wantValid:       true,
		},
		{
			name:            "valid command with shell access",
			command:         "echo hello | cat",
			allowShellAccess: true,
			wantValid:       true,
		},
		{
			name:            "blocked command - sudo",
			command:         "sudo rm -rf /",
			allowShellAccess: false,
			wantValid:       false,
			expectedRule:    "blocked_pattern",
		},
		{
			name:            "blocked command - rm -rf /",
			command:         "rm -rf /",
			allowShellAccess: false,
			wantValid:       false,
			expectedRule:    "blocked_pattern",
		},
		{
			name:            "command too long",
			command:         string(make([]byte, 1001)),
			allowShellAccess: false,
			wantValid:       false,
			expectedRule:    "max_length",
		},
		{
			name:            "empty command",
			command:         "",
			allowShellAccess: false,
			wantValid:       false,
			expectedRule:    "empty_command",
		},
		{
			name:            "disallowed command",
			command:         "malicious_command",
			allowShellAccess: false,
			wantValid:       false,
			expectedRule:    "allowed_commands",
		},
		{
			name:            "shell features without permission",
			command:         "echo hello | cat",
			allowShellAccess: false,
			wantValid:       false,
			expectedRule:    "shell_access",
		},
		{
			name:            "valid ls command",
			command:         "ls -la",
			allowShellAccess: false,
			wantValid:       true,
		},
		{
			name:            "valid cat command",
			command:         "cat file.txt",
			allowShellAccess: false,
			wantValid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCommand(tt.command, tt.allowShellAccess)

			if result.Valid != tt.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v. Reason: %s", tt.wantValid, result.Valid, result.Reason)
			}

			if !tt.wantValid && tt.expectedRule != "" && result.Rule != tt.expectedRule {
				t.Errorf("Expected rule %s, got %s", tt.expectedRule, result.Rule)
			}
		})
	}
}

func TestValidateScript(t *testing.T) {
	tests := []struct {
		name         string
		script       string
		wantValid    bool
		expectedRule string
	}{
		{
			name:      "valid simple script",
			script:    "#!/bin/bash\necho 'hello'",
			wantValid: true,
		},
		{
			name:      "valid multi-line script",
			script:    "#!/bin/bash\necho 'line1'\necho 'line2'\necho 'line3'",
			wantValid: true,
		},
		{
			name:         "script with blocked pattern",
			script:       "#!/bin/bash\nsudo rm -rf /",
			wantValid:    false,
			expectedRule: "blocked_pattern",
		},
		{
			name:         "script too long",
			script:       string(make([]byte, 10001)),
			wantValid:    false,
			expectedRule: "max_length",
		},
		{
			name:      "script with comments",
			script:    "#!/bin/bash\n# This is a comment\necho 'hello'",
			wantValid: true,
		},
		{
			name:      "script with empty lines",
			script:    "#!/bin/bash\n\necho 'hello'\n\n",
			wantValid: true,
		},
		{
			name:         "script with disallowed command",
			script:       "#!/bin/bash\nmalicious_command",
			wantValid:    false,
			expectedRule: "allowed_commands",
		},
		{
			name:      "script with variable assignment",
			script:    "#!/bin/bash\nVAR=value\necho $VAR",
			wantValid: true,
		},
		{
			name:      "script with if statement",
			script:    "#!/bin/bash\nif [ -f file ]; then\necho 'exists'\nfi",
			wantValid: true,
		},
		{
			name:         "script with dangerous construct",
			script:       "#!/bin/bash\neval 'malicious'",
			wantValid:    false,
			expectedRule: "dangerous_constructs",
		},
		{
			name:         "script with shutdown command",
			script:       "#!/bin/bash\nshutdown -h now",
			wantValid:    false,
			expectedRule: "dangerous_constructs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateScript(tt.script)

			if result.Valid != tt.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v. Reason: %s", tt.wantValid, result.Valid, result.Reason)
			}

			if !tt.wantValid && tt.expectedRule != "" && result.Rule != tt.expectedRule {
				t.Errorf("Expected rule %s, got %s", tt.expectedRule, result.Rule)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal input",
			input:    "echo hello",
			expected: "echo hello",
		},
		{
			name:     "input with null bytes",
			input:    "echo\x00hello",
			expected: "echohello",
		},
		{
			name:     "input with control characters",
			input:    "echo\rhello",
			expected: "echohello",
		},
		{
			name:     "input with newlines",
			input:    "echo\nhello",
			expected: "echo\nhello",
		},
		{
			name:     "input with tabs",
			input:    "echo\thello",
			expected: "echo\thello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestContainsShellFeatures(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "simple command",
			command:  "echo hello",
			expected: false,
		},
		{
			name:     "command with pipe",
			command:  "echo hello | cat",
			expected: true,
		},
		{
			name:     "command with redirect",
			command:  "echo hello > file",
			expected: true,
		},
		{
			name:     "command with &&",
			command:  "echo hello && echo world",
			expected: true,
		},
		{
			name:     "command with ||",
			command:  "echo hello || echo world",
			expected: true,
		},
		{
			name:     "command with semicolon",
			command:  "echo hello; echo world",
			expected: true,
		},
		{
			name:     "command with variable expansion",
			command:  "echo $VAR",
			expected: true,
		},
		{
			name:     "command with command substitution",
			command:  "echo $(date)",
			expected: true,
		},
		{
			name:     "command with wildcard",
			command:  "ls *.txt",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsShellFeatures(tt.command)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for command: %s", tt.expected, result, tt.command)
			}
		})
	}
}

func TestContainsDangerousScriptConstructs(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		expected bool
	}{
		{
			name:     "safe script",
			script:   "#!/bin/bash\necho hello",
			expected: false,
		},
		{
			name:     "script with eval",
			script:   "#!/bin/bash\neval 'malicious'",
			expected: true,
		},
		{
			name:     "script with exec",
			script:   "#!/bin/bash\nexec something",
			expected: true,
		},
		{
			name:     "script with shutdown",
			script:   "#!/bin/bash\nshutdown -h now",
			expected: true,
		},
		{
			name:     "script with reboot",
			script:   "#!/bin/bash\nreboot",
			expected: true,
		},
		{
			name:     "script with sudo",
			script:   "#!/bin/bash\nsudo something",
			expected: true,
		},
		{
			name:     "script with su",
			script:   "#!/bin/bash\nsu root",
			expected: true,
		},
		{
			name:     "script with dangerous chmod",
			script:   "#!/bin/bash\nchmod 777 /",
			expected: true,
		},
		{
			name:     "script with crontab",
			script:   "#!/bin/bash\ncrontab -e",
			expected: true,
		},
		{
			name:     "script accessing /dev/",
			script:   "#!/bin/bash\ncat /dev/null",
			expected: true,
		},
		{
			name:     "script accessing /proc/",
			script:   "#!/bin/bash\ncat /proc/cpuinfo",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsDangerousScriptConstructs(tt.script)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for script: %s", tt.expected, result, tt.script)
			}
		})
	}
}

func TestValidateWorkingDirectory(t *testing.T) {
	testDir := t.TempDir()
	os.Setenv("REPO_PATH", testDir)
	defer os.Unsetenv("REPO_PATH")

	// Create a subdirectory
	subDir := filepath.Join(testDir, "subdir")
	os.MkdirAll(subDir, 0755)

	tests := []struct {
		name         string
		workingDir   string
		wantValid    bool
		expectedRule string
	}{
		{
			name:      "empty directory (uses REPO_PATH)",
			workingDir: "",
			wantValid: true,
		},
		{
			name:      "valid subdirectory",
			workingDir: "subdir",
			wantValid: true,
		},
		{
			name:      "valid absolute path within REPO_PATH",
			workingDir: subDir,
			wantValid: true,
		},
		{
			name:         "path traversal attempt",
			workingDir:   "../",
			wantValid:    false,
			expectedRule: "path_traversal",
		},
		{
			name:         "non-existent directory",
			workingDir:   "nonexistent",
			wantValid:    false,
			expectedRule: "directory_exists",
		},
		{
			name:         "path outside REPO_PATH",
			workingDir:   "/tmp",
			wantValid:    false,
			expectedRule: "path_restriction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateWorkingDirectory(tt.workingDir)

			if result.Valid != tt.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v. Reason: %s", tt.wantValid, result.Valid, result.Reason)
			}

			if !tt.wantValid && tt.expectedRule != "" && result.Rule != tt.expectedRule {
				t.Errorf("Expected rule %s, got %s", tt.expectedRule, result.Rule)
			}
		})
	}
}

func TestValidateEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		wantValid    bool
		expectedRule string
	}{
		{
			name:      "valid environment variables",
			envVars:   map[string]string{"TEST_VAR": "value", "ANOTHER_VAR": "value2"},
			wantValid: true,
		},
		{
			name:      "empty environment variables",
			envVars:   map[string]string{},
			wantValid: true,
		},
		{
			name:         "invalid variable name - lowercase",
			envVars:      map[string]string{"invalid": "value"},
			wantValid:    false,
			expectedRule: "env_var_format",
		},
		{
			name:         "invalid variable name - starts with number",
			envVars:      map[string]string{"123VAR": "value"},
			wantValid:    false,
			expectedRule: "env_var_format",
		},
		{
			name:         "dangerous variable - PATH",
			envVars:      map[string]string{"PATH": "/malicious"},
			wantValid:    false,
			expectedRule: "dangerous_env_var",
		},
		{
			name:         "dangerous variable - LD_PRELOAD",
			envVars:      map[string]string{"LD_PRELOAD": "/malicious.so"},
			wantValid:    false,
			expectedRule: "dangerous_env_var",
		},
		{
			name:         "dangerous variable - SHELL",
			envVars:      map[string]string{"SHELL": "/bin/sh"},
			wantValid:    false,
			expectedRule: "dangerous_env_var",
		},
		{
			name:      "valid variable with underscore",
			envVars:   map[string]string{"MY_TEST_VAR": "value"},
			wantValid: true,
		},
		{
			name:      "valid variable with numbers",
			envVars:   map[string]string{"VAR_123": "value"},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateEnvironmentVariables(tt.envVars)

			if result.Valid != tt.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v. Reason: %s", tt.wantValid, result.Valid, result.Reason)
			}

			if !tt.wantValid && tt.expectedRule != "" && result.Rule != tt.expectedRule {
				t.Errorf("Expected rule %s, got %s", tt.expectedRule, result.Rule)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name         string
		timeout      int
		isScript     bool
		wantValid    bool
		expectedRule string
	}{
		{
			name:      "valid timeout for command",
			timeout:   30,
			isScript:  false,
			wantValid: true,
		},
		{
			name:      "valid timeout for script",
			timeout:   60,
			isScript:  true,
			wantValid: true,
		},
		{
			name:         "zero timeout",
			timeout:      0,
			isScript:     false,
			wantValid:    false,
			expectedRule: "timeout_positive",
		},
		{
			name:         "negative timeout",
			timeout:      -1,
			isScript:     false,
			wantValid:    false,
			expectedRule: "timeout_positive",
		},
		{
			name:         "timeout exceeds max for command",
			timeout:      400,
			isScript:     false,
			wantValid:    false,
			expectedRule: "timeout_maximum",
		},
		{
			name:         "timeout exceeds max for script",
			timeout:      700,
			isScript:     true,
			wantValid:    false,
			expectedRule: "timeout_maximum",
		},
		{
			name:      "max timeout for command",
			timeout:   300,
			isScript:  false,
			wantValid: true,
		},
		{
			name:      "max timeout for script",
			timeout:   600,
			isScript:  true,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateTimeout(tt.timeout, tt.isScript)

			if result.Valid != tt.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v. Reason: %s", tt.wantValid, result.Valid, result.Reason)
			}

			if !tt.wantValid && tt.expectedRule != "" && result.Rule != tt.expectedRule {
				t.Errorf("Expected rule %s, got %s", tt.expectedRule, result.Rule)
			}
		})
	}
}


