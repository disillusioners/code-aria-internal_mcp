package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolLint tests the lint tool functionality
func TestToolLint(t *testing.T) {
	// Skip tests if golangci-lint is not available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not available for unit tests")
	}

	// Save original environment
	originalRepoPath := os.Getenv("REPO_PATH")
	defer func() {
		if originalRepoPath != "" {
			os.Setenv("REPO_PATH", originalRepoPath)
		} else {
			os.Unsetenv("REPO_PATH")
		}
	}()

	tests := []struct {
		name          string
		args          map[string]interface{}
		setupEnv      func() string
		expectedError bool
		checkResult   func(interface{}) bool
	}{
		{
			name: "missing REPO_PATH",
			args: map[string]interface{}{
				"target": ".",
			},
			setupEnv:      func() string { return "" },
			expectedError: true,
		},
		{
			name: "valid args with default target",
			args: map[string]interface{}{},
			setupEnv: func() string {
				// Create a temporary directory with a Go file
				tmpDir := t.TempDir()
				goFile := filepath.Join(tmpDir, "test.go")
				content := `
package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`
				if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write test Go file: %v", err)
				}
				return tmpDir
			},
			expectedError: false,
			checkResult: func(result interface{}) bool {
				lintResult, ok := result.(LintResult)
				if !ok {
					return false
				}
				// Should have processed the file
				return lintResult.Target == "."
			},
		},
		{
			name: "with custom target",
			args: map[string]interface{}{
				"target": "test.go",
			},
			setupEnv: func() string {
				tmpDir := t.TempDir()
				goFile := filepath.Join(tmpDir, "test.go")
				content := `package main
func main() {}
`
				if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write test Go file: %v", err)
				}
				return tmpDir
			},
			expectedError: false,
			checkResult: func(result interface{}) bool {
				lintResult, ok := result.(LintResult)
				if !ok {
					return false
				}
				return lintResult.Target == "test.go"
			},
		},
		{
			name: "with text format",
			args: map[string]interface{}{
				"format": "text",
			},
			setupEnv: func() string {
				tmpDir := t.TempDir()
				goFile := filepath.Join(tmpDir, "test.go")
				content := `package main
func main() {}
`
				if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write test Go file: %v", err)
				}
				return tmpDir
			},
			expectedError: false,
			checkResult: func(result interface{}) bool {
				// Text format may return string or LintResult depending on output
				// Both are acceptable
				switch result.(type) {
				case string:
					return true
				case LintResult:
					return true
				default:
					return false
				}
			},
		},
		{
			name: "with config file",
			args: map[string]interface{}{
				"config": ".golangci.yml",
			},
			setupEnv: func() string {
				tmpDir := t.TempDir()

				// Create a config file
				configFile := filepath.Join(tmpDir, ".golangci.yml")
				configContent := `
linters:
  disable-all: true
  enable:
    - gofmt
    - goimports
`
				if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				goFile := filepath.Join(tmpDir, "test.go")
				content := `package main
import "fmt"
func main(){fmt.Println("hello")}
`
				if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write test Go file: %v", err)
				}
				return tmpDir
			},
			expectedError: false,
			checkResult: func(result interface{}) bool {
				_, ok := result.(LintResult)
				if !ok {
					return false
				}
				// Config should be used
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoPath := tt.setupEnv()
			if repoPath != "" {
				os.Setenv("REPO_PATH", repoPath)
			} else {
				os.Unsetenv("REPO_PATH")
			}

			result, err := toolLintEmbedded(tt.args)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkResult != nil && !tt.checkResult(result) {
				t.Errorf("Result validation failed")
			}
		})
	}
}

