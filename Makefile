.PHONY: build run run-admin test test-unit test-integration test-bench lint lint-fix coverage clean help check ci setup install-tools install-hooks dev dev-api dev-admin dev-observability dev-down dev-logs dev-reset openapi-api openapi-admin web-install web-check web-lint web-test web-build

.DEFAULT_GOAL := help

APP_NAME := Barbara
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
	@go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/api

run: ## Run the public API locally (go run)
	@go run ./cmd/api

run-admin: ## Run the admin API locally (go run)
	@go run ./cmd/admin

# =============================================================================
# Docker Development Environment
# =============================================================================
# Services are profiled, so bring up only the surface you're working on. Each app
# profile pulls in the shared infra (Postgres, Redis, MinIO, OpenSearch,
# migrations); observability is a separate opt-in profile.

dev: ## Start both surfaces + shared dependencies
	@docker compose --profile api --profile admin up -d
	@echo ""
	@echo "  API:    http://localhost:8080"
	@echo "  Admin:  http://localhost:8081"
	@echo "  MinIO:  http://localhost:9001"
	@echo ""
	@echo "Run 'make dev-logs' to tail logs, 'make dev-observability' for telemetry."

dev-api: ## Start the public API + its dependencies
	@docker compose --profile api up -d
	@echo "API: http://localhost:8080  (make dev-logs to tail)"

dev-admin: ## Start the admin API + its dependencies
	@docker compose --profile admin up -d
	@echo "Admin: http://localhost:8081  (make dev-logs to tail)"

dev-observability: ## Start the optional telemetry stack (Grafana, Jaeger, Prometheus, Loki)
	@docker compose --profile observability up -d
	@echo ""
	@echo "  Grafana:    http://localhost:3000"
	@echo "  Jaeger:     http://localhost:16686"
	@echo "  Prometheus: http://localhost:9090"

dev-down: ## Stop the development environment (all profiles)
	@docker compose --profile api --profile admin --profile observability down

dev-logs: ## Tail running container logs
	@docker compose logs -f

dev-reset: ## Reset development environment (removes volumes)
	@docker compose --profile api --profile admin --profile observability down -v
	@echo "All volumes removed. Run 'make dev' to start fresh."

# =============================================================================
# Web Workspace
# =============================================================================
# The web/ pnpm workspace (Nuxt apps + generated SDKs) is driven from here so a
# Go-only checkout never needs to cd. Requires pnpm (see web/README.md).

openapi-api: ## Dump the public API OpenAPI spec into the api SDK package
	@go run ./cmd/apispec web/packages/api-sdk/data/openapi.json

openapi-admin: ## Dump the admin API OpenAPI spec into the admin SDK package
	@go run ./cmd/adminspec web/packages/admin-sdk/data/openapi.json

web-install: ## Install web workspace dependencies
	@cd web && pnpm install

web-check: ## Typecheck the web workspace
	@cd web && pnpm run typecheck

web-lint: ## Lint the web workspace
	@cd web && pnpm run lint

web-test: ## Run web workspace tests
	@cd web && pnpm run test

web-build: ## Build the web workspace (SDK packages + apps)
	@cd web && pnpm run build

# =============================================================================
# Testing
# =============================================================================

test: ## Run all tests: race detector + coverage profile (coverage.out)
	@# One pass produces both the race-checked result and the coverage profile,
	@# so tests are never run twice just to measure coverage. -coverpkg=./...
	@# attributes coverage to the production packages that the testing/integration
	@# suite exercises (integration tests live there, not in source). The Go
	@# toolchain is pinned to 1.25.3 in CI, which handles -coverpkg cleanly.
	@go test -race -tags testing -coverpkg=./... -covermode=atomic -coverprofile=coverage.out ./...

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

coverage: test ## Render the coverage report from the test run (coverage.html)
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

setup: ## Bootstrap the dev toolchain (Go, linters, air) — idempotent
	@bash tools/setup.sh

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

check: test lint web-check web-lint web-test ## Run Go + web tests, lint, and typecheck (quick validation)
	@echo "All checks passed!"

ci: clean lint test coverage test-bench web-check web-lint web-test web-build ## Full CI simulation
	@echo "CI simulation complete!"
