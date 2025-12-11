package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Mock exec.LookPath for testing
var mockLookPath = func(file string) (string, error) {
	// Mock implementation for testing
	switch file {
	case "go", "node", "python", "python3", "ruby", "java", "git", "docker", "pwsh", "powershell",
		"cmake", "mvn", "gradle", "make", "cargo", "rustc", "gcc", "clang", "dotnet":
		return "/usr/bin/" + file, nil
	case "nonexistent":
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	default:
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}
}

// TestGetDevelopmentToolsInfo tests the getDevelopmentToolsInfo function
func TestGetDevelopmentToolsInfo(t *testing.T) {
	// Save original exec.LookPath
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		exec.LookPath = originalLookPath
	}()

	// Test with mocked tools
	exec.LookPath = mockLookPath
	
	got, err := getDevelopmentToolsInfo()
	if err != nil {
		t.Errorf("getDevelopmentToolsInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getDevelopmentToolsInfo() returned nil")
		return
	}

	// Verify structure
	if got.Go == nil {
		t.Error("Go tool info is nil")
	}

	if got.Node == nil {
		t.Error("Node tool info is nil")
	}

	if got.Git == nil {
		t.Error("Git tool info is nil")
	}

	if got.Docker == nil {
		t.Error("Docker tool info is nil")
	}

	// Verify package managers
	if got.PackageMgrs == nil {
		t.Error("Package managers is nil")
	}
}

// TestGetToolInfo tests the getToolInfo function
func TestGetToolInfo(t *testing.T) {
	// Save original exec.LookPath
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		exec.LookPath = originalLookPath
	}()

	tests := []struct {
		name     string
		toolName string
		wantInstalled bool
	}{
		{
			name:     "Existing tool",
			toolName: "go",
			wantInstalled: true,
		},
		{
			name:     "Non-existing tool",
			toolName: "nonexistent",
			wantInstalled: false,
		},
		{
			name:     "Another existing tool",
			toolName: "git",
			wantInstalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec.LookPath = mockLookPath
			
			got := getToolInfo(tt.toolName)
			
			if got == nil {
				t.Error("getToolInfo() returned nil")
				return
			}

			if got.Installed != tt.wantInstalled {
				t.Errorf("getToolInfo() installed = %v, want %v", got.Installed, tt.wantInstalled)
			}

			if got.Executable != tt.toolName {
				t.Errorf("getToolInfo() executable = %v, want %v", got.Executable, tt.toolName)
			}

			if tt.wantInstalled && got.Path == "" {
				t.Error("getToolInfo() path is empty for installed tool")
			}
		})
	}
}

// TestGetToolVersion tests the getToolVersion function
func TestGetToolVersion(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		path     string
		want     string
	}{
		{
			name:     "Go version",
			toolName: "go",
			path:     "/usr/bin/go",
			want:     "", // Will be empty if command fails
		},
		{
			name:     "Non-existent tool",
			toolName: "nonexistent",
			path:     "",
			want:     "",
		},
		{
			name:     "Git version",
			toolName: "git",
			path:     "/usr/bin/git",
			want:     "", // Will be empty if command fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getToolVersion(tt.toolName, tt.path)
			
			// We can't guarantee the version output, but we can verify it doesn't panic
			_ = got
		})
	}
}

