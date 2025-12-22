package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

// setupTestDB initializes the master database connection and sets up the test environment
func setupTestDB(t *testing.T) {
	connStr := os.Getenv("POSTGRES_DB_DSN")
	if connStr == "" {
		t.Skip("POSTGRES_DB_DSN environment variable not set, skipping database tests")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to open master database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping master database: %v", err)
	}

	masterDB = db

	// Ensure mcp_connections table exists
	if err := ensureMCPConnectionsTable(); err != nil {
		t.Fatalf("Failed to ensure mcp_connections table: %v", err)
	}

	// Initialize master connection
	if err := initMasterConnection(); err != nil {
		t.Fatalf("Failed to initialize master connection: %v", err)
	}
}

// getTestConnectionName returns "master" as the default test connection name
func getTestConnectionName() string {
	return "master"
}

// TestToolListSchemas tests the list_schemas operation
func TestToolListSchemas(t *testing.T) {
	setupTestDB(t)

	params := map[string]interface{}{
		"connection_name": getTestConnectionName(),
	}

	result, err := toolListSchemas(params)
	if err != nil {
		t.Fatalf("toolListSchemas() error = %v", err)
	}

	// Parse result as JSON array
	var schemas []string
	if err := json.Unmarshal([]byte(result), &schemas); err != nil {
		t.Fatalf("Failed to parse result as JSON array: %v", err)
	}

	// Verify we got some schemas
	if len(schemas) == 0 {
		t.Error("Expected at least one schema, got none")
	}

	// Verify common schemas exist
	found := make(map[string]bool)
	for _, schema := range schemas {
		found[schema] = true
	}

	// At least 'public' should exist (though it might be filtered out)
	// We just log if it's not found, as some databases might not have it
	if !found["public"] {
		t.Logf("Note: 'public' schema not found. Available schemas: %v", schemas)
	}

	t.Logf("Found %d schemas: %v", len(schemas), schemas)
}

// TestToolListTables tests the list_tables operation
func TestToolListTables(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "List tables in public schema",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"schema":          "public",
			},
		},
		{
			name: "List tables with default schema",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolListTables(tt.params)
			if err != nil {
				t.Fatalf("toolListTables() error = %v", err)
			}

			// Parse result as JSON array
			var tables []map[string]interface{}
			if err := json.Unmarshal([]byte(result), &tables); err != nil {
				t.Fatalf("Failed to parse result as JSON array: %v", err)
			}

			// Verify structure
			for _, table := range tables {
				if _, ok := table["schema"]; !ok {
					t.Error("Table missing 'schema' field")
				}
				if _, ok := table["table_name"]; !ok {
					t.Error("Table missing 'table_name' field")
				}
				if _, ok := table["table_type"]; !ok {
					t.Error("Table missing 'table_type' field")
				}
			}

			t.Logf("Found %d tables in schema", len(tables))
		})
	}
}

// TestToolDescribeTable tests the describe_table operation
func TestToolDescribeTable(t *testing.T) {
	setupTestDB(t)

	// First, get a list of tables to test with
	params := map[string]interface{}{
		"connection_name": getTestConnectionName(),
		"schema":          "public",
	}

	tablesResult, err := toolListTables(params)
	if err != nil {
		t.Fatalf("Failed to list tables: %v", err)
	}

	var tables []map[string]interface{}
	if err := json.Unmarshal([]byte(tablesResult), &tables); err != nil {
		t.Fatalf("Failed to parse tables: %v", err)
	}

	if len(tables) == 0 {
		t.Skip("No tables found in public schema, skipping describe_table test")
	}

	// Test with the first table
	firstTable := tables[0]
	tableName := firstTable["table_name"].(string)
	schema := firstTable["schema"].(string)

	describeParams := map[string]interface{}{
		"connection_name": getTestConnectionName(),
		"table_name":     tableName,
		"schema":         schema,
	}

	result, err := toolDescribeTable(describeParams)
	if err != nil {
		t.Fatalf("toolDescribeTable() error = %v", err)
	}

	// Parse result
	var tableSchema map[string]interface{}
	if err := json.Unmarshal([]byte(result), &tableSchema); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify structure
	if _, ok := tableSchema["schema"]; !ok {
		t.Error("Result missing 'schema' field")
	}
	if _, ok := tableSchema["table"]; !ok {
		t.Error("Result missing 'table' field")
	}
	if _, ok := tableSchema["columns"]; !ok {
		t.Error("Result missing 'columns' field")
	}

	// Verify columns structure
	columns, ok := tableSchema["columns"].([]interface{})
	if !ok {
		t.Fatal("Columns is not an array")
	}

	if len(columns) == 0 {
		t.Error("Expected at least one column, got none")
	}

	// Verify first column structure
	if len(columns) > 0 {
		col, ok := columns[0].(map[string]interface{})
		if !ok {
			t.Fatal("Column is not an object")
		}

		requiredFields := []string{"name", "type", "nullable", "position"}
		for _, field := range requiredFields {
			if _, ok := col[field]; !ok {
				t.Errorf("Column missing required field: %s", field)
			}
		}
	}

	t.Logf("Described table %s.%s with %d columns", schema, tableName, len(columns))
}

