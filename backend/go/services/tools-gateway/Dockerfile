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
COPY backend/go/services/tools-gateway ./backend/go/services/tools-gateway

# Move into service module
WORKDIR /workspace/backend/go/services/tools-gateway

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /workspace/bin/tools-gateway \
    ./cmd/server

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary and assets from builder
COPY --from=builder /workspace/bin/tools-gateway /app/tools-gateway
COPY --from=builder /workspace/backend/go/services/tools-gateway/migrations /app/migrations
COPY --from=builder /workspace/backend/go/services/tools-gateway/docs /app/docs

# Change ownership
RUN chown -R appuser:appuser /app

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/tools-gateway"]
