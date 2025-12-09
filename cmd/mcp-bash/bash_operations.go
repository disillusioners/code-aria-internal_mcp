package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Global security policy - in production, this should be configurable
var defaultSecurityPolicy = SecurityPolicy{
	AllowedCommands: map[string]bool{
		// Basic file operations
		"ls": true, "cat": true, "head": true, "tail": true,
		"grep": true, "find": true, "locate": true, "which": true,
		"file": true, "stat": true, "du": true, "df": true,
		"dir": true, "echo": true, "pwd": true, "date": true, "hostname": true,
		
		// Development tools
		"git": true, "npm": true, "yarn": true, "make": true, "cmake": true,
		"go": true, "python": true, "python3": true, "node": true,
		"java": true, "javac": true, "mvn": true, "gradle": true,
		"cargo": true, "rustc": true, "gcc": true, "g++": true,
		"clang": true, "clang++": true,
		
		// System information
		"ps": true, "top": true, "free": true, "uname": true, "whoami": true,
		"id": true, "uptime": true, "lscpu": true,
		
		// Text processing
		"sed": true, "awk": true, "sort": true, "uniq": true, "cut": true,
		"tr": true, "wc": true, "diff": true, "patch": true,
		
		// Archive tools
		"tar": true, "gzip": true, "gunzip": true, "zip": true, "unzip": true,
		
		// Network tools (read-only)
		"curl": true, "wget": true, "ping": true, "nslookup": true, "dig": true,
		
		// Process management
		"kill": true, "killall": true, "pkill": true,
	},
	BlockedPatterns: []string{
		`rm\s+-rf\s+/`,           // Dangerous deletion
		`dd\s+if=/dev/zero`,      // Disk wiping
		`:\(\)\{.*\}\;`,          // Fork bombs
		`chmod\s+777\s+/`,        // Dangerous permissions
		`sudo\s+`,                // Privilege escalation
		`su\s+`,                  // User switching
		`passwd\s+`,              // Password changes
		`curl.*\|\s*sh`,          // Pipe to shell
		`wget.*\|\s*sh`,          // Pipe to shell
		`>\s+/dev/`,             // Writing to devices
		`rm\s+.*\s+/,`,         // Deleting in root
		`mkfs`,                  // Filesystem formatting
		`fdisk`,                  // Disk partitioning
		`iptables`,                // Firewall rules
		`service\s+.*\s+stop`,   // Stopping services
		`systemctl\s+.*\s+stop`, // Stopping services
		`shutdown`,               // System shutdown
		`reboot`,                 // System reboot
		`halt`,                   // System halt
		`poweroff`,               // Power off
	},
	MaxCommandLen:   1000,
	MaxScriptLen:    10000,
	DefaultTimeout:  30,
	MaxTimeout:      300,
	AllowShellAccess: false,
}

// toolExecuteCommand executes a single bash command
func toolExecuteCommand(args map[string]interface{}) (string, error) {
	// Extract and validate parameters
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command is required")
	}

	timeout := defaultSecurityPolicy.DefaultTimeout
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}
	if timeout > defaultSecurityPolicy.MaxTimeout {
		timeout = defaultSecurityPolicy.MaxTimeout
	}

	workingDir := os.Getenv("REPO_PATH")
	if wd, ok := args["working_directory"].(string); ok && wd != "" {
		workingDir = wd
	}

	allowShellAccess := false
	if asa, ok := args["allow_shell_access"].(bool); ok {
		allowShellAccess = asa
	}

	var envVars map[string]string
	if ev, ok := args["environment_vars"].(map[string]interface{}); ok {
		envVars = make(map[string]string)
		for k, v := range ev {
			if val, ok := v.(string); ok {
				envVars[k] = val
			}
		}
	}

	// Validate working directory
	if dirResult := validateWorkingDirectory(workingDir); !dirResult.Valid {
		return "", fmt.Errorf("working directory validation failed: %s", dirResult.Reason)
	}

	// Validate environment variables
	if envResult := validateEnvironmentVariables(envVars); !envResult.Valid {
		return "", fmt.Errorf("environment variable validation failed: %s", envResult.Reason)
	}

	// Validate timeout
	if timeoutResult := validateTimeout(timeout, false); !timeoutResult.Valid {
		return "", fmt.Errorf("timeout validation failed: %s", timeoutResult.Reason)
	}

	// Security validation
	securityResult := validateCommand(command, allowShellAccess)
	if !securityResult.Valid {
		auditLog("execute_command", command, "", workingDir, envVars, nil, securityResult, 0, false, -32001, "Security")
		return "", fmt.Errorf("security violation: %s", securityResult.Reason)
	}

	// Execute command
	result, err := executeCommandWithTimeout(command, workingDir, envVars, allowShellAccess, time.Duration(timeout)*time.Second)
	
	// Audit logging
	success := err == nil && result.ExitCode == 0
	errorCode := 0
	errorType := ""
	if !success {
		if result.Timeout {
			errorCode = -32002
			errorType = "Timeout"
		} else {
			errorCode = -32003
			errorType = "Execution"
		}
	}
	
	auditLog("execute_command", command, "", workingDir, envVars, result, securityResult, result.DurationMs, success, errorCode, errorType)

	if err != nil {
		return "", err
	}

	// Return JSON result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

