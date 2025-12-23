#!/bin/bash

# Test script for get_all_working_changes tool

echo "Testing get_all_working_changes tool..."

# Create a temporary test repository
TEST_DIR="/tmp/mcp_test_$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git repo
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create initial files
echo "package main

import \"fmt\"

func main() {
    fmt.Println(\"Hello World\")
}" > main.go

echo "package main

func helper() string {
    return \"helper function\"
}" > utils.go

git add .
git commit -m "Initial commit"

# Make some changes
echo "package main

import \"fmt\"

func main() {
    fmt.Println(\"Hello Code-Aria\")
}" > main.go

echo "package main

func helper() string {
    return \"updated helper function\"
}

func newFunc() string {
    return \"new function\"
}" > utils.go

# Add a new file
echo "package main

const VERSION = \"1.0.0\"" > version.go

# Delete a file (simulate)
# rm deleted.go  # We'll skip this for now

echo "Created test repository with changes:"
echo "Modified: main.go, utils.go"
echo "Added: version.go"
echo ""

# Export REPO_PATH for MCP server
export REPO_PATH="$TEST_DIR"

# Test the MCP server
echo "Testing get_all_working_changes tool..."

# Create test input
cat > test_input.json << 'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "apply_operations",
    "arguments": {
      "operations": [
        {
          "type": "get_all_working_changes"
        }
      ]
    }
  }
}
EOF

# Run the MCP server with test input
echo "Running: echo '$(cat test_input.json)' | ./cmd/mcp-git/mcp-git"
echo '$(cat test_input.json)' | ./cmd/mcp-git/mcp-git > test_output.json

echo ""
echo "=== MCP Server Output ==="
cat test_output.json

# Cleanup
cd /
rm -rf "$TEST_DIR"
rm -f test_input.json test_output.json

echo ""
echo "Test completed!"