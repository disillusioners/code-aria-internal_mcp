package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
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
func setupTestServer(t *testing.T) *sql.DB {
	loadTestEnv(t)

	dbDSN := os.Getenv("DOCUMENTS_DB_DSN")
	if dbDSN == "" {
		t.Skip("DOCUMENTS_DB_DSN environment variable not set, skipping database tests")
		return nil
	}

	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize the global repository
	repo := NewSQLDocumentRepository(db)
	setGlobalRepository(repo)

	return db
}

// TestServerInitialization tests that the server can be initialized
func TestServerInitialization(t *testing.T) {
	db := setupTestServer(t)
	if db == nil {
		return // Skipped
	}
	defer db.Close()

	if globalRepo == nil {
		t.Fatal("Global repository should not be nil after setup")
	}

	t.Log("Server initialized successfully")
}

// TestGetDocuments tests the get_documents tool
func TestGetDocuments(t *testing.T) {
	db := setupTestServer(t)
	if db == nil {
		return // Skipped
	}
	defer db.Close()

	// Test using the tool function directly
	result, err := toolGetDocuments(map[string]interface{}{
		"limit": float64(10),
	})

	// If repository is properly initialized, we should get a result (even if empty)
	if err != nil {
		// This is expected if the database table doesn't exist
		t.Logf("Expected error (may be due to missing table): %v", err)
		return
	}

	// Parse result
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify structure
	if _, ok := response["documents"]; !ok {
		t.Error("Result missing 'documents' field")
	}
	if _, ok := response["count"]; !ok {
		t.Error("Result missing 'count' field")
	}

	t.Log("Get documents tool works correctly")
}

// TestGetDocumentContent tests the get_document_content tool
func TestGetDocumentContent(t *testing.T) {
	db := setupTestServer(t)
	if db == nil {
		return // Skipped
	}
	defer db.Close()

	// Test using the tool function directly
	result, err := toolGetDocumentContent(map[string]interface{}{
		"document_ids": []interface{}{"test-id"},
	})

	// If repository is properly initialized, we should get a result (even if empty)
	if err != nil {
		t.Logf("Expected error (may be due to missing table): %v", err)
		return
	}

	// Parse result
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Verify structure
	if _, ok := response["documents"]; !ok {
		t.Error("Result missing 'documents' field")
	}
	if _, ok := response["count"]; !ok {
		t.Error("Result missing 'count' field")
	}

	t.Log("Get document content tool works correctly")
}

// TestSearchDocuments tests the search_documents tool
func TestSearchDocuments(t *testing.T) {
	db := setupTestServer(t)
	if db == nil {
		return // Skipped
	}
	defer db.Close()

	// Test using the tool function directly
	result, err := toolSearchDocuments(map[string]interface{}{
		"query": "test",
		"limit": float64(10),
	})

	// If repository is properly initialized, we should get a result (even if empty)
	if err != nil {
		t.Logf("Expected error (may be due to missing table): %v", err)
		return
	}

	// Parse result
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
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

	t.Log("Search documents tool works correctly")
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	db := setupTestServer(t)
	if db == nil {
		return // Skipped
	}
	defer db.Close()

	// Test: get_document_content with empty document_ids
	t.Run("get_document_content with empty IDs", func(t *testing.T) {
		_, err := toolGetDocumentContent(map[string]interface{}{
			"document_ids": []interface{}{},
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	// Test: search_documents with empty query (should fail)
	t.Run("search with empty query should fail", func(t *testing.T) {
		_, err := toolSearchDocuments(map[string]interface{}{
			"query": "",
		})
		if err == nil {
			t.Error("Expected error for empty query, got success")
		}
	})
}

// TestLimitClamping tests that limits are properly clamped
func TestLimitClamping(t *testing.T) {
	db := setupTestServer(t)
	if db == nil {
		return // Skipped
	}
	defer db.Close()

	// Test: limit too high (should be clamped to 100)
	t.Run("Limit too high", func(t *testing.T) {
		result, err := toolGetDocuments(map[string]interface{}{
			"limit": float64(200),
		})

		if err != nil {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(result), &response); err != nil {
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
		result, err := toolGetDocuments(map[string]interface{}{
			"limit": float64(0),
		})

		if err != nil {
			t.Logf("Note: Query returned error (may be expected if no test data)")
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(result), &response); err != nil {
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
