package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	// Initialize database connection
	if err := initDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer closeDatabase()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		closeDatabase()
		os.Exit(0)
	}()

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





