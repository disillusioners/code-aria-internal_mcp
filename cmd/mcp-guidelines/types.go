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

// Guideline data structures
type GuidelineCategory struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Color       string                 `json:"color,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsActive    bool                   `json:"is_active"`
	TenantID    string                 `json:"tenant_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	UpdatedBy   string                 `json:"updated_by,omitempty"`
}

type Guideline struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Content     string                 `json:"content"`
	CategoryID  *string                `json:"category_id,omitempty"`
	Category    *GuidelineCategory     `json:"category,omitempty"` // Populated when joined
	CategoryOld string                 `json:"category,omitempty"` // Deprecated: kept for backward compatibility
	Tags        []string               `json:"tags,omitempty"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	IsActive    bool                   `json:"is_active"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}
