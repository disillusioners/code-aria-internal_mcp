package main

import (
	"bufio"
	"encoding/json"
	"fmt"
)

// handleInitialize processes the MCP initialize request
func handleInitialize(scanner *bufio.Scanner, encoder *json.Encoder) error {
	if !scanner.Scan() {
		return fmt.Errorf("no initialize request")
	}

	var initReq MCPMessage
	if err := json.Unmarshal(scanner.Bytes(), &initReq); err != nil {
		return fmt.Errorf("failed to parse initialize: %w", err)
	}

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      initReq.ID,
		Result: InitializeResponse{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    "mcp-checkpoints",
				Version: "1.0.0",
			},
		},
	}

	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to send initialize response: %w", err)
	}

	if !scanner.Scan() {
		return fmt.Errorf("no initialized notification")
	}

	return nil
}

// handleRequest routes MCP requests to appropriate handlers
func handleRequest(msg *MCPMessage, encoder *json.Encoder) {
	switch msg.Method {
	case "tools/list":
		handleToolsList(msg, encoder)
	case "tools/call":
		handleToolCall(msg, encoder)
	default:
		sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown method: %s", msg.Method), nil)
	}
}

// handleToolsList returns the list of available tools
func handleToolsList(msg *MCPMessage, encoder *json.Encoder) {
	tools := []Tool{
		{
			Name:        "create_checkpoint",
			Description: "Create a checkpoint of current working directory changes",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Name for the checkpoint",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Optional description of what this checkpoint contains",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "list_checkpoints",
			Description: "List all available checkpoints",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "get_checkpoint",
			Description: "Get details of a specific checkpoint",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"checkpoint_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the checkpoint to retrieve",
					},
				},
				"required": []string{"checkpoint_id"},
			},
		},
		{
			Name:        "restore_checkpoint",
			Description: "Restore a checkpoint to the working directory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"checkpoint_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the checkpoint to restore",
					},
				},
				"required": []string{"checkpoint_id"},
			},
		},
		{
			Name:        "delete_checkpoint",
			Description: "Delete a checkpoint",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"checkpoint_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the checkpoint to delete",
					},
				},
				"required": []string{"checkpoint_id"},
			},
		},
		{
			Name:        "get_checkpoint_info",
			Description: "Get detailed information about a checkpoint including file list",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"checkpoint_id": map[string]interface{}{
						"type":        "string",
						"description": "ID of the checkpoint to get info for",
					},
				},
				"required": []string{"checkpoint_id"},
			},
		},
		{
			Name:        "apply_operations",
			Description: "Execute multiple checkpoint operations in a single batch call",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operations": map[string]interface{}{
						"type":        "array",
						"description": "List of operations to execute",
						"items": map[string]interface{}{
							"type":        "object",
							"description": "Operation object with 'type' field and operation-specific parameters",
							"properties": map[string]interface{}{
								"type": map[string]interface{}{
									"type":        "string",
									"description": "Operation type: create_checkpoint, list_checkpoints, get_checkpoint, restore_checkpoint, delete_checkpoint, get_checkpoint_info",
								},
							},
						},
					},
				},
				"required": []string{"operations"},
			},
		},
	}

	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ToolsListResponse{
			Tools: tools,
		},
	}

	encoder.Encode(response)
}

// handleToolCall processes tool call requests
func handleToolCall(msg *MCPMessage, encoder *json.Encoder) {
	var req ToolsCallRequest
	reqJSON, err := json.Marshal(msg.Params)
	if err != nil {
		sendError(encoder, msg.ID, -32602, fmt.Sprintf("failed to marshal params: %v", err), nil)
		return
	}
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		sendError(encoder, msg.ID, -32602, fmt.Sprintf("failed to unmarshal params: %v", err), nil)
		return
	}

	if req.Name == "apply_operations" {
		handleBatchOperations(msg, encoder, req.Arguments)
		return
	}

	// Route to appropriate tool handler
	var result string
	switch req.Name {
	case "create_checkpoint":
		result, err = toolCreateCheckpoint(req.Arguments)
	case "list_checkpoints":
		result, err = toolListCheckpoints(req.Arguments)
	case "get_checkpoint":
		result, err = toolGetCheckpoint(req.Arguments)
	case "restore_checkpoint":
		result, err = toolRestoreCheckpoint(req.Arguments)
	case "delete_checkpoint":
		result, err = toolDeleteCheckpoint(req.Arguments)
	case "get_checkpoint_info":
		result, err = toolGetCheckpointInfo(req.Arguments)
	default:
		err = fmt.Errorf("unknown tool: %s", req.Name)
	}

	if err != nil {
		sendError(encoder, msg.ID, -32603, fmt.Sprintf("Tool execution failed: %v", err), nil)
		return
	}

	// Send successful response
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: ToolsCallResponse{
			Content: []Content{
				{
					Type: "text",
					Text: result,
				},
			},
		},
	}

	encoder.Encode(response)
}

func handleBatchOperations(msg *MCPMessage, encoder *json.Encoder, args map[string]interface{}) {
	operations, ok := args["operations"].([]interface{})
	if !ok {
		sendError(encoder, msg.ID, -32602, "operations array is required", nil)
		return
	}

	if len(operations) == 0 {
		sendError(encoder, msg.ID, -32602, "operations array cannot be empty", nil)
		return
	}

	var results []map[string]interface{}

	for _, op := range operations {
		opMap, ok := op.(map[string]interface{})
		if !ok {
			results = append(results, map[string]interface{}{
				"operation": "unknown",
				"params":    map[string]interface{}{},
				"status":    "Error",
				"message":   "Invalid operation format",
			})
			continue
		}

		opType, ok := opMap["type"].(string)
		if !ok {
			results = append(results, map[string]interface{}{
				"operation": "unknown",
				"params":    map[string]interface{}{},
				"status":    "Error",
				"message":   "Operation type is required",
			})
			continue
		}

		// Extract operation-specific arguments as params
		params := make(map[string]interface{})
		for k, v := range opMap {
			if k != "type" {
				params[k] = v
			}
		}

		// Execute operation based on type
		var result string
		var err error

		switch opType {
		case "create_checkpoint":
			result, err = toolCreateCheckpoint(params)
		case "list_checkpoints":
			result, err = toolListCheckpoints(params)
		case "get_checkpoint":
			result, err = toolGetCheckpoint(params)
		case "restore_checkpoint":
			result, err = toolRestoreCheckpoint(params)
		case "delete_checkpoint":
			result, err = toolDeleteCheckpoint(params)
		case "get_checkpoint_info":
			result, err = toolGetCheckpointInfo(params)
		default:
			err = fmt.Errorf("unknown operation type: %s", opType)
		}

		if err != nil {
			results = append(results, map[string]interface{}{
				"operation": opType,
				"params":    params,
				"status":    "Error",
				"message":   err.Error(),
			})
		} else {
			// Parse JSON result if possible, otherwise use as string
			var parsedResult interface{}
			if jsonErr := json.Unmarshal([]byte(result), &parsedResult); jsonErr == nil {
				// Successfully parsed JSON, use parsed result
			} else {
				// Not JSON, use string as-is
				parsedResult = result
			}

			results = append(results, map[string]interface{}{
				"operation": opType,
				"params":    params,
				"status":    "Success",
				"result":    parsedResult,
			})
		}
	}

	// Return results in format expected by client: {"results": [...]}
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"results": results,
		},
	}

	encoder.Encode(response)
}

// sendError sends an error response
func sendError(encoder *json.Encoder, id interface{}, code int, message string, data interface{}) {
	response := MCPMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	encoder.Encode(response)
}