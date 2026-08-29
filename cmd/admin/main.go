// Package main is the entry point for the admin, authoring HTTP API.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/admin/contracts"
	"github.com/zoobz-io/barbara/admin/handlers"
	"github.com/zoobz-io/barbara/config"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
	"github.com/zoobz-io/barbara/internal/boot"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Println("starting admin...")
	ctx := context.Background()

	svc, port, cleanup, err := setup(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	capitan.Emit(ctx, events.StartupServerListening, events.StartupPortKey.Field(port))
	log.Printf("starting admin server on port %d...", port)
	return svc.Run("", port)
}

// setup builds the fully wired service: shared runtime, per-surface config,
// request auth, a frozen registry, and observability. It returns the service,
// the port to serve on, and a cleanup to run on shutdown. Split from run so the
// wiring is testable without starting the blocking HTTP server.
func setup(ctx context.Context) (*sum.Service, int, func(), error) {
	// Shared setup: sum init, common config, infra, stores, model boundaries,
	// jobs pipeline.
	rt, err := boot.Init(ctx)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to initialize runtime: %w", err)
	}

	// Per-surface config.
	if cfgErr := sum.Config[config.App](ctx, rt.K, nil); cfgErr != nil {
		_ = rt.Shutdown()
		return nil, 0, nil, fmt.Errorf("failed to load app config: %w", cfgErr)
	}

	// Request auth: register the identity/entitlement resolver and set it as the
	// engine's extractor for WithAuthentication() handlers. Stubbed until
	// janus/aegis lands — swap DefaultStub() for the mesh resolver here.
	auth.Wire(rt.K, rt.Svc.Engine(), auth.DefaultStub())

	// Admin is the tenant-agnostic internal platform (#46/#51): no tenant-scoped
	// authoring, just the cross-tenant capabilities gated behind an admin
	// entitlement. Cross-tenant search is the one seeded here.
	sum.Register[contracts.Search](rt.K, rt.Stores.Search)

	sum.Freeze(rt.K)
	capitan.Emit(ctx, events.StartupServicesReady)

	rt.Svc.Handle(handlers.All()...)

	// Observability.
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "barbara-admin"
	}
	otelProviders, err := boot.OTEL(ctx, serviceName)
	if err != nil {
		_ = rt.Shutdown()
		return nil, 0, nil, fmt.Errorf("failed to create otel providers: %w", err)
	}
	capitan.Emit(ctx, events.StartupOTELReady)

	ap, err := boot.Aperture(ctx, otelProviders)
	if err != nil {
		_ = otelProviders.Shutdown(ctx)
		_ = rt.Shutdown()
		return nil, 0, nil, fmt.Errorf("failed to create aperture: %w", err)
	}
	capitan.Emit(ctx, events.StartupApertureReady)

	appCfg := sum.MustUse[config.App](ctx)
	cleanup := func() {
		ap.Close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelProviders.Shutdown(shutdownCtx)
		_ = rt.Shutdown()
	}
	return rt.Svc, appCfg.AdminPort, cleanup, nil
}
