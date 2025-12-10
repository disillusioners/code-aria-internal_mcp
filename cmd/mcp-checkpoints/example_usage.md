# MCP Checkpoints Server - Usage Example

This document demonstrates how to use the MCP checkpoints server.

## Setup

1. Build the server:
```bash
cd code-aria-internal_mcp
make build-mcp-servers
```

2. Set up environment:
```bash
export REPO_PATH="/path/to/your/git/repository"
```

## Example MCP Session

Here's how the server would be used in an MCP session:

### 1. Initialize Connection
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {}
  }
}
```

### 2. List Available Tools
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

### 3. Create a Checkpoint
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "create_checkpoint",
    "arguments": {
      "name": "initial-implementation",
      "description": "Initial version of the new feature before testing"
    }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\n  \"id\": \"abc12345\",\n  \"name\": \"initial-implementation\",\n  \"description\": \"Initial version of the new feature before testing\",\n  \"timestamp\": \"2025-12-10T12:51:00Z\",\n  \"files\": [\"src/main.go\", \"README.md\"],\n  \"size\": 1024\n}"
      }
    ]
  }
}
```

### 4. List All Checkpoints
```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "list_checkpoints"
  }
}
```

### 5. Restore a Checkpoint
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "restore_checkpoint",
    "arguments": {
      "checkpoint_id": "abc12345"
    }
  }
}
```

## Integration with LangGraph

The checkpoints server can be used by LangGraph agents for:

1. **Before major changes**: Create a checkpoint of the current state
2. **After milestone completions**: Save progress incrementally
3. **Before risky operations**: Create a safety checkpoint
4. **Rollback scenarios**: Restore to a known good state

Example LangGraph workflow:
```python
def create_checkpoint_node(state):
    """Create checkpoint before making changes"""
    result = mcp_client.call_tool("create_checkpoint", {
        "name": f"checkpoint-{state['step']}",
        "description": f"Checkpoint before step {state['step']}"
    })
    return {"checkpoint_id": result["id"]}

def restore_if_failed(state):
    """Restore checkpoint if step failed"""
    if not state["success"]:
        mcp_client.call_tool("restore_checkpoint", {
            "checkpoint_id": state["checkpoint_id"]
        })
    return state
```