package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestIntegrationCompleteFlow tests the complete MCP protocol flow
func TestIntegrationCompleteFlow(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Test 1: Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Verify initialize response
	if initResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", initResp.JSONRPC)
	}
	if initResp.ID != float64(1) {
		t.Errorf("Expected ID 1, got %v", initResp.ID)
	}
	if initResp.Result == nil {
		t.Error("Initialize response missing result")
	} else {
		result, ok := initResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Initialize result is not a map")
		} else {
			if protocolVersion, ok := result["protocolVersion"].(string); !ok || protocolVersion != "2024-11-05" {
				t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
			}
			if serverInfo, ok := result["serverInfo"].(map[string]interface{}); ok {
				if name, ok := serverInfo["name"].(string); !ok || name != "mcp-systeminfo" {
					t.Errorf("Expected server name 'mcp-systeminfo', got %v", serverInfo["name"])
				}
			} else {
				t.Error("Server info missing or invalid")
			}
		}
	}

	// Test 2: Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test 3: List tools
	toolsListReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	if err := encoder.Encode(toolsListReq); err != nil {
		t.Fatalf("Failed to send tools/list request: %v", err)
	}

	// Read tools list response
	var toolsListResp MCPMessage
	if err := decoder.Decode(&toolsListResp); err != nil {
		t.Fatalf("Failed to decode tools/list response: %v", err)
	}

	// Verify tools list response
	if toolsListResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolsListResp.JSONRPC)
	}
	if toolsListResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", toolsListResp.ID)
	}
	if toolsListResp.Result == nil {
		t.Error("Tools list response missing result")
	} else {
		result, ok := toolsListResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tools list result is not a map")
		} else {
			if tools, ok := result["tools"].([]interface{}); !ok {
				t.Error("Tools list missing tools array")
			} else if len(tools) == 0 {
				t.Error("No tools returned")
			} else {
				// Check apply_operations tool
				foundApplyOps := false
				for _, tool := range tools {
					if toolMap, ok := tool.(map[string]interface{}); ok {
						if name, ok := toolMap["name"].(string); ok && name == "apply_operations" {
							foundApplyOps = true
							if desc, ok := toolMap["description"].(string); !ok || desc == "" {
								t.Error("apply_operations tool missing description")
							}
							if schema, ok := toolMap["inputSchema"].(map[string]interface{}); !ok {
								t.Error("apply_operations tool missing input schema")
							}
							break
						}
					}
				}
				if !foundApplyOps {
					t.Error("apply_operations tool not found in tools list")
				}
			}
		}
	}

	// Test 4: Call apply_operations with a single operation
	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"type": "get_system_info",
					},
				},
			},
		},
	}

	if err := encoder.Encode(toolCallReq); err != nil {
		t.Fatalf("Failed to send tool call request: %v", err)
	}

	// Read tool call response
	var toolCallResp MCPMessage
	if err := decoder.Decode(&toolCallResp); err != nil {
		t.Fatalf("Failed to decode tool call response: %v", err)
	}

	// Verify tool call response
	if toolCallResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolCallResp.JSONRPC)
	}
	if toolCallResp.ID != float64(3) {
		t.Errorf("Expected ID 3, got %v", toolCallResp.ID)
	}
	if toolCallResp.Result == nil {
		t.Error("Tool call response missing result")
	} else {
		result, ok := toolCallResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tool call result is not a map")
		} else {
			if results, ok := result["results"].([]interface{}); !ok {
				t.Error("Tool call result missing results array")
			} else if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
			} else {
				// Check the first result
				if result0, ok := results[0].(map[string]interface{}); ok {
					if operation, ok := result0["operation"].(string); !ok || operation != "get_system_info" {
						t.Errorf("Expected operation 'get_system_info', got %v", result0["operation"])
					}
					if status, ok := result0["status"].(string); !ok || status != "Success" {
						t.Errorf("Expected status 'Success', got %v", result0["status"])
					}
					if _, ok := result0["result"]; !ok {
						t.Error("Result missing result data")
					}
				} else {
					t.Error("First result is not a map")
				}
			}
		}
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}

	// Verify audit log was created
	if _, err := os.Stat(auditFile); os.IsNotExist(err) {
		t.Error("Audit file was not created")
	}
}

