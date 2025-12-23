# Code-Aria Internal MCP Servers

This project contains a set of Model Context Protocol (MCP) servers that are used to interact with codebases. These servers provide tools for file system operations, code analysis, git operations, code editing, secure bash command execution, PowerShell command execution, comprehensive system information gathering, checkpoint management, document repository access, and Go language linting.

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

**Overview Queries (File List)**:
- `get_git_status(repo_path)` - Get git status for uncommitted changes (file list)
- `get_changed_files(comparison_type, ...)` - Get list of changed files for different scenarios:
  - `comparison_type: "working"` - Uncommitted changes (same as get_git_status but structured)
  - `comparison_type: "branch"` - Files changed between branches (requires `base_branch`, optional `target_branch`)
  - `comparison_type: "commits"` - Files changed between commits (requires `base_commit`, optional `target_commit`)
  - `comparison_type: "last_commit"` - Files changed in last commit (HEAD~1..HEAD)
  - Optional `include_status` (boolean) - Include file status (A/M/D)

**Detail Queries (Per-File Diff)**:
- `get_file_diff(file_path, ...)` - Get detailed diff for a file with multiple comparison modes:
  - Branch comparison: `get_file_diff(file_path, base_branch="main")` - Compare file against a branch
  - Uncommitted changes: `get_file_diff(file_path, compare_working=true)` - Compare working directory vs HEAD
  - Commit comparison: `get_file_diff(file_path, base_commit="abc123", target_commit="def456")` - Compare between commits
  - Last commit: `get_file_diff(file_path, base_commit="HEAD~1", target_commit="HEAD")` - Compare last commit
  - Working directory (alternative): `get_file_diff(file_path, base_branch="HEAD")` - Compare working directory vs HEAD

**Metadata Queries**:
- `get_commit_history(file_path, limit)` - Get commit history for a file

### 4. mcp-code-edit

Provides code modification tools:
- `apply_diff(file_path, old_content, new_content)` - Apply a diff to a file (replace old_content with new_content)
- `replace_code(file_path, old_code, new_code)` - Replace a code block in a file (also accepts `old_content`/`new_content` as aliases)
- `create_file(file_path, content)` - Create a new file with content
- `delete_file(file_path)` - Delete a file
- `rename_file(old_path, new_path)` - Rename or move a file (also accepts `move_file` as alias)
- `copy_file(source_path, destination_path)` - Copy a file to a new location

### 5. mcp-bash

Provides secure bash command execution with comprehensive security measures:
- `execute_command(command, timeout, working_directory, allow_shell_access, environment_vars)` - Execute a single bash command with security restrictions
- `execute_script(script, timeout, working_directory, allow_shell_access, environment_vars, script_name)` - Execute multi-line bash scripts with enhanced security controls
- `check_command_exists(command, search_paths)` - Check if a command is available in the system PATH

**Security Features:**
- Command validation with allow/block lists
- Input sanitization and UTF-8 validation
- Working directory restrictions
- Environment variable filtering
- Comprehensive audit logging
- Timeout management and resource limits

### 6. mcp-powershell

Provides secure PowerShell command execution designed specifically for Windows environments:
- `execute_command(command, timeout, working_directory, allow_shell_access, environment_vars)` - Execute a single PowerShell command with security restrictions
- `execute_script(script, timeout, working_directory, allow_shell_access, environment_vars, script_name)` - Execute multi-line PowerShell scripts with enhanced security controls
- `check_command_exists(command, search_paths)` - Check if a PowerShell cmdlet, function, or external command is available

**Windows-Specific Features:**
- Supports both Windows PowerShell and PowerShell Core (pwsh)
- Native support for PowerShell cmdlets (Get-ChildItem, Set-Content, etc.)
- Handles Windows file paths and executables (.exe, .ps1, .cmd, .bat)
- Automatic detection of PowerShell installation
- Built-in Windows command support (dir, type, copy, move, etc.)

