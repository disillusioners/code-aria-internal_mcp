package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSuiteRunner helps run comprehensive test suites for the checkpoint MCP tool
type TestSuiteRunner struct {
	verbose      bool
	coverage     bool
	benchmarks   bool
	raceDetector bool
	timeout      time.Duration
}

// NewTestSuiteRunner creates a new test suite runner with default options
func NewTestSuiteRunner() *TestSuiteRunner {
	return &TestSuiteRunner{
		verbose:      false,
		coverage:     false,
		benchmarks:   false,
		raceDetector: false,
		timeout:      10 * time.Minute,
	}
}

// Verbose enables verbose output
func (tsr *TestSuiteRunner) Verbose() *TestSuiteRunner {
	tsr.verbose = true
	return tsr
}

// WithCoverage enables coverage reporting
func (tsr *TestSuiteRunner) WithCoverage() *TestSuiteRunner {
	tsr.coverage = true
	return tsr
}

// WithBenchmarks includes benchmark tests
func (tsr *TestSuiteRunner) WithBenchmarks() *TestSuiteRunner {
	tsr.benchmarks = true
	return tsr
}

// WithRaceDetector enables the race detector
func (tsr *TestSuiteRunner) WithRaceDetector() *TestSuiteRunner {
	tsr.raceDetector = true
	return tsr
}

// WithTimeout sets the test timeout
func (tsr *TestSuiteRunner) WithTimeout(timeout time.Duration) *TestSuiteRunner {
	tsr.timeout = timeout
	return tsr
}

// RunUnitTests runs only unit tests
func (tsr *TestSuiteRunner) RunUnitTests() error {
	fmt.Println("=== Running Unit Tests ===")

	args := []string{"test", "-v", "./..."}
	if tsr.coverage {
		args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
	}
	if tsr.raceDetector {
		args = append(args, "-race")
	}

	return tsr.runGoCommand(args)
}

// RunBenchmarks runs only benchmark tests
func (tsr *TestSuiteRunner) RunBenchmarks() error {
	fmt.Println("=== Running Benchmark Tests ===")

	args := []string{"test", "-bench=.", "-benchmem"}
	if tsr.verbose {
		args = append(args, "-v")
	}
	if tsr.coverage {
		args = append(args, "-coverprofile=bench_coverage.out")
	}

	return tsr.runGoCommand(args)
}

// RunEdgeCaseTests runs edge case and integration tests
func (tsr *TestSuiteRunner) RunEdgeCaseTests() error {
	fmt.Println("=== Running Edge Case Tests ===")

	args := []string{"test", "-v", "-run", "(Test.*Edge|Test.*Concurrent|Test.*Complex)"}
	if tsr.raceDetector {
		args = append(args, "-race")
	}

	return tsr.runGoCommand(args)
}

// RunAllTests runs all tests including unit, edge cases, and benchmarks
func (tsr *TestSuiteRunner) RunAllTests() error {
	fmt.Println("=== Running Complete Test Suite ===")

	// Run unit tests first
	if err := tsr.RunUnitTests(); err != nil {
		return fmt.Errorf("unit tests failed: %v", err)
	}

	// Run edge case tests
	if err := tsr.RunEdgeCaseTests(); err != nil {
		return fmt.Errorf("edge case tests failed: %v", err)
	}

	// Run benchmarks if requested
	if tsr.benchmarks {
		if err := tsr.RunBenchmarks(); err != nil {
			return fmt.Errorf("benchmarks failed: %v", err)
		}
	}

	return nil
}

// GenerateCoverageReport generates an HTML coverage report
func (tsr *TestSuiteRunner) GenerateCoverageReport() error {
	if !tsr.coverage {
		return fmt.Errorf("coverage not enabled. Use WithCoverage() to enable")
	}

	fmt.Println("=== Generating Coverage Report ===")

	// Check if coverage file exists
	if _, err := os.Stat("coverage.out"); os.IsNotExist(err) {
		return fmt.Errorf("coverage.out not found. Run tests with coverage first")
	}

	// Generate HTML report
	args := []string{"tool", "cover", "-html=coverage.out", "-o", "coverage.html"}
	if err := tsr.runGoCommand(args); err != nil {
		return err
	}

	fmt.Println("Coverage report generated: coverage.html")
	return nil
}

// runGoCommand executes a go command with the given arguments
func (tsr *TestSuiteRunner) runGoCommand(args []string) error {
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set timeout
	if tsr.timeout > 0 {
		timer := time.AfterFunc(tsr.timeout, func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		})
		defer timer.Stop()
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("command failed: %v", err)
	}

	return nil
}

// TestCategory represents a category of tests
type TestCategory string

const (
	TestCategoryUnit         TestCategory = "unit"
	TestCategoryIntegration  TestCategory = "integration"
	TestCategoryEdgeCase     TestCategory = "edgecase"
	TestCategoryPerformance  TestCategory = "performance"
	TestCategoryAll          TestCategory = "all"
)

