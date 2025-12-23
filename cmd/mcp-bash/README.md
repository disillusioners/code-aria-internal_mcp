# MCP Bash Command Execution Server

A secure MCP server for executing bash commands and scripts with comprehensive security measures and audit logging.

## Features

- **Secure Command Execution**: Validates commands against allow/block lists
- **Script Execution**: Supports multi-line bash scripts with enhanced security
- **Command Discovery**: Check if commands are available in the system
- **Comprehensive Security**: Multiple layers of security validation
- **Audit Logging**: Complete audit trail of all operations
- **Timeout Management**: Configurable timeouts to prevent hanging
- **Resource Limits**: Built-in resource constraints
- **Environment Control**: Secure environment variable handling

## Operations

### 1. `execute_command`
Executes a single bash command with security restrictions.

**Parameters:**
- `command` (string, required): The bash command to execute
- `timeout` (integer, optional): Command timeout in seconds (default: 30, max: 300)
- `working_directory` (string, optional): Directory to execute command in (default: REPO_PATH)
- `allow_shell_access` (boolean, optional): Allow shell features like pipes, redirects (default: false)
- `environment_vars` (object, optional): Additional environment variables for the command

**Example:**
```json
{
  "type": "execute_command",
  "command": "ls -la",
  "timeout": 30,
  "working_directory": "/tmp",
  "allow_shell_access": false,
  "environment_vars": {
    "CUSTOM_VAR": "value"
  }
}
```

### 2. `execute_script`
Executes a multi-line bash script with enhanced security controls.

**Parameters:**
- `script` (string, required): Multi-line bash script to execute
- `timeout` (integer, optional): Script timeout in seconds (default: 60, max: 600)
- `working_directory` (string, optional): Directory to execute script in (default: REPO_PATH)
- `allow_shell_access` (boolean, optional): Allow shell features (default: true for scripts)
- `environment_vars` (object, optional): Additional environment variables
- `script_name` (string, optional): Name for logging and identification

**Example:**
```json
{
  "type": "execute_script",
  "script": "#!/bin/bash\necho 'Hello World'\nls -la",
  "timeout": 60,
  "working_directory": "/tmp",
  "allow_shell_access": true,
  "script_name": "hello_script"
}
```

### 3. `check_command_exists`
Checks if a command is available in the system PATH.

**Parameters:**
- `command` (string, required): Command name to check
- `search_paths` (array, optional): Additional paths to search (default: system PATH)

**Example:**
```json
{
  "type": "check_command_exists",
  "command": "git",
  "search_paths": ["/usr/local/bin", "/usr/bin"]
}
```

## Security Features

### Command Validation
- **Allowed Commands**: Predefined list of permitted commands
- **Blocked Patterns**: Regex patterns for dangerous commands
- **Length Limits**: Maximum command/script length enforcement
- **UTF-8 Validation**: Ensures valid character encoding
- **Input Sanitization**: Removes dangerous characters

### Execution Security
- **Working Directory Restrictions**: Limits execution to approved paths
- **Environment Variable Filtering**: Blocks dangerous env vars
- **Shell Access Control**: Optional shell feature restrictions
- **Path Traversal Prevention**: Blocks directory escape attempts

### Audit Logging
- **Complete Operation Logging**: All commands/scripts are logged
- **Security Violation Tracking**: Failed validations are recorded
- **Performance Metrics**: Execution time and resource usage
- **User Attribution**: Tracks which user executed commands

## Response Format

All operations return responses following the MCP batch operations format:

```json
{
  "operation": "operation_type",
  "params": { ... },
  "status": "Success|Error|Timeout",
  "result": { ... },  // Only for successful operations
  "message": "..."     // Error message or additional info
}
```

