package main

import (
	"encoding/json"
	"fmt"
)

// toolCreateSavepoint creates a savepoint with the given name and description
func toolCreateSavepoint(args map[string]interface{}) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	manager, err := NewSavepointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	savepoint, err := manager.CreateSavepoint(name, description)
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalIndent(savepoint, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal savepoint: %w", err)
	}

	return string(resultJSON), nil
}

// toolListSavepoints returns all available savepoints
func toolListSavepoints(args map[string]interface{}) (string, error) {
	manager, err := NewSavepointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	savepoints, err := manager.ListSavepoints()
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalIndent(savepoints, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal savepoints: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetSavepoint returns a specific savepoint by ID
func toolGetSavepoint(args map[string]interface{}) (string, error) {
	id, ok := args["savepoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("savepoint_id is required")
	}

	manager, err := NewSavepointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	savepoint, err := manager.GetSavepoint(id)
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalIndent(savepoint, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal savepoint: %w", err)
	}

	return string(resultJSON), nil
}

// toolRestoreSavepoint restores a savepoint to the working directory
func toolRestoreSavepoint(args map[string]interface{}) (string, error) {
	id, ok := args["savepoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("savepoint_id is required")
	}

	manager, err := NewSavepointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	// Get savepoint info for response before restoring
	savepoint, err := manager.GetSavepoint(id)
	if err != nil {
		return "", fmt.Errorf("savepoint %s not found: %w", id, err)
	}

	if err := manager.RestoreSavepoint(id); err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Savepoint %s restored successfully", id),
		"savepoint": savepoint,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolDeleteSavepoint removes a savepoint
func toolDeleteSavepoint(args map[string]interface{}) (string, error) {
	id, ok := args["savepoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("savepoint_id is required")
	}

	manager, err := NewSavepointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	// Get savepoint info for response before deleting
	savepoint, err := manager.GetSavepoint(id)
	if err != nil {
		return "", fmt.Errorf("savepoint %s not found: %w", id, err)
	}

	if err := manager.DeleteSavepoint(id); err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Savepoint %s deleted successfully", id),
		"savepoint": savepoint,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetSavepointInfo returns detailed information about a savepoint
func toolGetSavepointInfo(args map[string]interface{}) (string, error) {
	id, ok := args["savepoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("savepoint_id is required")
	}

	manager, err := NewSavepointManager()
	if err != nil {
		return "", err
	}

	savepoint, err := manager.GetSavepoint(id)
	if err != nil {
		return "", fmt.Errorf("savepoint %s not found: %w", id, err)
	}

	// Get additional info like file count and total size
	fileCount := len(savepoint.Files)

	result := map[string]interface{}{
		"savepoint_id": savepoint.ID,
		"name":         savepoint.Name,
		"description":  savepoint.Description,
		"timestamp":    savepoint.Timestamp,
		"file_count":   fileCount,
		"total_size":   savepoint.Size,
		"files":        savepoint.Files,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