// TestIntegrationBatchOperations tests batch operations with multiple system information requests
func TestIntegrationBatchOperations(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test batch operations with multiple operations
	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"type": "get_os_info",
					},
					{
						"type": "get_hardware_info",
					},
					{
						"type": "get_environment_info",
					},
					{
						"type": "get_shell_info",
					},
					{
						"type": "get_development_tools",
					},
				},
			},
		},
	}

	if err := encoder.Encode(toolCallReq); err != nil {
		t.Fatalf("Failed to send tool call request: %v", err)
	}

	// Read tool call response
	var toolCallResp MCPMessage
	if err := decoder.Decode(&toolCallResp); err != nil {
		t.Fatalf("Failed to decode tool call response: %v", err)
	}

	// Verify tool call response
	if toolCallResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolCallResp.JSONRPC)
	}
	if toolCallResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", toolCallResp.ID)
	}
	if toolCallResp.Result == nil {
		t.Error("Tool call response missing result")
	} else {
		result, ok := toolCallResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tool call result is not a map")
		} else {
			if results, ok := result["results"].([]interface{}); !ok {
				t.Error("Tool call result missing results array")
			} else if len(results) != 5 {
				t.Errorf("Expected 5 results, got %d", len(results))
			} else {
				// Check each result
				expectedOps := []string{"get_os_info", "get_hardware_info", "get_environment_info", "get_shell_info", "get_development_tools"}
				for i, expectedOp := range expectedOps {
					if i >= len(results) {
						t.Errorf("Missing result for operation %s", expectedOp)
						continue
					}
					if result, ok := results[i].(map[string]interface{}); ok {
						if operation, ok := result["operation"].(string); !ok || operation != expectedOp {
							t.Errorf("Expected operation '%s', got %v", expectedOp, result["operation"])
						}
						if status, ok := result["status"].(string); !ok || status != "Success" {
							t.Errorf("Expected status 'Success' for %s, got %v", expectedOp, result["status"])
						}
						if _, ok := result["result"]; !ok {
							t.Errorf("Result missing result data for %s", expectedOp)
						}
					} else {
						t.Errorf("Result %d is not a map", i)
					}
				}
			}
		}
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}

	// Verify audit log contains entries for all operations
	if data, err := os.ReadFile(auditFile); err == nil {
		lines := strings.Split(string(data), "\n")
		// Count non-empty lines
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		// Should have at least 5 operation entries + startup/shutdown
		if count < 7 {
			t.Errorf("Expected at least 7 audit log entries, got %d", count)
		}
	} else {
		t.Errorf("Failed to read audit file: %v", err)
	}
}

