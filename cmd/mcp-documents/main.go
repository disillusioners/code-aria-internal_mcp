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

		var msg MCPMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Method != "" {
			handleRequest(&msg, encoder)
		}
	}
}