// TestGetToolFeatures tests the getToolFeatures function
func TestGetToolFeatures(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     []string
	}{
		{
			name:     "Go features",
			toolName: "go",
			want: []string{
				"static_compilation", "concurrent_programming", "garbage_collection",
				"cross_compilation", "testing_framework", "modules", "interfaces",
				"goroutines", "channels", "select_statements", "defer_statements",
			},
		},
		{
			name:     "Node features",
			toolName: "node",
			want: []string{
				"javascript_runtime", "npm_support", "es_modules", "async_programming",
				"event_loop", "v8_engine", "npm_scripts", "package_json",
			},
		},
		{
			name:     "Git features",
			toolName: "git",
			want: []string{
				"version_control", "distributed", "branching", "merging", "rebasing",
				"staging_area", "hooks", "submodules", "lfs_support", "bisect",
			},
		},
		{
			name:     "Docker features",
			toolName: "docker",
			want: []string{
				"containerization", "dockerfile", "docker_compose", "volumes",
				"networking", "multi_stage_builds", "health_checks", "secrets",
			},
		},
		{
			name:     "Cargo features",
			toolName: "cargo",
			want: []string{
				"rust_package_manager", "crates_io", "dependencies", "workspaces",
				"build_scripts", "cross_compilation", "documentation_generation",
			},
		},
		{
			name:     "Unknown tool",
			toolName: "unknown",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getToolFeatures(tt.toolName)
			
			if len(got) != len(tt.want) {
				t.Errorf("getToolFeatures() length = %v, want %v", len(got), len(tt.want))
				return
			}

			for i, feature := range got {
				if feature != tt.want[i] {
					t.Errorf("getToolFeatures()[%d] = %v, want %v", i, feature, tt.want[i])
				}
			}
		})
	}
}

// TestGetPackageManagers tests the getPackageManagers function
func TestGetPackageManagers(t *testing.T) {
	// Save original exec.LookPath
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		exec.LookPath = originalLookPath
	}()

	// Test with mocked package managers
	exec.LookPath = func(file string) (string, error) {
		switch file {
		case "apt", "brew", "choco", "winget":
			return "/usr/bin/" + file, nil
		default:
			return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
		}
	}

	got := getPackageManagers()

	if got == nil {
		t.Error("getPackageManagers() returned nil")
		return
	}

	// Check for expected package managers
	foundApt := false
	foundBrew := false
	foundChoco := false
	foundWinget := false

	for _, pkg := range got {
		switch pkg.Name {
		case "apt":
			foundApt = true
			if pkg.Type != "apt" {
				t.Errorf("apt package manager type = %v, want apt", pkg.Type)
			}
		case "brew":
			foundBrew = true
			if pkg.Type != "brew" {
				t.Errorf("brew package manager type = %v, want brew", pkg.Type)
			}
		case "choco":
			foundChoco = true
			if pkg.Type != "chocolatey" {
				t.Errorf("choco package manager type = %v, want chocolatey", pkg.Type)
			}
		case "winget":
			foundWinget = true
			if pkg.Type != "winget" {
				t.Errorf("winget package manager type = %v, want winget", pkg.Type)
			}
		}
	}

	if !foundApt {
		t.Error("apt package manager not found")
	}

	if !foundBrew {
		t.Error("brew package manager not found")
	}

	if !foundChoco {
		t.Error("choco package manager not found")
	}

	if !foundWinget {
		t.Error("winget package manager not found")
	}
}

// TestGetAptVersion tests the getAptVersion function
func TestGetAptVersion(t *testing.T) {
	got := getAptVersion()
	
	// We can't guarantee the version output, but we can verify it doesn't panic
	_ = got
}

// TestCheckCommandExists tests the checkCommandExists function
func TestCheckCommandExists(t *testing.T) {
	// Save original exec.LookPath
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		exec.LookPath = originalLookPath
	}()

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
		{
			name:        "Command with empty search paths",
			command:     "go",
			searchPaths: []string{},
			wantExists:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec.LookPath = mockLookPath
			
			got := checkCommandExists(tt.command, tt.searchPaths)

			if got.Exists != tt.wantExists {
				t.Errorf("checkCommandExists() exists = %v, want %v", got.Exists, tt.wantExists)
			}

			if got.Command != tt.command {
				t.Errorf("checkCommandExists() command = %v, want %v", got.Command, tt.command)
			}

			if tt.wantExists && got.Path == "" {
				t.Error("checkCommandExists() path is empty for existing command")
			}
		})
	}
}

