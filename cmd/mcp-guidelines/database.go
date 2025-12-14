package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

var db *sql.DB

// initDatabase initializes the database connection
func initDatabase() error {
	dsn := os.Getenv("GUIDELINES_DB_DSN")
	if dsn == "" {
		return fmt.Errorf("GUIDELINES_DB_DSN environment variable is required")
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set timezone to UTC for consistent timestamp handling
	if _, err := db.Exec("SET timezone = 'UTC'"); err != nil {
		return fmt.Errorf("failed to set timezone to UTC: %w", err)
	}

	return nil
}

// closeDatabase closes the database connection
func closeDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// getGuidelines queries guidelines with optional filters
func getGuidelines(tenantID *string, category *string, tags []string, isActive *bool, limit int) ([]Guideline, error) {
	query := `SELECT id, name, description, content, category, tags, tenant_id, is_active, metadata, created_at, updated_at
		FROM guidelines WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argPos)
		args = append(args, *tenantID)
		argPos++
	}

	if category != nil {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, *category)
		argPos++
	}

	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, *isActive)
		argPos++
	} else {
		// Default to active only if not specified
		query += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, true)
		argPos++
	}

	// Handle tags filter - check if any tag matches
	if len(tags) > 0 {
		query += fmt.Sprintf(" AND tags ?| $%d", argPos)
		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		args = append(args, string(tagsJSON))
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []Guideline
	for rows.Next() {
		var g Guideline
		var description, category, tenantID sql.NullString
		var tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&g.ID,
			&g.Name,
			&description,
			&g.Content,
			&category,
			&tagsJSON,
			&tenantID,
			&g.IsActive,
			&metadataJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if description.Valid {
			g.Description = description.String
		}
		if category.Valid {
			g.Category = category.String
		}
		if tenantID.Valid {
			g.TenantID = tenantID.String
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &g.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &g.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		guidelines = append(guidelines, g)
	}

	return guidelines, rows.Err()
}

// getGuidelinesByIDs retrieves guidelines by their IDs
func getGuidelinesByIDs(ids []string) ([]Guideline, error) {
	if len(ids) == 0 {
		return []Guideline{}, nil
	}

	// Build query with ANY array
	query := `SELECT id, name, description, content, category, tags, tenant_id, is_active, metadata, created_at, updated_at
		FROM guidelines WHERE id = ANY($1)`
	rows, err := db.Query(query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to query guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []Guideline
	for rows.Next() {
		var g Guideline
		var description, category, tenantID sql.NullString
		var tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&g.ID,
			&g.Name,
			&description,
			&g.Content,
			&category,
			&tagsJSON,
			&tenantID,
			&g.IsActive,
			&metadataJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if description.Valid {
			g.Description = description.String
		}
		if category.Valid {
			g.Category = category.String
		}
		if tenantID.Valid {
			g.TenantID = tenantID.String
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &g.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &g.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		guidelines = append(guidelines, g)
	}

	return guidelines, rows.Err()
}

// searchGuidelines searches guidelines by name, description, or content
func searchGuidelines(searchTerm string, tenantID *string, category *string, limit int) ([]Guideline, error) {
	query := `SELECT id, name, description, content, category, tags, tenant_id, is_active, metadata, created_at, updated_at
		FROM guidelines WHERE (name ILIKE $1 OR description ILIKE $1 OR content ILIKE $1)`
	args := []interface{}{"%" + searchTerm + "%"}
	argPos := 2

	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argPos)
		args = append(args, *tenantID)
		argPos++
	}

	if category != nil {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, *category)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []Guideline
	for rows.Next() {
		var g Guideline
		var description, category, tenantID sql.NullString
		var tagsJSON, metadataJSON []byte

		err := rows.Scan(
			&g.ID,
			&g.Name,
			&description,
			&g.Content,
			&category,
			&tagsJSON,
			&tenantID,
			&g.IsActive,
			&metadataJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if description.Valid {
			g.Description = description.String
		}
		if category.Valid {
			g.Category = category.String
		}
		if tenantID.Valid {
			g.TenantID = tenantID.String
		}

		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &g.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &g.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		guidelines = append(guidelines, g)
	}

	return guidelines, rows.Err()
}
