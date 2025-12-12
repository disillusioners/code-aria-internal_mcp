# Tests for mcp-lang-go

This directory contains comprehensive tests for the mcp-lang-go MCP server.

## Test Structure

### Unit Tests

- **`mcp_test.go`** - Tests for MCP protocol handling
  - Initialization handshake
  - Tool list functionality
  - Tool call processing
  - Batch operations
  - Error handling
  - JSON serialization
  - Concurrent access

- **`lint_test.go`** - Tests for Go linting functionality
  - Lint tool execution
  - Output parsing
  - Severity determination
  - Path resolution
  - Configuration handling
  - Text and JSON formats

### Integration Tests

- **`integration_test.go`** - End-to-end integration tests
  - Full MCP workflow
  - Real golangci-lint execution
  - Custom configuration files
  - Multiple operations
  - Error scenarios
  - Performance benchmarks

### Test Utilities

- **`testutils.go`** - Common testing utilities
  - Mock environment handling
  - Test transport layer
  - MCP client simulation
  - Assertion helpers
  - Temporary file/directory creation

## Running Tests

### Prerequisites

- Go 1.24+ installed
- `golangci-lint` in PATH (for integration tests)

### Test Commands

From the `code-aria-internal_mcp` directory:

```bash
# Run all unit tests
make test-mcp-lang-go

# Run tests with coverage report
make test-mcp-lang-go-coverage

# Run integration tests (requires golangci-lint)
make test-mcp-lang-go-integration

# Run benchmarks
make bench-mcp-lang-go

# Run all tests
make test-all
```

### Running Tests Directly

```bash
# Change to the mcp-lang-go directory
cd cmd/mcp-lang-go

# Run all tests
go test -v ./...

# Run specific test file
go test -v -run TestHandleInitialize

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run integration tests (requires build tag)
go test -tags=integration -v ./...
```

## Test Coverage

The test suite covers:

- **MCP Protocol Implementation**
  - Message serialization/deserialization
  - Initialize handshake
  - Tool discovery and calling
  - Error responses

- **Linting Functionality**
  - Command execution
  - Output parsing (line-number format)
  - Severity classification
  - Configuration file support
  - Multiple output formats

- **Error Handling**
  - Missing environment variables
  - Invalid tool parameters
  - Command execution failures
  - Malformed input

- **Concurrency**
  - Concurrent message processing
  - Thread safety

## Integration Test Requirements

Integration tests require:
1. `golangci-lint` installed and in PATH
2. Network access (for downloading Go modules)
3. File system write permissions

To install golangci-lint:
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2

# Or from releases page
# https://github.com/golangci/golangci-lint/releases
```

## Mock Testing

Tests use mocking to:
- Simulate MCP clients
- Control environment variables
- Create temporary Go projects
- Mock command execution where appropriate

## Performance

Benchmarks are included for:
- Message processing
- Output parsing
- Lint execution

Run benchmarks with:
```bash
go test -bench=. -benchmem ./...
```

## Adding New Tests

When adding new functionality:

1. Add unit tests in the appropriate test file
2. Add integration tests if it interacts with external tools
3. Use the provided test utilities for common operations
4. Update this README if new test types are added

### Test Naming Conventions

- `Test<FunctionName>` for unit tests
- `TestIntegration<Feature>` for integration tests
- `Benchmark<FunctionName>` for benchmarks

### Example Test Structure

```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected ExpectedType
        wantErr  bool
    }{
        // Test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation...
        })
    }
}
```

## Debugging Failed Tests

1. Use `-v` flag for verbose output
2. Check test logs for detailed error messages
3. Use `t.Log()` or `t.Logf()` for debugging output
4. For integration tests, check that golangci-lint is properly installed

## CI/CD Integration

These tests are designed to run in CI/CD environments:
- Unit tests can run without external dependencies
- Integration tests require golangci-lint installation
- Tests use temporary directories for isolation
- Environment is properly restored after tests