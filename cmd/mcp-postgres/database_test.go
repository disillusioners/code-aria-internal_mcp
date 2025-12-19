package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"
)

// getTestConnectionString returns the connection string from environment or skips the test
func getTestConnectionString(t *testing.T) string {
	connStr := os.Getenv("POSTGRES_DB_DSN")
	if connStr == "" {
		t.Skip("POSTGRES_DB_DSN environment variable not set, skipping database tests")
	}
	return connStr
}

// TestToolListSchemas tests the list_schemas operation
func TestToolListSchemas(t *testing.T) {
	connStr := getTestConnectionString(t)

	params := map[string]interface{}{
		"connection_string": connStr,
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
	connStr := getTestConnectionString(t)

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "List tables in public schema",
			params: map[string]interface{}{
				"connection_string": connStr,
				"schema":            "public",
			},
		},
		{
			name: "List tables with default schema",
			params: map[string]interface{}{
				"connection_string": connStr,
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
	connStr := getTestConnectionString(t)

	// First, get a list of tables to test with
	params := map[string]interface{}{
		"connection_string": connStr,
		"schema":            "public",
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
		"connection_string": connStr,
		"table_name":       tableName,
		"schema":           schema,
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
	connStr := getTestConnectionString(t)

	tests := []struct {
		name   string
		params map[string]interface{}
		valid  bool
	}{
		{
			name: "Simple SELECT query",
			params: map[string]interface{}{
				"connection_string": connStr,
				"query":             "SELECT 1 as test_value",
				"limit":              10.0,
			},
			valid: true,
		},
		{
			name: "SELECT query with parameters",
			params: map[string]interface{}{
				"connection_string": connStr,
				"query":             "SELECT $1::text as value",
				"params":            []interface{}{"test"},
				"limit":              10.0,
			},
			valid: true,
		},
		{
			name: "SELECT query with multiple parameters",
			params: map[string]interface{}{
				"connection_string": connStr,
				"query":             "SELECT $1::int as num, $2::text as text",
				"params":            []interface{}{1, "test"},
				"limit":              10.0,
			},
			valid: true,
		},
		{
			name: "Query with limit",
			params: map[string]interface{}{
				"connection_string": connStr,
				"query":             "SELECT generate_series(1, 100) as num",
				"limit":              5.0,
			},
			valid: true,
		},
		{
			name: "Missing query parameter",
			params: map[string]interface{}{
				"connection_string": connStr,
			},
			valid: false,
		},
		{
			name: "Non-SELECT query (should fail)",
			params: map[string]interface{}{
				"connection_string": connStr,
				"query":             "INSERT INTO test VALUES (1)",
			},
			valid: false,
		},
		{
			name: "Query with dangerous keyword (should fail)",
			params: map[string]interface{}{
				"connection_string": connStr,
				"query":             "SELECT * FROM test; DROP TABLE test;",
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
	connStr := getTestConnectionString(t)

	// First, try to find a table
	params := map[string]interface{}{
		"connection_string": connStr,
		"schema":            "public",
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
		"connection_string": connStr,
		"query":             "SELECT * FROM " + schema + "." + tableName + " LIMIT 5",
		"limit":              5.0,
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

// TestGetConnectionString tests connection string retrieval
func TestGetConnectionString(t *testing.T) {
	// Save original env var
	originalDSN := os.Getenv("POSTGRES_DB_DSN")
	defer os.Setenv("POSTGRES_DB_DSN", originalDSN)

	tests := []struct {
		name           string
		envVar         string
		params         map[string]interface{}
		wantErr        bool
		expectedSource string
	}{
		{
			name:   "From environment variable",
			envVar: "postgres://test@localhost/test",
			params: map[string]interface{}{},
			wantErr: false,
		},
		{
			name:   "From parameter override",
			envVar: "postgres://env@localhost/env",
			params: map[string]interface{}{
				"connection_string": "postgres://param@localhost/param",
			},
			wantErr: false,
		},
		{
			name:    "Missing both",
			envVar:  "",
			params:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name:   "Empty parameter string",
			envVar:  "postgres://test@localhost/test",
			params: map[string]interface{}{
				"connection_string": "",
			},
			wantErr: false, // Should fall back to env var
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envVar != "" {
				os.Setenv("POSTGRES_DB_DSN", tt.envVar)
			} else {
				os.Unsetenv("POSTGRES_DB_DSN")
			}

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

				// Verify parameter override takes precedence
				if paramConn, ok := tt.params["connection_string"].(string); ok && paramConn != "" {
					if connStr != paramConn {
						t.Errorf("Expected connection string from parameter, got %s", connStr)
					}
				} else {
					if connStr != tt.envVar {
						t.Errorf("Expected connection string from env var, got %s", connStr)
					}
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
	connStr := getTestConnectionString(t)

	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "Get connection info from environment",
			params: map[string]interface{}{
				"connection_string": connStr,
			},
		},
		{
			name: "Get connection info with parameter override",
			params: map[string]interface{}{
				"connection_string": "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
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

			if strings.Contains(maskedStr, "testpass") {
				t.Error("Password should be masked in connection string")
			}

			if !strings.Contains(maskedStr, "****") {
				t.Logf("Warning: Connection string doesn't contain masked password pattern. Got: %s", maskedStr)
			}

			// Verify source
			source, ok := info["source"].(string)
			if !ok {
				t.Fatal("source is not a string")
			}

			if tt.params["connection_string"] != nil {
				if source != "parameter_override" {
					t.Errorf("Expected source 'parameter_override', got '%s'", source)
				}
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
	connStr := getTestConnectionString(t)

	// Test that we can run multiple operations
	operations := []map[string]interface{}{
		{
			"type":             "list_schemas",
			"connection_string": connStr,
		},
		{
			"type":             "list_tables",
			"schema":           "public",
			"connection_string": connStr,
		},
		{
			"type":             "query",
			"query":            "SELECT version() as pg_version",
			"connection_string": connStr,
			"limit":             1.0,
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