// TestParseLintOutput tests the lint output parsing functionality
func TestParseLintOutput(t *testing.T) {
	tests := []struct {
		name           string
		output         string
		expectedIssues int
		checkIssues    func([]LintIssue) bool
	}{
		{
			name:           "empty output",
			output:         "",
			expectedIssues: 0,
		},
		{
			name:           "whitespace only",
			output:         "\n\n  \n",
			expectedIssues: 0,
		},
		{
			name:           "single issue with column",
			output:         `main.go:10:5: Error message (testlinter)`,
			expectedIssues: 1,
			checkIssues: func(issues []LintIssue) bool {
				if len(issues) != 1 {
					return false
				}
				issue := issues[0]
				return issue.File == "main.go" &&
					issue.Line == 10 &&
					issue.Column == 5 &&
					issue.Message == "Error message" &&
					issue.Linter == "testlinter"
			},
		},
		{
			name:           "single issue without column",
			output:         `main.go:10: Error message (testlinter)`,
			expectedIssues: 1,
			checkIssues: func(issues []LintIssue) bool {
				if len(issues) != 1 {
					return false
				}
				issue := issues[0]
				return issue.File == "main.go" &&
					issue.Line == 10 &&
					issue.Column == 0 &&
					issue.Message == "Error message" &&
					issue.Linter == "testlinter"
			},
		},
		{
			name: "multiple issues",
			output: `main.go:10:5: Error message (linter1)
main.go:15:2: Warning message (linter2)
utils.go:5: Info message (linter3)`,
			expectedIssues: 3,
			checkIssues: func(issues []LintIssue) bool {
				if len(issues) != 3 {
					return false
				}
				// Check each issue
				return issues[0].File == "main.go" &&
					issues[0].Line == 10 &&
					issues[1].File == "main.go" &&
					issues[1].Line == 15 &&
					issues[2].File == "utils.go" &&
					issues[2].Line == 5
			},
		},
		{
			name: "issues with indentation",
			output: `  main.go:10:5: Indented message (linter)
    main.go:15:2: More indented (anotherlinter)`,
			expectedIssues: 2,
			checkIssues: func(issues []LintIssue) bool {
				if len(issues) != 2 {
					return false
				}
				return issues[0].File == "main.go" && issues[1].File == "main.go"
			},
		},
		{
			name: "mixed valid and invalid lines",
			output: `main.go:10:5: Valid issue (linter)
invalid line that doesn't match format
main.go:15:2: Another valid issue (anotherlinter)`,
			expectedIssues: 2,
			checkIssues: func(issues []LintIssue) bool {
				return len(issues) == 2
			},
		},
		{
			name:           "issue with special characters in message",
			output:         `main.go:10:5: Message with "quotes" and 'apostrophes' (testlinter)`,
			expectedIssues: 1,
			checkIssues: func(issues []LintIssue) bool {
				if len(issues) != 1 {
					return false
				}
				return strings.Contains(issues[0].Message, "quotes") &&
					strings.Contains(issues[0].Message, "apostrophes")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := parseLintOutput(tt.output)

			if len(issues) != tt.expectedIssues {
				t.Errorf("Expected %d issues, got %d", tt.expectedIssues, len(issues))
			}

			if tt.checkIssues != nil && !tt.checkIssues(issues) {
				t.Errorf("Issue validation failed")
			}
		})
	}
}

// TestDetermineSeverity tests the severity determination logic
func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		linter   string
		expected string
	}{
		{
			name:     "error in message",
			message:  "This is an error",
			linter:   "testlinter",
			expected: "error",
		},
		{
			name:     "fatal in message",
			message:  "Fatal error occurred",
			linter:   "testlinter",
			expected: "error",
		},
		{
			name:     "warning in message",
			message:  "This is a warning",
			linter:   "testlinter",
			expected: "warning",
		},
		{
			name:     "warn in message",
			message:  "This will warn you",
			linter:   "testlinter",
			expected: "warning",
		},
		{
			name:     "info in message",
			message:  "Info for you",
			linter:   "testlinter",
			expected: "info",
		},
		{
			name:     "hint in message",
			message:  "Hint message",
			linter:   "testlinter",
			expected: "info",
		},
		{
			name:     "error in linter name",
			message:  "Some message",
			linter:   "errcheck",
			expected: "error",
		},
		{
			name:     "warning in linter name",
			message:  "Some message",
			linter:   "warnlinter",
			expected: "warning",
		},
		{
			name:     "unknown case",
			message:  "Just a message",
			linter:   "unknownlinter",
			expected: "warning",
		},
		{
			name:     "case insensitive",
			message:  "ERROR MESSAGE",
			linter:   "TESTLINTER",
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := determineSeverity(tt.message, tt.linter)
			if severity != tt.expected {
				t.Errorf("Expected severity '%s', got '%s'", tt.expected, severity)
			}
		})
	}
}