// TestIntegrationErrorHandling tests error handling across the complete protocol flow
func TestIntegrationErrorHandling(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test 1: Unknown method
	unknownMethodReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "unknown/method",
	}

	if err := encoder.Encode(unknownMethodReq); err != nil {
		t.Fatalf("Failed to send unknown method request: %v", err)
	}

	// Read error response
	var unknownMethodResp MCPMessage
	if err := decoder.Decode(&unknownMethodResp); err != nil {
		t.Fatalf("Failed to decode unknown method response: %v", err)
	}

	// Verify error response
	if unknownMethodResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", unknownMethodResp.JSONRPC)
	}
	if unknownMethodResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", unknownMethodResp.ID)
	}
	if unknownMethodResp.Error == nil {
		t.Error("Expected error response, got success")
	} else {
		if unknownMethodResp.Error.Code != -32601 {
			t.Errorf("Expected error code -32601, got %d", unknownMethodResp.Error.Code)
		}
		if !strings.Contains(unknownMethodResp.Error.Message, "Unknown method") {
			t.Errorf("Expected error message to contain 'Unknown method', got %s", unknownMethodResp.Error.Message)
		}
	}

	// Test 2: Unknown tool
	unknownToolReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "unknown_tool",
			"arguments": map[string]interface{}{
				"param": "value",
			},
		},
	}

	if err := encoder.Encode(unknownToolReq); err != nil {
		t.Fatalf("Failed to send unknown tool request: %v", err)
	}

	// Read error response
	var unknownToolResp MCPMessage
	if err := decoder.Decode(&unknownToolResp); err != nil {
		t.Fatalf("Failed to decode unknown tool response: %v", err)
	}

	// Verify error response
	if unknownToolResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", unknownToolResp.JSONRPC)
	}
	if unknownToolResp.ID != float64(3) {
		t.Errorf("Expected ID 3, got %v", unknownToolResp.ID)
	}
	if unknownToolResp.Error == nil {
		t.Error("Expected error response, got success")
	} else {
		if unknownToolResp.Error.Code != -32601 {
			t.Errorf("Expected error code -32601, got %d", unknownToolResp.Error.Code)
		}
		if !strings.Contains(unknownToolResp.Error.Message, "Unknown tool") {
			t.Errorf("Expected error message to contain 'Unknown tool', got %s", unknownToolResp.Error.Message)
		}
	}

	// Test 3: Invalid operation type in batch
	invalidOpReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"type": "invalid_operation_type",
					},
				},
			},
		},
	}

	if err := encoder.Encode(invalidOpReq); err != nil {
		t.Fatalf("Failed to send invalid operation request: %v", err)
	}

	// Read tool call response
	var invalidOpResp MCPMessage
	if err := decoder.Decode(&invalidOpResp); err != nil {
		t.Fatalf("Failed to decode invalid operation response: %v", err)
	}

	// Verify tool call response contains error in result
	if invalidOpResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", invalidOpResp.JSONRPC)
	}
	if invalidOpResp.ID != float64(4) {
		t.Errorf("Expected ID 4, got %v", invalidOpResp.ID)
	}
	if invalidOpResp.Result == nil {
		t.Error("Tool call response missing result")
	} else {
		result, ok := invalidOpResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tool call result is not a map")
		} else {
			if results, ok := result["results"].([]interface{}); !ok {
				t.Error("Tool call result missing results array")
			} else if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
			} else {
				// Check the first result
				if result0, ok := results[0].(map[string]interface{}); ok {
					if operation, ok := result0["operation"].(string); !ok || operation != "invalid_operation_type" {
						t.Errorf("Expected operation 'invalid_operation_type', got %v", result0["operation"])
					}
					if status, ok := result0["status"].(string); !ok || status != "Error" {
						t.Errorf("Expected status 'Error', got %v", result0["status"])
					}
					if message, ok := result0["message"].(string); !ok || !strings.Contains(message, "unknown operation type") {
						t.Errorf("Expected error message to contain 'unknown operation type', got %v", result0["message"])
					}
				} else {
					t.Error("First result is not a map")
				}
			}
		}
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}
}

// TestIntegrationConcurrentRequests tests concurrent request handling
func TestIntegrationConcurrentRequests(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Send multiple concurrent requests
	var wg sync.WaitGroup
	responses := make(chan MCPMessage, 10)
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create a separate encoder for this goroutine
			req := MCPMessage{
				JSONRPC: "2.0",
				ID:      id + 2,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "apply_operations",
					"arguments": map[string]interface{}{
						"operations": []map[string]interface{}{
							{
								"type": "get_os_info",
							},
						},
					},
				},
			}

			if err := encoder.Encode(req); err != nil {
				errors <- fmt.Errorf("failed to send request %d: %v", id, err)
				return
			}

			// Read response
			var resp MCPMessage
			if err := decoder.Decode(&resp); err != nil {
				errors <- fmt.Errorf("failed to decode response %d: %v", id, err)
				return
			}

			responses <- resp
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(responses)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Check responses
	responseCount := 0
	for resp := range responses {
		responseCount++
		if resp.JSONRPC != "2.0" {
			t.Errorf("Response %d: Expected JSON-RPC version 2.0, got %s", responseCount, resp.JSONRPC)
		}
		if resp.Result == nil {
			t.Errorf("Response %d: Missing result", responseCount)
		}
	}

	if responseCount != 5 {
		t.Errorf("Expected 5 responses, got %d", responseCount)
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}
}

