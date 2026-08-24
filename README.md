# Barbara

A template repository for building Go applications with the zoobz-io framework.

## Overview

Barbara provides a production-ready project structure built on [sum](https://github.com/zoobz-io/sum), following patterns established in real-world applications. It includes:

- Type-safe service registry via sum
- HTTP server with OpenAPI support via rocco
- Database access patterns via grub/astql
- Configuration management via fig
- Event system via capitan
- Comprehensive testing infrastructure

## Project Structure

```
Barbara/
├── cmd/              # One binary per surface
│   ├── api/          #   Public API entrypoint
│   └── admin/        #   Admin API entrypoint
├── api/              # Public API surface: contracts, handlers, wire, transformers
├── admin/            # Admin API surface: contracts, handlers, wire, transformers
├── database/         # Data layer
│   ├── models/       #   Domain models
│   ├── stores/       #   Data access implementations (shared by all surfaces)
│   └── migrations/   #   SQL migrations
├── config/           # Configuration types
├── events/           # Event definitions
├── internal/
│   ├── boot/         #   Shared startup: infra connections, store construction
│   └── otel/         #   OpenTelemetry setup
├── testing/          # Test infrastructure
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

One domain, served by one binary per surface. Each surface (`api/`, `admin/`) has its own contracts, handlers, wire types, and transformers over a single shared data layer. Setup common to every binary lives in `internal/boot`; each binary registers its own surface's contracts and boundaries, then freezes the registry.

Layered with clear dependency rules:

1. **contracts** (per surface) - Define narrow interfaces, depend only on database/models
2. **database/models** - Domain models, no internal dependencies
3. **database/stores** - Implement contracts, depend on models; shared by all surfaces
4. **handlers** (per surface) - HTTP layer, depend on contracts/wire/transformers
5. **wire** (per surface) - API types, depend on models (for transformation)
6. **transformers** (per surface) - Pure mapping functions between models and wire

The same store satisfies multiple surface contracts — each contract exposes only what that surface needs. Multi-store writes with atomicity invariants live only as transactional methods on the stores aggregate.

## License

MIT
