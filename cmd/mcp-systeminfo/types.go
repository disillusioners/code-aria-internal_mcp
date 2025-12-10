package main

import (
	"encoding/json"
	"time"
)

// MCP protocol types
type MCPMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type InitializeResponse struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type ToolsListResponse struct {
	Tools []Tool `json:"tools"`
}

type ToolsCallRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolsCallResponse struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// SystemInfo operation types
type SystemInfoOperation struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

type SystemInfoResult struct {
	Operation     string                 `json:"operation"`
	Params        map[string]interface{} `json:"params"`
	Status        string                 `json:"status"`
	Result        interface{}            `json:"result,omitempty"`
	Message       string                 `json:"message,omitempty"`
	ErrorCode     int                    `json:"error_code,omitempty"`
	ErrorType     string                 `json:"error_type,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// System information structures
type SystemInfo struct {
	Timestamp     time.Time                `json:"timestamp"`
	OS            OSInfo                   `json:"os"`
	Hardware      HardwareInfo             `json:"hardware"`
	Environment   EnvironmentInfo          `json:"environment"`
	Shell         ShellInfo                `json:"shell"`
	Development   DevelopmentToolsInfo     `json:"development"`
	Networking    NetworkInfo              `json:"networking"`
	Repositories  []RepositoryInfo         `json:"repositories,omitempty"`
	Recommendations []string               `json:"recommendations"`
}

type OSInfo struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Architecture   string `json:"architecture"`
	Platform       string `json:"platform"`
	Build          string `json:"build,omitempty"`
	Major          int    `json:"major"`
	Minor          int    `json:"minor"`
	Patch          int    `json:"patch,omitempty"`
	KernelVersion  string `json:"kernel_version,omitempty"`
	Distribution   string `json:"distribution,omitempty"`
	CodeName       string `json:"code_name,omitempty"`
}

type HardwareInfo struct {
	CPU           CPUInfo      `json:"cpu"`
	Memory        MemoryInfo   `json:"memory"`
	Storage       []StorageInfo `json:"storage"`
	Displays      []DisplayInfo `json:"displays,omitempty"`
	NetworkCards  []NetworkCardInfo `json:"network_cards,omitempty"`
}

type CPUInfo struct {
	ModelName     string    `json:"model_name"`
	Vendor        string    `json:"vendor"`
	Architecture  string    `json:"architecture"`
	Cores         int       `json:"cores"`
	Threads       int       `json:"threads"`
	Frequency     float64   `json:"frequency_ghz"`
	Cache         CacheInfo `json:"cache"`
	Features      []string  `json:"features,omitempty"`
}

type CacheInfo struct {
	L1 int `json:"l1_kb,omitempty"`
	L2 int `json:"l2_kb,omitempty"`
	L3 int `json:"l3_kb,omitempty"`
}

type MemoryInfo struct {
	Total     uint64  `json:"total_bytes"`
	Available uint64  `json:"available_bytes"`
	Used      uint64  `json:"used_bytes"`
	Free      uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	Swap      SwapInfo `json:"swap,omitempty"`
}

type SwapInfo struct {
	Total uint64 `json:"total_bytes,omitempty"`
	Used  uint64 `json:"used_bytes,omitempty"`
	Free  uint64 `json:"free_bytes,omitempty"`
}

type StorageInfo struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	FSType     string `json:"fs_type"`
	Total      uint64 `json:"total_bytes"`
	Free       uint64 `json:"free_bytes"`
	Used       uint64 `json:"used_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	ReadOnly   bool   `json:"read_only"`
}

type DisplayInfo struct {
	ID       int     `json:"id"`
	Resolution string `json:"resolution"`
	DPI      int     `json:"dpi,omitempty"`
	Primary  bool    `json:"primary"`
}

type NetworkCardInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	MAC        string   `json:"mac,omitempty"`
	IPv4       string   `json:"ipv4,omitempty"`
	IPv6       string   `json:"ipv6,omitempty"`
	Status     string   `json:"status"`
	Speed      int      `json:"speed_mbps,omitempty"`
}

type EnvironmentInfo struct {
	WorkingDir  string            `json:"working_directory"`
	HomeDir     string            `json:"home_directory"`
	Username    string            `json:"username"`
	Hostname    string            `json:"hostname"`
	Domain      string            `json:"domain,omitempty"`
	Path        []string          `json:"path"`
	EnvVars     map[string]string `json:"env_vars"`
	RepoPath    string            `json:"repo_path,omitempty"`
}

type ShellInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version,omitempty"`
	Path         string            `json:"path"`
	Type         string            `json:"type"` // "bash", "powershell", "zsh", etc.
	Features     []string          `json:"features,omitempty"`
	Aliases      map[string]string `json:"aliases,omitempty"`
	Functions    []string          `json:"functions,omitempty"`
}