// TestIntegrationAllOperationTypes tests all operation types
func TestIntegrationAllOperationTypes(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test all operation types
	operations := []map[string]interface{}{
		{"type": "get_system_info"},
		{"type": "get_os_info"},
		{"type": "get_hardware_info"},
		{"type": "get_environment_info"},
		{"type": "get_shell_info"},
		{"type": "get_development_tools"},
		{"type": "get_network_info"},
		{"type": "detect_repositories"},
		{"type": "check_command", "command": "go"},
		{"type": "get_recommendations"},
	}

	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": operations,
			},
		},
	}

	if err := encoder.Encode(toolCallReq); err != nil {
		t.Fatalf("Failed to send tool call request: %v", err)
	}

	// Read tool call response
	var toolCallResp MCPMessage
	if err := decoder.Decode(&toolCallResp); err != nil {
		t.Fatalf("Failed to decode tool call response: %v", err)
	}

	// Verify tool call response
	if toolCallResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolCallResp.JSONRPC)
	}
	if toolCallResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", toolCallResp.ID)
	}
	if toolCallResp.Result == nil {
		t.Error("Tool call response missing result")
	} else {
		result, ok := toolCallResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tool call result is not a map")
		} else {
			if results, ok := result["results"].([]interface{}); !ok {
				t.Error("Tool call result missing results array")
			} else if len(results) != len(operations) {
				t.Errorf("Expected %d results, got %d", len(operations), len(results))
			} else {
				// Check each result
				for i, op := range operations {
					if i >= len(results) {
						t.Errorf("Missing result for operation %v", op)
						continue
					}
					if result, ok := results[i].(map[string]interface{}); ok {
						expectedType := op["type"].(string)
						if operation, ok := result["operation"].(string); !ok || operation != expectedType {
							t.Errorf("Expected operation '%s', got %v", expectedType, result["operation"])
						}
						if status, ok := result["status"].(string); !ok || status != "Success" {
							t.Errorf("Expected status 'Success' for %s, got %v", expectedType, result["status"])
						}
						if _, ok := result["result"]; !ok {
							t.Errorf("Result missing result data for %s", expectedType)
						}
					} else {
						t.Errorf("Result %d is not a map", i)
					}
				}
			}
		}
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}
}

// TestIntegrationSecurityPolicyEnforcement tests security policy enforcement in the integration context
func TestIntegrationSecurityPolicyEnforcement(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test check_command with a potentially dangerous command
	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"type":    "check_command",
						"command": "rm -rf /",
					},
				},
			},
		},
	}

	if err := encoder.Encode(toolCallReq); err != nil {
		t.Fatalf("Failed to send tool call request: %v", err)
	}

	// Read tool call response
	var toolCallResp MCPMessage
	if err := decoder.Decode(&toolCallResp); err != nil {
		t.Fatalf("Failed to decode tool call response: %v", err)
	}

	// Verify tool call response
	if toolCallResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolCallResp.JSONRPC)
	}
	if toolCallResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", toolCallResp.ID)
	}
	if toolCallResp.Result == nil {
		t.Error("Tool call response missing result")
	} else {
		result, ok := toolCallResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tool call result is not a map")
		} else {
			if results, ok := result["results"].([]interface{}); !ok {
				t.Error("Tool call result missing results array")
			} else if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
			} else {
				// Check the first result
				if result0, ok := results[0].(map[string]interface{}); ok {
					if operation, ok := result0["operation"].(string); !ok || operation != "check_command" {
						t.Errorf("Expected operation 'check_command', got %v", result0["operation"])
					}
					if status, ok := result0["status"].(string); !ok || status != "Error" {
						t.Errorf("Expected status 'Error', got %v", result0["status"])
					}
					if message, ok := result0["message"].(string); !ok || !strings.Contains(message, "invalid command name format") {
						t.Errorf("Expected error message to contain 'invalid command name format', got %v", result0["message"])
					}
				} else {
					t.Error("First result is not a map")
				}
			}
		}
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}
}

