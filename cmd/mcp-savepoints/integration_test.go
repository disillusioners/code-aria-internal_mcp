package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
)

func TestMCPServerToolsList(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcp-integration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository
	_, err = git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	// Test tools/list request directly
	toolsListReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	// Process tools/list request directly
	handleRequest(&toolsListReq, encoder)

	// Parse the response
	outputBytes := output.Bytes()
	if len(outputBytes) == 0 {
		t.Fatal("No response from tools/list")
	}

	var toolsListResponse MCPMessage
	if err := json.Unmarshal(outputBytes, &toolsListResponse); err != nil {
		t.Fatalf("Failed to parse tools/list response: %v", err)
	}

	if toolsListResponse.Result == nil {
		t.Fatal("No result in tools/list response")
	}

	// Parse the tools list
	result, ok := toolsListResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Invalid tools/list response format")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("No tools found in response")
	}

	expectedTools := []string{
		"create_savepoint",
		"list_savepoints",
		"get_savepoint",
		"restore_savepoint",
		"delete_savepoint",
		"get_savepoint_info",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Verify all expected tools are present
	foundTools := make(map[string]bool)
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := toolMap["name"].(string)
		if !ok {
			continue
		}

		foundTools[name] = true
	}

	for _, expectedTool := range expectedTools {
		if !foundTools[expectedTool] {
			t.Errorf("Expected tool '%s' not found", expectedTool)
		}
	}
}

func TestMCPServerCreateSavepoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcp-create-savepoint-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repository and create changes
	repo, err := git.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	os.Setenv("REPO_PATH", tempDir)
	defer os.Unsetenv("REPO_PATH")

	// Test create_savepoint tool call
	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
	}

	// Marshal the params - MCP expects name and arguments directly in params
	params, err := json.Marshal(map[string]interface{}{
		"name": "create_savepoint",
		"arguments": map[string]interface{}{
			"name":        "integration-test",
			"description": "Integration test savepoint",
		},
	})
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	toolCallReq.Params = params

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	handleRequest(&toolCallReq, encoder)

	// Parse the response
	outputBytes := output.Bytes()
	if len(outputBytes) == 0 {
		t.Fatal("No response from create_savepoint")
	}

	var toolCallResponse MCPMessage
	if err := json.Unmarshal(outputBytes, &toolCallResponse); err != nil {
		t.Fatalf("Failed to parse create_savepoint response: %v", err)
	}

	if toolCallResponse.Result == nil {
		t.Fatal("No result in create_savepoint response")
	}

	// Parse the tool result
	result, ok := toolCallResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Invalid create_savepoint response format")
	}

	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatal("No content in response")
	}

	if len(content) == 0 {
		t.Fatal("Empty content in response")
	}

	contentItem, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("Invalid content item format")
	}

	text, ok := contentItem["text"].(string)
	if !ok {
		t.Fatal("No text in content item")
	}

	// Parse the savepoint JSON
	var savepoint Savepoint
	if err := json.Unmarshal([]byte(text), &savepoint); err != nil {
		t.Fatalf("Failed to parse savepoint: %v", err)
	}

	if savepoint.Name != "integration-test" {
		t.Errorf("Expected name 'integration-test', got '%s'", savepoint.Name)
	}

	if savepoint.Description != "Integration test savepoint" {
		t.Errorf("Expected description 'Integration test savepoint', got '%s'", savepoint.Description)
	}

	if len(savepoint.ID) != 8 {
		t.Errorf("Expected 8-character ID, got %d characters: %s", len(savepoint.ID), savepoint.ID)
	}

	if len(savepoint.Files) == 0 {
		t.Error("Expected at least one file in savepoint")
	}

	// Verify savepoint was actually created on disk
	savepointDir := filepath.Join(tempDir, ".mcp-savepoints", savepoint.ID)
	if _, err := os.Stat(savepointDir); os.IsNotExist(err) {
		t.Errorf("Savepoint directory does not exist: %s", savepointDir)
	}

	// Verify file was copied to savepoint
	savepointFile := filepath.Join(savepointDir, "test.txt")
	if _, err := os.Stat(savepointFile); os.IsNotExist(err) {
		t.Errorf("Savepoint file does not exist: %s", savepointFile)
	}

	// Verify file content
	contentData, err := os.ReadFile(savepointFile)
	if err != nil {
		t.Fatalf("Failed to read savepoint file: %v", err)
	}

	expectedContent := "test content"
	if string(contentData) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(contentData))
	}
}

func TestMCPServerErrorHandling(t *testing.T) {
	// Test missing REPO_PATH
	originalRepoPath := os.Getenv("REPO_PATH")
	os.Unsetenv("REPO_PATH")
	defer os.Setenv("REPO_PATH", originalRepoPath)

	// Test create_savepoint without REPO_PATH
	params, _ := json.Marshal(map[string]interface{}{
		"name": "create_savepoint",
		"arguments": map[string]interface{}{
			"name": "test-savepoint",
		},
	})

	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  params,
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	handleRequest(&toolCallReq, encoder)

	// Parse the error response
	outputBytes := output.Bytes()
	if len(outputBytes) == 0 {
		t.Fatal("No response from create_savepoint")
	}

	var errorResponse MCPMessage
	if err := json.Unmarshal(outputBytes, &errorResponse); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if errorResponse.Error == nil {
		t.Fatal("Expected error response, got success")
	}

	if errorResponse.Error.Code != -32603 {
		t.Errorf("Expected error code -32603, got %d", errorResponse.Error.Code)
	}

	if errorResponse.Error.Message == "" {
		t.Error("Expected error message, got empty")
	}

	// Test invalid tool name
	invalidParams, _ := json.Marshal(map[string]interface{}{
		"name":      "invalid_tool",
		"arguments": map[string]interface{}{},
	})

	invalidToolReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  invalidParams,
	}

	output.Reset()
	encoder = json.NewEncoder(&output)

	handleRequest(&invalidToolReq, encoder)

	// Parse the error response
	outputBytes = output.Bytes()
	if len(outputBytes) == 0 {
		t.Fatal("No response from invalid tool")
	}

	errorResponse = MCPMessage{}
	if err := json.Unmarshal(outputBytes, &errorResponse); err != nil {
		t.Fatalf("Failed to parse invalid tool error response: %v", err)
	}

	if errorResponse.Error == nil {
		t.Fatal("Expected error response for invalid tool, got success")
	}
}
