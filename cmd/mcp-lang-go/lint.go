package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// toolLintEmbedded executes golangci-lint with automatic binary management
func toolLintEmbedded(args map[string]interface{}) (interface{}, error) {
	// Get target (file, directory, or "." for entire repo)
	target := "."
	if t, ok := args["target"].(string); ok && t != "" {
		target = t
	}

	// Get format (json or text)
	format := "json"
	if f, ok := args["format"].(string); ok && f != "" {
		format = f
	}

	// Get config file path
	configPath := ""
	if c, ok := args["config"].(string); ok && c != "" {
		configPath = c
	}

	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return nil, fmt.Errorf("REPO_PATH environment variable not set")
	}

	// Resolve target path
	var targetPath string
	if target == "." {
		targetPath = repoPath
	} else if filepath.IsAbs(target) {
		targetPath = target
	} else {
		targetPath = filepath.Join(repoPath, target)
	}

	// Ensure we have golangci-lint binary
	golangciLintPath, err := ensureGolangciLint()
	if err != nil {
		return nil, fmt.Errorf("failed to setup golangci-lint: %v", err)
	}

	// Build command
	// Use line-number format for easier parsing (more reliable than default)
	cmd := exec.Command(golangciLintPath, "run", "--out-format", "line-number")

	// Add config file if specified or if default exists
	if configPath != "" {
		if !filepath.IsAbs(configPath) {
			configPath = filepath.Join(repoPath, configPath)
		}
		if _, err := os.Stat(configPath); err == nil {
			cmd.Args = append(cmd.Args, "--config", configPath)
		}
	} else {
		// Check for default config file
		defaultConfig := filepath.Join(repoPath, ".golangci.yml")
		if _, err := os.Stat(defaultConfig); err == nil {
			cmd.Args = append(cmd.Args, "--config", defaultConfig)
		}
		// Also check for .golangci.yaml (alternative extension)
		if _, err := os.Stat(filepath.Join(repoPath, ".golangci.yaml")); err == nil {
			cmd.Args = append(cmd.Args, "--config", filepath.Join(repoPath, ".golangci.yaml"))
		}
	}

	// Add target path
	cmd.Args = append(cmd.Args, targetPath)

	// Set working directory
	cmd.Dir = repoPath

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err = cmd.Run()
	output := stdout.String()
	errOutput := stderr.String()

	// Parse output
	result := LintResult{
		Target:      target,
		TotalIssues: 0,
		Issues:      []LintIssue{},
		Success:     err == nil && len(output) == 0,
	}

	if err != nil {
		// Check if it's just lint errors (exit code 1) or a real error
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				// Exit code 1 means linting found issues, which is expected
				result.Success = false
			} else {
				// Other exit codes indicate real errors
				result.Error = fmt.Sprintf("golangci-lint execution failed: %s", errOutput)
				return result, nil
			}
		} else {
			result.Error = fmt.Sprintf("failed to execute golangci-lint: %v", err)
			return result, nil
		}
	}

	// Parse lint output
	issues := parseLintOutput(output)
	result.Issues = issues
	result.TotalIssues = len(issues)

	// If format is text, return as string
	if format == "text" {
		if result.Error != "" {
			return result.Error, nil
		}
		if len(output) > 0 {
			return output, nil
		}
		return "No linting issues found", nil
	}

	// Return structured JSON result
	return result, nil
}

