package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
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
			fmt.Fprintf(os.Stderr, "[ERROR] mcp-git: Request size %d bytes exceeds 1MB buffer\n", len(line))
		} else if len(line) > 500_000 {
			fmt.Fprintf(os.Stderr, "[WARN] mcp-git: Large request size %d bytes\n", len(line))
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
		fmt.Fprintf(os.Stderr, "[ERROR] mcp-git: Scanner error: %v (buffer max: 10MB)\n", err)
	}
}
