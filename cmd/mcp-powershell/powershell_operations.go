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
	"runtime"
	"strings"
	"time"
)

// Global security policy - in production, this should be configurable
var defaultSecurityPolicy = SecurityPolicy{
	AllowedCommands: map[string]bool{
		// Basic file operations
		"Get-ChildItem": true, "Get-Content": true, "Set-Content": true, "Add-Content": true,
		"Get-Item": true, "Test-Path": true, "New-Item": true, "Remove-Item": true,
		"Copy-Item": true, "Move-Item": true, "Rename-Item": true, "Get-Location": true,
		"Set-Location": true, "Get-Date": true, "Get-Host": true, "Write-Host": true,
		"Write-Output": true, "Out-File": true, "Out-String": true,

		// Development tools
		"git": true, "npm": true, "yarn": true, "dotnet": true, "msbuild": true,
		"go": true, "python": true, "python3": true, "node": true,
		"java": true, "javac": true, "mvn": true, "gradle": true,
		"cargo": true, "rustc": true, "gcc": true, "g++": true,
		"clang": true, "clang++": true, "choco": true, "winget": true,

		// System information
		"Get-Process": true, "Get-Service": true, "Get-ComputerInfo": true, "Get-WmiObject": true,
		"Get-CimInstance": true, "Get-Variable": true, "Get-Module": true, "Get-Command": true,

		// Text processing
		"Select-String": true, "Select-Object": true, "Where-Object": true, "ForEach-Object": true,
		"Sort-Object": true, "Group-Object": true, "Measure-Object": true, "Compare-Object": true,
		"Join-String": true, "Split-String": true, "Replace-String": true,

		// Archive tools
		"Compress-Archive": true, "Expand-Archive": true, "tar": true, "zip": true,

		// Network tools (read-only)
		"Test-Connection": true, "Test-NetConnection": true, "Resolve-DnsName": true,
		"Invoke-WebRequest": true, "Invoke-RestMethod": true, "curl": true, "wget": true,

		// Process management
		"Stop-Process": true, "Start-Process": true, "Wait-Process": true,

		// Windows-specific commands
		"dir": true, "type": true, "copy": true, "move": true, "del": true,
		"md": true, "rd": true, "cls": true, "echo": true, "cd": true,
		"whoami": true, "hostname": true, "ipconfig": true, "netstat": true,
		"tasklist": true, "taskkill": true,
	},
	BlockedPatterns: []string{
		`Remove-Item\s+-Recurse\s+-Force\s+[C-Z]:\\`,  // Dangerous deletion
		`Format-Volume`,                                   // Disk formatting
		`Clear-Disk`,                                      // Disk clearing
		`Remove-Partition`,                               // Partition removal
		`Set-ExecutionPolicy\s+.*-Force`,                 // Forced execution policy
		`Invoke-Expression`,                              // Code execution
		`Start-Process.*-Verb\s+RunAs`,                  // Elevation
		`Enable-PSRemoting`,                              // Remote execution
		`Set-Service.*-Status\s+Stopped`,                // Stopping services
		`Stop-Computer`,                                   // System shutdown
		`Restart-Computer`,                               // System reboot
		`Set-LocalUser`,                                  // User management
		`Set-ADUser`,                                     // Active Directory
		`Set-Acl`,                                        // Permission changes
		`Take-Ownership`,                                 // Ownership changes
		`reg\s+delete`,                                   // Registry deletion
		`reg\s+add.*\/f`,                                 // Force registry add
		`schtasks.*\/delete`,                             // Task deletion
		`wevtutil.*cl`,                                   // Event log clearing
		`cipher\s+\/w`,                                   // Disk wiping
		`format`,                                         // Format command
		`diskpart`,                                       // Disk partitioning
		`net\s+user.*\/delete`,                           // User deletion
		`net\s+share.*\/delete`,                          // Share deletion
	},
	MaxCommandLen:          1000,
	MaxScriptLen:           10000,
	DefaultTimeout:         30,
	MaxTimeout:             300,
	AllowShellAccess:       false,
	AllowExecutionPolicy:   false, // Don't change execution policy by default
}

// toolExecuteCommand executes a single PowerShell command
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

