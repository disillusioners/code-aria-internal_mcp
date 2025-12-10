package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// getShellInfo gathers shell information
func getShellInfo() (*ShellInfo, error) {
	shellInfo := &ShellInfo{}

	// Determine current shell
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			shellPath = "cmd.exe"
			if _, err := exec.LookPath("powershell.exe"); err == nil {
				shellPath = "powershell.exe"
			}
		} else {
			shellPath = "/bin/bash"
		}
	}

	shellInfo.Path = shellPath
	shellInfo.Name = getShellNameFromPath(shellPath)
	shellInfo.Type = getShellType(shellInfo.Name)

	// Get shell version
	version := getShellVersion(shellInfo.Name)
	shellInfo.Version = version

	// Get shell features
	shellInfo.Features = getShellFeatures(shellInfo.Name)

	// Get shell aliases (if possible)
	if shellInfo.Type == "bash" || shellInfo.Type == "zsh" {
		aliases := getShellAliases(shellInfo.Path)
		shellInfo.Aliases = aliases
	}

	// Get shell functions (basic implementation)
	functions := getShellFunctions(shellInfo.Type)
	shellInfo.Functions = functions

	return shellInfo, nil
}

// getShellNameFromPath extracts shell name from path
func getShellNameFromPath(path string) string {
	if path == "" {
		return "unknown"
	}

	parts := strings.Split(path, string(os.PathSeparator))
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Remove extensions on Windows
		if runtime.GOOS == "windows" {
			if idx := strings.Index(name, "."); idx > 0 {
				name = name[:idx]
			}
		}
		return name
	}
	return path
}

// getShellType determines the type of shell
func getShellType(shellName string) string {
	shellName = strings.ToLower(shellName)
	switch {
	case strings.Contains(shellName, "bash"):
		return "bash"
	case strings.Contains(shellName, "zsh"):
		return "zsh"
	case strings.Contains(shellName, "fish"):
		return "fish"
	case strings.Contains(shellName, "powershell"), strings.Contains(shellName, "pwsh"):
		return "powershell"
	case strings.Contains(shellName, "cmd"):
		return "cmd"
	case strings.Contains(shellName, "sh"):
		return "sh"
	case strings.Contains(shellName, "ksh"):
		return "ksh"
	case strings.Contains(shellName, "csh"), strings.Contains(shellName, "tcsh"):
		return "csh"
	default:
		return "unknown"
	}
}

// getShellVersion gets shell version
func getShellVersion(shellName string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch strings.ToLower(shellName) {
	case "bash":
		cmd := exec.CommandContext(ctx, "bash", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			lines := strings.Split(stdout.String(), "\n")
			if len(lines) > 0 {
				parts := strings.Fields(lines[0])
				if len(parts) >= 4 {
					return parts[3]
				}
			}
		}

	case "zsh":
		cmd := exec.CommandContext(ctx, "zsh", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			if parts := strings.Fields(output); len(parts) >= 2 {
				return parts[1]
			}
		}

	case "powershell", "pwsh":
		// Try both pwsh and powershell
		for _, psName := range []string{"pwsh", "powershell"} {
			cmd := exec.CommandContext(ctx, psName, "-NoProfile", "-Command", "$PSVersionTable.PSVersion")
			var stdout strings.Builder
			cmd.Stdout = &stdout
			if err := cmd.Run(); err == nil {
				output := stdout.String()
				if strings.Contains(output, "Major") {
					// Try to extract version
					if idx := strings.Index(output, "{"); idx >= 0 {
						jsonStr := output[idx:]
						var versionInfo map[string]interface{}
						if err := json.Unmarshal([]byte(jsonStr), &versionInfo); err == nil {
							if psVersion, ok := versionInfo["PSVersion"].(map[string]interface{}); ok {
								if major, ok := psVersion["Major"].(float64); ok {
									if minor, ok := psVersion["Minor"].(float64); ok {
										return strings.TrimSpace(output)
									}
								}
							}
						}
					}
					return strings.TrimSpace(output)
				}
			}
		}

	case "cmd":
		// Windows CMD doesn't have an easy version command
		return "Windows Command Prompt"
	}

	return ""
}

// getShellFeatures returns a list of shell features
func getShellFeatures(shellName string) []string {
	shellType := getShellType(shellName)
	var features []string

	switch shellType {
	case "bash":
		features = []string{
			"command_history", "tab_completion", "command_substitution",
			"process_substitution", "arrays", "functions", "aliases",
			"brace_expansion", "globbing", "redirection", "pipelines",
			"job_control", "command_line_editing", "programmable_completion",
		}

	case "zsh":
		features = []string{
			"command_history", "tab_completion", "command_substitution",
			"process_substitution", "arrays", "associative_arrays", "functions",
			"aliases", "brace_expansion", "extended_globbing", "redirection",
			"pipelines", "job_control", "command_line_editing", "zle",
			"programmable_completion", "theme_support", "plugin_system",
		}

	case "powershell":
		features = []string{
			"command_history", "tab_completion", "pipeline_support",
			"objects", "modules", "functions", "aliases", "providers",
			"remoting", "workflow", "dsc", "classes", "enums",
			"error_handling", "structured_data", "xml_json_support",
		}

	case "fish":
		features = []string{
			"command_history", "tab_completion", "syntax_highlighting",
			"autosuggestions", "functions", "aliases", "variables",
			"job_control", "web_configuration", "universal_variables",
		}

	case "cmd":
		features = []string{
			"command_history", "batch_files", "environment_variables",
			"pipelines", "redirection", "built-in_commands",
		}

	default:
		features = []string{"basic_shell_features"}
	}

	return features
}

// getShellAliases gets shell aliases (basic implementation)
func getShellAliases(shellPath string) map[string]string {
	aliases := make(map[string]string)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// For bash, try to get aliases
	if strings.Contains(shellPath, "bash") {
		cmd := exec.CommandContext(ctx, "bash", "-i", "-c", "alias")
		var stdout strings.Builder
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			lines := strings.Split(stdout.String(), "\n")
			for _, line := range lines {
				if strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						alias := strings.TrimPrefix(parts[0], "alias ")
						value := strings.Trim(parts[1], "'\"")
						aliases[alias] = value
					}
				}
			}
		}
	}

	return aliases
}

// getShellFunctions returns a list of common shell functions
func getShellFunctions(shellType string) []string {
	switch shellType {
	case "bash":
		return []string{
			"cd", "pushd", "popd", "dirs", "history", "type", "which",
			"man", "help", "source", "exec", "exit", "return", "test",
			"[", "echo", "printf", "read", "mapfile", "readarray",
		}

	case "powershell":
		return []string{
			"Get-Help", "Get-Command", "Get-Member", "Get-ChildItem", "Set-Location",
			"Get-Location", "Write-Output", "Write-Host", "Read-Host", "Get-Content",
			"Set-Content", "Add-Content", "Test-Path", "New-Item", "Remove-Item",
		}

	case "cmd":
		return []string{
			"dir", "cd", "md", "rd", "del", "copy", "move", "ren",
			"type", "find", "sort", "more", "help", "exit", "cls",
		}

	default:
		return []string{}
	}
}