### Success Response Example
```json
{
  "operation": "execute_command",
  "params": {
    "command": "ls -la",
    "timeout": 30
  },
  "status": "Success",
  "result": {
    "exit_code": 0,
    "stdout": "total 0\ndrwxr-xr-x  2 user user  4096 Jan  1 12:00 .",
    "stderr": "",
    "duration_ms": 45,
    "command": "ls -la",
    "working_directory": "/tmp"
  }
}
```

### Error Response Example
```json
{
  "operation": "execute_command",
  "params": {
    "command": "rm -rf /"
  },
  "status": "Error",
  "message": "security violation: Blocked pattern detected: rm\\s+-rf\\s+/",
  "error_code": -32001,
  "error_type": "Security"
}
```

## Configuration

### Environment Variables
- `REPO_PATH`: Base directory for command execution
- `MCP_BASH_AUDIT`: Enable/disable audit logging (default: true)
- `MCP_BASH_AUDIT_FILE`: Custom audit log file path

### Security Policy
The server uses a configurable security policy with defaults:

**Allowed Commands Include:**
- File operations: `ls`, `cat`, `head`, `tail`, `wc`, `grep`, `find`, `file`, `stat`
- Development tools: `git`, `npm`, `yarn`, `make`, `go`, `python`, `node`
- System info: `ps`, `top`, `free`, `uname`, `whoami`, `id`, `date`
- Text processing: `sed`, `awk`, `sort`, `uniq`, `cut`, `tr`, `diff`
- Archive tools: `tar`, `gzip`, `zip`, `unzip`
- Network tools: `curl`, `wget`, `ping`, `nslookup`, `dig`
- Process management: `kill`, `killall`, `pkill`

**Blocked Patterns Include:**
- Dangerous deletion: `rm -rf /`, `dd if=/dev/zero`
- Fork bombs: `:(){...};`
- Privilege escalation: `sudo`, `su`
- System damage: `mkfs`, `fdisk`, `iptables`
- Service disruption: `service stop`, `systemctl stop`
- System control: `shutdown`, `reboot`, `halt`, `poweroff`

## Building

```bash
# Build the MCP bash server
go build -o mcp-bash ./cmd/mcp-bash

# Or use the Makefile
make build-mcp-servers
```

## Usage

The server follows the standard MCP protocol and communicates via stdio using JSON-RPC 2.0.

### Example Usage
```bash
# Start the server
./mcp-bash

# Send operations via stdin
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "apply_operations",
    "arguments": {
      "operations": [
        {
          "type": "execute_command",
          "command": "echo Hello World"
        }
      ]
    }
  }
}' | ./mcp-bash
```

## Integration

The server integrates with the existing MCP ecosystem:

1. **MCP Manager**: Automatically discovered and registered
2. **UnifiedBatchOperationsTool**: Routes operations to this server
3. **Operation Routing**: `execute_command`, `execute_script`, `check_command_exists`

## Error Codes

| Error Type | Code | Description |
|-------------|--------|-------------|
| Validation | -32602 | Invalid parameters or command format |
| Security | -32001 | Security policy violation |
| Timeout | -32002 | Command execution timeout |
| Execution | -32003 | Runtime execution error |
| Permission | -32004 | File system permission error |
| Resource | -32005 | Resource limit exceeded |

## Security Considerations

- **Principle of Least Privilege**: Commands run with minimal required permissions
- **Defense in Depth**: Multiple security validation layers
- **Complete Auditing**: All operations are logged for review
- **Fail Secure**: Default deny stance for unknown commands
- **Resource Limits**: Built-in protections against resource exhaustion

## Deployment

### Docker
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-bash ./cmd/mcp-bash

FROM alpine:latest
RUN apk add --no-cache bash
COPY --from=builder /app/mcp-bash /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/mcp-bash"]
```

### Kubernetes
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mcp-bash
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    readOnlyRootFilesystem: true
    capabilities:
      drop: ["ALL"]
    allowPrivilegeEscalation: false
  containers:
  - name: mcp-bash
    image: mcp-bash:latest
    env:
    - name: REPO_PATH
      value: "/workspace"