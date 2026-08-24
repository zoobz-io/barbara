# internal/otel

OpenTelemetry provider setup.

## Purpose

Configure and initialize OpenTelemetry providers for logging, metrics, and tracing. Exports to an OTEL Collector via OTLP HTTP.

## Usage

```go
import intotel "github.com/zoobz-io/sumatra/internal/otel"

// Create providers
otelProviders, err := intotel.New(ctx, intotel.Config{
    Endpoint:    "localhost:4318",  // OTEL Collector
    ServiceName: "Sumatra",
})
if err != nil {
    return fmt.Errorf("failed to create otel providers: %w", err)
}
defer func() { _ = otelProviders.Shutdown(ctx) }()

// Use with aperture to bridge capitan events
ap, err := aperture.New(
    capitan.Default(),
    otelProviders.Log,
    otelProviders.Metric,
    otelProviders.Trace,
)
```

## Configuration

| Field | Description | Default |
|-------|-------------|---------|
| `Endpoint` | OTLP HTTP endpoint | `localhost:4318` |
| `ServiceName` | Service name in telemetry | `Sumatra` |

## Environment Variables

The application reads these for configuration:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
OTEL_SERVICE_NAME=Sumatra
```

## Providers

- **Log**: `log.LoggerProvider` for structured logging
- **Metric**: `metric.MeterProvider` for counters, gauges, histograms
- **Trace**: `trace.TracerProvider` for distributed tracing

All providers export via OTLP HTTP to a collector, which routes to:
- Traces → Jaeger
- Logs → Loki
- Metrics → Prometheus

## Shutdown

Always defer shutdown to flush pending telemetry:

```go
defer func() { _ = otelProviders.Shutdown(ctx) }()
```
