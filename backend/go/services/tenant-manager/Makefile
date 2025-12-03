# =============================================================================
# Tenant Management Service - Makefile
# =============================================================================

# Variables
SERVICE_NAME := tenant-manager
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Go settings
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Docker settings
DOCKER_REGISTRY ?= ghcr.io/voicecustomer
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(SERVICE_NAME)

# Database settings
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/tenant_management?sslmode=disable
MIGRATIONS_PATH := ./migrations

# Build flags
LDFLAGS := -ldflags "-s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GoVersion=$(GO_VERSION)"

.PHONY: all build clean test lint fmt vet run docker-build docker-push migrate-up migrate-down generate help

# Default target
all: lint test build

# =============================================================================
# Build targets
# =============================================================================

## build: Build the service binary
build:
	@echo "Building $(SERVICE_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build $(LDFLAGS) -o bin/$(SERVICE_NAME) ./cmd/server

## build-linux: Build for Linux (for Docker)
build-linux:
	@echo "Building $(SERVICE_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build $(LDFLAGS) -o bin/$(SERVICE_NAME)-linux-amd64 ./cmd/server

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean -cache -testcache

# =============================================================================
# Development targets
# =============================================================================

## run: Run the service locally
run:
	@echo "Running $(SERVICE_NAME)..."
	go run ./cmd/server

## run-dev: Run with hot reload (requires air)
run-dev:
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@which goimports > /dev/null && goimports -w . || true

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

## tidy: Run go mod tidy
tidy:
	@echo "Tidying modules..."
	go mod tidy

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

# =============================================================================
# Test targets
# =============================================================================

## test: Run all tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

## test-short: Run tests without integration tests
test-short:
	@echo "Running short tests..."
	go test -v -short ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# =============================================================================
# Database targets
# =============================================================================

## migrate-up: Run database migrations
migrate-up:
	@echo "Running migrations..."
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back migration..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

## migrate-drop: Drop all migrations
migrate-drop:
	@echo "Dropping all migrations..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" drop -f

## migrate-create: Create a new migration (usage: make migrate-create NAME=migration_name)
migrate-create:
	@echo "Creating migration $(NAME)..."
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

## migrate-status: Show migration status
migrate-status:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

# =============================================================================
# Code generation targets
# =============================================================================

## generate: Run go generate
generate:
	@echo "Running go generate..."
	go generate ./...

## generate-openapi: Generate OpenAPI spec from code annotations
generate-openapi:
	@echo "Generating OpenAPI spec..."
	@which swag > /dev/null || go install github.com/swaggo/swag/cmd/swag@latest
	swag init -g cmd/server/main.go -o api/openapi --parseDependency --parseInternal

## generate-proto: Generate gRPC code from proto files
generate-proto:
	@echo "Generating gRPC code..."
	@which protoc-gen-go > /dev/null || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@which protoc-gen-go-grpc > /dev/null || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/*.proto

## generate-mocks: Generate mocks for testing
generate-mocks:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	go generate ./internal/...

# =============================================================================
# Docker targets
# =============================================================================

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-t $(DOCKER_IMAGE):latest \
		.

## docker-push: Push Docker image to registry
docker-push:
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

## docker-run: Run Docker container locally
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it \
		-p 8080:8080 \
		-p 9090:9090 \
		-p 9091:9091 \
		--env-file .env \
		$(DOCKER_IMAGE):latest

## docker-compose-up: Start all services with Docker Compose
docker-compose-up:
	docker-compose up -d

## docker-compose-down: Stop all services
docker-compose-down:
	docker-compose down

## docker-compose-logs: View logs
docker-compose-logs:
	docker-compose logs -f $(SERVICE_NAME)

# =============================================================================
# Security targets
# =============================================================================

## security-scan: Run security scanners
security-scan:
	@echo "Running security scan..."
	@which gosec > /dev/null || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -fmt json -out security-report.json ./...

## vuln-check: Check for known vulnerabilities
vuln-check:
	@echo "Checking for vulnerabilities..."
	@which govulncheck > /dev/null || go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# =============================================================================
# Help
# =============================================================================

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -E 's/## /  /'
