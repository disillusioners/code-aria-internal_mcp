# mcp-postgres

MCP server for read-only access to PostgreSQL databases. This server provides tools to query databases, list tables and schemas, and describe table structures.

## Overview

The `mcp-postgres` server connects to PostgreSQL databases and exposes read-only database operations as MCP tools. This allows AI agents and workflows to inspect database schemas and query data safely.

## Features

- Read-only database access (SELECT queries only)
- List all schemas in a database
- List tables in a schema with metadata
- Describe table schemas (columns, types, constraints, indexes)
- Execute parameterized SELECT queries
- Connection string configuration via environment variable or per-operation override
- Automatic query validation to prevent data modification
- Result limiting for safety

## Prerequisites

- Go 1.24.1 or higher
- PostgreSQL database
- Database connection string

## Configuration

The server requires a database connection string via the `POSTGRES_DB_DSN` environment variable, or you can provide it per-operation.

### Environment Variables

You can set the environment variable directly:

```bash
export POSTGRES_DB_DSN="postgres://user:password@host:port/dbname?sslmode=disable"
```

Or use a `.env` file (recommended for development):

1. Copy `env.example` to `.env`:
   ```bash
   cp env.example .env
   ```

2. Edit `.env` and update the database connection string:
   ```bash
   POSTGRES_DB_DSN=postgres://user:password@host:port/dbname?sslmode=disable
   ```

### Connection String Format

PostgreSQL connection strings follow this format:
```
postgres://[user[:password]@][netloc][:port][/dbname][?param1=value1&...]
```

Common parameters:
- `sslmode=disable` - Disable SSL (for local development)
- `sslmode=require` - Require SSL
- `connect_timeout=10` - Connection timeout in seconds

## MCP Tools

The server exposes a single `apply_operations` tool that supports batch execution of multiple operations.

### Operations

#### list_schemas

List all schemas in the database (excluding system schemas).

**Parameters:**
- `connection_string` (string, optional): Override default connection string

**Returns:** Array of schema names

**Example:**
```json
{
  "type": "list_schemas"
}
```

#### list_tables

List tables in a schema.

**Parameters:**
- `schema` (string, optional): Schema name (defaults to 'public')
- `connection_string` (string, optional): Override default connection string

**Returns:** Array of table objects with schema, table_name, and table_type

**Example:**
```json
{
  "type": "list_tables",
  "schema": "public"
}
```

#### describe_table

Get detailed table schema information including columns, types, constraints, and indexes.

**Parameters:**
- `table_name` (string, required): Table name
- `schema` (string, optional): Schema name (defaults to 'public')
- `connection_string` (string, optional): Override default connection string

**Returns:** Table schema object with columns, constraints, and indexes

**Example:**
```json
{
  "type": "describe_table",
  "table_name": "users",
  "schema": "public"
}
```

#### query

Execute a SELECT query.

**Parameters:**
- `query` (string, required): SELECT query to execute
- `params` (array, optional): Query parameters for parameterized queries
- `limit` (integer, optional): Maximum rows to return (default: 1000, max: 10000)
- `connection_string` (string, optional): Override default connection string

**Returns:** Array of result objects (one per row)

**Example:**
```json
{
  "type": "query",
  "query": "SELECT id, name, email FROM users WHERE active = $1",
  "params": [true],
  "limit": 100
}
```

## Usage

### Building

```bash
go build -o mcp-postgres ./cmd/mcp-postgres
```

### Running

Set the database connection string and run:

```bash
export POSTGRES_DB_DSN="postgres://user:password@localhost:5432/mydb?sslmode=disable"
./mcp-postgres
```

The server communicates via stdio using the MCP protocol.

### Batch Operations

You can execute multiple operations in a single call:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "apply_operations",
    "arguments": {
      "operations": [
        {
          "type": "list_schemas"
        },
        {
          "type": "list_tables",
          "schema": "public"
        },
        {
          "type": "describe_table",
          "table_name": "users",
          "schema": "public"
        },
        {
          "type": "query",
          "query": "SELECT COUNT(*) as total FROM users",
          "limit": 1
        }
      ]
    }
  }
}
```

## Security

- **Read-only enforcement**: Only SELECT queries are allowed. All other SQL statements (INSERT, UPDATE, DELETE, DROP, etc.) are rejected.
- **Query validation**: Queries are validated before execution to ensure they are SELECT-only.
- **Parameterized queries**: Support for parameterized queries prevents SQL injection.
- **Result limiting**: Default limit of 1000 rows, configurable up to 10000 rows.
- **Connection string security**: Connection strings in responses are automatically excluded for security.

## Error Handling

The server returns proper MCP error responses for:
- Invalid parameters
- Database connection errors
- Query execution errors
- Missing required parameters
- Non-SELECT queries (security violation)

## Testing

To test the server manually:

```bash
# Set database connection
export POSTGRES_DB_DSN="postgres://user:password@localhost:5432/mydb?sslmode=disable"

# Test initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | ./mcp-postgres

# Test tools/list
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./mcp-postgres

# Test list_schemas
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"list_schemas"}]}}}' | ./mcp-postgres
```

## Troubleshooting

### Database Connection Errors

If you see "failed to open database" errors:
- Verify `POSTGRES_DB_DSN` is set correctly
- Check database credentials
- Ensure database is accessible from the server location
- Verify PostgreSQL is running

### Query Rejected Errors

If queries are rejected:
- Ensure queries are SELECT-only
- Check for forbidden keywords (INSERT, UPDATE, DELETE, etc.)
- Verify query syntax is correct

### No Results Returned

If queries return empty results:
- Verify table/schema names are correct
- Check that data exists in the database
- Ensure you have SELECT permissions on the tables

## License

Part of the Code-Aria Internal MCP Servers project.

