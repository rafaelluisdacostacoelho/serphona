.PHONY: help build run test clean docker-build docker-run migrate-up migrate-down

# Variables
APP_NAME=tools-gateway
DOCKER_IMAGE=serphona/$(APP_NAME)
VERSION?=latest
PORT?=8085

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
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
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