// toolExecuteScript executes a multi-line bash script
func toolExecuteScript(args map[string]interface{}) (string, error) {
	// Extract and validate parameters
	script, ok := args["script"].(string)
	if !ok {
		return "", fmt.Errorf("script is required")
	}

	timeout := 60 // Default for scripts
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}
	if timeout > 600 { // Max for scripts
		timeout = 600
	}

	workingDir := os.Getenv("REPO_PATH")
	if wd, ok := args["working_directory"].(string); ok && wd != "" {
		workingDir = wd
	}

	allowShellAccess := true // Default for scripts
	if asa, ok := args["allow_shell_access"].(bool); ok {
		allowShellAccess = asa
	}

	scriptName := fmt.Sprintf("script_%d", time.Now().Unix())
	if sn, ok := args["script_name"].(string); ok && sn != "" {
		scriptName = sn
	}

	var envVars map[string]string
	if ev, ok := args["environment_vars"].(map[string]interface{}); ok {
		envVars = make(map[string]string)
		for k, v := range ev {
			if val, ok := v.(string); ok {
				envVars[k] = val
			}
		}
	}

	// Validate working directory
	if dirResult := validateWorkingDirectory(workingDir); !dirResult.Valid {
		return "", fmt.Errorf("working directory validation failed: %s", dirResult.Reason)
	}

	// Validate environment variables
	if envResult := validateEnvironmentVariables(envVars); !envResult.Valid {
		return "", fmt.Errorf("environment variable validation failed: %s", envResult.Reason)
	}

	// Validate timeout
	if timeoutResult := validateTimeout(timeout, true); !timeoutResult.Valid {
		return "", fmt.Errorf("timeout validation failed: %s", timeoutResult.Reason)
	}

	// Security validation
	securityResult := validateScript(script)
	if !securityResult.Valid {
		auditLog("execute_script", "", script, workingDir, envVars, nil, securityResult, 0, false, -32001, "Security")
		return "", fmt.Errorf("security violation: %s", securityResult.Reason)
	}

	// Execute script
	result, err := executeScriptWithTimeout(script, workingDir, envVars, allowShellAccess, time.Duration(timeout)*time.Second, scriptName)
	
	// Audit logging
	success := err == nil && result.ExitCode == 0
	errorCode := 0
	errorType := ""
	if !success {
		if result.Timeout {
			errorCode = -32002
			errorType = "Timeout"
		} else {
			errorCode = -32003
			errorType = "Execution"
		}
	}
	
	auditLog("execute_script", "", script, workingDir, envVars, result, securityResult, result.DurationMs, success, errorCode, errorType)

	if err != nil {
		return "", err
	}

	// Return JSON result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

