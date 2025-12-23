package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	// Initialize master database connection
	if err := initMasterDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize master database: %v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// Initialize handshake
	if err := handleInitialize(scanner, encoder); err != nil {
		fmt.Fprintf(os.Stderr, "Initialize failed: %v\n", err)
		os.Exit(1)
	}

	// Handle requests
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg MCPMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Method != "" {
			handleRequest(&msg, encoder)
		}
	}
}

// initMasterDB initializes the master database connection and sets up the mcp_connections table
func initMasterDB() error {
	masterDSN := os.Getenv("POSTGRES_DB_DSN")
	if masterDSN == "" {
		return fmt.Errorf("POSTGRES_DB_DSN environment variable is required")
	}

	db, err := sql.Open("postgres", masterDSN)
	if err != nil {
		return fmt.Errorf("failed to open master database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping master database: %w", err)
	}

	masterDB = db

	// Ensure mcp_connections table exists
	if err := ensureMCPConnectionsTable(); err != nil {
		return fmt.Errorf("failed to ensure mcp_connections table: %w", err)
	}

	// Initialize master connection in table
	if err := initMasterConnection(); err != nil {
		return fmt.Errorf("failed to initialize master connection: %w", err)
	}

	return nil
}

func handleInitialize(scanner *bufio.Scanner, encoder *json.Encoder) error {
	// Read initialize request
	if !scanner.Scan() {
		return fmt.Errorf("no initialize request")
	}

	var initReq MCPMessage
	if err := json.Unmarshal(scanner.Bytes(), &initReq); err != nil {
		return fmt.Errorf("failed to parse initialize: %w", err)
	}

	// Send initialize response
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      initReq.ID,
		Result: InitializeResponse{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    "mcp-postgres",
				Version: "1.0.0",
			},
		},
	}

	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to send initialize response: %w", err)
	}

	// Read initialized notification
	if !scanner.Scan() {
		return fmt.Errorf("no initialized notification")
	}

	var initializedMsg MCPMessage
	if err := json.Unmarshal(scanner.Bytes(), &initializedMsg); err != nil {
		return fmt.Errorf("failed to parse initialized notification: %w", err)
	}

	// Validate that this is the initialized notification
	if initializedMsg.Method != "notifications/initialized" {
		return fmt.Errorf("expected initialized notification, got method: %s", initializedMsg.Method)
	}

	return nil
}

func handleRequest(msg *MCPMessage, encoder *json.Encoder) {
	switch msg.Method {
	case "tools/list":
		handleToolsList(msg, encoder)
	case "tools/call":
		handleToolCall(msg, encoder)
	default:
		sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown method: %s", msg.Method), nil)
	}
}

