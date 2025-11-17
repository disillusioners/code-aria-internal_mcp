package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	if !scanner.Scan() {
		return fmt.Errorf("no initialize request")
	}

	var initReq MCPMessage
	if err := json.Unmarshal(scanner.Bytes(), &initReq); err != nil {
		return fmt.Errorf("failed to parse initialize: %w", err)
	}

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      initReq.ID,
		Result: InitializeResponse{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    "mcp-codebase",
				Version: "1.0.0",
			},
		},
	}

	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to send initialize response: %w", err)
	}

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
						"type":        "array",
						"description": "List of operations to execute",
						"items": map[string]interface{}{
							"type":        "object",
							"description": "Operation object with 'type' field and operation-specific parameters",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "Operation type: search_code, get_file_dependencies, analyze_function, get_code_context",
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
				"index":   i,
				"type":    "unknown",
				"success": false,
				"error":   "Invalid operation format",
			})
			errorCount++
			continue
		}

		opType, ok := opMap["type"].(string)
		if !ok {
			results = append(results, map[string]interface{}{
				"index":   i,
				"type":    "unknown",
				"success": false,
				"error":   "Operation type is required",
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
		case "search_code":
			result, err = toolSearchCode(opArgs)
		case "get_file_dependencies":
			result, err = toolGetFileDependencies(opArgs)
		case "analyze_function":
			result, err = toolAnalyzeFunction(opArgs)
		case "get_code_context":
			result, err = toolGetCodeContext(opArgs)
		default:
			err = fmt.Errorf("unknown operation type: %s", opType)
		}

		if err != nil {
			results = append(results, map[string]interface{}{
				"index":   i,
				"type":    opType,
				"success": false,
				"error":   err.Error(),
				"result":  nil,
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
				"index":   i,
				"type":    opType,
				"success": true,
				"result":  parsedResult,
				"error":   nil,
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

func toolSearchCode(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("query is required")
	}

	filePatterns := []string{"*"}
	if patterns, ok := args["file_patterns"].([]interface{}); ok {
		filePatterns = make([]string, len(patterns))
		for i, p := range patterns {
			filePatterns[i] = p.(string)
		}
	}

	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	pattern, err := regexp.Compile(query)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []map[string]interface{}

	err = filepath.WalkDir(repoPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		// Check file pattern
		matched := false
		for _, fp := range filePatterns {
			if matched, _ = filepath.Match(fp, filepath.Base(path)); matched {
				break
			}
		}
		if !matched {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if pattern.MatchString(line) {
				relPath, _ := filepath.Rel(repoPath, path)
				matches = append(matches, map[string]interface{}{
					"file":  relPath,
					"line":  i + 1,
					"match": strings.TrimSpace(line),
				})
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	result, err := json.Marshal(matches)
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}
	return string(result), nil
}

func toolGetFileDependencies(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	fullPath := resolvePath(filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Simple import extraction for Go files
	var imports []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "import") {
			// Handle import blocks
			if strings.Contains(line, "(") {
				continue
			}
			if strings.HasPrefix(line, "import \"") {
				imp := strings.TrimPrefix(line, "import \"")
				imp = strings.TrimSuffix(imp, "\"")
				imports = append(imports, imp)
			}
		} else if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"") {
			// Inside import block
			imp := strings.Trim(line, "\"")
			imports = append(imports, imp)
		}
	}

	result, err := json.Marshal(imports)
	if err != nil {
		return "", fmt.Errorf("failed to marshal imports: %w", err)
	}
	return string(result), nil
}

func toolAnalyzeFunction(args map[string]interface{}) (string, error) {
	functionName, ok := args["function_name"].(string)
	if !ok {
		return "", fmt.Errorf("function_name is required")
	}

	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	fullPath := resolvePath(filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Simple function finder
	pattern := regexp.MustCompile(fmt.Sprintf(`func\s+%s\s*\([^)]*\)\s*(?:\([^)]*\))?\s*(?:\{[^}]*\})?`, regexp.QuoteMeta(functionName)))
	matches := pattern.FindAllString(string(data), -1)

	if len(matches) == 0 {
		return "", fmt.Errorf("function %s not found", functionName)
	}

	result, err := json.Marshal(map[string]interface{}{
		"name":      functionName,
		"file":      filePath,
		"signature": matches[0],
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(result), nil
}

func toolGetCodeContext(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	fullPath := resolvePath(filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	startLine := 0
	endLine := len(lines)

	if lineRange, ok := args["line_range"].(string); ok {
		var start, end int
		fmt.Sscanf(lineRange, "%d:%d", &start, &end)
		if start > 0 {
			startLine = start - 1
		}
		if end > 0 {
			endLine = end
		}
	}

	// Add context lines
	contextStart := startLine - 5
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := endLine + 5
	if contextEnd > len(lines) {
		contextEnd = len(lines)
	}

	selectedLines := lines[contextStart:contextEnd]
	result := strings.Join(selectedLines, "\n")

	return result, nil
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

// MCP types
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
