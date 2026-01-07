.PHONY: help build run test clean docker-build docker-run migrate-up migrate-down

# Variables
APP_NAME=tools-gateway
DOCKER_IMAGE=serphona/$(APP_NAME)
VERSION?=latest
PORT?=8085
PKG ?= ./...
GO_COVER ?= go tool cover
COBERTURA_DIR ?= coverage
GOCOV ?= $(shell command -v gocov 2>/dev/null)
GOCOV_HTML ?= $(shell command -v gocov-html 2>/dev/null)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(APP_NAME) cmd/server/main.go
	@echo "Build complete: bin/$(APP_NAME)"

run: ## Run the application locally
	@echo "Running $(APP_NAME) on port $(PORT)..."
	@go run cmd/server/main.go

dev: ## Run in development mode with hot reload (requires air)
	@air

test: ## Run tests
	@echo "Running tests..."
	@go test -v $(PKG)

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COBERTURA_DIR)
	@go test -v -coverprofile=$(COBERTURA_DIR)/coverage.out $(PKG)
	@$(GO_COVER) -func=$(COBERTURA_DIR)/coverage.out
	@$(GO_COVER) -html=$(COBERTURA_DIR)/coverage.out -o $(COBERTURA_DIR)/coverage.html
	@echo "Coverage report generated: $(COBERTURA_DIR)/coverage.html"

coverage-tools-check:
	@command -v gocov >/dev/null || { \
		echo "gocov not found; install (Go >=1.22 recommended):"; \
		echo "  go install github.com/axw/gocov/gocov@v1.2.1"; \
		echo "  (se Go mais antigo, tente @v1.1.0)"; \
		exit 1; \
	}
	@command -v gocov-html >/dev/null || { \
		echo "gocov-html not found; install:"; \
		echo "  go install github.com/matm/gocov-html/cmd/gocov-html@v1.4.0"; \
		exit 1; \
	}

test-coverage-html: coverage-tools-check ## Run tests with gocov + gocov-html
	@mkdir -p $(COBERTURA_DIR)
	@go test -v -coverprofile=$(COBERTURA_DIR)/coverage.out $(PKG)
	@gocov convert $(COBERTURA_DIR)/coverage.out > $(COBERTURA_DIR)/coverage.json
	@gocov-html $(COBERTURA_DIR)/coverage.json > $(COBERTURA_DIR)/coverage.html
	@echo "Coverage report generated: $(COBERTURA_DIR)/coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf $(COBERTURA_DIR)
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE):$(VERSION)"
	@docker build -t $(DOCKER_IMAGE):$(VERSION) .
	@docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest
	@echo "Docker image built successfully"

docker-run: ## Run Docker container
	@echo "Running Docker container on port $(PORT)..."
	@docker run -d \
		--name $(APP_NAME) \
		-p $(PORT):8085 \
		--env-file .env \
		$(DOCKER_IMAGE):$(VERSION)
	@echo "Container started. View logs with: docker logs -f $(APP_NAME)"

docker-stop: ## Stop Docker container
	@echo "Stopping Docker container..."
	@docker stop $(APP_NAME) || true
	@docker rm $(APP_NAME) || true

docker-logs: ## View Docker container logs
	@docker logs -f $(APP_NAME)

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@psql $(DATABASE_URL) < migrations/001_create_tools_tables.up.sql
	@echo "Migrations complete"

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@psql $(DATABASE_URL) < migrations/001_create_tools_tables.down.sql
	@echo "Migrations rolled back"

db-create: ## Create database
	@echo "Creating database..."
	@createdb serphona_tools || echo "Database already exists"

db-drop: ## Drop database
	@echo "Dropping database..."
	@dropdb serphona_tools || echo "Database doesn't exist"

db-reset: db-drop db-create migrate-up ## Reset database

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

.DEFAULT_GOAL := help
