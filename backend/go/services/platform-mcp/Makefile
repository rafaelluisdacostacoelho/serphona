.PHONY: help build run test test-coverage lint fmt tidy clean

BINARY=platform-mcp
MAIN=./cmd/server/main.go
PKG=./...
COVERAGE_DIR=coverage

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
