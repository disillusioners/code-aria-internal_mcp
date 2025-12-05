#!/bin/bash

# Simple test for mcp-git server
set -e

echo "Simple test for mcp-git server..."

# Create a temporary test repository
TEST_DIR="/tmp/mcp-git-simple-test"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git repo
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create a test file
echo "Hello World" > test.txt
git add test.txt
git commit -m "Initial commit"

echo "Modified content" > test.txt

echo "Test repository created at $TEST_DIR"

# Set REPO_PATH environment variable
export REPO_PATH="$TEST_DIR"

# Build the mcp-git binary
echo "Building mcp-git binary..."
cd /Users/nguyenminhkha/All/Code/opensource-projects/code-aria/code-aria-internal_mcp
go build -o mcp-git-binary ./cmd/mcp-git

# Test just the initialization
echo "Testing initialization..."
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}' | ./mcp-git-binary

echo ""
echo "Testing initialized notification..."
echo '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}' | ./mcp-git-binary

echo ""
echo "Testing tools/list..."
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./mcp-git-binary

echo "Simple test completed!"