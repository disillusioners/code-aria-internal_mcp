package main

import (
	"encoding/json"
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

// Go-specific operation types
type LintOperation struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

// LintResult represents the structured result of a lint operation
type LintResult struct {
	Target      string      `json:"target"`
	TotalIssues int         `json:"total_issues"`
	Issues      []LintIssue `json:"issues"`
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
}

// LintIssue represents a single lint issue
type LintIssue struct {
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column,omitempty"`
	Severity    string   `json:"severity"` // error, warning, info
	Linter      string   `json:"linter"`
	Message     string   `json:"message"`
	Fix         string   `json:"fix,omitempty"`
	SourceLines []string `json:"source_lines,omitempty"`
}


