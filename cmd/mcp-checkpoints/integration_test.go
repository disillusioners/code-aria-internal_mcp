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
		"create_checkpoint",
		"list_checkpoints",
		"get_checkpoint",
		"restore_checkpoint",
		"delete_checkpoint",
		"get_checkpoint_info",
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

func TestMCPServerCreateCheckpoint(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mcp-create-checkpoint-test")
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

	// Test create_checkpoint tool call
	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
	}

	// Marshal the params - MCP expects name and arguments directly in params
	params, err := json.Marshal(map[string]interface{}{
		"name": "create_checkpoint",
		"arguments": map[string]interface{}{
			"name":        "integration-test",
			"description": "Integration test checkpoint",
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
		t.Fatal("No response from create_checkpoint")
	}

	var toolCallResponse MCPMessage
	if err := json.Unmarshal(outputBytes, &toolCallResponse); err != nil {
		t.Fatalf("Failed to parse create_checkpoint response: %v", err)
	}

	if toolCallResponse.Result == nil {
		t.Fatal("No result in create_checkpoint response")
	}

	// Parse the tool result
	result, ok := toolCallResponse.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Invalid create_checkpoint response format")
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

	// Parse the checkpoint JSON
	var checkpoint Checkpoint
	if err := json.Unmarshal([]byte(text), &checkpoint); err != nil {
		t.Fatalf("Failed to parse checkpoint: %v", err)
	}

	if checkpoint.Name != "integration-test" {
		t.Errorf("Expected name 'integration-test', got '%s'", checkpoint.Name)
	}

	if checkpoint.Description != "Integration test checkpoint" {
		t.Errorf("Expected description 'Integration test checkpoint', got '%s'", checkpoint.Description)
	}

	if len(checkpoint.ID) != 8 {
		t.Errorf("Expected 8-character ID, got %d characters: %s", len(checkpoint.ID), checkpoint.ID)
	}

	if len(checkpoint.Files) == 0 {
		t.Error("Expected at least one file in checkpoint")
	}

	// Verify checkpoint was actually created on disk
	checkpointDir := filepath.Join(tempDir, ".mcp-checkpoints", checkpoint.ID)
	if _, err := os.Stat(checkpointDir); os.IsNotExist(err) {
		t.Errorf("Checkpoint directory does not exist: %s", checkpointDir)
	}

	// Verify file was copied to checkpoint
	checkpointFile := filepath.Join(checkpointDir, "test.txt")
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		t.Errorf("Checkpoint file does not exist: %s", checkpointFile)
	}

	// Verify file content
	contentData, err := os.ReadFile(checkpointFile)
	if err != nil {
		t.Fatalf("Failed to read checkpoint file: %v", err)
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

	// Test create_checkpoint without REPO_PATH
	params, _ := json.Marshal(map[string]interface{}{
		"name": "create_checkpoint",
		"arguments": map[string]interface{}{
			"name": "test-checkpoint",
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
		t.Fatal("No response from create_checkpoint")
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
		"name": "invalid_tool",
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