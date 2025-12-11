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
		name          string
		input         string
		wantErr       bool
		expectFields  []string
	}{
		{
			name: "Valid initialize request",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}`,
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
			name:    "No initialize request",
			input:   `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			wantErr: true,
		},
		{
			name: "Initialize with custom ID",
			input: `{"jsonrpc":"2.0","id":"test-id","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}`,
			wantErr: false,
			expectFields: []string{
				"protocolVersion", "capabilities", "serverInfo",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a scanner with the test input
			scanner := bufio.NewScanner(strings.NewReader(tt.input + "\n" + `{"jsonrpc":"2.0","method":"notifications/initialized"}`))
			
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
					if name, ok := serverInfo["name"].(string); !ok || name != "mcp-systeminfo" {
						t.Error("handleInitialize() incorrect server name")
					}
					
					if version, ok := serverInfo["version"].(string); !ok || version != "1.0.0" {
						t.Error("handleInitialize() incorrect server version")
					}
				} else {
					t.Error("handleInitialize() missing or invalid serverInfo")
				}
			}
		})
	}
}

// TestHandleRequest tests the handleRequest function
func TestHandleRequest(t *testing.T) {
	tests := []struct {
		name    string
		msg     *MCPMessage
		wantErr bool
	}{
		{
			name: "Tools list request",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/list",
			},
			wantErr: false,
		},
		{
			name: "Tool call request",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"apply_operations","arguments":{"operations":[{"type":"get_system_info"}]}}`),
			},
			wantErr: false,
		},
		{
			name: "Unknown method",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "unknown/method",
			},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Missing method",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      4,
			},
			wantErr: false, // Should be ignored, not return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			encoder := json.NewEncoder(&output)
			
			// Capture the original function to restore it later
			// We can't easily mock the internal functions, so we'll test the overall behavior
			
			handleRequest(tt.msg, encoder)
			
			// Check that something was written to output
			if output.Len() == 0 && tt.msg.Method != "" {
				t.Error("handleRequest() wrote no output for valid request")
			}
			
			// Parse the response to verify it's valid JSON
			if output.Len() > 0 {
				var response MCPMessage
				if err := json.Unmarshal(output.Bytes(), &response); err != nil {
					t.Errorf("handleRequest() returned invalid JSON: %v", err)
				}
				
				// Check that ID is preserved
				if response.ID != tt.msg.ID {
					t.Errorf("handleRequest() ID mismatch: got %v, want %v", response.ID, tt.msg.ID)
				}
				
				// Check JSON-RPC version
				if response.JSONRPC != "2.0" {
					t.Error("handleRequest() response missing jsonrpc version")
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
	
	// Parse the response
	var response MCPMessage
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Errorf("handleToolsList() returned invalid JSON: %v", err)
		return
	}
	
	// Check response structure
	if response.JSONRPC != "2.0" {
		t.Error("handleToolsList() response missing jsonrpc version")
	}
	
	if response.ID != msg.ID {
		t.Errorf("handleToolsList() ID mismatch: got %v, want %v", response.ID, msg.ID)
	}
	
	if response.Result == nil {
		t.Error("handleToolsList() response missing result")
		return
	}
	
	// Convert result to ToolsListResponse
	resultBytes, _ := json.Marshal(response.Result)
	var toolsListResponse ToolsListResponse
	if err := json.Unmarshal(resultBytes, &toolsListResponse); err != nil {
		t.Errorf("handleToolsList() invalid result format: %v", err)
		return
	}
	
	// Check that tools are returned
	if len(toolsListResponse.Tools) == 0 {
		t.Error("handleToolsList() no tools returned")
	}
	
	// Check for expected tool
	found := false
	for _, tool := range toolsListResponse.Tools {
		if tool.Name == "apply_operations" {
			found = true
			
			// Check tool description
			if tool.Description == "" {
				t.Error("handleToolsList() tool missing description")
			}
			
			// Check input schema
			if tool.InputSchema == nil {
				t.Error("handleToolsList() tool missing input schema")
			}
			
			// Check schema structure
			if schema, ok := tool.InputSchema.(map[string]interface{}); ok {
				if _, ok := schema["type"]; !ok {
					t.Error("handleToolsList() input schema missing type")
				}
				
				if _, ok := schema["properties"]; !ok {
					t.Error("handleToolsList() input schema missing properties")
				}
				
				if _, ok := schema["required"]; !ok {
					t.Error("handleToolsList() input schema missing required")
				}
			} else {
				t.Error("handleToolsList() input schema is not a map")
			}
			
			break
		}
	}
	
	if !found {
		t.Error("handleToolsList() apply_operations tool not found")
	}
}

// TestHandleToolCall tests the handleToolCall function
func TestHandleToolCall(t *testing.T) {
	tests := []struct {
		name    string
		msg     *MCPMessage
		wantErr bool
	}{
		{
			name: "Valid apply_operations call",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"apply_operations","arguments":{"operations":[{"type":"get_system_info"}]}}`),
			},
			wantErr: false,
		},
		{
			name: "Invalid params",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"apply_operations","arguments":{}}`),
			},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Unknown tool",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"unknown_tool","arguments":{}}`),
			},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Invalid JSON params",
			msg: &MCPMessage{
				JSONRPC: "2.0",
				ID:      4,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"apply_operations","arguments":{`),
			},
			wantErr: false, // Should send error response, not return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			encoder := json.NewEncoder(&output)
			
			handleToolCall(tt.msg, encoder)
			
			// Parse the response
			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Errorf("handleToolCall() returned invalid JSON: %v", err)
				return
			}
			
			// Check response structure
			if response.JSONRPC != "2.0" {
				t.Error("handleToolCall() response missing jsonrpc version")
			}
			
			if response.ID != tt.msg.ID {
				t.Errorf("handleToolCall() ID mismatch: got %v, want %v", response.ID, tt.msg.ID)
			}
			
			// Should have either result or error
			if response.Result == nil && response.Error == nil {
				t.Error("handleToolCall() response missing both result and error")
			}
		})
	}
}

// TestHandleBatchOperations tests the handleBatchOperations function
func TestHandleBatchOperations(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid operations",
			args: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "get_system_info",
					},
					map[string]interface{}{
						"type": "get_os_info",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing operations",
			args: map[string]interface{}{},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Empty operations",
			args: map[string]interface{}{
				"operations": []interface{}{},
			},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Invalid operation format",
			args: map[string]interface{}{
				"operations": []interface{}{
					"invalid_operation",
				},
			},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Missing operation type",
			args: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"params": map[string]interface{}{},
					},
				},
			},
			wantErr: false, // Should send error response, not return error
		},
		{
			name: "Unknown operation type",
			args: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "unknown_operation",
					},
				},
			},
			wantErr: false, // Should send error response, not return error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
			}

			var output bytes.Buffer
			encoder := json.NewEncoder(&output)
			
			handleBatchOperations(msg, encoder, tt.args)
			
			// Parse the response
			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Errorf("handleBatchOperations() returned invalid JSON: %v", err)
				return
			}
			
			// Check response structure
			if response.JSONRPC != "2.0" {
				t.Error("handleBatchOperations() response missing jsonrpc version")
			}
			
			if response.ID != msg.ID {
				t.Errorf("handleBatchOperations() ID mismatch: got %v, want %v", response.ID, msg.ID)
			}
			
			// Should have result
			if response.Result == nil {
				t.Error("handleBatchOperations() response missing result")
				return
			}
			
			// Check result structure
			if resultMap, ok := response.Result.(map[string]interface{}); ok {
				if _, ok := resultMap["results"]; !ok {
					t.Error("handleBatchOperations() result missing results field")
				}
			} else {
				t.Error("handleBatchOperations() result is not a map")
			}
		})
	}
}

// TestSendError tests the sendError function
func TestSendError(t *testing.T) {
	tests := []struct {
		name    string
		id      interface{}
		code    int
		message string
		data    interface{}
	}{
		{
			name:    "Error with numeric ID",
			id:      1,
			code:    -32601,
			message: "Method not found",
			data:    nil,
		},
		{
			name:    "Error with string ID",
			id:      "test-id",
			code:    -32602,
			message: "Invalid params",
			data:    map[string]interface{}{"field": "value"},
		},
		{
			name:    "Error with no data",
			id:      nil,
			code:    -32603,
			message: "Internal error",
			data:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			encoder := json.NewEncoder(&output)
			
			sendError(encoder, tt.id, tt.code, tt.message, tt.data)
			
			// Parse the response
			var response MCPMessage
			if err := json.Unmarshal(output.Bytes(), &response); err != nil {
				t.Errorf("sendError() returned invalid JSON: %v", err)
				return
			}
			
			// Check response structure
			if response.JSONRPC != "2.0" {
				t.Error("sendError() response missing jsonrpc version")
			}
			
			if response.ID != tt.id {
				t.Errorf("sendError() ID mismatch: got %v, want %v", response.ID, tt.id)
			}
			
			if response.Result != nil {
				t.Error("sendError() response should not have result")
			}
			
			if response.Error == nil {
				t.Error("sendError() response missing error")
				return
			}
			
			// Check error structure
			if response.Error.Code != tt.code {
				t.Errorf("sendError() code mismatch: got %v, want %v", response.Error.Code, tt.code)
			}
			
			if response.Error.Message != tt.message {
				t.Errorf("sendError() message mismatch: got %v, want %v", response.Error.Message, tt.message)
			}
			
			if (response.Error.Data != nil) != (tt.data != nil) {
				t.Errorf("sendError() data mismatch: got %v, want %v", response.Error.Data, tt.data)
			}
		})
	}
}

// TestMCPMessageSerialization tests MCPMessage JSON serialization
func TestMCPMessageSerialization(t *testing.T) {
	tests := []struct {
		name string
		msg  MCPMessage
	}{
		{
			name: "Request message",
			msg: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/list",
			},
		},
		{
			name: "Response message",
			msg: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Result: map[string]interface{}{
					"tools": []Tool{
						{
							Name:        "test_tool",
							Description: "Test tool",
							InputSchema: map[string]interface{}{
								"type": "object",
							},
						},
					},
				},
			},
		},
		{
			name: "Error message",
			msg: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Error: &MCPError{
					Code:    -32601,
					Message: "Method not found",
				},
			},
		},
		{
			name: "Notification message",
			msg: MCPMessage{
				JSONRPC: "2.0",
				Method:  "notifications/initialized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(tt.msg)
			if err != nil {
				t.Errorf("MCPMessage serialization error: %v", err)
				return
			}
			
			// Deserialize back
			var msg2 MCPMessage
			if err := json.Unmarshal(jsonBytes, &msg2); err != nil {
				t.Errorf("MCPMessage deserialization error: %v", err)
				return
			}
			
			// Check that key fields match
			if msg2.JSONRPC != tt.msg.JSONRPC {
				t.Errorf("MCPMessage JSONRPC mismatch: got %v, want %v", msg2.JSONRPC, tt.msg.JSONRPC)
			}
			
			if msg2.ID != tt.msg.ID {
				t.Errorf("MCPMessage ID mismatch: got %v, want %v", msg2.ID, tt.msg.ID)
			}
			
			if msg2.Method != tt.msg.Method {
				t.Errorf("MCPMessage Method mismatch: got %v, want %v", msg2.Method, tt.msg.Method)
			}
			
			// For result and error, just check that they're both nil or both non-nil
			if (msg2.Result == nil) != (tt.msg.Result == nil) {
				t.Error("MCPMessage Result nil mismatch")
			}
			
			if (msg2.Error == nil) != (tt.msg.Error == nil) {
				t.Error("MCPMessage Error nil mismatch")
			}
		})
	}
}

// TestMCPErrorSerialization tests MCPError JSON serialization
func TestMCPErrorSerialization(t *testing.T) {
	tests := []struct {
		name string
		err  MCPError
	}{
		{
			name: "Error with only code and message",
			err: MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		},
		{
			name: "Error with data",
			err: MCPError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    map[string]interface{}{"field": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize to JSON
			jsonBytes, err := json.Marshal(tt.err)
			if err != nil {
				t.Errorf("MCPError serialization error: %v", err)
				return
			}
			
			// Deserialize back
			var err2 MCPError
			if err := json.Unmarshal(jsonBytes, &err2); err != nil {
				t.Errorf("MCPError deserialization error: %v", err)
				return
			}
			
			// Check that fields match
			if err2.Code != tt.err.Code {
				t.Errorf("MCPError Code mismatch: got %v, want %v", err2.Code, tt.err.Code)
			}
			
			if err2.Message != tt.err.Message {
				t.Errorf("MCPError Message mismatch: got %v, want %v", err2.Message, tt.err.Message)
			}
			
			if (err2.Data != nil) != (tt.err.Data != nil) {
				t.Error("MCPError Data nil mismatch")
			}
		})
	}
}