//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIntegrationFullWorkflow tests a complete MCP workflow
func TestIntegrationFullWorkflow(t *testing.T) {
	// Skip if golangci-lint is not available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not available for integration tests")
	}

	// Create a temporary Go project
	tmpDir := CreateTempDir(t)

	// Initialize a Go module
	cmd := exec.Command("go", "mod", "init", "test.example.com/mcp-test")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize go module: %v", err)
	}

	// Create test files with various issues
	goFiles := map[string]string{
		"main.go": `
package main

import "fmt"
import "os" // This import is unused

func main() {
	fmt.Println("Hello")
	var x int
	x = 5
	_ = x // Variable declared but not used
}

func unusedFunction() {
	fmt.Println("This function is never called")
}
`,
		"utils.go": `
package main

import (
	"fmt"
	"strings"
)

func FunctionWithIssues() {
	var err error
	// Error not checked
	fmt.Sprintln("test")

	// Inefficient string concatenation
	var result string
	for i := 0; i < 10; i++ {
		result += string(rune(i))
	}

	_ = err
}

func ComplexFunction(param1, param2, param3, param4, param5 string,
	param6, param7, param8, param9, param10 string) string {
	// Too many parameters
	return param1 + param2 + param3 + param4 + param5 + param6 + param7 + param8 + param9 + param10
}
`,
	}

	for filename, content := range goFiles {
		CreateTestFile(t, tmpDir, filename, content)
	}

	// Create a golangci-lint config
	configContent := `
run:
  timeout: 5m

linters:
  disable-all: true
  enable:
    - deadcode
    - unused
    - goimports
    - govet
    - errcheck
    - gocyclo
    - funlen
    - gofmt
    - ineffassign
    - misspell
    - unconvert
    - unparam
    - nakedret
    - prealloc
    - gocritic

linters-settings:
  funlen:
    lines: 50
  gocyclo:
    min-complexity: 10
  goimports:
    local-prefixes: test.example.com
`
	CreateTestFile(t, tmpDir, ".golangci.yml", configContent)

	// Set up environment
	originalEnv := CaptureEnvironment()
	defer func() {
		NewMockEnvironment().Restore(originalEnv)
	}()

	mockEnv := NewMockEnvironment()
	mockEnv.Set("REPO_PATH", tmpDir)
	mockEnv.Apply()

	// Start the MCP server process
	serverCmd := exec.Command("go", "run", ".", "mcp-lang-go")
	serverCmd.Dir = "/home/nea/code/code-aria/code-aria-internal_mcp/cmd/mcp-lang-go"

	// Create pipes for communication
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	defer stdin.Close()

	stdout, err := serverCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}
	defer stdout.Close()

	serverCmd.Stderr = os.Stderr // Keep stderr visible for debugging

	// Start the server
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer serverCmd.Wait()
	defer serverCmd.Process.Kill()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create JSON encoder/decoder
	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(stdout)

	// Perform initialization handshake
	initReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: json.RawMessage(`{
			"protocolVersion": "2024-11-05",
			"capabilities": {},
			"clientInfo": {"name": "test-client", "version": "1.0.0"}
		}`),
	}

	if err := encoder.Encode(initReq); err != nil {
		t.Fatalf("Failed to send initialize request: %v", err)
	}

	// Read initialize response
	var initResp MCPMessage
	if err := decoder.Decode(&initResp); err != nil {
		t.Fatalf("Failed to read initialize response: %v", err)
	}

	if initResp.Error != nil {
		t.Fatalf("Initialize error: %s", initResp.Error.Message)
	}

	// Send initialized notification
	initNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialized",
	}

	if err := encoder.Encode(initNotif); err != nil {
		t.Fatalf("Failed to send initialized notification: %v", err)
	}

	// Test tools/list
	toolsReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	if err := encoder.Encode(toolsReq); err != nil {
		t.Fatalf("Failed to send tools/list request: %v", err)
	}

	var toolsResp MCPMessage
	if err := decoder.Decode(&toolsResp); err != nil {
		t.Fatalf("Failed to read tools/list response: %v", err)
	}

	if toolsResp.Error != nil {
		t.Fatalf("Tools list error: %s", toolsResp.Error.Message)
	}

	// Verify tool exists
	result := toolsResp.Result.(map[string]interface{})
	tools := result["tools"].([]interface{})
	if len(tools) == 0 {
		t.Fatal("No tools returned")
	}

	// Test apply_operations with lint
	lintArgs := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"type":   "lint",
				"target": ".",
				"format": "json",
			},
		},
	}

	lintReq := MCPMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: json.RawMessage(MustMarshal(map[string]interface{}{
			"name":      "apply_operations",
			"arguments": lintArgs,
		})),
	}

	if err := encoder.Encode(lintReq); err != nil {
		t.Fatalf("Failed to send lint request: %v", err)
	}

	var lintResp MCPMessage
	if err := decoder.Decode(&lintResp); err != nil {
		t.Fatalf("Failed to read lint response: %v", err)
	}

	if lintResp.Error != nil {
		t.Fatalf("Lint error: %s", lintResp.Error.Message)
	}

	// Verify lint results
	lintResult := lintResp.Result.(map[string]interface{})
	results := lintResult["results"].([]interface{})

	if len(results) == 0 {
		t.Fatal("No lint results returned")
	}

	firstResult := results[0].(map[string]interface{})
	AssertEqual(t, firstResult["status"], "Success")

	// Check that issues were found
	lintData := firstResult["result"].(map[string]interface{})
	AssertEqual(t, lintData["target"], ".")

	if lintData["total_issues"].(float64) == 0 {
		t.Error("Expected lint issues but got none")
	}
}

