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
			input:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`,
			wantErr: false,
			expectFields: []string{
				"protocolVersion", "capabilities", "serverInfo",
			},
		},
		{
			name:    "Invalid JSON",
			input:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"`,
			wantErr: true,
		},
		{
			name:    "Initialize with custom ID",
			input:   `{"jsonrpc":"2.0","id":"test-id","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`,
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

				// Verify server info
				resultMap, ok := response.Result.(map[string]interface{})
				if !ok {
					// Try to unmarshal as InitializeResponse
					var initResp InitializeResponse
					resultBytes, _ := json.Marshal(response.Result)
					if err := json.Unmarshal(resultBytes, &initResp); err == nil {
						if initResp.ServerInfo.Name != "mcp-postgres" {
							t.Errorf("Server name = %v, want mcp-postgres", initResp.ServerInfo.Name)
						}
					}
					return
				}

				serverInfo, ok := resultMap["serverInfo"].(map[string]interface{})
				if !ok {
					t.Error("handleInitialize() response missing serverInfo")
					return
				}

				if name, ok := serverInfo["name"].(string); !ok || name != "mcp-postgres" {
					t.Errorf("Server name = %v, want mcp-postgres", name)
				}
			}
		})
	}
}

// TestHandleToolsList tests the tools/list handler
func TestHandleToolsList(t *testing.T) {
	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	handleToolsList(msg, encoder)

	// Parse response
	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.JSONRPC != "2.0" {
		t.Error("Response missing jsonrpc version")
	}

	if response.Result == nil {
		t.Fatal("Response missing result")
	}

	// Verify tools list
	resultBytes, _ := json.Marshal(response.Result)
	var toolsResp ToolsListResponse
	if err := json.Unmarshal(resultBytes, &toolsResp); err != nil {
		t.Fatalf("Failed to parse tools list: %v", err)
	}

	if len(toolsResp.Tools) == 0 {
		t.Error("Expected at least one tool, got none")
	}

	// Verify apply_operations tool exists
	found := false
	for _, tool := range toolsResp.Tools {
		if tool.Name == "apply_operations" {
			found = true
			if tool.Description == "" {
				t.Error("Tool missing description")
			}
			if tool.InputSchema == nil {
				t.Error("Tool missing input schema")
			}
			break
		}
	}

	if !found {
		t.Error("Expected 'apply_operations' tool, not found")
	}
}

// TestHandleToolCall tests the tool call handler
func TestHandleToolCall(t *testing.T) {
	tests := []struct {
		name    string
		request ToolsCallRequest
		wantErr bool
	}{
		{
			name: "Valid apply_operations call",
			request: ToolsCallRequest{
				Name: "apply_operations",
				Arguments: map[string]interface{}{
					"operations": []interface{}{
						map[string]interface{}{
							"type": "list_schemas",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Unknown tool",
			request: ToolsCallRequest{
				Name: "unknown_tool",
			},
			wantErr: true,
		},
		{
			name: "Empty operations array",
			request: ToolsCallRequest{
				Name: "apply_operations",
				Arguments: map[string]interface{}{
					"operations": []interface{}{},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing operations",
			request: ToolsCallRequest{
				Name:      "apply_operations",
				Arguments: map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  mustMarshal(tt.request),
			}

			var output bytes.Buffer
			encoder := json.NewEncoder(&output)

			handleToolCall(msg, encoder)

			// Parse response
			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if tt.wantErr {
				if response.Error == nil {
					t.Error("Expected error response, got none")
				} else {
					t.Logf("Got expected error: %v", response.Error.Message)
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %v", response.Error.Message)
				}
				if response.Result == nil {
					t.Error("Expected result, got none")
				}
			}
		})
	}
}

// TestHandleBatchOperations tests batch operation processing
func TestHandleBatchOperations(t *testing.T) {
	tests := []struct {
		name       string
		operations []interface{}
		wantErr    bool
	}{
		{
			name: "Single valid operation",
			operations: []interface{}{
				map[string]interface{}{
					"type": "list_schemas",
				},
			},
			wantErr: false,
		},
		{
			name: "Multiple operations",
			operations: []interface{}{
				map[string]interface{}{
					"type": "list_schemas",
				},
				map[string]interface{}{
					"type": "list_tables",
					"schema": "public",
				},
			},
			wantErr: false,
		},
		{
			name: "Operation with invalid type",
			operations: []interface{}{
				map[string]interface{}{
					"type": "invalid_operation",
				},
			},
			wantErr: false, // Should return error in result, not fail completely
		},
		{
			name: "Operation missing type",
			operations: []interface{}{
				map[string]interface{}{
					"schema": "public",
				},
			},
			wantErr: false, // Should return error in result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
			}

			args := map[string]interface{}{
				"operations": tt.operations,
			}

			var output bytes.Buffer
			encoder := json.NewEncoder(&output)

			handleBatchOperations(msg, encoder, args)

			// Parse response
			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response.Error != nil {
				if !tt.wantErr {
					t.Errorf("Unexpected error: %v", response.Error.Message)
				}
				return
			}

			if response.Result == nil {
				t.Fatal("Expected result, got none")
			}

			// Verify result structure
			resultMap, ok := response.Result.(map[string]interface{})
			if !ok {
				t.Fatal("Result is not a map")
			}

			results, ok := resultMap["results"].([]interface{})
			if !ok {
				t.Fatal("Results is not an array")
			}

			if len(results) != len(tt.operations) {
				t.Errorf("Expected %d results, got %d", len(tt.operations), len(results))
			}
		})
	}
}

// TestSendError tests error response generation
func TestSendError(t *testing.T) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)

	sendError(encoder, 1, -32601, "Test error", map[string]interface{}{"data": "test"})

	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if response.Error.Code != -32601 {
		t.Errorf("Error code = %d, want -32601", response.Error.Code)
	}

	if response.Error.Message != "Test error" {
		t.Errorf("Error message = %s, want 'Test error'", response.Error.Message)
	}
}

// TestOptimizeParams tests parameter optimization
func TestOptimizeParams(t *testing.T) {
	tests := []struct {
		name     string
		opType   string
		params   map[string]interface{}
		validate func(t *testing.T, optimized map[string]interface{})
	}{
		{
			name:   "Preserve metadata fields",
			opType: "list_tables",
			params: map[string]interface{}{
				"schema":            "public",
				"connection_string": "postgres://test",
			},
			validate: func(t *testing.T, optimized map[string]interface{}) {
				if _, ok := optimized["schema"]; !ok {
					t.Error("Schema should be preserved")
				}
				if _, ok := optimized["connection_string"]; ok {
					t.Error("Connection string should be excluded")
				}
			},
		},
		{
			name:   "Truncate long query strings",
			opType: "query",
			params: map[string]interface{}{
				"query": strings.Repeat("SELECT * FROM test;\n", 30),
			},
			validate: func(t *testing.T, optimized map[string]interface{}) {
				query, ok := optimized["query"].(string)
				if !ok {
					t.Fatal("Query should be a string")
				}
				lines := strings.Split(query, "\n")
				if len(lines) > 25 {
					t.Errorf("Query should be truncated, got %d lines", len(lines))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimized := optimizeParams(tt.opType, tt.params)
			tt.validate(t, optimized)
		})
	}
}

// Helper function to marshal JSON
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}

