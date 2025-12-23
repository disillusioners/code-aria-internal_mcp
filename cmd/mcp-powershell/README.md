# MCP PowerShell Server

The mcp-powershell server provides secure PowerShell command execution capabilities for Windows environments. It implements the Model Context Protocol (MCP) to allow AI agents and other tools to execute PowerShell commands and scripts safely.

## Features

### Core Operations
- **execute_command** - Execute single PowerShell commands with security restrictions
- **execute_script** - Execute multi-line PowerShell scripts with enhanced security controls
- **check_command_exists** - Check if PowerShell cmdlets, functions, or external commands are available

### Windows-Specific Capabilities
- **PowerShell Detection**: Automatically detects and uses PowerShell Core (pwsh) or Windows PowerShell
- **Native Cmdlet Support**: Full support for PowerShell cmdlets like `Get-ChildItem`, `Set-Content`, etc.
- **Windows Commands**: Built-in support for traditional Windows commands (dir, type, copy, move, etc.)
- **Executable Handling**: Properly handles Windows executable extensions (.exe, .ps1, .cmd, .bat)
- **Path Management**: Native Windows file path handling

### Security Features
- **Command Validation**: Comprehensive allow/block lists for PowerShell cmdlets and external commands
- **Script Security**: Multi-layer validation for PowerShell scripts including dangerous construct detection
- **Input Sanitization**: UTF-8 validation and character sanitization
- **Working Directory Restrictions**: Prevents execution outside approved directories
- **Environment Variable Filtering**: Blocks dangerous environment variables
- **Audit Logging**: Comprehensive logging of all operations with security validation results
- **Timeout Management**: Configurable timeouts with maximum limits
- **Pattern Blocking**: Regex-based detection of dangerous PowerShell patterns

## Usage Examples

### Execute Simple Commands

```json
{
  "operations": [
    {
      "type": "execute_command",
      "command": "Get-ChildItem -Path . -Filter *.go"
    }
  ]
}
```

### Execute Scripts

```json
{
  "operations": [
    {
      "type": "execute_script",
      "script": "Get-Process | Where-Object {$_.CPU -gt 100} | Select-Object Name, CPU",
      "timeout": 30
    }
  ]
}
```

### Check Command Availability

```json
{
  "operations": [
    {
      "type": "check_command_exists",
      "command": "Get-Command"
    }
  ]
}
```

### Environment Variables

```json
{
  "operations": [
    {
      "type": "execute_command",
      "command": "Write-Output $env:PATH",
      "environment_vars": {
        "CUSTOM_VAR": "value"
      }
    }
  ]
}
```

## Allowed Commands

The server includes an extensive allow list of safe PowerShell cmdlets and Windows commands:

### File Operations
- `Get-ChildItem`, `Get-Content`, `Set-Content`, `Add-Content`
- `Get-Item`, `Test-Path`, `New-Item`, `Remove-Item`
- `Copy-Item`, `Move-Item`, `Rename-Item`, `Get-Location`

### Development Tools
- `git`, `npm`, `yarn`, `dotnet`, `msbuild`
- `go`, `python`, `node`, `java`, `javac`
- `cargo`, `rustc`, `gcc`, `clang`

### System Information
- `Get-Process`, `Get-Service`, `Get-ComputerInfo`
- `Get-WmiObject`, `Get-CimInstance`, `Get-Command`

### Text Processing
- `Select-String`, `Select-Object`, `Where-Object`
- `Sort-Object`, `Group-Object`, `Replace-String`

### Network Tools
- `Test-Connection`, `Test-NetConnection`, `Resolve-DnsName`
- `Invoke-WebRequest`, `Invoke-RestMethod`

## Blocked Patterns

The server blocks dangerous PowerShell constructs including:
- `Invoke-Expression` (code execution)
- `Start-Process -Verb RunAs` (elevation)
- `Set-ExecutionPolicy -Force` (policy changes)
- `Add-Type` (assembly loading)
- `Set-Acl` (permission modifications)
- Registry manipulation commands
- Service control commands
- Disk formatting and partitioning

