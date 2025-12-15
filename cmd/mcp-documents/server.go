package main

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
	"github.com/mark3labs/mcp-go/server"
)

// NewMCPDocumentsServer initializes the database connection, creates the
// document server and wires it into an MCP server instance.
func NewMCPDocumentsServer() (*server.MCPServer, *sql.DB, error) {
	// Get database connection string from environment
	dbDSN := os.Getenv("DOCUMENTS_DB_DSN")
	if dbDSN == "" {
		return nil, nil, ErrMissingDSN
	}

	// Connect to database
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		return nil, nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, nil, err
	}

	// Create repository and document server
	repo := NewSQLDocumentRepository(db)
	documentServer := NewDocumentServer(repo)

	// Create MCP server
	s := server.NewMCPServer(
		"mcp-documents",
		"1.0.0",
		server.WithLogging(),
	)

	// Register tools
	registerDocumentTools(s, documentServer)

	return s, db, nil
}

// ErrMissingDSN is returned when the DOCUMENTS_DB_DSN environment variable is not set.
var ErrMissingDSN = &ConfigError{Message: "DOCUMENTS_DB_DSN environment variable is required"}

// ConfigError represents a configuration problem that should fail fast.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return e.Message
}


