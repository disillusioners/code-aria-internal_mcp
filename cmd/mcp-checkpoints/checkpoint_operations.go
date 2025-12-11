package main

import (
	"encoding/json"
	"fmt"
)

// toolCreateCheckpoint creates a checkpoint with the given name and description
func toolCreateCheckpoint(args map[string]interface{}) (string, error) {
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}

	description := ""
	if desc, ok := args["description"].(string); ok {
		description = desc
	}

	manager, err := NewCheckpointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	checkpoint, err := manager.CreateCheckpoint(name, description)
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return string(resultJSON), nil
}

// toolListCheckpoints returns all available checkpoints
func toolListCheckpoints(args map[string]interface{}) (string, error) {
	manager, err := NewCheckpointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	checkpoints, err := manager.ListCheckpoints()
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalIndent(checkpoints, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal checkpoints: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetCheckpoint returns a specific checkpoint by ID
func toolGetCheckpoint(args map[string]interface{}) (string, error) {
	id, ok := args["checkpoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("checkpoint_id is required")
	}

	manager, err := NewCheckpointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	checkpoint, err := manager.GetCheckpoint(id)
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return string(resultJSON), nil
}

// toolRestoreCheckpoint restores a checkpoint to the working directory
func toolRestoreCheckpoint(args map[string]interface{}) (string, error) {
	id, ok := args["checkpoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("checkpoint_id is required")
	}

	manager, err := NewCheckpointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	// Get checkpoint info for response before restoring
	checkpoint, err := manager.GetCheckpoint(id)
	if err != nil {
		return "", fmt.Errorf("checkpoint %s not found: %w", id, err)
	}

	if err := manager.RestoreCheckpoint(id); err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Checkpoint %s restored successfully", id),
		"checkpoint": checkpoint,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolDeleteCheckpoint removes a checkpoint
func toolDeleteCheckpoint(args map[string]interface{}) (string, error) {
	id, ok := args["checkpoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("checkpoint_id is required")
	}

	manager, err := NewCheckpointManager()
	if err != nil {
		return "", err
	}
	defer manager.Close()

	// Get checkpoint info for response before deleting
	checkpoint, err := manager.GetCheckpoint(id)
	if err != nil {
		return "", fmt.Errorf("checkpoint %s not found: %w", id, err)
	}

	if err := manager.DeleteCheckpoint(id); err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Checkpoint %s deleted successfully", id),
		"checkpoint": checkpoint,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// toolGetCheckpointInfo returns detailed information about a checkpoint
func toolGetCheckpointInfo(args map[string]interface{}) (string, error) {
	id, ok := args["checkpoint_id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("checkpoint_id is required")
	}

	manager, err := NewCheckpointManager()
	if err != nil {
		return "", err
	}

	checkpoint, err := manager.GetCheckpoint(id)
	if err != nil {
		return "", fmt.Errorf("checkpoint %s not found: %w", id, err)
	}

	// Get additional info like file count and total size
	fileCount := len(checkpoint.Files)

	result := map[string]interface{}{
		"checkpoint_id": checkpoint.ID,
		"name":          checkpoint.Name,
		"description":   checkpoint.Description,
		"timestamp":     checkpoint.Timestamp,
		"file_count":    fileCount,
		"total_size":    checkpoint.Size,
		"files":         checkpoint.Files,
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}