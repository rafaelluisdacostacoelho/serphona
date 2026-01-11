# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set workspace to include shared libs
WORKDIR /workspace

# Use public Go proxy to improve download reliability
ENV GOPROXY=https://goproxy.io,direct

# Copy shared Go libs and service code (expects build context at repo root)
COPY backend/go/libs ./backend/go/libs
COPY backend/go/services/billing-service ./backend/go/services/billing-service

# Move into service module
WORKDIR /workspace/backend/go/services/billing-service

# Download dependencies
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /workspace/bin/billing-service ./cmd/server

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates postgresql-client

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary and assets from builder
COPY --from=builder /workspace/bin/billing-service /app/billing-service
COPY --from=builder /workspace/backend/go/services/billing-service/migrations /app/migrations
COPY --from=builder /workspace/backend/go/services/billing-service/docs /app/docs

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./billing-service"]
