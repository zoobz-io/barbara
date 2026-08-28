// Package main is the full-reindex command: it rebuilds the OpenSearch index
// from Postgres, the system of record. Run it operationally when the index is
// lost, corrupted, or a mapping change needs a rebuild — it walks every
// published document across all tenants and re-projects it into OpenSearch,
// idempotently.
//
// It is a one-shot command, not an HTTP endpoint: the reindex is tenant-agnostic
// operational tooling, and a cross-tenant rebuild has no place on the per-tenant
// request surface.
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
	log.Println("starting reindex...")
	n, err := reindex(context.Background())
	if err != nil {
		return err
	}
	log.Printf("reindex complete: %d published documents projected into OpenSearch", n)
	return nil
}

// reindex boots the shared runtime, rebuilds the index from Postgres, and shuts
// down. Split from run so the wiring is testable without the process exiting.
func reindex(ctx context.Context) (int, error) {
	rt, err := boot.Init(ctx)
	if err != nil {
		return 0, fmt.Errorf("initializing runtime: %w", err)
	}
	defer func() { _ = rt.Shutdown() }()
	return rt.Stores.Reindex(ctx)
}
