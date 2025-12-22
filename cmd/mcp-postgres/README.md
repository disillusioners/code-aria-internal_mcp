# mcp-postgres

MCP server for read-only access to PostgreSQL databases with connection management. This server provides tools to query databases, list tables and schemas, describe table structures, and manage database connections.

## Overview

The `mcp-postgres` server connects to multiple PostgreSQL databases through a connection management system. The master database (configured via `POSTGRES_DB_DSN`) stores connection configurations in the `mcp_connections` table, allowing you to query any configured database by name. This allows AI agents and workflows to inspect database schemas and query data safely across multiple databases.

## Features

- **Multi-database support**: Query any database configured in the connection management system
- **Connection management**: CRUD operations for managing database connections
- **Read-only database access**: SELECT queries only
- **List all schemas** in a database
- **List tables** in a schema with metadata
- **Describe table schemas** (columns, types, constraints, indexes)
- **Execute parameterized SELECT queries**
- **Automatic query validation** to prevent data modification
- **Result limiting** for safety
- **Password masking** in all responses

## Prerequisites

- Go 1.24.1 or higher
- PostgreSQL database (for master connection)
- Database connection string for master database

## Configuration

The server requires a master database connection string via the `POSTGRES_DB_DSN` environment variable. This database is used to store connection configurations in the `mcp_connections` table.

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

2. Edit `.env` and update the master database connection string:
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

### Initialization

On first startup, the server will:
1. Connect to the master database using `POSTGRES_DB_DSN`
2. Create the `mcp_connections` table if it doesn't exist
3. Automatically add the master connection as "master" in the connections table

## MCP Tools

The server exposes a single `apply_operations` tool that supports batch execution of multiple operations.

### Database Operations

#### list_schemas

List all schemas in the database (excluding system schemas).

**Parameters:**
- `connection_name` (string, required): Name of the connection to use (e.g., "master")

**Returns:** Array of schema names

**Example:**
```json
{
  "type": "list_schemas",
  "connection_name": "master"
}
```

#### list_tables

List tables in a schema.

**Parameters:**
- `connection_name` (string, required): Name of the connection to use
- `schema` (string, optional): Schema name (defaults to 'public')

**Returns:** Array of table objects with schema, table_name, and table_type

**Example:**
```json
{
  "type": "list_tables",
  "connection_name": "master",
  "schema": "public"
}
```

#### describe_table

Get detailed table schema information including columns, types, constraints, and indexes.

**Parameters:**
- `connection_name` (string, required): Name of the connection to use
- `table_name` (string, required): Table name
- `schema` (string, optional): Schema name (defaults to 'public')

**Returns:** Table schema object with columns, constraints, and indexes

**Example:**
```json
{
  "type": "describe_table",
  "connection_name": "master",
  "table_name": "users",
  "schema": "public"
}
```

#### query

Execute a SELECT query.

**Parameters:**
- `connection_name` (string, required): Name of the connection to use
- `query` (string, required): SELECT query to execute
- `params` (array, optional): Query parameters for parameterized queries
- `limit` (integer, optional): Maximum rows to return (default: 1000, max: 10000)

**Returns:** Array of result objects (one per row)

**Example:**
```json
{
  "type": "query",
  "connection_name": "master",
  "query": "SELECT id, name, email FROM users WHERE active = $1",
  "params": [true],
  "limit": 100
}
```

#### get_connection_info

Get connection information including host, port, database, user (password is masked for security).

**Parameters:**
- `connection_name` (string, required): Name of the connection to query

**Returns:** Connection info object with masked connection string and parsed components

**Example:**
```json
{
  "type": "get_connection_info",
  "connection_name": "master"
}
```

### Connection Management Operations

#### create_connection

Create a new database connection configuration.

**Parameters:**
- `name` (string, required): Unique connection name
- `host` (string, required): Database host
- `port` (integer, optional): Database port (default: 5432)
- `database` (string, required): Database name
- `user` (string, required): Database user
- `password` (string, required): Database password
- `sslmode` (string, optional): SSL mode (default: 'disable')
- `description` (string, optional): Connection description

**Returns:** Created connection object (password masked)

**Example:**
```json
{
  "type": "create_connection",
  "name": "prod_db",
  "host": "prod.example.com",
  "port": 5432,
  "database": "production",
  "user": "readonly_user",
  "password": "secret_password",
  "sslmode": "require",
  "description": "Production database connection"
}
```

#### list_connections

List all configured connections (passwords are masked).

**Parameters:** None

**Returns:** Array of connection objects (passwords masked)

**Example:**
```json
{
  "type": "list_connections"
}
```

#### get_connection

Get a connection configuration by name (password is masked).

