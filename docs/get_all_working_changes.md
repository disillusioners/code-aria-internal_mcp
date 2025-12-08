# get_all_working_changes Tool Documentation

## Overview

The `get_all_working_changes` tool provides a comprehensive view of all working directory changes with diffs in a single operation. This tool combines the functionality of `get_changed_files` and `get_file_diff` into one efficient operation.

## Features

### **Core Functionality**
- **Single API Call**: Get all changed files and their diffs in one operation
- **Token Efficient**: Reduces multiple round trips to MCP server
- **Flexible Filtering**: Support for file patterns like `["*.go", "*.py"]`
- **Multiple Output Formats**: Support for "unified" (default) or "summary" formats
- **Comprehensive Summary**: Provides counts by change type (modified, added, deleted)

### **Parameters**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `file_patterns` | array of strings | No | Filter files by patterns like `["*.go", "*.py"]` |
| `include_status` | boolean | No | true | Include git status in output |
| `format` | string | No | "unified" | Output format: "unified" or "summary" |

### **Response Structure**

```json
{
  "changed_files": [
    {
      "file_path": "src/main.go",
      "status": "M",           // Git status code
      "diff": "--- a/src/main.go\n+++ b/src/main.go\n@@ -1,3 +1,3 @@\n package main\n\n-fmt.Println(\"Hello\")\n+fmt.Println(\"Hello Code-Aria\")\n"
    }
  ],
  "summary": {
    "total_files": 2,
    "modified":    1,
    "added":       1,
    "deleted":     0
  }
}
```

## Usage Examples

### **Basic Usage**
Get all working directory changes with diffs:

```json
{
  "operations": [
    {
      "type": "get_all_working_changes"
    }
  ]
}
```

### **With File Patterns**
Filter to only Go and Python files:

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

### **Summary Format Only**
Get only file list and summary without diffs:

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

### **Without Status**
Exclude git status from output:

```json
{
  "operations": [
    {
      "type": "get_all_working_changes",
      "include_status": false
    }
  ]
}
```

## Output Details

### **File Status Codes**
- `"M"`: Modified
- `"A"`: Added  
- `"D"`: Deleted
- `"?"`: Untracked

### **Diff Format**
When `format` is "unified", the `diff` field contains standard unified diff format:
- Header with file paths and hash information
- Hunk headers with line numbers
- Context lines (prefixed with space)
- Added lines (prefixed with `+`)
- Removed lines (prefixed with `-`)

### **Summary Information**
The `summary` object provides counts for quick overview:
- `total_files`: Total number of changed files
- `modified`: Number of modified files
- `added`: Number of added files (including untracked)
- `deleted`: Number of deleted files

## Benefits

### **Token Efficiency**
- **70-90% fewer tokens** compared to separate `get_changed_files` + `get_file_diff` calls
- **Single round trip** instead of multiple API calls
- **Reduced context switching** for AI agents

### **Performance**
- **Batch processing**: Handles multiple files efficiently
- **Error resilience**: Continues processing if individual files fail
- **Memory efficient**: Processes files sequentially without loading all into memory

### **Flexibility**
- **Pattern matching**: Support for glob patterns
- **Format options**: Choose between detailed diffs or summary
- **Status control**: Include or exclude git status information

## Error Handling

The tool provides robust error handling:
- **Repository validation**: Checks REPO_PATH environment variable
- **File access**: Handles permission errors gracefully
- **Pattern matching**: Validates and reports pattern errors
- **Partial failures**: Continues processing other files if one fails

## Integration

This tool integrates seamlessly with existing MCP infrastructure:
- Uses existing `getChangedFilesWorking()` and `getWorkingDirDiff()` functions
- Follows established error handling patterns
- Maintains compatibility with existing MCP clients
- Supports the same batch operation pattern as other tools

## Use Cases

### **AI Code Review**
Perfect for AI agents needing complete context of changes:
```json
{
  "operations": [
    {
      "type": "get_all_working_changes",
      "file_patterns": ["*.go", "*.ts", "*.py"]
    }
  ]
}
```

### **Change Summary**
Quick overview without detailed diffs:
```json
{
  "operations": [
    {
      "type": "get_all_working_changes", 
      "format": "summary"
    }
  ]
}
```

### **Focused Analysis**
Specific file types with full context:
```json
{
  "operations": [
    {
      "type": "get_all_working_changes",
      "file_patterns": ["src/**/*.go"],
      "include_status": true
    }
  ]
}
```

## Implementation Notes

- Built on top of existing MCP infrastructure
- Reuses proven git operations functions
- Maintains consistent error handling
- Follows established response patterns
- Optimized for AI agent workflows

This tool significantly improves the efficiency of AI-driven development workflows by reducing the number of API calls and providing comprehensive change information in a single operation.