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

## Operating Modes

The mcp-postgres server operates in two distinct modes depending on whether the `POSTGRES_DB_DSN` environment variable is set:

### PostgreSQL Mode (Default)

When `POSTGRES_DB_DSN` is set to a valid PostgreSQL connection string:

- A master PostgreSQL database stores connection configurations
- The "master" connection is automatically created on startup
- Database operations can optionally specify `connection_name` (defaults to "master")
- Best for: Production environments with existing PostgreSQL infrastructure

**Example:**
```bash
export POSTGRES_DB_DSN="postgres://user:pass@localhost:5432/masterdb"
./mcp-postgres
```

### SQLite Fallback Mode

When `POSTGRES_DB_DSN` is empty, unset, or not provided:

- An embedded SQLite database (`mcp-postgres.db`) stores connections in the current directory
- No master connection exists
- Database operations **must** specify `connection_name` (required parameter)
- Best for: Development, testing, or scenarios without PostgreSQL

**Example:**
```bash
unset POSTGRES_DB_DSN  # or leave empty
./mcp-postgres
```

**Key Differences:**

| Feature | PostgreSQL Mode | SQLite Fallback Mode |
|---------|----------------|---------------------|
| Database | PostgreSQL server | Embedded SQLite file |
| Storage | External PostgreSQL | `mcp-postgres.db` file |
| Master Connection | Auto-created as "master" | Does not exist |
| connection_name Parameter | Optional (defaults to "master") | **Required** |
| Default Connection | Available (master) | None - must create explicitly |
| Use Case | Production, shared environments | Development, local testing |

**Important:** In SQLite mode, you must always create connections first using `create_connection` and then reference them by name in all database operations.

## Prerequisites

- Go 1.24.1 or higher
- PostgreSQL database (for master connection) OR SQLite for fallback mode
- Database connection string for master database (optional for SQLite fallback)

## Configuration

The server requires a master database connection string via the `POSTGRES_DB_DSN` environment variable. This database is used to store connection configurations in the `mcp_connections` table.

### SQLite Fallback Mode

If `POSTGRES_DB_DSN` is not set or left empty, the server will automatically use an embedded SQLite database (`mcp-postgres.db`) in the current working directory instead of PostgreSQL. This fallback mode is useful for:

- Local development without requiring a PostgreSQL server
- Testing and experimentation
- Scenarios where PostgreSQL is not available

**Key differences in SQLite fallback mode:**
- Database file: `mcp-postgres.db` (created automatically)
- No master connection is created (only applicable in PostgreSQL mode)
- All other features work identically (connection management, querying, etc.)
- Data persists between runs in the local file

### Environment Variables

You can set the environment variable directly:

```bash
# Use PostgreSQL
export POSTGRES_DB_DSN="postgres://user:password@host:port/dbname?sslmode=disable"

# Or use SQLite fallback (leave empty or unset)
unset POSTGRES_DB_DSN
```

Or use a `.env` file (recommended for development):

1. Copy `env.example` to `.env`:
   ```bash
   cp env.example .env
   ```

