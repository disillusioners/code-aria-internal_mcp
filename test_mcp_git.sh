#!/bin/bash

# Test script for mcp-git server
set -e

echo "Testing mcp-git server functionality..."

# Create a temporary test repository
TEST_DIR="/tmp/mcp-git-test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git repo
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create some test files and commits
echo "Hello World" > test.txt
git add test.txt
git commit -m "Initial commit"

echo "Modified content" > test.txt
mkdir subdir
echo "Subdir file" > subdir/sub.txt

echo "Test repository created at $TEST_DIR"

# Set REPO_PATH environment variable
export REPO_PATH="$TEST_DIR"

# Test the mcp-git server
# Build the mcp-git binary first from the correct directory
echo "Building mcp-git binary..."
cd /Users/nguyenminhkha/All/Code/opensource-projects/code-aria/code-aria-internal_mcp
go build -o mcp-git-binary ./cmd/mcp-git
cd /Users/nguyenminhkha/All/Code/opensource-projects/code-aria/code-aria-internal_mcp

echo "Testing git status operation..."
echo '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"get_git_status"}]}}' | ./mcp-git-binary

echo ""
echo "Testing commit history operation..."
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"get_commit_history","file_path":"test.txt","limit":5}]}}' | ./mcp-git-binary

echo ""
echo "Testing file diff operation..."
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"get_file_diff","file_path":"test.txt"}]}}' | ./mcp-git-binary

echo ""
echo "Testing changed files operation..."
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"get_changed_files","comparison_type":"working"}]}}' | ./mcp-git-binary

echo ""
echo "Test completed successfully!"