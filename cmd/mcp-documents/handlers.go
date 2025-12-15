package main

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DocumentServer implements the MCP tools for working with documents.
// It delegates all data access to a DocumentRepository.
type DocumentServer struct {
	repo DocumentRepository
}

// NewDocumentServer constructs a new DocumentServer.
func NewDocumentServer(repo DocumentRepository) *DocumentServer {
	return &DocumentServer{repo: repo}
}

// registerDocumentTools wires the document tools into the MCP server.
func registerDocumentTools(s *server.MCPServer, documentServer *DocumentServer) {
	getDocumentsTool := mcp.NewTool(
		"get_documents",
		mcp.WithDescription("Get documents with optional filtering by tenant_id, category, tags, or active status"),
		mcp.WithString("tenant_id",
			mcp.Description("Filter by tenant ID"),
		),
		mcp.WithString("category_id",
			mcp.Description("Filter by category ID"),
		),
		mcp.WithArray("tags",
			mcp.Description("Filter by tags (document must have all specified tags)"),
			mcp.WithStringItems(),
		),
		mcp.WithBoolean("is_active",
			mcp.Description("Filter by active status (true for active, false for inactive)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of documents to return (default: 50)"),
			mcp.Min(1),
			mcp.Max(100),
		),
	)

	getDocumentContentTool := mcp.NewTool(
		"get_document_content",
		mcp.WithDescription("Get the full content of specific documents by their IDs"),
		mcp.WithArray("document_ids",
			mcp.Description("Array of document IDs to retrieve content for"),
			mcp.WithStringItems(),
			mcp.Required(),
		),
	)

	searchDocumentsTool := mcp.NewTool(
		"search_documents",
		mcp.WithDescription("Search documents by name, description, or content text"),
		mcp.WithString("query",
			mcp.Description("Search query to match against document name, description, or content"),
			mcp.Required(),
		),
		mcp.WithString("tenant_id",
			mcp.Description("Filter by tenant ID"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 50)"),
			mcp.Min(1),
			mcp.Max(100),
		),
	)

	s.AddTool(getDocumentsTool, documentServer.handleGetDocuments)
	s.AddTool(getDocumentContentTool, documentServer.handleGetDocumentContent)
	s.AddTool(searchDocumentsTool, documentServer.handleSearchDocuments)
}

// handleGetDocuments parses arguments from the CallToolRequest and delegates to the repository.
func (s *DocumentServer) handleGetDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Optional tenant_id
	var tenantID *string
	if v, ok := args["tenant_id"]; ok {
		if tid, ok := v.(string); ok && tid != "" {
			tenantID = &tid
		}
	}

	// Optional category_id
	var categoryID *string
	if v, ok := args["category_id"]; ok {
		if cid, ok := v.(string); ok && cid != "" {
			categoryID = &cid
		}
	}

	// Optional tags
	tags := request.GetStringSlice("tags", nil)

	// Optional is_active with tri-state (unset vs true/false)
	var isActive *bool
	if v, ok := args["is_active"]; ok {
		switch b := v.(type) {
		case bool:
			isActive = &b
		case string:
			// best-effort parse
			if b == "true" {
				val := true
				isActive = &val
			} else if b == "false" {
				val := false
				isActive = &val
			}
		}
	}

	// Limit with clamping
	limit := request.GetInt("limit", 50)
	if limit < 1 {
		limit = 1
	} else if limit > 100 {
		limit = 100
	}

	// Delegate to repository
	documents, err := s.repo.GetDocuments(ctx, tenantID, categoryID, tags, isActive, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
		"limit":     limit,
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGetDocumentContent gets full document content for the given IDs.
func (s *DocumentServer) handleGetDocumentContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	documentIDs := request.GetStringSlice("document_ids", nil)
	if len(documentIDs) == 0 {
		return mcp.NewToolResultError("at least one document ID is required"), nil
	}

	// Delegate to repository
	documents, err := s.repo.GetDocumentContent(ctx, documentIDs)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleSearchDocuments searches documents by query and optional tenant.
func (s *DocumentServer) handleSearchDocuments(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil || query == "" {
		return mcp.NewToolResultError("query is required and must be a non-empty string"), nil
	}

	args := request.GetArguments()

	var tenantID *string
	if v, ok := args["tenant_id"]; ok {
		if tid, ok := v.(string); ok && tid != "" {
			tenantID = &tid
		}
	}

	limit := request.GetInt("limit", 50)
	if limit < 1 {
		limit = 1
	} else if limit > 100 {
		limit = 100
	}

	// Delegate to repository
	documents, err := s.repo.SearchDocuments(ctx, query, tenantID, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	result := map[string]interface{}{
		"query":     query,
		"documents": documents,
		"count":     len(documents),
		"limit":     limit,
	}

	resultJSON, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(resultJSON)), nil
}