func handleToolsList(msg *MCPMessage, encoder *json.Encoder) {
	tools := []Tool{
		{
			Name: "apply_operations",
			Description: `Execute multiple PostgreSQL operations in a single batch call. This tool provides read-only access to PostgreSQL databases and connection management.

Available operations:

Database Operations:
1. list_schemas - List all schemas in the database (excluding system schemas like pg_catalog, information_schema)
   Parameters: connection_name (optional, defaults to 'master') - Name of the connection to use
   Returns: Array of schema names

2. list_tables - List all tables in a specified schema with metadata (schema name, table name, table type)
   Parameters: connection_name (optional, defaults to 'master'), schema (optional, defaults to 'public')
   Returns: Array of table objects with schema, table_name, and table_type fields

3. describe_table - Get detailed table schema information including columns, data types, constraints, indexes, and metadata
   Parameters: connection_name (optional, defaults to 'master'), table_name (required), schema (optional, defaults to 'public')
   Returns: Table schema object with columns array containing name, type, nullable, default, constraints, indexes, and position

4. query - Execute a SELECT query to retrieve data from the database
   Parameters: connection_name (optional, defaults to 'master'), query (required, must be a SELECT statement), params (optional array for parameterized queries), limit (optional, default 1000, max 10000)
   Returns: Array of result objects (one per row) with column names as keys
   Security: Only SELECT queries are allowed. INSERT, UPDATE, DELETE, DROP, and other modification operations are rejected.

5. get_connection_info - Get connection information including host, port, database, user (password is masked for security)
   Parameters: connection_name (optional, defaults to 'master')
   Returns: Connection info object with masked connection string and parsed components (host, port, database, user, sslmode, description)

Connection Management Operations:
6. create_connection - Create a new database connection configuration
   Parameters: name (required), host (required), port (optional, default 5432), database (required), user (required), password (required), sslmode (optional, default 'disable'), description (optional)
   Returns: Created connection object (password masked)

7. list_connections - List all configured connections (passwords are masked)
   Parameters: None
   Returns: Array of connection objects (passwords masked)

8. get_connection - Get a connection configuration by name (password is masked)
   Parameters: name (required)
   Returns: Connection object (password masked)

9. update_connection - Update a connection configuration
   Parameters: name (required), other fields optional (host, port, database, user, password, sslmode, description)
   Returns: Updated connection object (password masked)

10. delete_connection - Delete a connection configuration (cannot delete 'master' connection)
    Parameters: name (required)
    Returns: Success confirmation

11. rename_connection - Rename a connection (cannot rename 'master' connection)
    Parameters: old_name (required), new_name (required)
    Returns: Renamed connection object (password masked)

Connection Management: Connections are stored in the master database (configured via POSTGRES_DB_DSN). The master connection is automatically created on startup with the name 'master'. Use connection management operations to add, view, update, or remove connections. For database operations, if connection_name is not provided, it defaults to 'master'.

Examples:
- List schemas (uses master by default): {"type": "list_schemas"}
- List schemas with explicit connection: {"type": "list_schemas", "connection_name": "master"}
- List tables: {"type": "list_tables", "schema": "public"}
- Describe a table: {"type": "describe_table", "table_name": "users", "schema": "public"}
- Query with parameters: {"type": "query", "query": "SELECT * FROM users WHERE id = $1", "params": [123], "limit": 10}
- Query different database: {"type": "query", "connection_name": "prod_db", "query": "SELECT * FROM products LIMIT 10"}
- Create connection: {"type": "create_connection", "name": "prod_db", "host": "prod.example.com", "database": "mydb", "user": "myuser", "password": "mypass"}
- List connections: {"type": "list_connections"}
- Get connection: {"type": "get_connection", "name": "prod_db"}
- Rename connection: {"type": "rename_connection", "old_name": "prod_db", "new_name": "production_db"}`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operations": map[string]interface{}{
						"type":        "array",
						"description": "List of PostgreSQL operations to execute. Each operation is an object with a 'type' field and operation-specific parameters.",
						"items": map[string]interface{}{
							"type":        "object",
							"description": "Operation object. Must include 'type' field. Available types: list_schemas, list_tables, describe_table, query, get_connection_info, create_connection, list_connections, get_connection, update_connection, delete_connection, rename_connection",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"enum":        []string{"list_schemas", "list_tables", "describe_table", "query", "get_connection_info", "create_connection", "list_connections", "get_connection", "update_connection", "delete_connection", "rename_connection"},
									"description": "Operation type. Database operations: 'list_schemas', 'list_tables', 'describe_table', 'query', 'get_connection_info'. Connection management: 'create_connection', 'list_connections', 'get_connection', 'update_connection', 'delete_connection', 'rename_connection'.",
								},
								"connection_name": map[string]interface{}{
									"type":        "string",
									"description": "Connection name. Optional for database operations (list_schemas, list_tables, describe_table, query, get_connection_info). Defaults to 'master' if not provided. Must be a configured connection name.",
								},
								"schema": map[string]interface{}{
									"type":        "string",
									"description": "Schema name. Used by list_tables and describe_table operations. Defaults to 'public' if not specified.",
								},
								"table_name": map[string]interface{}{
									"type":        "string",
									"description": "Table name. Required for describe_table operation. Should be the name of the table you want to inspect.",
								},
								"query": map[string]interface{}{
									"type":        "string",
									"description": "SQL SELECT query to execute. Required for query operation. Must be a SELECT statement only - INSERT, UPDATE, DELETE, DROP and other modification operations are rejected for security. Supports parameterized queries using $1, $2, etc. placeholders.",
								},
								"params": map[string]interface{}{
									"type":        "array",
									"description": "Query parameters for parameterized queries. Used with query operation. Array of values that correspond to $1, $2, etc. placeholders in the query string. Example: [123, 'text'] for query with $1 and $2.",
								},
								"limit": map[string]interface{}{
									"type":        "integer",
									"description": "Maximum number of rows to return. Used with query operation. Default: 1000, Maximum: 10000. Automatically adds LIMIT clause if not present in query.",
									"minimum":     1,
									"maximum":     10000,
								},
								"name": map[string]interface{}{
									"type":        "string",
									"description": "Connection name. Required for create_connection, get_connection, update_connection, delete_connection operations.",
								},
								"host": map[string]interface{}{
									"type":        "string",
									"description": "Database host. Required for create_connection. Optional for update_connection.",
								},
								"port": map[string]interface{}{
									"type":        "integer",
									"description": "Database port. Optional for create_connection and update_connection (default: 5432).",
									"minimum":     1,
									"maximum":     65535,
								},
								"database": map[string]interface{}{
									"type":        "string",
									"description": "Database name. Required for create_connection. Optional for update_connection.",
								},
								"user": map[string]interface{}{
									"type":        "string",
									"description": "Database user. Required for create_connection. Optional for update_connection.",
								},
								"password": map[string]interface{}{
									"type":        "string",
									"description": "Database password. Required for create_connection. Optional for update_connection (only needed if changing password).",
								},
								"sslmode": map[string]interface{}{
									"type":        "string",
									"description": "SSL mode. Optional for create_connection and update_connection (default: 'disable'). Common values: 'disable', 'require', 'verify-ca', 'verify-full'.",
								},
								"description": map[string]interface{}{
									"type":        "string",
									"description": "Connection description. Optional for create_connection and update_connection.",
								},
								"old_name": map[string]interface{}{
									"type":        "string",
									"description": "Old connection name. Required for rename_connection operation.",
								},
								"new_name": map[string]interface{}{
									"type":        "string",
									"description": "New connection name. Required for rename_connection operation.",
								},
							},
							"required": []string{"type"},
						},
					},
				},
				"required": []string{"operations"},
			},
		},
	}

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ToolsListResponse{
			Tools: tools,
		},
	}

	if err := encoder.Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode tools list response: %v\n", err)
	}
}