// TestIntegrationWithConfigFile tests using a custom config file
func TestIntegrationWithConfigFile(t *testing.T) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not available for integration tests")
	}

	tmpDir := CreateTempDir(t)

	// Initialize Go module
	cmd := exec.Command("go", "mod", "init", "test.example.com/config-test")
	cmd.Dir = tmpDir
	AssertNoError(t, cmd.Run())

	// Create a Go file with issues
	CreateTestFile(t, tmpDir, "test.go", `
package main

import "fmt"
import "os" // Unused import

func main() {
	var x int
	fmt.Println(x) // Variable declared but not used
}
`)

	// Create custom config
	configContent := `
linters:
  disable-all: true
  enable:
    - unused
    - goimports

linters-settings:
  unused:
    check-exported: false
`
	CreateTestFile(t, tmpDir, "custom.yml", configContent)

	// Set up environment
	originalEnv := CaptureEnvironment()
	defer func() {
		NewMockEnvironment().Restore(originalEnv)
	}()

	mockEnv := NewMockEnvironment()
	mockEnv.Set("REPO_PATH", tmpDir)
	mockEnv.Apply()

	// Test the tool directly
	args := map[string]interface{}{
		"target": ".",
		"config": "custom.yml",
		"format": "json",
	}

	result, err := toolLint(args)
	AssertNoError(t, err)

	lintResult, ok := result.(LintResult)
	AssertEqual(t, ok, true)
	AssertEqual(t, lintResult.Target, ".")
}

// TestIntegrationTextFormat tests the text output format
func TestIntegrationTextFormat(t *testing.T) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not available for integration tests")
	}

	tmpDir := CreateTempDir(t)

	// Initialize Go module
	cmd := exec.Command("go", "mod", "init", "test.example.com/text-test")
	cmd.Dir = tmpDir
	AssertNoError(t, cmd.Run())

	// Create a Go file
	CreateTestFile(t, tmpDir, "test.go", `
package main

func main() {
	var x int
	_ = x
}
`)

	// Set up environment
	originalEnv := CaptureEnvironment()
	defer func() {
		NewMockEnvironment().Restore(originalEnv)
	}()

	mockEnv := NewMockEnvironment()
	mockEnv.Set("REPO_PATH", tmpDir)
	mockEnv.Apply()

	// Test text format
	args := map[string]interface{}{
		"target": "test.go",
		"format": "text",
	}

	result, err := toolLint(args)
	AssertNoError(t, err)

	// Text format should return a string
	_, ok := result.(string)
	AssertEqual(t, ok, true)
}

