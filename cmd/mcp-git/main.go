package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
				Name:    "mcp-git",
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
									"description": "Operation type: get_git_status, get_file_diff, get_commit_history, get_changed_files",
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
		case "get_git_status":
			result, err = toolGetGitStatus(opArgs)
		case "get_file_diff":
			result, err = toolGetFileDiff(opArgs)
		case "get_commit_history":
			result, err = toolGetCommitHistory(opArgs)
		case "get_changed_files":
			result, err = toolGetChangedFiles(opArgs)
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

func toolGetGitStatus(args map[string]interface{}) (string, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

func toolGetFileDiff(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	fullPath := resolvePath(filePath)
	relPath, _ := filepath.Rel(repoPath, fullPath)

	var cmd *exec.Cmd

	// Priority 1: Check if compare_working is true (uncommitted changes)
	if compareWorking, ok := args["compare_working"].(bool); ok && compareWorking {
		cmd = exec.Command("git", "diff", "HEAD", "--", relPath)
	} else if baseCommit, ok := args["base_commit"].(string); ok && baseCommit != "" {
		// Priority 2: Commit comparison (base_commit and optionally target_commit)
		targetCommit := "HEAD"
		if tc, ok := args["target_commit"].(string); ok && tc != "" {
			targetCommit = tc
		}
		cmd = exec.Command("git", "diff", baseCommit, targetCommit, "--", relPath)
	} else {
		// Priority 3: Branch comparison (current behavior)
		baseBranch := "main"
		if bb, ok := args["base_branch"].(string); ok && bb != "" {
			baseBranch = bb
		}
		// If base_branch is "HEAD" or empty, compare working directory
		if baseBranch == "HEAD" || baseBranch == "" {
			cmd = exec.Command("git", "diff", "HEAD", "--", relPath)
		} else {
			cmd = exec.Command("git", "diff", baseBranch, "--", relPath)
		}
	}

	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

func toolGetCommitHistory(args map[string]interface{}) (string, error) {
	filePath, ok := args["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path is required")
	}

	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	fullPath := resolvePath(filePath)
	relPath, _ := filepath.Rel(repoPath, fullPath)

	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", limit), "--pretty=format:%H|%an|%ae|%ad|%s", "--date=iso", "--", relPath)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get commit history: %w\nOutput: %s", err, string(output))
	}

	outputStr := strings.TrimSpace(string(output))
	var commits []string
	if outputStr != "" {
		commits = strings.Split(outputStr, "\n")
	}

	var result []map[string]interface{}
	for _, commit := range commits {
		if commit == "" {
			continue
		}
		parts := strings.SplitN(commit, "|", 5)
		if len(parts) == 5 {
			result = append(result, map[string]interface{}{
				"hash":    parts[0],
				"author":  parts[1],
				"email":   parts[2],
				"date":    parts[3],
				"message": parts[4],
			})
		}
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal commit history: %w", err)
	}
	return string(jsonResult), nil
}

func toolGetChangedFiles(args map[string]interface{}) (string, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	comparisonType, ok := args["comparison_type"].(string)
	if !ok {
		return "", fmt.Errorf("comparison_type is required (branch, commits, working, last_commit)")
	}

	includeStatus := true
	if is, ok := args["include_status"].(bool); ok {
		includeStatus = is
	}

	var cmd *exec.Cmd

	switch comparisonType {
	case "working":
		// Use git status for uncommitted changes
		cmd = exec.Command("git", "status", "--porcelain")
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get git status: %w\nOutput: %s", err, string(output))
		}

		// Parse porcelain output
		var result []map[string]interface{}
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Porcelain format: XY filename
			// X = staged status, Y = unstaged status
			// M = modified, A = added, D = deleted, etc.
			if len(line) >= 3 {
				status := string(line[0])
				if status == " " {
					status = string(line[1])
				}
				filePath := strings.TrimSpace(line[2:])
				entry := map[string]interface{}{
					"file_path": filePath,
				}
				if includeStatus {
					entry["status"] = status
				}
				result = append(result, entry)
			}
		}

		jsonResult, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal changed files: %w", err)
		}
		return string(jsonResult), nil

	case "branch":
		baseBranch, ok := args["base_branch"].(string)
		if !ok || baseBranch == "" {
			return "", fmt.Errorf("base_branch is required for branch comparison")
		}

		targetBranch := ""
		if tb, ok := args["target_branch"].(string); ok && tb != "" {
			targetBranch = tb
		} else {
			// Get current branch
			cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
			cmd.Dir = repoPath
			output, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("failed to get current branch: %w\nOutput: %s", err, string(output))
			}
			targetBranch = strings.TrimSpace(string(output))
		}

		if includeStatus {
			cmd = exec.Command("git", "diff", "--name-status", fmt.Sprintf("%s..%s", baseBranch, targetBranch))
		} else {
			cmd = exec.Command("git", "diff", "--name-only", fmt.Sprintf("%s..%s", baseBranch, targetBranch))
		}
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get branch diff: %w\nOutput: %s", err, string(output))
		}

		return parseDiffNameOutput(string(output), includeStatus)

	case "commits":
		baseCommit, ok := args["base_commit"].(string)
		if !ok || baseCommit == "" {
			return "", fmt.Errorf("base_commit is required for commit comparison")
		}

		targetCommit := "HEAD"
		if tc, ok := args["target_commit"].(string); ok && tc != "" {
			targetCommit = tc
		}

		if includeStatus {
			cmd = exec.Command("git", "diff", "--name-status", baseCommit, targetCommit)
		} else {
			cmd = exec.Command("git", "diff", "--name-only", baseCommit, targetCommit)
		}
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get commit diff: %w\nOutput: %s", err, string(output))
		}

		return parseDiffNameOutput(string(output), includeStatus)

	case "last_commit":
		if includeStatus {
			cmd = exec.Command("git", "diff", "--name-status", "HEAD~1", "HEAD")
		} else {
			cmd = exec.Command("git", "diff", "--name-only", "HEAD~1", "HEAD")
		}
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to get last commit diff: %w\nOutput: %s", err, string(output))
		}

		return parseDiffNameOutput(string(output), includeStatus)

	default:
		return "", fmt.Errorf("invalid comparison_type: %s (must be: branch, commits, working, last_commit)", comparisonType)
	}
}

func parseDiffNameOutput(output string, includeStatus bool) (string, error) {
	var result []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		entry := make(map[string]interface{})
		if includeStatus {
			// Format: STATUS\tfile_path
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				entry["status"] = parts[0]
				entry["file_path"] = parts[1]
			} else {
				entry["file_path"] = line
			}
		} else {
			entry["file_path"] = line
		}
		result = append(result, entry)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal changed files: %w", err)
	}
	return string(jsonResult), nil
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
