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

// PowerShell operation types
type PowerShellOperation struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

type PowerShellResult struct {
	Operation     string                 `json:"operation"`
	Params        map[string]interface{} `json:"params"`
	Status        string                 `json:"status"`
	Result        interface{}            `json:"result,omitempty"`
	Message       string                 `json:"message,omitempty"`
	ErrorCode     int                    `json:"error_code,omitempty"`
	ErrorType     string                 `json:"error_type,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

// Command execution result
type CommandResult struct {
	ExitCode       int    `json:"exit_code"`
	Stdout         string `json:"stdout"`
	Stderr         string `json:"stderr"`
	DurationMs     int64  `json:"duration_ms"`
	Command        string `json:"command"`
	WorkingDir     string `json:"working_directory,omitempty"`
	Timeout        bool   `json:"timeout,omitempty"`
	LinesExecuted  int    `json:"lines_executed,omitempty"`
	ScriptName     string `json:"script_name,omitempty"`
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
	Script       string                 `json:"script,omitempty"`
	User         string                 `json:"user,omitempty"`
	WorkingDir   string                 `json:"working_directory,omitempty"`
	Environment  map[string]string      `json:"environment,omitempty"`
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
	MaxScriptLen       int             `json:"max_script_len"`
	DefaultTimeout     int             `json:"default_timeout"`
	MaxTimeout         int             `json:"max_timeout"`
	AllowShellAccess   bool           `json:"allow_shell_access"`
	AllowExecutionPolicy bool         `json:"allow_execution_policy"`
}

// Command exists result
type CommandExistsResult struct {
	Exists  bool   `json:"exists"`
	Command string `json:"command"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Error   string `json:"error,omitempty"`
}