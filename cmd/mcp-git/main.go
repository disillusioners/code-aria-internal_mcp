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
			Name:        "get_git_status",
			Description: "Get git status for the repository",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the repository",
					},
				},
			},
		},
		{
			Name:        "get_file_diff",
			Description: "Get diff for a file against a base branch",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file",
					},
					"base_branch": map[string]interface{}{
						"type":        "string",
						"description": "Base branch to compare against",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "get_commit_history",
			Description: "Get commit history for a file",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of commits",
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
	case "get_git_status":
		result, err = toolGetGitStatus(req.Arguments)
	case "get_file_diff":
		result, err = toolGetFileDiff(req.Arguments)
	case "get_commit_history":
		result, err = toolGetCommitHistory(req.Arguments)
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

func toolGetGitStatus(args map[string]interface{}) (string, error) {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return "", fmt.Errorf("REPO_PATH not set")
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
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

	baseBranch := "main"
	if bb, ok := args["base_branch"].(string); ok && bb != "" {
		baseBranch = bb
	}

	fullPath := resolvePath(filePath)
	relPath, _ := filepath.Rel(repoPath, fullPath)

	cmd := exec.Command("git", "diff", baseBranch, "--", relPath)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
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
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit history: %w", err)
	}

	commits := strings.Split(strings.TrimSpace(string(output)), "\n")
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

	jsonResult, _ := json.Marshal(result)
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

