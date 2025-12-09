package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
		if matched, _ := regexp.MatchString(pattern, command); matched {
			return &SecurityResult{
				Valid:   false,
				Reason:  fmt.Sprintf("Blocked pattern detected: %s", pattern),
				Rule:    "blocked_pattern",
				Pattern: pattern,
			}
		}
	}

	// Extract base command and check if allowed
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return &SecurityResult{
			Valid:  false,
			Reason: "Empty command",
			Rule:   "empty_command",
		}
	}

	baseCmd := parts[0]
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

// validateScript validates a script against security policies
func validateScript(script string) *SecurityResult {
	// Check script length
	if len(script) > defaultSecurityPolicy.MaxScriptLen {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Script too long (max %d characters)", defaultSecurityPolicy.MaxScriptLen),
			Rule:   "max_length",
		}
	}

	// Check for valid UTF-8
	if !utf8.ValidString(script) {
		return &SecurityResult{
			Valid:  false,
			Reason: "Script contains invalid UTF-8 characters",
			Rule:   "utf8_validation",
		}
	}

	// Line-by-line validation
	lines := strings.Split(script, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		// Check each line against blocked patterns
		for _, pattern := range defaultSecurityPolicy.BlockedPatterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				return &SecurityResult{
					Valid:   false,
					Reason:  fmt.Sprintf("Blocked pattern detected at line %d: %s", i+1, pattern),
					Rule:    "blocked_pattern",
					Pattern: pattern,
				}
			}
		}

		// Extract base command from line and check if allowed
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		baseCmd := parts[0]
		// Skip variable assignments, function definitions, etc.
		if strings.Contains(baseCmd, "=") || strings.HasPrefix(baseCmd, "function") || 
		   strings.HasPrefix(baseCmd, "if") || strings.HasPrefix(baseCmd, "for") ||
		   strings.HasPrefix(baseCmd, "while") || strings.HasPrefix(baseCmd, "case") ||
		   strings.HasPrefix(baseCmd, "done") || strings.HasPrefix(baseCmd, "fi") ||
		   strings.HasPrefix(baseCmd, "then") || strings.HasPrefix(baseCmd, "else") ||
		   strings.HasPrefix(baseCmd, "elif") {
			continue
		}

		if !defaultSecurityPolicy.AllowedCommands[baseCmd] {
			return &SecurityResult{
				Valid:  false,
				Reason: fmt.Sprintf("Command not allowed at line %d: %s", i+1, baseCmd),
				Rule:   "allowed_commands",
			}
		}
	}

	// Check for dangerous script constructs
	if containsDangerousScriptConstructs(script) {
		return &SecurityResult{
			Valid:  false,
			Reason: "Script contains dangerous constructs",
			Rule:   "dangerous_constructs",
		}
	}

	return &SecurityResult{Valid: true}
}

// sanitizeInput sanitizes input by removing dangerous characters
func sanitizeInput(input string) string {
	// Remove null bytes and control characters
	result := strings.Builder{}
	for _, r := range input {
		if r != 0 && r != '\r' && (r == '\n' || r == '\t' || r >= 32 && r <= 126) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// containsShellFeatures checks if command contains shell features
func containsShellFeatures(command string) bool {
	// Check for shell operators and constructs
	shellPatterns := []string{
		`\|`,           // Pipe
		`>`,            // Redirect
		`<`,            // Redirect
		`>>`,           // Append redirect
		`&&`,           // AND operator
		`\|\|`,         // OR operator
		`;`,            // Command separator
		`&`,            // Background execution
		`\$\(`,         // Command substitution
		`\${`,          // Parameter expansion
		`\*`,           // Wildcard
		`\?`,           // Wildcard
		`\[.*\]`,       // Character class
		`\{.*\}`,       // Brace expansion
		"`",            // Backticks
	}

	for _, pattern := range shellPatterns {
		if matched, _ := regexp.MatchString(pattern, command); matched {
			return true
		}
	}

	return false
}

// containsDangerousScriptConstructs checks for dangerous script constructs
func containsDangerousScriptConstructs(script string) bool {
	dangerousPatterns := []string{
		`eval\s+`,              // eval command
		`exec\s+`,              // exec command
		`source\s+.*\|\s*`,    // source with pipe
		`\.\s+.*\|\s*`,        // source with pipe
		`shopt\s+-s`,          // shell options
		`set\s+-.*e`,          // set with options
		`trap\s+`,             // trap command
		`SIG.*`,               // Signal handling
		`/dev/`,              // Device files
		`/proc/`,             // Proc filesystem
		`/sys/`,              // Sys filesystem
		`mkfs`,               // Filesystem formatting
		`fdisk`,              // Disk partitioning
		`iptables`,            // Firewall rules
		`service\s+.*\s+stop`, // Stopping services
		`systemctl\s+.*\s+stop`, // Stopping services
		`shutdown`,            // System shutdown
		`reboot`,              // System reboot
		`halt`,                // System halt
		`poweroff`,            // Power off
		`passwd`,              // Password changes
		`su\s+`,              // User switching
		`sudo\s+`,            // Privilege escalation
		`chmod\s+.*[457][457][457]`, // Dangerous permissions
		`chown\s+.*root`,     // Ownership changes to root
		`crontab`,            // Cron jobs
		`at\s+`,              // At jobs
		`nohup`,              // No hangup
		`screen`,              // Screen sessions
		`tmux`,               // Tmux sessions
	}

	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, script); matched {
			return true
		}
	}

	return false
}

