package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Global security policy - more permissive for read-only operations
var defaultSecurityPolicy = SecurityPolicy{
	AllowedCommands: map[string]bool{
		// System information commands
		"uname": true, "hostname": true, "whoami": true, "id": true, "lscpu": true,
		"free": true, "df": true, "mount": true, "lsblk": true, "lspci": true,
		"lsusb": true, "dmidecode": true, "uptime": true, "date": true,
		"env": true, "printenv": true, "which": true, "whereis": true,
		"wmic": true, "systeminfo": true, "powershell": true, "cmd": true,
		"ps": true, "top": true, "htop": true, "netstat": true, "ss": true,
		"ip": true, "ifconfig": true, "route": true, "ping": true, "curl": true,
		"wget": true, "git": true, "npm": true, "node": true, "go": true,
		"python": true, "python3": true, "pip": true, "pip3": true,
		"java": true, "javac": true, "mvn": true, "gradle": true,
		"docker": true, "docker-compose": true, "kubectl": true,
		"cargo": true, "rustc": true, "gcc": true, "g++": true, "clang": true,
		"make": true, "cmake": true, "dotnet": true, "pwsh": true,

		// Package managers
		"apt": true, "apt-get": true, "yum": true, "dnf": true, "pacman": true,
		"zypper": true, "brew": true, "port": true, "pkg": true, "pkgin": true,
		"choco": true, "winget": true, "scoop": true,
	},
	BlockedPatterns: []string{
		// Block dangerous operations even though these are read-only
		`rm\s+`,        // File deletion
		`dd\s+`,        // Disk operations
		`mkfs`,         // Filesystem operations
		`fdisk`,        // Disk partitioning
		`format`,       // Windows format
		`del\s+`,       // Windows delete
		`rmdir`,        // Directory removal
		`shutdown`,     // System shutdown
		`reboot`,       // System reboot
		`halt`,         // System halt
		`poweroff`,     // Power off
		`passwd`,       // Password changes
		`su\s+`,        // User switching
		`sudo\s+`,      // Privilege escalation (unless specifically needed)
		`chmod\s+.*[457][457][457]`, // Dangerous permissions
		`chown\s+.*root`,           // Ownership changes to root
	},
	MaxCommandLen:    500,
	DefaultTimeout:   10,
	MaxTimeout:       30,
	AllowShellAccess: false,
}