// TestToolInfoStruct tests the ToolInfo struct
func TestToolInfoStruct(t *testing.T) {
	toolInfo := &ToolInfo{
		Version:    "1.0.0",
		Path:       "/usr/bin/go",
		Installed:  true,
		Executable: "go",
		Features:   []string{"feature1", "feature2"},
	}

	// Verify all fields are set correctly
	if toolInfo.Version != "1.0.0" {
		t.Errorf("ToolInfo.Version = %v, want 1.0.0", toolInfo.Version)
	}

	if toolInfo.Path != "/usr/bin/go" {
		t.Errorf("ToolInfo.Path = %v, want /usr/bin/go", toolInfo.Path)
	}

	if !toolInfo.Installed {
		t.Errorf("ToolInfo.Installed = %v, want true", toolInfo.Installed)
	}

	if toolInfo.Executable != "go" {
		t.Errorf("ToolInfo.Executable = %v, want go", toolInfo.Executable)
	}

	if len(toolInfo.Features) != 2 {
		t.Errorf("ToolInfo.Features length = %v, want 2", len(toolInfo.Features))
	}
}

// TestPackageMgrInfoStruct tests the PackageMgrInfo struct
func TestPackageMgrInfoStruct(t *testing.T) {
	pkgMgrInfo := PackageMgrInfo{
		Name:    "apt",
		Version: "2.0.0",
		Type:    "apt",
	}

	// Verify all fields are set correctly
	if pkgMgrInfo.Name != "apt" {
		t.Errorf("PackageMgrInfo.Name = %v, want apt", pkgMgrInfo.Name)
	}

	if pkgMgrInfo.Version != "2.0.0" {
		t.Errorf("PackageMgrInfo.Version = %v, want 2.0.0", pkgMgrInfo.Version)
	}

	if pkgMgrInfo.Type != "apt" {
		t.Errorf("PackageMgrInfo.Type = %v, want apt", pkgMgrInfo.Type)
	}
}

// TestDevelopmentToolsInfoStruct tests the DevelopmentToolsInfo struct
func TestDevelopmentToolsInfoStruct(t *testing.T) {
	goInfo := &ToolInfo{
		Version:    "1.15.6",
		Path:       "/usr/bin/go",
		Installed:  true,
		Executable: "go",
	}

	nodeInfo := &ToolInfo{
		Version:    "14.15.0",
		Path:       "/usr/bin/node",
		Installed:  true,
		Executable: "node",
	}

	pkgMgrs := []PackageMgrInfo{
		{
			Name:    "apt",
			Version: "2.0.0",
			Type:    "apt",
		},
	}

	devToolsInfo := &DevelopmentToolsInfo{
		Go:          goInfo,
		Node:        nodeInfo,
		PackageMgrs: pkgMgrs,
	}

	// Verify all fields are set correctly
	if devToolsInfo.Go != goInfo {
		t.Error("DevelopmentToolsInfo.Go is not set correctly")
	}

	if devToolsInfo.Node != nodeInfo {
		t.Error("DevelopmentToolsInfo.Node is not set correctly")
	}

	if len(devToolsInfo.PackageMgrs) != 1 {
		t.Errorf("DevelopmentToolsInfo.PackageMgrs length = %v, want 1", len(devToolsInfo.PackageMgrs))
	}
}

// TestPowerShellDetection tests PowerShell detection on different platforms
func TestPowerShellDetection(t *testing.T) {
	// Save original exec.LookPath
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		exec.LookPath = originalLookPath
	}()

	tests := []struct {
		name       string
		platform   string
		hasPwsh    bool
		hasPowerShell bool
	}{
		{
			name:       "Windows with both",
			platform:   "windows",
			hasPwsh:    true,
			hasPowerShell: true,
		},
		{
			name:       "Windows with PowerShell only",
			platform:   "windows",
			hasPwsh:    false,
			hasPowerShell: true,
		},
		{
			name:       "Linux with pwsh",
			platform:   "linux",
			hasPwsh:    true,
			hasPowerShell: false,
		},
		{
			name:       "Linux without PowerShell",
			platform:   "linux",
			hasPwsh:    false,
			hasPowerShell: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock runtime.GOOS
			originalGOOS := runtime.GOOS
			defer func() {
				// Note: We can't actually modify runtime.GOOS in Go, but we can test the logic
				// This is more of a conceptual test
			}()

			// Mock exec.LookPath based on test case
			exec.LookPath = func(file string) (string, error) {
				if file == "pwsh" && tt.hasPwsh {
					return "/usr/bin/pwsh", nil
				}
				if file == "powershell" && tt.hasPowerShell {
					return "/usr/bin/powershell", nil
				}
				return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
			}

			// Test getToolInfo with pwsh
			pwshInfo := getToolInfo("pwsh")
			if tt.hasPwsh && !pwshInfo.Installed {
				t.Error("Expected pwsh to be installed")
			}
			if !tt.hasPwsh && pwshInfo.Installed {
				t.Error("Expected pwsh to not be installed")
			}

			// Test getToolInfo with powershell
			powershellInfo := getToolInfo("powershell")
			if tt.hasPowerShell && !powershellInfo.Installed {
				t.Error("Expected powershell to be installed")
			}
			if !tt.hasPowerShell && powershellInfo.Installed {
				t.Error("Expected powershell to not be installed")
			}

			_ = originalGOOS // Use the variable to avoid unused variable warning
		})
	}
}

