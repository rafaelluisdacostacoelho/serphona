# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set workspace to include shared libs
WORKDIR /workspace

# Use public Go proxy to improve download reliability
ENV GOPROXY=https://goproxy.io,direct

# Copy shared Go libs and service code (expects build context at repo root)
COPY backend/go/libs ./backend/go/libs
COPY backend/go/services/agent-orchestrator ./backend/go/services/agent-orchestrator

# Move into service module
WORKDIR /workspace/backend/go/services/agent-orchestrator

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /workspace/bin/agent-orchestrator \
    ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary and assets from builder
COPY --from=builder /workspace/bin/agent-orchestrator /app/agent-orchestrator
COPY --from=builder /workspace/backend/go/services/agent-orchestrator/docs /app/docs
COPY --from=builder /workspace/backend/go/services/agent-orchestrator/migrations /app/migrations

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/agent-orchestrator"]
