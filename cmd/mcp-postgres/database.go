package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
)

// getConnectionString returns the connection string, preferring per-call override over env var
func getConnectionString(params map[string]interface{}) (string, error) {
	// Check for per-call override first
	if connStr, ok := params["connection_string"].(string); ok && connStr != "" {
		return connStr, nil
	}

	// Fall back to environment variable
	connStr := os.Getenv("POSTGRES_DB_DSN")
	if connStr == "" {
		return "", fmt.Errorf("POSTGRES_DB_DSN environment variable is required, or provide connection_string parameter")
	}

	return connStr, nil
}

// maskPasswordInConnectionString masks the password in a PostgreSQL connection string
func maskPasswordInConnectionString(connStr string) string {
	// Pattern: postgres://user:password@host:port/dbname?params
	// We want to mask the password part
	re := regexp.MustCompile(`(postgres://[^:]+:)([^@]+)(@.+)`)
	masked := re.ReplaceAllString(connStr, "${1}****${3}")

	// If the pattern didn't match, try without password (postgres://user@host)
	// Or if it's already masked, return as is
	if masked == connStr {
		// Check if it's a connection string without password
		reNoPass := regexp.MustCompile(`postgres://([^:@]+)@`)
		if reNoPass.MatchString(connStr) {
			return connStr // No password to mask
		}
		// If it doesn't match standard format, return masked version
		return "****"
	}

	return masked
}

// toolGetConnectionInfo returns connection information with masked password
func toolGetConnectionInfo(params map[string]interface{}) (string, error) {
	connStr, err := getConnectionString(params)
	if err != nil {
		return "", err
	}

	// Parse connection string to extract components
	// Format: postgres://user:password@host:port/dbname?params
	re := regexp.MustCompile(`postgres://(?:([^:]+):([^@]+)@)?([^:/]+)(?::(\d+))?/([^?]+)?(?:\?(.+))?`)
	matches := re.FindStringSubmatch(connStr)

	info := map[string]interface{}{
		"connection_string_masked": maskPasswordInConnectionString(connStr),
		"source":                   "environment",
	}

	// If connection_string was provided in params, mark it as parameter override
	if _, ok := params["connection_string"].(string); ok {
		info["source"] = "parameter_override"
	}

	// Extract components if possible
	if len(matches) > 1 && matches[1] != "" {
		info["user"] = matches[1]
	}
	if len(matches) > 3 && matches[3] != "" {
		info["host"] = matches[3]
	}
	if len(matches) > 4 && matches[4] != "" {
		info["port"] = matches[4]
	}
	if len(matches) > 5 && matches[5] != "" {
		info["database"] = matches[5]
	}
	if len(matches) > 6 && matches[6] != "" {
		// Parse query parameters
		paramsStr := matches[6]
		paramsMap := make(map[string]string)
		for _, param := range strings.Split(paramsStr, "&") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 {
				paramsMap[parts[0]] = parts[1]
			}
		}
		if len(paramsMap) > 0 {
			info["parameters"] = paramsMap
		}
	}

	// Test connection if possible (without exposing password)
	info["connection_configured"] = true

	resultJSON, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// openDatabase opens a database connection with the given connection string
func openDatabase(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set timezone to UTC for consistent timestamp handling
	if _, err := db.Exec("SET timezone = 'UTC'"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set timezone to UTC: %w", err)
	}

	return db, nil
}

