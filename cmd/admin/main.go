// Package main is the entry point for the admin, authoring HTTP API.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/sum"

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

	// Shared setup: sum init, common config, infra, stores, model boundaries,
	// jobs pipeline.
	rt, err := boot.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize runtime: %w", err)
	}
	defer func() { _ = rt.Shutdown() }()

	// Per-surface config.
	if cfgErr := sum.Config[config.App](ctx, rt.K, nil); cfgErr != nil {
		return fmt.Errorf("failed to load app config: %w", cfgErr)
	}

	// Request auth: register the identity/entitlement resolver and set it as the
	// engine's extractor for WithAuthentication() handlers. Stubbed until
	// janus/aegis lands — swap DefaultStub() for the mesh resolver here.
	auth.Wire(rt.K, rt.Svc.Engine(), auth.DefaultStub())

	// Authoring contracts and handlers land with the feature tickets (#13+).

	sum.Freeze(rt.K)
	capitan.Emit(ctx, events.StartupServicesReady)

	// Observability.
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "barbara-admin"
	}
	otelProviders, err := boot.OTEL(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("failed to create otel providers: %w", err)
	}
	defer func() { _ = otelProviders.Shutdown(ctx) }()
	log.Println("observability initialized")
	capitan.Emit(ctx, events.StartupOTELReady)

	ap, err := boot.Aperture(ctx, otelProviders)
	if err != nil {
		return fmt.Errorf("failed to create aperture: %w", err)
	}
	defer ap.Close()
	capitan.Emit(ctx, events.StartupApertureReady)

	appCfg := sum.MustUse[config.App](ctx)
	capitan.Emit(ctx, events.StartupServerListening, events.StartupPortKey.Field(appCfg.AdminPort))
	log.Printf("starting admin server on port %d...", appCfg.AdminPort)
	return rt.Svc.Run("", appCfg.AdminPort)
}
