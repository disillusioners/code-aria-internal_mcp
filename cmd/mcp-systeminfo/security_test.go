package main

import (
	"strings"
	"testing"
	"time"
)

// TestValidateCommand tests the validateCommand function
func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name             string
		command          string
		allowShellAccess bool
		wantValid        bool
		wantReason       string
		wantRule         string
	}{
		{
			name:             "Valid allowed command",
			command:          "uname",
			allowShellAccess: false,
			wantValid:        true,
		},
		{
			name:             "Valid allowed command with args",
			command:          "uname -a",
			allowShellAccess: false,
			wantValid:        true,
		},
		{
			name:             "Command too long",
			command:          strings.Repeat("a", 501),
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Command too long (max 500 characters)",
			wantRule:         "max_length",
		},
		{
			name:             "Invalid UTF-8",
			command:          string([]byte{0xff, 0xfe, 0xfd}),
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Command contains invalid UTF-8 characters",
			wantRule:         "utf8_validation",
		},
		{
			name:             "Blocked pattern - rm",
			command:          "rm -rf /",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Blocked pattern detected: rm\\s+",
			wantRule:         "blocked_pattern",
		},
		{
			name:             "Blocked pattern - dd",
			command:          "dd if=/dev/zero of=/dev/sda",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Blocked pattern detected: dd\\s+",
			wantRule:         "blocked_pattern",
		},
		{
			name:             "Blocked pattern - format",
			command:          "format C:",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Blocked pattern detected: format",
			wantRule:         "blocked_pattern",
		},
		{
			name:             "Blocked pattern - shutdown",
			command:          "shutdown now",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Blocked pattern detected: shutdown",
			wantRule:         "blocked_pattern",
		},
		{
			name:             "Blocked pattern - passwd",
			command:          "passwd user",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Blocked pattern detected: passwd",
			wantRule:         "blocked_pattern",
		},
		{
			name:             "Blocked pattern - sudo",
			command:          "sudo apt update",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Blocked pattern detected: sudo\\s+",
			wantRule:         "blocked_pattern",
		},
		{
			name:             "Not allowed command",
			command:          "maliciouscommand",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Command not allowed: maliciouscommand",
			wantRule:         "allowed_commands",
		},
		{
			name:             "Shell features not allowed",
			command:          "ls | grep test",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Shell features allowed",
			command:          "ls | grep test",
			allowShellAccess: true,
			wantValid:        true,
		},
		{
			name:             "Command with redirect",
			command:          "ls > file.txt",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with append redirect",
			command:          "echo test >> file.txt",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with semicolon",
			command:          "ls; rm -rf /",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with AND operator",
			command:          "ls && echo done",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with OR operator",
			command:          "ls || echo failed",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with background execution",
			command:          "sleep 10 &",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with command substitution",
			command:          "echo $(whoami)",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with parameter expansion",
			command:          "echo ${HOME}",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with parentheses",
			command:          "(ls; echo done)",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with braces",
			command:          "echo {1,2,3}",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with wildcard",
			command:          "ls *.txt",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with question mark",
			command:          "ls file?.txt",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with character class",
			command:          "ls file[0-9].txt",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Shell features not allowed",
			wantRule:         "shell_access",
		},
		{
			name:             "Command with option flag",
			command:          "-la",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Unable to determine base command",
			wantRule:         "command_extraction",
		},
		{
			name:             "Empty command",
			command:          "",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Unable to determine base command",
			wantRule:         "command_extraction",
		},
		{
			name:             "Command with invalid characters",
			command:          "ls\x00test",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Command contains invalid characters",
			wantRule:         "character_validation",
		},
		{
			name:             "Command with carriage return",
			command:          "ls\rtest",
			allowShellAccess: false,
			wantValid:        false,
			wantReason:       "Command contains invalid characters",
			wantRule:         "character_validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateCommand(tt.command, tt.allowShellAccess)

			if got.Valid != tt.wantValid {
				t.Errorf("validateCommand() valid = %v, want %v", got.Valid, tt.wantValid)
			}

			if !tt.wantValid {
				if got.Reason != tt.wantReason {
					t.Errorf("validateCommand() reason = %v, want %v", got.Reason, tt.wantReason)
				}

				if got.Rule != tt.wantRule {
					t.Errorf("validateCommand() rule = %v, want %v", got.Rule, tt.wantRule)
				}
			}
		})
	}
}

