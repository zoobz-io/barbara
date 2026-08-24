# tools/dev

Development environment configuration.

## Files

| File | Purpose |
|------|---------|
| `Dockerfile.dev` | Development image with hot reload (air) |
| `otel-collector.yaml` | OTEL Collector pipeline configuration |
| `prometheus.yaml` | Prometheus scrape configuration |
| `grafana/provisioning/` | Grafana datasource auto-configuration |

## Observability Stack

The docker-compose.yml sets up a complete observability stack:

```
┌─────────────────────────────────────────────────────────────────┐
│                         Application                              │
│                              │                                   │
│                    OTLP HTTP (port 4318)                        │
│                              ▼                                   │
│                     ┌────────────────┐                          │
│                     │ OTEL Collector │                          │
│                     └────────────────┘                          │
│                       │     │     │                             │
│          ┌────────────┘     │     └────────────┐               │
│          ▼                  ▼                  ▼               │
│    ┌──────────┐      ┌──────────┐       ┌──────────┐          │
│    │  Jaeger  │      │   Loki   │       │Prometheus│          │
│    │ (traces) │      │  (logs)  │       │(metrics) │          │
│    └──────────┘      └──────────┘       └──────────┘          │
│          │                │                  │                 │
│          └────────────────┼──────────────────┘                 │
│                           ▼                                     │
│                     ┌──────────┐                                │
│                     │ Grafana  │                                │
│                     │  (UI)    │                                │
│                     └──────────┘                                │
└─────────────────────────────────────────────────────────────────┘
```

## Ports

| Service | Port | Purpose |
|---------|------|---------|
| App | 8080 | Application HTTP |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache |
| MinIO | 9000/9001 | Object storage / Console |
| OTEL Collector | 4318 | OTLP HTTP receiver |
| Jaeger | 16686 | Trace UI |
| Loki | 3100 | Log aggregation |
| Prometheus | 9090 | Metrics UI |
| Grafana | 3000 | Unified dashboard |

## Usage

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f app

# Stop all services
docker compose down

# Reset volumes (clean slate)
docker compose down -v
```

## Viewing Telemetry

- **Traces**: http://localhost:16686 (Jaeger)
- **Metrics**: http://localhost:9090 (Prometheus)
- **Logs**: Query via Grafana or Loki API
- **Unified**: http://localhost:3000 (Grafana)
