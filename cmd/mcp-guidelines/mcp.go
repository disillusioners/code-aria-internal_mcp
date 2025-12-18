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
				Name:    "mcp-guidelines",
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
			Name:        "get_guidelines",
			Description: "Get guidelines filtered by tenant_id, category, tags, or active status",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tenant_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by tenant ID",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "Filter by tags (any match)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"is_active": map[string]interface{}{
						"type":        "boolean",
						"description": "Filter by active status (default: true)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Limit results (default: 50, max: 100)",
						"minimum":     1,
						"maximum":     100,
					},
				},
			},
		},
		{
			Name:        "get_guideline_content",
			Description: "Get full content of specific guidelines by IDs",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"guideline_ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of guideline IDs",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"guideline_ids"},
			},
		},
		{
			Name:        "search_guidelines",
			Description: "Search guidelines by name, description, or content text",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search_term": map[string]interface{}{
						"type":        "string",
						"description": "Search query",
					},
					"tenant_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by tenant ID",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Limit results (default: 20, max: 50)",
						"minimum":     1,
						"maximum":     50,
					},
				},
				"required": []string{"search_term"},
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

	var result string
	var toolErr error

	switch req.Name {
	case "get_guidelines":
		result, toolErr = toolGetGuidelines(req.Arguments)
	case "get_guideline_content":
		result, toolErr = toolGetGuidelineContent(req.Arguments)
	case "search_guidelines":
		result, toolErr = toolSearchGuidelines(req.Arguments)
	default:
		sendError(encoder, msg.ID, -32601, fmt.Sprintf("Unknown tool: %s", req.Name), nil)
		return
	}

	if toolErr != nil {
		sendError(encoder, msg.ID, -32603, toolErr.Error(), nil)
		return
	}

	// Parse JSON result if possible, otherwise use as string
	var parsedResult interface{}
	if jsonErr := json.Unmarshal([]byte(result), &parsedResult); jsonErr == nil {
		// Successfully parsed JSON, use parsed result
	} else {
		// Not JSON, use string as-is
		parsedResult = result
	}

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
			IsError: false,
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






