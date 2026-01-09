.PHONY: help build run test test-coverage lint fmt tidy clean swagger

BINARY=platform-mcp
MAIN=./cmd/server/main.go
PKG=./...
COVERAGE_DIR=coverage
SWAG_OUT ?= docs
SWAG_BIN ?= $(shell command -v swag 2>/dev/null || echo "$(shell go env GOPATH)/bin/swag")

help:
	@echo "Platform MCP - commands"
	@echo "  make build           Build binary"
	@echo "  make run             Run server"
	@echo "  make test            Run tests"
	@echo "  make test-coverage   Run tests with coverage report"
	@echo "  make lint            Run go vet"
	@echo "  make fmt             gofmt all files"
	@echo "  make tidy            go mod tidy"
	@echo "  make clean           Remove build and coverage artifacts"

build:
	@echo "Building $(BINARY)..."
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY) $(MAIN)

run:
	go run $(MAIN)

fmt:
	gofmt -w ./

lint:
	go vet $(PKG)

test:
	go test $(PKG)

test-coverage:
	mkdir -p $(COVERAGE_DIR)
	go test -coverprofile=$(COVERAGE_DIR)/coverage.out $(PKG)
	go tool cover -func=$(COVERAGE_DIR)/coverage.out

tidy:
	go mod tidy

clean:
	rm -rf bin $(COVERAGE_DIR)

swagger:
	@SWAG_BIN=$(SWAG_BIN); \
	if [ ! -x "$$SWAG_BIN" ]; then \
		echo "swag nao encontrado; instalando..."; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
		SWAG_BIN="$(shell go env GOPATH)/bin/swag"; \
	fi; \
	PATH="$(shell go env GOPATH)/bin:$$PATH" "$$SWAG_BIN" init -g $(MAIN) -d cmd/server,internal -o $(SWAG_OUT)
