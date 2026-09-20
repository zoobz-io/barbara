// Package main is the release-metadata rebuild command: it recomputes every
// release's change rows and counts against the release before it from the
// stored entries. Run it operationally to backfill releases that predate the
// metadata columns, or to repair rows that drifted — it walks every app across
// all tenants and rebuilds each app's history in order, idempotently. The
// kind, label, and provenance of a release are facts of the cut, not derived,
// and are left as they are.
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
	log.Println("starting release metadata rebuild...")
	n, err := rebuild(context.Background())
	if err != nil {
		return err
	}
	log.Printf("rebuild complete: changes and counts of %d releases rebuilt from their entries", n)
	return nil
}

// rebuild boots the shared runtime, rebuilds every release's metadata from its
// entries, and shuts down. Split from run so the wiring is testable without
// the process exiting.
func rebuild(ctx context.Context) (int, error) {
	rt, err := boot.Init(ctx)
	if err != nil {
		return 0, fmt.Errorf("initializing runtime: %w", err)
	}
	defer func() { _ = rt.Shutdown() }()
	return rt.Stores.RebuildReleases(ctx)
}
