# MCP Checkpoints Server

A Model Context Protocol (MCP) server that provides checkpoint functionality for Git repositories. This server allows LLMs to create, manage, and restore checkpoints of working directory changes without committing them to Git.

## Features

- **Create Checkpoints**: Save the current working directory state with metadata
- **List Checkpoints**: View all available checkpoints with timestamps and descriptions
- **Restore Checkpoints**: Restore a specific checkpoint to the working directory
- **Delete Checkpoints**: Remove checkpoints that are no longer needed
- **Checkpoint Info**: Get detailed information about a checkpoint including file list

## Usage

### Environment Variables

- `REPO_PATH`: Path to the Git repository (required)

### Available Tools

#### create_checkpoint
Creates a checkpoint of the current working directory changes.

```json
{
  "name": "checkpoint_name",
  "description": "Optional description of the checkpoint"
}
```

#### list_checkpoints
Lists all available checkpoints.

```json
{}
```

#### get_checkpoint
Gets details of a specific checkpoint.

```json
{
  "checkpoint_id": "checkpoint_id"
}
```

#### restore_checkpoint
Restores a checkpoint to the working directory.

```json
{
  "checkpoint_id": "checkpoint_id"
}
```

#### delete_checkpoint
Deletes a checkpoint.

```json
{
  "checkpoint_id": "checkpoint_id"
}
```

#### get_checkpoint_info
Gets detailed information about a checkpoint including file list.

```json
{
  "checkpoint_id": "checkpoint_id"
}
```

## Storage

Checkpoints are stored in the `.mcp-checkpoints` directory within the repository:

```
.mcp-checkpoints/
├── abc12345/
│   ├── metadata.json
│   ├── src/
│   │   └── main.go
│   └── README.md
└── def67890/
    ├── metadata.json
    └── package.json
```

Each checkpoint contains:
- A unique 8-character ID
- Metadata file with checkpoint information
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

This server can be used alongside other MCP servers (mcp-git, mcp-filesystem, etc.) to provide a complete checkpoint management solution for AI-powered code generation workflows.