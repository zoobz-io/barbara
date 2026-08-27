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

	"github.com/jmoiron/sqlx"
	minio "github.com/minio/minio-go/v7"
	miniocreds "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zoobz-io/aperture"
	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/grub"
	grubminio "github.com/zoobz-io/grub/minio"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/config"
	intotel "github.com/zoobz-io/barbara/internal/otel"

	_ "github.com/lib/pq" // postgres driver
)

// Database opens a PostgreSQL connection from config. The caller owns the
// connection — defer Close.
func Database(ctx context.Context) (*sqlx.DB, error) {
	cfg := sum.MustUse[config.Database](ctx)
	db, err := sqlx.ConnectContext(ctx, "postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	return db, nil
}

// Bucket creates the object-storage provider from config (MinIO in dev,
// S3-compatible in prod). Assets are stored here; the provider is wrapped by an
// asset store when that capability lands.
func Bucket(ctx context.Context) (grub.BucketProvider, error) {
	cfg := sum.MustUse[config.Storage](ctx)
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  miniocreds.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}
	return grubminio.New(client, cfg.Bucket), nil
}

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
