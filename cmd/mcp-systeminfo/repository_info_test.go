package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Mock os.Getwd for testing
var mockGetwd = func() (string, error) {
	return "/home/user/project", nil
}

// Mock os.Stat for testing
var mockStat = func(name string) (os.FileInfo, error) {
	// Mock implementation for testing
	switch name {
	case "/home/user/project/.git":
		return &mockFileInfo{name: ".git", isDir: true}, nil
	case "/home/user/.git":
		return &mockFileInfo{name: ".git", isDir: true}, nil
	case "/home/user/project/.svn":
		return &mockFileInfo{name: ".svn", isDir: true}, nil
	case "/home/user/project/.hg":
		return &mockFileInfo{name: ".hg", isDir: true}, nil
	case "/home/user/project/nonexistent":
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	default:
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	}
}

// Mock os.ReadFile for testing
var mockReadFile = func(filename string) ([]byte, error) {
	// Mock implementation for testing
	switch filename {
	case "/etc/resolv.conf":
		return []byte("nameserver 8.8.8.8\nnameserver 8.8.4.4\n"), nil
	default:
		return nil, &os.PathError{Op: "open", Path: filename, Err: os.ErrNotExist}
	}
}

// Mock os.Getenv for testing
var mockGetenv = func(key string) string {
	switch key {
	case "REPO_PATH":
		return "/home/user/other-project"
	default:
		return ""
	}
}

// Mock file info implementation
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0755 }
func (m *mockFileInfo) ModTime() time.Time { return time.Now() }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// Mock filepath.Abs for testing
var mockAbs = func(path string) (string, error) {
	if strings.HasPrefix(path, "/") {
		return path, nil
	}
	return "/home/user/" + path, nil
}

// Mock filepath.Dir for testing
var mockDir = func(path string) string {
	if path == "/home/user/project" {
		return "/home/user"
	}
	if path == "/home/user" {
		return "/home"
	}
	if path == "/home" {
		return "/"
	}
	return "/"
}

// TestDetectRepositories tests the detectRepositories function
func TestDetectRepositories(t *testing.T) {
	// Save original functions
	originalGetwd := os.Getwd
	originalStat := os.Stat
	originalGetenv := os.Getenv
	originalAbs := filepath.Abs
	originalDir := filepath.Dir
	
	// Restore after test
	defer func() {
		os.Getwd = originalGetwd
		os.Stat = originalStat
		os.Getenv = originalGetenv
		filepath.Abs = originalAbs
		filepath.Dir = originalDir
	}()

	// Set mock functions
	os.Getwd = mockGetwd
	os.Stat = mockStat
	os.Getenv = mockGetenv
	filepath.Abs = mockAbs
	filepath.Dir = mockDir

	got, err := detectRepositories()
	if err != nil {
		t.Errorf("detectRepositories() error = %v", err)
		return
	}

	if got == nil {
		t.Error("detectRepositories() returned nil")
		return
	}

	// Verify at least one repository was found (mocked .git directory)
	if len(got) == 0 {
		t.Error("detectRepositories() found no repositories")
	}
}

