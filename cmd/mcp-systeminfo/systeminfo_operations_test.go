package main

import (
	"encoding/json"
	"os"
	"runtime"
	"testing"
	"time"
)

// TestGetOSInfo tests the getOSInfo function
func TestGetOSInfo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "Valid OS Info",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getOSInfo()
			if (err != nil) != tt.wantErr {
				t.Errorf("getOSInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil {
				t.Error("getOSInfo() returned nil")
				return
			}

			// Verify basic fields
			if got.Name == "" {
				t.Error("OS name is empty")
			}

			if got.Architecture == "" {
				t.Error("OS architecture is empty")
			}

			if got.Platform == "" {
				t.Error("OS platform is empty")
			}

			// Verify architecture matches runtime
			if got.Architecture != runtime.GOARCH {
				t.Errorf("Architecture mismatch: got %s, want %s", got.Architecture, runtime.GOARCH)
			}
		})
	}
}

// TestGetHardwareInfo tests the getHardwareInfo function
func TestGetHardwareInfo(t *testing.T) {
	got, err := getHardwareInfo()
	if err != nil {
		t.Errorf("getHardwareInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getHardwareInfo() returned nil")
		return
	}

	// Verify CPU info
	if got.CPU.Architecture == "" {
		t.Error("CPU architecture is empty")
	}

	if got.CPU.Cores == 0 {
		t.Error("CPU cores is 0")
	}

	if got.CPU.Threads == 0 {
		t.Error("CPU threads is 0")
	}

	// Verify memory info
	if got.Memory.Total == 0 {
		t.Error("Memory total is 0")
	}

	// Verify storage info
	if len(got.Storage) == 0 {
		t.Error("No storage devices found")
	}
}

// TestGetEnvironmentInfo tests the getEnvironmentInfo function
func TestGetEnvironmentInfo(t *testing.T) {
	got, err := getEnvironmentInfo()
	if err != nil {
		t.Errorf("getEnvironmentInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getEnvironmentInfo() returned nil")
		return
	}

	// Verify working directory
	if got.WorkingDir == "" {
		t.Error("Working directory is empty")
	}

	// Verify username
	if got.Username == "" {
		t.Error("Username is empty")
	}

	// Verify hostname
	if got.Hostname == "" {
		t.Error("Hostname is empty")
	}

	// Verify PATH is not empty
	if len(got.Path) == 0 {
		t.Error("PATH is empty")
	}

	// Verify environment variables
	if len(got.EnvVars) == 0 {
		t.Error("No environment variables found")
	}
}

// TestGetShellInfo tests the getShellInfo function
func TestGetShellInfo(t *testing.T) {
	got, err := getShellInfo()
	if err != nil {
		t.Errorf("getShellInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getShellInfo() returned nil")
		return
	}

	// Verify shell name
	if got.Name == "" {
		t.Error("Shell name is empty")
	}

	// Verify shell type
	if got.Type == "" {
		t.Error("Shell type is empty")
	}

	// Verify shell path
	if got.Path == "" {
		t.Error("Shell path is empty")
	}

	// Verify features
	if len(got.Features) == 0 {
		t.Error("No shell features found")
	}
}

// TestGetDevelopmentToolsInfo tests the getDevelopmentToolsInfo function
func TestGetDevelopmentToolsInfo(t *testing.T) {
	got, err := getDevelopmentToolsInfo()
	if err != nil {
		t.Errorf("getDevelopmentToolsInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getDevelopmentToolsInfo() returned nil")
		return
	}

	// Verify at least some tools are checked
	// We can't guarantee any specific tool is installed, but we can verify the structure
	if got.Go == nil {
		t.Error("Go tool info is nil")
	}

	if got.Node == nil {
		t.Error("Node tool info is nil")
	}

	if got.Git == nil {
		t.Error("Git tool info is nil")
	}
}

// TestGetNetworkInfo tests the getNetworkInfo function
func TestGetNetworkInfo(t *testing.T) {
	got, err := getNetworkInfo()
	if err != nil {
		t.Errorf("getNetworkInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getNetworkInfo() returned nil")
		return
	}

	// Verify hostname
	if got.Hostname == "" {
		t.Error("Network hostname is empty")
	}

	// Verify connected status
	if !got.Connected {
		t.Error("Network is not connected")
	}
}

// TestDetectRepositories tests the detectRepositories function
func TestDetectRepositories(t *testing.T) {
	got, err := detectRepositories()
	if err != nil {
		t.Errorf("detectRepositories() error = %v", err)
		return
	}

	// We can't guarantee any repositories exist, but we can verify the function returns without error
	if got == nil {
		t.Error("detectRepositories() returned nil")
	}
}