// TestToolQuery tests the query operation
func TestToolQuery(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name   string
		params map[string]interface{}
		valid  bool
	}{
		{
			name: "Simple SELECT query",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"query":          "SELECT 1 as test_value",
				"limit":          10.0,
			},
			valid: true,
		},
		{
			name: "SELECT query with parameters",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"query":          "SELECT $1::text as value",
				"params":         []interface{}{"test"},
				"limit":          10.0,
			},
			valid: true,
		},
		{
			name: "SELECT query with multiple parameters",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"query":          "SELECT $1::int as num, $2::text as text",
				"params":         []interface{}{1, "test"},
				"limit":          10.0,
			},
			valid: true,
		},
		{
			name: "Query with limit",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"query":          "SELECT generate_series(1, 100) as num",
				"limit":          5.0,
			},
			valid: true,
		},
		{
			name: "Missing query parameter",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
			},
			valid: false,
		},
		{
			name: "Non-SELECT query (should fail)",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"query":          "INSERT INTO test VALUES (1)",
			},
			valid: false,
		},
		{
			name: "Query with dangerous keyword (should fail)",
			params: map[string]interface{}{
				"connection_name": getTestConnectionName(),
				"query":          "SELECT * FROM test; DROP TABLE test;",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolQuery(tt.params)

			if tt.valid {
				if err != nil {
					t.Fatalf("toolQuery() error = %v", err)
				}

				// Parse result as JSON array
				var rows []map[string]interface{}
				if err := json.Unmarshal([]byte(result), &rows); err != nil {
					t.Fatalf("Failed to parse result as JSON array: %v", err)
				}

				t.Logf("Query returned %d rows", len(rows))

				// Verify limit was respected (if specified)
				if limit, ok := tt.params["limit"].(float64); ok {
					maxRows := int(limit)
					if len(rows) > maxRows {
						t.Errorf("Query returned %d rows, expected max %d", len(rows), maxRows)
					}
				}
			} else {
				if err == nil {
					t.Error("Expected error for invalid query, got none")
				} else {
					t.Logf("Correctly rejected invalid query: %v", err)
				}
			}
		})
	}
}

// TestToolQueryWithRealTable tests querying a real table if available
func TestToolQueryWithRealTable(t *testing.T) {
	setupTestDB(t)

	// First, try to find a table
	params := map[string]interface{}{
		"connection_name": getTestConnectionName(),
		"schema":          "public",
	}

	tablesResult, err := toolListTables(params)
	if err != nil {
		t.Fatalf("Failed to list tables: %v", err)
	}

	var tables []map[string]interface{}
	if err := json.Unmarshal([]byte(tablesResult), &tables); err != nil {
		t.Fatalf("Failed to parse tables: %v", err)
	}

	if len(tables) == 0 {
		t.Skip("No tables found, skipping real table query test")
	}

	// Query the first table
	firstTable := tables[0]
	tableName := firstTable["table_name"].(string)
	schema := firstTable["schema"].(string)

	queryParams := map[string]interface{}{
		"connection_name": getTestConnectionName(),
		"query":          "SELECT * FROM " + schema + "." + tableName + " LIMIT 5",
		"limit":          5.0,
	}

	result, err := toolQuery(queryParams)
	if err != nil {
		t.Fatalf("toolQuery() error = %v", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(result), &rows); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	t.Logf("Query returned %d rows from %s.%s", len(rows), schema, tableName)
}

// TestGetConnectionString tests connection string retrieval by name
func TestGetConnectionString(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Valid connection name",
			params: map[string]interface{}{
				"connection_name": "master",
			},
			wantErr: false,
		},
		{
			name: "Missing connection_name",
			params: map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "Invalid connection name",
			params: map[string]interface{}{
				"connection_name": "nonexistent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr, err := getConnectionString(tt.params)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				}
			} else {
				if err != nil {
					t.Fatalf("getConnectionString() error = %v", err)
				}
				if connStr == "" {
					t.Error("Expected connection string, got empty")
				}
			}
		})
	}
}

