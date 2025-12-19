package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

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
			Description: `Execute multiple PostgreSQL read operations in a single batch call. This tool provides read-only access to PostgreSQL databases.

Available operations:
1. list_schemas - List all schemas in the database (excluding system schemas like pg_catalog, information_schema)
   Parameters: connection_string (optional, overrides POSTGRES_DB_DSN env var)
   Returns: Array of schema names

2. list_tables - List all tables in a specified schema with metadata (schema name, table name, table type)
   Parameters: schema (optional, defaults to 'public'), connection_string (optional)
   Returns: Array of table objects with schema, table_name, and table_type fields

3. describe_table - Get detailed table schema information including columns, data types, constraints, indexes, and metadata
   Parameters: table_name (required), schema (optional, defaults to 'public'), connection_string (optional)
   Returns: Table schema object with columns array containing name, type, nullable, default, constraints, indexes, and position

4. query - Execute a SELECT query to retrieve data from the database
   Parameters: query (required, must be a SELECT statement), params (optional array for parameterized queries), limit (optional, default 1000, max 10000), connection_string (optional)
   Returns: Array of result objects (one per row) with column names as keys
   Security: Only SELECT queries are allowed. INSERT, UPDATE, DELETE, DROP, and other modification operations are rejected.

5. get_connection_info - Get connection information including host, port, database, user (password is masked for security)
   Parameters: connection_string (optional, overrides POSTGRES_DB_DSN env var)
   Returns: Connection info object with masked connection string, source (environment/parameter), and parsed components (host, port, database, user, parameters)

Connection: The connection string can be provided via POSTGRES_DB_DSN environment variable or per-operation via connection_string parameter. Format: postgres://user:password@host:port/dbname?sslmode=disable

Examples:
- List schemas: {"type": "list_schemas"}
- List tables in public schema: {"type": "list_tables", "schema": "public"}
- Describe a table: {"type": "describe_table", "table_name": "users", "schema": "public"}
- Query with parameters: {"type": "query", "query": "SELECT * FROM users WHERE id = $1", "params": [123], "limit": 10}
- Get connection info: {"type": "get_connection_info"}`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operations": map[string]interface{}{
						"type":        "array",
						"description": "List of PostgreSQL read operations to execute. Each operation is an object with a 'type' field and operation-specific parameters.",
						"items": map[string]interface{}{
							"type":        "object",
							"description": "Operation object. Must include 'type' field. Available types: list_schemas, list_tables, describe_table, query",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"enum":        []string{"list_schemas", "list_tables", "describe_table", "query", "get_connection_info"},
									"description": "Operation type. 'list_schemas': List all schemas. 'list_tables': List tables in a schema. 'describe_table': Get table schema details. 'query': Execute SELECT query. 'get_connection_info': Get connection information with masked password.",
								},
								"connection_string": map[string]interface{}{
									"type":        "string",
									"description": "Optional PostgreSQL connection string. Overrides POSTGRES_DB_DSN environment variable if provided. Format: postgres://user:password@host:port/dbname?sslmode=disable",
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

	// Convert results to JSON string for Content
	resultsJSON, err := json.Marshal(map[string]interface{}{
		"results": results,
	})
	if err != nil {
		sendError(encoder, msg.ID, -32603, fmt.Sprintf("failed to marshal results: %v", err), nil)
		return
	}

	// Return results in MCP ToolsCallResponse format
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
		fmt.Fprintf(os.Stderr, "Failed to encode response: %v\n", err)
	}
}

// optimizeParams optimizes params for response by truncating long string values (> 20 lines)
func optimizeParams(opType string, params map[string]interface{}) map[string]interface{} {
	optimized := make(map[string]interface{})

	// Fields to always preserve as-is (metadata)
	preserveFields := map[string]bool{
		"connection_string": false, // Don't expose connection strings in responses
		"schema":            true,
		"table_name":        true,
		"limit":             true,
	}

	// For postgres operations, truncate long string values (> 20 lines)
	for k, v := range params {
		if k == "connection_string" {
			// Don't include connection strings in responses for security
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