**Security Features:**
- PowerShell-specific command validation with allow/block lists
- Protection against dangerous PowerShell constructs (Invoke-Expression, execution policy changes, etc.)
- Windows-specific security pattern detection
- Input sanitization and UTF-8 validation
- Working directory restrictions
- Environment variable filtering
- Comprehensive audit logging
- Timeout management and resource limits

### 7. mcp-systeminfo

Provides comprehensive system information gathering to help LLMs understand the operating environment:
- `get_system_info()` - Complete system overview (OS, hardware, environment, tools, network, repositories)
- `get_os_info()` - Operating system details (name, version, architecture, distribution)
- `get_hardware_info()` - Hardware information (CPU, memory, storage, displays, network cards)
- `get_environment_info()` - Environment variables and paths (filtered for security)
- `get_shell_info()` - Shell information and capabilities
- `get_development_tools()` - Development tools detection and versions
- `get_network_info()` - Network configuration and connectivity status
- `detect_repositories()` - Version control repository detection
- `check_command(command)` - Check if a command is available and its version
- `get_recommendations()` - System-specific recommendations for development

**Cross-Platform Support:**
- **Windows**: Full support with PowerShell and Windows-specific commands
- **Linux**: Comprehensive support with /proc filesystem and standard Unix tools
- **macOS**: Native support with system commands and platform-specific features

**Key Features:**
- **OS Detection**: Detailed operating system information including distribution and version
- **Hardware Analysis**: CPU, memory, storage, and network interface information
- **Development Tools**: Automatic detection of compilers, interpreters, package managers, and build tools
- **Repository Detection**: Identifies Git, SVN, and Mercurial repositories with status
- **Network Analysis**: IP addresses, DNS, proxy settings, and connectivity checks
- **Environment Inspection**: Secure environment variable analysis (sensitive data filtered)
- **Smart Recommendations**: Context-aware recommendations based on system configuration
- **Security-First**: Read-only operations with comprehensive input validation and audit logging

**Use Cases:**
- **Environment Context**: Help LLMs understand the development environment before generating code
- **Tool Selection**: Automatically detect available tools and choose appropriate commands
- **Cross-Platform Compatibility**: Generate commands that work on the target OS
- **Resource Planning**: Understand system resources for task planning
- **Repository Awareness**: Detect existing code repositories for context-aware assistance

### 8. mcp-guidelines

Provides read-only access to guidelines from the PostgreSQL database for customizing AI agent behavior:

- `get_guidelines(tenant_id, category, tags, is_active, limit)` - Get guidelines filtered by tenant, category, tags, or active status
- `get_guideline_content(guideline_ids)` - Get full content of specific guidelines by IDs
- `search_guidelines(search_term, tenant_id, category, limit)` - Search guidelines by name, description, or content text

**Key Features:**
- **Database Integration**: Connects directly to PostgreSQL database (same as API/Worker)
- **Flexible Filtering**: Filter by tenant, category, tags, or active status
- **Full-Text Search**: Search across name, description, and content fields
- **Read-Only**: Secure read-only operations with parameterized queries
- **Tenant Isolation**: Support for multi-tenant guideline access

**Use Cases:**
- **Workflow Customization**: Attach guidelines to workflows to customize AI agent behavior
- **Code Standards**: Enforce coding standards and best practices
- **Project-Specific Rules**: Apply project-specific guidelines during code generation
- **Context-Aware Assistance**: Provide relevant guidelines based on task context

**Configuration:**
- Requires `GUIDELINES_DB_DSN` environment variable with PostgreSQL connection string
- Format: `postgres://user:password@host:port/dbname?sslmode=disable`

### 9. mcp-postgres

Provides read-only access to PostgreSQL databases for querying data and inspecting schemas:

- `list_schemas` - List all schemas in the database
- `list_tables(schema)` - List tables in a schema with metadata
- `describe_table(table_name, schema)` - Get detailed table schema (columns, types, constraints, indexes)
- `query(query, params, limit)` - Execute parameterized SELECT queries

