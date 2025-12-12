package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestHandleInitialize tests the MCP initialization handshake
func TestHandleInitialize(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedError bool
	}{
		{
			name: "valid initialize request and notification",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized"}`,
			expectedError: false,
		},
		{
			name: "invalid JSON",
			input: `{"invalid json`,
			expectedError: true,
		},
		{
			name:          "empty input",
			input:         "",
			expectedError: true,
		},
		{
			name: "missing initialized notification",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := newTestScanner(tt.input)
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)

			err := handleInitialize(scanner, encoder)

			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestHandleToolsList tests the tools list endpoint
func TestHandleToolsList(t *testing.T) {
	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	handleToolsList(msg, encoder)

	var response MCPMessage
	if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc version '2.0', got '%s'", response.JSONRPC)
	}

	toolsList, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", response.Result)
	}

	tools, ok := toolsList["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools to be an array, got %T", toolsList["tools"])
	}

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	tool := tools[0].(map[string]interface{})
	if tool["name"] != "apply_operations" {
		t.Errorf("Expected tool name 'apply_operations', got '%s'", tool["name"])
	}

	if tool["description"] != "Execute multiple Go language operations in a single batch call" {
		t.Errorf("Unexpected tool description: %s", tool["description"])
	}
}

// TestHandleToolCall tests the tool call endpoint
func TestHandleToolCall(t *testing.T) {
	tests := []struct {
		name          string
		msg           MCPMessage
		expectedError bool
	}{
		{
			name: "valid apply_operations call",
			msg: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name": "apply_operations", "arguments": {"operations": [{"type": "lint", "target": "."}]}}`),
			},
			expectedError: false,
		},
		{
			name: "unknown tool",
			msg: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name": "unknown_tool", "arguments": {}}`),
			},
			expectedError: true,
		},
		{
			name: "invalid params JSON",
			msg: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "tools/call",
				Params:  json.RawMessage(`{"invalid json`),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)

			handleToolCall(&tt.msg, encoder)

			var response MCPMessage
			if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectedError {
				if response.Error == nil {
					t.Error("Expected error response but got success")
				}
			} else {
				if response.Error != nil {
					t.Errorf("Unexpected error: %s", response.Error.Message)
				}
			}
		})
	}
}

// TestHandleBatchOperations tests batch operation processing
func TestHandleBatchOperations(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectedOps   int
		expectedError bool
	}{
		{
			name: "valid operations",
			args: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "lint",
						"target": ".",
					},
					map[string]interface{}{
						"type":   "lint",
						"target": "main.go",
					},
				},
			},
			expectedOps:   2,
			expectedError: false,
		},
		{
			name: "empty operations array",
			args: map[string]interface{}{
				"operations": []interface{}{},
			},
			expectedError: true,
		},
		{
			name: "missing operations field",
			args: map[string]interface{}{
				"other": "value",
			},
			expectedError: true,
		},
		{
			name: "invalid operation format",
			args: map[string]interface{}{
				"operations": []interface{}{
					"invalid operation",
				},
			},
			expectedOps:   1,
			expectedError: false, // Should handle gracefully with error result
		},
		{
			name: "unknown operation type",
			args: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"type": "unknown",
					},
				},
			},
			expectedOps:   1,
			expectedError: false, // Should handle gracefully with error result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
			}
			var buf bytes.Buffer
			encoder := json.NewEncoder(&buf)

			handleBatchOperations(msg, encoder, tt.args)

			var response MCPMessage
			if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectedError {
				if response.Error == nil {
					t.Error("Expected error response but got success")
				}
				return
			}

			if response.Error != nil {
				t.Errorf("Unexpected error: %s", response.Error.Message)
				return
			}

			result := response.Result.(map[string]interface{})
			results := result["results"].([]interface{})

			if tt.expectedOps > 0 && len(results) != tt.expectedOps {
				t.Errorf("Expected %d results, got %d", tt.expectedOps, len(results))
			}
		})
	}
}

// TestSendError tests error response generation
func TestSendError(t *testing.T) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	sendError(encoder, 1, -32601, "test error", map[string]string{"detail": "test detail"})

	var response MCPMessage
	if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc version '2.0', got '%s'", response.JSONRPC)
	}

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if response.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", response.Error.Code)
	}

	if response.Error.Message != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", response.Error.Message)
	}

	if response.Error.Data == nil {
		t.Error("Expected error data")
	}
}

// TestMainLoop tests the main message processing loop
func TestMainLoop(t *testing.T) {
	// Create a temporary input with multiple messages
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}
{}
{"jsonrpc":"2.0","id":2,"method":"unknown","params":{}}
`

	scanner := newTestScanner(input)
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	// Process first message for initialize
	msg := &MCPMessage{}
	if scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), msg); err == nil {
			handleRequest(msg, encoder)
		}
	}

	// Process remaining messages
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

	// Verify responses were generated
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected output from message processing")
	}
}

// Helper function to create a test scanner
func newTestScanner(input string) *bufio.Scanner {
	return bufio.NewScanner(strings.NewReader(input))
}

// TestMCPMessageSerialization tests JSON serialization of MCP messages
func TestMCPMessageSerialization(t *testing.T) {
	msg := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal MCPMessage: %v", err)
	}

	var unmarshaled MCPMessage
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal MCPMessage: %v", err)
	}

	if unmarshaled.JSONRPC != msg.JSONRPC {
		t.Errorf("JSONRPC mismatch: expected %s, got %s", msg.JSONRPC, unmarshaled.JSONRPC)
	}

	// ID can be float64 when unmarshaled from JSON
	if fmt.Sprintf("%v", unmarshaled.ID) != fmt.Sprintf("%v", msg.ID) {
		t.Errorf("ID mismatch: expected %v, got %v", msg.ID, unmarshaled.ID)
	}

	if unmarshaled.Method != msg.Method {
		t.Errorf("Method mismatch: expected %s, got %s", msg.Method, unmarshaled.Method)
	}
}

// BenchmarkHandleRequest benchmarks the handleRequest function
func BenchmarkHandleRequest(b *testing.B) {
	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		handleRequest(msg, encoder)
	}
}

// TestConcurrentAccess tests concurrent message processing
func TestConcurrentAccess(t *testing.T) {
	const numGoroutines = 10
	const numMessages = 5

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numMessages; j++ {
				msg := &MCPMessage{
					JSONRPC: "2.0",
					ID:      id*numMessages + j,
					Method:  "tools/list",
				}

				var buf bytes.Buffer
				encoder := json.NewEncoder(&buf)

				handleRequest(msg, encoder)

				var response MCPMessage
				if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
					errChan <- err
					return
				}

				if response.Error != nil {
					errChan <- errors.New(response.Error.Message)
					return
				}
			}
			errChan <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent access error: %v", err)
		}
	}
}