// GetTestFiles returns test files for a given category
func GetTestFiles(category TestCategory) []string {
	switch category {
	case TestCategoryUnit:
		return []string{
			"checkpoint_manager_test.go",
			"checkpoint_operations_test.go",
		}
	case TestCategoryIntegration:
		return []string{
			"integration_test.go",
		}
	case TestCategoryEdgeCase:
		return []string{
			"checkpoint_manager_edge_test.go",
			"test_utils_example_test.go",
		}
	case TestCategoryPerformance:
		return []string{
			"checkpoint_benchmark_test.go",
		}
	case TestCategoryAll:
		return []string{
			"checkpoint_manager_test.go",
			"checkpoint_operations_test.go",
			"integration_test.go",
			"checkpoint_manager_edge_test.go",
			"checkpoint_benchmark_test.go",
			"test_utils_example_test.go",
		}
	default:
		return []string{}
	}
}

// RunTestsByCategory runs tests for a specific category
func RunTestsByCategory(category TestCategory, verbose bool) error {
	files := GetTestFiles(category)
	if len(files) == 0 {
		return fmt.Errorf("no test files found for category: %s", category)
	}

	fmt.Printf("=== Running %s Tests ===\n", strings.Title(string(category)))

	args := []string{"test"}
	if verbose {
		args = append(args, "-v")
	}

	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			fmt.Printf("Warning: Test file %s not found, skipping\n", file)
			continue
		}
		args = append(args, file)
	}

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ValidateTestEnvironment checks if the test environment is properly set up
func ValidateTestEnvironment() error {
	fmt.Println("=== Validating Test Environment ===")

	// Check if we're in the right directory
	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		return fmt.Errorf("main.go not found. Please run tests from the mcp-checkpoints directory")
	}

	// Check for test files
	testFiles := GetTestFiles(TestCategoryAll)
	foundFiles := 0
	for _, file := range testFiles {
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			foundFiles++
		}
	}

	fmt.Printf("Found %d/%d test files\n", foundFiles, len(testFiles))

	// Check go.mod
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found. Please ensure you're in a Go module")
	}

	// Check if git is available (required for tests)
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH. Git is required for running tests")
	}

	fmt.Println("Test environment is valid")
	return nil
}

// CleanupTestArtifacts cleans up test artifacts
func CleanupTestArtifacts() error {
	fmt.Println("=== Cleaning Up Test Artifacts ===")

	artifacts := []string{
		"coverage.out",
		"coverage.html",
		"bench_coverage.out",
		"*.test",
		".mcp-checkpoints", // Leftover from failed tests
	}

	for _, pattern := range artifacts {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			if strings.Contains(match, ".mcp-checkpoints") {
				// Be careful with recursive delete
				err = os.RemoveAll(match)
			} else {
				err = os.Remove(match)
			}

			if err != nil {
				fmt.Printf("Warning: Failed to remove %s: %v\n", match, err)
			} else {
				fmt.Printf("Removed: %s\n", match)
			}
		}
	}

	return nil
}

// ShowTestSummary provides a summary of available tests
func ShowTestSummary() {
	fmt.Println("=== Checkpoint MCP Test Suite Summary ===")
	fmt.Println()

	categories := []TestCategory{
		TestCategoryUnit,
		TestCategoryIntegration,
		TestCategoryEdgeCase,
		TestCategoryPerformance,
	}

	for _, category := range categories {
		files := GetTestFiles(category)
		fmt.Printf("%s Tests:\n", strings.Title(string(category)))
		for _, file := range files {
			if _, err := os.Stat(file); !os.IsNotExist(err) {
				fmt.Printf("  ✓ %s\n", file)
			} else {
				fmt.Printf("  ✗ %s (missing)\n", file)
			}
		}
		fmt.Println()
	}

	fmt.Println("Available utilities:")
	fmt.Println("  - test_utils.go: Helper utilities for writing tests")
	fmt.Println("  - test_utils_example_test.go: Examples of using test utilities")
	fmt.Println("  - test_suite.go: Test suite runner and utilities")
	fmt.Println()
}

// Example usage of TestSuiteRunner
func ExampleRunTestSuite() {
	runner := NewTestSuiteRunner().
		Verbose().
		WithCoverage().
		WithBenchmarks().
		WithRaceDetector().
		WithTimeout(15 * time.Minute)

	if err := runner.ValidateTestEnvironment(); err != nil {
		fmt.Printf("Test environment validation failed: %v\n", err)
		return
	}

	runner.ShowTestSummary()

	if err := runner.RunAllTests(); err != nil {
		fmt.Printf("Test suite failed: %v\n", err)
		return
	}

	if err := runner.GenerateCoverageReport(); err != nil {
		fmt.Printf("Failed to generate coverage report: %v\n", err)
	}

	fmt.Println("Test suite completed successfully!")
}