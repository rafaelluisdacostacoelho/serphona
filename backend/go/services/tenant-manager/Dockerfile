# =============================================================================
# Tenant Management Service - Multi-stage Dockerfile
# =============================================================================
# Build: docker build -t tenant-management:latest .
# Run:   docker run -p 8080:8080 -p 9090:9090 tenant-management:latest
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build
# -----------------------------------------------------------------------------
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user for runtime
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o /build/bin/tenant-management \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
FROM scratch

# Import from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Copy binary
COPY --from=builder /build/bin/tenant-management /app/tenant-management

# Copy migrations (if running migrations from container)
COPY --from=builder /build/migrations /app/migrations

# Use non-root user
USER appuser

# Set working directory
WORKDIR /app

# Expose ports
# 8080 - HTTP API
# 9090 - gRPC API
# 9091 - Prometheus metrics
EXPOSE 8080 9090 9091

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/tenant-management", "healthcheck"] || exit 1

# Default environment variables
ENV SERVER_HOST=0.0.0.0 \
    SERVER_PORT=8080 \
    GRPC_PORT=9090 \
    METRICS_PORT=9091 \
    LOG_LEVEL=info \
    ENVIRONMENT=production

# Run the binary
ENTRYPOINT ["/app/tenant-management"]
