package main

import (
	"context"
	"encoding/json"
	"fmt"
)

var globalRepo DocumentRepository

// setGlobalRepository sets the global repository instance for operation handlers
func setGlobalRepository(repo DocumentRepository) {
	globalRepo = repo
}

// toolGetDocuments handles the get_documents operation
func toolGetDocuments(args map[string]interface{}) (string, error) {
	if globalRepo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	ctx := context.Background()

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
	var tags []string
	if tagsInterface, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

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
	limit := 50
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
		if limit > 100 {
			limit = 100
		}
		if limit < 1 {
			limit = 1
		}
	}

	// Delegate to repository
	documents, err := globalRepo.GetDocuments(ctx, tenantID, categoryID, tags, isActive, limit)
	if err != nil {
		return "", fmt.Errorf("failed to get documents: %w", err)
	}

	result := map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
		"limit":     limit,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetDocumentContent handles the get_document_content operation
func toolGetDocumentContent(args map[string]interface{}) (string, error) {
	if globalRepo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	ctx := context.Background()

	documentIDsInterface, ok := args["document_ids"].([]interface{})
	if !ok {
		return "", fmt.Errorf("document_ids array is required")
	}

	if len(documentIDsInterface) == 0 {
		return "[]", nil
	}

	var documentIDs []string
	for _, id := range documentIDsInterface {
		if idStr, ok := id.(string); ok {
			documentIDs = append(documentIDs, idStr)
		}
	}

	if len(documentIDs) == 0 {
		return "", fmt.Errorf("document_ids must contain valid string IDs")
	}

	// Delegate to repository
	documents, err := globalRepo.GetDocumentContent(ctx, documentIDs)
	if err != nil {
		return "", fmt.Errorf("failed to get document content: %w", err)
	}

	result := map[string]interface{}{
		"documents": documents,
		"count":     len(documents),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolSearchDocuments handles the search_documents operation
func toolSearchDocuments(args map[string]interface{}) (string, error) {
	if globalRepo == nil {
		return "", fmt.Errorf("repository not initialized")
	}

	ctx := context.Background()

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required and must be a non-empty string")
	}

	var tenantID *string
	if v, ok := args["tenant_id"]; ok {
		if tid, ok := v.(string); ok && tid != "" {
			tenantID = &tid
		}
	}

	limit := 50
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
		if limit > 100 {
			limit = 100
		}
		if limit < 1 {
			limit = 1
		}
	}

	// Delegate to repository
	documents, err := globalRepo.SearchDocuments(ctx, query, tenantID, limit)
	if err != nil {
		return "", fmt.Errorf("failed to search documents: %w", err)
	}

	result := map[string]interface{}{
		"query":     query,
		"documents": documents,
		"count":     len(documents),
		"limit":     limit,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