// toolGetSystemInfo returns comprehensive system information
func toolGetSystemInfo(args map[string]interface{}) (string, error) {
	// Get all system information components
	osInfo, _ := getOSInfo()
	hardwareInfo, _ := getHardwareInfo()
	environmentInfo, _ := getEnvironmentInfo()
	shellInfo, _ := getShellInfo()
	devToolsInfo, _ := getDevelopmentToolsInfo()
	networkInfo, _ := getNetworkInfo()
	reposInfo, _ := detectRepositories()
	recommendations := getSystemRecommendations(osInfo, hardwareInfo, devToolsInfo)

	systemInfo := SystemInfo{
		Timestamp:      time.Now().UTC(),
		OS:             *osInfo,
		Hardware:       *hardwareInfo,
		Environment:    *environmentInfo,
		Shell:          *shellInfo,
		Development:    *devToolsInfo,
		Networking:     *networkInfo,
		Repositories:   reposInfo,
		Recommendations: recommendations,
	}

	// Audit logging
	auditLog("get_system_info", "", "", environmentInfo.WorkingDir, nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(systemInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal system info: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetOSInfo returns operating system information
func toolGetOSInfo(args map[string]interface{}) (string, error) {
	osInfo, err := getOSInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get OS info: %w", err)
	}

	// Audit logging
	auditLog("get_os_info", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(osInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OS info: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetHardwareInfo returns hardware information
func toolGetHardwareInfo(args map[string]interface{}) (string, error) {
	hardwareInfo, err := getHardwareInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get hardware info: %w", err)
	}

	// Audit logging
	auditLog("get_hardware_info", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(hardwareInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal hardware info: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetEnvironmentInfo returns environment information
func toolGetEnvironmentInfo(args map[string]interface{}) (string, error) {
	environmentInfo, err := getEnvironmentInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get environment info: %w", err)
	}

	// Audit logging
	auditLog("get_environment_info", "", "", environmentInfo.WorkingDir, nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(environmentInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal environment info: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetShellInfo returns shell information
func toolGetShellInfo(args map[string]interface{}) (string, error) {
	shellInfo, err := getShellInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get shell info: %w", err)
	}

	// Audit logging
	auditLog("get_shell_info", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(shellInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal shell info: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetDevelopmentTools returns development tools information
func toolGetDevelopmentTools(args map[string]interface{}) (string, error) {
	devToolsInfo, err := getDevelopmentToolsInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get development tools info: %w", err)
	}

	// Audit logging
	auditLog("get_development_tools", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(devToolsInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal development tools info: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetNetworkInfo returns network information
func toolGetNetworkInfo(args map[string]interface{}) (string, error) {
	networkInfo, err := getNetworkInfo()
	if err != nil {
		return "", fmt.Errorf("failed to get network info: %w", err)
	}

	// Audit logging
	auditLog("get_network_info", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(networkInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal network info: %w", err)
	}
	return string(resultJSON), nil
}

// toolDetectRepositories detects version control repositories
func toolDetectRepositories(args map[string]interface{}) (string, error) {
	reposInfo, err := detectRepositories()
	if err != nil {
		return "", fmt.Errorf("failed to detect repositories: %w", err)
	}

	// Audit logging
	auditLog("detect_repositories", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(reposInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal repositories info: %w", err)
	}
	return string(resultJSON), nil
}

// toolCheckCommand checks if a command is available
func toolCheckCommand(args map[string]interface{}) (string, error) {
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
	auditLog("check_command", command, "", "", nil, nil, nil, 0, result.Exists, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal command check result: %w", err)
	}
	return string(resultJSON), nil
}

// toolGetRecommendations returns system-specific recommendations
func toolGetRecommendations(args map[string]interface{}) (string, error) {
	osInfo, _ := getOSInfo()
	hardwareInfo, _ := getHardwareInfo()
	devToolsInfo, _ := getDevelopmentToolsInfo()

	recommendations := getSystemRecommendations(osInfo, hardwareInfo, devToolsInfo)

	// Audit logging
	auditLog("get_recommendations", "", "", "", nil, nil, nil, 0, true, 0, "")

	// Return JSON result
	resultJSON, err := json.Marshal(map[string]interface{}{
		"recommendations": recommendations,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal recommendations: %w", err)
	}
	return string(resultJSON), nil
}

// getOSInfo gathers operating system information
func getOSInfo() (*OSInfo, error) {
	osInfo := &OSInfo{
		Name:         runtime.GOOS,
		Architecture: runtime.GOARCH,
		Platform:     runtime.GOARCH, // Will be updated with more specific info
	}

	if runtime.GOOS == "windows" {
		return getWindowsOSInfo(osInfo)
	} else {
		return getUnixOSInfo(osInfo)
	}
}

// getWindowsOSInfo gets Windows-specific OS information
func getWindowsOSInfo(osInfo *OSInfo) (*OSInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try PowerShell first for detailed info
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-ComputerInfo | Select-Object WindowsProductName, WindowsVersion, OsHardwareAbstractionLayer | ConvertTo-Json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err == nil {
		var result map[string]interface{}
		if json.Unmarshal(stdout.Bytes(), &result) == nil {
			if product, ok := result["WindowsProductName"].(string); ok {
				osInfo.Name = product
			}
			if version, ok := result["WindowsVersion"].(string); ok {
				osInfo.Version = version
			}
			if hal, ok := result["OsHardwareAbstractionLayer"].(string); ok {
				osInfo.KernelVersion = hal
			}
		}
	}

	// Fallback to systeminfo command
	if osInfo.Name == "windows" {
		cmd = exec.CommandContext(ctx, "systeminfo")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			output := stdout.String()
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, "OS Name:") {
					osInfo.Name = strings.TrimSpace(strings.Split(line, ":")[1])
				}
				if strings.Contains(line, "OS Version:") {
					osInfo.Version = strings.TrimSpace(strings.Split(line, ":")[1])
				}
				if strings.Contains(line, "System Type:") {
					osInfo.Architecture = strings.TrimSpace(strings.Split(line, ":")[1])
				}
			}
		}
	}

	osInfo.Platform = "Windows"

	// Parse version components
	if osInfo.Version != "" {
		parseVersion(osInfo)
	}

	return osInfo, nil
}

// getUnixOSInfo gets Unix/Linux/macOS-specific OS information
func getUnixOSInfo(osInfo *OSInfo) (*OSInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get detailed OS information using uname
	cmd := exec.CommandContext(ctx, "uname", "-a")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err == nil {
		output := stdout.String()
		parts := strings.Fields(output)
		if len(parts) >= 3 {
			osInfo.KernelVersion = strings.Join(parts[2:], " ")
		}
	}

	// Try to get distribution information
	if runtime.GOOS == "linux" {
		// Try /etc/os-release
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					osInfo.Name = strings.Trim(strings.Split(line, "=")[1], `"`)
				}
				if strings.HasPrefix(line, "VERSION_ID=") {
					osInfo.Version = strings.Trim(strings.Split(line, "=")[1], `"`)
				}
				if strings.HasPrefix(line, "ID=") {
					osInfo.Distribution = strings.Trim(strings.Split(line, "=")[1], `"`)
				}
			}
		}

		// Fallback to lsb_release
		if osInfo.Name == "linux" {
			cmd = exec.CommandContext(ctx, "lsb_release", "-a")
			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			if err := cmd.Run(); err == nil {
				output := stdout.String()
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if strings.Contains(line, "Distributor ID:") {
						osInfo.Distribution = strings.TrimSpace(strings.Split(line, ":")[1])
					}
					if strings.Contains(line, "Description:") {
						osInfo.Name = strings.TrimSpace(strings.Split(line, ":")[1])
					}
					if strings.Contains(line, "Release:") {
						osInfo.Version = strings.TrimSpace(strings.Split(line, ":")[1])
					}
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		cmd = exec.CommandContext(ctx, "sw_vers", "-productVersion")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			osInfo.Version = strings.TrimSpace(stdout.String())
			osInfo.Name = "macOS"
			osInfo.Distribution = "macOS"
		}
	}

	// Set platform name
	switch runtime.GOOS {
	case "linux":
		osInfo.Platform = "Linux"
	case "darwin":
		osInfo.Platform = "macOS"
	default:
		osInfo.Platform = runtime.GOOS
	}

	// Parse version components
	if osInfo.Version != "" {
		parseVersion(osInfo)
	}

	return osInfo, nil
}

// parseVersion parses version string into major, minor, patch components
func parseVersion(osInfo *OSInfo) {
	parts := strings.Split(osInfo.Version, ".")
	if len(parts) >= 1 {
		if major, err := strconv.Atoi(parts[0]); err == nil {
			osInfo.Major = major
		}
	}
	if len(parts) >= 2 {
		if minor, err := strconv.Atoi(parts[1]); err == nil {
			osInfo.Minor = minor
		}
	}
	if len(parts) >= 3 {
		if patch, err := strconv.Atoi(parts[2]); err == nil {
			osInfo.Patch = patch
		}
	}
}

// getHardwareInfo gathers hardware information
func getHardwareInfo() (*HardwareInfo, error) {
	hardwareInfo := &HardwareInfo{}

	// Get CPU information
	cpuInfo, _ := getCPUInfo()
	hardwareInfo.CPU = *cpuInfo

	// Get memory information
	memInfo, _ := getMemoryInfo()
	hardwareInfo.Memory = *memInfo

	// Get storage information
	storageInfo, _ := getStorageInfo()
	hardwareInfo.Storage = storageInfo

	// Get display information (if available)
	displayInfo, _ := getDisplayInfo()
	hardwareInfo.Displays = displayInfo

	// Get network cards information (if available)
	networkCardsInfo, _ := getNetworkCardsInfo()
	hardwareInfo.NetworkCards = networkCardsInfo

	return hardwareInfo, nil
}

// getCPUInfo gathers CPU information
func getCPUInfo() (*CPUInfo, error) {
	cpuInfo := &CPUInfo{
		Architecture: runtime.GOARCH,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		// Windows CPU info using PowerShell
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-WmiObject -Class Win32_Processor | Select-Object Name, Manufacturer, MaxClockSpeed, NumberOfCores, NumberOfLogicalProcessors | ConvertTo-Json")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			var result map[string]interface{}
			if json.Unmarshal(stdout.Bytes(), &result) == nil {
				if name, ok := result["Name"].(string); ok {
					cpuInfo.ModelName = name
				}
				if manufacturer, ok := result["Manufacturer"].(string); ok {
					cpuInfo.Vendor = manufacturer
				}
				if speed, ok := result["MaxClockSpeed"].(float64); ok {
					cpuInfo.Frequency = speed / 1000.0 // Convert MHz to GHz
				}
				if cores, ok := result["NumberOfCores"].(float64); ok {
					cpuInfo.Cores = int(cores)
				}
				if threads, ok := result["NumberOfLogicalProcessors"].(float64); ok {
					cpuInfo.Threads = int(threads)
				}
			}
		}
	} else {
		// Unix/Linux CPU info
		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "model name") {
					cpuInfo.ModelName = strings.TrimSpace(strings.Split(line, ":")[1])
				}
				if strings.HasPrefix(line, "vendor_id") {
					cpuInfo.Vendor = strings.TrimSpace(strings.Split(line, ":")[1])
				}
				if strings.HasPrefix(line, "cpu cores") {
					if cores, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
						cpuInfo.Cores = cores
					}
				}
				if strings.HasPrefix(line, "siblings") {
					if threads, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
						cpuInfo.Threads = threads
					}
				}
				if strings.HasPrefix(line, "cpu MHz") {
					if freq, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(line, ":")[1]), 64); err == nil {
						cpuInfo.Frequency = freq / 1000.0 // Convert MHz to GHz
					}
				}
			}
		}

		// Fallback to lscpu
		cmd := exec.CommandContext(ctx, "lscpu")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			output := stdout.String()
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, "Model name:") {
					cpuInfo.ModelName = strings.TrimSpace(strings.Split(line, ":")[1])
				}
				if strings.Contains(line, "Vendor ID:") {
					cpuInfo.Vendor = strings.TrimSpace(strings.Split(line, ":")[1])
				}
				if strings.Contains(line, "CPU(s):") {
					if threads, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
						cpuInfo.Threads = threads
					}
				}
				if strings.Contains(line, "Core(s) per socket:") {
					if cores, err := strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1])); err == nil {
						cpuInfo.Cores = cores
					}
				}
				if strings.Contains(line, "CPU MHz:") {
					if freq, err := strconv.ParseFloat(strings.TrimSpace(strings.Split(line, ":")[1]), 64); err == nil {
						cpuInfo.Frequency = freq / 1000.0 // Convert MHz to GHz
					}
				}
			}
		}
	}

	// Set defaults if not found
	if cpuInfo.Threads == 0 {
		cpuInfo.Threads = 1
	}
	if cpuInfo.Cores == 0 {
		cpuInfo.Cores = cpuInfo.Threads
	}

	return cpuInfo, nil
}