// TestIntegrationErrorHandling tests error handling scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	tmpDir := CreateTempDir(t)

	// Don't set REPO_PATH to test error handling
	originalEnv := CaptureEnvironment()
	defer func() {
		NewMockEnvironment().Restore(originalEnv)
	}()

	os.Unsetenv("REPO_PATH")

	args := map[string]interface{}{
		"target": ".",
	}

	_, err := toolLint(args)
	AssertError(t, err, "REPO_PATH environment variable not set")
}

// TestIntegrationMultipleOperations tests multiple operations in one call
func TestIntegrationMultipleOperations(t *testing.T) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		t.Skip("golangci-lint not available for integration tests")
	}

	tmpDir := CreateTempDir(t)

	// Initialize Go module
	cmd := exec.Command("go", "mod", "init", "test.example.com/multi-test")
	cmd.Dir = tmpDir
	AssertNoError(t, cmd.Run())

	// Create multiple Go files
	files := map[string]string{
		"file1.go": `package main
func f1() { var x int; _ = x }`,
		"file2.go": `package main
import "fmt"
func f2() { fmt.Println() }`,
	}

	for filename, content := range files {
		CreateTestFile(t, tmpDir, filename, content)
	}

	// Set up environment
	originalEnv := CaptureEnvironment()
	defer func() {
		NewMockEnvironment().Restore(originalEnv)
	}()

	mockEnv := NewMockEnvironment()
	mockEnv.Set("REPO_PATH", tmpDir)
	mockEnv.Apply()

	// Test batch operations
	operations := []interface{}{
		map[string]interface{}{
			"type":   "lint",
			"target": "file1.go",
		},
		map[string]interface{}{
			"type":   "lint",
			"target": "file2.go",
		},
		map[string]interface{}{
			"type":   "lint",
			"target": ".",
		},
	}

	args := map[string]interface{}{
		"operations": operations,
	}

	// Test through the batch handler
	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      1,
	}

	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	handleBatchOperations(msg, encoder, args)

	// Parse response
	var response MCPMessage
	if err := json.Unmarshal([]byte(buf.String()), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	AssertEqual(t, response.Error, nil)

	result := response.Result.(map[string]interface{})
	results := result["results"].([]interface{})
	AssertEqual(t, len(results), 3)

	// Check each result
	for i, result := range results {
		opResult := result.(map[string]interface{})
		AssertEqual(t, opResult["status"], "Success")

		// Verify the target matches
		opData := opResult["result"].(map[string]interface{})
		expectedTarget := operations[i].(map[string]interface{})["target"]
		AssertEqual(t, opData["target"], expectedTarget)
	}
}

// BenchmarkIntegrationLint benchmarks the lint operation on real code
func BenchmarkIntegrationLint(b *testing.B) {
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		b.Skip("golangci-lint not available for integration tests")
	}

	tmpDir := CreateTempDir(b)

	// Initialize Go module
	cmd := exec.Command("go", "mod", "init", "test.example.com/bench-test")
	cmd.Dir = tmpDir
	AssertNoError(b, cmd.Run())

	// Create a larger Go file for benchmarking
	var content strings.Builder
	content.WriteString("package main\n\n")
	content.WriteString("import \"fmt\"\n\n")

	// Generate many functions
	for i := 0; i < 100; i++ {
		content.WriteString(fmt.Sprintf(`
func BenchmarkFunc%d() {
	var x%d int
	for j := 0; j < 10; j++ {
		x%d += j
	}
	fmt.Printf("Value: %%d\n", x%d)
}
`, i, i, i, i))
	}

	CreateTestFile(b, tmpDir, "bench.go", content.String())

	// Set up environment
	originalEnv := CaptureEnvironment()
	defer func() {
		NewMockEnvironment().Restore(originalEnv)
	}()

	mockEnv := NewMockEnvironment()
	mockEnv.Set("REPO_PATH", tmpDir)
	mockEnv.Apply()

	args := map[string]interface{}{
		"target": "bench.go",
		"format": "json",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := toolLint(args)
		if err != nil {
			b.Fatalf("Lint failed: %v", err)
		}
	}
}

// Helper function to marshal JSON and ignore errors (for test constants)
func MustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}