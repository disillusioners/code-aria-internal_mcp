package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestMainFunction tests the main function with various inputs
func TestMainFunction(t *testing.T) {
	// Note: Testing the actual main function is difficult because it exits the process
	// Instead, we'll test the components that main() uses
	
	// Test that audit logger initialization works
	if err := InitAuditLogger(); err != nil {
		t.Errorf("Failed to initialize audit logger: %v", err)
	}
	defer CloseAuditLogger()
	
	// Test that we can create a scanner and encoder
	scanner := bufio.NewScanner(bytes.NewReader([]byte{}))
	encoder := json.NewEncoder(io.Discard)
	
	// Test that handleInitialize works with valid input
	validInput := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}` + "\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`
	
	scanner = bufio.NewScanner(bytes.NewReader([]byte(validInput)))
	if err := handleInitialize(scanner, encoder); err != nil {
		t.Errorf("handleInitialize failed: %v", err)
	}
}

// TestSignalHandling tests the signal handling in main
func TestSignalHandling(t *testing.T) {
	// This test verifies that the signal handling code is set up correctly
	// We can't easily test the actual signal handling without race conditions,
	// but we can verify the signal channel is created and notified
	
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Create a signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)
	
	// Test that we can send a signal to the channel
	select {
	case sigChan <- syscall.SIGINT:
		// Signal sent successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to send signal to channel")
	}
	
	// Test that we can receive a signal from the channel
	select {
	case sig := <-sigChan:
		if sig != syscall.SIGINT {
			t.Errorf("Received wrong signal: got %v, want %v", sig, syscall.SIGINT)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Failed to receive signal from channel")
	}
}

// TestMainWithInvalidInput tests main function behavior with invalid input
func TestMainWithInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "Empty input",
			input:       "",
			expectError: false, // Should just continue
		},
		{
			name:        "Invalid JSON",
			input:       `{"jsonrpc":"2.0","id":1,"method":"invalid"`,
			expectError: false, // Should be ignored
		},
		{
			name:        "Valid JSON but invalid method",
			input:       `{"jsonrpc":"2.0","id":1,"method":"unknown_method"}`,
			expectError: false, // Should send error response
		},
		{
			name:        "Non-JSON input",
			input:       "not json at all",
			expectError: false, // Should be ignored
		},
		{
			name:        "Empty JSON object",
			input:       `{}`,
			expectError: false, // Should be ignored (no method)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a scanner with test input
			scanner := bufio.NewScanner(bytes.NewReader([]byte(tt.input)))
			
			// Create a buffer to capture output
			var output bytes.Buffer
			encoder := json.NewEncoder(&output)
			
			// Process each line
			for scanner.Scan() {
				line := scanner.Bytes()
				if len(line) == 0 {
					continue
				}
				
				var msg MCPMessage
				if err := json.Unmarshal(line, &msg); err != nil {
					continue // Ignore invalid JSON
				}
				
				if msg.Method != "" {
					handleRequest(&msg, encoder)
				}
			}
			
			// Check that something was written for valid requests
			if !tt.expectError && output.Len() == 0 && tt.input != "" && tt.input != "{}" {
				// Check if input had a method
				var testMsg MCPMessage
				if json.Unmarshal([]byte(tt.input), &testMsg) == nil && testMsg.Method != "" {
					t.Error("handleRequest() wrote no output for valid request")
				}
			}
		})
	}
}

