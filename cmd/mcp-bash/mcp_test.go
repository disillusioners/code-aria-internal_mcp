package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

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
		name      string
		request   ToolsCallRequest
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