**Key Features:**
- **Read-Only Access**: Only SELECT queries allowed, all data modification operations rejected
- **Schema Inspection**: Explore database structure without modifying data
- **Parameterized Queries**: Support for safe parameterized queries to prevent SQL injection
- **Flexible Configuration**: Connection string via environment variable or per-operation override
- **Result Limiting**: Automatic result limiting for safety (default 1000 rows, max 10000)

**Use Cases:**
- **Database Exploration**: Inspect database schemas and table structures
- **Data Querying**: Execute safe SELECT queries to retrieve data
- **Schema Analysis**: Understand database structure for code generation
- **Data Inspection**: Query data for debugging or analysis purposes

**Configuration:**
- Requires `POSTGRES_DB_DSN` environment variable with PostgreSQL connection string
- Format: `postgres://user:password@host:port/dbname?sslmode=disable`
- Can be overridden per-operation via `connection_string` parameter

### 10. mcp-savepoints

Provides checkpoint management for creating and restoring working directory states:

- `create_checkpoint(name, description)` - Create a checkpoint of current working directory changes
- `list_checkpoints()` - List all available checkpoints
- `get_checkpoint(checkpoint_id)` - Get details of a specific checkpoint
- `restore_checkpoint(checkpoint_id)` - Restore a checkpoint to the working directory
- `delete_checkpoint(checkpoint_id)` - Delete a checkpoint
- `get_checkpoint_info(checkpoint_id)` - Get detailed information about a checkpoint including file list
- `apply_operations(operations)` - Execute multiple checkpoint operations in a single batch call

**Key Features:**
- **State Management**: Create snapshots of working directory changes
- **Quick Restoration**: Restore previous states without manual file operations
- **Batch Operations**: Execute multiple operations efficiently in a single call
- **File Tracking**: Track which files are included in each checkpoint
- **Safe Operations**: Non-destructive checkpoint creation and deletion

**Use Cases:**
- **Experimentation**: Try changes safely and revert if needed
- **Branch Comparison**: Save states before switching branches
- **Error Recovery**: Quickly restore working state after failed changes
- **Development Workflow**: Save progress points during complex refactoring

**Storage:**
- Checkpoints are stored in `.mcp/checkpoints` directory in the repository root
- Each checkpoint contains metadata and compressed file changes
- Automatic cleanup of old checkpoints can be configured

### 11. mcp-documents

Provides read-only access to documents from the PostgreSQL database for AI agent context:

- `get_documents(tenant_id, category_id, tags, is_active, limit)` - Get documents filtered by tenant, category, tags, or active status
- `get_document_content(document_ids)` - Get full content of specific documents by IDs
- `search_documents(query, tenant_id, limit)` - Search documents by query text
- `apply_operations(operations)` - Execute multiple document operations in a single batch call

**Key Features:**
- **Database Integration**: Connects directly to PostgreSQL database (same as API/Worker)
- **Flexible Filtering**: Filter by tenant, category, tags, or active status
- **Full-Text Search**: Search across document content with relevance ranking
- **Batch Operations**: Execute multiple operations efficiently in a single call
- **Read-Only**: Secure read-only operations with parameterized queries
- **Tenant Isolation**: Support for multi-tenant document access

**Use Cases:**
- **Context Retrieval**: Provide relevant documents as context for AI agents
- **Knowledge Base**: Access project documentation and guidelines
- **Document Search**: Find documents based on content queries
- **Workflow Integration**: Attach documents to workflows for enhanced AI assistance

**Configuration:**
- Requires `DOCUMENTS_DB_DSN` environment variable with PostgreSQL connection string
- Format: `postgres://user:password@host:port/dbname?sslmode=disable`

### 12. mcp-lang-go

Provides Go language-specific tools for code quality and linting:

- `lint(target, format, config)` - Run golangci-lint on Go code with embedded binary management
- `apply_operations(operations)` - Execute multiple Go operations in a single batch call

**Key Features:**
- **Embedded Binary**: Automatically manages golangci-lint binary (downloads if needed)
- **Flexible Targeting**: Lint specific files, directories, or entire repository
- **Multiple Formats**: Output in JSON or text format
- **Config Support**: Use project-specific `.golangci.yml` or custom config files
- **Batch Operations**: Execute multiple operations efficiently in a single call

