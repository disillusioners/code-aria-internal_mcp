package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// DocumentRepository defines the persistence operations used by the MCP handlers.
type DocumentRepository interface {
	GetDocuments(ctx context.Context, tenantID, categoryID *string, tags []string, isActive *bool, limit int) ([]map[string]interface{}, error)
	GetDocumentContent(ctx context.Context, documentIDs []string) ([]map[string]interface{}, error)
	SearchDocuments(ctx context.Context, query string, tenantID *string, limit int) ([]map[string]interface{}, error)
}

// SQLDocumentRepository is a Postgres-backed implementation of DocumentRepository.
type SQLDocumentRepository struct {
	db *sql.DB
}

// NewSQLDocumentRepository constructs a new SQLDocumentRepository.
func NewSQLDocumentRepository(db *sql.DB) *SQLDocumentRepository {
	return &SQLDocumentRepository{db: db}
}

func (r *SQLDocumentRepository) GetDocuments(
	ctx context.Context,
	tenantID, categoryID *string,
	tags []string,
	isActive *bool,
	limit int,
) ([]map[string]interface{}, error) {
	// Build query
	query := `
		SELECT id, name, description, content, category_id, tags, tenant_id, is_active, metadata, created_at, updated_at
		FROM documents
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *tenantID)
		argIndex++
	}

	if categoryID != nil {
		query += fmt.Sprintf(" AND category_id = $%d", argIndex)
		args = append(args, *categoryID)
		argIndex++
	}

	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *isActive)
		argIndex++
	}

	// Add tag filtering if specified
	if len(tags) > 0 {
		for _, tag := range tags {
			query += fmt.Sprintf(" AND tags @> $%d", argIndex)
			args = append(args, []string{tag})
			argIndex++
		}
	}

	query += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d", argIndex)
	args = append(args, limit)

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []map[string]interface{}
	for rows.Next() {
		var id, name, description, content, tenantID sql.NullString
		var categoryID sql.NullString
		var tagsJSON []byte
		var metadataJSON []byte
		var isActive bool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&id, &name, &description, &content,
			&categoryID, &tagsJSON, &tenantID, &isActive,
			&metadataJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		doc := map[string]interface{}{
			"id":        id.String,
			"name":      name.String,
			"is_active": isActive,
		}

		if description.Valid {
			doc["description"] = description.String
		}

		if content.Valid {
			// For listing, only include first 500 characters of content
			if len(content.String) > 500 {
				doc["content_preview"] = content.String[:500] + "..."
			} else {
				doc["content_preview"] = content.String
			}
		}

		if categoryID.Valid {
			doc["category_id"] = categoryID.String
		}

		if len(tagsJSON) > 0 {
			var tags []string
			if err := json.Unmarshal(tagsJSON, &tags); err == nil {
				doc["tags"] = tags
			}
		}

		if tenantID.Valid {
			doc["tenant_id"] = tenantID.String
		}

		if len(metadataJSON) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
				doc["metadata"] = metadata
			}
		}

		if createdAt.Valid {
			doc["created_at"] = createdAt.Time
		}

		if updatedAt.Valid {
			doc["updated_at"] = updatedAt.Time
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

func (r *SQLDocumentRepository) GetDocumentContent(
	ctx context.Context,
	documentIDs []string,
) ([]map[string]interface{}, error) {
	if len(documentIDs) == 0 {
		return nil, fmt.Errorf("at least one document ID is required")
	}

	// Build IN clause placeholders
	placeholders := make([]string, len(documentIDs))
	args := make([]interface{}, len(documentIDs))
	for i, id := range documentIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, description, content, category_id, tags, tenant_id, is_active, metadata, created_at, updated_at
		FROM documents
		WHERE id IN (%s)
		ORDER BY name`, strings.Join(placeholders, ", "))

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []map[string]interface{}
	for rows.Next() {
		var id, name, description, content, tenantID sql.NullString
		var categoryID sql.NullString
		var tagsJSON []byte
		var metadataJSON []byte
		var isActive bool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&id, &name, &description, &content,
			&categoryID, &tagsJSON, &tenantID, &isActive,
			&metadataJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		doc := map[string]interface{}{
			"id":        id.String,
			"name":      name.String,
			"is_active": isActive,
		}

		if content.Valid {
			doc["content"] = content.String
		}

		if description.Valid {
			doc["description"] = description.String
		}

		if categoryID.Valid {
			doc["category_id"] = categoryID.String
		}

		if len(tagsJSON) > 0 {
			var tags []string
			if err := json.Unmarshal(tagsJSON, &tags); err == nil {
				doc["tags"] = tags
			}
		}

		if tenantID.Valid {
			doc["tenant_id"] = tenantID.String
		}

		if len(metadataJSON) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
				doc["metadata"] = metadata
			}
		}

		if createdAt.Valid {
			doc["created_at"] = createdAt.Time
		}

		if updatedAt.Valid {
			doc["updated_at"] = updatedAt.Time
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

func (r *SQLDocumentRepository) SearchDocuments(
	ctx context.Context,
	query string,
	tenantID *string,
	limit int,
) ([]map[string]interface{}, error) {
	// Build search query
	searchQuery := `
		SELECT id, name, description, content, category_id, tags, tenant_id, is_active, metadata, created_at, updated_at
		FROM documents
		WHERE (name ILIKE $1 OR description ILIKE $1 OR content ILIKE $1)`

	args := []interface{}{"%" + query + "%"}
	argIndex := 2

	if tenantID != nil {
		searchQuery += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *tenantID)
		argIndex++
	}

	searchQuery += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d", argIndex)
	args = append(args, limit)

	// Execute query
	rows, err := r.db.QueryContext(ctx, searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	var documents []map[string]interface{}
	for rows.Next() {
		var id, name, description, content, tenantID sql.NullString
		var categoryID sql.NullString
		var tagsJSON []byte
		var metadataJSON []byte
		var isActive bool
		var createdAt, updatedAt sql.NullTime

		err := rows.Scan(
			&id, &name, &description, &content,
			&categoryID, &tagsJSON, &tenantID, &isActive,
			&metadataJSON, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		doc := map[string]interface{}{
			"id":        id.String,
			"name":      name.String,
			"is_active": isActive,
		}

		if description.Valid {
			doc["description"] = description.String
		}

		// Highlight matches in content preview
		if content.Valid {
			contentPreview := content.String
			if len(contentPreview) > 1000 {
				contentPreview = contentPreview[:1000] + "..."
			}
			doc["content_preview"] = contentPreview
		}

		if categoryID.Valid {
			doc["category_id"] = categoryID.String
		}

		if len(tagsJSON) > 0 {
			var tags []string
			if err := json.Unmarshal(tagsJSON, &tags); err == nil {
				doc["tags"] = tags
			}
		}

		if tenantID.Valid {
			doc["tenant_id"] = tenantID.String
		}

		if len(metadataJSON) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataJSON, &metadata); err == nil {
				doc["metadata"] = metadata
			}
		}

		if createdAt.Valid {
			doc["created_at"] = createdAt.Time
		}

		if updatedAt.Valid {
			doc["updated_at"] = updatedAt.Time
		}

		documents = append(documents, doc)
	}

	return documents, nil
}