// validateWorkingDirectory validates working directory path
func validateWorkingDirectory(workingDir string) *SecurityResult {
	if workingDir == "" {
		return &SecurityResult{Valid: true} // Empty is OK, will use REPO_PATH
	}

	// Resolve path
	resolvedPath := workingDir
	if !filepath.IsAbs(workingDir) {
		repoPath := os.Getenv("REPO_PATH")
		if repoPath != "" {
			resolvedPath = filepath.Join(repoPath, workingDir)
		}
	}

	// Clean path
	resolvedPath = filepath.Clean(resolvedPath)

	// Check for path traversal
	if strings.Contains(resolvedPath, "..") {
		return &SecurityResult{
			Valid:  false,
			Reason: "Path traversal not allowed",
			Rule:   "path_traversal",
		}
	}

	// Check if path exists and is directory
	if info, err := os.Stat(resolvedPath); err != nil {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Working directory does not exist: %s", workingDir),
			Rule:   "directory_exists",
		}
	} else if !info.IsDir() {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Path is not a directory: %s", workingDir),
			Rule:   "not_directory",
		}
	}

	// Check if within allowed paths
	repoPath := os.Getenv("REPO_PATH")
	if repoPath != "" {
		if !strings.HasPrefix(resolvedPath, repoPath) {
			return &SecurityResult{
				Valid:  false,
				Reason: fmt.Sprintf("Working directory outside REPO_PATH: %s", workingDir),
				Rule:   "path_restriction",
			}
		}
	}

	return &SecurityResult{Valid: true}
}

// validateEnvironmentVariables validates environment variables
func validateEnvironmentVariables(envVars map[string]string) *SecurityResult {
	// Check environment variable names
	for name := range envVars {
		// Check name format
		if !regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`).MatchString(name) {
			return &SecurityResult{
				Valid:  false,
				Reason: fmt.Sprintf("Invalid environment variable name: %s", name),
				Rule:   "env_var_format",
			}
		}

		// Check for dangerous environment variables
		dangerousVars := []string{
			"PATH", "LD_PRELOAD", "LD_LIBRARY_PATH", "DYLD_INSERT_LIBRARIES",
			"IFS", "PS1", "PS2", "PS4", "PROMPT_COMMAND",
			"BASH_ENV", "ENV", "SHELL", "HOME", "USER",
		}

		for _, dangerous := range dangerousVars {
			if name == dangerous {
				return &SecurityResult{
					Valid:  false,
					Reason: fmt.Sprintf("Dangerous environment variable not allowed: %s", name),
					Rule:   "dangerous_env_var",
				}
			}
		}
	}

	return &SecurityResult{Valid: true}
}

// validateTimeout validates timeout value
func validateTimeout(timeout int, isScript bool) *SecurityResult {
	maxTimeout := defaultSecurityPolicy.MaxTimeout
	if isScript {
		maxTimeout = 600 // 10 minutes for scripts
	}

	if timeout <= 0 {
		return &SecurityResult{
			Valid:  false,
			Reason: "Timeout must be greater than 0",
			Rule:   "timeout_positive",
		}
	}

	if timeout > maxTimeout {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Timeout exceeds maximum allowed (%d seconds)", maxTimeout),
			Rule:   "timeout_maximum",
		}
	}

	return &SecurityResult{Valid: true}
}