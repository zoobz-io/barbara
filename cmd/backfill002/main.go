// Package main is the one-shot plan-002 backfill: it seeds apps and
// collections from existing path-like document keys and cuts release 1 per app
// from the published pointers, so publish state survives the pointer's
// removal. Run it once per environment after migration 005 applies and before
// the tightening migration lands.
//
// It is a one-shot command, not an HTTP endpoint: the backfill is
// tenant-agnostic operational tooling (cf. cmd/reindex), idempotent, and safe
// to rerun — a tenant that already has an app is skipped.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/internal/boot"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	log.Println("starting 002 backfill...")
	res, err := backfill(context.Background())
	if err != nil {
		return err
	}
	log.Printf("backfill complete: %d tenants seeded, %d collections created, %d documents placed, %d releases cut",
		res.Tenants, res.Collections, res.Documents, res.Releases)
	return nil
}

// backfill boots the shared runtime, runs the backfill, and shuts down. Split
// from run so the wiring is testable without the process exiting.
func backfill(ctx context.Context) (*stores.Backfill002Result, error) {
	rt, err := boot.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("initializing runtime: %w", err)
	}
	defer func() { _ = rt.Shutdown() }()
	return rt.Stores.Backfill002(ctx)
}