// TestIntegrationAuditLogging tests audit logging during actual operations
func TestIntegrationAuditLogging(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Perform multiple operations to generate audit logs
	operations := []map[string]interface{}{
		{"type": "get_system_info"},
		{"type": "get_os_info"},
		{"type": "check_command", "command": "go"},
		{"type": "check_command", "command": "invalid-command-format!"},
	}

	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": operations,
			},
		},
	}

	if err := encoder.Encode(toolCallReq); err != nil {
		t.Fatalf("Failed to send tool call request: %v", err)
	}

	// Read tool call response
	var toolCallResp MCPMessage
	if err := decoder.Decode(&toolCallResp); err != nil {
		t.Fatalf("Failed to decode tool call response: %v", err)
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}

	// Verify audit log contains entries for all operations
	if data, err := os.ReadFile(auditFile); err == nil {
		lines := strings.Split(string(data), "\n")
		// Count non-empty lines
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		// Should have at least 4 operation entries + startup/shutdown
		if count < 6 {
			t.Errorf("Expected at least 6 audit log entries, got %d", count)
		}

		// Parse audit entries
		var entries []map[string]interface{}
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err == nil {
				entries = append(entries, entry)
			}
		}

		// Check for startup entry
		foundStartup := false
		for _, entry := range entries {
			if eventType, ok := entry["event_type"].(string); ok && eventType == "startup" {
				foundStartup = true
				break
			}
		}
		if !foundStartup {
			t.Error("Startup audit entry not found")
		}

		// Check for operation entries
		operationCount := 0
		for _, entry := range entries {
			if eventType, ok := entry["event_type"].(string); ok && eventType == "operation" {
				operationCount++
				if operation, ok := entry["operation"].(string); ok {
					if operation == "" {
						t.Error("Operation entry missing operation field")
					}
				} else {
					t.Error("Operation entry missing operation field")
				}
				if timestamp, ok := entry["timestamp"].(string); ok {
					if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
						t.Errorf("Invalid timestamp format: %v", timestamp)
					}
				} else {
					t.Error("Operation entry missing timestamp field")
				}
				if success, ok := entry["success"].(bool); ok {
					// Success field is present
				} else {
					t.Error("Operation entry missing success field")
				}
			}
		}
		if operationCount < 4 {
			t.Errorf("Expected at least 4 operation entries, got %d", operationCount)
		}

		// Check for shutdown entry
		foundShutdown := false
		for _, entry := range entries {
			if eventType, ok := entry["event_type"].(string); ok && eventType == "shutdown" {
				foundShutdown = true
				break
			}
		}
		if !foundShutdown {
			t.Error("Shutdown audit entry not found")
		}
	} else {
		t.Errorf("Failed to read audit file: %v", err)
	}
}

