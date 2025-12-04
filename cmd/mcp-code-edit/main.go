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

	if err := handleInitialize(scanner, encoder); err != nil {
		fmt.Fprintf(os.Stderr, "Initialize failed: %v\n", err)
		os.Exit(1)
	}

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
				Name:    "mcp-code-edit",
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
									"description": "Operation type: apply_diff, replace_code, create_file, delete_file, rename_file, move_file, copy_file",
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
		case "apply_diff":
			result, err = toolApplyDiff(opArgs)
		case "replace_code":
			result, err = toolReplaceCode(opArgs)
		case "create_file":
			result, err = toolCreateFile(opArgs)
		case "delete_file":
			result, err = toolDeleteFile(opArgs)
		case "rename_file", "move_file":
			result, err = toolRenameFile(opArgs)
		case "copy_file":
			result, err = toolCopyFile(opArgs)
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

// getFilePath extracts file path from args, accepting both "path" and "file_path"
func getFilePath(args map[string]interface{}) (string, error) {
	if filePath, ok := args["file_path"].(string); ok && filePath != "" {
		return filePath, nil
	}
	if filePath, ok := args["path"].(string); ok && filePath != "" {
		return filePath, nil
	}
	return "", fmt.Errorf("file_path or path is required")
}

// applyUnifiedDiff applies a unified diff to a file
func applyUnifiedDiff(fileContent, diff string) (string, error) {
	fileLines := strings.Split(fileContent, "\n")
	diffLines := strings.Split(diff, "\n")

	var newLines []string
	fileIdx := 0
	inHunk := false

	for i := 0; i < len(diffLines); i++ {
		line := diffLines[i]

		// Skip diff header lines
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			continue
		}

		// Parse hunk header (e.g., "@@ -1,4 +1,9 @@")
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			// Extract the old line range
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				oldRange := parts[1] // e.g., "-1,4"
				if strings.HasPrefix(oldRange, "-") {
					oldRange = oldRange[1:]
					rangeParts := strings.Split(oldRange, ",")
					if len(rangeParts) > 0 {
						var startLine int
						fmt.Sscanf(rangeParts[0], "%d", &startLine)
						// Adjust to 0-based index
						if startLine > 0 {
							fileIdx = startLine - 1
						}
					}
				}
			}
			continue
		}

		if !inHunk {
			continue
		}

		// Process diff lines
		if len(line) == 0 {
			// Empty line - preserve it
			if fileIdx < len(fileLines) {
				newLines = append(newLines, fileLines[fileIdx])
				fileIdx++
			} else {
				newLines = append(newLines, "")
			}
		} else if line[0] == ' ' {
			// Context line - keep existing line
			if fileIdx < len(fileLines) {
				newLines = append(newLines, fileLines[fileIdx])
				fileIdx++
			}
		} else if line[0] == '-' {
			// Removed line - skip it (don't add to newLines, advance fileIdx)
			if fileIdx < len(fileLines) {
				fileIdx++
			}
		} else if line[0] == '+' {
			// Added line - add it (don't advance fileIdx)
			newLines = append(newLines, line[1:])
		}
	}

	// Add remaining lines from original file
	for fileIdx < len(fileLines) {
		newLines = append(newLines, fileLines[fileIdx])
		fileIdx++
	}

	return strings.Join(newLines, "\n"), nil
}

