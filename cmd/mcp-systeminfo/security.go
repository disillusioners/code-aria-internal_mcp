package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// validateCommand validates a command against security policies
func validateCommand(command string, allowShellAccess bool) *SecurityResult {
	// Check command length
	if len(command) > defaultSecurityPolicy.MaxCommandLen {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Command too long (max %d characters)", defaultSecurityPolicy.MaxCommandLen),
			Rule:   "max_length",
		}
	}

	// Check for valid UTF-8
	if !utf8.ValidString(command) {
		return &SecurityResult{
			Valid:  false,
			Reason: "Command contains invalid UTF-8 characters",
			Rule:   "utf8_validation",
		}
	}

	// Sanitize input
	sanitized := sanitizeInput(command)
	if sanitized != command {
		return &SecurityResult{
			Valid:  false,
			Reason: "Command contains invalid characters",
			Rule:   "character_validation",
		}
	}

	// Check against blocked patterns
	for _, pattern := range defaultSecurityPolicy.BlockedPatterns {
		if matched, _ := regexp.MatchString(strings.ToLower(pattern), strings.ToLower(command)); matched {
			return &SecurityResult{
				Valid:   false,
				Reason:  fmt.Sprintf("Blocked pattern detected: %s", pattern),
				Rule:    "blocked_pattern",
				Pattern: pattern,
			}
		}
	}

	// Extract base command and check if allowed
	baseCmd := extractBaseCommand(command)
	if baseCmd == "" {
		return &SecurityResult{
			Valid:  false,
			Reason: "Unable to determine base command",
			Rule:   "command_extraction",
		}
	}

	if !defaultSecurityPolicy.AllowedCommands[baseCmd] {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Command not allowed: %s", baseCmd),
			Rule:   "allowed_commands",
		}
	}

	// Check shell access restrictions
	if !allowShellAccess && !defaultSecurityPolicy.AllowShellAccess {
		if containsShellFeatures(command) {
			return &SecurityResult{
				Valid:  false,
				Reason: "Shell features not allowed",
				Rule:   "shell_access",
			}
		}
	}

	return &SecurityResult{Valid: true}
}

// sanitizeInput sanitizes input by removing dangerous characters
func sanitizeInput(input string) string {
	// Remove null bytes and control characters (except tab and newline)
	result := strings.Builder{}
	for _, r := range input {
		if r != 0 && r != '\r' && (r == '\n' || r == '\t' || r >= 32 && r <= 126) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// extractBaseCommand extracts the base command from a command line
func extractBaseCommand(command string) string {
	// Remove leading pipes and semicolons
	command = strings.TrimSpace(command)
	command = regexp.MustCompile(`^[|;]+`).ReplaceAllString(command, "")

	// Handle pipelines
	if strings.Contains(command, "|") {
		parts := strings.Split(command, "|")
		if len(parts) > 0 {
			command = strings.TrimSpace(parts[0])
		}
	}

	// Split by whitespace and take first part
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	// Handle option flags (starting with -)
	if strings.HasPrefix(fields[0], "-") {
		return ""
	}

	return fields[0]
}

// containsShellFeatures checks if command contains shell features
func containsShellFeatures(command string) bool {
	// Check for shell operators and constructs
	shellPatterns := []string{
		`\|`,                    // Pipeline
		`>`,                     // Redirect
		`<`,                     // Redirect
		`>>`,                    // Append redirect
		`;`,                     // Command separator
		`&&`,                    // AND operator
		`\|\|`,                  // OR operator
		`&`,                     // Background execution
		`\$\(`,                  // Command substitution
		`\$\{`,                  // Parameter expansion
		`\(`,                    // Parentheses (can be grouping)
		`\{`,                    // Brace expansion
		`\*`,                    // Wildcard
		`\?`,                    // Wildcard
		`\[.*\]`,                // Character class
		`".*\|.*"`,              // String with pipeline
		`'.*\|.*'`,              // String with pipeline
	}

	for _, pattern := range shellPatterns {
		if matched, _ := regexp.MatchString(pattern, command); matched {
			return true
		}
	}

	return false
}

// executeCommandWithTimeout executes a command with timeout (for systeminfo server)
func executeCommandWithTimeout(command string, timeout int) (*CommandResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// Execute command
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	result := &CommandResult{
		ExitCode:   0,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: duration.Milliseconds(),
		Command:    command,
		Timeout:    false,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Timeout = true
		return result, fmt.Errorf("command timed out after %v", time.Duration(timeout)*time.Second)
	}

	return result, err
}