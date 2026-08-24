// Package main is the entry point for the application.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	intotel "github.com/zoobz-io/sumatra/internal/otel"
	"github.com/zoobz-io/aperture"
	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/sum"
	"github.com/zoobz-io/sumatra/events"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Println("starting...")
	ctx := context.Background()

	// Initialize sum service and registry.
	svc := sum.New()
	k := sum.Start()

	// =========================================================================
	// 1. Load Configuration
	// =========================================================================

	// Load all configs via sum.Config[T]().
	// if err := sum.Config[config.App](ctx, k, nil); err != nil {
	// 	return fmt.Errorf("failed to load app config: %w", err)
	// }
	// if err := sum.Config[config.Database](ctx, k, nil); err != nil {
	// 	return fmt.Errorf("failed to load database config: %w", err)
	// }
	// if err := sum.Config[config.Observability](ctx, k, nil); err != nil {
	// 	return fmt.Errorf("failed to load observability config: %w", err)
	// }

	// =========================================================================
	// 2. Connect to Infrastructure
	// =========================================================================

	// Database
	// dbCfg := sum.MustUse[config.Database](ctx)
	// db, err := sqlx.Connect("postgres", dbCfg.DSN())
	// if err != nil {
	// 	return fmt.Errorf("failed to connect to database: %w", err)
	// }
	// defer func() { _ = db.Close() }()
	// log.Println("database connected")
	// capitan.Emit(ctx, events.StartupDatabaseConnected)

	// Storage (MinIO)
	// storageCfg := sum.MustUse[config.Storage](ctx)
	// minioClient, err := minio.New(storageCfg.Endpoint, &minio.Options{...})
	// bucketProvider := grubminio.New(minioClient, storageCfg.Bucket)
	// log.Println("storage connected")
	// capitan.Emit(ctx, events.StartupStorageConnected)

	// =========================================================================
	// 3. Create and Register Stores
	// =========================================================================

	// Import: "github.com/zoobz-io/sumatra/api/stores"
	// Import: "github.com/zoobz-io/sumatra/api/contracts"
	//
	// allStores, err := stores.New(db, renderer, bucketProvider)
	// if err != nil {
	// 	return fmt.Errorf("failed to create stores: %w", err)
	// }
	// sum.Register[contracts.YourContract](k, allStores.YourStore)

	// =========================================================================
	// 4. Register Boundaries
	// =========================================================================

	// Model boundaries
	// sum.NewBoundary[models.YourModel](k)

	// Wire boundaries
	// wire.RegisterBoundaries(k)

	// =========================================================================
	// 5. Freeze Registry
	// =========================================================================

	sum.Freeze(k)
	capitan.Emit(ctx, events.StartupServicesReady)

	// =========================================================================
	// 6. Initialize Observability (OTEL + Aperture)
	// =========================================================================

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "localhost:4318"
	}
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "Sumatra"
	}

	otelProviders, err := intotel.New(ctx, intotel.Config{
		Endpoint:    otelEndpoint,
		ServiceName: serviceName,
	})
	if err != nil {
		return fmt.Errorf("failed to create otel providers: %w", err)
	}
	defer func() { _ = otelProviders.Shutdown(ctx) }()
	log.Println("observability initialized")
	capitan.Emit(ctx, events.StartupOTELReady)

	// Initialize aperture to bridge capitan events → OTEL.
	ap, err := aperture.New(
		capitan.Default(),
		otelProviders.Log,
		otelProviders.Metric,
		otelProviders.Trace,
	)
	if err != nil {
		return fmt.Errorf("failed to create aperture: %w", err)
	}
	defer ap.Close()
	capitan.Emit(ctx, events.StartupApertureReady)

	// Optional: Apply aperture schema for metrics/traces configuration.
	// schema, err := aperture.LoadSchemaFromYAML(schemaBytes)
	// if err != nil {
	// 	return fmt.Errorf("failed to load aperture schema: %w", err)
	// }
	// if err := ap.Apply(schema); err != nil {
	// 	return fmt.Errorf("failed to apply aperture schema: %w", err)
	// }

	// =========================================================================
	// 7. Register Handlers and Run
	// =========================================================================

	// Import: "github.com/zoobz-io/sumatra/api/handlers"
	// svc.Handle(handlers.All()...)

	// appCfg := sum.MustUse[config.App](ctx)
	// capitan.Emit(ctx, events.StartupServerListening, events.StartupPortKey.Field(appCfg.Port))
	// log.Printf("starting server on port %d...", appCfg.Port)
	// return svc.Run("", appCfg.Port)

	_ = svc // Remove when using svc.Handle() above.
	_ = ap  // Remove when using ap.Apply() above.

	return fmt.Errorf("not implemented: add your initialization logic")
}
