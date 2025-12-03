.PHONY: help build run test test-coverage test-race clean fmt lint docker-build docker-run docker-stop docker-logs deps-update deps-download proto gen

# Variables
BINARY_NAME=voice-gateway
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
MAIN_PATH=./cmd/server/main.go
BUILD_DIR=./bin
DOCKER_IMAGE=serphona/voice-gateway
DOCKER_TAG=latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

## help: Display this help message
help:
	@echo "Voice Gateway - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build              Build the application"
	@echo "  make run                Run the application"
	@echo "  make watch              Run with auto-reload (requires air)"
	@echo "  make clean              Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  make test               Run tests"
	@echo "  make test-coverage      Run tests with coverage"
	@echo "  make test-race          Run tests with race detector"
	@echo "  make test-verbose       Run tests with verbose output"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt                Format code"
	@echo "  make lint               Run linter"
	@echo "  make vet                Run go vet"
	@echo "  make check              Run fmt, vet, and lint"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build       Build Docker image"
	@echo "  make docker-run         Run Docker container"
	@echo "  make docker-stop        Stop Docker container"
	@echo "  make docker-logs        View Docker logs"
	@echo "  make docker-compose-up  Start all services"
	@echo "  make docker-compose-down Stop all services"
	@echo ""
	@echo "Dependencies:"
	@echo "  make deps-download      Download dependencies"
	@echo "  make deps-update        Update dependencies"
	@echo "  make deps-tidy          Tidy dependencies"
	@echo ""

## build: Build the application binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## build-linux: Build for Linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_UNIX)"

## build-windows: Build for Windows
build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_WINDOWS) -v $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_WINDOWS)"

## build-all: Build for all platforms
build-all: build-linux build-windows
	@echo "All builds complete"

## run: Run the application
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run $(MAIN_PATH)

## watch: Run with auto-reload (requires air)
watch:
	@echo "Starting with auto-reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -race -short ./...

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./...

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

## fmt: Format all Go files
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
		exit 1; \
	fi

## check: Run fmt, vet, and lint
check: fmt vet lint
	@echo "All checks passed"

## deps-download: Download dependencies
deps-download:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "Dependencies downloaded"

## deps-update: Update dependencies
deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy
	@echo "Dependencies updated"

## deps-tidy: Tidy dependencies
deps-tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy
	@echo "Dependencies tidied"

## deps-verify: Verify dependencies
deps-verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify
	@echo "Dependencies verified"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

## docker-build-no-cache: Build Docker image without cache
docker-build-no-cache:
	@echo "Building Docker image (no cache)..."
	docker build --no-cache -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

## docker-run: Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -d \
		--name $(BINARY_NAME) \
		-p 8080:8080 \
		-p 9091:9091 \
		--env-file .env \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "Container started: $(BINARY_NAME)"

## docker-stop: Stop Docker container
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true
	@echo "Container stopped"

## docker-logs: View Docker logs
docker-logs:
	docker logs -f $(BINARY_NAME)

## docker-shell: Open shell in Docker container
docker-shell:
	docker exec -it $(BINARY_NAME) sh

## docker-compose-up: Start all services with docker-compose
docker-compose-up:
	@echo "Starting all services..."
	docker-compose up -d
	@echo "Services started. View logs with: make docker-compose-logs"

## docker-compose-down: Stop all services
docker-compose-down:
	@echo "Stopping all services..."
	docker-compose down
	@echo "Services stopped"

## docker-compose-logs: View docker-compose logs
docker-compose-logs:
	docker-compose logs -f

## docker-compose-build: Build and start all services
docker-compose-build:
	@echo "Building and starting all services..."
	docker-compose up --build -d
	@echo "Services built and started"

## docker-compose-restart: Restart voice-gateway service
docker-compose-restart:
	docker-compose restart voice-gateway

## install: Install the application
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(MAIN_PATH)
	@echo "Installation complete"

## version: Show version information
version:
	@echo "Voice Gateway"
	@echo "Go version: $$(go version)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build date: $$(date -u '+%Y-%m-%d %H:%M:%S')"

## health: Check if service is running
health:
	@echo "Checking service health..."
	@curl -f http://localhost:8080/health || echo "Service not running"

## dev: Start development environment
dev: docker-compose-up
	@echo "Development environment ready!"
	@echo "- Voice Gateway: http://localhost:8080"
	@echo "- Prometheus: http://localhost:9090"
	@echo "- Grafana: http://localhost:3000"
	@echo ""
	@echo "Run 'make run' to start the service locally"

## prod-check: Pre-production checklist
prod-check: check test test-race
	@echo "✅ Code formatted"
	@echo "✅ Linter passed"
	@echo "✅ Tests passed"
	@echo "✅ Race detector passed"
	@echo ""
	@echo "Ready for production!"

# Default target
.DEFAULT_GOAL := help
