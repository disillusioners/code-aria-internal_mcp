package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// MockCommand implements a mock for exec.Command
type MockCommand struct {
	Cmd    string
	Args   []string
	Dir    string
	Output string
	Err    error
}

// TestTransport provides a transport layer for testing MCP communication
type TestTransport struct {
	Input  io.Reader
	Output io.Writer
}

// NewTestTransport creates a new test transport
func NewTestTransport(input string) *TestTransport {
	return &TestTransport{
		Input:  strings.NewReader(input),
		Output: &strings.Builder{},
	}
}

// SendMessage sends a message through the transport
func (t *TestTransport) SendMessage(msg *MCPMessage) error {
	encoder := json.NewEncoder(t.Output)
	return encoder.Encode(msg)
}

// ReceiveMessage receives a message from the transport
func (t *TestTransport) ReceiveMessage() (*MCPMessage, error) {
	scanner := bufio.NewScanner(t.Input)
	if !scanner.Scan() {
		return nil, io.EOF
	}

	var msg MCPMessage
	err := json.Unmarshal(scanner.Bytes(), &msg)
	return &msg, err
}

// MockEnvironment provides a way to mock environment variables for testing
type MockEnvironment struct {
	values map[string]string
}

// NewMockEnvironment creates a new mock environment
func NewMockEnvironment() *MockEnvironment {
	return &MockEnvironment{
		values: make(map[string]string),
	}
}

// Set sets an environment variable
func (m *MockEnvironment) Set(key, value string) {
	m.values[key] = value
}

// Get gets an environment variable
func (m *MockEnvironment) Get(key string) string {
	return m.values[key]
}

// Apply applies the mock environment
func (m *MockEnvironment) Apply() {
	for key, value := range m.values {
		os.Setenv(key, value)
	}
}

// Restore restores the original environment
func (m *MockEnvironment) Restore(original map[string]string) {
	// Clear current values
	for key := range m.values {
		os.Unsetenv(key)
	}
	// Restore original values
	for key, value := range original {
		os.Setenv(key, value)
	}
}

// CaptureEnvironment captures the current environment state
func CaptureEnvironment() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

// TestMCPClient simulates an MCP client for testing
type TestMCPClient struct {
	transport *TestTransport
	id        int
}

// NewTestMCPClient creates a new test MCP client
func NewTestMCPClient(input string) *TestMCPClient {
	return &TestMCPClient{
		transport: NewTestTransport(input),
		id:        1,
	}
}

// Initialize performs the MCP initialization handshake
func (c *TestMCPClient) Initialize(t *testing.T) error {
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test-client", "version": "1.0.0"}
		}`),
	}

	if err := c.transport.SendMessage(&initReq); err != nil {
		return err
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	return c.transport.SendMessage(&initNotif)
}

// ListTools requests the list of available tools
func (c *TestMCPClient) ListTools(t *testing.T) (*MCPMessage, error) {
	req := MCPMessage{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
	}

	if err := c.transport.SendMessage(&req); err != nil {
		return nil, err
	}

	return c.receiveMessage()
}

// CallTool calls a specific tool
func (c *TestMCPClient) CallTool(t *testing.T, name string, args map[string]interface{}) (*MCPMessage, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := MCPMessage{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
		Params:  json.RawMessage(paramsBytes),
	}

	if err := c.transport.SendMessage(&req); err != nil {
		return nil, err
	}

	return c.receiveMessage()
}

// receiveMessage receives a message from the transport
func (c *TestMCPClient) receiveMessage() (*MCPMessage, error) {
	scanner := bufio.NewScanner(c.transport.Input.(*strings.Reader))
	if !scanner.Scan() {
		return nil, io.EOF
	}

	var msg MCPMessage
	err := json.Unmarshal(scanner.Bytes(), &msg)
	return &msg, err
}

// nextID returns the next message ID
func (c *TestMCPClient) nextID() int {
	id := c.id
	c.id++
	return id
}

// GetOutput returns the output from the transport
func (c *TestMCPClient) GetOutput() string {
	if builder, ok := c.transport.Output.(*strings.Builder); ok {
		return builder.String()
	}
	return ""
}

// TestTimeout provides timeout functionality for tests
type TestTimeout struct {
	duration time.Duration
}

// NewTestTimeout creates a new test timeout
func NewTestTimeout(d time.Duration) *TestTimeout {
	return &TestTimeout{duration: d}
}

// Run runs a function with a timeout
func (tt *TestTimeout) Run(t *testing.T, testFunc func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		testFunc()
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(tt.duration):
		t.Fatal("Test timed out")
	}
}

// AssertNoError asserts that an error is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError asserts that an error is not nil
func AssertError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
	if expectedMsg != "" && !strings.Contains(err.Error(), expectedMsg) {
		t.Fatalf("Expected error containing '%s', got '%s'", expectedMsg, err.Error())
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertContains asserts that a string contains a substring
func AssertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("Expected '%s' to contain '%s'", s, substr)
	}
}

// CreateTestFile creates a test file with the given content
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := dir + "/" + filename
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	return path
}

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "mcp-lang-go-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// WaitForCondition waits for a condition to be true or times out
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	start := time.Now()
	for time.Since(start) < timeout {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Condition not met within timeout: %s", message)
}