// TestCheckCommandExists tests the checkCommandExists function
func TestCheckCommandExists(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		searchPaths []string
		wantExists  bool
	}{
		{
			name:       "Existing command",
			command:    "go",
			wantExists: true,
		},
		{
			name:       "Non-existing command",
			command:    "nonexistentcommand12345",
			wantExists: false,
		},
		{
			name:        "Command with search paths",
			command:     "go",
			searchPaths: []string{"/usr/bin", "/bin"},
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkCommandExists(tt.command, tt.searchPaths)

			if got.Exists != tt.wantExists {
				t.Errorf("checkCommandExists() exists = %v, want %v", got.Exists, tt.wantExists)
			}

			if got.Command != tt.command {
				t.Errorf("checkCommandExists() command = %v, want %v", got.Command, tt.command)
			}
		})
	}
}

// TestParseVersion tests the parseVersion function
func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		wantOS   OSInfo
	}{
		{
			name:    "Simple version",
			version: "1.2.3",
			wantOS: OSInfo{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
		},
		{
			name:    "Major.Minor only",
			version: "10.5",
			wantOS: OSInfo{
				Major: 10,
				Minor: 5,
			},
		},
		{
			name:    "Major only",
			version: "7",
			wantOS: OSInfo{
				Major: 7,
			},
		},
		{
			name:    "Invalid version",
			version: "invalid.version.string",
			wantOS:  OSInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osInfo := &OSInfo{Version: tt.version}
			parseVersion(osInfo)

			if osInfo.Major != tt.wantOS.Major {
				t.Errorf("parseVersion() major = %v, want %v", osInfo.Major, tt.wantOS.Major)
			}

			if osInfo.Minor != tt.wantOS.Minor {
				t.Errorf("parseVersion() minor = %v, want %v", osInfo.Minor, tt.wantOS.Minor)
			}

			if osInfo.Patch != tt.wantOS.Patch {
				t.Errorf("parseVersion() patch = %v, want %v", osInfo.Patch, tt.wantOS.Patch)
			}
		})
	}
}

// TestIsSensitiveEnvVar tests the isSensitiveEnvVar function
func TestIsSensitiveEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantBool bool
	}{
		{
			name:     "Password variable",
			key:      "DB_PASSWORD",
			wantBool: true,
		},
		{
			name:     "Token variable",
			key:      "API_TOKEN",
			wantBool: true,
		},
		{
			name:     "Secret variable",
			key:      "SECRET_KEY",
			wantBool: true,
		},
		{
			name:     "Normal variable",
			key:      "PATH",
			wantBool: false,
		},
		{
			name:     "Home variable",
			key:      "HOME",
			wantBool: false,
		},
		{
			name:     "Auth variable",
			key:      "AUTHORIZATION",
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitiveEnvVar(tt.key)
			if got != tt.wantBool {
				t.Errorf("isSensitiveEnvVar() = %v, want %v", got, tt.wantBool)
			}
		})
	}
}

// TestToolGetSystemInfo tests the toolGetSystemInfo function
func TestToolGetSystemInfo(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetSystemInfo(args)
	if err != nil {
		t.Errorf("toolGetSystemInfo() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetSystemInfo() returned empty string")
		return
	}

	// Verify it's valid JSON
	var systemInfo SystemInfo
	if err := json.Unmarshal([]byte(got), &systemInfo); err != nil {
		t.Errorf("toolGetSystemInfo() returned invalid JSON: %v", err)
	}

	// Verify timestamp is recent
	if time.Since(systemInfo.Timestamp) > time.Minute {
		t.Error("toolGetSystemInfo() timestamp is not recent")
	}
}

// TestToolGetOSInfo tests the toolGetOSInfo function
func TestToolGetOSInfo(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetOSInfo(args)
	if err != nil {
		t.Errorf("toolGetOSInfo() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetOSInfo() returned empty string")
		return
	}

	// Verify it's valid JSON
	var osInfo OSInfo
	if err := json.Unmarshal([]byte(got), &osInfo); err != nil {
		t.Errorf("toolGetOSInfo() returned invalid JSON: %v", err)
	}
}

// TestToolGetHardwareInfo tests the toolGetHardwareInfo function
func TestToolGetHardwareInfo(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetHardwareInfo(args)
	if err != nil {
		t.Errorf("toolGetHardwareInfo() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetHardwareInfo() returned empty string")
		return
	}

	// Verify it's valid JSON
	var hardwareInfo HardwareInfo
	if err := json.Unmarshal([]byte(got), &hardwareInfo); err != nil {
		t.Errorf("toolGetHardwareInfo() returned invalid JSON: %v", err)
	}
}

// TestToolGetEnvironmentInfo tests the toolGetEnvironmentInfo function
func TestToolGetEnvironmentInfo(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetEnvironmentInfo(args)
	if err != nil {
		t.Errorf("toolGetEnvironmentInfo() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetEnvironmentInfo() returned empty string")
		return
	}

	// Verify it's valid JSON
	var envInfo EnvironmentInfo
	if err := json.Unmarshal([]byte(got), &envInfo); err != nil {
		t.Errorf("toolGetEnvironmentInfo() returned invalid JSON: %v", err)
	}
}