// TestValidateSelectQuery tests query validation
func TestValidateSelectQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "Valid SELECT",
			query:   "SELECT * FROM users",
			wantErr: false,
		},
		{
			name:    "SELECT with WHERE",
			query:   "SELECT id, name FROM users WHERE active = true",
			wantErr: false,
		},
		{
			name:    "SELECT with JOIN",
			query:   "SELECT u.id, p.name FROM users u JOIN profiles p ON u.id = p.user_id",
			wantErr: false,
		},
		{
			name:    "SELECT with comments",
			query:   "-- This is a comment\nSELECT * FROM users",
			wantErr: false,
		},
		{
			name:    "INSERT query (should fail)",
			query:   "INSERT INTO users VALUES (1, 'test')",
			wantErr: true,
		},
		{
			name:    "UPDATE query (should fail)",
			query:   "UPDATE users SET name = 'test'",
			wantErr: true,
		},
		{
			name:    "DELETE query (should fail)",
			query:   "DELETE FROM users",
			wantErr: true,
		},
		{
			name:    "DROP query (should fail)",
			query:   "DROP TABLE users",
			wantErr: true,
		},
		{
			name:    "SELECT with INSERT in comment (should pass)",
			query:   "SELECT * FROM users -- INSERT INTO test",
			wantErr: false,
		},
		{
			name:    "SELECT with dangerous keyword in string (should fail for security)",
			query:   "SELECT 'INSERT INTO test' as query",
			wantErr: true, // Rejected for security - validation is strict
		},
		{
			name:    "Multiple statements with SELECT (should fail)",
			query:   "SELECT * FROM users; INSERT INTO test VALUES (1)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSelectQuery(tt.query)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got none")
				} else {
					t.Logf("Correctly rejected query: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("validateSelectQuery() error = %v, expected no error", err)
				}
			}
		})
	}
}

// TestToolGetConnectionInfo tests the get_connection_info operation
func TestToolGetConnectionInfo(t *testing.T) {
	setupTestDB(t)

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "Get connection info for master",
			params: map[string]interface{}{
				"connection_name": "master",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toolGetConnectionInfo(tt.params)
			if err != nil {
				t.Fatalf("toolGetConnectionInfo() error = %v", err)
			}

			// Parse result
			var info map[string]interface{}
			if err := json.Unmarshal([]byte(result), &info); err != nil {
				t.Fatalf("Failed to parse result: %v", err)
			}

			// Verify required fields
			if _, ok := info["connection_string_masked"]; !ok {
				t.Error("Result missing 'connection_string_masked' field")
			}

			if _, ok := info["source"]; !ok {
				t.Error("Result missing 'source' field")
			}

			// Verify password is masked
			maskedStr, ok := info["connection_string_masked"].(string)
			if !ok {
				t.Fatal("connection_string_masked is not a string")
			}

			if !strings.Contains(maskedStr, "****") {
				t.Logf("Warning: Connection string doesn't contain masked password pattern. Got: %s", maskedStr)
			}

			// Verify connection name
			connName, ok := info["name"].(string)
			if !ok {
				t.Fatal("name is not a string")
			}

			if connName != "master" {
				t.Errorf("Expected connection name 'master', got '%s'", connName)
			}

			t.Logf("Connection info: %+v", info)
		})
	}
}

