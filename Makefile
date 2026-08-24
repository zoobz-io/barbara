.PHONY: build run test test-unit test-integration test-bench lint lint-fix coverage clean help check ci install-tools install-hooks dev dev-down dev-logs dev-reset

.DEFAULT_GOAL := help

APP_NAME := Sumatra
BIN_DIR := bin

help: ## Display available commands
	@echo "$(APP_NAME) Development Commands"
	@echo "================================"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# =============================================================================
# Build & Run
# =============================================================================

build: ## Build the application binary
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/app

run: ## Run the application locally
	@go run ./cmd/app

# =============================================================================
# Docker Development Environment
# =============================================================================

dev: ## Start development environment (docker compose)
	@docker compose up -d
	@echo ""
	@echo "Services started:"
	@echo "  App:        http://localhost:8080"
	@echo "  Grafana:    http://localhost:3000"
	@echo "  Jaeger:     http://localhost:16686"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  MinIO:      http://localhost:9001"
	@echo ""
	@echo "Run 'make dev-logs' to tail application logs"

dev-down: ## Stop development environment
	@docker compose down

dev-logs: ## Tail application logs
	@docker compose logs -f app

dev-reset: ## Reset development environment (removes volumes)
	@docker compose down -v
	@echo "All volumes removed. Run 'make dev' to start fresh."

# =============================================================================
# Testing
# =============================================================================

test: ## Run all tests with race detector
	@go test -v -race -tags testing ./...

test-unit: ## Run unit tests only (short mode)
	@go test -v -race -tags testing -short ./...

test-integration: ## Run integration tests
	@go test -v -race -tags testing ./testing/integration/...

test-bench: ## Run benchmarks
	@go test -tags testing -bench=. -benchmem -benchtime=1s ./testing/benchmarks/...

# =============================================================================
# Code Quality
# =============================================================================

lint: ## Run linters
	@golangci-lint run --config=.golangci.yml --timeout=5m

lint-fix: ## Run linters with auto-fix
	@golangci-lint run --config=.golangci.yml --fix

coverage: ## Generate coverage report (unit + integration)
	@go test -tags testing -coverprofile=coverage-unit.out -covermode=atomic ./...
	@go test -tags testing -coverprofile=coverage-integration.out -covermode=atomic ./testing/integration/... 2>/dev/null || true
	@echo "mode: atomic" > coverage.out
	@tail -n +2 coverage-unit.out >> coverage.out
	@tail -n +2 coverage-integration.out >> coverage.out 2>/dev/null || true
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
	@echo "Coverage report: coverage.html"

# =============================================================================
# Maintenance
# =============================================================================

clean: ## Remove generated files
	@rm -rf $(BIN_DIR) tmp
	@rm -f coverage.out coverage.html coverage.txt coverage-unit.out coverage-integration.out
	@find . -name "*.test" -delete
	@find . -name "*.prof" -delete
	@find . -name "*.out" -delete

install-tools: ## Install development tools
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2
	@go install github.com/air-verse/air@latest

install-hooks: ## Install git pre-commit hook
	@mkdir -p .git/hooks
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make check' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

# =============================================================================
# CI
# =============================================================================

check: test lint ## Run tests and lint (quick validation)
	@echo "All checks passed!"

ci: clean lint test coverage test-bench ## Full CI simulation
	@echo "CI simulation complete!"
