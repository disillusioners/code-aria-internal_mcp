package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

var (
	masterDB *sql.DB
	dbType   string // "postgres" or "sqlite"
)

// Helper function to get the appropriate timestamp function based on database type
func getTimestampExpr() string {
	if dbType == "sqlite" {
		return "datetime('now')"
	}
	return "NOW()"
}

// Helper function to get the appropriate parameter placeholder based on database type
func getParam(index int) string {
	if dbType == "sqlite" {
		return "?"
	}
	return fmt.Sprintf("$%d", index)
}

// ensureMCPConnectionsTable creates the mcp_connections table if it doesn't exist
func ensureMCPConnectionsTable() error {
	if masterDB == nil {
		return fmt.Errorf("master database connection not initialized")
	}

	var createTableSQL string

	if dbType == "sqlite" {
		// SQLite-compatible schema
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS mcp_connections (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT UNIQUE NOT NULL,
				host TEXT NOT NULL,
				port INTEGER NOT NULL DEFAULT 5432,
				database TEXT NOT NULL,
				user_name TEXT NOT NULL,
				password TEXT NOT NULL,
				sslmode TEXT DEFAULT 'disable',
				description TEXT,
				created_at TEXT DEFAULT CURRENT_TIMESTAMP,
				updated_at TEXT DEFAULT CURRENT_TIMESTAMP
			)
		`
	} else {
		// PostgreSQL schema (existing)
		createTableSQL = `
			CREATE TABLE IF NOT EXISTS mcp_connections (
				id SERIAL PRIMARY KEY,
				name VARCHAR(255) UNIQUE NOT NULL,
				host VARCHAR(255) NOT NULL,
				port INTEGER NOT NULL DEFAULT 5432,
				database VARCHAR(255) NOT NULL,
				user_name VARCHAR(255) NOT NULL,
				password VARCHAR(255) NOT NULL,
				sslmode VARCHAR(50) DEFAULT 'disable',
				description TEXT,
				created_at TIMESTAMP DEFAULT NOW(),
				updated_at TIMESTAMP DEFAULT NOW()
			)
		`
	}

	_, err := masterDB.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create mcp_connections table: %w", err)
	}

	return nil
}

// initMasterConnection initializes the master connection in the mcp_connections table
func initMasterConnection() error {
	if masterDB == nil {
		return fmt.Errorf("master database connection not initialized")
	}

	// Skip for SQLite - no master connection to initialize
	if dbType == "sqlite" {
		return nil
	}

	// Get master connection string from environment
	masterDSN := os.Getenv("POSTGRES_DB_DSN")
	if masterDSN == "" {
		return fmt.Errorf("POSTGRES_DB_DSN environment variable is required")
	}

	// Parse connection string to extract components
	parsedURL, err := url.Parse(masterDSN)
	if err != nil {
		return fmt.Errorf("failed to parse POSTGRES_DB_DSN: %w", err)
	}

	host := parsedURL.Hostname()
	portStr := parsedURL.Port()
	port := 5432
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port in POSTGRES_DB_DSN: %w", err)
		}
	}

	database := strings.TrimPrefix(parsedURL.Path, "/")
	user := parsedURL.User.Username()
	password, _ := parsedURL.User.Password()

	// Get sslmode from query parameters
	sslmode := "disable"
	if parsedURL.Query().Get("sslmode") != "" {
		sslmode = parsedURL.Query().Get("sslmode")
	}

	// Check if master connection already exists
	var exists bool
	err = masterDB.QueryRow("SELECT EXISTS(SELECT 1 FROM mcp_connections WHERE name = 'master')").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for existing master connection: %w", err)
	}

	if exists {
		// Update existing master connection
		var updateSQL string
		if dbType == "sqlite" {
			updateSQL = `
				UPDATE mcp_connections
				SET host = ?, port = ?, database = ?, user_name = ?, password = ?, sslmode = ?, updated_at = datetime('now')
				WHERE name = 'master'
			`
		} else {
			updateSQL = `
				UPDATE mcp_connections
				SET host = $1, port = $2, database = $3, user_name = $4, password = $5, sslmode = $6, updated_at = NOW()
				WHERE name = 'master'
			`
		}

		if dbType == "sqlite" {
			_, err = masterDB.Exec(updateSQL, host, port, database, user, password, sslmode)
		} else {
			_, err = masterDB.Exec(updateSQL, host, port, database, user, password, sslmode)
		}
		if err != nil {
			return fmt.Errorf("failed to update master connection: %w", err)
		}
	} else {
		// Insert new master connection
		var insertSQL string
		if dbType == "sqlite" {
			insertSQL = `
				INSERT INTO mcp_connections (name, host, port, database, user_name, password, sslmode, description)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`
		} else {
			insertSQL = `
				INSERT INTO mcp_connections (name, host, port, database, user_name, password, sslmode, description)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`
		}

		if dbType == "sqlite" {
			_, err = masterDB.Exec(insertSQL, "master", host, port, database, user, password, sslmode, "Master connection from POSTGRES_DB_DSN")
		} else {
			_, err = masterDB.Exec(insertSQL, "master", host, port, database, user, password, sslmode, "Master connection from POSTGRES_DB_DSN")
		}
		if err != nil {
			return fmt.Errorf("failed to insert master connection: %w", err)
		}
	}

	return nil
}

// getConnectionByName retrieves a connection configuration by name
func getConnectionByName(name string) (*ConnectionConfig, error) {
	if masterDB == nil {
		return nil, fmt.Errorf("master database connection not initialized")
	}

	var config ConnectionConfig
	var createdAt, updatedAt time.Time

	err := masterDB.QueryRow(`
		SELECT id, name, host, port, database, user_name, password, sslmode, description, created_at, updated_at
		FROM mcp_connections
		WHERE name = $1
	`, name).Scan(
		&config.ID, &config.Name, &config.Host, &config.Port, &config.Database,
		&config.User, &config.Password, &config.SSLMode, &config.Description,
		&createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("connection '%s' not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query connection: %w", err)
	}

	config.CreatedAt = createdAt
	config.UpdatedAt = updatedAt

	return &config, nil
}

// buildConnectionString builds a PostgreSQL connection string from a ConnectionConfig
func buildConnectionString(config *ConnectionConfig) string {
	// Build connection string: postgres://user:password@host:port/database?sslmode=...
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		url.QueryEscape(config.User),
		url.QueryEscape(config.Password),
		config.Host,
		config.Port,
		url.QueryEscape(config.Database),
	)

	if config.SSLMode != "" {
		connStr += fmt.Sprintf("?sslmode=%s", url.QueryEscape(config.SSLMode))
	}

	return connStr
}

// maskPassword masks a password string for display
func maskPassword(password string) string {
	if password == "" {
		return ""
	}
	return "****"
}

// getConnectionStringByName gets a connection string by connection name
func getConnectionStringByName(connectionName string) (string, error) {
	config, err := getConnectionByName(connectionName)
	if err != nil {
		return "", err
	}
	return buildConnectionString(config), nil
}

// getConnectionString returns the connection string by connection name (defaults to "master" if not provided)
func getConnectionString(params map[string]interface{}) (string, error) {
	connectionName, ok := params["connection_name"].(string)
	if !ok || connectionName == "" {
		// In SQLite mode, there is no master connection
		if dbType == "sqlite" {
			return "", fmt.Errorf("connection_name is required when using SQLite mode (no default 'master' connection exists)")
		}
		connectionName = "master" // Default to master connection for PostgreSQL
	}

	return getConnectionStringByName(connectionName)
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
	connectionName, ok := params["connection_name"].(string)
	if !ok || connectionName == "" {
		// In SQLite mode, there is no master connection
		if dbType == "sqlite" {
			return "", fmt.Errorf("connection_name is required when using SQLite mode (no default 'master' connection exists)")
		}
		connectionName = "master" // Default to master connection for PostgreSQL
	}

	config, err := getConnectionByName(connectionName)
	if err != nil {
		return "", err
	}

	// Build masked connection string
	connStr := buildConnectionString(config)
	maskedConnStr := maskPasswordInConnectionString(connStr)

	info := map[string]interface{}{
		"name":                      config.Name,
		"connection_string_masked":  maskedConnStr,
		"host":                      config.Host,
		"port":                      config.Port,
		"database":                  config.Database,
		"user":                      config.User,
		"sslmode":                   config.SSLMode,
		"description":               config.Description,
		"created_at":                config.CreatedAt.Format(time.RFC3339),
		"updated_at":                config.UpdatedAt.Format(time.RFC3339),
		"connection_configured":     true,
	}

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

// toolCreateConnection creates a new connection configuration
func toolCreateConnection(params map[string]interface{}) (string, error) {
	if masterDB == nil {
		return "", fmt.Errorf("master database connection not initialized")
	}

	// Extract required parameters
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}

	host, ok := params["host"].(string)
	if !ok || host == "" {
		return "", fmt.Errorf("host parameter is required")
	}

	database, ok := params["database"].(string)
	if !ok || database == "" {
		return "", fmt.Errorf("database parameter is required")
	}

	user, ok := params["user"].(string)
	if !ok || user == "" {
		return "", fmt.Errorf("user parameter is required")
	}

	password, ok := params["password"].(string)
	if !ok || password == "" {
		return "", fmt.Errorf("password parameter is required")
	}

	// Extract optional parameters
	port := 5432
	if p, ok := params["port"].(float64); ok {
		port = int(p)
	}

	sslmode := "disable"
	if s, ok := params["sslmode"].(string); ok && s != "" {
		sslmode = s
	}

	description := ""
	if d, ok := params["description"].(string); ok {
		description = d
	}

	// Insert new connection
	var id int
	var createdAt, updatedAt time.Time
	err := masterDB.QueryRow(`
		INSERT INTO mcp_connections (name, host, port, database, user_name, password, sslmode, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, name, host, port, database, user, password, sslmode, description).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		// Check if it's a unique constraint violation
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return "", fmt.Errorf("connection with name '%s' already exists", name)
		}
		return "", fmt.Errorf("failed to create connection: %w", err)
	}

	// Return created connection (password masked)
	result := ConnectionConfig{
		ID:          id,
		Name:        name,
		Host:        host,
		Port:        port,
		Database:    database,
		User:        user,
		SSLMode:     sslmode,
		Description: description,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolListConnections lists all connections (passwords masked)
func toolListConnections(params map[string]interface{}) (string, error) {
	if masterDB == nil {
		return "", fmt.Errorf("master database connection not initialized")
	}

	rows, err := masterDB.Query(`
		SELECT id, name, host, port, database, user_name, sslmode, description, created_at, updated_at
		FROM mcp_connections
		ORDER BY name
	`)
	if err != nil {
		return "", fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []ConnectionConfig
	for rows.Next() {
		var conn ConnectionConfig
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&conn.ID, &conn.Name, &conn.Host, &conn.Port, &conn.Database,
			&conn.User, &conn.SSLMode, &conn.Description, &createdAt, &updatedAt,
		)
		if err != nil {
			return "", fmt.Errorf("failed to scan connection: %w", err)
		}

		conn.CreatedAt = createdAt
		conn.UpdatedAt = updatedAt
		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("error iterating connections: %w", err)
	}

	resultJSON, err := json.Marshal(connections)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetConnection gets a connection by name (password masked)
func toolGetConnection(params map[string]interface{}) (string, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}

	config, err := getConnectionByName(name)
	if err != nil {
		return "", err
	}

	// Create a copy without password for response
	result := ConnectionConfig{
		ID:          config.ID,
		Name:        config.Name,
		Host:        config.Host,
		Port:        config.Port,
		Database:    config.Database,
		User:        config.User,
		SSLMode:     config.SSLMode,
		Description: config.Description,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolUpdateConnection updates a connection configuration
func toolUpdateConnection(params map[string]interface{}) (string, error) {
	if masterDB == nil {
		return "", fmt.Errorf("master database connection not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}

	// Check if connection exists
	existing, err := getConnectionByName(name)
	if err != nil {
		return "", err
	}

	// Build update query dynamically based on provided parameters
	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if host, ok := params["host"].(string); ok && host != "" {
		updates = append(updates, fmt.Sprintf("host = %s", getParam(argIndex)))
		args = append(args, host)
		argIndex++
		existing.Host = host
	}

	if port, ok := params["port"].(float64); ok {
		updates = append(updates, fmt.Sprintf("port = %s", getParam(argIndex)))
		args = append(args, int(port))
		argIndex++
		existing.Port = int(port)
	}

	if database, ok := params["database"].(string); ok && database != "" {
		updates = append(updates, fmt.Sprintf("database = %s", getParam(argIndex)))
		args = append(args, database)
		argIndex++
		existing.Database = database
	}

	if user, ok := params["user"].(string); ok && user != "" {
		updates = append(updates, fmt.Sprintf("user_name = %s", getParam(argIndex)))
		args = append(args, user)
		argIndex++
		existing.User = user
	}

	if password, ok := params["password"].(string); ok && password != "" {
		updates = append(updates, fmt.Sprintf("password = %s", getParam(argIndex)))
		args = append(args, password)
		argIndex++
	}

	if sslmode, ok := params["sslmode"].(string); ok {
		updates = append(updates, fmt.Sprintf("sslmode = %s", getParam(argIndex)))
		args = append(args, sslmode)
		argIndex++
		existing.SSLMode = sslmode
	}

	if description, ok := params["description"].(string); ok {
		updates = append(updates, fmt.Sprintf("description = %s", getParam(argIndex)))
		args = append(args, description)
		argIndex++
		existing.Description = description
	}

	if len(updates) == 0 {
		// No updates provided, return existing connection
		result := ConnectionConfig{
			ID:          existing.ID,
			Name:        existing.Name,
			Host:        existing.Host,
			Port:        existing.Port,
			Database:    existing.Database,
			User:        existing.User,
			SSLMode:     existing.SSLMode,
			Description: existing.Description,
			CreatedAt:   existing.CreatedAt,
			UpdatedAt:   existing.UpdatedAt,
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("failed to marshal result: %w", err)
		}
		return string(resultJSON), nil
	}

	// Add updated_at and name for WHERE clause
	updates = append(updates, fmt.Sprintf("updated_at = %s", getTimestampExpr()))
	args = append(args, name)
	argIndex++

	query := fmt.Sprintf(`
		UPDATE mcp_connections
		SET %s
		WHERE name = %s
		RETURNING updated_at
	`, strings.Join(updates, ", "), getParam(argIndex))

	var updatedAt time.Time
	err = masterDB.QueryRow(query, args...).Scan(&updatedAt)
	if err != nil {
		return "", fmt.Errorf("failed to update connection: %w", err)
	}

	existing.UpdatedAt = updatedAt

	// Return updated connection (password masked)
	result := ConnectionConfig{
		ID:          existing.ID,
		Name:        existing.Name,
		Host:        existing.Host,
		Port:        existing.Port,
		Database:    existing.Database,
		User:        existing.User,
		SSLMode:     existing.SSLMode,
		Description: existing.Description,
		CreatedAt:   existing.CreatedAt,
		UpdatedAt:   existing.UpdatedAt,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolDeleteConnection deletes a connection by name
func toolDeleteConnection(params map[string]interface{}) (string, error) {
	if masterDB == nil {
		return "", fmt.Errorf("master database connection not initialized")
	}

	name, ok := params["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name parameter is required")
	}

	// Prevent deletion of master connection
	if name == "master" {
		return "", fmt.Errorf("cannot delete the master connection")
	}

	result, err := masterDB.Exec("DELETE FROM mcp_connections WHERE name = $1", name)
	if err != nil {
		return "", fmt.Errorf("failed to delete connection: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return "", fmt.Errorf("connection '%s' not found", name)
	}

	response := map[string]interface{}{
		"message":       fmt.Sprintf("Connection '%s' deleted successfully", name),
		"name":          name,
		"rows_affected": rowsAffected,
	}

	resultJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolRenameConnection renames a connection
func toolRenameConnection(params map[string]interface{}) (string, error) {
	if masterDB == nil {
		return "", fmt.Errorf("master database connection not initialized")
	}

	oldName, ok := params["old_name"].(string)
	if !ok || oldName == "" {
		return "", fmt.Errorf("old_name parameter is required")
	}

	newName, ok := params["new_name"].(string)
	if !ok || newName == "" {
		return "", fmt.Errorf("new_name parameter is required")
	}

	// Prevent renaming master connection
	if oldName == "master" {
		return "", fmt.Errorf("cannot rename the master connection")
	}

	// Check if old connection exists
	_, err := getConnectionByName(oldName)
	if err != nil {
		return "", fmt.Errorf("connection '%s' not found: %w", oldName, err)
	}

	// Check if new name already exists
	var exists bool
	err = masterDB.QueryRow("SELECT EXISTS(SELECT 1 FROM mcp_connections WHERE name = $1)", newName).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("failed to check for existing connection: %w", err)
	}
	if exists {
		return "", fmt.Errorf("connection with name '%s' already exists", newName)
	}

	// Update the connection name
	_, err = masterDB.Exec(fmt.Sprintf(`
		UPDATE mcp_connections
		SET name = %s, updated_at = %s
		WHERE name = %s
	`, getParam(1), getTimestampExpr(), getParam(2)), newName, oldName)
	if err != nil {
		return "", fmt.Errorf("failed to rename connection: %w", err)
	}

	// Get the renamed connection
	config, err := getConnectionByName(newName)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve renamed connection: %w", err)
	}

	// Return renamed connection (password masked)
	result := ConnectionConfig{
		ID:          config.ID,
		Name:        config.Name,
		Host:        config.Host,
		Port:        config.Port,
		Database:    config.Database,
		User:        config.User,
		SSLMode:     config.SSLMode,
		Description: config.Description,
		CreatedAt:   config.CreatedAt,
		UpdatedAt:   config.UpdatedAt,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
