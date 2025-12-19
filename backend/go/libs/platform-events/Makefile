.PHONY: help test test-unit test-coverage test-coverage-html coverage-tools-check test-integration clean

GO_TEST ?= go test
GO_COVER ?= go tool cover
PKG ?= ./...
TEST_ARGS ?=
GOCOV ?= $(shell command -v gocov 2>/dev/null)
GOCOV_HTML ?= $(shell command -v gocov-html 2>/dev/null)

OS := $(shell uname -s 2>/dev/null || echo Windows)
INTEGRATION_RUNNER := ./test/integration/run-integration-tests.sh
ifeq ($(OS),Windows)
	INTEGRATION_RUNNER := test\\integration\\run-integration-tests.bat
endif

help: ## Show available targets
	@echo "platform-events targets:"
	@echo "  make test            Run unit tests"
	@echo "  make test-coverage   Run unit tests with coverage (coverage.out + coverage.html)"
	@echo "  make test-coverage-html Run unit tests with gocov + gocov-html (coverage.out + coverage.json + coverage.html)"
	@echo "  make test-integration [--with-compose] Run integration suite"
	@echo "  make clean           Remove coverage artifacts"

test: test-unit ## Run unit tests
test-unit: ## Run unit tests
	@$(GO_TEST) $(PKG)

test-coverage: ## Run unit tests with coverage report
	@$(GO_TEST) -coverprofile=coverage.out $(PKG)
	@$(GO_COVER) -func=coverage.out
	@$(GO_COVER) -html=coverage.out -o coverage.html

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

test-coverage-html: coverage-tools-check ## Run unit tests with richer HTML coverage (requires gocov + gocov-html)
	@$(GO_TEST) -coverprofile=coverage.out $(PKG)
	@gocov convert coverage.out > coverage.json
	@gocov-html coverage.json > coverage.html

test-integration: ## Run integration tests (Kafka required; pass --with-compose to start stack)
	@$(INTEGRATION_RUNNER) $(TEST_ARGS)

clean: ## Remove coverage artifacts
	@$(RM) coverage.out coverage.html coverage.json
