package main

import (
	"encoding/json"
	"fmt"
)

// toolGetGuidelines handles the get_guidelines tool call
func toolGetGuidelines(args map[string]interface{}) (string, error) {
	var tenantID *string
	if tid, ok := args["tenant_id"].(string); ok && tid != "" {
		tenantID = &tid
	}

	var category *string
	if cat, ok := args["category"].(string); ok && cat != "" {
		category = &cat
	}

	var tags []string
	if tagsInterface, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	var isActive *bool
	if active, ok := args["is_active"].(bool); ok {
		isActive = &active
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

	guidelines, err := getGuidelines(tenantID, category, tags, isActive, limit)
	if err != nil {
		return "", fmt.Errorf("failed to get guidelines: %w", err)
	}

	resultJSON, err := json.Marshal(guidelines)
	if err != nil {
		return "", fmt.Errorf("failed to marshal guidelines: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetGuidelineContent handles the get_guideline_content tool call
func toolGetGuidelineContent(args map[string]interface{}) (string, error) {
	guidelineIDsInterface, ok := args["guideline_ids"].([]interface{})
	if !ok {
		return "", fmt.Errorf("guideline_ids array is required")
	}

	if len(guidelineIDsInterface) == 0 {
		return "[]", nil
	}

	var guidelineIDs []string
	for _, id := range guidelineIDsInterface {
		if idStr, ok := id.(string); ok {
			guidelineIDs = append(guidelineIDs, idStr)
		}
	}

	if len(guidelineIDs) == 0 {
		return "", fmt.Errorf("guideline_ids must contain valid string IDs")
	}

	guidelines, err := getGuidelinesByIDs(guidelineIDs)
	if err != nil {
		return "", fmt.Errorf("failed to get guideline content: %w", err)
	}

	resultJSON, err := json.Marshal(guidelines)
	if err != nil {
		return "", fmt.Errorf("failed to marshal guidelines: %w", err)
	}

	return string(resultJSON), nil
}

// toolSearchGuidelines handles the search_guidelines tool call
func toolSearchGuidelines(args map[string]interface{}) (string, error) {
	searchTerm, ok := args["search_term"].(string)
	if !ok || searchTerm == "" {
		return "", fmt.Errorf("search_term is required")
	}

	var tenantID *string
	if tid, ok := args["tenant_id"].(string); ok && tid != "" {
		tenantID = &tid
	}

	var category *string
	if cat, ok := args["category"].(string); ok && cat != "" {
		category = &cat
	}

	limit := 20
	if lim, ok := args["limit"].(float64); ok {
		limit = int(lim)
		if limit > 50 {
			limit = 50
		}
		if limit < 1 {
			limit = 1
		}
	}

	guidelines, err := searchGuidelines(searchTerm, tenantID, category, limit)
	if err != nil {
		return "", fmt.Errorf("failed to search guidelines: %w", err)
	}

	resultJSON, err := json.Marshal(guidelines)
	if err != nil {
		return "", fmt.Errorf("failed to marshal guidelines: %w", err)
	}

	return string(resultJSON), nil
}













