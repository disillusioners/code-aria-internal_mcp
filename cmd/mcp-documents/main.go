package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	srv, db, err := NewMCPDocumentsServer()
	if err != nil {
		log.Fatalf("Failed to initialize MCP Documents server: %v", err)
	}
	defer db.Close()

	log.Printf("Starting MCP Documents server on stdio")
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
