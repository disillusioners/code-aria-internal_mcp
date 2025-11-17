package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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
			Name:        "search_code",
			Description: "Search for code patterns or keywords in the codebase",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (regex pattern)",
					},
					"file_patterns": map[string]interface{}{
						"type":        "array",
						"description": "File patterns to search in (e.g., ['*.go', '*.ts'])",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_file_dependencies",
			Description: "Get imports and dependencies for a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "analyze_function",
			Description: "Get function details and signature",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"function_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the function",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file containing the function",
					},
				},
				"required": []string{"function_name", "file_path"},
			},
		},
		{
			Name:        "get_code_context",
			Description: "Get code with surrounding context",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file",
					},
					"line_range": map[string]interface{}{
						"type":        "string",
						"description": "Line range (e.g., '10:20')",
					},
				},
				"required": []string{"file_path"},
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
	case "search_code":
		result, err = toolSearchCode(req.Arguments)
	case "get_file_dependencies":
		result, err = toolGetFileDependencies(req.Arguments)
	case "analyze_function":
		result, err = toolAnalyzeFunction(req.Arguments)
	case "get_code_context":
		result, err = toolGetCodeContext(req.Arguments)
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

	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
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

	result, _ := json.Marshal(matches)
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

	result, _ := json.Marshal(imports)
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

	result, _ := json.Marshal(map[string]interface{}{
		"name":      functionName,
		"file":      filePath,
		"signature": matches[0],
	})
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