2. Edit `.env`:
   ```bash
   # For PostgreSQL mode:
   POSTGRES_DB_DSN=postgres://user:password@host:port/dbname?sslmode=disable

   # For SQLite fallback mode:
   POSTGRES_DB_DSN=
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

**PostgreSQL Mode (POSTGRES_DB_DSN is set):**
1. Connect to the master database using `POSTGRES_DB_DSN`
2. Create the `mcp_connections` table if it doesn't exist
3. Automatically add the master connection as "master" in the connections table

**SQLite Fallback Mode (POSTGRES_DB_DSN is empty/not set):**
1. Create/open SQLite database file `mcp-postgres.db` in the current directory
2. Create the `mcp_connections` table if it doesn't exist
3. No master connection is created (you can add connections via `create_connection`)

## MCP Tools

The server exposes a single `apply_operations` tool that supports batch execution of multiple operations.

### Database Operations

#### list_schemas

List all schemas in the database (excluding system schemas).

**Parameters:**
- `connection_name` (string, optional): Name of the connection to use.
  - **PostgreSQL mode**: Defaults to 'master' if not provided
  - **SQLite mode**: Required (no default connection exists)

**Returns:** Array of schema names

**Example:**
```json
{
  "type": "list_schemas",
  "connection_name": "my_connection"
}
```

#### list_tables

List tables in a schema.

**Parameters:**
- `connection_name` (string, optional): Name of the connection to use.
  - **PostgreSQL mode**: Defaults to 'master' if not provided
  - **SQLite mode**: Required (no default connection exists)
- `schema` (string, optional): Schema name (defaults to 'public')

**Returns:** Array of table objects with schema, table_name, and table_type

**Example:**
```json
{
  "type": "list_tables",
  "connection_name": "my_connection",
  "schema": "public"
}
```

#### describe_table

Get detailed table schema information including columns, types, constraints, and indexes.

**Parameters:**
- `connection_name` (string, optional): Name of the connection to use.
  - **PostgreSQL mode**: Defaults to 'master' if not provided
  - **SQLite mode**: Required (no default connection exists)
- `table_name` (string, required): Table name
- `schema` (string, optional): Schema name (defaults to 'public')

**Returns:** Table schema object with columns, constraints, and indexes

**Example:**
```json
{
  "type": "describe_table",
  "connection_name": "my_connection",
  "table_name": "users",
  "schema": "public"
}
```

#### query

Execute a SELECT query.

**Parameters:**
- `connection_name` (string, optional): Name of the connection to use.
  - **PostgreSQL mode**: Defaults to 'master' if not provided
  - **SQLite mode**: Required (no default connection exists)
- `query` (string, required): SELECT query to execute
- `params` (array, optional): Query parameters for parameterized queries
- `limit` (integer, optional): Maximum rows to return (default: 1000, max: 10000)

**Returns:** Array of result objects (one per row)

**Example:**
```json
{
  "type": "query",
  "connection_name": "my_connection",
  "query": "SELECT id, name, email FROM users WHERE active = $1",
  "params": [true],
  "limit": 100
}
```

#### get_connection_info

Get connection information including host, port, database, user (password is masked for security).

**Parameters:**
- `connection_name` (string, optional): Name of the connection to query.
  - **PostgreSQL mode**: Defaults to 'master' if not provided
  - **SQLite mode**: Required (no default connection exists)

**Returns:** Connection info object with masked connection string and parsed components

**Example:**
```json
{
  "type": "get_connection_info",
  "connection_name": "my_connection"
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
# PostgreSQL mode
export POSTGRES_DB_DSN="postgres://user:password@localhost:5432/masterdb?sslmode=disable"
./mcp-postgres

# SQLite fallback mode (no PostgreSQL required)
unset POSTGRES_DB_DSN
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
          "connection_name": "my_connection"
        },
        {
          "type": "list_tables",
          "connection_name": "my_connection",
          "schema": "public"
        },
        {
          "type": "describe_table",
          "connection_name": "my_connection",
          "table_name": "users",
          "schema": "public"
        },
        {
          "type": "query",
          "connection_name": "my_connection",
          "query": "SELECT COUNT(*) as total FROM users",
          "limit": 1
        }
      ]
    }
  }
}
```

**Note:** In SQLite mode, you must create connections first before using them. In PostgreSQL mode, the "master" connection is available by default.

### Connection Management Workflow

**PostgreSQL Mode:**
1. **Start the server** with `POSTGRES_DB_DSN` pointing to your master database
2. **Master connection** is automatically created as "master"
3. **Use database operations** without specifying connection_name (defaults to "master")
4. **Add additional connections** using `create_connection` if needed
5. **Use connections** by name in database operations

**SQLite Fallback Mode:**
1. **Start the server** without `POSTGRES_DB_DSN` (or leave it empty)
2. **SQLite database** is created as `mcp-postgres.db` in the current directory
3. **Create connections** using `create_connection` (no master connection exists)
4. **Always specify connection_name** in database operations (required parameter)
5. **Use connections** by name in database operations

**Creating a connection (works in both modes):**
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

**Using a connection:**
```json
{
  "type": "query",
  "connection_name": "prod_db",
  "query": "SELECT * FROM users LIMIT 10"
}
```

**Important:** In SQLite mode, always specify `connection_name`. If you don't, you'll get an error: `"connection_name is required when using SQLite mode (no default 'master' connection exists)"`

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

### Missing connection_name Parameter

If you see "connection_name is required when using SQLite mode" errors:
- You're running in SQLite fallback mode (POSTGRES_DB_DSN is not set)
- In SQLite mode, you must always specify `connection_name` in database operations
- First create a connection using `create_connection`, then use it by name
- Examples of operations that require connection_name in SQLite mode:
  - `list_schemas`
  - `list_tables`
  - `describe_table`
  - `query`
  - `get_connection_info`

**To fix:** Either:
1. Add `connection_name` to your operations (recommended for SQLite mode)
2. Or set `POSTGRES_DB_DSN` to use PostgreSQL mode with a default master connection

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

**PostgreSQL Mode:**
If the master connection is not available:
- Check that `POSTGRES_DB_DSN` is set correctly
- Verify the master database is accessible
- Check server logs for initialization errors
- The master connection should be automatically created on startup

**SQLite Fallback Mode:**
- The master connection does not exist in SQLite mode
- You need to create connections explicitly using `create_connection`
- Check that `mcp-postgres.db` file is writable in the current directory
- Verify that SQLite mode is active (check stderr for "using SQLite database" message)

## License

Part of the Code-Aria Internal MCP Servers project.