// getMemoryInfo gathers memory information
func getMemoryInfo() (*MemoryInfo, error) {
	memInfo := &MemoryInfo{}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		// Windows memory info using PowerShell
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-WmiObject -Class Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory | ConvertTo-Json")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			var result map[string]interface{}
			if json.Unmarshal(stdout.Bytes(), &result) == nil {
				if total, ok := result["TotalVisibleMemorySize"].(float64); ok {
					memInfo.Total = uint64(total * 1024) // Convert KB to bytes
				}
				if free, ok := result["FreePhysicalMemory"].(float64); ok {
					memInfo.Free = uint64(free * 1024) // Convert KB to bytes
				}
			}
		}
	} else {
		// Unix/Linux memory info
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					if total, err := strconv.ParseUint(strings.Fields(line)[1], 10, 64); err == nil {
						memInfo.Total = total * 1024 // Convert KB to bytes
					}
				}
				if strings.HasPrefix(line, "MemAvailable:") {
					if available, err := strconv.ParseUint(strings.Fields(line)[1], 10, 64); err == nil {
						memInfo.Available = available * 1024 // Convert KB to bytes
					}
				}
				if strings.HasPrefix(line, "MemFree:") {
					if free, err := strconv.ParseUint(strings.Fields(line)[1], 10, 64); err == nil {
						memInfo.Free = free * 1024 // Convert KB to bytes
					}
				}
			}
		}

		// Fallback to free command
		cmd := exec.CommandContext(ctx, "free", "-b")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			output := stdout.String()
			lines := strings.Split(output, "\n")
			if len(lines) >= 2 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 3 {
					if total, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						memInfo.Total = total
					}
					if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
						memInfo.Used = used
					}
					if free, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
						memInfo.Free = free
					}
				}
			}
		}
	}

	// Calculate derived values
	if memInfo.Total > 0 {
		memInfo.Used = memInfo.Total - memInfo.Free
		if memInfo.Available == 0 {
			memInfo.Available = memInfo.Free
		}
		memInfo.UsagePercent = float64(memInfo.Used) / float64(memInfo.Total) * 100
	}

	return memInfo, nil
}