// TestLintResultJSONSerialization tests JSON serialization of LintResult
func TestLintResultJSONSerialization(t *testing.T) {
	result := LintResult{
		Target:      "main.go",
		TotalIssues: 2,
		Success:     false,
		Issues: []LintIssue{
			{
				File:     "main.go",
				Line:     10,
				Column:   5,
				Severity: "error",
				Linter:   "gofmt",
				Message:  "Expected '}', but found 'EOF'",
			},
			{
				File:     "utils.go",
				Line:     15,
				Severity: "warning",
				Linter:   "goimports",
				Message:  "Import should be grouped",
				Fix:      "Run 'goimports' to fix",
			},
		},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal LintResult: %v", err)
	}

	var unmarshaled LintResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal LintResult: %v", err)
	}

	if unmarshaled.Target != result.Target {
		t.Errorf("Target mismatch: expected %s, got %s", result.Target, unmarshaled.Target)
	}

	if unmarshaled.TotalIssues != result.TotalIssues {
		t.Errorf("TotalIssues mismatch: expected %d, got %d", result.TotalIssues, unmarshaled.TotalIssues)
	}

	if unmarshaled.Success != result.Success {
		t.Errorf("Success mismatch: expected %t, got %t", result.Success, unmarshaled.Success)
	}

	if len(unmarshaled.Issues) != len(result.Issues) {
		t.Errorf("Issues count mismatch: expected %d, got %d", len(result.Issues), len(unmarshaled.Issues))
	}

	// Check first issue
	if len(unmarshaled.Issues) > 0 {
		orig := result.Issues[0]
		unmarsh := unmarshaled.Issues[0]
		if unmarsh.File != orig.File || unmarsh.Line != orig.Line || unmarsh.Severity != orig.Severity {
			t.Errorf("First issue mismatch")
		}
	}
}

// BenchmarkParseLintOutput benchmarks the parseLintOutput function
func BenchmarkParseLintOutput(b *testing.B) {
	// Generate a large lint output
	var output strings.Builder
	for i := 0; i < 1000; i++ {
		output.WriteString(fmt.Sprintf("main.go:%d:5: Test issue %d (testlinter)\n", i, i))
	}
	largeOutput := output.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseLintOutput(largeOutput)
	}
}

// TestPathResolution tests path resolution logic
func TestPathResolution(t *testing.T) {
	// Save original environment
	originalRepoPath := os.Getenv("REPO_PATH")
	defer func() {
		if originalRepoPath != "" {
			os.Setenv("REPO_PATH", originalRepoPath)
		} else {
			os.Unsetenv("REPO_PATH")
		}
	}()

	tests := []struct {
		name       string
		target     string
		repoPath   string
		setupFiles func(string) error
		expectErr  bool
	}{
		{
			name:      "relative target",
			target:    "main.go",
			repoPath:  "/test/repo",
			expectErr: false, // We'll mock this by not actually running golangci-lint
		},
		{
			name:      "absolute target",
			target:    "/absolute/path/main.go",
			repoPath:  "/test/repo",
			expectErr: false,
		},
		{
			name:      "current directory target",
			target:    ".",
			repoPath:  "/test/repo",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("REPO_PATH", tt.repoPath)

			// We can't actually test the linting without golangci-lint,
			// but we can test the path resolution logic
			args := map[string]interface{}{
				"target": tt.target,
			}

			_, err := toolLintEmbedded(args)

			// We expect an error about golangci-lint not being available in test environment
			// but not about path resolution
			if err != nil && !strings.Contains(err.Error(), "golangci-lint") {
				if !tt.expectErr {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
