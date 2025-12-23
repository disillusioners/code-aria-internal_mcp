package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestHandleInitialize tests the handleInitialize function
func TestHandleInitialize(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantErr      bool
		expectFields []string
	}{
		{
			name:    "Valid initialize request",
			input:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}`,
			wantErr: false,
			expectFields: []string{
				"protocolVersion", "capabilities", "serverInfo",
			},
		},
		{
			name:    "Invalid JSON",
			input:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}`,
			wantErr: true,
		},
		{
			name:    "No initialize request - wrong method",
			input:   `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			wantErr: true, // Should fail because method is not "initialize"
		},
		{
			name:    "Initialize with custom ID",
			input:   `{"jsonrpc":"2.0","id":"test-id","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}`,
			wantErr: false,
			expectFields: []string{
				"protocolVersion", "capabilities", "serverInfo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a scanner with the test input
			// Add initialized notification only if we expect success
			input := tt.input
			if !tt.wantErr {
				input = tt.input + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}`
			}
			scanner := bufio.NewScanner(strings.NewReader(input))

			// Create a buffer to capture output
			var output bytes.Buffer
			encoder := json.NewEncoder(&output)

			err := handleInitialize(scanner, encoder)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleInitialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Parse the response to verify structure
				var response MCPMessage
				if err := json.Unmarshal(output.Bytes(), &response); err != nil {
					t.Errorf("handleInitialize() returned invalid JSON: %v", err)
					return
				}

				// Check response structure
				if response.JSONRPC != "2.0" {
					t.Error("handleInitialize() response missing jsonrpc version")
				}

				if response.Result == nil {
					t.Error("handleInitialize() response missing result")
					return
				}

				// Convert result to map for field checking
				resultMap, ok := response.Result.(map[string]interface{})
				if !ok {
					t.Error("handleInitialize() result is not a map")
					return
				}

				// Check expected fields
				for _, field := range tt.expectFields {
					if _, ok := resultMap[field]; !ok {
						t.Errorf("handleInitialize() response missing field: %s", field)
					}
				}

				// Check server info
				if serverInfo, ok := resultMap["serverInfo"].(map[string]interface{}); ok {
					if name, ok := serverInfo["name"].(string); !ok || name != "mcp-guidelines" {
						t.Error("handleInitialize() incorrect server name")
					}

					if version, ok := serverInfo["version"].(string); !ok || version != "1.0.0" {
						t.Error("handleInitialize() incorrect server version")
					}
				} else {
					t.Error("handleInitialize() response missing serverInfo")
				}
			}
		})
	}
}

// TestHandleToolsList tests the handleToolsList function
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
		t.Fatalf("handleToolsList() returned invalid JSON: %v", err)
	}

	if response.JSONRPC != "2.0" {
		t.Error("handleToolsList() response missing jsonrpc version")
	}

	if response.Result == nil {
		t.Fatal("handleToolsList() response missing result")
	}

	resultMap, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("handleToolsList() result is not a map")
	}

	tools, ok := resultMap["tools"].([]interface{})
	if !ok {
		t.Fatal("handleToolsList() tools is not an array")
	}

	if len(tools) < 3 {
		t.Errorf("handleToolsList() expected at least 3 tools, got %d", len(tools))
	}

	// Check for expected tool names
	expectedTools := []string{"get_guidelines", "get_guideline_content", "search_guidelines"}
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				toolNames[name] = true
			}
		}
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("handleToolsList() missing expected tool: %s", expected)
		}
	}
}

// TestHandleToolCall tests the handleToolCall function with invalid tool names
func TestHandleToolCallInvalidTool(t *testing.T) {
	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"invalid_tool","arguments":{}}`),
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	handleToolCall(msg, encoder)

	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("handleToolCall() returned invalid JSON: %v", err)
	}

	if response.Error == nil {
		t.Error("handleToolCall() should return error for invalid tool")
	}

	if response.Error.Code != -32601 {
		t.Errorf("handleToolCall() error code = %d, want -32601", response.Error.Code)
	}
}

// TestSendError tests the sendError function
func TestSendError(t *testing.T) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	sendError(encoder, 1, -32603, "Test error", map[string]interface{}{"detail": "test"})

	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("sendError() returned invalid JSON: %v", err)
	}

	if response.JSONRPC != "2.0" {
		t.Error("sendError() response missing jsonrpc version")
	}

	if response.Error == nil {
		t.Fatal("sendError() response missing error")
	}

	if response.Error.Code != -32603 {
		t.Errorf("sendError() error code = %d, want -32603", response.Error.Code)
	}

	if response.Error.Message != "Test error" {
		t.Errorf("sendError() error message = %s, want 'Test error'", response.Error.Message)
	}
}