## Configuration

### Environment Variables

- `REPO_PATH` - Base directory for file operations
- `MCP_POWERSHELL_AUDIT_FILE` - Path for audit log file (default: mcp-powershell-audit.log)
- `MCP_POWERSHELL_AUDIT_DISABLED` - Disable audit logging ("true" to disable)

### Security Policy

The security policy can be customized by modifying the `defaultSecurityPolicy` variable in `powershell_operations.go`:

```go
var defaultSecurityPolicy = SecurityPolicy{
    AllowedCommands: map[string]bool{
        // Custom allowed commands
    },
    BlockedPatterns: []string{
        // Custom blocked patterns
    },
    MaxCommandLen:     1000,
    MaxScriptLen:      10000,
    DefaultTimeout:    30,
    MaxTimeout:        300,
    AllowShellAccess:  false,
}
```

## Error Handling

The server provides detailed error information including:

- **Security Violations** (-32001): When commands violate security policies
- **Timeout Errors** (-32002): When operations exceed timeout limits
- **Execution Errors** (-32003): When PowerShell execution fails

Each error includes the specific reason and context for debugging.

## Audit Logging

All operations are logged with:
- Timestamp and operation type
- Command/script content
- User and working directory
- Environment variables
- Security validation results
- Execution results and duration
- Success/failure status

Audit logs are written in JSON format for easy parsing and analysis.

## Building

```bash
# Build the PowerShell server
go build -o mcp-powershell.exe ./cmd/mcp-powershell

# Or use the Makefile
make build-mcp-servers
```

## Testing

Test the server by sending MCP protocol messages:

```bash
# List available tools
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | ./mcp-powershell.exe

# Execute a simple command
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"execute_command","command":"Get-Date"}]}}}' | ./mcp-powershell.exe
```

## Platform Support

- **Windows**: Native support with Windows PowerShell 5.1+ or PowerShell Core 6.0+
- **Linux/macOS**: Supports PowerShell Core (pwsh) for cross-platform PowerShell execution

## Security Considerations

1. **Least Privilege**: Run the server with minimal required permissions
2. **Network Isolation**: Avoid executing commands that access network resources
3. **File System**: Restrict working directories to safe locations
4. **Audit Review**: Regularly review audit logs for suspicious activity
5. **Policy Updates**: Update allow/block lists based on your security requirements

## Integration

The mcp-powershell server integrates with:
- **Genkit MCP Plugin**: Use as an MCP tool in Genkit workflows
- **LangGraph**: Power PowerShell-based automation in LangGraph agents
- **Custom MCP Clients**: Any MCP-compatible client

## Examples

### File Operations

```json
{
  "operations": [
    {
      "type": "execute_command",
      "command": "Get-ChildItem -Path . -Recurse -Filter *.ps1"
    }
  ]
}
```

### Process Management

```json
{
  "operations": [
    {
      "type": "execute_command",
      "command": "Get-Process | Where-Object {$_.Name -like '*node*'}"
    }
  ]
}
```

### Development Workflow

```json
{
  "operations": [
    {
      "type": "execute_script",
      "script": "Get-Location\nWrite-Output 'Current directory:'\nGet-ChildItem\n\nif (Test-Path 'package.json') {\n    Write-Output 'Node.js project detected'\n    npm test\n} else {\n    Write-Output 'No package.json found'\n}",
      "timeout": 60
    }
  ]
}
```

## Troubleshooting

### Common Issues

1. **PowerShell Not Found**: Ensure PowerShell is installed and in PATH
2. **Execution Policy**: Server uses `-ExecutionPolicy Bypass` for scripts
3. **Access Denied**: Check file permissions and working directory access
4. **Timeout Violation**: Increase timeout values for long-running operations

### Debug Mode

Enable verbose logging by setting environment variables:
```bash
export MCP_POWERSHELL_AUDIT_DISABLED=false
export MCP_POWERSHELL_AUDIT_FILE=mcp-powershell-debug.log
```