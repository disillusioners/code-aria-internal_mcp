# MCP SystemInfo Server

The mcp-systeminfo server provides comprehensive system information gathering capabilities for AI agents and LLMs. It helps understand the operating environment, available tools, and system context to make informed decisions about code generation and command execution.

## Overview

SystemInfo gathers detailed information about:
- **Operating System**: Name, version, architecture, distribution, kernel
- **Hardware**: CPU, memory, storage, network interfaces, displays
- **Environment**: Working directory, environment variables (filtered), PATH
- **Shell**: Type, version, features, aliases, functions
- **Development Tools**: Compilers, interpreters, package managers, build tools
- **Network**: IP addresses, DNS, proxy settings, connectivity
- **Repositories**: Git, SVN, Mercurial repository detection and status
- **Recommendations**: Context-aware system-specific recommendations

## Key Features

### Cross-Platform Support
- **Windows**: Full support with PowerShell commands and Windows-specific APIs
- **Linux**: Comprehensive support with /proc filesystem and standard Unix tools
- **macOS**: Native support with system commands and platform-specific features

### Security-First Design
- **Read-Only Operations**: All operations are read-only and non-destructive
- **Input Validation**: Comprehensive validation of all commands and parameters
- **Sensitive Data Filtering**: Environment variables are filtered to exclude sensitive information
- **Audit Logging**: All operations are logged with timestamps and context
- **Timeout Management**: All operations have configurable timeouts with maximum limits

### Smart Recommendations
- **Environment-Aware**: Recommendations based on detected OS, tools, and configuration
- **Tool-Specific**: Suggests installation of missing development tools
- **Resource-Conscious**: Warns about high memory usage, low disk space, etc.

## Available Operations

### Core Information Gathering

#### get_system_info()
Returns a complete system overview including all available information.

```json
{
  "operations": [
    {
      "type": "get_system_info"
    }
  ]
}
```

**Response includes:**
- OS information and version details
- Hardware specifications
- Environment variables (filtered)
- Shell capabilities and features
- Development tools and versions
- Network configuration
- Repository detection results
- System-specific recommendations

#### get_os_info()
Returns detailed operating system information.

```json
{
  "operations": [
    {
      "type": "get_os_info"
    }
  ]
}
```

**Response includes:**
- OS name and version
- Architecture and platform
- Distribution information (Linux)
- Kernel version
- Build information
- Version components (major, minor, patch)

#### get_hardware_info()
Returns hardware and system resource information.

```json
{
  "operations": [
    {
      "type": "get_hardware_info"
    }
  ]
}
```

**Response includes:**
- CPU model, cores, threads, frequency
- Memory total, used, available, usage percentage
- Storage devices with usage statistics
- Network interfaces and configurations
- Display information (when available)

#### get_environment_info()
Returns environment configuration.

```json
{
  "operations": [
    {
      "type": "get_environment_info"
    }
  ]
}
```

**Response includes:**
- Current working directory
- Home directory
- Username and hostname
- PATH environment variable
- Environment variables (sensitive data filtered)
- REPO_PATH if set

#### get_shell_info()
Returns shell information and capabilities.

```json
{
  "operations": [
    {
      "type": "get_shell_info"
    }
  ]
}
```

**Response includes:**
- Shell name and version
- Shell type (bash, zsh, powershell, etc.)
- Available features
- Shell aliases (when available)
- Common functions

#### get_development_tools()
Returns development tools information.

```json
{
  "operations": [
    {
      "type": "get_development_tools"
    }
  ]
}
```

**Response includes:**
- Installed compilers and interpreters
- Version information for each tool
- Package managers and their versions
- Build tools and utilities
- Tool-specific features

#### get_network_info()
Returns network configuration and connectivity status.

```json
{
  "operations": [
    {
      "type": "get_network_info"
    }
  ]
}
```

**Response includes:**
- Hostname and domain
- IP addresses (local and public)
- MAC address
- Gateway and DNS servers
- Proxy configuration
- Connectivity status

#### detect_repositories()
Detects version control repositories.

```json
{
  "operations": [
    {
      "type": "detect_repositories"
    }
  ]
}
```

**Response includes:**
- Repository paths and types (Git, SVN, Mercurial)
- Current branch and commit
- Remote URLs
- Repository status (clean/dirty)
- Last activity timestamp

### Utility Operations

#### check_command()
Check if a specific command is available and get its version.

```json
{
  "operations": [
    {
      "type": "check_command",
      "command": "git",
      "search_paths": ["/usr/bin", "/usr/local/bin"]
    }
  ]
}
```

#### get_recommendations()
Get system-specific recommendations.

```json
{
  "operations": [
    {
      "type": "get_recommendations"
    }
  ]
}
```

## Example Usage

### Complete System Analysis

```json
{
  "operations": [
    {
      "type": "get_system_info"
    }
  ]
}
```

### Tool Availability Check

```json
{
  "operations": [
    {
      "type": "get_development_tools"
    }
  ]
}
```

### Environment Context

```json
{
  "operations": [
    {
      "type": "get_environment_info"
    },
    {
      "type": "get_shell_info"
    }
  ]
}
```

### Repository Detection

```json
{
  "operations": [
    {
      "type": "detect_repositories"
    }
  ]
}
```

## Response Format

All responses follow a consistent format:

```json
{
  "results": [
    {
      "operation": "operation_type",
      "params": {...},
      "status": "Success|Error",
      "result": {
        // Operation-specific data
      },
      "message": "Error message (if status is Error)"
    }
  ]
}
```

