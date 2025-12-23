package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

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
	query := `SELECT g.id, g.name, g.description, g.content, g.category_id, g.tags, g.tenant_id, g.is_active, g.metadata, g.created_at, g.updated_at,
		       gc.id, gc.name, gc.description, gc.color, gc.icon, gc.metadata, gc.is_active, gc.tenant_id, gc.created_at, gc.updated_at, gc.created_by, gc.updated_by
		FROM guidelines g
		LEFT JOIN guideline_categories gc ON g.category_id = gc.id
		WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if tenantID != nil {
		query += fmt.Sprintf(" AND g.tenant_id = $%d", argPos)
		args = append(args, *tenantID)
		argPos++
	}

	if category != nil {
		query += fmt.Sprintf(" AND g.category_id = $%d", argPos)
		args = append(args, *category)
		argPos++
	}

	if isActive != nil {
		query += fmt.Sprintf(" AND g.is_active = $%d", argPos)
		args = append(args, *isActive)
		argPos++
	} else {
		// Default to active only if not specified
		query += fmt.Sprintf(" AND g.is_active = $%d", argPos)
		args = append(args, true)
		argPos++
	}

	// Handle tags filter - check if any tag matches
	if len(tags) > 0 {
		query += fmt.Sprintf(" AND g.tags ?| $%d", argPos)
		tagsJSON, err := json.Marshal(tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tags: %w", err)
		}
		args = append(args, string(tagsJSON))
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY g.created_at DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []Guideline
	for rows.Next() {
		var g Guideline
		var description, categoryID, tenantID sql.NullString
		var tagsJSON, metadataJSON []byte
		var catID sql.NullString
		var cat GuidelineCategory
		var catDescription, catColor, catIcon, catCreatedBy, catUpdatedBy sql.NullString
		var catMetadataJSON []byte

		err := rows.Scan(
			&g.ID,
			&g.Name,
			&description,
			&g.Content,
			&categoryID,
			&tagsJSON,
			&tenantID,
			&g.IsActive,
			&metadataJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
			&catID,
			&cat.Name,
			&catDescription,
			&catColor,
			&catIcon,
			&catMetadataJSON,
			&cat.IsActive,
			&cat.TenantID,
			&cat.CreatedAt,
			&cat.UpdatedAt,
			&catCreatedBy,
			&catUpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if description.Valid {
			g.Description = description.String
		}
		if categoryID.Valid {
			g.CategoryID = &categoryID.String
		}
		if tenantID.Valid {
			g.TenantID = tenantID.String
		}

		if catID.Valid {
			cat.ID = catID.String
			if catDescription.Valid {
				cat.Description = catDescription.String
			}
			if catColor.Valid {
				cat.Color = catColor.String
			}
			if catIcon.Valid {
				cat.Icon = catIcon.String
			}
			if catCreatedBy.Valid {
				cat.CreatedBy = catCreatedBy.String
			}
			if catUpdatedBy.Valid {
				cat.UpdatedBy = catUpdatedBy.String
			}
			if len(catMetadataJSON) > 0 {
				if err := json.Unmarshal(catMetadataJSON, &cat.Metadata); err != nil {
					return nil, fmt.Errorf("failed to unmarshal category metadata: %w", err)
				}
			}
			g.Category = &cat
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

	// Build query with ANY array and JOIN category
	query := `SELECT g.id, g.name, g.description, g.content, g.category_id, g.tags, g.tenant_id, g.is_active, g.metadata, g.created_at, g.updated_at,
		       gc.id, gc.name, gc.description, gc.color, gc.icon, gc.metadata, gc.is_active, gc.tenant_id, gc.created_at, gc.updated_at, gc.created_by, gc.updated_by
		FROM guidelines g
		LEFT JOIN guideline_categories gc ON g.category_id = gc.id
		WHERE g.id = ANY($1)`
	rows, err := db.Query(query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to query guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []Guideline
	for rows.Next() {
		var g Guideline
		var description, categoryID, tenantID sql.NullString
		var tagsJSON, metadataJSON []byte
		var catID sql.NullString
		var cat GuidelineCategory
		var catDescription, catColor, catIcon, catCreatedBy, catUpdatedBy sql.NullString
		var catMetadataJSON []byte

		err := rows.Scan(
			&g.ID,
			&g.Name,
			&description,
			&g.Content,
			&categoryID,
			&tagsJSON,
			&tenantID,
			&g.IsActive,
			&metadataJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
			&catID,
			&cat.Name,
			&catDescription,
			&catColor,
			&catIcon,
			&catMetadataJSON,
			&cat.IsActive,
			&cat.TenantID,
			&cat.CreatedAt,
			&cat.UpdatedAt,
			&catCreatedBy,
			&catUpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if description.Valid {
			g.Description = description.String
		}
		if categoryID.Valid {
			g.CategoryID = &categoryID.String
		}
		if tenantID.Valid {
			g.TenantID = tenantID.String
		}

		if catID.Valid {
			cat.ID = catID.String
			if catDescription.Valid {
				cat.Description = catDescription.String
			}
			if catColor.Valid {
				cat.Color = catColor.String
			}
			if catIcon.Valid {
				cat.Icon = catIcon.String
			}
			if catCreatedBy.Valid {
				cat.CreatedBy = catCreatedBy.String
			}
			if catUpdatedBy.Valid {
				cat.UpdatedBy = catUpdatedBy.String
			}
			if len(catMetadataJSON) > 0 {
				if err := json.Unmarshal(catMetadataJSON, &cat.Metadata); err != nil {
					return nil, fmt.Errorf("failed to unmarshal category metadata: %w", err)
				}
			}
			g.Category = &cat
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
	query := `SELECT g.id, g.name, g.description, g.content, g.category_id, g.tags, g.tenant_id, g.is_active, g.metadata, g.created_at, g.updated_at,
		       gc.id, gc.name, gc.description, gc.color, gc.icon, gc.metadata, gc.is_active, gc.tenant_id, gc.created_at, gc.updated_at, gc.created_by, gc.updated_by
		FROM guidelines g
		LEFT JOIN guideline_categories gc ON g.category_id = gc.id
		WHERE (g.name ILIKE $1 OR g.description ILIKE $1 OR g.content ILIKE $1)`
	args := []interface{}{"%" + searchTerm + "%"}
	argPos := 2

	if tenantID != nil {
		query += fmt.Sprintf(" AND g.tenant_id = $%d", argPos)
		args = append(args, *tenantID)
		argPos++
	}

	if category != nil {
		query += fmt.Sprintf(" AND g.category_id = $%d", argPos)
		args = append(args, *category)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY g.created_at DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []Guideline
	for rows.Next() {
		var g Guideline
		var description, categoryID, tenantID sql.NullString
		var tagsJSON, metadataJSON []byte
		var catID sql.NullString
		var cat GuidelineCategory
		var catDescription, catColor, catIcon, catCreatedBy, catUpdatedBy sql.NullString
		var catMetadataJSON []byte

		err := rows.Scan(
			&g.ID,
			&g.Name,
			&description,
			&g.Content,
			&categoryID,
			&tagsJSON,
			&tenantID,
			&g.IsActive,
			&metadataJSON,
			&g.CreatedAt,
			&g.UpdatedAt,
			&catID,
			&cat.Name,
			&catDescription,
			&catColor,
			&catIcon,
			&catMetadataJSON,
			&cat.IsActive,
			&cat.TenantID,
			&cat.CreatedAt,
			&cat.UpdatedAt,
			&catCreatedBy,
			&catUpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if description.Valid {
			g.Description = description.String
		}
		if categoryID.Valid {
			g.CategoryID = &categoryID.String
		}
		if tenantID.Valid {
			g.TenantID = tenantID.String
		}

		if catID.Valid {
			cat.ID = catID.String
			if catDescription.Valid {
				cat.Description = catDescription.String
			}
			if catColor.Valid {
				cat.Color = catColor.String
			}
			if catIcon.Valid {
				cat.Icon = catIcon.String
			}
			if catCreatedBy.Valid {
				cat.CreatedBy = catCreatedBy.String
			}
			if catUpdatedBy.Valid {
				cat.UpdatedBy = catUpdatedBy.String
			}
			if len(catMetadataJSON) > 0 {
				if err := json.Unmarshal(catMetadataJSON, &cat.Metadata); err != nil {
					return nil, fmt.Errorf("failed to unmarshal category metadata: %w", err)
				}
			}
			g.Category = &cat
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

// Category CRUD functions
func createCategory(category *GuidelineCategory) error {
	query := `
		INSERT INTO guideline_categories (id, name, description, color, icon, metadata, is_active, tenant_id, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, err := json.Marshal(category.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = db.Exec(query,
		category.ID,
		category.Name,
		category.Description,
		category.Color,
		category.Icon,
		metadataJSON,
		category.IsActive,
		category.TenantID,
		category.CreatedAt,
		category.UpdatedAt,
		category.CreatedBy,
	)

	return err
}

func getCategory(id string) (*GuidelineCategory, error) {
	query := `
		SELECT id, name, description, color, icon, metadata, is_active, tenant_id, created_at, updated_at, created_by, updated_by
		FROM guideline_categories
		WHERE id = $1
	`

	var category GuidelineCategory
	var description, color, icon, createdBy, updatedBy sql.NullString
	var metadataJSON []byte

	err := db.QueryRow(query, id).Scan(
		&category.ID,
		&category.Name,
		&description,
		&color,
		&icon,
		&metadataJSON,
		&category.IsActive,
		&category.TenantID,
		&category.CreatedAt,
		&category.UpdatedAt,
		&createdBy,
		&updatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found: %s", id)
		}
		return nil, err
	}

	if description.Valid {
		category.Description = description.String
	}
	if color.Valid {
		category.Color = color.String
	}
	if icon.Valid {
		category.Icon = icon.String
	}
	if createdBy.Valid {
		category.CreatedBy = createdBy.String
	}
	if updatedBy.Valid {
		category.UpdatedBy = updatedBy.String
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &category.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &category, nil
}

func listCategories(tenantID *string, isActive *bool) ([]GuidelineCategory, error) {
	query := `
		SELECT id, name, description, color, icon, metadata, is_active, tenant_id, created_at, updated_at, created_by, updated_by
		FROM guideline_categories
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argPos)
		args = append(args, *tenantID)
		argPos++
	}

	if isActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", argPos)
		args = append(args, *isActive)
		argPos++
	}

	query += " ORDER BY name"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()

	var categories []GuidelineCategory
	for rows.Next() {
		var category GuidelineCategory
		var description, color, icon, createdBy, updatedBy sql.NullString
		var metadataJSON []byte

		err := rows.Scan(
			&category.ID,
			&category.Name,
			&description,
			&color,
			&icon,
			&metadataJSON,
			&category.IsActive,
			&category.TenantID,
			&category.CreatedAt,
			&category.UpdatedAt,
			&createdBy,
			&updatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		if description.Valid {
			category.Description = description.String
		}
		if color.Valid {
			category.Color = color.String
		}
		if icon.Valid {
			category.Icon = icon.String
		}
		if createdBy.Valid {
			category.CreatedBy = createdBy.String
		}
		if updatedBy.Valid {
			category.UpdatedBy = updatedBy.String
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &category.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		categories = append(categories, category)
	}

	return categories, rows.Err()
}

func updateCategory(category *GuidelineCategory) error {
	query := `
		UPDATE guideline_categories
		SET name = $2, description = $3, color = $4, icon = $5, metadata = $6, is_active = $7, updated_at = $8, updated_by = $9
		WHERE id = $1
	`

	metadataJSON, err := json.Marshal(category.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	result, err := db.Exec(query,
		category.ID,
		category.Name,
		category.Description,
		category.Color,
		category.Icon,
		metadataJSON,
		category.IsActive,
		category.UpdatedAt,
		category.UpdatedBy,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category not found: %s", category.ID)
	}

	return nil
}

func deleteCategory(id string) error {
	query := `
		UPDATE guideline_categories
		SET is_active = false, updated_at = $2
		WHERE id = $1
	`

	result, err := db.Exec(query, id, time.Now().UTC())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("category not found: %s", id)
	}

	return nil
}
