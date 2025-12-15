.PHONY: help build run test clean migrate-up migrate-down docker-build docker-run

# Variables
SERVICE_NAME=billing-service
DOCKER_IMAGE=serphona/$(SERVICE_NAME)
DOCKER_TAG=latest
GO_VERSION=1.24

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build the service
	go build -o bin/$(SERVICE_NAME) ./cmd/server

run: ## Run the service locally
	go run ./cmd/server/main.go

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@for file in migrations/*up.sql; do \
		echo "Applying $$file..."; \
		psql $(DATABASE_URL) -f $$file; \
	done

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@for file in migrations/*down.sql; do \
		echo "Reverting $$file..."; \
		psql $(DATABASE_URL) -f $$file; \
	done

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run -p 8081:8081 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: ## Push Docker image to registry
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

lint: ## Run linter
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	goimports -w .

dev: ## Run in development mode with hot reload
	air

.DEFAULT_GOAL := help