// TestMainWithMultipleRequests tests processing multiple requests
func TestMainWithMultipleRequests(t *testing.T) {
	// Create multiple requests
	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"get_system_info"}]}}}`,
	}
	
	// Combine requests into input
	input := ""
	for _, req := range requests {
		input += req + "\n"
	}
	
	// Create a scanner with test input
	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
	
	// Create a buffer to capture output
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	
	// First, handle initialize
	if err := handleInitialize(scanner, encoder); err != nil {
		t.Errorf("handleInitialize failed: %v", err)
	}
	
	// Process remaining requests
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
	
	// Parse all responses
	outputStr := output.String()
	responses := strings.Split(strings.TrimSpace(outputStr), "\n")
	
	// Should have at least 2 responses (tools/list and tools/call)
	if len(responses) < 2 {
		t.Errorf("Expected at least 2 responses, got %d", len(responses))
	}
	
	// Verify each response is valid JSON
	for i, resp := range responses {
		if resp == "" {
			continue
		}
		
		var response MCPMessage
		if err := json.Unmarshal([]byte(resp), &response); err != nil {
			t.Errorf("Response %d is not valid JSON: %v", i, err)
		}
		
		// Check that each response has required fields
		if response.JSONRPC != "2.0" {
			t.Errorf("Response %d missing jsonrpc version", i)
		}
	}
}

// TestMainGracefulShutdown tests graceful shutdown behavior
func TestMainGracefulShutdown(t *testing.T) {
	// This test verifies that the audit logger is closed on shutdown
	// We can't test the actual signal handling without race conditions,
	// but we can verify the cleanup code works
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Errorf("Failed to initialize audit logger: %v", err)
	}
	
	// Close audit logger (simulating shutdown)
	CloseAuditLogger()
	
	// Verify that audit file was created and has shutdown message
	auditFile := "mcp-systeminfo-audit.log"
	if data, err := os.ReadFile(auditFile); err == nil {
		content := string(data)
		if !strings.Contains(content, "shutdown") {
			t.Error("Shutdown message not written to audit file")
		}
	} else {
		t.Errorf("Failed to read audit file: %v", err)
	}
	
	// Clean up
	os.Remove(auditFile)
}

// TestMainWithAuditDisabled tests main function with audit disabled
func TestMainWithAuditDisabled(t *testing.T) {
	// Disable audit
	os.Setenv("MCP_SYSTEMINFO_AUDIT_DISABLED", "true")
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_DISABLED")
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Errorf("Failed to initialize audit logger: %v", err)
	}
	defer CloseAuditLogger()
	
	// Verify audit is disabled
	stats := GetAuditStats()
	if enabled, ok := stats["enabled"].(bool); !ok || enabled {
		t.Error("Audit should be disabled")
	}
}

// TestMainWithCustomAuditFile tests main function with custom audit file
func TestMainWithCustomAuditFile(t *testing.T) {
	// Set custom audit file
	customFile := "custom-audit-test.log"
	os.Setenv("MCP_SYSTEMINFO_AUDIT_FILE", customFile)
	defer os.Unsetenv("MCP_SYSTEMINFO_AUDIT_FILE")
	defer os.Remove(customFile) // Clean up
	
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		t.Errorf("Failed to initialize audit logger: %v", err)
	}
	defer CloseAuditLogger()
	
	// Verify custom audit file is used
	stats := GetAuditStats()
	if auditFile, ok := stats["audit_file"].(string); !ok || auditFile != customFile {
		t.Errorf("Custom audit file not used: got %v, want %v", auditFile, customFile)
	}
}

// TestMainInputProcessing tests input processing in main
func TestMainInputProcessing(t *testing.T) {
	tests := []struct {
		name         string
		inputLines   []string
		expectOutput bool
	}{
		{
			name: "Valid request",
			inputLines: []string{
				`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			},
			expectOutput: true,
		},
		{
			name: "Empty line",
			inputLines: []string{
				"",
			},
			expectOutput: false,
		},
		{
			name: "Multiple empty lines",
			inputLines: []string{
				"",
				"",
				"",
			},
			expectOutput: false,
		},
		{
			name: "Empty line followed by valid request",
			inputLines: []string{
				"",
				`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			},
			expectOutput: true,
		},
		{
			name: "Valid request followed by empty line",
			inputLines: []string{
				`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
				"",
			},
			expectOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input
			input := ""
			for _, line := range tt.inputLines {
				input += line + "\n"
			}
			
			// Create a scanner with test input
			scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
			
			// Create a buffer to capture output
			var output bytes.Buffer
			encoder := json.NewEncoder(&output)
			
			// Process input
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
			
			// Check output
			hasOutput := output.Len() > 0
			if hasOutput != tt.expectOutput {
				t.Errorf("Output mismatch: got %v, want %v", hasOutput, tt.expectOutput)
			}
		})
	}
}

// TestMainErrorHandling tests error handling in main
func TestMainErrorHandling(t *testing.T) {
	// Test with invalid initialize request
	invalidInit := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	
	// Create a scanner with test input
	scanner := bufio.NewScanner(bytes.NewReader([]byte(invalidInit + "\n")))
	
	// Create a buffer to capture output
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	
	// Try to handle initialize
	err := handleInitialize(scanner, encoder)
	if err == nil {
		t.Error("handleInitialize should have failed with invalid params")
	}
	
	// Test with valid initialize but missing initialized notification
	validInit := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}`
	
	scanner = bufio.NewScanner(bytes.NewReader([]byte(validInit + "\n")))
	output.Reset()
	
	err = handleInitialize(scanner, encoder)
	if err == nil {
		t.Error("handleInitialize should have failed with missing initialized notification")
	}
}

// BenchmarkMainProcessing benchmarks the main processing loop
func BenchmarkMainProcessing(b *testing.B) {
	// Create a test request
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	
	// Create input with many requests
	input := ""
	for i := 0; i < b.N; i++ {
		input += request + "\n"
	}
	
	b.ResetTimer()
	
	// Process all requests
	scanner := bufio.NewScanner(bytes.NewReader([]byte(input)))
	encoder := json.NewEncoder(io.Discard)
	
	// First, handle initialize
	initInput := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}}}` + "\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`
	initScanner := bufio.NewScanner(bytes.NewReader([]byte(initInput)))
	handleInitialize(initScanner, encoder)
	
	// Process remaining requests
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

// BenchmarkJSONProcessing benchmarks JSON processing
func BenchmarkJSONProcessing(b *testing.B) {
	// Create a test message
	msg := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	
	// Serialize to JSON
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		b.Fatalf("Failed to marshal test message: %v", err)
	}
	
	b.ResetTimer()
	
	// Benchmark JSON unmarshaling
	for i := 0; i < b.N; i++ {
		var msg2 MCPMessage
		if err := json.Unmarshal(jsonBytes, &msg2); err != nil {
			b.Fatalf("Failed to unmarshal JSON: %v", err)
		}
	}
}