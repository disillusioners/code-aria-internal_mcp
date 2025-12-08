# Implementation Plan for get_all_working_changes Tool

## Overview
Implement a new MCP tool `get_all_working_changes` that provides comprehensive view of all working directory changes with diffs in a single operation.

## Files to Modify

### 1. `cmd/mcp-git/mcp.go`
**Changes needed:**
- Line 76: Update operation type description to include `get_all_working_changes`
- Line 179: Add case for `get_all_working_changes` operation

```go
// Line 76 change:
"description": "Operation type: get_git_status, get_file_diff, get_commit_history, get_changed_files, get_all_working_changes",

// Line 179 add case:
case "get_all_working_changes":
    result, err = toolGetAllWorkingChanges(params)
```

### 2. `cmd/mcp-git/git_operations.go`
**Add new function:**

```go
// toolGetAllWorkingChanges returns all working directory changes with diffs
func toolGetAllWorkingChanges(args map[string]interface{}) (string, error) {
    repoPath := os.Getenv("REPO_PATH")
    if repoPath == "" {
        return "", fmt.Errorf("REPO_PATH not set")
    }

    // Parse optional parameters
    includeStatus := true
    if is, ok := args["include_status"].(bool); ok {
        includeStatus = is
    }

    format := "unified"
    if f, ok := args["format"].(string); ok {
        format = f
    }

    var filePatterns []string
    if patterns, ok := args["file_patterns"].([]interface{}); ok {
        filePatterns = make([]string, len(patterns))
        for i, p := range patterns {
            filePatterns[i] = p.(string)
        }
    }

    // Get changed files
    changedFilesJSON, err := getChangedFilesWorking(repoPath, includeStatus)
    if err != nil {
        return "", fmt.Errorf("failed to get changed files: %w", err)
    }

    var changedFiles []map[string]interface{}
    if err := json.Unmarshal([]byte(changedFilesJSON), &changedFiles); err != nil {
        return "", fmt.Errorf("failed to parse changed files: %w", err)
    }

    // Filter by file patterns if provided
    var filteredFiles []map[string]interface{}
    for _, file := range changedFiles {
        filePath := file["file_path"].(string)
        
        // Check if file matches any pattern
        if len(filePatterns) == 0 {
            filteredFiles = append(filteredFiles, file)
            continue
        }
        
        matched := false
        for _, pattern := range filePatterns {
            if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
                matched = true
                break
            }
            // Also support path patterns
            if matched, _ := filepath.Match(pattern, filePath); matched {
                matched = true
                break
            }
        }
        
        if matched {
            filteredFiles = append(filteredFiles, file)
        }
    }

    // Generate diffs for each file if format is "unified"
    var resultFiles []map[string]interface{}
    summary := map[string]int{
        "total_files": 0,
        "modified":    0,
        "added":       0,
        "deleted":     0,
    }

    for _, file := range filteredFiles {
        filePath := file["file_path"].(string)
        status := ""
        if includeStatus {
            if s, ok := file["status"].(string); ok {
                status = s
            }
        }

        fileResult := map[string]interface{}{
            "file_path": filePath,
        }
        
        if includeStatus {
            fileResult["status"] = status
        }

        // Generate diff if format is "unified"
        if format == "unified" {
            diff, err := getWorkingDirDiff(repoPath, filePath)
            if err != nil {
                diff = fmt.Sprintf("Error generating diff: %s", err.Error())
            }
            fileResult["diff"] = diff
        }

        resultFiles = append(resultFiles, fileResult)

        // Update summary
        summary["total_files"] = summary["total_files"].(int) + 1
        switch status {
        case "M":
            summary["modified"] = summary["modified"].(int) + 1
        case "A":
            summary["added"] = summary["added"].(int) + 1
        case "D":
            summary["deleted"] = summary["deleted"].(int) + 1
        case "?":
            summary["added"] = summary["added"].(int) + 1
        }
    }

    // Build final result
    finalResult := map[string]interface{}{
        "changed_files": resultFiles,
        "summary":      summary,
    }

    resultJSON, err := json.Marshal(finalResult)
    if err != nil {
        return "", fmt.Errorf("failed to marshal result: %w", err)
    }

    return string(resultJSON), nil
}
```

## Implementation Steps

1. **Update mcp.go** to include new operation type
2. **Add toolGetAllWorkingChanges function** to git_operations.go
3. **Test with various scenarios**:
   - No file patterns (all files)
   - With file patterns (*.go, *.py)
   - Different formats (unified, summary)
   - With/without status
4. **Error handling validation**:
   - Invalid repository path
   - File read errors
   - Pattern matching errors
5. **Documentation updates**

## Expected Usage Examples

### Basic Usage
```json
{
  "operations": [
    {
      "type": "get_all_working_changes"
    }
  ]
}
```

### With File Patterns
```json
{
  "operations": [
    {
      "type": "get_all_working_changes",
      "file_patterns": ["*.go", "*.py"]
    }
  ]
}
```

### Summary Format Only
```json
{
  "operations": [
    {
      "type": "get_all_working_changes",
      "format": "summary",
      "include_status": true
    }
  ]
}
```

## Expected Output

```json
{
  "results": [
    {
      "operation": "get_all_working_changes",
      "status": "Success",
      "result": {
        "changed_files": [
          {
            "file_path": "src/main.go",
            "status": "M",
            "diff": "--- a/src/main.go\n+++ b/src/main.go\n@@ -1,3 +1,3 @@\n package main\n\n-fmt.Println(\"Hello\")\n+fmt.Println(\"Hello Code-Aria\")\n"
          },
          {
            "file_path": "src/utils.go",
            "status": "A",
            "diff": "--- /dev/null\n+++ b/src/utils.go\n@@ -0,0 +1,5 @@\n+package main\n+\n+func helper() {\n+\n}\n"
          }
        ],
        "summary": {
          "total_files": 2,
          "modified": 1,
          "added": 1,
          "deleted": 0
        }
      }
    }
  ]
}
```

## Benefits

1. **Single API Call**: Get all changes in one operation
2. **Token Efficient**: Reduces multiple round trips
3. **Flexible Filtering**: Support for file patterns
4. **Comprehensive**: Files, status, and diffs in one response
5. **Consistent**: Follows existing MCP patterns
6. **Error Resilient**: Continues processing if individual files fail

## Testing Scenarios

1. **Empty working directory**: Should return empty arrays
2. **Only staged changes**: Should return empty (only worktree changes)
3. **Mixed changes**: Modified, added, deleted files
4. **File pattern filtering**: Only matching files included
5. **Large diffs**: Handle files with many changes
6. **Binary files**: Skip or handle gracefully
7. **Permission errors**: Continue with other files