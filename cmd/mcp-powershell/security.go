package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// validateCommand validates a PowerShell command against security policies
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
	baseCmd := extractPowerShellBaseCommand(command)
	if baseCmd == "" {
		return &SecurityResult{
			Valid:  false,
			Reason: "Unable to determine base command",
			Rule:   "command_extraction",
		}
	}

	// Check if it's a PowerShell cmdlet, function, or external command
	if !isPowerShellCommand(baseCmd) && !defaultSecurityPolicy.AllowedCommands[baseCmd] {
		return &SecurityResult{
			Valid:  false,
			Reason: fmt.Sprintf("Command not allowed: %s", baseCmd),
			Rule:   "allowed_commands",
		}
	}

	// Check shell access restrictions
	if !allowShellAccess && !defaultSecurityPolicy.AllowShellAccess {
		if containsPowerShellShellFeatures(command) {
			return &SecurityResult{
				Valid:  false,
				Reason: "Shell features not allowed",
				Rule:   "shell_access",
			}
		}
	}

	return &SecurityResult{Valid: true}
}

// validateScript validates a PowerShell script against security policies
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
			if matched, _ := regexp.MatchString(strings.ToLower(pattern), strings.ToLower(line)); matched {
				return &SecurityResult{
					Valid:   false,
					Reason:  fmt.Sprintf("Blocked pattern detected at line %d: %s", i+1, pattern),
					Rule:    "blocked_pattern",
					Pattern: pattern,
				}
			}
		}

		// Extract base command from line and check if allowed
		baseCmd := extractPowerShellBaseCommand(line)
		if baseCmd == "" {
			continue
		}

		// Skip variable assignments, function definitions, control structures
		if isPowerShellControlStructure(line) {
			continue
		}

		if !isPowerShellCommand(baseCmd) && !defaultSecurityPolicy.AllowedCommands[baseCmd] {
			return &SecurityResult{
				Valid:  false,
				Reason: fmt.Sprintf("Command not allowed at line %d: %s", i+1, baseCmd),
				Rule:   "allowed_commands",
			}
		}
	}

	// Check for dangerous script constructs
	if containsDangerousPowerShellConstructs(script) {
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
	// Remove null bytes and control characters (except tab and newline)
	result := strings.Builder{}
	for _, r := range input {
		if r != 0 && r != '\r' && (r == '\n' || r == '\t' || r >= 32 && r <= 126) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// extractPowerShellBaseCommand extracts the base command from a PowerShell command line
func extractPowerShellBaseCommand(command string) string {
	// Remove leading pipes and semicolons
	command = strings.TrimSpace(command)
	command = regexp.MustCompile(`^[|;]+`).ReplaceAllString(command, "")

	// Handle PowerShell pipeline
	if strings.Contains(command, "|") {
		parts := strings.Split(command, "|")
		if len(parts) > 0 {
			command = strings.TrimSpace(parts[0])
		}
	}

	// Handle assignment (should skip for command extraction)
	if strings.Contains(command, "=") && !strings.Contains(command, "==") && !strings.Contains(command, "-eq") {
		return "" // Skip assignment lines
	}

	// Split by whitespace and take first part
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}

	// Handle PowerShell parameter names (starting with -)
	if strings.HasPrefix(fields[0], "-") {
		return ""
	}

	// Handle variable references
	if strings.HasPrefix(fields[0], "$") {
		return ""
	}

	return fields[0]
}

// containsPowerShellShellFeatures checks if command contains PowerShell shell features
func containsPowerShellShellFeatures(command string) bool {
	// Check for PowerShell operators and constructs
	shellPatterns := []string{
		`\|`,                    // Pipeline
		`>`,                     // Redirect
		`<`,                     // Redirect
		`>>`,                    // Append redirect
		`;`,                     // Command separator
		`&&`,                    // AND operator (rare in PowerShell but possible)
		`\|\|`,                  // OR operator (rare in PowerShell but possible)
		`&`,                     // Call operator
		`\$\(`,                  // Subexpression
		`\$\{`,                  // Variable property access
		`\(`,                    // Parentheses (can be grouping)
		`\{.*\}`,                // Script block
		`@\(.*\)`,               // Array expression
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

// containsDangerousPowerShellConstructs checks for dangerous PowerShell constructs
func containsDangerousPowerShellConstructs(script string) bool {
	dangerousPatterns := []string{
		`Invoke-Expression`,                           // Code execution
		`Invoke-Command.*-ComputerName.*-ScriptBlock`, // Remote code execution
		`Start-Process.*-Verb\s+RunAs`,                // Elevation
		`Add-Type`,                                    // Load .NET assemblies
		`New-Object.*Reflection\.Assembly`,            // Reflection
		`System\.Reflection`,                          // Reflection namespace
		`Set-Acl`,                                     // ACL modification
		`Get-Acl.*|.*Set-Acl`,                        // ACL manipulation
		`Enable-PSRemoting`,                           // Enable remoting
		`Set-WSManQuickConfig`,                        // WSMan configuration
		`Register-PSSessionConfiguration`,             // Session configuration
		`Set-ExecutionPolicy.*-Force`,                // Forced execution policy
		`Unblock-File.*-Force`,                        // Forced file unblock
		`New-Service`,                                 // Service creation
		`Set-Service.*-Status.*Start`,                // Service start
		`Set-Service.*-Status.*Stop`,                 // Service stop (sometimes legitimate)
		`Remove-Service`,                             // Service removal
		`New-LocalUser`,                              // User creation
		`Remove-LocalUser`,                           // User deletion
		`Set-LocalUser`,                              // User modification
		`Add-LocalGroupMember`,                       // Group membership
		`Remove-LocalGroupMember`,                    // Group membership removal
		`New-ScheduledTask`,                          // Task creation
		`Register-ScheduledTask`,                     // Task registration
		`Unregister-ScheduledTask.*-Force`,           // Forced task removal
		`Set-Content.*-Stream.*-Force`,               // Alternative data streams
		`Get-Content.*-Stream`,                       // Alternative data streams
		`Export-Clixml`,                              // XML export with potential for serialization attacks
		`Import-Clixml`,                              // XML import with potential for deserialization attacks
		`ConvertTo-SecureString.*-AsPlainText`,       // Plain text to secure string
		`ConvertFrom-SecureString.*-AsPlainText`,     // Secure string to plain text
		`Get-Credential.*-Store`,                     // Credential storage
		`New-PSDrive.*-PSProvider.*FileSystem.*Root.*[C-Z]:\\`, // Drive mapping
		`Remove-PSDrive.*-Force`,                     // Forced drive removal
		`Format-Volume`,                              // Disk formatting
		`Clear-Disk`,                                 // Disk clearing
		`Remove-Partition`,                           // Partition removal
		`Initialize-Disk`,                            // Disk initialization
		`Update-HostedCache`,                         // Cache manipulation
		`Clear-HostedCache`,                          // Cache clearing
		`Clear-RecycleBin.*-Force`,                   // Forced recycle bin clearing
		`Remove-Item.*-Recurse.*-Force.*[C-Z]:\\`,    // Dangerous deletion
		`reg.exe.*delete`,                            // Registry deletion
		`reg.exe.*add.*\/f`,                         // Forced registry add
		`wevtutil.*cl`,                               // Event log clearing
		`cipher.exe.*\/w`,                            // Disk wiping
		`sdelete.*-z`,                                // Disk wiping with SDelete
		`Format.exe`,                                 // Format command
		`diskpart.exe`,                               // Disk partitioning
		`net.exe.*user.*\/delete`,                    // User deletion
		`net.exe.*share.*\/delete`,                   // Share deletion
		`net.exe.*stop.*\/y`,                         // Forced service stop
		`sc.exe.*delete`,                             // Service deletion
		`shutdown.exe.*\/s.*\/f`,                    // Forced shutdown
		`shutdown.exe.*\/r.*\/f`,                    // Forced reboot
		`powershell.exe.*-EncodedCommand`,           // Encoded commands
		`powershell.exe.*-ec`,                        // Short form encoded commands
		`powershell.exe.*-NoProfile.*-WindowStyle.*Hidden.*-Command`, // Hidden PowerShell
	}

	lowerScript := strings.ToLower(script)
	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(strings.ToLower(pattern), lowerScript); matched {
			return true
		}
	}

	return false
}

// isPowerShellControlStructure checks if a line is a PowerShell control structure
func isPowerShellControlStructure(line string) bool {
	controlStructures := []string{
		"if ", "else", "elseif", "switch ", "foreach ", "for ", "while ", "do ",
		"try ", "catch ", "finally ", "throw ", "trap ", "break ", "continue ",
		"return ", "exit ", "function ", "filter ", "workflow ", "class ", "enum ",
		"param(", "begin ", "process ", "end ", "dynamicparam ", "using ",
		"#requires ", "#region ", "#endregion ",
	}

	lowerLine := strings.ToLower(strings.TrimSpace(line))
	for _, structure := range controlStructures {
		if strings.HasPrefix(lowerLine, structure) {
			return true
		}
	}

	// Check for assignment patterns (variable assignment)
	assignmentPattern := regexp.MustCompile(`^\$[a-zA-Z_][a-zA-Z0-9_]*\s*=`)
	if assignmentPattern.MatchString(line) {
		return true
	}

	// Check for PowerShell parameter blocks
	if strings.Contains(lowerLine, "param(") {
		return true
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
		// Check name format (PowerShell allows more formats)
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(name) {
			return &SecurityResult{
				Valid:  false,
				Reason: fmt.Sprintf("Invalid environment variable name: %s", name),
				Rule:   "env_var_format",
			}
		}

		// Check for dangerous environment variables (Windows-specific)
		dangerousVars := []string{
			"PATH", "PATHEXT", "COMSPEC", "SYSTEMROOT", "SYSTEMDRIVE",
			"WINDIR", "TEMP", "TMP", "USERPROFILE", "HOMEPATH", "HOMEDRIVE",
			"COMPUTERNAME", "USERNAME", "USERDOMAIN", "LOGONSERVER",
			"PROGRAMFILES", "PROGRAMFILES(X86)", "PROGRAMDATA", "PUBLIC",
			"ALLUSERSPROFILE", "APPDATA", "LOCALAPPDATA",
			"PROCESSOR_ARCHITECTURE", "NUMBER_OF_PROCESSORS", "OS",
			"PSMODULEPATH", "PSExecutionPolicyPreference",
		}

		for _, dangerous := range dangerousVars {
			if strings.EqualFold(name, dangerous) {
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