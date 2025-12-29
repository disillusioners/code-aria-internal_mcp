# =============================================================================
# Code-Aria MCP Servers - Base Image with Python Runtime
# =============================================================================
# This image contains:
# - All 12 MCP server executables in /usr/local/bin
# - Python 3.12 runtime with build dependencies
# - Serves as the base for code-aria-langgraph image
# =============================================================================

# =============================================================================
# Stage 1: Builder - Build MCP servers
# =============================================================================
FROM golang:1.24-alpine AS builder

# Install git for go modules
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all 12 MCP servers with optimizations (static binaries, stripped)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-filesystem ./cmd/mcp-filesystem && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-codebase ./cmd/mcp-codebase && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-git ./cmd/mcp-git && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-code-edit ./cmd/mcp-code-edit && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-bash ./cmd/mcp-bash && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-powershell ./cmd/mcp-powershell && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-systeminfo ./cmd/mcp-systeminfo && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-savepoints ./cmd/mcp-savepoints && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-lang-go ./cmd/mcp-lang-go && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-guidelines ./cmd/mcp-guidelines && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-documents ./cmd/mcp-documents && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcp-postgres ./cmd/mcp-postgres && \
    echo "MCP servers built successfully"

# Verify all binaries were created
RUN ls -lh mcp-* && \
    md5sum mcp-filesystem mcp-codebase mcp-git mcp-code-edit mcp-bash mcp-powershell mcp-systeminfo mcp-savepoints mcp-lang-go mcp-guidelines mcp-documents mcp-postgres

# =============================================================================
# Stage 2: Runtime - Python with MCP executables
# =============================================================================
FROM python:3.12-slim AS runtime

# Install runtime and build dependencies
# - curl: for health checks
# - git: needed by MCP-git and general operations
# - build-essential: for Python package compilation
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    git \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN groupadd -g 1000 appgroup && \
    useradd -u 1000 -g appgroup -m -s /bin/bash appuser

# Copy MCP server executables from builder
COPY --from=builder /app/mcp-filesystem /usr/local/bin/
COPY --from=builder /app/mcp-codebase /usr/local/bin/
COPY --from=builder /app/mcp-git /usr/local/bin/
COPY --from=builder /app/mcp-code-edit /usr/local/bin/
COPY --from=builder /app/mcp-bash /usr/local/bin/
COPY --from=builder /app/mcp-powershell /usr/local/bin/
COPY --from=builder /app/mcp-systeminfo /usr/local/bin/
COPY --from=builder /app/mcp-savepoints /usr/local/bin/
COPY --from=builder /app/mcp-lang-go /usr/local/bin/
COPY --from=builder /app/mcp-guidelines /usr/local/bin/
COPY --from=builder /app/mcp-documents /usr/local/bin/
COPY --from=builder /app/mcp-postgres /usr/local/bin/

# Make MCP executables executable and verify
RUN chmod +x /usr/local/bin/mcp-* && \
    ls -lh /usr/local/bin/mcp-* && \
    echo "MCP servers installed to /usr/local/bin"

# Set working directory
WORKDIR /app

# Create directories for application code
RUN mkdir -p /app && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV PATH="/usr/local/bin:${PATH}"

# Default command (can be overridden)
CMD ["/bin/bash"]
