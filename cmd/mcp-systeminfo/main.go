package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Initialize audit logger
	if err := InitAuditLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize audit logger: %v\n", err)
	}

	// Ensure cleanup on exit
	defer CloseAuditLogger()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		CloseAuditLogger()
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