// Package main is the asset-bookkeeping rebuild command: it replaces every
// app's asset folder rollups, per-kind stats, and daily series with what a
// full listing of its objects says. Run it operationally to backfill apps
// whose objects predate the bookkeeping tables, or to repair rows that
// drifted after a bookkeeping failure — it walks every app across all tenants
// and rebuilds each from object storage, idempotently.
//
// It is a one-shot command, not an HTTP endpoint: the rebuild is
// tenant-agnostic operational tooling, and a cross-tenant rebuild has no
// place on the per-tenant request surface.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zoobz-io/barbara/internal/boot"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Println("starting asset bookkeeping rebuild...")
	n, err := rebuild(context.Background())
	if err != nil {
		return err
	}
	log.Printf("rebuild complete: asset bookkeeping of %d apps rebuilt from object storage", n)
	return nil
}

// rebuild boots the shared runtime, rebuilds every app's asset bookkeeping
// from object storage, and shuts down. Split from run so the wiring is
// testable without the process exiting.
func rebuild(ctx context.Context) (int, error) {
	rt, err := boot.Init(ctx)
	if err != nil {
		return 0, fmt.Errorf("initializing runtime: %w", err)
	}
	defer func() { _ = rt.Shutdown() }()
	return rt.Stores.RebuildAssets(ctx)
}