// TestSanitizeInput tests the sanitizeInput function
func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal input",
			input:    "ls -la",
			expected: "ls -la",
		},
		{
			name:     "Input with null byte",
			input:    "ls\x00-la",
			expected: "ls-la",
		},
		{
			name:     "Input with carriage return",
			input:    "ls\r-la",
			expected: "ls-la",
		},
		{
			name:     "Input with tab",
			input:    "ls\t-la",
			expected: "ls\t-la",
		},
		{
			name:     "Input with newline",
			input:    "ls\n-la",
			expected: "ls\n-la",
		},
		{
			name:     "Input with control characters",
			input:    "ls\x01\x02\x03-la",
			expected: "ls-la",
		},
		{
			name:     "Input with valid characters",
			input:    "ls -la /home/user",
			expected: "ls -la /home/user",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Input with only control characters",
			input:    "\x00\x01\x02\x03",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeInput(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeInput() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestExtractBaseCommand tests the extractBaseCommand function
func TestExtractBaseCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "Simple command",
			command:  "ls",
			expected: "ls",
		},
		{
			name:     "Command with arguments",
			command:  "ls -la",
			expected: "ls",
		},
		{
			name:     "Command with path",
			command:  "/usr/bin/ls -la",
			expected: "/usr/bin/ls",
		},
		{
			name:     "Command with leading pipes",
			command:  "| ls -la",
			expected: "ls",
		},
		{
			name:     "Command with leading semicolons",
			command:  "; ls -la",
			expected: "ls",
		},
		{
			name:     "Command with pipeline",
			command:  "ls -la | grep test",
			expected: "ls",
		},
		{
			name:     "Command with multiple pipes",
			command:  "ls | grep test | wc -l",
			expected: "ls",
		},
		{
			name:     "Command with mixed pipes and semicolons",
			command:  "ls; cat file | grep test",
			expected: "ls",
		},
		{
			name:     "Command with option flag",
			command:  "-la",
			expected: "",
		},
		{
			name:     "Empty command",
			command:  "",
			expected: "",
		},
		{
			name:     "Command with only whitespace",
			command:  "   ",
			expected: "",
		},
		{
			name:     "Command with trailing pipes",
			command:  "ls |",
			expected: "ls",
		},
		{
			name:     "Command with trailing semicolons",
			command:  "ls ;",
			expected: "ls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBaseCommand(tt.command)
			if got != tt.expected {
				t.Errorf("extractBaseCommand() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestContainsShellFeatures tests the containsShellFeatures function
func TestContainsShellFeatures(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "Simple command",
			command:  "ls -la",
			expected: false,
		},
		{
			name:     "Command with pipeline",
			command:  "ls | grep test",
			expected: true,
		},
		{
			name:     "Command with redirect",
			command:  "ls > file.txt",
			expected: true,
		},
		{
			name:     "Command with input redirect",
			command:  "sort < file.txt",
			expected: true,
		},
		{
			name:     "Command with append redirect",
			command:  "echo test >> file.txt",
			expected: true,
		},
		{
			name:     "Command with semicolon",
			command:  "ls; echo done",
			expected: true,
		},
		{
			name:     "Command with AND operator",
			command:  "ls && echo done",
			expected: true,
		},
		{
			name:     "Command with OR operator",
			command:  "ls || echo failed",
			expected: true,
		},
		{
			name:     "Command with background execution",
			command:  "sleep 10 &",
			expected: true,
		},
		{
			name:     "Command with command substitution",
			command:  "echo $(whoami)",
			expected: true,
		},
		{
			name:     "Command with parameter expansion",
			command:  "echo ${HOME}",
			expected: true,
		},
		{
			name:     "Command with parentheses",
			command:  "(ls; echo done)",
			expected: true,
		},
		{
			name:     "Command with braces",
			command:  "echo {1,2,3}",
			expected: true,
		},
		{
			name:     "Command with wildcard",
			command:  "ls *.txt",
			expected: true,
		},
		{
			name:     "Command with question mark",
			command:  "ls file?.txt",
			expected: true,
		},
		{
			name:     "Command with character class",
			command:  "ls file[0-9].txt",
			expected: true,
		},
		{
			name:     "Command with pipeline in string",
			command:  "echo \"test | pipeline\"",
			expected: true,
		},
		{
			name:     "Command with pipeline in single quotes",
			command:  "echo 'test | pipeline'",
			expected: true,
		},
		{
			name:     "Complex command with multiple features",
			command:  "find / -name \"*.log\" 2>/dev/null | xargs grep -l \"error\" && echo \"Found errors\" || echo \"No errors\"",
			expected: true,
		},
		{
			name:     "Command with just text",
			command:  "echo hello world",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsShellFeatures(tt.command)
			if got != tt.expected {
				t.Errorf("containsShellFeatures() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestExecuteCommandWithTimeout tests the executeCommandWithTimeout function
func TestExecuteCommandWithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		timeout     int
		wantErr     bool
		wantTimeout bool
	}{
		{
			name:        "Simple command",
			command:     "echo hello",
			timeout:     5,
			wantErr:     false,
			wantTimeout: false,
		},
		{
			name:        "Command that fails",
			command:     "exit 1",
			timeout:     5,
			wantErr:     true,
			wantTimeout: false,
		},
		{
			name:        "Command that times out",
			command:     "sleep 10",
			timeout:     1,
			wantErr:     true,
			wantTimeout: true,
		},
		{
			name:        "Non-existent command",
			command:     "nonexistentcommand12345",
			timeout:     5,
			wantErr:     true,
			wantTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeCommandWithTimeout(tt.command, tt.timeout)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeCommandWithTimeout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantTimeout && !got.Timeout {
				t.Error("executeCommandWithTimeout() expected timeout but got.Timeout = false")
			}

			if !tt.wantTimeout && got.Timeout {
				t.Error("executeCommandWithTimeout() unexpected timeout but got.Timeout = true")
			}

			if got.Command != tt.command {
				t.Errorf("executeCommandWithTimeout() command = %v, want %v", got.Command, tt.command)
			}

			if got.DurationMs < 0 {
				t.Error("executeCommandWithTimeout() duration is negative")
			}
		})
	}
}

// TestDefaultSecurityPolicy tests the defaultSecurityPolicy global variable
func TestDefaultSecurityPolicy(t *testing.T) {
	// Test that defaultSecurityPolicy is properly initialized
	if defaultSecurityPolicy.AllowedCommands == nil {
		t.Error("defaultSecurityPolicy.AllowedCommands is nil")
	}

	if len(defaultSecurityPolicy.AllowedCommands) == 0 {
		t.Error("defaultSecurityPolicy.AllowedCommands is empty")
	}

	// Check some expected allowed commands
	expectedCommands := []string{
		"uname", "hostname", "whoami", "id", "lscpu",
		"free", "df", "mount", "lsblk", "lspci",
		"lsusb", "dmidecode", "uptime", "date",
	}

	for _, cmd := range expectedCommands {
		if !defaultSecurityPolicy.AllowedCommands[cmd] {
			t.Errorf("Expected command %s not in allowed commands", cmd)
		}
	}

	// Check blocked patterns
	if len(defaultSecurityPolicy.BlockedPatterns) == 0 {
		t.Error("defaultSecurityPolicy.BlockedPatterns is empty")
	}

	// Check some expected blocked patterns
	expectedPatterns := []string{
		`rm\s+`, `dd\s+`, `mkfs`, `fdisk`, `format`,
		`del\s+`, `rmdir`, `shutdown`, `reboot`, `halt`,
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, blockedPattern := range defaultSecurityPolicy.BlockedPatterns {
			if blockedPattern == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected pattern %s not in blocked patterns", pattern)
		}
	}

	// Check other settings
	if defaultSecurityPolicy.MaxCommandLen <= 0 {
		t.Error("defaultSecurityPolicy.MaxCommandLen is not positive")
	}

	if defaultSecurityPolicy.DefaultTimeout <= 0 {
		t.Error("defaultSecurityPolicy.DefaultTimeout is not positive")
	}

	if defaultSecurityPolicy.MaxTimeout <= 0 {
		t.Error("defaultSecurityPolicy.MaxTimeout is not positive")
	}

	if defaultSecurityPolicy.AllowShellAccess {
		t.Error("defaultSecurityPolicy.AllowShellAccess should be false by default")
	}
}

// BenchmarkValidateCommand benchmarks the validateCommand function
func BenchmarkValidateCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		validateCommand("uname -a", false)
	}
}

// BenchmarkSanitizeInput benchmarks the sanitizeInput function
func BenchmarkSanitizeInput(b *testing.B) {
	input := "ls -la /home/user"
	for i := 0; i < b.N; i++ {
		sanitizeInput(input)
	}
}

// BenchmarkExtractBaseCommand benchmarks the extractBaseCommand function
func BenchmarkExtractBaseCommand(b *testing.B) {
	command := "ls -la | grep test"
	for i := 0; i < b.N; i++ {
		extractBaseCommand(command)
	}
}

// BenchmarkContainsShellFeatures benchmarks the containsShellFeatures function
func BenchmarkContainsShellFeatures(b *testing.B) {
	command := "echo hello world"
	for i := 0; i < b.N; i++ {
		containsShellFeatures(command)
	}
}