// TestToolGetShellInfo tests the toolGetShellInfo function
func TestToolGetShellInfo(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetShellInfo(args)
	if err != nil {
		t.Errorf("toolGetShellInfo() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetShellInfo() returned empty string")
		return
	}

	// Verify it's valid JSON
	var shellInfo ShellInfo
	if err := json.Unmarshal([]byte(got), &shellInfo); err != nil {
		t.Errorf("toolGetShellInfo() returned invalid JSON: %v", err)
	}
}

// TestToolGetDevelopmentTools tests the toolGetDevelopmentTools function
func TestToolGetDevelopmentTools(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetDevelopmentTools(args)
	if err != nil {
		t.Errorf("toolGetDevelopmentTools() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetDevelopmentTools() returned empty string")
		return
	}

	// Verify it's valid JSON
	var devToolsInfo DevelopmentToolsInfo
	if err := json.Unmarshal([]byte(got), &devToolsInfo); err != nil {
		t.Errorf("toolGetDevelopmentTools() returned invalid JSON: %v", err)
	}
}

// TestToolGetNetworkInfo tests the toolGetNetworkInfo function
func TestToolGetNetworkInfo(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetNetworkInfo(args)
	if err != nil {
		t.Errorf("toolGetNetworkInfo() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetNetworkInfo() returned empty string")
		return
	}

	// Verify it's valid JSON
	var networkInfo NetworkInfo
	if err := json.Unmarshal([]byte(got), &networkInfo); err != nil {
		t.Errorf("toolGetNetworkInfo() returned invalid JSON: %v", err)
	}
}

// TestToolDetectRepositories tests the toolDetectRepositories function
func TestToolDetectRepositories(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolDetectRepositories(args)
	if err != nil {
		t.Errorf("toolDetectRepositories() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolDetectRepositories() returned empty string")
		return
	}

	// Verify it's valid JSON
	var reposInfo []RepositoryInfo
	if err := json.Unmarshal([]byte(got), &reposInfo); err != nil {
		t.Errorf("toolDetectRepositories() returned invalid JSON: %v", err)
	}
}

// TestToolCheckCommand tests the toolCheckCommand function
func TestToolCheckCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid command",
			args: map[string]interface{}{
				"command": "go",
			},
			wantErr: false,
		},
		{
			name: "Missing command",
			args: map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "Invalid command format",
			args: map[string]interface{}{
				"command": "invalid command with spaces",
			},
			wantErr: true,
		},
		{
			name: "Command with search paths",
			args: map[string]interface{}{
				"command":       "go",
				"search_paths":  []interface{}{"/usr/bin", "/bin"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toolCheckCommand(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("toolCheckCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == "" {
				t.Error("toolCheckCommand() returned empty string")
			}
		})
	}
}

// TestToolGetRecommendations tests the toolGetRecommendations function
func TestToolGetRecommendations(t *testing.T) {
	args := map[string]interface{}{}
	got, err := toolGetRecommendations(args)
	if err != nil {
		t.Errorf("toolGetRecommendations() error = %v", err)
		return
	}

	if got == "" {
		t.Error("toolGetRecommendations() returned empty string")
		return
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(got), &result); err != nil {
		t.Errorf("toolGetRecommendations() returned invalid JSON: %v", err)
	}

	// Check for recommendations field
	if _, ok := result["recommendations"]; !ok {
		t.Error("toolGetRecommendations() result missing recommendations field")
	}
}

// TestGetSystemRecommendations tests the getSystemRecommendations function
func TestGetSystemRecommendations(t *testing.T) {
	osInfo := &OSInfo{
		Name:         "Linux",
		Version:      "5.4.0",
		Architecture: "x86_64",
		Platform:     "Linux",
		Distribution: "ubuntu",
	}

	hardwareInfo := &HardwareInfo{
		CPU: CPUInfo{
			Cores:   4,
			Threads: 8,
		},
		Memory: MemoryInfo{
			Total:       8589934592, // 8GB
			UsagePercent: 50.0,
		},
		Storage: []StorageInfo{
			{
				Device:      "/dev/sda1",
				Mountpoint:  "/",
				Total:       107374182400, // 100GB
				UsagePercent: 50.0,
			},
		},
	}

	devToolsInfo := &DevelopmentToolsInfo{
		Git: &ToolInfo{
			Installed:  true,
			Version:    "2.25.1",
			Executable: "git",
		},
		Go: &ToolInfo{
			Installed:  true,
			Version:    "1.15.6",
			Executable: "go",
		},
	}

	got := getSystemRecommendations(osInfo, hardwareInfo, devToolsInfo)

	if len(got) == 0 {
		t.Error("getSystemRecommendations() returned no recommendations")
	}

	// Check for expected recommendations
	found := false
	for _, rec := range got {
		if rec == "System appears well-configured for development" {
			found = true
			break
		}
	}
	if !found {
		t.Error("getSystemRecommendations() missing expected recommendation")
	}
}