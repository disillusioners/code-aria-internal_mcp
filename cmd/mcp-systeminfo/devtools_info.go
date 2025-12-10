package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// getDevelopmentToolsInfo gathers development tools information
func getDevelopmentToolsInfo() (*DevelopmentToolsInfo, error) {
	devToolsInfo := &DevelopmentToolsInfo{}

	// Check various development tools
	devToolsInfo.Go = getToolInfo("go")
	devToolsInfo.Node = getToolInfo("node")
	devToolsInfo.Python = getToolInfo("python")
	devToolsInfo.Python3 = getToolInfo("python3")
	devToolsInfo.Ruby = getToolInfo("ruby")
	devToolsInfo.Java = getToolInfo("java")
	devToolsInfo.Git = getToolInfo("git")
	devToolsInfo.Docker = getToolInfo("docker")
	devToolsInfo.PowerShell = getToolInfo("pwsh")
	if devToolsInfo.PowerShell == nil || !devToolsInfo.PowerShell.Installed {
		devToolsInfo.PowerShell = getToolInfo("powershell")
	}
	devToolsInfo.CMake = getToolInfo("cmake")
	devToolsInfo.Maven = getToolInfo("mvn")
	devToolsInfo.Gradle = getToolInfo("gradle")
	devToolsInfo.Make = getToolInfo("make")
	devToolsInfo.Cargo = getToolInfo("cargo")
	devToolsInfo.Rustc = getToolInfo("rustc")
	devToolsInfo.GCC = getToolInfo("gcc")
	devToolsInfo.Clang = getToolInfo("clang")
	devToolsInfo.Dotnet = getToolInfo("dotnet")

	// Get package managers
	devToolsInfo.PackageMgrs = getPackageManagers()

	return devToolsInfo, nil
}

// getToolInfo checks if a tool is installed and gets its information
func getToolInfo(toolName string) *ToolInfo {
	toolInfo := &ToolInfo{
		Executable: toolName,
		Installed:  false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check if tool exists
	path, err := exec.LookPath(toolName)
	if err != nil {
		return toolInfo // Not installed
	}

	toolInfo.Installed = true
	toolInfo.Path = path

	// Get version information
	version := getToolVersion(toolName, path)
	toolInfo.Version = version

	// Get tool-specific features
	features := getToolFeatures(toolName)
	toolInfo.Features = features

	return toolInfo
}

// getToolVersion gets version information for a specific tool
func getToolVersion(toolName, path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	switch toolName {
	case "go":
		cmd := exec.CommandContext(ctx, "go", "version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			if strings.Contains(output, "go version") {
				parts := strings.Fields(output)
				if len(parts) >= 3 {
					return parts[2]
				}
			}
		}

	case "node":
		cmd := exec.CommandContext(ctx, "node", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}

	case "python", "python3":
		cmd := exec.CommandContext(ctx, toolName, "--version")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stderr.String())
		}

	case "git":
		cmd := exec.CommandContext(ctx, "git", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			if strings.Contains(output, "git version") {
				parts := strings.Fields(output)
				if len(parts) >= 3 {
					return strings.Join(parts[2:], " ")
				}
			}
		}

	case "docker":
		cmd := exec.CommandContext(ctx, "docker", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}

	case "cargo":
		cmd := exec.CommandContext(ctx, "cargo", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}

	case "rustc":
		cmd := exec.CommandContext(ctx, "rustc", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}

	case "gcc":
		cmd := exec.CommandContext(ctx, "gcc", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			lines := strings.Split(output, "\n")
			if len(lines) > 0 {
				return lines[0]
			}
		}

	case "clang":
		cmd := exec.CommandContext(ctx, "clang", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			lines := strings.Split(output, "\n")
			if len(lines) > 0 {
				return lines[0]
			}
		}

	case "dotnet":
		cmd := exec.CommandContext(ctx, "dotnet", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}

	case "java":
		cmd := exec.CommandContext(ctx, "java", "-version")
		var stderr strings.Builder
		cmd.Stderr = &stderr
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stderr.String())
			if strings.Contains(output, "version") {
				return output
			}
		}

	case "cmake":
		cmd := exec.CommandContext(ctx, "cmake", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			lines := strings.Split(output, "\n")
			if len(lines) > 0 {
				return lines[0]
			}
		}

	case "make":
		cmd := exec.CommandContext(ctx, "make", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			lines := strings.Split(output, "\n")
			if len(lines) > 0 {
				return lines[0]
			}
		}

	case "powershell", "pwsh":
		cmd := exec.CommandContext(ctx, toolName, "-NoProfile", "-Command", "$PSVersionTable.PSVersion")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			return strings.TrimSpace(stdout.String())
		}

	case "mvn":
		cmd := exec.CommandContext(ctx, "mvn", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			lines := strings.Split(output, "\n")
			if len(lines) > 0 {
				return lines[0]
			}
		}

	case "gradle":
		cmd := exec.CommandContext(ctx, "gradle", "--version")
		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			output := strings.TrimSpace(stdout.String())
			if strings.Contains(output, "Gradle") {
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if strings.Contains(line, "Gradle") {
						return strings.TrimSpace(line)
					}
				}
			}
		}

	default:
		// Try common version flags
		versionFlags := []string{"--version", "-V", "-v", "version", "ver"}
		for _, flag := range versionFlags {
			cmd := exec.CommandContext(ctx, toolName, flag)
			var stdout strings.Builder
			cmd.Stdout = &stdout
			if err := cmd.Run(); err == nil {
				if output := strings.TrimSpace(stdout.String()); output != "" {
					return output
				}
			}
		}
	}

	return ""
}

