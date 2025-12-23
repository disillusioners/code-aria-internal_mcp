# MCP Savepoints Server

A Model Context Protocol (MCP) server that provides savepoint functionality for Git repositories. This server allows LLMs to create, manage, and restore savepoints of working directory changes without committing them to Git.

## Features

- **Create Savepoints**: Save the current working directory state with metadata
- **List Savepoints**: View all available savepoints with timestamps and descriptions
- **Restore Savepoints**: Restore a specific savepoint to the working directory
- **Delete Savepoints**: Remove savepoints that are no longer needed
- **Savepoint Info**: Get detailed information about a savepoint including file list

## Usage

### Environment Variables

- `REPO_PATH`: Path to the Git repository (required)

### Available Tools

#### create_savepoint
Creates a savepoint of the current working directory changes.

```json
{
  "name": "savepoint_name",
  "description": "Optional description of the savepoint"
}
```

#### list_savepoints
Lists all available savepoints.

```json
{}
```

#### get_savepoint
Gets details of a specific savepoint.

```json
{
  "savepoint_id": "savepoint_id"
}
```

#### restore_savepoint
Restores a savepoint to the working directory.

```json
{
  "savepoint_id": "savepoint_id"
}
```

#### delete_savepoint
Deletes a savepoint.

```json
{
  "savepoint_id": "savepoint_id"
}
```

#### get_savepoint_info
Gets detailed information about a savepoint including file list.

```json
{
  "savepoint_id": "savepoint_id"
}
```

## Storage

Savepoints are stored in the `.mcp-savepoints` directory within the repository:

```
.mcp-savepoints/
├── abc12345/
│   ├── metadata.json
│   ├── src/
│   │   └── main.go
│   └── README.md
└── def67890/
    ├── metadata.json
    └── package.json
```

Each savepoint contains:
- A unique 8-character ID
- Metadata file with savepoint information
- Complete copies of all changed files

## Building

```bash
cd code-aria-internal_mcp
make build-mcp-servers
```

## Installation

```bash
cd code-aria-internal_mcp
make mcp-servers
```

## Integration

This server can be used alongside other MCP servers (mcp-git, mcp-filesystem, etc.) to provide a complete savepoint management solution for AI-powered code generation workflows.