// TestMaskPasswordInConnectionString tests password masking
func TestMaskPasswordInConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard connection string with password",
			input:    "postgres://user:password@localhost:5432/dbname",
			expected: "postgres://user:****@localhost:5432/dbname",
		},
		{
			name:     "Connection string with query parameters",
			input:    "postgres://user:pass123@host:5432/db?sslmode=disable",
			expected: "postgres://user:****@host:5432/db?sslmode=disable",
		},
		{
			name:     "Connection string without password",
			input:    "postgres://user@localhost:5432/dbname",
			expected: "postgres://user@localhost:5432/dbname", // No password to mask
		},
		{
			name:     "Empty connection string",
			input:    "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPasswordInConnectionString(tt.input)
			
			// Verify password is masked (contains ****) only if password exists
			hasPassword := strings.Contains(tt.input, "://") && strings.Contains(tt.input, ":") && strings.Contains(tt.input, "@")
			if hasPassword {
				// Check if there's actually a password (format: user:password@)
				re := regexp.MustCompile(`://[^:]+:([^@]+)@`)
				matches := re.FindStringSubmatch(tt.input)
				if len(matches) > 1 && matches[1] != "" {
					// Has password, should be masked
					if !strings.Contains(result, "****") {
						t.Errorf("Password should be masked. Got: %s", result)
					}
					// Verify original password is not present
					if strings.Contains(result, matches[1]) {
						t.Errorf("Original password '%s' should not appear in masked string. Got: %s", matches[1], result)
					}
				} else {
					// No password, should not be masked
					if strings.Contains(result, "****") {
						t.Errorf("No password to mask, but found ****. Got: %s", result)
					}
				}
			}

			t.Logf("Input: %s -> Output: %s", tt.input, result)
		})
	}
}

// TestBatchOperations tests multiple operations in sequence
func TestBatchOperations(t *testing.T) {
	setupTestDB(t)

	// Test that we can run multiple operations
	operations := []map[string]interface{}{
		{
			"type":           "list_schemas",
			"connection_name": getTestConnectionName(),
		},
		{
			"type":           "list_tables",
			"schema":         "public",
			"connection_name": getTestConnectionName(),
		},
		{
			"type":           "query",
			"query":          "SELECT version() as pg_version",
			"connection_name": getTestConnectionName(),
			"limit":          1.0,
		},
	}

	for i, op := range operations {
		t.Run(op["type"].(string), func(t *testing.T) {
			var result string
			var err error

			switch op["type"] {
			case "list_schemas":
				result, err = toolListSchemas(op)
			case "list_tables":
				result, err = toolListTables(op)
			case "query":
				result, err = toolQuery(op)
			default:
				t.Fatalf("Unknown operation type: %s", op["type"])
			}

			if err != nil {
				t.Fatalf("Operation %d (%s) failed: %v", i, op["type"], err)
			}

			if result == "" {
				t.Errorf("Operation %d (%s) returned empty result", i, op["type"])
			}

			// Verify it's valid JSON
			var parsed interface{}
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Errorf("Operation %d (%s) returned invalid JSON: %v", i, op["type"], err)
			}
		})
	}
}

// TestToolCreateConnection tests the create_connection operation
func TestToolCreateConnection(t *testing.T) {
	setupTestDB(t)

	// Clean up test connection if it exists
	_, _ = masterDB.Exec("DELETE FROM mcp_connections WHERE name = 'test_conn'")

	params := map[string]interface{}{
		"name":     "test_conn",
		"host":     "localhost",
		"port":     5432.0,
		"database": "testdb",
		"user":     "testuser",
		"password": "testpass",
		"sslmode":  "disable",
	}

	result, err := toolCreateConnection(params)
	if err != nil {
		t.Fatalf("toolCreateConnection() error = %v", err)
	}

	// Parse result
	var conn ConnectionConfig
	if err := json.Unmarshal([]byte(result), &conn); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify connection was created
	if conn.Name != "test_conn" {
		t.Errorf("Expected name 'test_conn', got '%s'", conn.Name)
	}
	if conn.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", conn.Host)
	}
	if conn.Password != "" {
		t.Error("Password should be masked in response")
	}

	// Clean up
	_, _ = masterDB.Exec("DELETE FROM mcp_connections WHERE name = 'test_conn'")
}