// TestIntegrationWithActualMain tests the actual main.go entry point with controlled input/output
func TestIntegrationWithActualMain(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create a test script that will run the main function
	testScript := filepath.Join(tempDir, "test_main.go")
	testCode := `package main

import (
	"os"
	"os/exec"
)

func main() {
	// Run the actual main function
	cmd := exec.Command("go", "run", "main.go", "mcp.go", "types.go", "systeminfo_operations.go", "security.go", "audit.go", "shell_info.go", "network_info.go", "repository_info.go", "devtools_info.go")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
`

	if err := os.WriteFile(testScript, []byte(testCode), 0644); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the test script in a goroutine
	cmd := exec.Command("go", "run", testScript)
	cmd.Dir = "cmd/mcp-systeminfo"
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test script: %v", err)
	}

	// Give the server time to start
	time.Sleep(500 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test tools list
	toolsListReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	if err := encoder.Encode(toolsListReq); err != nil {
		t.Fatalf("Failed to send tools/list request: %v", err)
	}

	// Read tools list response
	var toolsListResp MCPMessage
	if err := decoder.Decode(&toolsListResp); err != nil {
		t.Fatalf("Failed to decode tools/list response: %v", err)
	}

	// Verify tools list response
	if toolsListResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolsListResp.JSONRPC)
	}
	if toolsListResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", toolsListResp.ID)
	}
	if toolsListResp.Result == nil {
		t.Error("Tools list response missing result")
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		t.Errorf("Command error: %v", err)
	}

	// Verify audit log was created
	if _, err := os.Stat(auditFile); os.IsNotExist(err) {
		t.Error("Audit file was not created")
	}
}

// TestIntegrationJSONRPCSpecification tests that all responses conform to the MCP specification
func TestIntegrationJSONRPCSpecification(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Test various request/response pairs
	testCases := []struct {
		name     string
		request  MCPMessage
		validate func(MCPMessage) error
	}{
		{
			name: "initialize",
			request: MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"clientInfo": map[string]interface{}{
						"name":    "test-client",
						"version": "1.0.0",
					},
				},
			},
			validate: func(resp MCPMessage) error {
				if resp.JSONRPC != "2.0" {
					return fmt.Errorf("Expected JSON-RPC version 2.0, got %s", resp.JSONRPC)
				}
				if resp.ID != float64(1) {
					return fmt.Errorf("Expected ID 1, got %v", resp.ID)
				}
				if resp.Result == nil {
					return fmt.Errorf("Initialize response missing result")
				}
				return nil
			},
		},
		{
			name: "tools/list",
			request: MCPMessage{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/list",
			},
			validate: func(resp MCPMessage) error {
				if resp.JSONRPC != "2.0" {
					return fmt.Errorf("Expected JSON-RPC version 2.0, got %s", resp.JSONRPC)
				}
				if resp.ID != float64(2) {
					return fmt.Errorf("Expected ID 2, got %v", resp.ID)
				}
				if resp.Result == nil {
					return fmt.Errorf("Tools list response missing result")
				}
				return nil
			},
		},
		{
			name: "tools/call",
			request: MCPMessage{
				JSONRPC: "2.0",
				ID:      3,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "apply_operations",
					"arguments": map[string]interface{}{
						"operations": []map[string]interface{}{
							{
								"type": "get_system_info",
							},
						},
					},
				},
			},
			validate: func(resp MCPMessage) error {
				if resp.JSONRPC != "2.0" {
					return fmt.Errorf("Expected JSON-RPC version 2.0, got %s", resp.JSONRPC)
				}
				if resp.ID != float64(3) {
					return fmt.Errorf("Expected ID 3, got %v", resp.ID)
				}
				if resp.Result == nil {
					return fmt.Errorf("Tool call response missing result")
				}
				return nil
			},
		},
		{
			name: "unknown_method",
			request: MCPMessage{
				JSONRPC: "2.0",
				ID:      4,
				Method:  "unknown/method",
			},
			validate: func(resp MCPMessage) error {
				if resp.JSONRPC != "2.0" {
					return fmt.Errorf("Expected JSON-RPC version 2.0, got %s", resp.JSONRPC)
				}
				if resp.ID != float64(4) {
					return fmt.Errorf("Expected ID 4, got %v", resp.ID)
				}
				if resp.Error == nil {
					return fmt.Errorf("Expected error response, got success")
				}
				if resp.Error.Code != -32601 {
					return fmt.Errorf("Expected error code -32601, got %d", resp.Error.Code)
				}
				return nil
			},
		},
	}

	// Initialize first
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Send request
			if err := encoder.Encode(tc.request); err != nil {
				t.Fatalf("Failed to send %s request: %v", tc.name, err)
			}

			// Read response
			var resp MCPMessage
			if err := decoder.Decode(&resp); err != nil {
				t.Fatalf("Failed to decode %s response: %v", tc.name, err)
			}

			// Validate response
			if err := tc.validate(resp); err != nil {
				t.Errorf("%s validation failed: %v", tc.name, err)
			}

			// Additional JSON-RPC specification validation
			// All responses must have jsonrpc field
			if resp.JSONRPC != "2.0" {
				t.Errorf("%s: Expected JSON-RPC version 2.0, got %s", tc.name, resp.JSONRPC)
			}

			// All responses must have either result or error, but not both
			if resp.Result != nil && resp.Error != nil {
				t.Errorf("%s: Response has both result and error", tc.name)
			}
			if resp.Result == nil && resp.Error == nil {
				t.Errorf("%s: Response has neither result nor error", tc.name)
			}

			// Error responses must have code and message
			if resp.Error != nil {
				if resp.Error.Code == 0 {
					t.Errorf("%s: Error response missing code", tc.name)
				}
				if resp.Error.Message == "" {
					t.Errorf("%s: Error response missing message", tc.name)
				}
			}
		})
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}
}