func toolApplyDiff(args map[string]interface{}) (string, error) {
	filePath, err := getFilePath(args)
	if err != nil {
		return "", err
	}

	fullPath := resolvePath(filePath)

	// Read current file
	currentContent, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", filePath)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	currentStr := string(currentContent)
	var newFileContent string

	// Check if diff format is provided
	if diff, ok := args["diff"].(string); ok && diff != "" {
		// Apply unified diff format
		newFileContent, err = applyUnifiedDiff(currentStr, diff)
		if err != nil {
			return "", fmt.Errorf("failed to apply diff: %w", err)
		}
	} else {
		// Use old_content/new_content format
		oldContent, ok := args["old_content"].(string)
		if !ok {
			return "", fmt.Errorf("old_content or diff is required")
		}

		newContent, ok := args["new_content"].(string)
		if !ok {
			return "", fmt.Errorf("new_content or diff is required")
		}

		// Replace old_content with new_content
		if !strings.Contains(currentStr, oldContent) {
			return "", fmt.Errorf("old_content not found in file")
		}

		newFileContent = strings.Replace(currentStr, oldContent, newContent, 1)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(newFileContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return "Diff applied successfully", nil
}

func toolReplaceCode(args map[string]interface{}) (string, error) {
	filePath, err := getFilePath(args)
	if err != nil {
		return "", err
	}

	// Support both old_code/old_content and new_code/new_content for flexibility
	var oldCode string
	var ok bool

	// Try old_code first, then fall back to old_content
	if oldCode, ok = args["old_code"].(string); !ok {
		if oldCode, ok = args["old_content"].(string); !ok {
			return "", fmt.Errorf("old_code or old_content is required")
		}
	}

	var newCode string
	// Try new_code first, then fall back to new_content
	if newCode, ok = args["new_code"].(string); !ok {
		if newCode, ok = args["new_content"].(string); !ok {
			return "", fmt.Errorf("new_code or new_content is required")
		}
	}

	fullPath := resolvePath(filePath)

	currentContent, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	currentStr := string(currentContent)

	if !strings.Contains(currentStr, oldCode) {
		return "", fmt.Errorf("old_code not found in file")
	}

	newFileContent := strings.Replace(currentStr, oldCode, newCode, 1)

	if err := os.WriteFile(fullPath, []byte(newFileContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return "Code replaced successfully", nil
}

func toolCreateFile(args map[string]interface{}) (string, error) {
	filePath, err := getFilePath(args)
	if err != nil {
		return "", err
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("content is required")
	}

	fullPath := resolvePath(filePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); err == nil {
		return "", fmt.Errorf("file already exists")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return "File created successfully", nil
}

func toolDeleteFile(args map[string]interface{}) (string, error) {
	filePath, err := getFilePath(args)
	if err != nil {
		return "", err
	}

	fullPath := resolvePath(filePath)

	if err := os.Remove(fullPath); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	return "File deleted successfully", nil
}

func toolRenameFile(args map[string]interface{}) (string, error) {
	// Accept both old_path/new_path and source_path/destination_path
	var oldPath, newPath string
	var ok bool

	if oldPath, ok = args["old_path"].(string); !ok || oldPath == "" {
		if oldPath, ok = args["source_path"].(string); !ok || oldPath == "" {
			return "", fmt.Errorf("old_path or source_path is required")
		}
	}

	if newPath, ok = args["new_path"].(string); !ok || newPath == "" {
		if newPath, ok = args["destination_path"].(string); !ok || newPath == "" {
			return "", fmt.Errorf("new_path or destination_path is required")
		}
	}

	oldFullPath := resolvePath(oldPath)
	newFullPath := resolvePath(newPath)

	// Check if source file exists
	if _, err := os.Stat(oldFullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("source file does not exist: %s", oldPath)
	}

	// Check if destination already exists
	if _, err := os.Stat(newFullPath); err == nil {
		return "", fmt.Errorf("destination file already exists: %s", newPath)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(newFullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Perform rename/move operation
	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	return "File renamed successfully", nil
}

func toolCopyFile(args map[string]interface{}) (string, error) {
	// Accept both source_path/destination_path and old_path/new_path
	var sourcePath, destPath string
	var ok bool

	if sourcePath, ok = args["source_path"].(string); !ok || sourcePath == "" {
		if sourcePath, ok = args["old_path"].(string); !ok || sourcePath == "" {
			return "", fmt.Errorf("source_path or old_path is required")
		}
	}

	if destPath, ok = args["destination_path"].(string); !ok || destPath == "" {
		if destPath, ok = args["new_path"].(string); !ok || destPath == "" {
			return "", fmt.Errorf("destination_path or new_path is required")
		}
	}

	sourceFullPath := resolvePath(sourcePath)
	destFullPath := resolvePath(destPath)

	// Check if source file exists
	sourceInfo, err := os.Stat(sourceFullPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("source file does not exist: %s", sourcePath)
	}
	if err != nil {
		return "", fmt.Errorf("failed to stat source file: %w", err)
	}

	// Check if source is a directory
	if sourceInfo.IsDir() {
		return "", fmt.Errorf("source path is a directory, not a file: %s", sourcePath)
	}

	// Check if destination already exists
	if _, err := os.Stat(destFullPath); err == nil {
		return "", fmt.Errorf("destination file already exists: %s", destPath)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destFullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source file
	sourceContent, err := os.ReadFile(sourceFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to destination with same permissions
	fileMode := sourceInfo.Mode()
	if err := os.WriteFile(destFullPath, sourceContent, fileMode); err != nil {
		return "", fmt.Errorf("failed to write destination file: %w", err)
	}

	return "File copied successfully", nil
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