// getStorageInfo gathers storage/disk information
func getStorageInfo() ([]StorageInfo, error) {
	var storageInfo []StorageInfo

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		// Windows storage info using PowerShell
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-WmiObject -Class Win32_LogicalDisk | Select-Object DeviceID, DriveType, Size, FreeSpace, VolumeName | ConvertTo-Json")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			var results []map[string]interface{}
			if json.Unmarshal(stdout.Bytes(), &results) == nil {
				for _, result := range results {
					if device, ok := result["DeviceID"].(string); ok {
						storage := StorageInfo{
							Device:     device,
							Mountpoint: device,
							FSType:     "NTFS", // Default assumption
						}
						if size, ok := result["Size"].(float64); ok {
							storage.Total = uint64(size)
						}
						if free, ok := result["FreeSpace"].(float64); ok {
							storage.Free = uint64(free)
						}
						if storage.Total > 0 {
							storage.Used = storage.Total - storage.Free
							storage.UsagePercent = float64(storage.Used) / float64(storage.Total) * 100
						}
						storageInfo = append(storageInfo, storage)
					}
				}
			}
		}
	} else {
		// Unix/Linux storage info
		cmd := exec.CommandContext(ctx, "df", "-B1")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			output := stdout.String()
			lines := strings.Split(output, "\n")
			for i, line := range lines {
				if i == 0 || len(line) == 0 {
					continue // Skip header and empty lines
				}
				fields := strings.Fields(line)
				if len(fields) >= 6 {
					storage := StorageInfo{
						Device:     fields[0],
						Mountpoint: fields[5],
					}
					if total, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
						storage.Total = total
					}
					if used, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
						storage.Used = used
					}
					if free, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
						storage.Free = free
					}
					if storage.Total > 0 {
						storage.UsagePercent = float64(storage.Used) / float64(storage.Total) * 100
					}
					storageInfo = append(storageInfo, storage)
				}
			}
		}
	}

	return storageInfo, nil
}

