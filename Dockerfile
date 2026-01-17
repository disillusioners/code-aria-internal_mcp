# =============================================================================
# Code-Aria MCP Servers - Minimal Go-Only Image
# =============================================================================
# This image contains:
# - All 12 MCP server executables in /usr/local/bin
# - Minimal Alpine Linux runtime (no Python, no build tools)
# - Serves as the base for code-aria-langgraph image
# =============================================================================

# =============================================================================
# Stage 1: Builder - Build MCP servers (Go only)
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
# Stage 2: Runtime - Minimal Alpine (Go only, no Python)
# =============================================================================
FROM alpine:3.19 AS runtime

# Install only runtime dependencies
# - git: needed by MCP-git and general operations
# - ca-certificates: for HTTPS connections
RUN apk add --no-cache \
    git \
    ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

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

# Set ownership
RUN chown -R appuser:appgroup /usr/local/bin

# Set working directory
WORKDIR /home/appuser

# Switch to non-root user
USER appuser

# Set environment variables
ENV PATH="/usr/local/bin:${PATH}"

# Health check - verify MCP executables exist and are executable
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD sh -c 'test -x /usr/local/bin/mcp-filesystem && test -x /usr/local/bin/mcp-git && echo "MCP servers healthy"'

# Default command - list available MCP servers
CMD ["/bin/sh", "-c", "ls -la /usr/local/bin/mcp-*"]
