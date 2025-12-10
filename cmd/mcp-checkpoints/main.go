package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	log.Println("MCP Checkpoints Server starting...")

	// Initialize checkpoints directory
	if err := initCheckpointDir(); err != nil {
		log.Fatalf("Failed to initialize checkpoint directory: %v", err)
	}

	// Setup MCP communication
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// Handle initialize request
	if err := handleInitialize(scanner, encoder); err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}

	// Send initialized notification
	initializedNotif := MCPMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	if err := encoder.Encode(initializedNotif); err != nil {
		log.Fatalf("Failed to send initialized notification: %v", err)
	}

	log.Println("MCP Checkpoints Server initialized")

	// Handle requests
	for scanner.Scan() {
		var msg MCPMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		handleRequest(&msg, encoder)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Scanner error: %v", err)
	}
}

// initCheckpointDir creates the checkpoint directory if it doesn't exist
func initCheckpointDir() error {
	repoPath := os.Getenv("REPO_PATH")
	if repoPath == "" {
		return fmt.Errorf("REPO_PATH environment variable not set")
	}

	checkpointDir := fmt.Sprintf("%s/.mcp-checkpoints", repoPath)
	return os.MkdirAll(checkpointDir, 0755)
}