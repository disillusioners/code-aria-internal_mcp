package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
)

func TestHandleInitialize(t *testing.T) {
	// Create test input
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}`),
	}

	reqJSON, _ := json.Marshal(initReq)
	input := string(reqJSON) + "\n" + `{"jsonrpc":"2.0","method":"initialized"}` + "\n"

	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	err := handleInitialize(scanner, encoder)
	if err != nil {
		t.Fatalf("handleInitialize failed: %v", err)
	}

	// Verify response
	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != 1 {
		t.Errorf("Expected ID 1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	// Verify result structure
	resultMap, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result is not a map")
	}

	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion 2024-11-05, got %v", resultMap["protocolVersion"])
	}

	serverInfo, ok := resultMap["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("serverInfo is not a map")
	}

	if serverInfo["name"] != "mcp-bash" {
		t.Errorf("Expected server name mcp-bash, got %v", serverInfo["name"])
	}
}

func TestHandleToolsList(t *testing.T) {
	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	handleToolsList(msg, encoder)

	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error != nil {
		t.Errorf("Unexpected error: %v", response.Error)
	}

	resultMap, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result is not a map")
	}

	tools, ok := resultMap["tools"].([]interface{})
	if !ok {
		t.Fatalf("tools is not an array")
	}

	if len(tools) == 0 {
		t.Error("Expected at least one tool")
	}

	// Verify apply_operations tool exists
	found := false
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}
		if toolMap["name"] == "apply_operations" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected apply_operations tool not found")
	}
}

func TestHandleToolCall(t *testing.T) {
	tests := []struct {
		name     string
		request  ToolsCallRequest
		wantError bool
	}{
		{
			name: "valid apply_operations",
			request: ToolsCallRequest{
				Name: "apply_operations",
				Arguments: map[string]interface{}{
					"operations": []interface{}{
						map[string]interface{}{
							"type":    "execute_command",
							"command": "echo test",
						},
					},
				},
			},
			wantError: false,
		},
		{
			name: "unknown tool",
			request: ToolsCallRequest{
				Name: "unknown_tool",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramsJSON, _ := json.Marshal(tt.request)
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  paramsJSON,
			}

			var output bytes.Buffer
			encoder := json.NewEncoder(&output)

			handleToolCall(msg, encoder)

			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			hasError := response.Error != nil
			if hasError != tt.wantError {
				t.Errorf("Expected error: %v, got error: %v", tt.wantError, hasError)
			}
		})
	}
}

func TestHandleBatchOperations(t *testing.T) {
	tests := []struct {
		name        string
		operations  []interface{}
		wantError   bool
		expectCount int
	}{
		{
			name: "single valid operation",
			operations: []interface{}{
				map[string]interface{}{
					"type":    "execute_command",
					"command": "echo hello",
				},
			},
			wantError:   false,
			expectCount: 1,
		},
		{
			name: "multiple operations",
			operations: []interface{}{
				map[string]interface{}{
					"type":    "execute_command",
					"command": "echo first",
				},
				map[string]interface{}{
					"type":    "execute_command",
					"command": "echo second",
				},
			},
			wantError:   false,
			expectCount: 2,
		},
		{
			name:        "empty operations",
			operations:  []interface{}{},
			wantError:   true,
			expectCount: 0,
		},
		{
			name: "invalid operation format",
			operations: []interface{}{
				"not a map",
			},
			wantError:   false,
			expectCount: 1,
		},
		{
			name: "missing operation type",
			operations: []interface{}{
				map[string]interface{}{
					"command": "echo test",
				},
			},
			wantError:   false,
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
			}

			var output bytes.Buffer
			encoder := json.NewEncoder(&output)

			args := map[string]interface{}{
				"operations": tt.operations,
			}

			handleBatchOperations(msg, encoder, args)

			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.wantError {
				if response.Error == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if response.Error != nil {
				t.Errorf("Unexpected error: %v", response.Error)
				return
			}

			resultMap, ok := response.Result.(map[string]interface{})
			if !ok {
				t.Fatalf("Result is not a map")
			}

			results, ok := resultMap["results"].([]interface{})
			if !ok {
				t.Fatalf("results is not an array")
			}

			if len(results) != tt.expectCount {
				t.Errorf("Expected %d results, got %d", tt.expectCount, len(results))
			}
		})
	}
}

func TestSendError(t *testing.T) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	sendError(encoder, 1, -32601, "Test error", "test data")

	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Error == nil {
		t.Error("Expected error but got none")
		return
	}

	if response.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", response.Error.Code)
	}

	if response.Error.Message != "Test error" {
		t.Errorf("Expected error message 'Test error', got '%s'", response.Error.Message)
	}
}

func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantError bool
	}{
		{
			name:      "tools/list",
			method:    "tools/list",
			wantError: false,
		},
		{
			name:      "tools/call",
			method:    "tools/call",
			wantError: false,
		},
		{
			name:      "unknown method",
			method:    "unknown/method",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  tt.method,
			}

			var output bytes.Buffer
			encoder := json.NewEncoder(&output)

			handleRequest(msg, encoder)

			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			hasError := response.Error != nil
			if hasError != tt.wantError {
				t.Errorf("Expected error: %v, got error: %v", tt.wantError, hasError)
			}
		})
	}
}