func handleToolCall(msg *MCPMessage, encoder *json.Encoder) {
	var req ToolsCallRequest
	reqJSON, err := json.Marshal(msg.Params)
	if err != nil {
		sendError(encoder, msg.ID, -32602, fmt.Sprintf("failed to marshal params: %v", err), nil)
		return
	}
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		sendError(encoder, msg.ID, -32602, fmt.Sprintf("failed to unmarshal params: %v", err), nil)
		return
	}

	if req.Name == "apply_operations" {
		handleBatchOperations(msg, encoder, req.Arguments)
		return
	}

	// Individual tool calls are no longer exposed, but kept for internal use
	sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown tool: %s. Use apply_operations for batch operations", req.Name), nil)
}

func handleBatchOperations(msg *MCPMessage, encoder *json.Encoder, args map[string]interface{}) {
	operations, ok := args["operations"].([]interface{})
	if !ok {
		sendError(encoder, msg.ID, -32602, "operations array is required", nil)
		return
	}

	if len(operations) == 0 {
		sendError(encoder, msg.ID, -32602, "operations array cannot be empty", nil)
		return
	}

	var results []map[string]interface{}

	for _, op := range operations {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			results = append(results, map[string]interface{}{
				"operation": "unknown",
				"params":    map[string]interface{}{},
				"status":    "Error",
				"message":   "Invalid operation format",
			})
			continue
		}

		opType, ok := opMap["type"].(string)
		if !ok {
			results = append(results, map[string]interface{}{
				"operation": "unknown",
				"params":    map[string]interface{}{},
				"status":    "Error",
				"message":   "Operation type is required",
			})
			continue
		}

		// Extract operation-specific arguments as params
		params := make(map[string]interface{})
		for k, v := range opMap {
			if k != "type" {
				params[k] = v
			}
		}

		// Execute operation based on type
		var result string
		var err error

		switch opType {
		case "list_schemas":
			result, err = toolListSchemas(params)
		case "list_tables":
			result, err = toolListTables(params)
		case "describe_table":
			result, err = toolDescribeTable(params)
		case "query":
			result, err = toolQuery(params)
		case "get_connection_info":
			result, err = toolGetConnectionInfo(params)
		case "create_connection":
			result, err = toolCreateConnection(params)
		case "list_connections":
			result, err = toolListConnections(params)
		case "get_connection":
			result, err = toolGetConnection(params)
		case "update_connection":
			result, err = toolUpdateConnection(params)
		case "delete_connection":
			result, err = toolDeleteConnection(params)
		case "rename_connection":
			result, err = toolRenameConnection(params)
		default:
			err = fmt.Errorf("unknown operation type: %s", opType)
		}

		// Optimize params before adding to results
		optimizedParams := optimizeParams(opType, params)

		if err != nil {
			results = append(results, map[string]interface{}{
				"operation": opType,
				"params":    optimizedParams,
				"status":    "Error",
				"message":   err.Error(),
			})
		} else {
			// Parse JSON result if possible, otherwise use as string
			var parsedResult interface{}
			if jsonErr := json.Unmarshal([]byte(result), &parsedResult); jsonErr == nil {
				// Successfully parsed JSON, use parsed result
			} else {
				// Not JSON, use string as-is
				parsedResult = result
			}

			results = append(results, map[string]interface{}{
				"operation": opType,
				"params":    optimizedParams,
				"status":    "Success",
				"result":    parsedResult,
			})
		}
	}

	// Serialize results to JSON text for MCP-compliant response format
	resultsJSON, err := json.Marshal(map[string]interface{}{
		"results": results,
	})
	if err != nil {
		sendError(encoder, msg.ID, -32700, fmt.Sprintf("Failed to marshal results: %v", err), nil)
		return
	}

	// Return results in MCP-compliant format using ToolsCallResponse
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ToolsCallResponse{
			Content: []Content{
				{
					Type: "text",
					Text: string(resultsJSON),
				},
			},
			IsError: false,
		},
	}

	if err := encoder.Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode batch operations response: %v\n", err)
	}
}

