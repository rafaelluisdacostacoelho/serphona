.PHONY: help build run test clean migrate-up migrate-down docker-build docker-run

# Variables
SERVICE_NAME=billing-service
DOCKER_IMAGE=serphona/$(SERVICE_NAME)
DOCKER_TAG=latest
GO_VERSION=1.24
SEED_TEST_WALLETS?=false
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

deps: ## Download dependencies
	go mod download
	go mod tidy

build: ## Build the service
	go build -o bin/$(SERVICE_NAME) ./cmd/server

run: ## Run the service locally
	go run ./cmd/server/main.go

test: ## Run tests
	go test -v $(PKG)

test-coverage: ## Run tests with coverage
	@mkdir -p $(COBERTURA_DIR)
	go test -v -coverprofile=$(COBERTURA_DIR)/coverage.out $(PKG)
	$(GO_COVER) -func=$(COBERTURA_DIR)/coverage.out
	$(GO_COVER) -html=$(COBERTURA_DIR)/coverage.out -o $(COBERTURA_DIR)/coverage.html
	@echo "Coverage report: $(COBERTURA_DIR)/coverage.html"

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
	go test -v -coverprofile=$(COBERTURA_DIR)/coverage.out $(PKG)
	gocov convert $(COBERTURA_DIR)/coverage.out > $(COBERTURA_DIR)/coverage.json
	gocov-html $(COBERTURA_DIR)/coverage.json > $(COBERTURA_DIR)/coverage.html
	@echo "Coverage report: $(COBERTURA_DIR)/coverage.html"

test-integration: ## Run integration tests (requires integration build tag)
	go test -v -tags=integration $(PKG)

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf $(COBERTURA_DIR)

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@for file in migrations/*up.sql; do \
		echo "Applying $$file..."; \
		psql $(DATABASE_URL) -v seed_test_wallets=$(SEED_TEST_WALLETS) -f $$file; \
	done

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@for file in migrations/*down.sql; do \
		echo "Reverting $$file..."; \
		psql $(DATABASE_URL) -v seed_test_wallets=$(SEED_TEST_WALLETS) -f $$file; \
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