**Linting Parameters:**
- `target`: File path, directory path, or `.` for entire repository (optional, defaults to `.`)
- `format`: Output format - `json` or `text` (optional, defaults to `json`)
- `config`: Path to golangci-lint config file (optional, defaults to `.golangci.yml` if exists)

**Use Cases:**
- **Code Quality**: Enforce Go code quality standards
- **Pre-commit Checks**: Run linting before committing changes
- **CI/CD Integration**: Automated code quality checks in pipelines
- **Development Assistance**: Real-time feedback on code issues

**Binary Management:**
- Automatically downloads golangci-lint binary if not present
- Caches binary in `~/.cache/mcp-lang-go/` directory
- Supports custom golangci-lint version via environment variable
- Cross-platform support (Linux, macOS, Windows)

## Prerequisites

- Go 1.24.1 or higher
- Git (for mcp-git server)
- Bash (for mcp-bash server)
- PowerShell (for mcp-powershell server) - Windows PowerShell 5.1+ or PowerShell Core 6.0+
- PostgreSQL (for mcp-guidelines, mcp-postgres, and mcp-documents servers) - Database access
- golangci-lint (auto-managed by mcp-lang-go server)

## Installation

### Quick Start

Build and install all MCP servers to your PATH:

```bash
make mcp-servers
```

This command will:
1. Build all 12 MCP server executables (`mcp-filesystem`, `mcp-codebase`, `mcp-git`, `mcp-code-edit`, `mcp-bash`, `mcp-powershell`, `mcp-systeminfo`, `mcp-guidelines`, `mcp-postgres`, `mcp-savepoints`, `mcp-documents`, `mcp-lang-go`)
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
- Compiles all 12 MCP server executables
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
go build -o mcp-bash ./cmd/mcp-bash
go build -o mcp-powershell ./cmd/mcp-powershell
go build -o mcp-systeminfo ./cmd/mcp-systeminfo
go build -o mcp-guidelines ./cmd/mcp-guidelines
go build -o mcp-postgres ./cmd/mcp-postgres
go build -o mcp-savepoints ./cmd/mcp-savepoints
go build -o mcp-documents ./cmd/mcp-documents
go build -o mcp-lang-go ./cmd/mcp-lang-go
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
│   ├── mcp-code-edit/
│   │   └── main.go
│   ├── mcp-bash/
│   │   ├── main.go
│   │   ├── bash_operations.go
│   │   ├── security.go
│   │   ├── audit.go
│   │   ├── mcp.go
│   │   └── types.go
│   ├── mcp-powershell/
│   │   ├── main.go
│   │   ├── powershell_operations.go
│   │   ├── security.go
│   │   ├── audit.go
│   │   ├── mcp.go
│   │   └── types.go
│   ├── mcp-systeminfo/
│   │   ├── main.go
│   │   ├── systeminfo_operations.go
│   │   ├── shell_info.go
│   │   ├── devtools_info.go
│   │   ├── network_info.go
│   │   ├── repository_info.go
│   │   ├── security.go
│   │   ├── audit.go
│   │   ├── mcp.go
│   │   └── types.go
│   ├── mcp-guidelines/
│   │   ├── main.go
│   │   ├── mcp.go
│   │   ├── database.go
│   │   └── types.go
│   ├── mcp-postgres/
│   │   ├── main.go
│   │   ├── database.go
│   │   ├── types.go
│   │   └── README.md
│   ├── mcp-savepoints/
│   │   ├── main.go
│   │   ├── checkpoint_manager.go
│   │   ├── checkpoint_operations.go
│   │   ├── mcp.go
│   │   └── types.go
│   ├── mcp-documents/
│   │   ├── main.go
│   │   ├── document_repository.go
│   │   ├── handlers.go
│   │   ├── mcp.go
│   │   └── types.go
│   └── mcp-lang-go/
│       ├── main.go
│       ├── lint.go
│       ├── mcp.go
│       └── types.go
├── Makefile
├── Makefile.windows
├── Makefile.unix
├── install-windows.ps1
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
