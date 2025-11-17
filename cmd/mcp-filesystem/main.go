package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
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
			Name:        "apply_operations",
			Description: "Execute multiple operations in a single batch call",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operations": map[string]interface{}{
						"type": "array",
						"description": "List of operations to execute",
						"items": map[string]interface{}{
							"type": "object",
							"description": "Operation object with 'type' field and operation-specific parameters",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "Operation type: read_file, list_directory, get_file_tree, file_exists, create_directory",
								},
							},
						},
					},
				},
				"required": []string{"operations"},
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
	reqJSON, err := json.Marshal(msg.Params)
	if err != nil {
		sendError(encoder, msg.ID, -32602, fmt.Sprintf("failed to marshal params: %v", err), nil)
		return
	}
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		sendError(encoder, msg.ID, -32602, fmt.Sprintf("failed to unmarshal params: %v", err), nil)
		return
	}

	if req.Name == "apply_operations" {
		handleBatchOperations(msg, encoder, req.Arguments)
		return
	}

	// Individual tool calls are no longer exposed, but kept for internal use
	sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown tool: %s. Use apply_operations for batch operations", req.Name), nil)
}

func handleBatchOperations(msg *MCPMessage, encoder *json.Encoder, args map[string]interface{}) {
	operations, ok := args["operations"].([]interface{})
	if !ok {
		sendError(encoder, msg.ID, -32602, "operations array is required", nil)
		return
	}

	if len(operations) == 0 {
		sendError(encoder, msg.ID, -32602, "operations array cannot be empty", nil)
		return
	}

	var results []map[string]interface{}
	var successCount int
	var errorCount int

	for i, op := range operations {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			results = append(results, map[string]interface{}{
				"index": i,
				"type":  "unknown",
				"success": false,
				"error":  "Invalid operation format",
			})
			errorCount++
			continue
		}

		opType, ok := opMap["type"].(string)
		if !ok {
			results = append(results, map[string]interface{}{
				"index": i,
				"type":  "unknown",
				"success": false,
				"error":  "Operation type is required",
			})
			errorCount++
			continue
		}

		// Extract operation-specific arguments
		opArgs := make(map[string]interface{})
		for k, v := range opMap {
			if k != "type" {
				opArgs[k] = v
			}
		}

		// Execute operation based on type
		var result string
		var err error

		switch opType {
		case "read_file":
			result, err = toolReadFile(opArgs)
		case "list_directory":
			result, err = toolListDirectory(opArgs)
		case "get_file_tree":
			result, err = toolGetFileTree(opArgs)
		case "file_exists":
			result, err = toolFileExists(opArgs)
		case "create_directory":
			result, err = toolCreateDirectory(opArgs)
		default:
			err = fmt.Errorf("unknown operation type: %s", opType)
		}

		if err != nil {
			results = append(results, map[string]interface{}{
				"index": i,
				"type":  opType,
				"success": false,
				"error":  err.Error(),
				"result": nil,
			})
			errorCount++
		} else {
			// Parse JSON result if possible, otherwise use as string
			var parsedResult interface{}
			if jsonErr := json.Unmarshal([]byte(result), &parsedResult); jsonErr == nil {
				// Successfully parsed JSON, use parsed result
			} else {
				// Not JSON, use string as-is
				parsedResult = result
			}

			results = append(results, map[string]interface{}{
				"index": i,
				"type":  opType,
				"success": true,
				"result": parsedResult,
				"error":  nil,
			})
			successCount++
		}
	}

	// Create response
	summary := fmt.Sprintf("Batch operations completed: %d succeeded, %d failed", successCount, errorCount)
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"content": []Content{
				{Type: "text", Text: summary},
			},
			"results": results,
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

	result, err := json.Marshal(names)
	if err != nil {
		return "", fmt.Errorf("failed to marshal directory listing: %w", err)
	}
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

	err := filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(fullPath, path)
		// Handle root directory correctly - if rel is ".", depth should be 0
		var depth int
		if rel == "." {
			depth = 0
		} else {
			parts := strings.Split(rel, string(filepath.Separator))
			// Filter out empty parts (can happen with trailing separators)
			var nonEmptyParts []string
			for _, part := range parts {
				if part != "" {
					nonEmptyParts = append(nonEmptyParts, part)
				}
			}
			depth = len(nonEmptyParts)
		}
		if depth > maxDepth {
			return filepath.SkipDir
		}

		if rel != "." {
			if d.IsDir() {
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

	result, err := json.Marshal(tree)
	if err != nil {
		return "", fmt.Errorf("failed to marshal file tree: %w", err)
	}
	return string(result), nil
}

func toolFileExists(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	fullPath := resolvePath(path)
	info, err := os.Stat(fullPath)
	
	result := map[string]interface{}{
		"path":    path,
		"exists":  err == nil,
	}
	
	if err == nil {
		result["is_file"] = !info.IsDir()
		result["is_directory"] = info.IsDir()
		result["resolved_path"] = fullPath
	} else if os.IsNotExist(err) {
		result["error"] = "path does not exist"
	} else {
		// Some other error occurred (permission denied, etc.)
		result["error"] = err.Error()
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
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
			result, err := json.Marshal(map[string]interface{}{
				"message": fmt.Sprintf("Directory already exists: %s", path),
				"path":    path,
			})
			if err != nil {
				return "", fmt.Errorf("failed to marshal result: %w", err)
			}
			return string(result), nil
		}
		return "", fmt.Errorf("path exists but is not a directory: %s", path)
	}
	
	// Create directory recursively
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory '%s': %w", path, err)
	}

	result, err := json.Marshal(map[string]interface{}{
		"message": fmt.Sprintf("Directory created successfully: %s", path),
		"path":    path,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(result), nil
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