// TestScanDirectoryForRepos tests the scanDirectoryForRepos function
func TestScanDirectoryForRepos(t *testing.T) {
	// Save original functions
	originalStat := os.Stat
	originalDir := filepath.Dir
	
	// Restore after test
	defer func() {
		os.Stat = originalStat
		filepath.Dir = originalDir
	}()

	tests := []struct {
		name           string
		dir            string
		mockStat       func(name string) (os.FileInfo, error)
		mockDir        func(path string) string
		wantReposCount int
	}{
		{
			name: "Directory with Git repo",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.git" {
					return &mockFileInfo{name: ".git", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			mockDir: func(path string) string {
				if path == "/home/user/project" {
					return "/home/user"
				}
				return "/"
			},
			wantReposCount: 1,
		},
		{
			name: "Directory with parent Git repo",
			dir:  "/home/user/project/subdir",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/subdir/.git" {
					return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
				}
				if name == "/home/user/project/.git" {
					return &mockFileInfo{name: ".git", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			mockDir: func(path string) string {
				if path == "/home/user/project/subdir" {
					return "/home/user/project"
				}
				if path == "/home/user/project" {
					return "/home/user"
				}
				return "/"
			},
			wantReposCount: 1,
		},
		{
			name: "Directory with no repos",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			mockDir: func(path string) string {
				return "/"
			},
			wantReposCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Stat = tt.mockStat
			filepath.Dir = tt.mockDir
			
			got := scanDirectoryForRepos(tt.dir)
			
			if len(got) != tt.wantReposCount {
				t.Errorf("scanDirectoryForRepos() count = %v, want %v", len(got), tt.wantReposCount)
			}
		})
	}
}

// TestCheckDirectoryRepo tests the checkDirectoryRepo function
func TestCheckDirectoryRepo(t *testing.T) {
	// Save original functions
	originalStat := os.Stat
	
	// Restore after test
	defer func() {
		os.Stat = originalStat
	}()

	tests := []struct {
		name     string
		dir      string
		mockStat func(name string) (os.FileInfo, error)
		wantType string
	}{
		{
			name: "Git repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.git" {
					return &mockFileInfo{name: ".git", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			wantType: "git",
		},
		{
			name: "SVN repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.svn" {
					return &mockFileInfo{name: ".svn", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			wantType: "svn",
		},
		{
			name: "Mercurial repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.hg" {
					return &mockFileInfo{name: ".hg", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			wantType: "hg",
		},
		{
			name: "No repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			wantType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Stat = tt.mockStat
			
			got := checkDirectoryRepo(tt.dir)
			
			if tt.wantType == "" {
				if got != nil {
					t.Errorf("checkDirectoryRepo() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("checkDirectoryRepo() = nil, want repository of type %s", tt.wantType)
				} else if got.Type != tt.wantType {
					t.Errorf("checkDirectoryRepo() type = %v, want %v", got.Type, tt.wantType)
				}
			}
		})
	}
}

// TestCheckGitRepo tests the checkGitRepo function
func TestCheckGitRepo(t *testing.T) {
	// Save original functions
	originalStat := os.Stat
	
	// Restore after test
	defer func() {
		os.Stat = originalStat
	}()

	tests := []struct {
		name     string
		dir      string
		mockStat func(name string) (os.FileInfo, error)
		want     *RepositoryInfo
	}{
		{
			name: "Valid Git repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.git" {
					return &mockFileInfo{name: ".git", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			want: &RepositoryInfo{
				Path: "/home/user/project",
				Type: "git",
			},
		},
		{
			name: "Not a Git repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Stat = tt.mockStat
			
			got := checkGitRepo(tt.dir)
			
			if tt.want == nil {
				if got != nil {
					t.Errorf("checkGitRepo() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("checkGitRepo() = nil, want RepositoryInfo")
				} else {
					if got.Path != tt.want.Path {
						t.Errorf("checkGitRepo() path = %v, want %v", got.Path, tt.want.Path)
					}
					if got.Type != tt.want.Type {
						t.Errorf("checkGitRepo() type = %v, want %v", got.Type, tt.want.Type)
					}
				}
			}
		})
	}
}

// TestCheckSVNRepo tests the checkSVNRepo function
func TestCheckSVNRepo(t *testing.T) {
	// Save original functions
	originalStat := os.Stat
	
	// Restore after test
	defer func() {
		os.Stat = originalStat
	}()

	tests := []struct {
		name     string
		dir      string
		mockStat func(name string) (os.FileInfo, error)
		want     *RepositoryInfo
	}{
		{
			name: "Valid SVN repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.svn" {
					return &mockFileInfo{name: ".svn", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			want: &RepositoryInfo{
				Path: "/home/user/project",
				Type: "svn",
			},
		},
		{
			name: "Not an SVN repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Stat = tt.mockStat
			
			got := checkSVNRepo(tt.dir)
			
			if tt.want == nil {
				if got != nil {
					t.Errorf("checkSVNRepo() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("checkSVNRepo() = nil, want RepositoryInfo")
				} else {
					if got.Path != tt.want.Path {
						t.Errorf("checkSVNRepo() path = %v, want %v", got.Path, tt.want.Path)
					}
					if got.Type != tt.want.Type {
						t.Errorf("checkSVNRepo() type = %v, want %v", got.Type, tt.want.Type)
					}
				}
			}
		})
	}
}

// TestCheckMercurialRepo tests the checkMercurialRepo function
func TestCheckMercurialRepo(t *testing.T) {
	// Save original functions
	originalStat := os.Stat
	
	// Restore after test
	defer func() {
		os.Stat = originalStat
	}()

	tests := []struct {
		name     string
		dir      string
		mockStat func(name string) (os.FileInfo, error)
		want     *RepositoryInfo
	}{
		{
			name: "Valid Mercurial repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				if name == "/home/user/project/.hg" {
					return &mockFileInfo{name: ".hg", isDir: true}, nil
				}
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			want: &RepositoryInfo{
				Path: "/home/user/project",
				Type: "hg",
			},
		},
		{
			name: "Not a Mercurial repository",
			dir:  "/home/user/project",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Stat = tt.mockStat
			
			got := checkMercurialRepo(tt.dir)
			
			if tt.want == nil {
				if got != nil {
					t.Errorf("checkMercurialRepo() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("checkMercurialRepo() = nil, want RepositoryInfo")
				} else {
					if got.Path != tt.want.Path {
						t.Errorf("checkMercurialRepo() path = %v, want %v", got.Path, tt.want.Path)
					}
					if got.Type != tt.want.Type {
						t.Errorf("checkMercurialRepo() type = %v, want %v", got.Type, tt.want.Type)
					}
				}
			}
		})
	}
}

// TestGetSystemRecommendations tests the getSystemRecommendations function
func TestGetSystemRecommendations(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name          string
		osInfo        *OSInfo
		hardwareInfo  *HardwareInfo
		devToolsInfo  *DevelopmentToolsInfo
		wantContains  []string
	}{
		{
			name: "Windows system",
			osInfo: &OSInfo{
				Name: "Windows",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"Windows Terminal",
				"Windows Subsystem for Linux",
			},
		},
		{
			name: "Linux system",
			osInfo: &OSInfo{
				Name:         "Linux",
				Distribution: "ubuntu",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"terminal emulator",
				"apt update",
			},
		},
		{
			name: "macOS system",
			osInfo: &OSInfo{
				Name: "macOS",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"Homebrew",
				"iTerm2",
			},
		},
		{
			name: "High memory usage",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{
				Memory: MemoryInfo{
					UsagePercent: 85.0,
				},
			},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"High memory usage",
			},
		},
		{
			name: "Low disk space",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{
				Storage: []StorageInfo{
					{
						Mountpoint:  "/",
						UsagePercent: 95.0,
					},
				},
			},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"Low disk space",
			},
		},
		{
			name: "Missing development tools",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{
				Git: &ToolInfo{
					Installed: false,
				},
				Go: &ToolInfo{
					Installed: false,
				},
				Node: &ToolInfo{
					Installed: false,
				},
				Docker: &ToolInfo{
					Installed: false,
				},
			},
			wantContains: []string{
				"Consider installing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSystemRecommendations(tt.osInfo, tt.hardwareInfo, tt.devToolsInfo)
			
			if len(got) == 0 {
				t.Error("getSystemRecommendations() returned no recommendations")
			}
			
			// Check for expected content in recommendations
			for _, wantContain := range tt.wantContains {
				found := false
				for _, rec := range got {
					if strings.Contains(rec, wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getSystemRecommendations() missing recommendation containing %s", wantContain)
				}
			}
		})
	}
}

// TestRepositoryInfoStruct tests the RepositoryInfo struct
func TestRepositoryInfoStruct(t *testing.T) {
	lastActivity, _ := time.Parse("2006-01-02 15:04:05 -0700", "2023-01-01 12:00:00 +0000")
	
	repoInfo := &RepositoryInfo{
		Path:         "/home/user/project",
		Type:         "git",
		RemoteURL:    "https://github.com/user/project.git",
		Branch:       "main",
		Commit:       "a1b2c3d4e5f6",
		Status:       "clean",
		Modified:     false,
		Staged:       false,
		Untracked:    false,
		LastActivity: lastActivity,
	}

	// Verify all fields are set correctly
	if repoInfo.Path != "/home/user/project" {
		t.Errorf("RepositoryInfo.Path = %v, want /home/user/project", repoInfo.Path)
	}

	if repoInfo.Type != "git" {
		t.Errorf("RepositoryInfo.Type = %v, want git", repoInfo.Type)
	}

	if repoInfo.RemoteURL != "https://github.com/user/project.git" {
		t.Errorf("RepositoryInfo.RemoteURL = %v, want https://github.com/user/project.git", repoInfo.RemoteURL)
	}

	if repoInfo.Branch != "main" {
		t.Errorf("RepositoryInfo.Branch = %v, want main", repoInfo.Branch)
	}

	if repoInfo.Commit != "a1b2c3d4e5f6" {
		t.Errorf("RepositoryInfo.Commit = %v, want a1b2c3d4e5f6", repoInfo.Commit)
	}

	if repoInfo.Status != "clean" {
		t.Errorf("RepositoryInfo.Status = %v, want clean", repoInfo.Status)
	}

	if repoInfo.Modified {
		t.Errorf("RepositoryInfo.Modified = %v, want false", repoInfo.Modified)
	}

	if repoInfo.Staged {
		t.Errorf("RepositoryInfo.Staged = %v, want false", repoInfo.Staged)
	}

	if repoInfo.Untracked {
		t.Errorf("RepositoryInfo.Untracked = %v, want false", repoInfo.Untracked)
	}

	if !repoInfo.LastActivity.Equal(lastActivity) {
		t.Errorf("RepositoryInfo.LastActivity = %v, want %v", repoInfo.LastActivity, lastActivity)
	}
}

// TestRepositoryDetectionWithREPOPATH tests repository detection with REPO_PATH environment variable
func TestRepositoryDetectionWithREPOPATH(t *testing.T) {
	// Save original functions
	originalGetwd := os.Getwd
	originalStat := os.Stat
	originalGetenv := os.Getenv
	originalAbs := filepath.Abs
	originalDir := filepath.Dir
	
	// Restore after test
	defer func() {
		os.Getwd = originalGetwd
		os.Stat = originalStat
		os.Getenv = originalGetenv
		filepath.Abs = originalAbs
		filepath.Dir = originalDir
	}()

	// Set mock functions
	os.Getwd = mockGetwd
	os.Stat = func(name string) (os.FileInfo, error) {
		switch name {
		case "/home/user/project/.git":
			return &mockFileInfo{name: ".git", isDir: true}, nil
		case "/home/user/other-project/.git":
			return &mockFileInfo{name: ".git", isDir: true}, nil
		default:
			return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
		}
	}
	os.Getenv = func(key string) string {
		if key == "REPO_PATH" {
			return "/home/user/other-project"
		}
		return ""
	}
	filepath.Abs = mockAbs
	filepath.Dir = mockDir

	got, err := detectRepositories()
	if err != nil {
		t.Errorf("detectRepositories() error = %v", err)
		return
	}

	if got == nil {
		t.Error("detectRepositories() returned nil")
		return
	}

	// Should find repositories in both current directory and REPO_PATH
	if len(got) < 2 {
		t.Errorf("detectRepositories() found %d repositories, want at least 2", len(got))
	}
}

// TestRepositoryDetectionWithoutREPOPATH tests repository detection without REPO_PATH environment variable
func TestRepositoryDetectionWithoutREPOPATH(t *testing.T) {
	// Save original functions
	originalGetwd := os.Getwd
	originalStat := os.Stat
	originalGetenv := os.Getenv
	originalAbs := filepath.Abs
	originalDir := filepath.Dir
	
	// Restore after test
	defer func() {
		os.Getwd = originalGetwd
		os.Stat = originalStat
		os.Getenv = originalGetenv
		filepath.Abs = originalAbs
		filepath.Dir = originalDir
	}()

	// Set mock functions
	os.Getwd = mockGetwd
	os.Stat = func(name string) (os.FileInfo, error) {
		if name == "/home/user/project/.git" {
			return &mockFileInfo{name: ".git", isDir: true}, nil
		}
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
	}
	os.Getenv = func(key string) string {
		return "" // No REPO_PATH
	}
	filepath.Abs = mockAbs
	filepath.Dir = mockDir

	got, err := detectRepositories()
	if err != nil {
		t.Errorf("detectRepositories() error = %v", err)
		return
	}

	if got == nil {
		t.Error("detectRepositories() returned nil")
		return
	}

	// Should find repository only in current directory and parent directories
	if len(got) == 0 {
		t.Error("detectRepositories() found no repositories")
	}
}

// TestErrorHandlingInRepositoryDetection tests error handling in repository detection
func TestErrorHandlingInRepositoryDetection(t *testing.T) {
	// Save original functions
	originalGetwd := os.Getwd
	originalStat := os.Stat
	originalAbs := filepath.Abs
	originalDir := filepath.Dir
	
	// Restore after test
	defer func() {
		os.Getwd = originalGetwd
		os.Stat = originalStat
		filepath.Abs = originalAbs
		filepath.Dir = originalDir
	}()

	// Set mock functions that return errors
	os.Getwd = func() (string, error) {
		return "", &os.PathError{Op: "getwd", Path: ".", Err: os.ErrNotExist}
	}

	got, err := detectRepositories()
	
	// Should return an error when os.Getwd fails
	if err == nil {
		t.Error("detectRepositories() should return an error when os.Getwd fails")
	}

	if got != nil {
		t.Error("detectRepositories() should return nil when os.Getwd fails")
	}

	// Test with os.Stat error
	os.Getwd = mockGetwd
	os.Stat = func(name string) (os.FileInfo, error) {
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrPermission}
	}
	filepath.Abs = mockAbs
	filepath.Dir = mockDir

	got, err = detectRepositories()
	
	// Should handle os.Stat errors gracefully
	if err != nil {
		t.Errorf("detectRepositories() error = %v, want nil", err)
		return
	}

	if got == nil {
		t.Error("detectRepositories() returned nil")
		return
	}
}

// TestPlatformSpecificRecommendations tests platform-specific recommendations
func TestPlatformSpecificRecommendations(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name         string
		osInfo       *OSInfo
		hardwareInfo *HardwareInfo
		devToolsInfo *DevelopmentToolsInfo
		platform     string
	}{
		{
			name: "Windows without PowerShell",
			osInfo: &OSInfo{
				Name: "Windows",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{
				PowerShell: nil,
			},
			platform: "windows",
		},
		{
			name: "Linux without package manager",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{
				PackageMgrs: []PackageMgrInfo{},
			},
			platform: "linux",
		},
		{
			name: "macOS without Homebrew",
			osInfo: &OSInfo{
				Name: "macOS",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{
				PackageMgrs: []PackageMgrInfo{},
			},
			platform: "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for platform-specific recommendations
			// In a real implementation, we would mock runtime.GOOS
			// and verify the recommendations
			
			got := getSystemRecommendations(tt.osInfo, tt.hardwareInfo, tt.devToolsInfo)
			
			// Verify recommendations are generated
			if len(got) == 0 {
				t.Error("getSystemRecommendations() returned no recommendations")
			}
			
			// Verify general recommendations are included
			foundGeneral := false
			for _, rec := range got {
				if strings.Contains(rec, "MCP PowerShell server") || strings.Contains(rec, "MCP Bash server") {
					foundGeneral = true
					break
				}
			}
			if !foundGeneral {
				t.Error("getSystemRecommendations() missing general recommendations")
			}
		})
	}
}

// TestHardwareSpecificRecommendations tests hardware-specific recommendations
func TestHardwareSpecificRecommendations(t *testing.T) {
	tests := []struct {
		name         string
		osInfo       *OSInfo
		hardwareInfo *HardwareInfo
		devToolsInfo *DevelopmentToolsInfo
		wantContains []string
	}{
		{
			name: "High memory usage",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{
				Memory: MemoryInfo{
					UsagePercent: 85.0,
				},
			},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"High memory usage",
			},
		},
		{
			name: "Multi-core CPU",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{
				CPU: CPUInfo{
					Threads: 8,
				},
			},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"Multi-core CPU",
			},
		},
		{
			name: "Low disk space",
			osInfo: &OSInfo{
				Name: "Linux",
			},
			hardwareInfo: &HardwareInfo{
				Storage: []StorageInfo{
					{
						Mountpoint:  "/",
						UsagePercent: 95.0,
					},
				},
			},
			devToolsInfo: &DevelopmentToolsInfo{},
			wantContains: []string{
				"Low disk space",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSystemRecommendations(tt.osInfo, tt.hardwareInfo, tt.devToolsInfo)
			
			// Check for expected content in recommendations
			for _, wantContain := range tt.wantContains {
				found := false
				for _, rec := range got {
					if strings.Contains(rec, wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getSystemRecommendations() missing recommendation containing %s", wantContain)
				}
			}
		})
	}
}

// TestDevelopmentToolsRecommendations tests development tools recommendations
func TestDevelopmentToolsRecommendations(t *testing.T) {
	// Save original runtime.GOOS
	originalGOOS := runtime.GOOS
	
	// Restore after test
	defer func() {
		// Note: We can't actually modify runtime.GOOS in Go
		// This is more of a conceptual test
		_ = originalGOOS
	}()

	tests := []struct {
		name         string
		osInfo       *OSInfo
		hardwareInfo *HardwareInfo
		devToolsInfo *DevelopmentToolsInfo
		platform     string
		wantContains []string
	}{
		{
			name: "Missing tools on Windows",
			osInfo: &OSInfo{
				Name: "Windows",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{
				Git: &ToolInfo{Installed: false},
				Go: &ToolInfo{Installed: false},
				Node: &ToolInfo{Installed: false},
				Docker: &ToolInfo{Installed: false},
			},
			platform: "windows",
			wantContains: []string{
				"Consider installing",
				"Windows package manager",
			},
		},
		{
			name: "Missing tools on macOS",
			osInfo: &OSInfo{
				Name: "macOS",
			},
			hardwareInfo: &HardwareInfo{},
			devToolsInfo: &DevelopmentToolsInfo{
				Git: &ToolInfo{Installed: false},
				Go: &ToolInfo{Installed: false},
				Node: &ToolInfo{Installed: false},
				Docker: &ToolInfo{Installed: false},
			},
			platform: "darwin",
			wantContains: []string{
				"Consider installing",
				"Homebrew",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test for platform-specific development tools recommendations
			// In a real implementation, we would mock runtime.GOOS
			// and verify the recommendations
			
			got := getSystemRecommendations(tt.osInfo, tt.hardwareInfo, tt.devToolsInfo)
			
			// Check for expected content in recommendations
			for _, wantContain := range tt.wantContains {
				found := false
				for _, rec := range got {
					if strings.Contains(rec, wantContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getSystemRecommendations() missing recommendation containing %s", wantContain)
				}
			}
		})
	}
}