package boot

import (
	"context"

	"github.com/zoobz-io/sum"
)

// Runtime holds the pieces built by Init and shared by every binary.
// Contract registration, wire boundaries and Freeze are the caller's
// responsibility — they differ per surface (public API, admin API).
type Runtime struct {
	Svc *sum.Service
	K   sum.Key
	// Uncomment as infrastructure lands:
	// DB     *sqlx.DB
	// Stores *stores.Stores
}

// Init performs the setup common to every binary: sum init, shared config
// load, infra connections, store construction and model boundary
// registration. The registry is left unfrozen so each binary can register
// its own contracts before freezing.
//
// The caller owns any returned clients — defer Close on them.
func Init(_ context.Context) (*Runtime, error) {
	svc := sum.New()
	k := sum.Start()

	// Shared config — everything every binary needs. Per-surface config
	// (ports, auth) is loaded by each binary after Init.
	// if err := sum.Config[config.Database](ctx, k, nil); err != nil {
	// 	return nil, fmt.Errorf("loading database config: %w", err)
	// }

	// Infrastructure connections.
	// db, err := Database(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("connecting to database: %w", err)
	// }
	// capitan.Emit(ctx, events.StartupDatabaseConnected)

	// Store construction — one aggregate shared by all surfaces.
	// allStores := stores.New(db, astqlpg.New())

	// Model boundaries are the same for every binary that touches the stores.
	// sum.NewBoundary[models.User](k)

	return &Runtime{Svc: svc, K: k}, nil
}
