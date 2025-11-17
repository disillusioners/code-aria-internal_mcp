# Code-Aria Internal MCP Servers

This project contains a set of Model Context Protocol (MCP) servers that are used to interact with codebases. These servers provide tools for file system operations, code analysis, git operations, and code editing.

## Overview

The MCP servers in this project are standalone executables that communicate via stdio using the Model Context Protocol. They are designed to be used by AI agents and other tools that need to interact with codebases programmatically.

## MCP Servers

### 1. mcp-filesystem

Provides file system operations:
- `read_file(path)` - Read file contents
- `list_directory(path)` - List files in a directory
- `get_file_tree(root_path, max_depth)` - Get directory tree structure
- `file_exists(path)` - Check if a file or directory exists
- `create_directory(path)` - Create a directory and all parent directories

### 2. mcp-codebase

Provides code analysis tools:
- `search_code(query, file_patterns)` - Search for code patterns or keywords (regex)
- `get_file_dependencies(file_path)` - Get imports and dependencies for a file
- `analyze_function(function_name, file_path)` - Get function details and signature
- `get_code_context(file_path, line_range)` - Get code with surrounding context

### 3. mcp-git

Provides git operations:
- `get_git_status(repo_path)` - Get git status for the repository
- `get_file_diff(file_path, base_branch)` - Get diff for a file against a base branch
- `get_commit_history(file_path, limit)` - Get commit history for a file

### 4. mcp-code-edit

Provides code modification tools:
- `apply_diff(file_path, old_content, new_content)` - Apply a diff to a file (replace old_content with new_content)
- `replace_code(file_path, old_code, new_code)` - Replace a code block in a file
- `create_file(file_path, content)` - Create a new file with content
- `delete_file(file_path)` - Delete a file

## Prerequisites

- Go 1.24.1 or higher
- Git (for mcp-git server)

## Installation

### Quick Start

Build and install all MCP servers to your PATH:

```bash
make mcp-servers
```

This command will:
1. Build all 4 MCP server executables (`mcp-filesystem`, `mcp-codebase`, `mcp-git`, `mcp-code-edit`)
2. Automatically detect the best installation directory (`~/bin`, `~/.local/bin`, or `/usr/local/bin`)
3. Copy executables to the installation directory
4. Set executable permissions
5. Warn you if the installation directory needs to be added to your PATH

### Makefile Targets

The Makefile provides several targets for managing MCP servers:

**`make mcp-servers`** - Build and install all MCP servers (recommended)
- Builds all executables and installs them to PATH
- Convenience target that combines build and install

**`make build-mcp-servers`** - Build only
- Compiles all 4 MCP server executables
- Outputs executables to the project root directory
- Does not install them

**`make install-mcp-servers`** - Install built executables
- Requires executables to be built first (automatically builds if missing)
- Copies executables to a directory in PATH
- Automatically detects the best installation location

**`make clean-mcp-servers`** - Clean build artifacts
- Removes all MCP server executables from the project root

### Manual Build

If you prefer to build manually:

```bash
# Build all MCP servers
go build -o mcp-filesystem ./cmd/mcp-filesystem
go build -o mcp-codebase ./cmd/mcp-codebase
go build -o mcp-git ./cmd/mcp-git
go build -o mcp-code-edit ./cmd/mcp-code-edit
```

### Installation Directory Selection

The `install-mcp-servers` target automatically selects an installation directory in this order:

1. `~/bin` (if writable or can be created)
2. `~/.local/bin` (if writable or can be created)
3. `/usr/local/bin` (if writable)

**Note**: If the installation directory is not in your PATH, add it to your shell profile:

```bash
# For zsh
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# For bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

## Usage

### Environment Variables

All MCP servers use the `REPO_PATH` environment variable to determine the repository root. Paths passed to tools are resolved relative to this directory.

```bash
export REPO_PATH=/path/to/your/repository
```

### Running Servers

MCP servers communicate via stdio using JSON-RPC 2.0. They are typically invoked by MCP clients (like Genkit's MCP plugin) rather than run directly.

For testing purposes, you can run a server directly:

```bash
# Set repository path
export REPO_PATH=/path/to/repo

# Run server (will read from stdin, write to stdout)
./mcp-filesystem
```

### Protocol

The servers implement the Model Context Protocol (MCP) version 2024-11-05. They communicate using JSON-RPC 2.0 messages over stdio:

1. **Initialize**: Client sends initialize request, server responds with capabilities
2. **Initialized**: Client sends initialized notification
3. **Tools List**: Client can request available tools via `tools/list`
4. **Tool Call**: Client can call tools via `tools/call`

## Project Structure

```
code-aria-internal_mcp/
├── cmd/
│   ├── mcp-filesystem/
│   │   └── main.go
│   ├── mcp-codebase/
│   │   └── main.go
│   ├── mcp-git/
│   │   └── main.go
│   └── mcp-code-edit/
│       └── main.go
├── Makefile
├── go.mod
└── README.md
```

## Dependencies

The MCP servers use only the Go standard library - no external dependencies are required. This keeps the servers lightweight and easy to deploy.

## Integration

These MCP servers are designed to be used with:
- **Genkit MCP Plugin**: Official plugin for Google's Genkit framework
- **MCP Clients**: Any client that implements the Model Context Protocol
- **AI Agents**: Agents that need to interact with codebases programmatically

## Development

### Building

```bash
make build-mcp-servers
```

### Testing

Each server can be tested manually by sending JSON-RPC messages via stdin:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./mcp-filesystem
```

### Cleaning

```bash
make clean-mcp-servers
```

## License

This project is part of the Code-Aria system.