// getToolFeatures returns tool-specific features
func getToolFeatures(toolName string) []string {
	switch toolName {
	case "go":
		return []string{
			"static_compilation", "concurrent_programming", "garbage_collection",
			"cross_compilation", "testing_framework", "modules", "interfaces",
			"goroutines", "channels", "select_statements", "defer_statements",
		}

	case "node":
		return []string{
			"javascript_runtime", "npm_support", "es_modules", "async_programming",
			"event_loop", "v8_engine", "npm_scripts", "package_json",
		}

	case "git":
		return []string{
			"version_control", "distributed", "branching", "merging", "rebasing",
			"staging_area", "hooks", "submodules", "lfs_support", "bisect",
		}

	case "docker":
		return []string{
			"containerization", "dockerfile", "docker_compose", "volumes",
			"networking", "multi_stage_builds", "health_checks", "secrets",
		}

	case "cargo":
		return []string{
			"rust_package_manager", "crates_io", "dependencies", "workspaces",
			"build_scripts", "cross_compilation", "documentation_generation",
		}

	default:
		return []string{}
	}
}

// getPackageManagers returns information about available package managers
func getPackageManagers() []PackageMgrInfo {
	var packageManagers []PackageMgrInfo

	// Check for various package managers
	pkgManagerTools := map[string]string{
		"apt":     "apt",
		"apt-get": "apt",
		"yum":     "yum",
		"dnf":     "dnf",
		"pacman":  "pacman",
		"zypper":  "zypper",
		"brew":    "brew",
		"port":    "macports",
		"pkg":     "pkg",
		"choco":   "chocolatey",
		"winget":  "winget",
		"scoop":   "scoop",
	}

	for tool, pkgType := range pkgManagerTools {
		if toolInfo := getToolInfo(tool); toolInfo.Installed {
			version := toolInfo.Version
			if version == "" {
				// Try to get version in a different way
				if tool == "apt" || tool == "apt-get" {
					version = getAptVersion()
				}
			}

			packageManagers = append(packageManagers, PackageMgrInfo{
				Name:    tool,
				Version: version,
				Type:    pkgType,
			})
		}
	}

	return packageManagers
}

// getAptVersion gets APT version specifically
func getAptVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "apt", "--version")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		output := strings.TrimSpace(stdout.String())
		lines := strings.Split(output, "\n")
		if len(lines) > 0 {
			return lines[0]
		}
	}
	return ""
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
			cmdPath := path
			if !strings.HasSuffix(path, string([]rune{os.PathSeparator})) {
				cmdPath = path + string([]rune{os.PathSeparator}) + command
			}
			if _, err := os.Stat(cmdPath); err == nil {
				result.Exists = true
				result.Path = cmdPath
				// Try to get version
				if version := getToolVersion(command, cmdPath); version != "" {
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
		if version := getToolVersion(command, path); version != "" {
			result.Version = version
		}
	} else {
		result.Error = err.Error()
	}

	return result
}