# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/analytics-query-service \
    ./cmd/server/main.go

# Runtime stage
FROM scratch

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy binary
COPY --from=builder /app/analytics-query-service /analytics-query-service

# Expose HTTP port
EXPOSE 8084

# Run as non-root user
USER 65534:65534

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/analytics-query-service"]

# Run the application
ENTRYPOINT ["/analytics-query-service"]