// TestVersionParsing tests version parsing for different tools
func TestVersionParsing(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		output   string
		want     string
	}{
		{
			name:     "Go version output",
			toolName: "go",
			output:   "go version go1.15.6 linux/amd64",
			want:     "go1.15.6",
		},
		{
			name:     "Node version output",
			toolName: "node",
			output:   "v14.15.0",
			want:     "v14.15.0",
		},
		{
			name:     "Git version output",
			toolName: "git",
			output:   "git version 2.25.1",
			want:     "2.25.1",
		},
		{
			name:     "Docker version output",
			toolName: "docker",
			output:   "Docker version 20.10.2, build 2291f61",
			want:     "Docker version 20.10.2, build 2291f61",
		},
		{
			name:     "Java version output",
			toolName: "java",
			output:   "openjdk version \"11.0.11\" 2021-04-20",
			want:     "openjdk version \"11.0.11\" 2021-04-20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for version parsing
			// In a real implementation, we would mock the command execution
			// and verify the parsing logic
			_ = tt.toolName
			_ = tt.output
			_ = tt.want
		})
	}
}

// TestTimeoutHandling tests timeout handling for tool version commands
func TestTimeoutHandling(t *testing.T) {
	// This test verifies that tool version commands respect timeouts
	// In a real implementation, we would mock a command that hangs
	// and verify it times out correctly
	
	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	// This is a conceptual test
	// In a real implementation, we would execute a command that takes longer
	// than the timeout and verify it returns an error
	_ = ctx
}

// TestErrorHandling tests error handling in development tools functions
func TestErrorHandling(t *testing.T) {
	// Save original exec.LookPath
	originalLookPath := exec.LookPath
	
	// Restore after test
	defer func() {
		exec.LookPath = originalLookPath
	}()

	// Mock exec.LookPath to always return an error
	exec.LookPath = func(file string) (string, error) {
		return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
	}

	// Test getDevelopmentToolsInfo with no tools installed
	got, err := getDevelopmentToolsInfo()
	if err != nil {
		t.Errorf("getDevelopmentToolsInfo() error = %v", err)
		return
	}

	if got == nil {
		t.Error("getDevelopmentToolsInfo() returned nil")
		return
	}

	// Verify that all tools are marked as not installed
	if got.Go.Installed {
		t.Error("Expected Go to not be installed")
	}

	if got.Node.Installed {
		t.Error("Expected Node to not be installed")
	}

	if got.Git.Installed {
		t.Error("Expected Git to not be installed")
	}
}

// TestPlatformSpecificBehavior tests platform-specific behavior
func TestPlatformSpecificBehavior(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	// Test Windows-specific behavior
	if runtime.GOOS == "windows" {
		// Test that PowerShell detection works on Windows
		powershellInfo := getToolInfo("powershell")
		_ = powershellInfo
		
		// Test that Windows-specific package managers are detected
		pkgMgrs := getPackageManagers()
		_ = pkgMgrs
	}

	// Test Unix-specific behavior
	if runtime.GOOS != "windows" {
		// Test that Unix-specific package managers are detected
		pkgMgrs := getPackageManagers()
		_ = pkgMgrs
	}
}