// TestIntegrationRealCommandExecution tests real command execution in a controlled environment
func TestIntegrationRealCommandExecution(t *testing.T) {
	// Skip on Windows as some commands might not be available
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows")
	}

	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Create pipes for stdin/stdout
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	// Start the main function in a goroutine
	done := make(chan error, 1)
	go func() {
		// Save original stdin/stdout
		origStdin := os.Stdin
		origStdout := os.Stdout
		defer func() {
			os.Stdin = origStdin
			os.Stdout = origStdout
		}()

		// Replace stdin/stdout with our pipes
		os.Stdin = stdinR
		os.Stdout = stdoutW

		// Call main function
		done <- nil
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Create encoder/decoder for communication
	encoder := json.NewEncoder(stdinW)
	decoder := json.NewDecoder(stdoutR)

	// Initialize
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to decode initialize response: %v", err)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test check_command with a real command
	toolCallReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "apply_operations",
			"arguments": map[string]interface{}{
				"operations": []map[string]interface{}{
					{
						"type":    "check_command",
						"command": "uname",
					},
				},
			},
		},
	}

	if err := encoder.Encode(toolCallReq); err != nil {
		t.Fatalf("Failed to send tool call request: %v", err)
	}

	// Read tool call response
	var toolCallResp MCPMessage
	if err := decoder.Decode(&toolCallResp); err != nil {
		t.Fatalf("Failed to decode tool call response: %v", err)
	}

	// Verify tool call response
	if toolCallResp.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", toolCallResp.JSONRPC)
	}
	if toolCallResp.ID != float64(2) {
		t.Errorf("Expected ID 2, got %v", toolCallResp.ID)
	}
	if toolCallResp.Result == nil {
		t.Error("Tool call response missing result")
	} else {
		result, ok := toolCallResp.Result.(map[string]interface{})
		if !ok {
			t.Error("Tool call result is not a map")
		} else {
			if results, ok := result["results"].([]interface{}); !ok {
				t.Error("Tool call result missing results array")
			} else if len(results) != 1 {
				t.Errorf("Expected 1 result, got %d", len(results))
			} else {
				// Check the first result
				if result0, ok := results[0].(map[string]interface{}); ok {
					if operation, ok := result0["operation"].(string); !ok || operation != "check_command" {
						t.Errorf("Expected operation 'check_command', got %v", result0["operation"])
					}
					if status, ok := result0["status"].(string); !ok || status != "Success" {
						t.Errorf("Expected status 'Success', got %v", result0["status"])
					}
					if resultData, ok := result0["result"].(map[string]interface{}); ok {
						if exists, ok := resultData["exists"].(bool); !ok || !exists {
							t.Errorf("Expected command to exist, got %v", resultData["exists"])
						}
						if command, ok := resultData["command"].(string); !ok || command != "uname" {
							t.Errorf("Expected command 'uname', got %v", resultData["command"])
						}
					} else {
						t.Error("Result data is not a map")
					}
				} else {
					t.Error("First result is not a map")
				}
			}
		}
	}

	// Close pipes
	stdinW.Close()
	stdoutW.Close()

	// Wait for goroutine to finish
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server did not shut down properly")
	}
}

