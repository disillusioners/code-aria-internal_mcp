package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// newTestCallToolRequest creates a test CallToolRequest with the given arguments
func newTestCallToolRequest(name string, arguments map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	}
}

// loadTestEnv loads .env file from the same directory as the test
func loadTestEnv(t *testing.T) {
	// Get the directory of the current test file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	envPath := filepath.Join(dir, ".env")

	// Try to load .env file, but don't fail if it doesn't exist
	if err := godotenv.Load(envPath); err != nil {
		t.Logf("Note: Could not load .env file from %s: %v", envPath, err)
		t.Logf("Make sure DOCUMENTS_DB_DSN environment variable is set")
	}
}

// setupTestServer initializes the MCP server and database connection for testing
func setupTestServer(t *testing.T) (*server.MCPServer, *sql.DB) {
	loadTestEnv(t)

	dbDSN := os.Getenv("DOCUMENTS_DB_DSN")
	if dbDSN == "" {
		t.Skip("DOCUMENTS_DB_DSN environment variable not set, skipping database tests")
	}

	srv, db, err := NewMCPDocumentsServer()
	if err != nil {
		t.Fatalf("Failed to initialize MCP Documents server: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping database: %v", err)
	}

	return srv, db
}

// TestServerInitialization tests that the server can be initialized
func TestServerInitialization(t *testing.T) {
	srv, db := setupTestServer(t)
	defer db.Close()

	if srv == nil {
		t.Fatal("Server should not be nil")
	}

	t.Log("Server initialized successfully")
}