type DevelopmentToolsInfo struct {
	Go          *ToolInfo  `json:"go,omitempty"`
	Node        *ToolInfo  `json:"node,omitempty"`
	Python      *ToolInfo  `json:"python,omitempty"`
	Python3     *ToolInfo  `json:"python3,omitempty"`
	Ruby        *ToolInfo  `json:"ruby,omitempty"`
	Java        *ToolInfo  `json:"java,omitempty"`
	Git         *ToolInfo  `json:"git,omitempty"`
	Docker      *ToolInfo  `json:"docker,omitempty"`
	PowerShell  *ToolInfo  `json:"powershell,omitempty"`
	CMake       *ToolInfo  `json:"cmake,omitempty"`
	Maven       *ToolInfo  `json:"maven,omitempty"`
	Gradle      *ToolInfo  `json:"gradle,omitempty"`
	Make        *ToolInfo  `json:"make,omitempty"`
	Cargo       *ToolInfo  `json:"cargo,omitempty"`
	Rustc       *ToolInfo  `json:"rustc,omitempty"`
	GCC         *ToolInfo  `json:"gcc,omitempty"`
	Clang       *ToolInfo  `json:"clang,omitempty"`
	Dotnet      *ToolInfo  `json:"dotnet,omitempty"`
	PackageMgrs []PackageMgrInfo `json:"package_managers,omitempty"`
}

type ToolInfo struct {
	Version    string   `json:"version"`
	Path       string   `json:"path"`
	Installed  bool     `json:"installed"`
	Executable string   `json:"executable"`
	Features   []string `json:"features,omitempty"`
}

type PackageMgrInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Type    string `json:"type"` // "apt", "yum", "brew", "choco", "winget", etc.
}

type NetworkInfo struct {
	Hostname      string            `json:"hostname"`
	Domain        string            `json:"domain,omitempty"`
	IPAddress     string            `json:"ip_address"`
	PublicIP      string            `json:"public_ip,omitempty"`
	MAC           string            `json:"mac_address,omitempty"`
	Gateway       string            `json:"gateway,omitempty"`
	DNS           []string          `json:"dns,omitempty"`
	Proxy         ProxyInfo         `json:"proxy,omitempty"`
	Connected     bool              `json:"connected"`
	InternetAccess bool             `json:"internet_access"`
}

type ProxyInfo struct {
	HTTP  string `json:"http,omitempty"`
	HTTPS string `json:"https,omitempty"`
	FTP   string `json:"ftp,omitempty"`
	NoProxy []string `json:"no_proxy,omitempty"`
}

type RepositoryInfo struct {
	Path         string            `json:"path"`
	Type         string            `json:"type"` // "git", "svn", "hg", etc.
	RemoteURL    string            `json:"remote_url,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Commit       string            `json:"commit,omitempty"`
	Status       string            `json:"status,omitempty"`
	Modified     bool              `json:"modified"`
	Staged       bool              `json:"staged"`
	Untracked    bool              `json:"untracked"`
	LastActivity time.Time         `json:"last_activity,omitempty"`
}

// Command execution result
type CommandResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
	Command    string `json:"command"`
	Timeout    bool   `json:"timeout,omitempty"`
}

// Security validation result
type SecurityResult struct {
	Valid   bool   `json:"valid"`
	Reason  string `json:"reason,omitempty"`
	Rule    string `json:"rule,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

// Audit log entry
type AuditLog struct {
	Timestamp    time.Time              `json:"timestamp"`
	Operation    string                 `json:"operation"`
	Command      string                 `json:"command,omitempty"`
	User         string                 `json:"user,omitempty"`
	WorkingDir   string                 `json:"working_directory,omitempty"`
	Result       *CommandResult         `json:"result,omitempty"`
	Security     *SecurityResult        `json:"security,omitempty"`
	DurationMs   int64                  `json:"duration_ms,omitempty"`
	Success      bool                   `json:"success"`
	ErrorCode    int                    `json:"error_code,omitempty"`
	ErrorType    string                 `json:"error_type,omitempty"`
}

// Security policy configuration
type SecurityPolicy struct {
	AllowedCommands    map[string]bool `json:"allowed_commands"`
	BlockedPatterns    []string        `json:"blocked_patterns"`
	MaxCommandLen      int             `json:"max_command_len"`
	DefaultTimeout     int             `json:"default_timeout"`
	MaxTimeout         int             `json:"max_timeout"`
	AllowShellAccess   bool           `json:"allow_shell_access"`
}

// Command exists result
type CommandExistsResult struct {
	Exists  bool   `json:"exists"`
	Command string `json:"command"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}