// TestIntegrationResourceLeaks tests for resource leaks
func TestIntegrationResourceLeaks(t *testing.T) {
	// Create a temporary audit file for testing
	tempDir := t.TempDir()
	auditFile := filepath.Join(tempDir, "test-audit.log")

	// Set environment variables for testing
	t.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", auditFile)
	t.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "false")

	// Run multiple iterations to check for resource leaks
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			// Create pipes for stdin/stdout
			stdinR, stdinW := io.Pipe()
			stdoutR, stdoutW := io.Pipe()

			// Start the main function in a goroutine
			done := make(chan error, 1)
			go func() {
				// Save original stdin/stdout
				origStdin := os.Stdin
				origStdout := os.Stdout
				defer func() {
					os.Stdin = origStdin
					os.Stdout = origStdout
				}()

				// Replace stdin/stdout with our pipes
				os.Stdin = stdinR
				os.Stdout = stdoutW

				// Call main function
				done <- nil
			}()

			// Give the server time to start
			time.Sleep(50 * time.Millisecond)

			// Create encoder/decoder for communication
			encoder := json.NewEncoder(stdinW)
			decoder := json.NewDecoder(stdoutR)

			// Initialize
			initReq := MCPMessage{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"clientInfo": map[string]interface{}{
						"name":    "test-client",
						"version": "1.0.0",
					},
				},
			}

			if err := encoder.Encode(initReq); err != nil {
				t.Fatalf("Failed to send initialize request: %v", err)
			}

			// Read initialize response
			var initResp MCPMessage
			if err := decoder.Decode(&initResp); err != nil {
				t.Fatalf("Failed to decode initialize response: %v", err)
			}

			// Send initialized notification
			initNotif := MCPMessage{
				JSONRPC: "2.0",
				Method:  "initialized",
			}

			if err := encoder.Encode(initNotif); err != nil {
				t.Fatalf("Failed to send initialized notification: %v", err)
			}

			// Perform a simple operation
			toolCallReq := MCPMessage{
				JSONRPC: "2.0",
				ID:      2,
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "apply_operations",
					"arguments": map[string]interface{}{
						"operations": []map[string]interface{}{
							{
								"type": "get_os_info",
							},
						},
					},
				},
			}

			if err := encoder.Encode(toolCallReq); err != nil {
				t.Fatalf("Failed to send tool call request: %v", err)
			}

			// Read tool call response
			var toolCallResp MCPMessage
			if err := decoder.Decode(&toolCallResp); err != nil {
				t.Fatalf("Failed to decode tool call response: %v", err)
			}

			// Close pipes
			stdinW.Close()
			stdoutW.Close()

			// Wait for goroutine to finish
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("Server error: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Error("Server did not shut down properly")
			}
		})
	}

	// Verify audit log was created and has entries
	if data, err := os.ReadFile(auditFile); err == nil {
		lines := strings.Split(string(data), "\n")
		// Count non-empty lines
		count := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		// Should have at least 10 operation entries + startup/shutdown for each iteration
		if count < 20 {
			t.Errorf("Expected at least 20 audit log entries, got %d", count)
		}
	} else {
		t.Errorf("Failed to read audit file: %v", err)
	}
}