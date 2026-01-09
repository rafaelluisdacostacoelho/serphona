.PHONY: help build run test test-coverage lint fmt tidy clean deps docker-build docker-run swagger

SERVICE_NAME=agent-orchestrator
DOCKER_IMAGE=serphona/$(SERVICE_NAME)
DOCKER_TAG?=latest
PKG ?= ./...
GO_COVER ?= go tool cover
COBERTURA_DIR ?= coverage
SWAG_MAIN ?= cmd/server/main.go
SWAG_OUT ?= docs
SWAG_BIN ?= $(shell command -v swag 2>/dev/null || echo "$(shell go env GOPATH)/bin/swag")

help: ## Mostra este help
	@echo 'Uso: make <alvo>'
	@echo ''
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Baixa dependencias
	go mod download
	go mod tidy

build: ## Compila o binario
	go build -o bin/$(SERVICE_NAME) ./cmd/server

run: ## Executa localmente
	go run ./cmd/server/main.go

test: ## Roda testes
	go test -v $(PKG)

test-coverage: ## Roda testes com cobertura
	@mkdir -p $(COBERTURA_DIR)
	go test -v -coverprofile=$(COBERTURA_DIR)/coverage.out $(PKG)
	$(GO_COVER) -func=$(COBERTURA_DIR)/coverage.out
	$(GO_COVER) -html=$(COBERTURA_DIR)/coverage.out -o $(COBERTURA_DIR)/coverage.html
	@echo "Relatorio: $(COBERTURA_DIR)/coverage.html"

lint: ## Roda linter
	golangci-lint run ./...

fmt: ## Formata codigo
	go fmt ./...
	goimports -w .

tidy: ## go mod tidy
	go mod tidy

clean: ## Limpa artefatos
	rm -rf bin/
	rm -rf $(COBERTURA_DIR)

swagger: ## Gera docs Swagger (requer swag)
	@SWAG_BIN=$(SWAG_BIN); \
	if [ ! -x "$$SWAG_BIN" ]; then \
		echo "swag nao encontrado; instalando..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		SWAG_BIN="$(shell go env GOPATH)/bin/swag"; \
	fi; \
	PATH="$(shell go env GOPATH)/bin:$$PATH" "$$SWAG_BIN" init -g $(SWAG_MAIN) -d cmd/server,internal -o $(SWAG_OUT)

docker-build: ## Build da imagem Docker
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Roda container Docker
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

.DEFAULT_GOAL := help