// validateSelectQuery ensures the query is a SELECT statement only
func validateSelectQuery(query string) error {
	// Remove comments and normalize whitespace
	query = regexp.MustCompile(`--.*`).ReplaceAllString(query, "")
	query = regexp.MustCompile(`/\*.*?\*/`).ReplaceAllString(query, "")
	query = strings.TrimSpace(query)

	// Check if query starts with SELECT (case-insensitive)
	selectRegex := regexp.MustCompile(`(?i)^\s*SELECT\s+`)
	if !selectRegex.MatchString(query) {
		return fmt.Errorf("only SELECT queries are allowed for security reasons")
	}

	// Check for dangerous keywords that could modify data
	dangerousKeywords := []string{
		"INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"TRUNCATE", "GRANT", "REVOKE", "EXEC", "EXECUTE",
	}

	upperQuery := strings.ToUpper(query)
	for _, keyword := range dangerousKeywords {
		// Use word boundaries to avoid false positives
		pattern := fmt.Sprintf(`\b%s\b`, keyword)
		matched, _ := regexp.MatchString(pattern, upperQuery)
		if matched {
			return fmt.Errorf("query contains forbidden keyword: %s (read-only access only)", keyword)
		}
	}

	return nil
}

// toolListSchemas lists all schemas in the database
func toolListSchemas(params map[string]interface{}) (string, error) {
	connStr, err := getConnectionString(params)
	if err != nil {
		return "", err
	}

	db, err := openDatabase(connStr)
	if err != nil {
		return "", err
	}
	defer db.Close()

	query := `
		SELECT schema_name 
		FROM information_schema.schemata 
		WHERE schema_name NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		ORDER BY schema_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schemaName string
		if err := rows.Scan(&schemaName); err != nil {
			return "", fmt.Errorf("failed to scan schema: %w", err)
		}
		schemas = append(schemas, schemaName)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating schemas: %w", err)
	}

	result, err := json.Marshal(schemas)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// toolListTables lists tables in a schema
func toolListTables(params map[string]interface{}) (string, error) {
	connStr, err := getConnectionString(params)
	if err != nil {
		return "", err
	}

	db, err := openDatabase(connStr)
	if err != nil {
		return "", err
	}
	defer db.Close()

	schema := "public"
	if s, ok := params["schema"].(string); ok && s != "" {
		schema = s
	}

	query := `
		SELECT 
			table_schema,
			table_name,
			table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		ORDER BY table_schema, table_name
	`

	rows, err := db.Query(query, schema)
	if err != nil {
		return "", fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	type TableInfo struct {
		Schema    string `json:"schema"`
		TableName string `json:"table_name"`
		TableType string `json:"table_type"`
	}

	var tables []TableInfo
	for rows.Next() {
		var info TableInfo
		if err := rows.Scan(&info.Schema, &info.TableName, &info.TableType); err != nil {
			return "", fmt.Errorf("failed to scan table: %w", err)
		}
		tables = append(tables, info)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating tables: %w", err)
	}

	result, err := json.Marshal(tables)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(result), nil
}

// toolDescribeTable gets detailed table schema information
func toolDescribeTable(params map[string]interface{}) (string, error) {
	connStr, err := getConnectionString(params)
	if err != nil {
		return "", err
	}

	db, err := openDatabase(connStr)
	if err != nil {
		return "", err
	}
	defer db.Close()

	tableName, ok := params["table_name"].(string)
	if !ok || tableName == "" {
		return "", fmt.Errorf("table_name is required")
	}

	schema := "public"
	if s, ok := params["schema"].(string); ok && s != "" {
		schema = s
	}

	// Get column information
	columnQuery := `
		SELECT 
			column_name,
			data_type,
			character_maximum_length,
			is_nullable,
			column_default,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`

	rows, err := db.Query(columnQuery, schema, tableName)
	if err != nil {
		return "", fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	type ColumnInfo struct {
		Name         string   `json:"name"`
		Type         string   `json:"type"`
		MaxLength    *int     `json:"max_length,omitempty"`
		Nullable     bool     `json:"nullable"`
		DefaultValue *string  `json:"default,omitempty"`
		Position     int      `json:"position"`
		Constraints  []string `json:"constraints,omitempty"`
		Indexes      []string `json:"indexes,omitempty"`
	}

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var maxLen sql.NullInt64
		var nullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &maxLen, &nullable, &defaultValue, &col.Position); err != nil {
			return "", fmt.Errorf("failed to scan column: %w", err)
		}

		if maxLen.Valid {
			ml := int(maxLen.Int64)
			col.MaxLength = &ml
		}
		col.Nullable = nullable == "YES"
		if defaultValue.Valid {
			def := defaultValue.String
			col.DefaultValue = &def
		}

		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating columns: %w", err)
	}

	// Get constraints
	constraintQuery := `
		SELECT 
			c.column_name,
			tc.constraint_type,
			tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage c 
			ON tc.constraint_name = c.constraint_name
			AND tc.table_schema = c.table_schema
		WHERE tc.table_schema = $1 AND tc.table_name = $2
	`

	constraintRows, err := db.Query(constraintQuery, schema, tableName)
	if err == nil {
		defer constraintRows.Close()

		constraintMap := make(map[string][]string)
		for constraintRows.Next() {
			var colName, constraintType, constraintName string
			if err := constraintRows.Scan(&colName, &constraintType, &constraintName); err == nil {
				constraintMap[colName] = append(constraintMap[colName], fmt.Sprintf("%s (%s)", constraintType, constraintName))
			}
		}

		// Add constraints to columns
		for i := range columns {
			if constraints, ok := constraintMap[columns[i].Name]; ok {
				columns[i].Constraints = constraints
			}
		}
	}

	// Get indexes
	indexQuery := `
		SELECT 
			i.relname AS index_name,
			a.attname AS column_name
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = $1 AND t.relname = $2
		ORDER BY i.relname, a.attname
	`

	indexRows, err := db.Query(indexQuery, schema, tableName)
	if err == nil {
		defer indexRows.Close()

		indexMap := make(map[string][]string)
		for indexRows.Next() {
			var indexName, colName string
			if err := indexRows.Scan(&indexName, &colName); err == nil {
				indexMap[colName] = append(indexMap[colName], indexName)
			}
		}

		// Add indexes to columns
		for i := range columns {
			if indexes, ok := indexMap[columns[i].Name]; ok {
				columns[i].Indexes = indexes
			}
		}
	}

	type TableSchema struct {
		Schema  string       `json:"schema"`
		Table   string       `json:"table"`
		Columns []ColumnInfo `json:"columns"`
	}

	result := TableSchema{
		Schema:  schema,
		Table:   tableName,
		Columns: columns,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolQuery executes a SELECT query
func toolQuery(params map[string]interface{}) (string, error) {
	connStr, err := getConnectionString(params)
	if err != nil {
		return "", err
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query parameter is required")
	}

	// Validate that it's a SELECT query only
	if err := validateSelectQuery(query); err != nil {
		return "", err
	}

	db, err := openDatabase(connStr)
	if err != nil {
		return "", err
	}
	defer db.Close()

	// Apply limit if specified
	limit := 1000
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
		if limit < 1 {
			limit = 1
		}
		if limit > 10000 {
			limit = 10000
		}
	}

	// Add LIMIT clause if not present
	upperQuery := strings.ToUpper(strings.TrimSpace(query))
	if !strings.Contains(upperQuery, "LIMIT") {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
	}

	// Handle parameterized queries
	var rows *sql.Rows
	if paramsArray, ok := params["params"].([]interface{}); ok && len(paramsArray) > 0 {
		// Convert params to []interface{} for variadic args
		args := make([]interface{}, len(paramsArray))
		copy(args, paramsArray)
		rows, err = db.Query(query, args...)
	} else {
		rows, err = db.Query(query)
	}

	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns: %w", err)
	}

	// Scan results
	var results []map[string]interface{}
	for rows.Next() {
		// Create slice of pointers for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return "", fmt.Errorf("failed to scan row: %w", err)
		}

		// Build map from column names to values
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Handle special types
			if b, ok := val.([]byte); ok {
				// Try to parse as JSON, otherwise use as string
				var jsonVal interface{}
				if err := json.Unmarshal(b, &jsonVal); err == nil {
					val = jsonVal
				} else {
					val = string(b)
				}
			}
			row[col] = val
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating rows: %w", err)
	}

	resultJSON, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