// optimizeParams optimizes params for response by truncating long string values (> 20 lines)
func optimizeParams(opType string, params map[string]interface{}) map[string]interface{} {
	optimized := make(map[string]interface{})

	// Fields to always preserve as-is (metadata)
	preserveFields := map[string]bool{
		"connection_name": true,
		"schema":          true,
		"table_name":      true,
		"limit":           true,
		"name":            true,
		"host":            true,
		"port":            true,
		"database":        true,
		"user":            true,
		"sslmode":         true,
		"description":     true,
	}

	// Fields to never expose (security)
	sensitiveFields := map[string]bool{
		"password": true,
	}

	// For postgres operations, truncate long string values (> 20 lines)
	for k, v := range params {
		if sensitiveFields[k] {
			// Don't include sensitive fields in responses
			continue
		}
		if preserveFields[k] {
			// Always preserve metadata fields as-is
			optimized[k] = v
		} else if strVal, ok := v.(string); ok {
			// Truncate string values > 20 lines
			lines := strings.Split(strVal, "\n")
			if len(lines) > 20 {
				truncated := strings.Join(lines[:20], "\n")
				optimized[k] = fmt.Sprintf("%s\n... (truncated, %d total lines)", truncated, len(lines))
			} else {
				optimized[k] = v
			}
		} else {
			// Preserve non-string values as-is
			optimized[k] = v
		}
	}

	return optimized
}

func sendError(encoder *json.Encoder, id interface{}, code int, message string, data interface{}) {
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode error response: %v\n", err)
	}
}
