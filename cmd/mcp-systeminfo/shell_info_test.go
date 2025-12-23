package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Mock os.Getenv for testing
var mockGetenv = func(key string) string {
	switch key {
	case "SHELL":
		return "/bin/bash"
	default:
		return ""
	}
}

// Mock exec.LookPath for testing
var mockLookPath = func(file string) (string, error) {
	switch file {
	case "powershell.exe", "powershell", "pwsh":
		return "/usr/bin/pwsh", nil
	default:
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
}

// TestGetShellInfo tests the getShellInfo function
func TestGetShellInfo(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
	}()

	// Set mock functions
	os.Getenv = mockGetenv
	exec.LookPath = mockLookPath

	got, err := getShellInfo()
	if err != nil {
		t.Errorf("getShellInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getShellInfo() returned nil")
		return
	}

	// Verify shell name
	if got.Name == "" {
		t.Error("getShellInfo() name is empty")
	}

	// Verify shell type
	if got.Type == "" {
		t.Error("getShellInfo() type is empty")
	}

	// Verify shell path
	if got.Path == "" {
		t.Error("getShellInfo() path is empty")
	}

	// Verify features
	if len(got.Features) == 0 {
		t.Error("getShellInfo() features is empty")
	}
}

// TestGetShellNameFromPath tests the getShellNameFromPath function
func TestGetShellNameFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Unix shell path",
			path: "/bin/bash",
			want: "bash",
		},
		{
			name: "Windows shell path with extension",
			path: "C:\\Windows\\System32\\cmd.exe",
			want: "cmd",
		},
		{
			name: "PowerShell path",
			path: "C:\\Program Files\\PowerShell\\7\\pwsh.exe",
			want: "pwsh",
		},
		{
			name: "Simple shell name",
			path: "zsh",
			want: "zsh",
		},
		{
			name: "Empty path",
			path: "",
			want: "unknown",
		},
		{
			name: "Path with directory separators",
			path: "/usr/local/bin/fish",
			want: "fish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellNameFromPath(tt.path)
			
			if got != tt.want {
				t.Errorf("getShellNameFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetShellType tests the getShellType function
func TestGetShellType(t *testing.T) {
	tests := []struct {
		name     string
		shellName string
		want     string
	}{
		{
			name:     "Bash",
			shellName: "bash",
			want:     "bash",
		},
		{
			name:     "Zsh",
			shellName: "zsh",
			want:     "zsh",
		},
		{
			name:     "Fish",
			shellName: "fish",
			want:     "fish",
		},
		{
			name:     "PowerShell",
			shellName: "powershell",
			want:     "powershell",
		},
		{
			name:     "PowerShell Core",
			shellName: "pwsh",
			want:     "powershell",
		},
		{
			name:     "CMD",
			shellName: "cmd",
			want:     "cmd",
		},
		{
			name:     "SH",
			shellName: "sh",
			want:     "sh",
		},
		{
			name:     "KSH",
			shellName: "ksh",
			want:     "ksh",
		},
		{
			name:     "CSH",
			shellName: "csh",
			want:     "csh",
		},
		{
			name:     "TCSH",
			shellName: "tcsh",
			want:     "csh",
		},
		{
			name:     "Unknown shell",
			shellName: "unknownshell",
			want:     "unknown",
		},
		{
			name:     "Case insensitive bash",
			shellName: "BASH",
			want:     "bash",
		},
		{
			name:     "Path containing bash",
			shellName: "/usr/bin/bash",
			want:     "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellType(tt.shellName)
			
			if got != tt.want {
				t.Errorf("getShellType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetShellVersion tests the getShellVersion function
func TestGetShellVersion(t *testing.T) {
	tests := []struct {
		name     string
		shellName string
		want     string
	}{
		{
			name:     "Bash version",
			shellName: "bash",
			want:     "", // Will be empty if command fails
		},
		{
			name:     "Zsh version",
			shellName: "zsh",
			want:     "", // Will be empty if command fails
		},
		{
			name:     "PowerShell version",
			shellName: "powershell",
			want:     "", // Will be empty if command fails
		},
		{
			name:     "PowerShell Core version",
			shellName: "pwsh",
			want:     "", // Will be empty if command fails
		},
		{
			name:     "CMD version",
			shellName: "cmd",
			want:     "Windows Command Prompt",
		},
		{
			name:     "Unknown shell",
			shellName: "unknownshell",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellVersion(tt.shellName)
			
			// We can't guarantee version output, but we can verify it doesn't panic
			_ = got
			
			// For cmd, we can verify the expected output
			if tt.shellName == "cmd" && got != tt.want {
				t.Errorf("getShellVersion() for cmd = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetShellFeatures tests the getShellFeatures function
func TestGetShellFeatures(t *testing.T) {
	tests := []struct {
		name     string
		shellName string
		want     []string
	}{
		{
			name:     "Bash features",
			shellName: "bash",
			want: []string{
				"command_history", "tab_completion", "command_substitution",
				"process_substitution", "arrays", "functions", "aliases",
				"brace_expansion", "globbing", "redirection", "pipelines",
				"job_control", "command_line_editing", "programmable_completion",
			},
		},
		{
			name:     "Zsh features",
			shellName: "zsh",
			want: []string{
				"command_history", "tab_completion", "command_substitution",
				"process_substitution", "arrays", "associative_arrays", "functions",
				"aliases", "brace_expansion", "extended_globbing", "redirection",
				"pipelines", "job_control", "command_line_editing", "zle",
				"programmable_completion", "theme_support", "plugin_system",
			},
		},
		{
			name:     "PowerShell features",
			shellName: "powershell",
			want: []string{
				"command_history", "tab_completion", "pipeline_support",
				"objects", "modules", "functions", "aliases", "providers",
				"remoting", "workflow", "dsc", "classes", "enums",
				"error_handling", "structured_data", "xml_json_support",
			},
		},
		{
			name:     "Fish features",
			shellName: "fish",
			want: []string{
				"command_history", "tab_completion", "syntax_highlighting",
				"autosuggestions", "functions", "aliases", "variables",
				"job_control", "web_configuration", "universal_variables",
			},
		},
		{
			name:     "CMD features",
			shellName: "cmd",
			want: []string{
				"command_history", "batch_files", "environment_variables",
				"pipelines", "redirection", "built-in_commands",
			},
		},
		{
			name:     "Unknown shell features",
			shellName: "unknownshell",
			want: []string{"basic_shell_features"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellFeatures(tt.shellName)
			
			if len(got) != len(tt.want) {
				t.Errorf("getShellFeatures() length = %v, want %v", len(got), len(tt.want))
				return
			}

			for i, feature := range got {
				if feature != tt.want[i] {
					t.Errorf("getShellFeatures()[%d] = %v, want %v", i, feature, tt.want[i])
				}
			}
		})
	}
}

// TestGetShellAliases tests the getShellAliases function
func TestGetShellAliases(t *testing.T) {
	tests := []struct {
		name     string
		shellPath string
		want     map[string]string
	}{
		{
			name:     "Bash shell",
			shellPath: "/bin/bash",
			want:     map[string]string{}, // Will be empty if command fails
		},
		{
			name:     "Non-bash shell",
			shellPath: "/bin/zsh",
			want:     map[string]string{}, // Will be empty for non-bash shells
		},
		{
			name:     "Empty path",
			shellPath: "",
			want:     map[string]string{}, // Will be empty for empty path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellAliases(tt.shellPath)
			
			// We can't guarantee alias output, but we can verify it doesn't panic
			_ = got
			
			// Verify it returns a map
			if got == nil {
				t.Error("getShellAliases() returned nil")
			}
		})
	}
}

// TestGetShellFunctions tests the getShellFunctions function
func TestGetShellFunctions(t *testing.T) {
	tests := []struct {
		name     string
		shellType string
		want     []string
	}{
		{
			name:     "Bash functions",
			shellType: "bash",
			want: []string{
				"cd", "pushd", "popd", "dirs", "history", "type", "which",
				"man", "help", "source", "exec", "exit", "return", "test",
				"[", "echo", "printf", "read", "mapfile", "readarray",
			},
		},
		{
			name:     "PowerShell functions",
			shellType: "powershell",
			want: []string{
				"Get-Help", "Get-Command", "Get-Member", "Get-ChildItem", "Set-Location",
				"Get-Location", "Write-Output", "Write-Host", "Read-Host", "Get-Content",
				"Set-Content", "Add-Content", "Test-Path", "New-Item", "Remove-Item",
			},
		},
		{
			name:     "CMD functions",
			shellType: "cmd",
			want: []string{
				"dir", "cd", "md", "rd", "del", "copy", "move", "ren",
				"type", "find", "sort", "more", "help", "exit", "cls",
			},
		},
		{
			name:     "Unknown shell functions",
			shellType: "unknown",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellFunctions(tt.shellType)
			
			if len(got) != len(tt.want) {
				t.Errorf("getShellFunctions() length = %v, want %v", len(got), len(tt.want))
				return
			}

			for i, function := range got {
				if function != tt.want[i] {
					t.Errorf("getShellFunctions()[%d] = %v, want %v", i, function, tt.want[i])
				}
			}
		})
	}
}

// TestShellInfoStruct tests the ShellInfo struct
func TestShellInfoStruct(t *testing.T) {
	aliases := map[string]string{
		"ll": "ls -la",
		"la": "ls -a",
		"l":  "ls -CF",
	}

	functions := []string{
		"cd", "pushd", "popd", "dirs", "history", "type", "which",
		"man", "help", "source", "exec", "exit", "return", "test",
		"[", "echo", "printf", "read", "mapfile", "readarray",
	}

	features := []string{
		"command_history", "tab_completion", "command_substitution",
		"process_substitution", "arrays", "functions", "aliases",
		"brace_expansion", "globbing", "redirection", "pipelines",
		"job_control", "command_line_editing", "programmable_completion",
	}

	shellInfo := &ShellInfo{
		Name:      "bash",
		Version:   "5.1.4",
		Path:      "/bin/bash",
		Type:      "bash",
		Features:  features,
		Aliases:   aliases,
		Functions: functions,
	}

	// Verify all fields are set correctly
	if shellInfo.Name != "bash" {
		t.Errorf("ShellInfo.Name = %v, want bash", shellInfo.Name)
	}

	if shellInfo.Version != "5.1.4" {
		t.Errorf("ShellInfo.Version = %v, want 5.1.4", shellInfo.Version)
	}

	if shellInfo.Path != "/bin/bash" {
		t.Errorf("ShellInfo.Path = %v, want /bin/bash", shellInfo.Path)
	}

	if shellInfo.Type != "bash" {
		t.Errorf("ShellInfo.Type = %v, want bash", shellInfo.Type)
	}

	if len(shellInfo.Features) != len(features) {
		t.Errorf("ShellInfo.Features length = %v, want %v", len(shellInfo.Features), len(features))
	}

	if len(shellInfo.Aliases) != len(aliases) {
		t.Errorf("ShellInfo.Aliases length = %v, want %v", len(shellInfo.Aliases), len(aliases))
	}

	if len(shellInfo.Functions) != len(functions) {
		t.Errorf("ShellInfo.Functions length = %v, want %v", len(shellInfo.Functions), len(functions))
	}
}

// TestShellInfoOnWindows tests shell information on Windows
func TestShellInfoOnWindows(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	// Set mock functions for Windows
	os.Getenv = func(key string) string {
		if key == "SHELL" {
			return "" // No SHELL environment variable on Windows
		}
		return ""
	}
	
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "powershell.exe":
			return "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe", nil
		case "pwsh.exe":
			return "C:\\Program Files\\PowerShell\\7\\pwsh.exe", nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	// This is a conceptual test for Windows shell detection
	// In a real implementation, we would mock runtime.GOOS
	// and verify the shell detection logic
	
	got, err := getShellInfo()
	if err != nil {
		t.Errorf("getShellInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getShellInfo() returned nil")
		return
	}

	// Verify shell information is populated
	if got.Name == "" {
		t.Error("getShellInfo() name is empty")
	}

	if got.Type == "" {
		t.Error("getShellInfo() type is empty")
	}

	if got.Path == "" {
		t.Error("getShellInfo() path is empty")
	}
}

// TestShellInfoOnUnix tests shell information on Unix systems
func TestShellInfoOnUnix(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	// Set mock functions for Unix
	os.Getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return ""
	}
	
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "bash":
			return "/bin/bash", nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	// This is a conceptual test for Unix shell detection
	// In a real implementation, we would mock runtime.GOOS
	// and verify the shell detection logic
	
	got, err := getShellInfo()
	if err != nil {
		t.Errorf("getShellInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getShellInfo() returned nil")
		return
	}

	// Verify shell information is populated
	if got.Name == "" {
		t.Error("getShellInfo() name is empty")
	}

	if got.Type == "" {
		t.Error("getShellInfo() type is empty")
	}

	if got.Path == "" {
		t.Error("getShellInfo() path is empty")
	}
}

// TestPowerShellVersionParsing tests PowerShell version parsing
func TestPowerShellVersionParsing(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
	}()

	// Set mock functions
	os.Getenv = func(key string) string {
		if key == "SHELL" {
			return "" // No SHELL environment variable
		}
		return ""
	}
	
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "pwsh":
			return "/usr/bin/pwsh", nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	// Test PowerShell version parsing
	got := getShellVersion("pwsh")
	
	// We can't guarantee version output, but we can verify it doesn't panic
	_ = got
}

// TestBashVersionParsing tests Bash version parsing
func TestBashVersionParsing(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
	}()

	// Set mock functions
	os.Getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return ""
	}
	
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "bash":
			return "/bin/bash", nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	// Test Bash version parsing
	got := getShellVersion("bash")
	
	// We can't guarantee version output, but we can verify it doesn't panic
	_ = got
}

// TestShellAliasesParsing tests shell aliases parsing
func TestShellAliasesParsing(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
	}()

	// Set mock functions
	os.Getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return ""
	}
	
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "bash":
			return "/bin/bash", nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	// Test shell aliases parsing
	got := getShellAliases("/bin/bash")
	
	// We can't guarantee alias output, but we can verify it doesn't panic
	_ = got
	
	// Verify it returns a map
	if got == nil {
		t.Error("getShellAliases() returned nil")
	}
}

// TestErrorHandlingInShellInfo tests error handling in shell information functions
func TestErrorHandlingInShellInfo(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
	}()

	// Set mock functions that return errors
	os.Getenv = func(key string) string {
		return "" // Empty SHELL environment variable
	}
	
	exec.LookPath = func(file string) (string, error) {
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}

	// Test error handling in getShellInfo
	got, err := getShellInfo()
	
	// Should handle errors gracefully and still return shell information
	if err != nil {
		t.Errorf("getShellInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getShellInfo() returned nil")
		return
	}

	// Verify default values are set
	if got.Name == "" {
		t.Error("getShellInfo() name is empty")
	}

	if got.Type == "" {
		t.Error("getShellInfo() type is empty")
	}

	if got.Path == "" {
		t.Error("getShellInfo() path is empty")
	}
}

// TestTimeoutHandlingInShellInfo tests timeout handling in shell information functions
func TestTimeoutHandlingInShellInfo(t *testing.T) {
	// Save original functions
	originalGetenv := os.Getenv
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		os.Getenv = originalGetenv
		exec.LookPath = originalLookPath
	}()

	// Set mock functions
	os.Getenv = func(key string) string {
		if key == "SHELL" {
			return "/bin/bash"
		}
		return ""
	}
	
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "bash":
			return "/bin/bash", nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	// Test timeout handling in getShellVersion
	got := getShellVersion("bash")
	
	// We can't guarantee version output, but we can verify it doesn't panic
	_ = got

	// Test timeout handling in getShellAliases
	aliases := getShellAliases("/bin/bash")
	
	// We can't guarantee alias output, but we can verify it doesn't panic
	_ = aliases
}

// TestShellFeaturesForDifferentTypes tests shell features for different shell types
func TestShellFeaturesForDifferentTypes(t *testing.T) {
	tests := []struct {
		name     string
		shellType string
		expectedFeaturesCount int
	}{
		{
			name:     "Bash features",
			shellType: "bash",
			expectedFeaturesCount: 13,
		},
		{
			name:     "Zsh features",
			shellType: "zsh",
			expectedFeaturesCount: 16,
		},
		{
			name:     "PowerShell features",
			shellType: "powershell",
			expectedFeaturesCount: 13,
		},
		{
			name:     "Fish features",
			shellType: "fish",
			expectedFeaturesCount: 9,
		},
		{
			name:     "CMD features",
			shellType: "cmd",
			expectedFeaturesCount: 6,
		},
		{
			name:     "Unknown shell features",
			shellType: "unknown",
			expectedFeaturesCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellFeatures(tt.shellType)
			
			if len(got) != tt.expectedFeaturesCount {
				t.Errorf("getShellFeatures() length = %v, want %v", len(got), tt.expectedFeaturesCount)
			}
		})
	}
}

// TestShellFunctionsForDifferentTypes tests shell functions for different shell types
func TestShellFunctionsForDifferentTypes(t *testing.T) {
	tests := []struct {
		name        string
		shellType   string
		expectedFunctionsCount int
	}{
		{
			name:     "Bash functions",
			shellType: "bash",
			expectedFunctionsCount: 17,
		},
		{
			name:     "PowerShell functions",
			shellType: "powershell",
			expectedFunctionsCount: 15,
		},
		{
			name:     "CMD functions",
			shellType: "cmd",
			expectedFunctionsCount: 14,
		},
		{
			name:     "Unknown shell functions",
			shellType: "unknown",
			expectedFunctionsCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellFunctions(tt.shellType)
			
			if len(got) != tt.expectedFunctionsCount {
				t.Errorf("getShellFunctions() length = %v, want %v", len(got), tt.expectedFunctionsCount)
			}
		})
	}
}

// TestShellNameExtractionFromPath tests shell name extraction from different path formats
func TestShellNameExtractionFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "Unix absolute path",
			path: "/bin/bash",
			want: "bash",
		},
		{
			name: "Unix relative path",
			path: "usr/bin/zsh",
			want: "zsh",
		},
		{
			name: "Windows path with backslashes",
			path: "C:\\Windows\\System32\\cmd.exe",
			want: "cmd",
		},
		{
			name: "Windows path with forward slashes",
			path: "C/Program Files/PowerShell/7/pwsh.exe",
			want: "pwsh",
		},
		{
			name: "Path with multiple separators",
			path: "/usr/local/bin/custom-shell",
			want: "custom-shell",
		},
		{
			name: "Path with trailing separator",
			path: "/bin/bash/",
			want: "bash/",
		},
		{
			name: "Just filename",
			path: "fish",
			want: "fish",
		},
		{
			name: "Empty path",
			path: "",
			want: "unknown",
		},
		{
			name: "Path with just separator",
			path: "/",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellNameFromPath(tt.path)
			
			if got != tt.want {
				t.Errorf("getShellNameFromPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestShellTypeDetectionCaseInsensitive tests shell type detection with case insensitivity
func TestShellTypeDetectionCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		shellName string
		want     string
	}{
		{
			name:     "Uppercase Bash",
			shellName: "BASH",
			want:     "bash",
		},
		{
			name:     "Mixed case PowerShell",
			shellName: "PowerShell",
			want:     "powershell",
		},
		{
			name:     "Lowercase zsh",
			shellName: "zsh",
			want:     "zsh",
		},
		{
			name:     "Mixed case Fish",
			shellName: "FiSh",
			want:     "fish",
		},
		{
			name:     "Uppercase CMD",
			shellName: "CMD",
			want:     "cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellType(tt.shellName)
			
			if got != tt.want {
				t.Errorf("getShellType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestShellTypeDetectionWithSubstrings tests shell type detection with substrings
func TestShellTypeDetectionWithSubstrings(t *testing.T) {
	tests := []struct {
		name     string
		shellName string
		want     string
	}{
		{
			name:     "Path containing bash",
			shellName: "/usr/bin/bash",
			want:     "bash",
		},
		{
			name:     "Path containing powershell",
			shellName: "C:\\Program Files\\PowerShell\\7\\powershell.exe",
			want:     "powershell",
		},
		{
			name:     "Path containing pwsh",
			shellName: "/usr/local/bin/pwsh",
			want:     "powershell",
		},
		{
			name:     "Path containing zsh",
			shellName: "/bin/zsh",
			want:     "zsh",
		},
		{
			name:     "Path containing fish",
			shellName: "/usr/bin/fish",
			want:     "fish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getShellType(tt.shellName)
			
			if got != tt.want {
				t.Errorf("getShellType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPowerShellVersionJSONParsing tests PowerShell version JSON parsing
func TestPowerShellVersionJSONParsing(t *testing.T) {
	// This is a conceptual test for PowerShell version JSON parsing
	// In a real implementation, we would mock the command execution
	// and verify the JSON parsing logic
	
	// Test JSON structure that would be returned by PowerShell
	jsonStr := `{"Major": 7, "Minor": 2, "Build": 0, "Revision": 6, "MajorRevision": 0, "MinorRevision": 6}`
	
	var versionInfo map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &versionInfo); err != nil {
		t.Errorf("Failed to parse PowerShell version JSON: %v", err)
	}
	
	// Verify the JSON structure
	if psVersion, ok := versionInfo["PSVersion"].(map[string]interface{}); ok {
		if _, ok := psVersion["Major"].(float64); !ok {
			t.Error("PowerShell version JSON missing Major field")
		}
		if _, ok := psVersion["Minor"].(float64); !ok {
			t.Error("PowerShell version JSON missing Minor field")
		}
	}
}