// getDisplayInfo gathers display information (basic implementation)
func getDisplayInfo() ([]DisplayInfo, error) {
	// This is a basic implementation - in a real scenario, you might want to use
	// platform-specific APIs or libraries like xrandr (Linux) or DirectX (Windows)
	return []DisplayInfo{}, nil
}

// getNetworkCardsInfo gathers network card information
func getNetworkCardsInfo() ([]NetworkCardInfo, error) {
	var networkCards []NetworkCardInfo

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if runtime.GOOS == "windows" {
		// Windows network info using PowerShell
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Get-NetAdapter | Select-Object Name, InterfaceDescription, MacAddress, LinkSpeed | ConvertTo-Json")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			var results []map[string]interface{}
			if json.Unmarshal(stdout.Bytes(), &results) == nil {
				for _, result := range results {
					if name, ok := result["Name"].(string); ok {
						card := NetworkCardInfo{
							Name: name,
						}
						if desc, ok := result["InterfaceDescription"].(string); ok {
							card.Type = desc
						}
						if mac, ok := result["MacAddress"].(string); ok {
							card.MAC = mac
						}
						if speed, ok := result["LinkSpeed"].(float64); ok {
							card.Speed = int(speed / 1000000) // Convert to Mbps
						}
						networkCards = append(networkCards, card)
					}
				}
			}
		}
	} else {
		// Unix/Linux network info
		cmd := exec.CommandContext(ctx, "ip", "addr", "show")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err == nil {
			output := stdout.String()
			lines := strings.Split(output, "\n")
			var currentCard *NetworkCardInfo

			for _, line := range lines {
				if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
					// Interface details
					if currentCard != nil {
						if strings.Contains(line, "link/ether") {
							fields := strings.Fields(line)
							if len(fields) >= 2 {
								currentCard.MAC = fields[1]
							}
						}
						if strings.Contains(line, "inet ") {
							fields := strings.Fields(line)
							if len(fields) >= 2 {
								ip := strings.Split(fields[1], "/")[0]
								if currentCard.IPv4 == "" {
									currentCard.IPv4 = ip
								}
							}
						}
					}
				} else {
					// New interface
					if currentCard != nil {
						networkCards = append(networkCards, *currentCard)
					}
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						currentCard = &NetworkCardInfo{
							Name:   strings.TrimSuffix(fields[1], ":"),
							Status: "up",
						}
					}
				}
			}
			if currentCard != nil {
				networkCards = append(networkCards, *currentCard)
			}
		}
	}

	return networkCards, nil
}

