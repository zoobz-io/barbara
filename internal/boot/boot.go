// Package boot provides per-concern infrastructure connection functions.
//
// Each function builds one client and returns it. Callers own lifecycle —
// defer Close/Shutdown on returned clients. Callers emit startup events
// after successful connection.
package boot

import (
	"context"
	"fmt"
	"os"

	"github.com/zoobz-io/aperture"
	"github.com/zoobz-io/capitan"

	intotel "github.com/zoobz-io/barbara/internal/otel"
)

// Database creates a PostgreSQL connection from config.
// Uncomment once config.Database exists.
//
// func Database(ctx context.Context) (*sqlx.DB, error) {
// 	cfg := sum.MustUse[config.Database](ctx)
// 	db, err := sqlx.Connect("postgres", cfg.DSN())
// 	if err != nil {
// 		return nil, fmt.Errorf("connecting to database: %w", err)
// 	}
// 	return db, nil
// }

// OTEL creates OpenTelemetry providers. serviceName identifies the process
// in traces and metrics.
//
// Endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT for now; switch to
// sum.MustUse[config.OTEL] once the config type exists.
func OTEL(ctx context.Context, serviceName string) (*intotel.Providers, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4318"
	}
	providers, err := intotel.New(ctx, intotel.Config{
		Endpoint:    endpoint,
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("creating otel providers: %w", err)
	}
	return providers, nil
}

// Aperture creates an aperture bridge from capitan events to OTEL providers.
func Aperture(_ context.Context, providers *intotel.Providers) (*aperture.Aperture, error) {
	ap, err := aperture.New(
		capitan.Default(),
		providers.Log,
		providers.Metric,
		providers.Trace,
	)
	if err != nil {
		return nil, fmt.Errorf("creating aperture: %w", err)
	}
	return ap, nil
}
