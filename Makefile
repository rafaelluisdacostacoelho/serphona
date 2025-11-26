# ==============================================================================
# SERPHONA - Makefile
# ==============================================================================

.PHONY: help dev down build test lint clean

# Colors
CYAN := \033[36m
GREEN := \033[32m
YELLOW := \033[33m
RESET := \033[0m

help: ## Show this help
	@echo "$(CYAN)Serphona - Voice of Customer Platform$(RESET)"
	@echo ""
	@echo "$(YELLOW)Usage:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}'

# ==============================================================================
# DEVELOPMENT
# ==============================================================================

dev: ## Start full development environment
	docker-compose -f docker-compose.dev.yml up -d
	@echo "$(GREEN)Infrastructure started!$(RESET)"
	@echo ""
	@echo "Services:"
	@echo "  PostgreSQL:  localhost:5432"
	@echo "  ClickHouse:  localhost:8123"
	@echo "  Kafka:       localhost:9092"
	@echo "  Redis:       localhost:6379"
	@echo "  MinIO:       localhost:9000"
	@echo "  MinIO Console: localhost:9001"

down: ## Stop development environment
	docker-compose -f docker-compose.dev.yml down

logs: ## Show logs from dev environment
	docker-compose -f docker-compose.dev.yml logs -f

ps: ## Show running containers
	docker-compose -f docker-compose.dev.yml ps

# ==============================================================================
# FRONTEND
# ==============================================================================

.PHONY: frontend-install frontend-dev frontend-build frontend-lint frontend-test

frontend-install: ## Install frontend dependencies
	cd frontend/console && npm install

frontend-dev: ## Start frontend dev server
	cd frontend/console && npm run dev

frontend-build: ## Build frontend for production
	cd frontend/console && npm run build

frontend-lint: ## Lint frontend code
	cd frontend/console && npm run lint

frontend-test: ## Run frontend tests
	cd frontend/console && npm run test

# ==============================================================================
# BACKEND GO
# ==============================================================================

.PHONY: go-tidy go-build go-test go-lint

GO_SERVICES := agent-orchestrator tools-gateway tenant-manager analytics-query-service billing-service auth-gateway

go-tidy: ## Tidy Go modules
	@for svc in $(GO_SERVICES); do \
		echo "$(CYAN)Tidying $$svc...$(RESET)"; \
		cd backend/go/services/$$svc && go mod tidy && cd -; \
	done

go-build: ## Build all Go services
	@for svc in $(GO_SERVICES); do \
		echo "$(CYAN)Building $$svc...$(RESET)"; \
		cd backend/go/services/$$svc && go build -o bin/server ./cmd/server && cd -; \
	done

go-test: ## Run Go tests
	@for svc in $(GO_SERVICES); do \
		echo "$(CYAN)Testing $$svc...$(RESET)"; \
		cd backend/go/services/$$svc && go test ./... && cd -; \
	done

go-lint: ## Lint Go code
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest)
	@for svc in $(GO_SERVICES); do \
		echo "$(CYAN)Linting $$svc...$(RESET)"; \
		cd backend/go/services/$$svc && golangci-lint run && cd -; \
	done

# Run specific Go service
run-tenant-manager: ## Run tenant-manager service
	cd backend/go/services/tenant-manager && go run cmd/server/main.go

run-auth-gateway: ## Run auth-gateway service
	cd backend/go/services/auth-gateway && go run cmd/server/main.go

run-billing-service: ## Run billing-service
	cd backend/go/services/billing-service && go run cmd/server/main.go

# ==============================================================================
# BACKEND PYTHON
# ==============================================================================

.PHONY: py-install py-test py-lint

PYTHON_SERVICES := analytics-processor-service reporting-export-service

py-install: ## Install Python dependencies
	@for svc in $(PYTHON_SERVICES); do \
		echo "$(CYAN)Installing $$svc dependencies...$(RESET)"; \
		cd backend/python/$$svc && pip install -r requirements.txt && cd -; \
	done

py-test: ## Run Python tests
	@for svc in $(PYTHON_SERVICES); do \
		echo "$(CYAN)Testing $$svc...$(RESET)"; \
		cd backend/python/$$svc && pytest && cd -; \
	done

py-lint: ## Lint Python code
	@which ruff > /dev/null || pip install ruff
	@for svc in $(PYTHON_SERVICES); do \
		echo "$(CYAN)Linting $$svc...$(RESET)"; \
		cd backend/python/$$svc && ruff check . && cd -; \
	done

run-analytics-processor: ## Run analytics-processor service
	cd backend/python/analytics-processor-service && python -m analytics_processor.main

# ==============================================================================
# BUILD & DEPLOY
# ==============================================================================

.PHONY: build docker-build docker-push

build: go-build frontend-build ## Build all services
	@echo "$(GREEN)All services built!$(RESET)"

docker-build: ## Build all Docker images
	@echo "$(CYAN)Building Docker images...$(RESET)"
	docker build -t serphona/tenant-manager:latest backend/go/services/tenant-manager
	docker build -t serphona/auth-gateway:latest backend/go/services/auth-gateway
	docker build -t serphona/billing-service:latest backend/go/services/billing-service
	docker build -t serphona/analytics-processor:latest backend/python/analytics-processor-service
	docker build -t serphona/frontend-console:latest frontend/console
	@echo "$(GREEN)Docker images built!$(RESET)"

# ==============================================================================
# TESTING
# ==============================================================================

test: go-test py-test frontend-test ## Run all tests
	@echo "$(GREEN)All tests passed!$(RESET)"

lint: go-lint py-lint frontend-lint ## Lint all code
	@echo "$(GREEN)Linting complete!$(RESET)"

# ==============================================================================
# DATABASE
# ==============================================================================

.PHONY: db-migrate db-seed db-reset

db-migrate: ## Run database migrations
	cd backend/go/services/tenant-manager && go run cmd/migrate/main.go up

db-seed: ## Seed database with sample data
	cd backend/go/services/tenant-manager && go run cmd/seed/main.go

db-reset: ## Reset database (drop + migrate + seed)
	cd backend/go/services/tenant-manager && go run cmd/migrate/main.go down
	cd backend/go/services/tenant-manager && go run cmd/migrate/main.go up
	cd backend/go/services/tenant-manager && go run cmd/seed/main.go

# ==============================================================================
# INFRASTRUCTURE
# ==============================================================================

.PHONY: tf-init tf-plan tf-apply

tf-init: ## Initialize Terraform
	cd infra/terraform/envs/dev && terraform init

tf-plan: ## Plan Terraform changes
	cd infra/terraform/envs/dev && terraform plan

tf-apply: ## Apply Terraform changes
	cd infra/terraform/envs/dev && terraform apply

# ==============================================================================
# CLEANUP
# ==============================================================================

clean: ## Clean build artifacts
	rm -rf backend/go/services/*/bin
	rm -rf frontend/console/dist
	rm -rf backend/python/*/__pycache__
	rm -rf backend/python/*/.pytest_cache
	@echo "$(GREEN)Cleaned!$(RESET)"
