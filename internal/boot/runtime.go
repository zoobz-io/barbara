package boot

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	astqlpg "github.com/zoobz-io/astql/postgres"
	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/grub"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/config"
	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/database/stores"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/jobs"
)

// Jobs pipeline tuning. The runner polls the outbox on DefaultInterval in
// batches of DefaultBatch; each OpenSearch write retries up to jobsMaxAttempts
// with exponential backoff from jobsBaseDelay.
const (
	jobsMaxAttempts = 5
	jobsBaseDelay   = 200 * time.Millisecond
)

// Runtime holds the pieces built by Init and shared by every binary. Contract
// registration, wire boundaries and Freeze are the caller's responsibility —
// they differ per surface (public API, admin API). The caller owns lifecycle:
// defer Shutdown.
type Runtime struct {
	Svc    *sum.Service
	K      sum.Key
	DB     *sqlx.DB
	Stores *stores.Stores
	Bucket grub.BucketProvider

	runner *jobs.Runner
	cancel context.CancelFunc
}

// Init performs the setup common to every binary: sum init, shared config load,
// infra connections (Postgres, object storage, OpenSearch), index bootstrap,
// store construction, model boundary registration, and starting the jobs
// pipeline. The registry is left unfrozen so each binary can register its own
// contracts before freezing.
//
// The caller owns the returned Runtime — defer Shutdown to stop the jobs runner
// and close connections.
func Init(ctx context.Context) (*Runtime, error) {
	svc := sum.New()
	k := sum.Start()

	// Shared config — everything every binary needs. Per-surface config
	// (ports, auth) is loaded by each binary after Init.
	for _, load := range []func() error{
		func() error { return sum.Config[config.Database](ctx, k, nil) },
		func() error { return sum.Config[config.Storage](ctx, k, nil) },
		func() error { return sum.Config[config.OpenSearch](ctx, k, nil) },
	} {
		if err := load(); err != nil {
			return nil, fmt.Errorf("loading shared config: %w", err)
		}
	}

	// Postgres — system of record.
	db, err := Database(ctx)
	if err != nil {
		return nil, err
	}
	capitan.Emit(ctx, events.StartupDatabaseConnected)

	// Object storage — assets.
	bucket, err := Bucket(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	capitan.Emit(ctx, events.StartupStorageConnected)

	// OpenSearch — serving store; ensure indices exist before serving.
	searchProvider, err := OpenSearch(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	capitan.Emit(ctx, events.StartupSearchConnected)

	osCfg := sum.MustUse[config.OpenSearch](ctx)
	if err := EnsureIndices(ctx, osCfg.Addr); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ensuring indices: %w", err)
	}
	capitan.Emit(ctx, events.StartupIndicesReady)

	// Stores aggregate — one instance shared by all surfaces.
	allStores := stores.New(db, astqlpg.New(), searchProvider)

	// Model boundaries — the same for every binary that touches the stores.
	sum.NewBoundary[models.Document](k)
	sum.NewBoundary[models.Version](k)
	sum.NewBoundary[models.Job](k)
	sum.NewBoundary[models.DocumentIndex](k)

	// Jobs pipeline — drains the outbox into OpenSearch. Runs until Shutdown.
	pipeline := jobs.NewPipeline(allStores.Search, jobsMaxAttempts, jobsBaseDelay)
	runner := jobs.NewRunner(allStores.Jobs, pipeline, jobs.DefaultInterval, jobs.DefaultBatch)
	runCtx, cancel := context.WithCancel(context.Background())
	runner.Start(runCtx)
	capitan.Emit(ctx, events.StartupJobsStarted)

	return &Runtime{
		Svc:    svc,
		K:      k,
		DB:     db,
		Stores: allStores,
		Bucket: bucket,
		runner: runner,
		cancel: cancel,
	}, nil
}

// Shutdown stops the jobs runner and closes the database connection. Safe to
// call once; the caller defers it after a successful Init.
func (rt *Runtime) Shutdown() error {
	if rt.cancel != nil {
		rt.cancel()
	}
	if rt.runner != nil {
		rt.runner.Stop()
	}
	if rt.DB != nil {
		return rt.DB.Close()
	}
	return nil
}