// ensureGolangciLint ensures golangci-lint is available by either using system binary or downloading it
func ensureGolangciLint() (string, error) {
	// First, try to find golangci-lint in PATH
	if path, err := exec.LookPath("golangci-lint"); err == nil {
		return path, nil
	}

	// If not found in PATH, try to create a local installation
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create ~/.local/bin directory if it doesn't exist
	localBinDir := filepath.Join(homeDir, ".local", "bin")
	if err := os.MkdirAll(localBinDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local bin directory: %v", err)
	}

	// Check if golangci-lint is already installed locally
	localBinary := filepath.Join(localBinDir, "golangci-lint")
	if _, err := os.Stat(localBinary); err == nil {
		return localBinary, nil
	}

	// Try to install golangci-lint using the official install script
	installScript := `
#!/bin/bash
set -e

# Detect architecture
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l) ARCH="armv7" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Download golangci-lint
VERSION="1.62.2"
FILENAME="golangci-lint-${VERSION}-${OS}-${ARCH}.tar.gz"
URL="https://github.com/golangci/golangci-lint/releases/download/v${VERSION}/${FILENAME}"

echo "Downloading golangci-lint from $URL..."
cd /tmp
if command -v curl >/dev/null 2>&1; then
    curl -sSfL "$URL" -o golangci-lint.tar.gz
elif command -v wget >/dev/null 2>&1; then
    wget -q "$URL" -O golangci-lint.tar.gz
else
    echo "Neither curl nor wget is available"
    exit 1
fi

# Extract
tar -xzf golangci-lint.tar.gz
cp "golangci-lint-${VERSION}-${OS}-${ARCH}/golangci-lint" "$1"
chmod +x "$1"

echo "golangci-lint installed successfully to $1"
`

	// Create temporary script file
	scriptFile := filepath.Join(os.TempDir(), "install-golangci-lint.sh")
	if err := os.WriteFile(scriptFile, []byte(installScript), 0755); err != nil {
		return "", fmt.Errorf("failed to create install script: %v", err)
	}
	defer os.Remove(scriptFile)

	// Execute install script
	cmd := exec.Command("bash", scriptFile, localBinary)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to install golangci-lint: %v", err)
	}

	// Verify installation
	if _, err := os.Stat(localBinary); err != nil {
		return "", fmt.Errorf("golangci-lint installation failed")
	}

	return localBinary, nil
}

// parseLintOutput parses golangci-lint output into structured issues
func parseLintOutput(output string) []LintIssue {
	if len(strings.TrimSpace(output)) == 0 {
		return []LintIssue{}
	}

	var issues []LintIssue
	lines := strings.Split(output, "\n")

	// golangci-lint line-number format:
	// file.go:line:column: message (linter)
	// file.go:line: message (linter)  (when column is not available)

	// Regex to match lint output in line-number format
	// Format: file:line:column: message (linter)
	// or: file:line: message (linter)
	re := regexp.MustCompile(`^(\s*)([^:]+):(\d+)(?::(\d+))?:\s*(.+?)\s*\(([^)]+)\)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 6 {
			// Try alternative format without column
			altRe := regexp.MustCompile(`^(\s*)([^:]+):(\d+):\s*(.+?)\s*\(([^)]+)\)$`)
			altMatches := altRe.FindStringSubmatch(line)
			if len(altMatches) >= 6 {
				file := altMatches[2]
				lineNum, _ := strconv.Atoi(altMatches[3])
				message := altMatches[4]
				linter := altMatches[5]

				// Determine severity from message or linter
				severity := determineSeverity(message, linter)

				issues = append(issues, LintIssue{
					File:     file,
					Line:     lineNum,
					Severity: severity,
					Linter:   linter,
					Message:  message,
				})
			}
			continue
		}

		file := matches[2]
		lineNum, _ := strconv.Atoi(matches[3])
		column := 0
		if matches[4] != "" {
			column, _ = strconv.Atoi(matches[4])
		}
		message := matches[5]
		linter := matches[6]

		// Determine severity from message or linter
		severity := determineSeverity(message, linter)

		issue := LintIssue{
			File:     file,
			Line:     lineNum,
			Severity: severity,
			Linter:   linter,
			Message:  message,
		}

		if column > 0 {
			issue.Column = column
		}

		issues = append(issues, issue)
	}

	return issues
}

// determineSeverity determines the severity of a lint issue
func determineSeverity(message, linter string) string {
	messageLower := strings.ToLower(message)

	// Check for explicit severity indicators
	if strings.Contains(messageLower, "error") || strings.Contains(messageLower, "fatal") {
		return "error"
	}
	if strings.Contains(messageLower, "warning") || strings.Contains(messageLower, "warn") {
		return "warning"
	}
	if strings.Contains(messageLower, "info") || strings.Contains(messageLower, "hint") {
		return "info"
	}

	// Check linter name for severity hints
	linterLower := strings.ToLower(linter)
	if strings.Contains(linterLower, "err") || strings.Contains(linterLower, "fatal") {
		return "error"
	}
	if strings.Contains(linterLower, "warn") {
		return "warning"
	}

	// Default to warning for unknown cases
	return "warning"
}