// toolExecuteScript executes a multi-line PowerShell script
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
	if !regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`).MatchString(command) {
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

	// Determine PowerShell path
	powerShellPath := getPowerShellPath()

	// Prepare command
	var cmd *exec.Cmd
	if allowShellAccess {
		// Allow shell features like pipelines, operators
		cmd = exec.CommandContext(ctx, powerShellPath, "-Command", command)
	} else {
		// More restrictive mode for individual commands
		cmd = exec.CommandContext(ctx, powerShellPath, "-NoProfile", "-Command", command)
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
		if runtime.GOOS == "windows" {
			// On Windows, PowerShell returns 0 for success and 1 for errors
			result.ExitCode = cmd.ProcessState.ExitCode()
		} else {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}
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

	// Determine PowerShell path
	powerShellPath := getPowerShellPath()

	// Create temporary script file
	tempDir := os.TempDir()
	scriptFile := filepath.Join(tempDir, fmt.Sprintf("%s.ps1", scriptName))

	// Write script to file
	if err := os.WriteFile(scriptFile, []byte(script), 0644); err != nil {
		return nil, fmt.Errorf("failed to create script file: %w", err)
	}
	defer os.Remove(scriptFile)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Prepare PowerShell execution policy arguments
	args := []string{"-NoProfile"}
	if !allowShellAccess {
		args = append(args, "-NoLogo")
	}
	if !defaultSecurityPolicy.AllowExecutionPolicy {
		args = append(args, "-ExecutionPolicy", "Bypass")
	}
	args = append(args, "-File", scriptFile)

	// Prepare command
	cmd := exec.CommandContext(ctx, powerShellPath, args...)

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
		if runtime.GOOS == "windows" {
			result.ExitCode = cmd.ProcessState.ExitCode()
		} else {
			result.ExitCode = cmd.ProcessState.ExitCode()
		}
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.Timeout = true
		return result, fmt.Errorf("script timed out after %v", timeout)
	}

	return result, err
}

// getPowerShellPath returns the path to PowerShell executable
func getPowerShellPath() string {
	// Try PowerShell Core first
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path
	}

	// Fall back to Windows PowerShell
	if path, err := exec.LookPath("powershell"); err == nil {
		return path
	}

	// Default fallback
	return "powershell"
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
			// Try different extensions for Windows executables
			extensions := []string{"", ".exe", ".ps1", ".cmd", ".bat"}
			for _, ext := range extensions {
				cmdPath := filepath.Join(path, command+ext)
				if _, err := os.Stat(cmdPath); err == nil {
					result.Exists = true
					result.Path = cmdPath
					// Try to get version
					if version := getCommandVersion(command, cmdPath); version != "" {
						result.Version = version
					}
					return result
				}
			}
		}
	}

	// Check if it's a PowerShell cmdlet or function
	if isPowerShellCommand(command) {
		result.Exists = true
		result.Path = fmt.Sprintf("PowerShell: %s", command)
		if version := getPowerShellVersion(); version != "" {
			result.Version = fmt.Sprintf("PowerShell %s", version)
		}
		return result
	}

	// Check in PATH
	if path, err := exec.LookPath(command); err == nil {
		result.Exists = true
		result.Path = path
		// Try to get version
		if version := getCommandVersion(command, path); version != "" {
			result.Version = version
		}
	} else {
		result.Error = err.Error()
	}

	return result
}

// isPowerShellCommand checks if a command is a built-in PowerShell command
func isPowerShellCommand(command string) bool {
	// Common PowerShell cmdlets that might not be in PATH
	powerShellCommands := map[string]bool{
		"Get-ChildItem": true, "Get-Content": true, "Set-Content": true,
		"Add-Content": true, "Get-Item": true, "Test-Path": true,
		"New-Item": true, "Remove-Item": true, "Copy-Item": true,
		"Move-Item": true, "Rename-Item": true, "Get-Location": true,
		"Set-Location": true, "Get-Date": true, "Get-Host": true,
		"Write-Host": true, "Write-Output": true, "Out-File": true,
		"Out-String": true, "Get-Process": true, "Get-Service": true,
		"Get-ComputerInfo": true, "Get-WmiObject": true,
		"Get-CimInstance": true, "Get-Variable": true, "Get-Module": true,
		"Get-Command": true, "Select-String": true, "Select-Object": true,
		"Where-Object": true, "ForEach-Object": true, "Sort-Object": true,
		"Group-Object": true, "Measure-Object": true, "Compare-Object": true,
		"Join-String": true, "Split-String": true, "Replace-String": true,
		"Compress-Archive": true, "Expand-Archive": true, "Test-Connection": true,
		"Test-NetConnection": true, "Resolve-DnsName": true,
		"Invoke-WebRequest": true, "Invoke-RestMethod": true,
		"Stop-Process": true, "Start-Process": true, "Wait-Process": true,
	}

	return powerShellCommands[command]
}

// getCommandVersion attempts to get version information for a command
func getCommandVersion(command, path string) string {
	powerShellPath := getPowerShellPath()

	// Try common version patterns
	versionCommands := []string{
		fmt.Sprintf(`%s --version`, command),
		fmt.Sprintf(`%s -Version`, command),
		fmt.Sprintf(`%s -v`, command),
		fmt.Sprintf(`%s /?`, command),
		fmt.Sprintf(`Get-Command %s -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Version`, command),
		fmt.Sprintf(`(Get-Item %s).VersionInfo.ProductVersion`, path),
	}

	for _, cmd := range versionCommands {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmdExec := exec.CommandContext(ctx, powerShellPath, "-Command", cmd)
		var stdout, stderr bytes.Buffer
		cmdExec.Stdout = &stdout
		cmdExec.Stderr = &stderr

		if err := cmdExec.Run(); err == nil {
			output := stdout.String()
			if output == "" {
				output = stderr.String()
			}
			if output != "" {
				// Return first line of version output
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) > 0 {
					version := strings.TrimSpace(lines[0])
					if version != "" && !strings.Contains(strings.ToLower(version), "error") {
						return version
					}
				}
			}
		}
		cancel()
	}

	return ""
}

// getPowerShellVersion returns the PowerShell version
func getPowerShellVersion() string {
	powerShellPath := getPowerShellPath()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, powerShellPath, "-Command", "$PSVersionTable.PSVersion")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err == nil {
		output := stdout.String()
		if strings.Contains(output, "Major") {
			// Extract major version
			re := regexp.MustCompile(`Major\s+(\d+)`)
			if match := re.FindStringSubmatch(output); len(match) > 1 {
				return match[1]
			}
		}
		return strings.TrimSpace(output)
	}

	return ""
}