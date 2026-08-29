# tools/dev

Development environment configuration.

## Files

| File | Purpose |
|------|---------|
| `Dockerfile.dev` | Development image with hot reload (air) |
| `air.api.toml` | Hot-reload config for the public API (`cmd/api`) |
| `air.admin.toml` | Hot-reload config for the admin API (`cmd/admin`) |
| `otel-collector.yaml` | OTEL Collector pipeline configuration |
| `prometheus.yaml` | Prometheus scrape configuration |
| `grafana/provisioning/` | Grafana datasource auto-configuration |

## Running the surfaces

Barbara runs one binary per surface (public `api`, authoring `admin`). The
compose stack is **profiled** so you start only what you need — each app profile
pulls in the shared dependencies (Postgres, Redis, MinIO, OpenSearch, and the
one-shot migrate/bucket init); observability is a separate opt-in profile.

```bash
make dev-api            # public API (:8080) + dependencies, hot-reloaded
make dev-admin          # admin API (:8081) + dependencies, hot-reloaded
make dev                # both surfaces + shared dependencies
make dev-observability  # optional telemetry stack (Grafana/Jaeger/Prometheus/Loki)
make dev-logs           # tail running containers
make dev-down           # stop everything
make dev-reset          # stop and wipe volumes
```

To run a surface on the host instead of in a container (against a running
stack): `make run` (api) or `make run-admin` (admin).

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
| API | 8080 | Public (site-facing) API |
| Admin | 8081 | Admin (authoring) API |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache |
| MinIO | 9000/9001 | Object storage / Console |
| OpenSearch | 9200 | Search / serving store |
| OTEL Collector | 4318 | OTLP HTTP receiver |
| Jaeger | 16686 | Trace UI |
| Loki | 3100 | Log aggregation |
| Prometheus | 9090 | Metrics UI |
| Grafana | 3000 | Unified dashboard |

## Usage

Prefer the `make dev-*` targets above. The raw compose equivalents:

```bash
# Start a surface + its dependencies
docker compose --profile api up -d      # or --profile admin
docker compose --profile observability up -d

# View logs / stop / reset
docker compose logs -f
docker compose down
docker compose down -v
```

## OpenSearch smoke check

OpenSearch is the serving store — site-facing pages and search read it
exclusively. After `make dev`, confirm the cluster is up:

```bash
curl -s http://localhost:9200/_cluster/health | jq .
```

A healthy single-node cluster reports `"status":"green"` or `"yellow"`
(yellow is expected — the mapping requests one replica that a single node
can't assign; the index is fully functional). The index mappings themselves
are applied at boot by `EnsureIndices` from the embedded files under
`database/migrations/opensearch/`, not by goose.

## Viewing Telemetry

- **Traces**: http://localhost:16686 (Jaeger)
- **Metrics**: http://localhost:9090 (Prometheus)
- **Logs**: Query via Grafana or Loki API
- **Unified**: http://localhost:3000 (Grafana)