## Use Cases

### Environment Understanding
LLLMs can use SystemInfo to:
- Understand the target OS before generating commands
- Choose appropriate shell syntax and commands
- Detect available development tools and libraries
- Understand file system structure and paths

### Cross-Platform Compatibility
- Generate platform-appropriate commands
- Detect package managers for installation instructions
- Understand environment-specific requirements
- Choose appropriate build tools and commands

### Context-Aware Assistance
- Detect existing repositories for context-aware help
- Understand current working directory and project structure
- Identify potential issues (low disk space, missing tools)
- Provide relevant recommendations based on environment

### Tool Selection
- Automatically detect available compilers and interpreters
- Choose appropriate package managers
- Detect build tools and their versions
- Recommend missing tools for development workflows

## Security Considerations

### Data Filtering
- Environment variables are filtered to exclude sensitive data
- Passwords, tokens, keys, and certificates are automatically excluded
- Only non-sensitive environment information is exposed

### Read-Only Operations
- All operations are designed to be read-only
- No system modifications are performed
- Safe for execution in any environment

### Audit Trail
- All operations are logged with timestamps
- Comprehensive audit information for security review
- Operation context and parameters are recorded

## Performance Considerations

### Timeout Management
- All operations have configurable timeouts
- Maximum timeout limits prevent resource exhaustion
- Failed operations don't block other operations

### Resource Usage
- Minimal system overhead
- Efficient data collection methods
- Caching of expensive operations when possible

## Configuration

### Environment Variables

- `MCP_SYSTEMINFO_AUDIT_FILE` - Path for audit log file (default: mcp-systeminfo-audit.log)
- `MCP_SYSTEMINFO_AUDIT_DISABLED` - Disable audit logging ("true" to disable)
- `REPO_PATH` - Repository path for context (optional)

### Security Policy

The security policy can be customized by modifying the `defaultSecurityPolicy` variable in `security.go`:

```go
var defaultSecurityPolicy = SecurityPolicy{
    AllowedCommands: map[string]bool{
        // Custom allowed commands
    },
    BlockedPatterns: []string{
        // Custom blocked patterns
    },
    MaxCommandLen:    500,
    DefaultTimeout:   10,
    MaxTimeout:       30,
    AllowShellAccess: false,
}
```

## Building

```bash
# Build the SystemInfo server
go build -o mcp-systeminfo ./cmd/mcp-systeminfo

# Or use the Makefile
make build-mcp-servers
```

## Testing

Test the server by sending MCP protocol messages:

```bash
# List available tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./mcp-systeminfo

# Get system information
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"get_system_info"}]}}}' | ./mcp-systeminfo
```

## Integration Examples

### Pre-Task Analysis
Before performing code generation or task execution:

```json
{
  "operations": [
    {
      "type": "get_os_info"
    },
    {
      "type": "get_development_tools"
    },
    {
      "type": "detect_repositories"
    }
  ]
}
```

### Environment Context
For understanding the current development environment:

```json
{
  "operations": [
    {
      "type": "get_environment_info"
    },
    {
      "type": "get_shell_info"
    },
    {
      "type": "get_network_info"
    }
  ]
}
```

### Tool Verification
Before using specific tools in commands:

```json
{
  "operations": [
    {
      "type": "check_command",
      "command": "docker"
    },
    {
      "type": "check_command",
      "command": "node"
    },
    {
      "type": "check_command",
      "command": "go"
    }
  ]
}
```

## Error Handling

The server provides detailed error information including:

- **Security Violations** (-32001): When operations violate security policies
- **Timeout Errors** (-32002): When operations exceed timeout limits
- **Execution Errors** (-32003): When system commands fail

Each error includes the specific reason and context for debugging.

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure the server has read access to system information files
2. **Command Not Found**: Some commands may not be available on all systems
3. **Timeout Issues**: Increase timeout values for slower systems
4. **Incomplete Information**: Some information may not be available on certain platforms

### Debug Mode

Enable verbose logging by setting environment variables:
```bash
export MCP_SYSTEMINFO_AUDIT_DISABLED=false
export MCP_SYSTEMINFO_AUDIT_FILE=mcp-systeminfo-debug.log
```

## Extensibility

The server is designed to be extensible:
- Add new information gathering functions
- Customize platform-specific implementations
- Extend security policies
- Add new recommendation logic
- Integrate additional system monitoring capabilities

## Platform-Specific Notes

### Windows
- Uses PowerShell for most system information
- Supports both Windows PowerShell and PowerShell Core
- Windows-specific commands and APIs
- Registry access for detailed system information

### Linux
- Uses /proc filesystem for detailed information
- Standard Unix tools for system queries
- Distribution-specific package managers and tools
- Supports various Linux distributions

### macOS
- Uses system commands and APIs
- Supports macOS-specific features
- Homebrew package manager detection
- macOS version and build information

## Best Practices

1. **Call Before Acting**: Use SystemInfo before generating commands to ensure compatibility
2. **Batch Operations**: Use multiple operations in a single request for efficiency
3. **Handle Errors Gracefully**: Check operation status and handle errors appropriately
4. **Cache Results**: Cache system information when appropriate to reduce overhead
5. **Respect Privacy**: Be aware that some system information may be sensitive

The mcp-systeminfo server provides a comprehensive foundation for AI agents to understand and work effectively with diverse computing environments.