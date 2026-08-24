# Sumatra

A template repository for building Go applications with the zoobz-io framework.

## Overview

Sumatra provides a production-ready project structure built on [sum](https://github.com/zoobz-io/sum), following patterns established in real-world applications. It includes:

- Type-safe service registry via sum
- HTTP server with OpenAPI support via rocco
- Database access patterns via grub/astql
- Configuration management via fig
- Event system via capitan
- Comprehensive testing infrastructure

## Project Structure

```
Sumatra/
├── cmd/app/          # Application entrypoint
├── config/           # Configuration types
├── contracts/        # Interface definitions
├── models/           # Domain models
├── stores/           # Data access implementations
├── handlers/         # HTTP handlers
├── wire/             # Request/response types
├── transformers/     # Model ↔ Wire mapping
├── events/           # Event definitions
├── testing/          # Test infrastructure
├── internal/otel/    # OpenTelemetry setup
├── migrations/       # SQL migrations
└── .github/workflows # CI/CD
```

Each directory contains a README explaining its purpose and usage patterns.

## Getting Started

```bash
# Install dependencies
go mod tidy

# Run the application
make run

# Run tests
make test

# Run linter
make lint

# Full CI check
make check
```

## Development

### Prerequisites

- Go 1.24+
- golangci-lint v2.7.2

### Install Tools

```bash
make install-tools
make install-hooks
```

### Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the application binary |
| `make run` | Run the application |
| `make test` | Run all tests with race detector |
| `make test-unit` | Run unit tests only |
| `make test-integration` | Run integration tests |
| `make test-bench` | Run benchmarks |
| `make lint` | Run linters |
| `make coverage` | Generate coverage report |
| `make check` | Run tests + lint |
| `make ci` | Full CI simulation |

## Architecture

The application follows a layered architecture with clear dependency rules:

1. **contracts** - Define interfaces, depend only on models
2. **models** - Domain models, no internal dependencies
3. **stores** - Implement contracts, depend on models
4. **handlers** - HTTP layer, depend on contracts/wire/transformers
5. **wire** - API types, depend on models (for transformation)
6. **transformers** - Pure mapping functions between models and wire

## License

MIT
