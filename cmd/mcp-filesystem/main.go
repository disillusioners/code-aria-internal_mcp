package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// Initialize handshake
	if err := handleInitialize(scanner, encoder); err != nil {
		fmt.Fprintf(os.Stderr, "Initialize failed: %v\n", err)
		os.Exit(1)
	}

	// Handle requests
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg MCPMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Method != "" {
			handleRequest(&msg, encoder)
		}
	}
}

func handleInitialize(scanner *bufio.Scanner, encoder *json.Encoder) error {
	// Read initialize request
	if !scanner.Scan() {
		return fmt.Errorf("no initialize request")
	}

	var initReq MCPMessage
	if err := json.Unmarshal(scanner.Bytes(), &initReq); err != nil {
		return fmt.Errorf("failed to parse initialize: %w", err)
	}

	// Send initialize response
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      initReq.ID,
		Result: InitializeResponse{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    "mcp-filesystem",
				Version: "1.0.0",
			},
		},
	}

	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to send initialize response: %w", err)
	}

	// Read initialized notification
	if !scanner.Scan() {
		return fmt.Errorf("no initialized notification")
	}

	return nil
}

func handleRequest(msg *MCPMessage, encoder *json.Encoder) {
	switch msg.Method {
	case "tools/list":
		handleToolsList(msg, encoder)
	case "tools/call":
		handleToolCall(msg, encoder)
	default:
		sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown method: %s", msg.Method), nil)
	}
}

func handleToolsList(msg *MCPMessage, encoder *json.Encoder) {
	tools := []Tool{
		{
			Name:        "read_file",
			Description: "Read the contents of a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "list_directory",
			Description: "List files and directories in a directory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the directory to list",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "get_file_tree",
			Description: "Get a directory tree structure",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"root_path": map[string]interface{}{
						"type":        "string",
						"description": "Root path for the tree",
					},
					"max_depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum depth to traverse",
					},
				},
				"required": []string{"root_path"},
			},
		},
		{
			Name:        "file_exists",
			Description: "Check if a file or directory exists",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to check",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "create_directory",
			Description: "Create a directory and all parent directories if they don't exist",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the directory to create",
					},
				},
				"required": []string{"path"},
			},
		},
	}

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ToolsListResponse{
			Tools: tools,
		},
	}

	encoder.Encode(response)
}

func handleToolCall(msg *MCPMessage, encoder *json.Encoder) {
	var req ToolsCallRequest
	reqJSON, _ := json.Marshal(msg.Params)
	json.Unmarshal(reqJSON, &req)

	var result string
	var err error

	switch req.Name {
	case "read_file":
		result, err = toolReadFile(req.Arguments)
	case "list_directory":
		result, err = toolListDirectory(req.Arguments)
	case "get_file_tree":
		result, err = toolGetFileTree(req.Arguments)
	case "file_exists":
		result, err = toolFileExists(req.Arguments)
	case "create_directory":
		result, err = toolCreateDirectory(req.Arguments)
	default:
		sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown tool: %s", req.Name), nil)
		return
	}

	if err != nil {
		sendError(encoder, msg.ID, -32603, err.Error(), nil)
		return
	}

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ToolsCallResponse{
			Content: []Content{
				{Type: "text", Text: result},
			},
		},
	}

	encoder.Encode(response)
}

func toolReadFile(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	// Resolve path relative to repo
	fullPath := resolvePath(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}

func toolListDirectory(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	fullPath := resolvePath(path)
	
	// Check if path exists first
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s. Use 'file_exists' to check if a directory exists, or use 'create_directory' to create it first", path)
	}
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to list directory '%s': %w. Hint: Use 'file_exists' to check if the directory exists, or 'create_directory' to create it", path, err)
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	result, _ := json.Marshal(names)
	return string(result), nil
}

func toolGetFileTree(args map[string]interface{}) (string, error) {
	rootPath, ok := args["root_path"].(string)
	if !ok {
		return "", fmt.Errorf("root_path is required")
	}

	maxDepth := 10
	if md, ok := args["max_depth"].(float64); ok {
		maxDepth = int(md)
	}

	fullPath := resolvePath(rootPath)
	var tree []string

	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(fullPath, path)
		depth := len(strings.Split(rel, string(filepath.Separator)))
		if depth > maxDepth {
			return filepath.SkipDir
		}

		if rel != "." {
			if info.IsDir() {
				tree = append(tree, rel+"/")
			} else {
				tree = append(tree, rel)
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	result, _ := json.Marshal(tree)
	return string(result), nil
}

func toolFileExists(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	fullPath := resolvePath(path)
	exists := true
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		exists = false
	}

	result, _ := json.Marshal(exists)
	return string(result), nil
}

func toolCreateDirectory(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	fullPath := resolvePath(path)
	
	// Check if it already exists
	if info, err := os.Stat(fullPath); err == nil {
		if info.IsDir() {
			return fmt.Sprintf("Directory already exists: %s", path), nil
		}
		return "", fmt.Errorf("path exists but is not a directory: %s", path)
	}
	
	// Create directory recursively
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory '%s': %w", path, err)
	}

	return fmt.Sprintf("Directory created successfully: %s", path), nil
}

func resolvePath(path string) string {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return path
	}

	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(repoPath, path)
}

func sendError(encoder *json.Encoder, id interface{}, code int, message string, data interface{}) {
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	encoder.Encode(response)
}

// MCP types (duplicated from internal/mcp for standalone server)
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