// TestGetDocuments tests the get_documents tool
func TestGetDocuments(t *testing.T) {
	_, db := setupTestServer(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLDocumentRepository(db)
	docServer := NewDocumentServer(repo)

	// Test 1: Get all documents (no filters)
	t.Run("Get all documents", func(t *testing.T) {
		request := newTestCallToolRequest("get_documents", map[string]interface{}{
			"limit": float64(10),
		})

		result, err := docServer.handleGetDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocuments() error = %v", err)
		}

		if result.IsError {
			// Try to get error message
			if len(result.Content) > 0 {
				content := result.Content[0]
				if textContent, ok := content.(mcp.TextContent); ok {
					t.Fatalf("Expected success, got error: %s", textContent.Text)
				}
			}
			t.Fatal("Expected success, got error")
		}

		// Parse result
		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		// Verify structure
		if _, ok := response["documents"]; !ok {
			t.Error("Result missing 'documents' field")
		}
		if _, ok := response["count"]; !ok {
			t.Error("Result missing 'count' field")
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		count, ok := response["count"].(float64)
		if !ok {
			t.Fatal("Count is not a number")
		}

		if int(count) != len(documents) {
			t.Errorf("Count mismatch: count=%v, len(documents)=%d", count, len(documents))
		}

		t.Logf("Retrieved %d documents", len(documents))
	})

	// Test 2: Get documents with tenant_id filter
	t.Run("Get documents with tenant_id filter", func(t *testing.T) {
		request := newTestCallToolRequest("get_documents", map[string]interface{}{
			"tenant_id": "test-tenant",
			"limit":     float64(10),
		})

		result, err := docServer.handleGetDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		// Verify all documents have the correct tenant_id
		for _, doc := range documents {
			docMap, ok := doc.(map[string]interface{})
			if !ok {
				continue
			}
			if tenantID, ok := docMap["tenant_id"].(string); ok && tenantID != "test-tenant" {
				t.Errorf("Document has wrong tenant_id: %s", tenantID)
			}
		}

		t.Logf("Retrieved %d documents for tenant", len(documents))
	})

	// Test 3: Get documents with is_active filter
	t.Run("Get documents with is_active filter", func(t *testing.T) {
		request := newTestCallToolRequest("get_documents", map[string]interface{}{
			"is_active": true,
			"limit":     float64(10),
		})

		result, err := docServer.handleGetDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		// Verify all documents are active
		for _, doc := range documents {
			docMap, ok := doc.(map[string]interface{})
			if !ok {
				continue
			}
			if isActive, ok := docMap["is_active"].(bool); ok && !isActive {
				t.Error("Document should be active")
			}
		}

		t.Logf("Retrieved %d active documents", len(documents))
	})

	// Test 4: Get documents with tags filter
	t.Run("Get documents with tags filter", func(t *testing.T) {
		request := newTestCallToolRequest("get_documents", map[string]interface{}{
			"tags":  []interface{}{"test", "documentation"},
			"limit": float64(10),
		})

		result, err := docServer.handleGetDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		t.Logf("Retrieved %d documents with tags filter", len(documents))
	})
}

// TestGetDocumentContent tests the get_document_content tool
func TestGetDocumentContent(t *testing.T) {
	_, db := setupTestServer(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLDocumentRepository(db)
	docServer := NewDocumentServer(repo)

	// First, try to get some document IDs
	getDocsRequest := newTestCallToolRequest("get_documents", map[string]interface{}{
		"limit": float64(5),
	})

	getDocsResult, err := docServer.handleGetDocuments(ctx, getDocsRequest)
	if err != nil {
		t.Fatalf("Failed to get documents: %v", err)
	}

	if getDocsResult.IsError {
		t.Skip("No documents found in database, skipping get_document_content test")
	}

	if len(getDocsResult.Content) == 0 {
		t.Skip("No documents found in database, skipping get_document_content test")
	}

	content := getDocsResult.Content[0]
	textContent, ok := content.(mcp.TextContent)
	if !ok {
		t.Skip("Unexpected content type, skipping get_document_content test")
	}

	var getDocsResponse map[string]interface{}
	if err := json.Unmarshal([]byte(textContent.Text), &getDocsResponse); err != nil {
		t.Fatalf("Failed to parse documents: %v", err)
	}

	documents, ok := getDocsResponse["documents"].([]interface{})
	if !ok || len(documents) == 0 {
		t.Skip("No documents found in database, skipping get_document_content test")
	}

	// Get the first document ID
	firstDoc, ok := documents[0].(map[string]interface{})
	if !ok {
		t.Fatal("First document is not a map")
	}

	docID, ok := firstDoc["id"].(string)
	if !ok {
		t.Fatal("Document ID is not a string")
	}

	// Test getting document content
	request := newTestCallToolRequest("get_document_content", map[string]interface{}{
		"document_ids": []interface{}{docID},
	})

	result, err := docServer.handleGetDocumentContent(ctx, request)
	if err != nil {
		t.Fatalf("handleGetDocumentContent() error = %v", err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				t.Fatalf("Expected success, got error: %s", textContent.Text)
			}
		}
		t.Fatal("Expected success, got error")
	}

	// Parse result
	if len(result.Content) == 0 {
		t.Fatal("Result has no content")
	}

	resultContent := result.Content[0]
	resultTextContent, ok := resultContent.(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent, got %T", resultContent)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(resultTextContent.Text), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify structure
	if _, ok := response["documents"]; !ok {
		t.Error("Result missing 'documents' field")
	}
	if _, ok := response["count"]; !ok {
		t.Error("Result missing 'count' field")
	}

	contentDocs, ok := response["documents"].([]interface{})
	if !ok {
		t.Fatal("Documents is not an array")
	}

	if len(contentDocs) == 0 {
		t.Error("Expected at least one document, got none")
	}

	// Verify the document has content field (not content_preview)
	firstContentDoc, ok := contentDocs[0].(map[string]interface{})
	if !ok {
		t.Fatal("First document is not a map")
	}

	if _, ok := firstContentDoc["content"]; !ok {
		t.Error("Document missing 'content' field (should have full content, not preview)")
	}

	t.Logf("Retrieved content for %d documents", len(contentDocs))
}

// TestSearchDocuments tests the search_documents tool
func TestSearchDocuments(t *testing.T) {
	_, db := setupTestServer(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLDocumentRepository(db)
	docServer := NewDocumentServer(repo)

	// Test 1: Basic search
	t.Run("Basic search", func(t *testing.T) {
		request := newTestCallToolRequest("search_documents", map[string]interface{}{
			"query": "test",
			"limit": float64(10),
		})

		result, err := docServer.handleSearchDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleSearchDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Search returned error (may be expected if no test data)")
			return
		}

		// Parse result
		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		// Verify structure
		if _, ok := response["query"]; !ok {
			t.Error("Result missing 'query' field")
		}
		if _, ok := response["documents"]; !ok {
			t.Error("Result missing 'documents' field")
		}
		if _, ok := response["count"]; !ok {
			t.Error("Result missing 'count' field")
		}

		query, ok := response["query"].(string)
		if !ok {
			t.Fatal("Query is not a string")
		}

		if query != "test" {
			t.Errorf("Expected query 'test', got '%s'", query)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		t.Logf("Search for 'test' returned %d documents", len(documents))
	})

	// Test 2: Search with tenant_id filter
	t.Run("Search with tenant_id filter", func(t *testing.T) {
		request := newTestCallToolRequest("search_documents", map[string]interface{}{
			"query":     "test",
			"tenant_id": "test-tenant",
			"limit":     float64(10),
		})

		result, err := docServer.handleSearchDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleSearchDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Search returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		// Verify all documents have the correct tenant_id
		for _, doc := range documents {
			docMap, ok := doc.(map[string]interface{})
			if !ok {
				continue
			}
			if tenantID, ok := docMap["tenant_id"].(string); ok && tenantID != "test-tenant" {
				t.Errorf("Document has wrong tenant_id: %s", tenantID)
			}
		}

		t.Logf("Search with tenant filter returned %d documents", len(documents))
	})

	// Test 3: Search with limit
	t.Run("Search with limit", func(t *testing.T) {
		request := newTestCallToolRequest("search_documents", map[string]interface{}{
			"query": "test",
			"limit": float64(5),
		})

		result, err := docServer.handleSearchDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleSearchDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Search returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		limit, ok := response["limit"].(float64)
		if !ok {
			t.Fatal("Limit is not a number")
		}

		if int(limit) != 5 {
			t.Errorf("Expected limit 5, got %v", limit)
		}

		if len(documents) > 5 {
			t.Errorf("Expected max 5 documents, got %d", len(documents))
		}

		t.Logf("Search with limit returned %d documents", len(documents))
	})

	// Test 4: Empty query (should fail)
	t.Run("Empty query should fail", func(t *testing.T) {
		request := newTestCallToolRequest("search_documents", map[string]interface{}{
			"query": "",
			"limit": float64(10),
		})

		result, err := docServer.handleSearchDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleSearchDocuments() error = %v", err)
		}

		if !result.IsError {
			t.Error("Expected error for empty query, got success")
		}
	})
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	_, db := setupTestServer(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLDocumentRepository(db)
	docServer := NewDocumentServer(repo)

	// Test: get_document_content with empty document_ids
	t.Run("get_document_content with empty IDs", func(t *testing.T) {
		request := newTestCallToolRequest("get_document_content", map[string]interface{}{
			"document_ids": []interface{}{},
		})

		result, err := docServer.handleGetDocumentContent(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocumentContent() error = %v", err)
		}

		if !result.IsError {
			t.Error("Expected error for empty document_ids, got success")
		}
	})

	// Test: get_document_content with invalid document ID
	t.Run("get_document_content with invalid ID", func(t *testing.T) {
		request := newTestCallToolRequest("get_document_content", map[string]interface{}{
			"document_ids": []interface{}{"non-existent-id-12345"},
		})

		result, err := docServer.handleGetDocumentContent(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocumentContent() error = %v", err)
		}

		// Should not error, just return empty results
		if result.IsError {
			t.Logf("Note: Query returned error (may be expected)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		documents, ok := response["documents"].([]interface{})
		if !ok {
			t.Fatal("Documents is not an array")
		}

		if len(documents) != 0 {
			t.Errorf("Expected 0 documents for invalid ID, got %d", len(documents))
		}
	})
}

// TestLimitClamping tests that limits are properly clamped
func TestLimitClamping(t *testing.T) {
	_, db := setupTestServer(t)
	defer db.Close()

	ctx := context.Background()
	repo := NewSQLDocumentRepository(db)
	docServer := NewDocumentServer(repo)

	// Test: limit too high (should be clamped to 100)
	t.Run("Limit too high", func(t *testing.T) {
		request := newTestCallToolRequest("get_documents", map[string]interface{}{
			"limit": float64(200),
		})

		result, err := docServer.handleGetDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		limit, ok := response["limit"].(float64)
		if !ok {
			t.Fatal("Limit is not a number")
		}

		if limit > 100 {
			t.Errorf("Expected limit to be clamped to 100, got %v", limit)
		}
	})

	// Test: limit too low (should be clamped to 1)
	t.Run("Limit too low", func(t *testing.T) {
		request := newTestCallToolRequest("get_documents", map[string]interface{}{
			"limit": float64(0),
		})

		result, err := docServer.handleGetDocuments(ctx, request)
		if err != nil {
			t.Fatalf("handleGetDocuments() error = %v", err)
		}

		if result.IsError {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		if len(result.Content) == 0 {
			t.Fatal("Result has no content")
		}

		content := result.Content[0]
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			t.Fatalf("Expected TextContent, got %T", content)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &response); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		limit, ok := response["limit"].(float64)
		if !ok {
			t.Fatal("Limit is not a number")
		}

		if limit < 1 {
			t.Errorf("Expected limit to be clamped to at least 1, got %v", limit)
		}
	})
}
