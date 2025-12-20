# mcp-guidelines

MCP server for accessing guidelines from the PostgreSQL database. This server provides read-only access to guideline documents that can be used to customize AI agent behavior during workflow execution.

## Overview

The `mcp-guidelines` server connects to the same PostgreSQL database as the API/Worker services and exposes guidelines as MCP tools. This allows LangGraph workflows to fetch and use guidelines during execution to customize AI agent behavior.

## Features

- Read-only access to guidelines from PostgreSQL database
- Filter guidelines by tenant, category, tags, or active status
- Search guidelines by name, description, or content
- Retrieve specific guidelines by ID
- Secure parameterized queries to prevent SQL injection

## Prerequisites

- Go 1.24.1 or higher
- PostgreSQL database with `guidelines` table (see migration 017)
- Database connection string

## Configuration

The server requires a database connection string via the `GUIDELINES_DB_DSN` environment variable.

### Environment Variables

You can set the environment variable directly:

```bash
export GUIDELINES_DB_DSN="postgres://user:password@host:port/dbname?sslmode=disable"
```

Or use a `.env` file (recommended for development):

1. Copy `env.example` to `.env`:
   ```bash
   cp env.example .env
   ```

2. Edit `.env` and update the database connection string:
   ```bash
   GUIDELINES_DB_DSN=postgres://user:password@host:port/dbname?sslmode=disable
   ```

The server will automatically load the `.env` file if it exists in the current directory.

### Database Schema

The server expects a `guidelines` table with the following structure:

```sql
CREATE TABLE guidelines (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    category VARCHAR(100),
    tags JSONB DEFAULT '[]',
    tenant_id VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## MCP Tools

### get_guidelines

Get guidelines filtered by tenant_id, category, tags, or active status.

**Parameters:**
- `tenant_id` (string, optional): Filter by tenant ID
- `category` (string, optional): Filter by category
- `tags` (array of strings, optional): Filter by tags (any match)
- `is_active` (boolean, optional): Filter by active status (default: true)
- `limit` (integer, optional): Limit results (default: 50, max: 100)

**Returns:** Array of guideline objects with id, name, description, content, category, tags, metadata

**Example:**
```json
{
  "name": "get_guidelines",
  "arguments": {
    "tenant_id": "tenant-123",
    "category": "coding-standards",
    "is_active": true,
    "limit": 10
  }
}
```

### get_guideline_content

Get full content of specific guidelines by IDs.

**Parameters:**
- `guideline_ids` (array of strings, required): Array of guideline IDs

**Returns:** Array of guideline objects with full content

**Example:**
```json
{
  "name": "get_guideline_content",
  "arguments": {
    "guideline_ids": ["guideline-1", "guideline-2"]
  }
}
```

### search_guidelines

Search guidelines by name, description, or content text.

**Parameters:**
- `search_term` (string, required): Search query
- `tenant_id` (string, optional): Filter by tenant ID
- `category` (string, optional): Filter by category
- `limit` (integer, optional): Limit results (default: 20, max: 50)

**Returns:** Array of matching guidelines

**Example:**
```json
{
  "name": "search_guidelines",
  "arguments": {
    "search_term": "React",
    "tenant_id": "tenant-123",
    "limit": 20
  }
}
```

## Usage

### Building

```bash
go build -o mcp-guidelines ./cmd/mcp-guidelines
```

### Running

Set the database connection string and run:

```bash
export GUIDELINES_DB_DSN="postgres://user:password@localhost:5432/codearia?sslmode=disable"
./mcp-guidelines
```

The server communicates via stdio using the MCP protocol.

### Integration with LangGraph

The server is designed to be used by LangGraph workflows. When a workflow executes:

1. LangGraph initializes the MCP server with the database connection
2. Graph nodes can call guideline tools to fetch relevant guidelines
3. Guideline content is injected into AI prompts to customize behavior

**Example workflow integration:**

```python
# In LangGraph workflow
guidelines = mcp_client.call_tool("get_guidelines", {
    "tenant_id": workflow.tenant_id,
    "category": "coding-standards",
    "is_active": True
})

# Inject guidelines into AI prompt
prompt = f"""
{task_description}

Guidelines:
{format_guidelines(guidelines)}
"""
```

## Security

- **Read-only operations**: The server only performs SELECT queries, no write operations
- **Parameterized queries**: All queries use parameterized statements to prevent SQL injection
- **Input validation**: All parameters are validated before use
- **Tenant isolation**: Guidelines can be filtered by tenant_id to prevent cross-tenant access

## Error Handling

The server returns proper MCP error responses for:
- Invalid parameters
- Database connection errors
- Query execution errors
- Missing required parameters

## Testing

To test the server manually:

```bash
# Set database connection
export GUIDELINES_DB_DSN="postgres://user:password@localhost:5432/codearia?sslmode=disable"

# Test initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | ./mcp-guidelines

# Test tools/list
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./mcp-guidelines

# Test get_guidelines
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_guidelines","arguments":{"limit":5}}}' | ./mcp-guidelines
```

## Troubleshooting

### Database Connection Errors

If you see "failed to open database" errors:
- Verify `GUIDELINES_DB_DSN` is set correctly
- Check database credentials
- Ensure database is accessible from the server location
- Verify the `guidelines` table exists

### No Results Returned

If queries return empty results:
- Check that guidelines exist in the database
- Verify filter parameters (tenant_id, category, etc.)
- Ensure `is_active` filter is not excluding results

## License

Part of the Code-Aria Internal MCP Servers project.








