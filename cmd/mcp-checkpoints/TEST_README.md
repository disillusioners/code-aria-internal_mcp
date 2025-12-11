# Checkpoint MCP Tool Test Suite

This document describes the comprehensive test suite for the checkpoint MCP tool, including how to run tests, test structure, and best practices.

## Test Overview

The checkpoint MCP tool test suite includes:

1. **Unit Tests** - Testing individual functions and methods
2. **Integration Tests** - Testing component interactions
3. **Edge Case Tests** - Testing unusual scenarios and error conditions
4. **Performance Tests** - Benchmarking various operations
5. **Test Utilities** - Helper functions for writing tests

## Test Files

### Core Test Files

- `checkpoint_manager_test.go` - Unit tests for CheckpointManager
- `checkpoint_operations_test.go` - Tests for MCP tool operations
- `integration_test.go` - Integration tests for MCP protocol compliance

### Additional Test Files

- `checkpoint_manager_edge_test.go` - Edge case and boundary tests
- `checkpoint_benchmark_test.go` - Performance benchmarks
- `test_utils_example_test.go` - Examples of using test utilities

### Utility Files

- `test_utils.go` - Helper utilities for writing tests
- `test_suite.go` - Test suite runner and management utilities

## Running Tests

### Quick Start

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...
```

### Using the Test Suite Runner

```go
// In your test file or main program
runner := checkpoint.NewTestSuiteRunner().
    Verbose().
    WithCoverage().
    WithBenchmarks().
    WithRaceDetector()

// Run all tests
err := runner.RunAllTests()
if err != nil {
    log.Fatal(err)
}

// Generate coverage report
err = runner.GenerateCoverageReport()
```

### Running Specific Test Categories

```bash
# Run only unit tests
go test -v -run "Test.*Unit" ./...

# Run edge case tests
go test -v -run "Test.*Edge" ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

### Individual Test Commands

```bash
# Run unit tests for checkpoint manager
go test -v checkpoint_manager_test.go

# Run edge case tests
go test -v checkpoint_manager_edge_test.go

# Run benchmarks
go test -v -bench=. checkpoint_benchmark_test.go

# Run with race detector
go test -race ./...

# Run with timeout
go test -timeout=30s ./...
```

## Test Structure

### Unit Tests (`checkpoint_manager_test.go`)

Tests for core functionality:
- `TestNewCheckpointManager` - Manager initialization
- `TestCreateCheckpoint` - Checkpoint creation
- `TestListCheckpoints` - Checkpoint listing
- `TestRestoreCheckpoint` - Checkpoint restoration
- `TestDeleteCheckpoint` - Checkpoint deletion
- `TestGenerateID` - ID generation
- `TestCopyFile` - File copying operations

### Tool Operation Tests (`checkpoint_operations_test.go`)

Tests for MCP tool handlers:
- `TestToolCreateCheckpoint` - create_checkpoint tool
- `TestToolListCheckpoints` - list_checkpoints tool
- `TestToolGetCheckpoint` - get_checkpoint tool
- `TestToolRestoreCheckpoint` - restore_checkpoint tool
- `TestToolDeleteCheckpoint` - delete_checkpoint tool
- `TestToolGetCheckpointInfo` - get_checkpoint_info tool

### Edge Case Tests (`checkpoint_manager_edge_test.go`)

Tests for unusual scenarios:
- Non-existent repository paths
- Special characters in names
- Very long descriptions
- Binary files
- Nested directory structures
- File permission variations
- Concurrent access
- Symlinks
- Ignored files

### Benchmark Tests (`checkpoint_benchmark_test.go`)

Performance benchmarks for:
- Checkpoint creation with various file sizes
- Checkpoint listing
- Checkpoint restoration
- Checkpoint deletion
- ID generation
- File copying
- Concurrent operations
- Memory usage

## Using Test Utilities

The `TestHelper` class provides convenient methods for setting up test environments:

```go
func TestMyFeature(t *testing.T) {
    helper := NewTestHelper(t)
    defer helper.Cleanup()

    // Create test files
    err := helper.CreateTestFiles(map[string]string{
        "main.go": "package main",
        "config.json": `{"debug": true}`,
    })
    if err != nil {
        t.Fatalf("Failed to create test files: %v", err)
    }

    // Create checkpoint
    checkpoint, err := helper.CreateCheckpoint("test", "Test checkpoint")
    if err != nil {
        t.Fatalf("Failed to create checkpoint: %v", err)
    }

    // Assertions
    helper.AssertFileExists(t, "main.go")
    helper.AssertFileContent(t, "main.go", "package main")
    helper.AssertCheckpointCount(t, 1)
}
```

## Test Environment Setup

### Prerequisites

1. Go 1.21 or later
2. Git installed and in PATH
3. Write permissions in the test directory

### Environment Variables

- `REPO_PATH` - Set automatically by test utilities
- `TMPDIR` - Optional, for specifying temporary directory location

## Best Practices

### Writing Tests

1. **Use TestHelper** - Leverage the TestHelper class for common operations
2. **Clean up** - Always clean up created checkpoints and temporary files
3. **Assert explicitly** - Use assertion methods rather than manual checks
4. **Test edge cases** - Consider unusual inputs and error conditions
5. **Test concurrency** - Verify thread safety where applicable

### Test Organization

1. **Group related tests** - Use table-driven tests for similar scenarios
2. **Descriptive names** - Use clear, descriptive test function names
3. **Documentation** - Add comments for complex test scenarios
4. **Subtests** - Use subtests for related test cases

### Performance Considerations

1. **Reuse setup** - Use setup functions to avoid repeated initialization
2. **Parallel tests** - Use `t.Parallel()` for independent tests
3. **Benchmark focus** - Focus benchmarks on critical paths
4. **Memory cleanup** - Ensure proper cleanup to avoid memory leaks

## Coverage Reports

### Generating Coverage

```bash
# Run tests with coverage
go test -coverprofile=coverage.out -covermode=atomic ./...

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# View coverage percentage
go tool cover -func=coverage.out
```

### Coverage Goals

- Aim for >80% code coverage
- Focus on critical paths and error handling
- Ensure all public functions are tested
- Cover edge cases and error conditions

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test Checkpoint MCP

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'

    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...

    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## Troubleshooting

### Common Issues

1. **Permission denied** - Check file permissions and REPO_PATH
2. **Git not found** - Ensure Git is installed and in PATH
3. **Test timeouts** - Increase timeout or optimize test performance
4. **Race conditions** - Run with `-race` flag to detect

### Debug Tips

1. **Verbose output** - Use `-v` flag for detailed output
2. **Test logging** - Add `t.Log()` statements for debugging
3. **Breakpoints** - Use Delve debugger for complex issues
4. **Test isolation** - Run tests individually to isolate failures

## Contributing

When adding new tests:

1. Follow existing test patterns and naming conventions
2. Use TestHelper for common operations
3. Add appropriate assertions
4. Document complex test scenarios
5. Update this README if adding new test categories

## Performance Benchmarks

### Expected Benchmarks

As of the latest version, expected performance characteristics:

| Operation | Files | File Size | Expected Time |
|-----------|-------|-----------|---------------|
| Create Checkpoint | 10 | 1KB | < 10ms |
| Create Checkpoint | 100 | 10KB | < 100ms |
| List Checkpoints | 1000 | - | < 10ms |
| Restore Checkpoint | 10 | 1MB | < 50ms |
| Delete Checkpoint | 100 | 10KB | < 20ms |

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkCreateCheckpoint ./...

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./...
```

## Future Enhancements

Planned improvements to the test suite:

1. **Fuzzing** - Add fuzz tests for input validation
2. **Property testing** - Add property-based tests
3. **Load testing** - Add tests for high-load scenarios
4. **Cross-platform** - Enhance Windows/macOS compatibility
5. **Integration with CI** - Improve CI/CD integration