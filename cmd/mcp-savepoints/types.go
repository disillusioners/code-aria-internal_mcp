package main

import "encoding/json"

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

// Savepoint-related types
type Savepoint struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Timestamp   string   `json:"timestamp"`
	Files       []string `json:"files"`
	Size        int64    `json:"size"`
}

type SavepointMetadata struct {
	Savepoint
	CreatedBy string `json:"created_by"`
}

// FileChange represents a file change with its status
type FileChange struct {
	Path   string
	Status string // "new", "modified", "deleted"
}

// SavepointWithStatus extends Savepoint with file status information
type SavepointWithStatus struct {
	Savepoint
	FilesWithStatus []FileStatusEntry
}

// FileStatusEntry represents a file in a savepoint with its status
type FileStatusEntry struct {
	Path   string
	Status string // "new", "modified", "deleted"
}

// FileRestoreOperation tracks a restore operation for rollback
type FileRestoreOperation struct {
	Type     string // "copy" or "delete"
	FilePath string
	Backup   string // backup path for rollback (if needed)
}
