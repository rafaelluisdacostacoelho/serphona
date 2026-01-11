# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set workspace to include shared libs
WORKDIR /workspace

# Use public Go proxy to improve download reliability
ENV GOPROXY=https://proxy.golang.org,direct

# Copy shared Go libs and service code (expects build context at repo root)
COPY backend/go/libs ./backend/go/libs
COPY backend/go/services/auth-gateway ./backend/go/services/auth-gateway

# Move into service module
WORKDIR /workspace/backend/go/services/auth-gateway

# Download dependencies (replacements resolve to ../../libs/*)
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /workspace/bin/auth-gateway \
    ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /workspace/bin/auth-gateway /app/auth-gateway
COPY --from=builder /workspace/backend/go/services/auth-gateway/docs /app/docs

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/auth-gateway"]