**Parameters:**
- `name` (string, required): Connection name

**Returns:** Connection object (password masked)

**Example:**
```json
{
  "type": "get_connection",
  "name": "prod_db"
}
```

#### update_connection

Update a connection configuration.

**Parameters:**
- `name` (string, required): Connection name to update
- `host` (string, optional): Database host
- `port` (integer, optional): Database port
- `database` (string, optional): Database name
- `user` (string, optional): Database user
- `password` (string, optional): Database password (only needed if changing)
- `sslmode` (string, optional): SSL mode
- `description` (string, optional): Connection description

**Returns:** Updated connection object (password masked)

**Example:**
```json
{
  "type": "update_connection",
  "name": "prod_db",
  "description": "Updated production database",
  "sslmode": "verify-full"
}
```

#### delete_connection

Delete a connection configuration (cannot delete 'master' connection).

**Parameters:**
- `name` (string, required): Connection name to delete

**Returns:** Success confirmation

**Example:**
```json
{
  "type": "delete_connection",
  "name": "old_connection"
}
```

## Usage

### Building

```bash
go build -o mcp-postgres ./cmd/mcp-postgres
```

### Running

Set the master database connection string and run:

```bash
export POSTGRES_DB_DSN="postgres://user:password@localhost:5432/masterdb?sslmode=disable"
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
          "type": "list_schemas",
          "connection_name": "master"
        },
        {
          "type": "list_tables",
          "connection_name": "master",
          "schema": "public"
        },
        {
          "type": "describe_table",
          "connection_name": "master",
          "table_name": "users",
          "schema": "public"
        },
        {
          "type": "query",
          "connection_name": "master",
          "query": "SELECT COUNT(*) as total FROM users",
          "limit": 1
        }
      ]
    }
  }
}
```

### Connection Management Workflow

1. **Start the server** with `POSTGRES_DB_DSN` pointing to your master database
2. **Master connection** is automatically created as "master"
3. **Add additional connections** using `create_connection`:
   ```json
   {
     "type": "create_connection",
     "name": "prod_db",
     "host": "prod.example.com",
     "database": "production",
     "user": "readonly",
     "password": "secret"
   }
   ```
4. **Use connections** by name in database operations:
   ```json
   {
     "type": "query",
     "connection_name": "prod_db",
     "query": "SELECT * FROM users LIMIT 10"
   }
   ```

## Security

- **Read-only enforcement**: Only SELECT queries are allowed. All other SQL statements (INSERT, UPDATE, DELETE, DROP, etc.) are rejected.
- **Query validation**: Queries are validated before execution to ensure they are SELECT-only.
- **Parameterized queries**: Support for parameterized queries prevents SQL injection.
- **Result limiting**: Default limit of 1000 rows, configurable up to 10000 rows.
- **Password security**: 
  - Passwords are stored in the `mcp_connections` table (consider encryption for production)
  - Passwords are always masked in responses (list/get operations)
  - Passwords are never exposed in operation responses
- **Master connection protection**: The "master" connection cannot be deleted

## Error Handling

The server returns proper MCP error responses for:
- Invalid parameters
- Database connection errors
- Query execution errors
- Missing required parameters (especially `connection_name`)
- Non-SELECT queries (security violation)
- Connection not found errors
- Duplicate connection name errors

## Testing

To test the server manually:

```bash
# Set master database connection
export POSTGRES_DB_DSN="postgres://user:password@localhost:5432/mydb?sslmode=disable"

# Test initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | ./mcp-postgres

# Test tools/list
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./mcp-postgres

# Test list_schemas with connection_name
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"list_schemas","connection_name":"master"}]}}}' | ./mcp-postgres

# Test create_connection
echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"apply_operations","arguments":{"operations":[{"type":"create_connection","name":"test_db","host":"localhost","database":"testdb","user":"testuser","password":"testpass"}]}}}' | ./mcp-postgres
```

## Troubleshooting

### Database Connection Errors

If you see "failed to open database" errors:
- Verify `POSTGRES_DB_DSN` is set correctly
- Check database credentials
- Ensure database is accessible from the server location
- Verify PostgreSQL is running

### Connection Not Found Errors

If you see "connection not found" errors:
- Verify the connection name is correct (case-sensitive)
- Use `list_connections` to see all available connections
- Ensure the connection was created successfully

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
- Verify you're using the correct connection name

### Master Connection Issues

If the master connection is not available:
- Check that `POSTGRES_DB_DSN` is set correctly
- Verify the master database is accessible
- Check server logs for initialization errors
- The master connection should be automatically created on startup

## License

Part of the Code-Aria Internal MCP Servers project.