// getEnvironmentInfo gathers environment information
func getEnvironmentInfo() (*EnvironmentInfo, error) {
	envInfo := &EnvironmentInfo{}

	// Working directory
	if wd, err := os.Getwd(); err == nil {
		envInfo.WorkingDir = wd
	}

	// Home directory
	if home, ok := os.LookupEnv("HOME"); ok {
		envInfo.HomeDir = home
	} else if home, ok := os.LookupEnv("USERPROFILE"); ok {
		envInfo.HomeDir = home
	}

	// Username
	if user, ok := os.LookupEnv("USER"); ok {
		envInfo.Username = user
	} else if user, ok := os.LookupEnv("USERNAME"); ok {
		envInfo.Username = user
	}

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		envInfo.Hostname = hostname
	}

	// Domain
	if domain, ok := os.LookupEnv("USERDOMAIN"); ok {
		envInfo.Domain = domain
	}

	// PATH
	if path, ok := os.LookupEnv("PATH"); ok {
		envInfo.Path = strings.Split(path, string(os.PathListSeparator))
	}

	// Repo path from environment
	if repoPath, ok := os.LookupEnv("REPO_PATH"); ok {
		envInfo.RepoPath = repoPath
	}

	// All environment variables (filtered for security)
	envInfo.EnvVars = make(map[string]string)
	for _, env := range os.Environ() {
		if key, value, found := strings.Cut(env, "="); found {
			// Filter out sensitive environment variables
			if !isSensitiveEnvVar(key) {
				envInfo.EnvVars[key] = value
			}
		}
	}

	return envInfo, nil
}

// isSensitiveEnvVar checks if an environment variable might contain sensitive information
func isSensitiveEnvVar(key string) bool {
	sensitiveVars := []string{
		"PASSWORD", "TOKEN", "SECRET", "KEY", "AUTH",
		"CREDENTIAL", "PRIVATE", "CERT", "API_",
		"SESSION", "COOKIE", "CSRF", "JWT",
	}
	keyUpper := strings.ToUpper(key)
	for _, sensitive := range sensitiveVars {
		if strings.Contains(keyUpper, sensitive) {
			return true
		}
	}
	return false
}