// TestToolListConnections tests the list_connections operation
func TestToolListConnections(t *testing.T) {
	setupTestDB(t)

	result, err := toolListConnections(nil)
	if err != nil {
		t.Fatalf("toolListConnections() error = %v", err)
	}

	// Parse result
	var connections []ConnectionConfig
	if err := json.Unmarshal([]byte(result), &connections); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify we have at least the master connection
	if len(connections) == 0 {
		t.Error("Expected at least one connection (master), got none")
	}

	// Verify master connection exists
	foundMaster := false
	for _, conn := range connections {
		if conn.Name == "master" {
			foundMaster = true
			if conn.Password != "" {
				t.Error("Password should be masked in list response")
			}
		}
	}

	if !foundMaster {
		t.Error("Expected to find 'master' connection in list")
	}
}

// TestToolGetConnection tests the get_connection operation
func TestToolGetConnection(t *testing.T) {
	setupTestDB(t)

	params := map[string]interface{}{
		"name": "master",
	}

	result, err := toolGetConnection(params)
	if err != nil {
		t.Fatalf("toolGetConnection() error = %v", err)
	}

	// Parse result
	var conn ConnectionConfig
	if err := json.Unmarshal([]byte(result), &conn); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify connection details
	if conn.Name != "master" {
		t.Errorf("Expected name 'master', got '%s'", conn.Name)
	}
	if conn.Password != "" {
		t.Error("Password should be masked in response")
	}
}

// TestToolUpdateConnection tests the update_connection operation
func TestToolUpdateConnection(t *testing.T) {
	setupTestDB(t)

	// Create a test connection first
	testParams := map[string]interface{}{
		"name":     "test_update",
		"host":     "localhost",
		"port":     5432.0,
		"database": "testdb",
		"user":     "testuser",
		"password": "testpass",
	}
	_, _ = toolCreateConnection(testParams)
	defer func() {
		_, _ = masterDB.Exec("DELETE FROM mcp_connections WHERE name = 'test_update'")
	}()

	// Update the connection
	updateParams := map[string]interface{}{
		"name":        "test_update",
		"description": "Updated description",
		"sslmode":     "require",
	}

	result, err := toolUpdateConnection(updateParams)
	if err != nil {
		t.Fatalf("toolUpdateConnection() error = %v", err)
	}

	// Parse result
	var conn ConnectionConfig
	if err := json.Unmarshal([]byte(result), &conn); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify update
	if conn.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", conn.Description)
	}
	if conn.SSLMode != "require" {
		t.Errorf("Expected sslmode 'require', got '%s'", conn.SSLMode)
	}
	if conn.Password != "" {
		t.Error("Password should be masked in response")
	}
}

// TestToolDeleteConnection tests the delete_connection operation
func TestToolDeleteConnection(t *testing.T) {
	setupTestDB(t)

	// Create a test connection first
	testParams := map[string]interface{}{
		"name":     "test_delete",
		"host":     "localhost",
		"port":     5432.0,
		"database": "testdb",
		"user":     "testuser",
		"password": "testpass",
	}
	_, _ = toolCreateConnection(testParams)

	// Delete the connection
	deleteParams := map[string]interface{}{
		"name": "test_delete",
	}

	result, err := toolDeleteConnection(deleteParams)
	if err != nil {
		t.Fatalf("toolDeleteConnection() error = %v", err)
	}

	// Verify deletion
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if message, ok := response["message"].(string); !ok || !strings.Contains(message, "deleted successfully") {
		t.Errorf("Expected success message, got: %v", response)
	}

	// Verify connection no longer exists
	_, err = toolGetConnection(deleteParams)
	if err == nil {
		t.Error("Expected error when getting deleted connection, got none")
	}
}

// TestToolDeleteMasterConnection tests that master connection cannot be deleted
func TestToolDeleteMasterConnection(t *testing.T) {
	setupTestDB(t)

	deleteParams := map[string]interface{}{
		"name": "master",
	}

	_, err := toolDeleteConnection(deleteParams)
	if err == nil {
		t.Error("Expected error when trying to delete master connection, got none")
	}

	if !strings.Contains(err.Error(), "cannot delete") {
		t.Errorf("Expected error about not being able to delete master, got: %v", err)
	}
}

