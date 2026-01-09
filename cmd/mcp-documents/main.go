package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database connection string from environment
	dbDSN := os.Getenv("DOCUMENTS_DB_DSN")
	if dbDSN == "" {
		fmt.Fprintf(os.Stderr, "DOCUMENTS_DB_DSN environment variable is required\n")
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	// Create repository and set as global
	repo := NewSQLDocumentRepository(db)
	setGlobalRepository(repo)

	scanner := bufio.NewScanner(os.Stdin)
	// Increase buffer size to handle large JSON-RPC messages
	scanner.Buffer(nil, 10*1024*1024) // 10MB buffer
	encoder := json.NewEncoder(os.Stdout)

	if err := handleInitialize(scanner, encoder); err != nil {
		fmt.Fprintf(os.Stderr, "Initialize failed: %v\n", err)
		os.Exit(1)
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Log large requests for debugging (warn if > 500KB, error if > 1MB)
		if len(line) > 1_000_000 {
			fmt.Fprintf(os.Stderr, "[ERROR] mcp-documents: Request size %d bytes exceeds 1MB buffer\n", len(line))
		} else if len(line) > 500_000 {
			fmt.Fprintf(os.Stderr, "[WARN] mcp-documents: Large request size %d bytes\n", len(line))
		}

		var msg MCPMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Method != "" {
			handleRequest(&msg, encoder)
		}
	}

	// Check for scanner errors (e.g., token too long)
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] mcp-documents: Scanner error: %v (buffer max: 10MB)\n", err)
	}
}