// toolCheckCommandExists checks if a command is available
func toolCheckCommandExists(args map[string]interface{}) (string, error) {
	// Extract and validate parameters
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("command is required")
	}

	// Validate command name format
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(command) {
		return "", fmt.Errorf("invalid command name format: %s", command)
	}

	var searchPaths []string
	if sp, ok := args["search_paths"].([]interface{}); ok {
		for _, path := range sp {
			if p, ok := path.(string); ok {
				searchPaths = append(searchPaths, p)
			}
		}
	}

	// Check if command exists
	result := checkCommandExists(command, searchPaths)
	
	// Audit logging (read-only operation)
	auditLog("check_command_exists", command, "", "", nil, nil, nil, 0, result.Exists, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

// executeCommandWithTimeout executes a command with timeout
func executeCommandWithTimeout(command, workingDir string, envVars map[string]string, allowShellAccess bool, timeout time.Duration) (*CommandResult, error) {
	startTime := time.Now()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Determine bash path for Windows
	bashPath := "bash"
	if _, err := os.Stat("C:\\Program Files\\Git\\bin\\bash.exe"); err == nil {
		bashPath = "C:\\Program Files\\Git\\bin\\bash.exe"
	}
	
	// Prepare command
	var cmd *exec.Cmd
	if allowShellAccess {
		cmd = exec.CommandContext(ctx, bashPath, "-c", command)
	} else {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return nil, fmt.Errorf("empty command")
		}
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}

	// Set working directory
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set environment variables
	if len(envVars) > 0 {
		env := os.Environ()
		for k, v := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Execute command
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	result := &CommandResult{
		ExitCode:   0,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		DurationMs: duration.Milliseconds(),
		Command:    command,
		WorkingDir: workingDir,
		Timeout:    false,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Timeout = true
		return result, fmt.Errorf("command timed out after %v", timeout)
	}

	return result, err
}

// executeScriptWithTimeout executes a script with timeout
func executeScriptWithTimeout(script, workingDir string, envVars map[string]string, allowShellAccess bool, timeout time.Duration, scriptName string) (*CommandResult, error) {
	startTime := time.Now()
	
	// Determine bash path for Windows
	bashPath := "bash"
	if _, err := os.Stat("C:\\Program Files\\Git\\bin\\bash.exe"); err == nil {
		bashPath = "C:\\Program Files\\Git\\bin\\bash.exe"
	}
	
	// Create temporary script file
	tempDir := os.TempDir()
	scriptFile := filepath.Join(tempDir, fmt.Sprintf("%s.sh", scriptName))
	
	// Write script to file
	if err := os.WriteFile(scriptFile, []byte(script), 0755); err != nil {
		return nil, fmt.Errorf("failed to create script file: %w", err)
	}
	defer os.Remove(scriptFile)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	// Prepare command
	var cmd *exec.Cmd
	if allowShellAccess {
		cmd = exec.CommandContext(ctx, bashPath, scriptFile)
	} else {
		cmd = exec.CommandContext(ctx, bashPath, "--restricted", scriptFile)
	}

	// Set working directory
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set environment variables
	if len(envVars) > 0 {
		env := os.Environ()
		for k, v := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Execute script
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	// Count lines in script
	lines := strings.Count(script, "\n") + 1

	result := &CommandResult{
		ExitCode:     0,
		Stdout:       stdout.String(),
		Stderr:       stderr.String(),
		DurationMs:   duration.Milliseconds(),
		Command:      scriptName,
		WorkingDir:   workingDir,
		Timeout:      false,
		LinesExecuted: lines,
		ScriptName:   scriptName,
	}

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Timeout = true
		return result, fmt.Errorf("script timed out after %v", timeout)
	}

	return result, err
}

// checkCommandExists checks if a command exists in PATH or specified paths
func checkCommandExists(command string, searchPaths []string) *CommandExistsResult {
	result := &CommandExistsResult{
		Exists:  false,
		Command: command,
	}

	// Check in specified paths first
	if len(searchPaths) > 0 {
		for _, path := range searchPaths {
			cmdPath := filepath.Join(path, command)
			if _, err := os.Stat(cmdPath); err == nil {
				result.Exists = true
				result.Path = cmdPath
				// Try to get version
				if version := getCommandVersion(command); version != "" {
					result.Version = version
				}
				return result
			}
		}
	}

	// Check in PATH
	if path, err := exec.LookPath(command); err == nil {
		result.Exists = true
		result.Path = path
		// Try to get version
		if version := getCommandVersion(command); version != "" {
			result.Version = version
		}
	} else {
		result.Error = err.Error()
	}

	return result
}

// getCommandVersion attempts to get version information for a command
func getCommandVersion(command string) string {
	// Common version flags
	versionFlags := []string{"--version", "-V", "-v", "version"}
	
	for _, flag := range versionFlags {
		cmd := exec.Command(command, flag)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		if err := cmd.Run(); err == nil {
			output := stdout.String()
			if output == "" {
				output = stderr.String()
			}
			if output != "" {
				// Return first line of version output
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) > 0 {
					return strings.TrimSpace(lines[0])
				}
			}
		}
	}
	
	return ""
}