// Package otel provides OpenTelemetry provider setup.
package otel

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Providers holds the three OTEL provider types.
type Providers struct {
	Log    log.LoggerProvider
	Metric metric.MeterProvider
	Trace  trace.TracerProvider

	// For shutdown
	sdkLogProvider *sdklog.LoggerProvider
	sdkMetric      *sdkmetric.MeterProvider
	sdkTrace       *sdktrace.TracerProvider
}

// Config holds OTEL provider configuration.
type Config struct {
	Endpoint    string
	ServiceName string
}

// New creates OTEL providers configured to export to the given endpoint.
func New(ctx context.Context, cfg Config) (*Providers, error) {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "localhost:4318"
	}
	// Strip protocol prefix - OTLP exporters add it themselves.
	cfg.Endpoint = strings.TrimPrefix(cfg.Endpoint, "http://")
	cfg.Endpoint = strings.TrimPrefix(cfg.Endpoint, "https://")

	if cfg.ServiceName == "" {
		cfg.ServiceName = "Sumatra"
	}

	// Create shared resource with service name.
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
	)

	p := &Providers{}

	// Log provider.
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(cfg.Endpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating log exporter: %w", err)
	}
	logProcessor := sdklog.NewBatchProcessor(logExporter)
	p.sdkLogProvider = sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(logProcessor),
	)
	p.Log = p.sdkLogProvider

	// Metric provider.
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		_ = p.Shutdown(ctx)
		return nil, fmt.Errorf("creating metric exporter: %w", err)
	}
	metricReader := sdkmetric.NewPeriodicReader(metricExporter)
	p.sdkMetric = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(metricReader),
	)
	p.Metric = p.sdkMetric
	otel.SetMeterProvider(p.sdkMetric)

	// Trace provider.
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		_ = p.Shutdown(ctx)
		return nil, fmt.Errorf("creating trace exporter: %w", err)
	}
	traceProcessor := sdktrace.NewBatchSpanProcessor(traceExporter)
	p.sdkTrace = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(traceProcessor),
	)
	p.Trace = p.sdkTrace
	otel.SetTracerProvider(p.sdkTrace)

	return p, nil
}

// Shutdown gracefully shuts down all providers.
func (p *Providers) Shutdown(ctx context.Context) error {
	var errs []error

	if p.sdkLogProvider != nil {
		if err := p.sdkLogProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if p.sdkMetric != nil {
		if err := p.sdkMetric.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if p.sdkTrace != nil {
		if err := p.sdkTrace.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
